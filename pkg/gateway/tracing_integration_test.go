package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTracingGateway creates a gateway handler with a trace store wired up.
func newTracingGateway(t *testing.T) (http.Handler, *gateway.TraceStore) {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(&echoService{})

	gw := gateway.New(cfg, reg)
	traceStore := gateway.NewTraceStore(100)

	handler := gateway.LoggingMiddlewareWithOpts(gw, nil, nil, gateway.LoggingMiddlewareOpts{
		TraceStore: traceStore,
	})
	return handler, traceStore
}

// s3Request builds a basic S3 ListBuckets request.
func s3Request() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

func TestTracing_SpanCreated(t *testing.T) {
	handler, _ := newTracingGateway(t)

	req := s3Request()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-Cloudmock-Trace-Id"), "response should have X-Cloudmock-Trace-Id")
	assert.NotEmpty(t, w.Header().Get("X-Cloudmock-Span-Id"), "response should have X-Cloudmock-Span-Id")
}

func TestTracing_TraceparentPropagated(t *testing.T) {
	handler, _ := newTracingGateway(t)

	incomingTraceID := "abcdef1234567890abcdef1234567890"
	incomingSpanID := "1234567890abcdef"
	traceparent := "00-" + incomingTraceID + "-" + incomingSpanID + "-01"

	req := s3Request()
	req.Header.Set("traceparent", traceparent)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// The response trace ID should match the incoming trace ID.
	assert.Equal(t, incomingTraceID, w.Header().Get("X-Cloudmock-Trace-Id"),
		"trace ID should be propagated from traceparent")

	// The span ID should be different (CloudMock generates a new one for this span).
	respSpanID := w.Header().Get("X-Cloudmock-Span-Id")
	assert.NotEmpty(t, respSpanID)
	assert.NotEqual(t, incomingSpanID, respSpanID,
		"span ID should be a new one, not the parent span ID")

	// The response traceparent should use the same trace ID but a new span ID.
	respTP := w.Header().Get("traceparent")
	require.NotEmpty(t, respTP)
	parts := strings.Split(respTP, "-")
	require.Len(t, parts, 4)
	assert.Equal(t, "00", parts[0])
	assert.Equal(t, incomingTraceID, parts[1])
	assert.Equal(t, respSpanID, parts[2])
	assert.Equal(t, "01", parts[3])
}

func TestTracing_ResponseHeaders(t *testing.T) {
	handler, _ := newTracingGateway(t)

	req := s3Request()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Validate traceparent format: 00-{32hex}-{16hex}-01
	tp := w.Header().Get("traceparent")
	require.NotEmpty(t, tp, "response should have traceparent header")

	re := regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-01$`)
	assert.Regexp(t, re, tp, "traceparent should be valid W3C format")
}

func TestTracing_SpanAttributes(t *testing.T) {
	handler, traceStore := newTracingGateway(t)

	req := s3Request()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	traceID := w.Header().Get("X-Cloudmock-Trace-Id")
	require.NotEmpty(t, traceID)

	trace := traceStore.Get(traceID)
	require.NotNil(t, trace, "trace should be in the store")
	assert.Equal(t, "s3", trace.Service)
	assert.Equal(t, "ListBuckets", trace.Action)
	assert.Equal(t, http.MethodGet, trace.Method)
	assert.Equal(t, http.StatusOK, trace.StatusCode)
}

func TestTracing_TraceStorePopulated(t *testing.T) {
	handler, traceStore := newTracingGateway(t)

	// Make multiple requests.
	for i := 0; i < 5; i++ {
		req := s3Request()
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	recent := traceStore.Recent("", nil, 10)
	assert.Len(t, recent, 5, "trace store should have 5 entries")
}

func TestTracing_AlwaysOnWithoutTraceStore(t *testing.T) {
	// Even without a trace store, response headers should include trace/span IDs.
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(&echoService{})
	gw := gateway.New(cfg, reg)

	// Use basic logging middleware without trace store.
	handler := gateway.LoggingMiddleware(gw, gateway.NewRequestLog(10), gateway.NewRequestStats())

	req := s3Request()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-Cloudmock-Trace-Id"))
	assert.NotEmpty(t, w.Header().Get("X-Cloudmock-Span-Id"))
	assert.NotEmpty(t, w.Header().Get("traceparent"))
}
