package sdk

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

func TestRecorder_CapturesTraffic(t *testing.T) {
	// Set up a fake AWS endpoint.
	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"TableNames":["my-table"]}`))
	}))
	defer fakeAWS.Close()

	rec := NewRecorder().Wrap(fakeAWS.Client().Transport)

	req, _ := http.NewRequest("POST", fakeAWS.URL+"/", strings.NewReader(`{"Limit":10}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.ListTables")

	resp, err := rec.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	entries := rec.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Service != "dynamodb" {
		t.Errorf("expected service 'dynamodb', got %q", e.Service)
	}
	if e.Action != "ListTables" {
		t.Errorf("expected action 'ListTables', got %q", e.Action)
	}
	if e.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", e.StatusCode)
	}
	if e.RequestBody != `{"Limit":10}` {
		t.Errorf("expected request body, got %q", e.RequestBody)
	}
	if !strings.Contains(e.ResponseBody, "my-table") {
		t.Errorf("expected response body to contain 'my-table', got %q", e.ResponseBody)
	}
}

func TestRecorder_PreservesResponse(t *testing.T) {
	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "test-value")
		w.WriteHeader(201)
		w.Write([]byte(`{"created":true}`))
	}))
	defer fakeAWS.Close()

	rec := NewRecorder().Wrap(fakeAWS.Client().Transport)

	req, _ := http.NewRequest("POST", fakeAWS.URL+"/", strings.NewReader("body"))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")

	resp, err := rec.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	// Verify the caller gets the original response unchanged.
	if resp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Custom") != "test-value" {
		t.Errorf("expected X-Custom header, got %q", resp.Header.Get("X-Custom"))
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != `{"created":true}` {
		t.Errorf("expected response body preserved, got %q", string(body))
	}
}

func TestRecorder_SaveToFile(t *testing.T) {
	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer fakeAWS.Close()

	rec := NewRecorder().Wrap(fakeAWS.Client().Transport)

	req, _ := http.NewRequest("GET", fakeAWS.URL+"/my-bucket", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")

	resp, err := rec.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "recording.json")
	if err := rec.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var recording map[string]any
	if err := json.Unmarshal(data, &recording); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	entries, ok := recording["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("expected 1 entry in JSON, got %v", recording["entries"])
	}

	entry := entries[0].(map[string]any)
	if entry["service"] != "s3" {
		t.Errorf("expected service 's3', got %v", entry["service"])
	}
}

func TestRecorder_MultipleRequests(t *testing.T) {
	callCount := 0
	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer fakeAWS.Close()

	rec := NewRecorder().Wrap(fakeAWS.Client().Transport)

	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("POST", fakeAWS.URL+"/", nil)
		req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/sqs/aws4_request, SignedHeaders=host, Signature=abc")
		resp, err := rec.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip %d: %v", i, err)
		}
		resp.Body.Close()
	}

	entries := rec.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Verify entries have increasing offsets.
	for i := 1; i < len(entries); i++ {
		if entries[i].OffsetMs < entries[i-1].OffsetMs {
			t.Errorf("entry %d offset (%f) < entry %d offset (%f)", i, entries[i].OffsetMs, i-1, entries[i-1].OffsetMs)
		}
	}
}
