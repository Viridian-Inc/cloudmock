package sdk

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// serviceFromTarget extracts the service name from the X-Amz-Target header.
// Format: "ServiceName_Version.Action" → lowercase service name.
var targetToService = map[string]string{
	"dynamodb":    "dynamodb",
	"dax":         "dynamodb",
	"amazonsqs":   "sqs",
	"amazonse":    "ses",
	"amazonsns":   "sns",
	"awskms":      "kms",
	"amazonkines": "kinesis",
	"tagging":     "tagging",
	"logs":        "logs",
	"awslambda":   "lambda",
}

func detectServiceFromRequest(req *http.Request, body []byte) string {
	// X-Amz-Target header: "DynamoDB_20120810.GetItem" → "dynamodb"
	if target := req.Header.Get("X-Amz-Target"); target != "" {
		prefix := strings.ToLower(target)
		if idx := strings.Index(prefix, "_"); idx > 0 {
			prefix = prefix[:idx]
		} else if idx := strings.Index(prefix, "."); idx > 0 {
			prefix = prefix[:idx]
		}
		for k, v := range targetToService {
			if strings.HasPrefix(prefix, k) {
				return v
			}
		}
		return prefix
	}

	// JSON protocol with X-Amz-Target in content-type (e.g. SQS JSON mode)
	if ct := req.Header.Get("Content-Type"); strings.Contains(ct, "amz-json") {
		// Check body for service hints
		if len(body) > 0 {
			b := string(body)
			if strings.Contains(b, "QueueUrl") || strings.Contains(b, "QueueName") {
				return "sqs"
			}
		}
	}

	// Query/form protocol: parse Action from body for SQS, SNS, STS, IAM
	if len(body) > 0 {
		b := string(body)
		// Quick check for known action prefixes in form body
		if strings.Contains(b, "Action=") {
			if strings.Contains(b, "Queue") || strings.Contains(b, "Message") || strings.Contains(b, "Purge") {
				return "sqs"
			}
			if strings.Contains(b, "Topic") || strings.Contains(b, "Publish") || strings.Contains(b, "Subscri") {
				return "sns"
			}
			if strings.Contains(b, "CallerIdentity") || strings.Contains(b, "AssumeRole") || strings.Contains(b, "SessionToken") {
				return "sts"
			}
			if strings.Contains(b, "User") || strings.Contains(b, "Role") || strings.Contains(b, "Policy") || strings.Contains(b, "Group") || strings.Contains(b, "AccessKey") || strings.Contains(b, "InstanceProfile") {
				return "iam"
			}
			if strings.Contains(b, "LogGroup") || strings.Contains(b, "LogStream") || strings.Contains(b, "LogEvent") {
				return "logs"
			}
		}
	}

	// S3: path-based detection.
	// Any non-root path that isn't internal is S3.
	// Root path (/) with no Action in body is also S3 (ListBuckets).
	path := req.URL.Path
	if path != "/" && !strings.HasPrefix(path, "/_") {
		return "s3"
	}
	if path == "/" && len(body) == 0 && req.Header.Get("X-Amz-Target") == "" {
		return "s3"
	}

	return ""
}

// inProcessTransport implements http.RoundTripper by calling ServeHTTP directly,
// bypassing all TCP/HTTP overhead for maximum performance in tests.
type inProcessTransport struct {
	handler http.Handler

	// Pool ResponseRecorders to avoid per-call allocation.
	recorderPool sync.Pool

	// Pool bytes.Buffers for response body construction.
	bufPool sync.Pool

	// Pool bytes.Buffers for pre-reading request bodies.
	reqBufPool sync.Pool

	// Pool http.Header maps to avoid per-call map allocation.
	headerPool sync.Pool
}

func newInProcessTransport(handler http.Handler) *inProcessTransport {
	return &inProcessTransport{
		handler: handler,
		recorderPool: sync.Pool{
			New: func() any {
				return httptest.NewRecorder()
			},
		},
		bufPool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 4096))
			},
		},
		reqBufPool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 1024))
			},
		},
		headerPool: sync.Pool{
			New: func() any {
				return make(http.Header, 8)
			},
		},
	}
}

