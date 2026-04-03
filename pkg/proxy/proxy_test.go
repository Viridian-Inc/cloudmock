package proxy

import (
	"encoding/json"
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
