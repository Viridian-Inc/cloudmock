package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/traffic"
)

// mockAWSServer creates a fake "AWS" server that handles S3, DynamoDB, and SQS
// requests and returns plausible-looking responses based on the service detected
// from the Authorization header.
func mockAWSServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(auth, "/s3/"):
			w.WriteHeader(200)
			w.Write([]byte(`{"ETag":"\"abc123\"","VersionId":"v1"}`))
		case strings.Contains(auth, "/dynamodb/"):
			w.WriteHeader(200)
			w.Write([]byte(`{"Item":{"id":{"S":"item-1"},"name":{"S":"widget"}}}`))
		case strings.Contains(auth, "/sqs/"):
			w.WriteHeader(200)
			w.Write([]byte(`{"MessageId":"msg-abc-123","MD5OfMessageBody":"d41d8cd98f00b204e9800998ecf8427e"}`))
		default:
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"unknown service"}`))
		}
	}))
}

func makeAuthHeader(service string) string {
	return "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/" + service + "/aws4_request, SignedHeaders=host, Signature=abc"
}

func TestIntegration_RecordMultipleServices(t *testing.T) {
	mock := mockAWSServer(t)
	defer mock.Close()

	fakeHost := strings.TrimPrefix(mock.URL, "http://")

	// Override endpoints for all three services to point at our mock server.
	origS3 := serviceEndpoints["s3"]
	origDynamo := serviceEndpoints["dynamodb"]
	origSQS := serviceEndpoints["sqs"]
	serviceEndpoints["s3"] = fakeHost
	serviceEndpoints["dynamodb"] = fakeHost
	serviceEndpoints["sqs"] = fakeHost
	defer func() {
		serviceEndpoints["s3"] = origS3
		serviceEndpoints["dynamodb"] = origDynamo
		serviceEndpoints["sqs"] = origSQS
	}()

	p := New("us-east-1")
	p.httpClient = &http.Client{Transport: &http.Transport{}}

	// --- S3-like request: PutObject ---
	s3Req := httptest.NewRequest("PUT", "/my-bucket/object.json", strings.NewReader(`{"data":"hello"}`))
	s3Req.Header.Set("Authorization", makeAuthHeader("s3"))
	s3Req.Header.Set("Content-Type", "application/json")
	s3W := httptest.NewRecorder()
	p.ServeHTTP(s3W, s3Req)

	if s3W.Code != 200 {
		t.Fatalf("S3 request: expected 200, got %d: %s", s3W.Code, s3W.Body.String())
	}

	// --- DynamoDB-like request: GetItem ---
	dynamoReq := httptest.NewRequest("POST", "/", strings.NewReader(`{"TableName":"users","Key":{"id":{"S":"item-1"}}}`))
	dynamoReq.Header.Set("Authorization", makeAuthHeader("dynamodb"))
	dynamoReq.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	dynamoW := httptest.NewRecorder()
	p.ServeHTTP(dynamoW, dynamoReq)

	if dynamoW.Code != 200 {
		t.Fatalf("DynamoDB request: expected 200, got %d: %s", dynamoW.Code, dynamoW.Body.String())
	}

	// --- SQS-like request: SendMessage ---
	sqsReq := httptest.NewRequest("POST", "/?Action=SendMessage", strings.NewReader("MessageBody=hello+world"))
	sqsReq.Header.Set("Authorization", makeAuthHeader("sqs"))
	sqsW := httptest.NewRecorder()
	p.ServeHTTP(sqsW, sqsReq)

	if sqsW.Code != 200 {
		t.Fatalf("SQS request: expected 200, got %d: %s", sqsW.Code, sqsW.Body.String())
	}

	// --- Verify all three requests were recorded ---
	entries := p.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 recorded entries, got %d", len(entries))
	}

	servicesSeen := make(map[string]bool)
	for _, e := range entries {
		servicesSeen[e.Service] = true
	}
	for _, svc := range []string{"s3", "dynamodb", "sqs"} {
		if !servicesSeen[svc] {
			t.Errorf("expected entry for service %q, but none found", svc)
		}
	}

	// Verify DynamoDB entry has correct action.
	for _, e := range entries {
		if e.Service == "dynamodb" && e.Action != "GetItem" {
			t.Errorf("expected DynamoDB action 'GetItem', got %q", e.Action)
		}
		if e.Service == "sqs" && e.Action != "SendMessage" {
			t.Errorf("expected SQS action 'SendMessage', got %q", e.Action)
		}
	}

	// Verify all status codes are 200.
	for _, e := range entries {
		if e.StatusCode != 200 {
			t.Errorf("entry %s/%s: expected status 200, got %d", e.Service, e.Action, e.StatusCode)
		}
	}
}

func TestIntegration_SaveAndLoadRecording(t *testing.T) {
	mock := mockAWSServer(t)
	defer mock.Close()

	fakeHost := strings.TrimPrefix(mock.URL, "http://")

	origS3 := serviceEndpoints["s3"]
	origDynamo := serviceEndpoints["dynamodb"]
	serviceEndpoints["s3"] = fakeHost
	serviceEndpoints["dynamodb"] = fakeHost
	defer func() {
		serviceEndpoints["s3"] = origS3
		serviceEndpoints["dynamodb"] = origDynamo
	}()

	p := New("us-east-1")
	p.httpClient = &http.Client{Transport: &http.Transport{}}

	// Make two requests.
	for _, tc := range []struct {
		method  string
		path    string
		auth    string
		target  string
		body    string
	}{
		{"GET", "/bucket/key.json", "s3", "", ""},
		{"POST", "/", "dynamodb", "DynamoDB_20120810.PutItem", `{"TableName":"t","Item":{}}`},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Authorization", makeAuthHeader(tc.auth))
		if tc.target != "" {
			req.Header.Set("X-Amz-Target", tc.target)
		}
		rr := httptest.NewRecorder()
		p.ServeHTTP(rr, req)
	}

	// Save to file.
	dir := t.TempDir()
	path := filepath.Join(dir, "integration.json")
	if err := p.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	// Load the file and verify.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var rec traffic.Recording
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatalf("Unmarshal into Recording: %v", err)
	}

	if rec.EntryCount != 2 {
		t.Errorf("expected EntryCount 2, got %d", rec.EntryCount)
	}
	if len(rec.Entries) != 2 {
		t.Errorf("expected 2 entries in loaded recording, got %d", len(rec.Entries))
	}
	if rec.Status != traffic.RecordingCompleted {
		t.Errorf("expected status %q, got %q", traffic.RecordingCompleted, rec.Status)
	}

	// Verify service names.
	services := make(map[string]bool)
	actions := make(map[string]bool)
	for _, e := range rec.Entries {
		services[e.Service] = true
		if e.Action != "" {
			actions[e.Action] = true
		}
		if e.StatusCode != 200 {
			t.Errorf("entry %s/%s: expected status 200, got %d", e.Service, e.Action, e.StatusCode)
		}
	}
	if !services["s3"] {
		t.Error("expected s3 entry in loaded recording")
	}
	if !services["dynamodb"] {
		t.Error("expected dynamodb entry in loaded recording")
	}
	if !actions["PutItem"] {
		t.Error("expected PutItem action in loaded recording")
	}
}
