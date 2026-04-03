package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProxy_DetectsService(t *testing.T) {
	tests := []struct {
		name string
		auth string
		want string
	}{
		{
			name: "s3 from auth header",
			auth: "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc",
			want: "s3",
		},
		{
			name: "dynamodb from auth header",
			auth: "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc",
			want: "dynamodb",
		},
		{
			name: "sts from auth header",
			auth: "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/sts/aws4_request, SignedHeaders=host, Signature=abc",
			want: "sts",
		},
		{
			name: "empty auth",
			auth: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			got := detectServiceFromAuth(req)
			if got != tt.want {
				t.Errorf("detectServiceFromAuth() = %q, want %q", got, tt.want)
			}
		})
	}
}

// setupFakeAWS creates a fake AWS test server and configures the proxy to use it
// for the given service. It returns the proxy and a cleanup function.
func setupFakeAWS(t *testing.T, service string, handler http.HandlerFunc) (*AWSProxy, func()) {
	t.Helper()
	fakeAWS := httptest.NewServer(handler)

	p := New("us-east-1")
	// Point the proxy's HTTP client at the fake server and override the endpoint.
	fakeHost := strings.TrimPrefix(fakeAWS.URL, "http://")
	origEndpoint := serviceEndpoints[service]
	serviceEndpoints[service] = fakeHost

	// Use a custom transport that speaks plain HTTP to the fake server.
	p.httpClient = &http.Client{Transport: &http.Transport{}}

	cleanup := func() {
		fakeAWS.Close()
		serviceEndpoints[service] = origEndpoint
	}
	return p, cleanup
}

func TestProxy_ForwardsRequest(t *testing.T) {
	p, cleanup := setupFakeAWS(t, "dynamodb", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"Tables":["test-table"]}`))
	})
	defer cleanup()

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"TableName":"test"}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.ListTables")

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "test-table") {
		t.Fatalf("expected response body to contain 'test-table', got %q", body)
	}
}

func TestProxy_RecordsEntry(t *testing.T) {
	p, cleanup := setupFakeAWS(t, "sqs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	})
	defer cleanup()

	req := httptest.NewRequest("POST", "/?Action=SendMessage", strings.NewReader("body"))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/sqs/aws4_request, SignedHeaders=host, Signature=abc")

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	entries := p.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Service != "sqs" {
		t.Errorf("expected service 'sqs', got %q", e.Service)
	}
	if e.Action != "SendMessage" {
		t.Errorf("expected action 'SendMessage', got %q", e.Action)
	}
	if e.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", e.StatusCode)
	}
	if e.RequestBody != "body" {
		t.Errorf("expected request body 'body', got %q", e.RequestBody)
	}
}

func TestProxy_SaveToFile(t *testing.T) {
	p, cleanup := setupFakeAWS(t, "s3", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	})
	defer cleanup()

	req := httptest.NewRequest("GET", "/my-bucket/key.txt", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	dir := t.TempDir()
	path := filepath.Join(dir, "recording.json")
	if err := p.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var rec map[string]any
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	entries, ok := rec["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("expected 1 entry in JSON, got %v", rec["entries"])
	}

	entry := entries[0].(map[string]any)
	if entry["service"] != "s3" {
		t.Errorf("expected service 's3' in JSON, got %v", entry["service"])
	}
}

func TestResolveEndpoint(t *testing.T) {
	tests := []struct {
		service string
		region  string
		want    string
	}{
		{"s3", "us-east-1", "s3.us-east-1.amazonaws.com"},
		{"iam", "us-east-1", "iam.amazonaws.com"},
		{"dynamodb", "eu-west-1", "dynamodb.eu-west-1.amazonaws.com"},
		{"unknown-service", "us-east-1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			got := resolveEndpoint(tt.service, tt.region)
			if got != tt.want {
				t.Errorf("resolveEndpoint(%q, %q) = %q, want %q", tt.service, tt.region, got, tt.want)
			}
		})
	}
}

func TestProxy_NoAuthHeader(t *testing.T) {
	p := New("us-east-1")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for missing auth, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "could not detect") {
		t.Errorf("unexpected error body: %s", body)
	}
}

func TestProxy_LargeBody(t *testing.T) {
	// Build a 1 MB request body.
	largeBody := strings.Repeat("x", 1024*1024)

	var receivedLen int
	p, cleanup := setupFakeAWS(t, "s3", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		receivedLen = len(data)
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	})
	defer cleanup()

	req := httptest.NewRequest("PUT", "/my-bucket/large-key", strings.NewReader(largeBody))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if receivedLen != len(largeBody) {
		t.Errorf("upstream received %d bytes, want %d", receivedLen, len(largeBody))
	}

	entries := p.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].RequestBody) != len(largeBody) {
		t.Errorf("recorded body length %d, want %d", len(entries[0].RequestBody), len(largeBody))
	}
}

func TestProxy_ErrorFromUpstream(t *testing.T) {
	p, cleanup := setupFakeAWS(t, "dynamodb", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"__type":"InternalServerError","message":"internal failure"}`))
	})
	defer cleanup()

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"TableName":"bad-table"}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Fatalf("expected 500 from upstream, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "InternalServerError") {
		t.Errorf("expected upstream error body, got %q", w.Body.String())
	}

	entries := p.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry recorded even for errors, got %d", len(entries))
	}
	if entries[0].StatusCode != 500 {
		t.Errorf("expected recorded status 500, got %d", entries[0].StatusCode)
	}
}