// RoundTrip executes the request by calling the handler's ServeHTTP directly.
func (t *inProcessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Ensure the request URL has a scheme and host so the recorder works correctly.
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	if req.URL.Host == "" {
		req.URL.Host = "cloudmock.local"
	}

	// Pre-read the request body into a pooled buffer so the handler reads from
	// a bytes.Reader (no extra allocation inside the gateway).
	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		rbuf := t.reqBufPool.Get().(*bytes.Buffer)
		rbuf.Reset()
		rbuf.ReadFrom(req.Body)
		req.Body.Close()
		bodyBytes = rbuf.Bytes()
		req.Body = &nopReadCloser{Reader: bytes.NewReader(bodyBytes)}
		defer func() {
			rbuf.Reset()
			t.reqBufPool.Put(rbuf)
		}()
	}

	// Inject X-Cloudmock-Service header for fast service detection (skips SigV4 parsing).
	if req.Header.Get("X-Cloudmock-Service") == "" {
		if svc := detectServiceFromRequest(req, bodyBytes); svc != "" {
			req.Header.Set("X-Cloudmock-Service", svc)
		}
	}

	// Propagate W3C traceparent from Go context if present (set by OTel SDK).
	if req.Header.Get("traceparent") == "" {
		if tp := extractTraceparentFromContext(req.Context()); tp != "" {
			req.Header.Set("traceparent", tp)
		}
	}

	// Get a pooled recorder and reset it for reuse.
	rec := t.recorderPool.Get().(*httptest.ResponseRecorder)
	rec.Body.Reset()
	rec.Code = 200
	// Reuse the existing HeaderMap — just clear entries instead of allocating
	// a fresh map. The map's backing buckets are retained.
	for k := range rec.HeaderMap {
		delete(rec.HeaderMap, k)
	}

	t.handler.ServeHTTP(rec, req)

	// Copy response body into a pooled buffer so we can return the recorder.
	buf := t.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Write(rec.Body.Bytes())

	// Build the http.Response directly instead of rec.Result() which clones
	// all headers (expensive). Take the header map from the recorder and
	// give it a pooled one for next reuse.
	hdr := rec.HeaderMap
	rec.HeaderMap = t.headerPool.Get().(http.Header)

	resp := &http.Response{
		StatusCode:    rec.Code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        hdr,
		ContentLength: int64(buf.Len()),
		Body:          &pooledReadCloser{Reader: bytes.NewReader(buf.Bytes()), buf: buf, pool: &t.bufPool, hdr: hdr, hdrPool: &t.headerPool},
		Request:       req,
	}

	// Return recorder to pool.
	t.recorderPool.Put(rec)

	return resp, nil
}

// pooledReadCloser wraps a reader and returns the underlying buffer and header
// map to their respective pools on Close.
type pooledReadCloser struct {
	io.Reader
	buf     *bytes.Buffer
	pool    *sync.Pool
	hdr     http.Header
	hdrPool *sync.Pool
}

func (p *pooledReadCloser) Close() error {
	if p.buf != nil {
		p.buf.Reset()
		p.pool.Put(p.buf)
		p.buf = nil
	}
	if p.hdr != nil {
		for k := range p.hdr {
			delete(p.hdr, k)
		}
		p.hdrPool.Put(p.hdr)
		p.hdr = nil
	}
	return nil
}

// nopReadCloser is a ReadCloser that wraps a Reader with a no-op Close.
// Unlike io.NopCloser, this avoids an interface allocation.
type nopReadCloser struct {
	io.Reader
}

func (nopReadCloser) Close() error { return nil }

// extractTraceparentFromContext checks for an OTel span in the context and
// returns a W3C traceparent string if one is active, or empty string otherwise.
func extractTraceparentFromContext(ctx context.Context) string {
	// Use the OTel propagation API to extract trace context.
	// This avoids a direct dependency on the OTel trace package internals.
	carrier := make(propagation.HeaderCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent")
}
