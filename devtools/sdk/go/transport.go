package cloudmock

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

var outboundRequestCounter atomic.Int64

// captureTransport is an http.RoundTripper that captures outbound HTTP calls
// and sends http:response events to the devtools server.
type captureTransport struct {
	inner http.RoundTripper
	conn  *connection
}

func (t *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	id := fmt.Sprintf("go_req_%d_%d", outboundRequestCounter.Add(1), time.Now().UnixMilli())
	start := time.Now()

	method := req.Method
	urlStr := req.URL.String()
	path := req.URL.Path

	// Inject correlation headers
	req.Header.Set("X-CloudMock-Source", t.conn.appName)
	req.Header.Set("X-CloudMock-Request-Id", id)

	resp, err := t.inner.RoundTrip(req)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		t.conn.send("http:error", map[string]any{
			"id":          id,
			"method":      method,
			"url":         urlStr,
			"path":        path,
			"error":       err.Error(),
			"duration_ms": duration,
		})
		return nil, err
	}

	// Read response body for capture (up to 4KB), then restore it
	var bodyStr string
	if resp.Body != nil {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr == nil {
			bodyStr = string(bodyBytes)
		}
		// Read the rest of the body that wasn't captured
		remaining, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Reassemble the body so the caller gets the full content
		fullBody := append(bodyBytes, remaining...)
		resp.Body = io.NopCloser(bytes.NewReader(fullBody))
	}

	// Capture request headers
	reqHeaders := make(map[string]any)
	for k, v := range req.Header {
		if len(v) == 1 {
			reqHeaders[k] = v[0]
		} else {
			reqHeaders[k] = v
		}
	}

	// Capture response headers
	respHeaders := make(map[string]any)
	for k, v := range resp.Header {
		if len(v) == 1 {
			respHeaders[k] = v[0]
		} else {
			respHeaders[k] = v
		}
	}

	t.conn.send("http:response", map[string]any{
		"id":               id,
		"method":           method,
		"url":              urlStr,
		"path":             path,
		"status":           resp.StatusCode,
		"duration_ms":      duration,
		"request_headers":  reqHeaders,
		"response_headers": respHeaders,
		"response_body":    bodyStr,
		"content_length":   resp.Header.Get("Content-Length"),
	})

	return resp, nil
}
