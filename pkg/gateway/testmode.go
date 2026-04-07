package gateway

import (
	"io"
	"net/http"
	"sync"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// bodyBufPool reuses byte slices for reading request bodies.
var bodyBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 4096)
		return &b
	},
}

// ctxPool reuses RequestContext structs.
var ctxPool = sync.Pool{
	New: func() any {
		return &service.RequestContext{}
	},
}

// TestModeHandler creates a minimal HTTP handler that bypasses all middleware
// (logging, chaos, CORS, rate limiting) for maximum throughput. Services are
// pre-resolved into a lock-free map at construction time.
//
// This handler skips: IAM auth, event bus, plugin manager, account registry,
// response recording, tracing, SLO, body capture, and all observability.
func TestModeHandler(cfg *service.CallerIdentity, region, accountID string, registry *routing.Registry) http.Handler {
	// Pre-resolve every service at startup so we never touch the registry mutex.
	all := registry.All()
	services := make(map[string]service.Service, len(all))
	for _, svc := range all {
		services[svc.Name()] = svc
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check fast path.
		if r.URL.Path == "/_cloudmock/health" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Detect service from Authorization header credential scope.
		svcName := routing.DetectService(r)
		if svcName == "" {
			w.Header().Set("Content-Type", "text/xml")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<ErrorResponse><Error><Code>MissingAuthenticationToken</Code><Message>No service detected</Message></Error></ErrorResponse>`))
			return
		}

		svc, ok := services[svcName]
		if !ok {
			w.Header().Set("Content-Type", "text/xml")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`<ErrorResponse><Error><Code>ServiceUnavailable</Code><Message>Service not registered</Message></Error></ErrorResponse>`))
			return
		}

		// Read body using pooled buffer.
		var body []byte
		if r.Body != nil {
			bufp := bodyBufPool.Get().(*[]byte)
			buf := (*bufp)[:0]
			buf, _ = appendReadAll(buf, r.Body)
			r.Body.Close()
			body = buf
			// Return to pool after handler completes (body is copied by handlers that need it).
			defer func() {
				*bufp = buf[:0]
				bodyBufPool.Put(bufp)
			}()
		}

		// Detect action inline — avoids second header lookup.
		action := routing.DetectAction(r)

		// Build params only when query string exists.
		var params map[string]string
		if r.URL.RawQuery != "" {
			qv := r.URL.Query()
			if len(qv) > 0 {
				params = make(map[string]string, len(qv))
				for k, v := range qv {
					if len(v) > 0 {
						params[k] = v[0]
					}
				}
			}
		}

		// Use pooled RequestContext.
		ctx := ctxPool.Get().(*service.RequestContext)
		ctx.Action = action
		ctx.Region = region
		ctx.AccountID = accountID
		ctx.Identity = cfg
		ctx.RawRequest = r
		ctx.Body = body
		ctx.Params = params
		ctx.Service = svcName

		resp, svcErr := svc.HandleRequest(ctx)

		// Reset and return to pool.
		ctx.Action = ""
		ctx.Region = ""
		ctx.AccountID = ""
		ctx.Identity = nil
		ctx.RawRequest = nil
		ctx.Body = nil
		ctx.Params = nil
		ctx.Service = ""
		ctxPool.Put(ctx)

		if svcErr != nil {
			if awsErr, ok := svcErr.(*service.AWSError); ok {
				format := service.FormatXML
				ct := r.Header.Get("Content-Type")
				if ct == "application/x-amz-json-1.0" || ct == "application/x-amz-json-1.1" || ct == "application/json" {
					format = service.FormatJSON
				}
				_ = service.WriteErrorResponse(w, awsErr, format)
			} else {
				http.Error(w, svcErr.Error(), http.StatusInternalServerError)
			}
			return
		}

		if resp == nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}

		// Raw body takes priority — write bytes directly without marshaling.
		if resp.RawBody != nil {
			if resp.RawContentType != "" {
				w.Header().Set("Content-Type", resp.RawContentType)
			}
			w.WriteHeader(resp.StatusCode)
			w.Write(resp.RawBody)
			return
		}

		if resp.Body == nil {
			w.WriteHeader(resp.StatusCode)
			return
		}

		switch resp.Format {
		case service.FormatJSON:
			_ = service.WriteJSONResponse(w, resp.StatusCode, resp.Body)
		default:
			_ = service.WriteXMLResponse(w, resp.StatusCode, resp.Body)
		}
	})
}

// appendReadAll reads all from r into buf, growing as needed. Avoids io.ReadAll's
// allocation of a new slice on every call.
func appendReadAll(buf []byte, r io.Reader) ([]byte, error) {
	for {
		if len(buf) == cap(buf) {
			// Grow buffer.
			newBuf := make([]byte, len(buf), cap(buf)*2+512)
			copy(newBuf, buf)
			buf = newBuf
		}
		n, err := r.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		if err != nil {
			if err == io.EOF {
				return buf, nil
			}
			return buf, err
		}
	}
}
