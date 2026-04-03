package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContract_BothEndpointsHit(t *testing.T) {
	var awsHits, cmHits int32
	awsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&awsHits, 1)
		w.WriteHeader(200)
		w.Write([]byte(`{"Items":[],"Count":0}`))
	}))
	defer awsServer.Close()
	cmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&cmHits, 1)
		w.WriteHeader(200)
		w.Write([]byte(`{"Items":[],"Count":0}`))
	}))
	defer cmServer.Close()

	cp := NewContractProxy("us-east-1", cmServer.URL, nil)
	cp.awsProxy.testEndpoint = awsServer.URL

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/dynamodb/aws4_request")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Scan")
	w := httptest.NewRecorder()
	cp.ServeHTTP(w, req)

	assert.Equal(t, int32(1), atomic.LoadInt32(&awsHits), "AWS endpoint should be hit once")
	assert.Equal(t, int32(1), atomic.LoadInt32(&cmHits), "CloudMock endpoint should be hit once")
	assert.Equal(t, 200, w.Code, "should return AWS response status")
}

func TestContract_AWSResponseReturned(t *testing.T) {
	awsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"source":"aws"}`))
	}))
	defer awsServer.Close()
	cmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"source":"cloudmock"}`))
	}))
	defer cmServer.Close()

	cp := NewContractProxy("us-east-1", cmServer.URL, nil)
	cp.awsProxy.testEndpoint = awsServer.URL

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/dynamodb/aws4_request")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Scan")
	w := httptest.NewRecorder()
	cp.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"source":"aws"`, "caller should receive the AWS response")
}

func TestContract_MatchRecorded(t *testing.T) {
	body := `{"Items":[],"Count":0}`
	awsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(body))
	}))
	defer awsServer.Close()
	cmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(body))
	}))
	defer cmServer.Close()

	cp := NewContractProxy("us-east-1", cmServer.URL, nil)
	cp.awsProxy.testEndpoint = awsServer.URL

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/dynamodb/aws4_request")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Scan")
	w := httptest.NewRecorder()
	cp.ServeHTTP(w, req)

	results := cp.Results()
	require.Len(t, results, 1)
	assert.True(t, results[0].Match, "identical responses should match")
	assert.Empty(t, results[0].Diffs)
}

func TestContract_MismatchDetected(t *testing.T) {
	awsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer awsServer.Close()
	cmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"ok":false}`))
	}))
	defer cmServer.Close()

	cp := NewContractProxy("us-east-1", cmServer.URL, nil)
	cp.awsProxy.testEndpoint = awsServer.URL

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/dynamodb/aws4_request")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Scan")
	w := httptest.NewRecorder()
	cp.ServeHTTP(w, req)

	results := cp.Results()
	require.Len(t, results, 1)
	assert.False(t, results[0].Match)
	assert.Equal(t, "status", results[0].Severity)
	assert.Equal(t, 200, results[0].AWSStatus)
	assert.Equal(t, 500, results[0].CloudMockStatus)
}

func TestContract_BodyDiffCaptured(t *testing.T) {
	awsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"Count":5,"Items":["a","b"]}`))
	}))
	defer awsServer.Close()
	cmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"Count":3,"Items":["a"]}`))
	}))
	defer cmServer.Close()

	cp := NewContractProxy("us-east-1", cmServer.URL, nil)
	cp.awsProxy.testEndpoint = awsServer.URL

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/dynamodb/aws4_request")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Scan")
	w := httptest.NewRecorder()
	cp.ServeHTTP(w, req)

	results := cp.Results()
	require.Len(t, results, 1)
	assert.False(t, results[0].Match)
	assert.NotEmpty(t, results[0].Diffs, "body diffs should be captured")
	assert.Equal(t, "data", results[0].Severity)
}

func TestContract_Report(t *testing.T) {
	awsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer awsServer.Close()

	// CloudMock matches on first request, mismatches on second.
	var cmCount int32
	cmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&cmCount, 1)
		if n == 1 {
			w.WriteHeader(200)
			w.Write([]byte(`{"ok":true}`))
		} else {
			w.WriteHeader(200)
			w.Write([]byte(`{"ok":false}`))
		}
	}))
	defer cmServer.Close()

	cp := NewContractProxy("us-east-1", cmServer.URL, nil)
	cp.awsProxy.testEndpoint = awsServer.URL

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
		req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/dynamodb/aws4_request")
		req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Scan")
		w := httptest.NewRecorder()
		cp.ServeHTTP(w, req)
	}

	report := cp.Report()
	assert.Equal(t, 2, report.TotalRequests)
	assert.Equal(t, 1, report.Matched)
	assert.Equal(t, 1, report.Mismatched)
	assert.Equal(t, 50.0, report.CompatibilityPct)
	assert.Len(t, report.Mismatches, 1)

	// Verify by-service breakdown.
	sr, ok := report.ByService["dynamodb"]
	require.True(t, ok, "should have dynamodb service report")
	assert.Equal(t, 2, sr.Total)
	assert.Equal(t, 1, sr.Matched)
	assert.Equal(t, 50.0, sr.Pct)

	// Verify JSON serialisation round-trips.
	data, err := json.Marshal(report)
	require.NoError(t, err)
	var decoded ContractReport
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, report.TotalRequests, decoded.TotalRequests)
}

func TestContract_IgnorePaths(t *testing.T) {
	awsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"Items":[],"RequestId":"aws-123","ResponseMetadata":{"RequestId":"aws-123"}}`))
	}))
	defer awsServer.Close()
	cmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"Items":[],"RequestId":"cm-456","ResponseMetadata":{"RequestId":"cm-456"}}`))
	}))
	defer cmServer.Close()

	cp := NewContractProxy("us-east-1", cmServer.URL, []string{"RequestId", "ResponseMetadata"})
	cp.awsProxy.testEndpoint = awsServer.URL

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/dynamodb/aws4_request")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Scan")
	w := httptest.NewRecorder()
	cp.ServeHTTP(w, req)

	results := cp.Results()
	require.Len(t, results, 1)
	assert.True(t, results[0].Match, "should match when only ignored paths differ")
}

func TestContract_Concurrent(t *testing.T) {
	awsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer awsServer.Close()
	cmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer cmServer.Close()

	cp := NewContractProxy("us-east-1", cmServer.URL, nil)
	cp.awsProxy.testEndpoint = awsServer.URL

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
			req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/dynamodb/aws4_request")
			req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Scan")
			w := httptest.NewRecorder()
			cp.ServeHTTP(w, req)
		}()
	}
	wg.Wait()

	results := cp.Results()
	assert.Len(t, results, n, "all concurrent requests should be recorded")

	report := cp.Report()
	assert.Equal(t, n, report.TotalRequests)
	assert.Equal(t, n, report.Matched)
	assert.Equal(t, 100.0, report.CompatibilityPct)
}
