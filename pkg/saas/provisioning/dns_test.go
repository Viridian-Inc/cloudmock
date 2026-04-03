package provisioning

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCloudflare_AddCNAME(t *testing.T) {
	var capturedBody cfCreateRecordRequest
	var capturedPath string
	var capturedAuth string

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedPath = r.URL.Path

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.WriteHeader(http.StatusOK)
		resp := cfAPIResponse{
			Success: true,
			Result:  json.RawMessage(`{"id":"rec_123"}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mock.Close()

	client := NewCloudflareClient("cf-test-token", "zone_abc")
	client.httpClient = &http.Client{
		Transport: &cfTestTransport{base: mock.URL, inner: http.DefaultTransport},
	}

	err := client.AddCNAME(context.Background(), "acme.cloudmock.io", "cm-acme.fly.dev")
	if err != nil {
		t.Fatalf("AddCNAME: %v", err)
	}

	if capturedBody.Type != "CNAME" {
		t.Errorf("Type = %q, want %q", capturedBody.Type, "CNAME")
	}
	if capturedBody.Name != "acme.cloudmock.io" {
		t.Errorf("Name = %q, want %q", capturedBody.Name, "acme.cloudmock.io")
	}
	if capturedBody.Content != "cm-acme.fly.dev" {
		t.Errorf("Content = %q, want %q", capturedBody.Content, "cm-acme.fly.dev")
	}
	if !capturedBody.Proxied {
		t.Error("Proxied should be true")
	}
	if capturedAuth != "Bearer cf-test-token" {
		t.Errorf("Authorization = %q, want %q", capturedAuth, "Bearer cf-test-token")
	}
	if !strings.Contains(capturedPath, "zone_abc") {
		t.Errorf("Path = %q, expected to contain zone ID", capturedPath)
	}
}

func TestCloudflare_RemoveCNAME(t *testing.T) {
	// RemoveCNAME first queries for the record ID, then deletes it.
	// We need to handle both requests.
	requestCount := 0
	var deleteMethod string
	var deletePath string

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if r.Method == http.MethodGet {
			// This is the findRecordID call.
			records := []cfDNSRecord{
				{ID: "rec_to_delete", Name: "acme.cloudmock.io", Type: "CNAME"},
			}
			recordsJSON, _ := json.Marshal(records)
			resp := cfAPIResponse{
				Success: true,
				Result:  recordsJSON,
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodDelete {
			deleteMethod = r.Method
			deletePath = r.URL.Path

			resp := cfAPIResponse{Success: true, Result: json.RawMessage(`{"id":"rec_to_delete"}`)}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected method: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer mock.Close()

	client := NewCloudflareClient("cf-test-token", "zone_abc")
	client.httpClient = &http.Client{
		Transport: &cfTestTransport{base: mock.URL, inner: http.DefaultTransport},
	}

	err := client.RemoveCNAME(context.Background(), "acme.cloudmock.io")
	if err != nil {
		t.Fatalf("RemoveCNAME: %v", err)
	}

	if requestCount != 2 {
		t.Errorf("expected 2 HTTP requests (find + delete), got %d", requestCount)
	}
	if deleteMethod != http.MethodDelete {
		t.Errorf("delete method = %q, want DELETE", deleteMethod)
	}
	if !strings.Contains(deletePath, "rec_to_delete") {
		t.Errorf("delete path = %q, expected to contain record ID", deletePath)
	}
}

// cfTestTransport rewrites Cloudflare API URLs to point to a test server.
type cfTestTransport struct {
	base  string
	inner http.RoundTripper
}

func (t *cfTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.base[len("http://"):]
	return t.inner.RoundTrip(req)
}
