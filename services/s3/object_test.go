package s3_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper: create a bucket (fails the test on non-200).
func mustCreateBucket(t *testing.T, handler http.Handler, bucket string) {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/"+bucket))
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /%s: expected 200, got %d – %s", bucket, w.Code, w.Body.String())
	}
}

// helper: put an object, fails the test on non-200.
func mustPutObject(t *testing.T, handler http.Handler, bucket, key, contentType, body string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/"+bucket+"/"+key, strings.NewReader(body))
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /%s/%s: expected 200, got %d – %s", bucket, key, w.Code, w.Body.String())
	}
}

// TestS3_PutGetObject tests a basic round-trip: put then get returns the same body.
func TestS3_PutGetObject(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "round-trip")

	content := "hello cloudmock object store"
	mustPutObject(t, handler, "round-trip", "greeting.txt", "text/plain", content)

	// GET /round-trip/greeting.txt → 200 with original body
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/round-trip/greeting.txt"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != content {
		t.Errorf("GET: expected body %q, got %q", content, got)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain" {
		t.Errorf("GET: expected Content-Type text/plain, got %q", ct)
	}
}

// TestS3_DeleteObject verifies that a deleted object returns 404 on subsequent GET.
func TestS3_DeleteObject(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "del-bucket")
	mustPutObject(t, handler, "del-bucket", "target.txt", "text/plain", "bye")

	// DELETE /del-bucket/target.txt → 204
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/del-bucket/target.txt"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE: expected 204, got %d", w.Code)
	}

	// GET /del-bucket/target.txt → 404
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/del-bucket/target.txt"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET after DELETE: expected 404, got %d", w.Code)
	}
}

// TestS3_HeadObject verifies that HEAD returns headers without a body.
func TestS3_HeadObject(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "head-bucket")

	// HEAD on nonexistent object → 404
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/head-bucket/missing.txt"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("HEAD nonexistent: expected 404, got %d", w.Code)
	}

	mustPutObject(t, handler, "head-bucket", "exists.bin", "application/octet-stream",
		"binary content")

	// HEAD on existing object → 200, headers set, no body
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/head-bucket/exists.bin"))
	if w.Code != http.StatusOK {
		t.Fatalf("HEAD existing: expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("HEAD: expected Content-Type application/octet-stream, got %q", ct)
	}
	if w.Header().Get("ETag") == "" {
		t.Error("HEAD: expected non-empty ETag header")
	}
	if w.Header().Get("Content-Length") == "" {
		t.Error("HEAD: expected non-empty Content-Length header")
	}
	if w.Body.Len() != 0 {
		t.Errorf("HEAD: expected empty body, got %q", w.Body.String())
	}
}

// TestS3_ListObjectsV2 verifies that all uploaded objects appear in the listing.
func TestS3_ListObjectsV2(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "list-bucket")

	keys := []string{"alpha.txt", "beta.txt", "gamma.txt"}
	for _, k := range keys {
		mustPutObject(t, handler, "list-bucket", k, "text/plain", "content of "+k)
	}

	// GET /list-bucket → 200, XML containing all keys
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/list-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("LIST: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, k := range keys {
		if !strings.Contains(body, k) {
			t.Errorf("LIST: expected body to contain %q\n%s", k, body)
		}
	}
}

// TestS3_ListObjectsV2_Prefix verifies that prefix filtering works correctly.
func TestS3_ListObjectsV2_Prefix(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "prefix-bucket")

	mustPutObject(t, handler, "prefix-bucket", "logs/2024/jan.log", "text/plain", "jan")
	mustPutObject(t, handler, "prefix-bucket", "logs/2024/feb.log", "text/plain", "feb")
	mustPutObject(t, handler, "prefix-bucket", "data/record.csv", "text/csv", "csv")

	// GET /prefix-bucket?prefix=logs/ → should return only the two log files
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/prefix-bucket?prefix=logs/", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("LIST prefix: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "jan.log") {
		t.Errorf("LIST prefix: expected jan.log in response\n%s", body)
	}
	if !strings.Contains(body, "feb.log") {
		t.Errorf("LIST prefix: expected feb.log in response\n%s", body)
	}
	if strings.Contains(body, "record.csv") {
		t.Errorf("LIST prefix: did not expect record.csv in response\n%s", body)
	}
}

// TestS3_CopyObject verifies that a copied object can be retrieved from the
// destination key with the same content.
func TestS3_CopyObject(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "src-bucket")
	mustCreateBucket(t, handler, "dst-bucket")

	content := "data to copy"
	mustPutObject(t, handler, "src-bucket", "original.txt", "text/plain", content)

	// PUT /dst-bucket/copy.txt with x-amz-copy-source → 200
	req := httptest.NewRequest(http.MethodPut, "/dst-bucket/copy.txt", bytes.NewReader(nil))
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("x-amz-copy-source", "/src-bucket/original.txt")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("COPY: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "CopyObjectResult") {
		t.Errorf("COPY: expected CopyObjectResult in response body\n%s", w.Body.String())
	}

	// GET /dst-bucket/copy.txt → should return the same content
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/dst-bucket/copy.txt"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET copy: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != content {
		t.Errorf("GET copy: expected %q, got %q", content, got)
	}
}

// TestS3_PutObject_NoSuchBucket verifies that putting an object into a
// nonexistent bucket returns 404.
func TestS3_PutObject_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	req := httptest.NewRequest(http.MethodPut, "/ghost-bucket/key.txt", strings.NewReader("data"))
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("PUT nonexistent bucket: expected 404, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("PUT nonexistent bucket: expected NoSuchBucket in body\n%s", w.Body.String())
	}
}