func TestProxy_ConcurrentRequests(t *testing.T) {
	const n = 50

	p, cleanup := setupFakeAWS(t, "sqs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	})
	defer cleanup()

	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		go func() {
			req := httptest.NewRequest("POST", "/?Action=SendMessage", strings.NewReader("msg"))
			req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/sqs/aws4_request, SignedHeaders=host, Signature=abc")
			rr := httptest.NewRecorder()
			p.ServeHTTP(rr, req)
			done <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}

	entries := p.Entries()
	if len(entries) != n {
		t.Errorf("expected %d recorded entries, got %d", n, len(entries))
	}
}

func TestProxy_PreservesHeaders(t *testing.T) {
	var gotTarget, gotContentType string

	p, cleanup := setupFakeAWS(t, "dynamodb", func(w http.ResponseWriter, r *http.Request) {
		gotTarget = r.Header.Get("X-Amz-Target")
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	})
	defer cleanup()

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.PutItem")
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if gotTarget != "DynamoDB_20120810.PutItem" {
		t.Errorf("X-Amz-Target not forwarded: got %q", gotTarget)
	}
	if gotContentType != "application/x-amz-json-1.0" {
		t.Errorf("Content-Type not forwarded: got %q", gotContentType)
	}

	entries := p.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].RequestHeaders["X-Amz-Target"] != "DynamoDB_20120810.PutItem" {
		t.Errorf("X-Amz-Target not in recorded headers: %v", entries[0].RequestHeaders)
	}
}

func TestProxy_RecordingToRecording(t *testing.T) {
	p, cleanup := setupFakeAWS(t, "s3", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	})
	defer cleanup()

	const n = 3
	for i := 0; i < n; i++ {
		req := httptest.NewRequest("GET", "/bucket/key", nil)
		req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
		rr := httptest.NewRecorder()
		p.ServeHTTP(rr, req)
	}

	rec := p.Recording()
	if rec.EntryCount != n {
		t.Errorf("expected EntryCount %d, got %d", n, rec.EntryCount)
	}
	if len(rec.Entries) != n {
		t.Errorf("expected %d entries, got %d", n, len(rec.Entries))
	}
	if rec.DurationSec < 0 {
		t.Errorf("expected non-negative DurationSec, got %d", rec.DurationSec)
	}

	// Verify entries are in chronological order (non-decreasing OffsetMs).
	for i := 1; i < len(rec.Entries); i++ {
		if rec.Entries[i].OffsetMs < rec.Entries[i-1].OffsetMs {
			t.Errorf("entry %d offset (%v) is before entry %d offset (%v)", i, rec.Entries[i].OffsetMs, i-1, rec.Entries[i-1].OffsetMs)
		}
	}
}
