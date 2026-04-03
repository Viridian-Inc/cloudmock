package sdk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func TestRecorder_EmptyBody(t *testing.T) {
	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"Items":[]}`))
	}))
	defer fakeAWS.Close()

	rec := NewRecorder().Wrap(fakeAWS.Client().Transport)

	// GET request with no body.
	req, _ := http.NewRequest("GET", fakeAWS.URL+"/my-bucket/key", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")

	resp, err := rec.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	entries := rec.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].RequestBody != "" {
		t.Errorf("expected empty request body, got %q", entries[0].RequestBody)
	}
	if entries[0].StatusCode != 200 {
		t.Errorf("expected status 200, got %d", entries[0].StatusCode)
	}
}

func TestRecorder_PostWithJSON(t *testing.T) {
	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer fakeAWS.Close()

	rec := NewRecorder().Wrap(fakeAWS.Client().Transport)

	body := `{"TableName":"users","Item":{"id":{"S":"123"},"name":{"S":"alice"}}}`
	req, _ := http.NewRequest("POST", fakeAWS.URL+"/", strings.NewReader(body))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.PutItem")
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")

	resp, err := rec.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	entries := rec.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.RequestBody != body {
		t.Errorf("expected request body %q, got %q", body, e.RequestBody)
	}
	if e.Action != "PutItem" {
		t.Errorf("expected action 'PutItem', got %q", e.Action)
	}
	if e.Service != "dynamodb" {
		t.Errorf("expected service 'dynamodb', got %q", e.Service)
	}
}

func TestRecorder_Wrap(t *testing.T) {
	// Verify Wrap() chains correctly: the custom transport is actually used.
	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-From-Custom", "yes")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer fakeAWS.Close()

	customTransport := fakeAWS.Client().Transport
	rec := NewRecorder().Wrap(customTransport)

	req, _ := http.NewRequest("GET", fakeAWS.URL+"/", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")

	resp, err := rec.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	// The response came through the custom transport (fake server).
	if resp.Header.Get("X-From-Custom") != "yes" {
		t.Errorf("expected X-From-Custom header from custom transport, got %q", resp.Header.Get("X-From-Custom"))
	}

	if len(rec.Entries()) != 1 {
		t.Errorf("expected 1 entry, got %d", len(rec.Entries()))
	}
}

func TestRecorder_Recording_Format(t *testing.T) {
	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer fakeAWS.Close()

	rec := NewRecorder().Wrap(fakeAWS.Client().Transport)

	req, _ := http.NewRequest("POST", fakeAWS.URL+"/", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/sqs/aws4_request, SignedHeaders=host, Signature=abc")

	resp, err := rec.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	recording := rec.Recording()

	if recording.ID == "" {
		t.Error("expected non-empty recording ID")
	}
	if recording.Name == "" {
		t.Error("expected non-empty recording Name")
	}
	if recording.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}
	if recording.StoppedAt == nil || recording.StoppedAt.IsZero() {
		t.Error("expected non-nil, non-zero StoppedAt")
	}
	if recording.EntryCount != 1 {
		t.Errorf("expected EntryCount 1, got %d", recording.EntryCount)
	}
	if len(recording.Entries) != 1 {
		t.Errorf("expected 1 entry in recording, got %d", len(recording.Entries))
	}
}

func TestRecorder_Concurrent(t *testing.T) {
	const n = 100

	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer fakeAWS.Close()

	rec := NewRecorder().Wrap(fakeAWS.Client().Transport)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("POST", fakeAWS.URL+"/", strings.NewReader(`{}`))
			req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc")
			resp, err := rec.RoundTrip(req)
			if err == nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	entries := rec.Entries()
	if len(entries) != n {
		t.Errorf("expected %d entries after concurrent requests, got %d", n, len(entries))
	}
}

func TestRecorder_LoadFromFile(t *testing.T) {
	fakeAWS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"result":"ok"}`))
	}))
	defer fakeAWS.Close()

	rec := NewRecorder().Wrap(fakeAWS.Client().Transport)

	req, _ := http.NewRequest("POST", fakeAWS.URL+"/", strings.NewReader(`{"Limit":5}`))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Scan")

	resp, err := rec.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "round-trip.json")
	if err := rec.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	// Load the file and verify the round-trip.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var loaded map[string]any
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if loaded["id"] == "" || loaded["id"] == nil {
		t.Error("expected non-empty id in saved recording")
	}
	if loaded["name"] == "" || loaded["name"] == nil {
		t.Error("expected non-empty name in saved recording")
	}

	entries, ok := loaded["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("expected 1 entry in saved file, got %v", loaded["entries"])
	}

	entry := entries[0].(map[string]any)
	if entry["service"] != "dynamodb" {
		t.Errorf("expected service 'dynamodb', got %v", entry["service"])
	}
	if entry["action"] != "Scan" {
		t.Errorf("expected action 'Scan', got %v", entry["action"])
	}
	if entry["status_code"] != float64(200) {
		t.Errorf("expected status_code 200, got %v", entry["status_code"])
	}
	if entry["request_body"] != `{"Limit":5}` {
		t.Errorf("expected request body, got %v", entry["request_body"])
	}
}
