package sdk

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

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
	if req.Body != nil && req.Body != http.NoBody {
		rbuf := t.reqBufPool.Get().(*bytes.Buffer)
		rbuf.Reset()
		rbuf.ReadFrom(req.Body)
		req.Body.Close()
		req.Body = &nopReadCloser{Reader: bytes.NewReader(rbuf.Bytes())}
		// Return the request buffer after ServeHTTP completes (deferred).
		defer func() {
			rbuf.Reset()
			t.reqBufPool.Put(rbuf)
		}()
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
