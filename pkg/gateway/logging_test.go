package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLog_CircularBuffer(t *testing.T) {
	rl := NewRequestLog(3)

	rl.Add(RequestEntry{Service: "s3", Action: "A"})
	rl.Add(RequestEntry{Service: "s3", Action: "B"})
	rl.Add(RequestEntry{Service: "s3", Action: "C"})

	entries := rl.Recent("", 0)
	require.Len(t, entries, 3)
	// Newest first
	assert.Equal(t, "C", entries[0].Action)
	assert.Equal(t, "B", entries[1].Action)
	assert.Equal(t, "A", entries[2].Action)

	// Add one more to wrap around
	rl.Add(RequestEntry{Service: "s3", Action: "D"})
	entries = rl.Recent("", 0)
	require.Len(t, entries, 3)
	assert.Equal(t, "D", entries[0].Action)
	assert.Equal(t, "C", entries[1].Action)
	assert.Equal(t, "B", entries[2].Action)
}

func TestRequestLog_FilterByService(t *testing.T) {
	rl := NewRequestLog(100)

	rl.Add(RequestEntry{Service: "s3", Action: "ListBuckets"})
	rl.Add(RequestEntry{Service: "dynamodb", Action: "PutItem"})
	rl.Add(RequestEntry{Service: "s3", Action: "GetObject"})

	entries := rl.Recent("s3", 0)
	require.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, "s3", e.Service)
	}
}

func TestRequestLog_Limit(t *testing.T) {
	rl := NewRequestLog(100)
	for i := 0; i < 10; i++ {
		rl.Add(RequestEntry{Service: "s3"})
	}

	entries := rl.Recent("", 5)
	assert.Len(t, entries, 5)
}

func TestRequestStats_IncrementAndSnapshot(t *testing.T) {
	rs := NewRequestStats()
	rs.Increment("s3")
	rs.Increment("s3")
	rs.Increment("dynamodb")

	snap := rs.Snapshot()
	assert.Equal(t, int64(2), snap["s3"])
	assert.Equal(t, int64(1), snap["dynamodb"])
}

func TestLoggingMiddleware(t *testing.T) {
	rl := NewRequestLog(100)
	rs := NewRequestStats()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(inner, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	entries := rl.Recent("", 0)
	require.Len(t, entries, 1)
	assert.Equal(t, "s3", entries[0].Service)
	assert.Equal(t, "ListBuckets", entries[0].Action)
	assert.Equal(t, http.MethodGet, entries[0].Method)
	assert.Equal(t, http.StatusOK, entries[0].StatusCode)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", entries[0].CallerID)

	snap := rs.Snapshot()
	assert.Equal(t, int64(1), snap["s3"])
}

func TestLoggingMiddleware_XAmzTarget(t *testing.T) {
	rl := NewRequestLog(100)
	rs := NewRequestStats()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(inner, rl, rs)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.PutItem")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	entries := rl.Recent("", 0)
	require.Len(t, entries, 1)
	assert.Equal(t, "dynamodb", entries[0].Service)
	assert.Equal(t, "PutItem", entries[0].Action)
}

func TestExtractCallerID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKID123/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	assert.Equal(t, "AKID123", extractCallerID(r))
}

func TestExtractCallerID_Empty(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.Equal(t, "", extractCallerID(r))
}
