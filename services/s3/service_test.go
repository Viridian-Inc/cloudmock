package s3_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	s3svc "github.com/neureaux/cloudmock/services/s3"
)

// newS3Gateway builds a full gateway stack with the S3 service registered.
func newS3Gateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(s3svc.New())

	return gateway.New(cfg, reg)
}

// s3Req builds an HTTP request with an S3 Authorization header.
func s3Req(t *testing.T, method, path string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

func TestS3_CreateAndListBuckets(t *testing.T) {
	handler := newS3Gateway(t)

	// PUT /my-test-bucket → 200
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/my-test-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /my-test-bucket: expected 200, got %d", w.Code)
	}

	// GET / → 200 and body contains "my-test-bucket"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "my-test-bucket") {
		t.Errorf("GET /: expected body to contain \"my-test-bucket\", got:\n%s", body)
	}
}

func TestS3_DeleteBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// PUT /delete-me → 200
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/delete-me"))
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /delete-me: expected 200, got %d", w.Code)
	}

	// DELETE /delete-me → 204
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/delete-me"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE /delete-me: expected 204, got %d", w.Code)
	}

	// GET / → body does NOT contain "delete-me"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "delete-me") {
		t.Errorf("GET /: expected body NOT to contain \"delete-me\", got:\n%s", body)
	}
}

func TestS3_HeadBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// HEAD /nope → 404
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/nope"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("HEAD /nope: expected 404, got %d", w.Code)
	}

	// PUT /exists → 200
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/exists"))
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /exists: expected 200, got %d", w.Code)
	}

	// HEAD /exists → 200
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/exists"))
	if w.Code != http.StatusOK {
		t.Fatalf("HEAD /exists: expected 200, got %d", w.Code)
	}
}
