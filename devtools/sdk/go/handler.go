package cloudmock

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// inboundHandler is an http.Handler middleware that captures inbound HTTP requests
// and sends http:inbound events to the devtools server. It wraps the ResponseWriter
// to capture the status code and response body.
type inboundHandler struct {
	next http.Handler
	conn *connection
}

func newInboundHandler(next http.Handler, conn *connection) *inboundHandler {
	return &inboundHandler{next: next, conn: conn}
}

func (h *inboundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	id := fmt.Sprintf("inbound_%d_%s", start.UnixMilli(), randomHex(6))

	// Wrap the ResponseWriter to capture status and body
	cw := &captureWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}

	// Serve the request
	h.next.ServeHTTP(cw, r)

	duration := time.Since(start).Milliseconds()

	// Capture request headers
	reqHeaders := make(map[string]any)
	for k, v := range r.Header {
		if len(v) == 1 {
			reqHeaders[k] = v[0]
		} else {
			reqHeaders[k] = v
		}
	}

	// Truncate body for transmission
	bodyStr := cw.body.String()
	if len(bodyStr) > 4096 {
		bodyStr = bodyStr[:4096]
	}

	h.conn.send("http:inbound", map[string]any{
		"id":              id,
		"direction":       "inbound",
		"method":          r.Method,
		"url":             r.URL.String(),
		"path":            r.URL.Path,
		"status":          cw.statusCode,
		"duration_ms":     duration,
		"request_headers": reqHeaders,
		"response_body":   bodyStr,
		"content_length":  w.Header().Get("Content-Length"),
		"user_agent":      r.UserAgent(),
		"remote_addr":     r.RemoteAddr,
	})
}

// captureWriter wraps http.ResponseWriter to capture the status code and response body.
type captureWriter struct {
	http.ResponseWriter
	statusCode    int
	body          *bytes.Buffer
	wroteHeader   bool
}

func (cw *captureWriter) WriteHeader(code int) {
	if !cw.wroteHeader {
		cw.statusCode = code
		cw.wroteHeader = true
	}
	cw.ResponseWriter.WriteHeader(code)
}

func (cw *captureWriter) Write(b []byte) (int, error) {
	// Capture up to 4KB of response body
	if cw.body.Len() < 4096 {
		remaining := 4096 - cw.body.Len()
		if len(b) <= remaining {
			cw.body.Write(b)
		} else {
			cw.body.Write(b[:remaining])
		}
	}
	return cw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher for streaming responses.
func (cw *captureWriter) Flush() {
	if f, ok := cw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap supports http.ResponseController (Go 1.20+).
func (cw *captureWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}

// ReadFrom implements io.ReaderFrom for efficient body forwarding.
func (cw *captureWriter) ReadFrom(src io.Reader) (int64, error) {
	if rf, ok := cw.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}
	return io.Copy(cw.ResponseWriter, src)
}

func randomHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[rand.Intn(len(hex))]
	}
	return string(b)
}
