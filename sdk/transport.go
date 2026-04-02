package sdk

import (
	"net/http"
	"net/http/httptest"
)

// inProcessTransport implements http.RoundTripper by calling ServeHTTP directly,
// bypassing all TCP/HTTP overhead for maximum performance in tests.
type inProcessTransport struct {
	handler http.Handler
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

	rec := httptest.NewRecorder()
	t.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}
