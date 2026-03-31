package s3_test

import (
	"encoding/xml"
	"fmt"
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

// s3ReqWithBody builds an HTTP request with a body and S3 Authorization header.
func s3ReqWithBody(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// ---- Bucket Operations ----

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

// ---- Bucket Error Cases ----

func TestS3_CreateBucket_AlreadyExists(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "dupe-bucket")

	// Second PUT with same name → 409 BucketAlreadyExists
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/dupe-bucket"))
	if w.Code != http.StatusConflict {
		t.Fatalf("PUT duplicate: expected 409, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "BucketAlreadyExists") {
		t.Errorf("expected BucketAlreadyExists in body, got:\n%s", w.Body.String())
	}
}

func TestS3_DeleteBucket_NotEmpty(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "full-bucket")
	mustPutObject(t, handler, "full-bucket", "obj.txt", "text/plain", "content")

	// DELETE /full-bucket → 409 BucketNotEmpty
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/full-bucket"))
	if w.Code != http.StatusConflict {
		t.Fatalf("DELETE non-empty: expected 409, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "BucketNotEmpty") {
		t.Errorf("expected BucketNotEmpty in body, got:\n%s", w.Body.String())
	}
}

func TestS3_DeleteBucket_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// DELETE /nonexistent → 404 NoSuchBucket
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/nonexistent"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("DELETE nonexistent: expected 404, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_HeadBucket_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// HEAD /phantom → 404
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/phantom"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("HEAD nonexistent: expected 404, got %d", w.Code)
	}
}

func TestS3_ListBuckets_Empty(t *testing.T) {
	handler := newS3Gateway(t)

	// GET / with no buckets → 200 with empty Buckets list
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/"))
	if w.Code != http.StatusOK {
		t.Fatalf("LIST empty: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "ListAllMyBucketsResult") {
		t.Errorf("expected ListAllMyBucketsResult in body, got:\n%s", body)
	}
}

func TestS3_ListBuckets_MultipleBuckets(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "alpha-bucket")
	mustCreateBucket(t, handler, "beta-bucket")
	mustCreateBucket(t, handler, "gamma-bucket")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/"))
	if w.Code != http.StatusOK {
		t.Fatalf("LIST: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, name := range []string{"alpha-bucket", "beta-bucket", "gamma-bucket"} {
		if !strings.Contains(body, name) {
			t.Errorf("LIST: expected %q in body, got:\n%s", name, body)
		}
	}
}

func TestS3_ListBuckets_XMLStructure(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "struct-bucket")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/"))
	if w.Code != http.StatusOK {
		t.Fatalf("LIST: expected 200, got %d", w.Code)
	}

	type xmlBucket struct {
		Name         string `xml:"Name"`
		CreationDate string `xml:"CreationDate"`
	}
	type xmlBuckets struct {
		Buckets []xmlBucket `xml:"Bucket"`
	}
	type result struct {
		XMLName xml.Name   `xml:"ListAllMyBucketsResult"`
		Owner   struct{ ID string } `xml:"Owner"`
		Buckets xmlBuckets `xml:"Buckets"`
	}

	var r result
	if err := xml.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatalf("failed to parse XML: %v\n%s", err, w.Body.String())
	}
	if len(r.Buckets.Buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(r.Buckets.Buckets))
	}
	if r.Buckets.Buckets[0].Name != "struct-bucket" {
		t.Errorf("expected bucket name struct-bucket, got %s", r.Buckets.Buckets[0].Name)
	}
	if r.Buckets.Buckets[0].CreationDate == "" {
		t.Error("expected non-empty CreationDate")
	}
}

// ---- Object Error Cases ----

func TestS3_GetObject_NoSuchKey(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "key-bucket")

	// GET nonexistent key → 404 NoSuchKey
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/key-bucket/nope.txt"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET nonexistent key: expected 404, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchKey") {
		t.Errorf("expected NoSuchKey in body, got:\n%s", w.Body.String())
	}
}

func TestS3_GetObject_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// GET from nonexistent bucket → 404 NoSuchBucket
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/ghost-bucket/file.txt"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET nonexistent bucket: expected 404, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_HeadObject_NoSuchKey(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "head-err-bucket")

	// HEAD nonexistent key → 404
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/head-err-bucket/missing.txt"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("HEAD nonexistent key: expected 404, got %d", w.Code)
	}
}

func TestS3_HeadObject_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// HEAD object in nonexistent bucket → 404
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/ghost-bucket/file.txt"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("HEAD nonexistent bucket: expected 404, got %d", w.Code)
	}
}

func TestS3_DeleteObject_Idempotent(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "idempotent-bucket")

	// DELETE nonexistent key → 204 (S3 DELETE is idempotent)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/idempotent-bucket/nope.txt"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE nonexistent key: expected 204, got %d – %s", w.Code, w.Body.String())
	}
}

func TestS3_DeleteObject_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// DELETE object from nonexistent bucket → 404 NoSuchBucket
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/phantom-bucket/key.txt"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("DELETE nonexistent bucket: expected 404, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_PutObject_OverwriteExisting(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "overwrite-bucket")
	mustPutObject(t, handler, "overwrite-bucket", "key.txt", "text/plain", "original")

	// Overwrite with new content
	mustPutObject(t, handler, "overwrite-bucket", "key.txt", "text/plain", "updated")

	// GET should return the new content
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/overwrite-bucket/key.txt"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "updated" {
		t.Errorf("GET: expected %q, got %q", "updated", got)
	}
}

func TestS3_PutObject_DefaultContentType(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "ct-bucket")

	// PUT without explicit Content-Type
	req := s3ReqWithBody(t, http.MethodPut, "/ct-bucket/noct.bin", "binary data")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	// HEAD should show default content type
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/ct-bucket/noct.bin"))
	if w.Code != http.StatusOK {
		t.Fatalf("HEAD: expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("expected default Content-Type application/octet-stream, got %q", ct)
	}
}

func TestS3_GetObject_ResponseHeaders(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "hdr-bucket")
	mustPutObject(t, handler, "hdr-bucket", "file.txt", "text/plain", "hello")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/hdr-bucket/file.txt"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d", w.Code)
	}

	if w.Header().Get("ETag") == "" {
		t.Error("GET: expected non-empty ETag header")
	}
	if w.Header().Get("Last-Modified") == "" {
		t.Error("GET: expected non-empty Last-Modified header")
	}
	if w.Header().Get("Content-Length") != "5" {
		t.Errorf("GET: expected Content-Length 5, got %q", w.Header().Get("Content-Length"))
	}
}

// ---- ListObjectsV2 ----

func TestS3_ListObjectsV2_EmptyBucket(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "empty-list-bucket")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/empty-list-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("LIST: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "<KeyCount>0</KeyCount>") {
		t.Errorf("expected KeyCount 0 in body, got:\n%s", body)
	}
}

func TestS3_ListObjectsV2_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// LIST objects in nonexistent bucket → 404
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/nope-bucket"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("LIST nonexistent bucket: expected 404, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_ListObjectsV2_Delimiter(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "delim-bucket")

	mustPutObject(t, handler, "delim-bucket", "photos/2024/jan.jpg", "image/jpeg", "j")
	mustPutObject(t, handler, "delim-bucket", "photos/2024/feb.jpg", "image/jpeg", "f")
	mustPutObject(t, handler, "delim-bucket", "photos/2025/mar.jpg", "image/jpeg", "m")
	mustPutObject(t, handler, "delim-bucket", "docs/readme.txt", "text/plain", "r")

	// LIST with delimiter=/ and prefix=photos/ should collapse into CommonPrefixes
	req := s3Req(t, http.MethodGet, "/delim-bucket?prefix=photos/&delimiter=/")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("LIST delimiter: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "<Prefix>photos/2024/</Prefix>") {
		t.Errorf("expected CommonPrefix photos/2024/ in body, got:\n%s", body)
	}
	if !strings.Contains(body, "<Prefix>photos/2025/</Prefix>") {
		t.Errorf("expected CommonPrefix photos/2025/ in body, got:\n%s", body)
	}
	// Should NOT contain actual object keys as Contents
	if strings.Contains(body, "<Key>photos/2024/jan.jpg</Key>") {
		t.Errorf("did not expect individual object keys when delimiter groups them, got:\n%s", body)
	}
}

func TestS3_ListObjectsV2_MaxKeys(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "maxkeys-bucket")

	for i := 0; i < 5; i++ {
		mustPutObject(t, handler, "maxkeys-bucket", fmt.Sprintf("key-%02d.txt", i),
			"text/plain", fmt.Sprintf("content-%d", i))
	}

	// LIST with max-keys=2 → should be truncated
	req := s3Req(t, http.MethodGet, "/maxkeys-bucket?max-keys=2")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("LIST max-keys: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "<IsTruncated>true</IsTruncated>") {
		t.Errorf("expected IsTruncated true, got:\n%s", body)
	}
	if !strings.Contains(body, "<MaxKeys>2</MaxKeys>") {
		t.Errorf("expected MaxKeys 2, got:\n%s", body)
	}
}

func TestS3_ListObjectsV2_ContinuationToken(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "cont-bucket")

	for i := 0; i < 5; i++ {
		mustPutObject(t, handler, "cont-bucket", fmt.Sprintf("item-%02d.txt", i),
			"text/plain", fmt.Sprintf("data-%d", i))
	}

	type listResult struct {
		XMLName               xml.Name `xml:"ListBucketResult"`
		IsTruncated           bool     `xml:"IsTruncated"`
		NextContinuationToken string   `xml:"NextContinuationToken"`
		Contents              []struct {
			Key string `xml:"Key"`
		} `xml:"Contents"`
	}

	// Paginate through all results collecting keys
	allKeys := make(map[string]bool)
	token := ""
	pageCount := 0

	for {
		url := "/cont-bucket?max-keys=2"
		if token != "" {
			url += "&continuation-token=" + token
		}
		req := s3Req(t, http.MethodGet, url)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("page %d: expected 200, got %d", pageCount+1, w.Code)
		}

		var page listResult
		if err := xml.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatalf("failed to parse page %d: %v", pageCount+1, err)
		}

		// First page must have items and be truncated
		if pageCount == 0 {
			if len(page.Contents) != 2 {
				t.Fatalf("expected 2 items on page 1, got %d", len(page.Contents))
			}
			if !page.IsTruncated {
				t.Fatal("expected page 1 to be truncated")
			}
		}

		for _, c := range page.Contents {
			allKeys[c.Key] = true
		}

		pageCount++
		if !page.IsTruncated || page.NextContinuationToken == "" {
			break
		}
		token = page.NextContinuationToken

		if pageCount > 10 {
			t.Fatal("too many pages, possible infinite loop")
		}
	}

	// Must have paginated (more than 1 page)
	if pageCount < 2 {
		t.Errorf("expected multiple pages, got %d", pageCount)
	}
	// All collected keys must be unique (no overlap between pages)
	if len(allKeys) == 0 {
		t.Fatal("expected items across pages, got 0")
	}
}

// ---- CopyObject Error Cases ----

func TestS3_CopyObject_InvalidSource(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "copy-err-bucket")

	// Copy with no slash in source → 400 InvalidArgument
	req := httptest.NewRequest(http.MethodPut, "/copy-err-bucket/dest.txt", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("x-amz-copy-source", "no-slash-here")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("COPY invalid source: expected 400, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "InvalidArgument") {
		t.Errorf("expected InvalidArgument in body, got:\n%s", w.Body.String())
	}
}

func TestS3_CopyObject_SourceKeyNotFound(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "copy-src-bucket")
	mustCreateBucket(t, handler, "copy-dst-bucket")

	// Copy from nonexistent key → 404 NoSuchKey
	req := httptest.NewRequest(http.MethodPut, "/copy-dst-bucket/dest.txt", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("x-amz-copy-source", "/copy-src-bucket/nonexistent.txt")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("COPY nonexistent key: expected 404, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchKey") {
		t.Errorf("expected NoSuchKey in body, got:\n%s", w.Body.String())
	}
}

func TestS3_CopyObject_SourceBucketNotFound(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "copy-only-dst")

	// Copy from nonexistent source bucket → 404 NoSuchBucket
	req := httptest.NewRequest(http.MethodPut, "/copy-only-dst/dest.txt", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("x-amz-copy-source", "/nonexistent-src-bucket/file.txt")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("COPY nonexistent src bucket: expected 404, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_CopyObject_SameBucket(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "same-bucket")
	mustPutObject(t, handler, "same-bucket", "original.txt", "text/plain", "copy me")

	// Copy within same bucket
	req := httptest.NewRequest(http.MethodPut, "/same-bucket/copy.txt", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("x-amz-copy-source", "/same-bucket/original.txt")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("COPY same bucket: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	// Verify copy
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/same-bucket/copy.txt"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET copy: expected 200, got %d", w.Code)
	}
	if w.Body.String() != "copy me" {
		t.Errorf("GET copy: expected %q, got %q", "copy me", w.Body.String())
	}
}

// ---- Multipart Error Cases ----

func TestS3_CreateMultipartUpload_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// POST to nonexistent bucket → 404 NoSuchBucket
	req := s3Req(t, http.MethodPost, "/nonexistent-bucket/file.bin?uploads")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("CreateMultipartUpload nonexistent bucket: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_UploadPart_NoSuchUpload(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "part-err-bucket")

	// UploadPart with nonexistent uploadId → 404 NoSuchUpload
	req := s3ReqWithBody(t, http.MethodPut,
		"/part-err-bucket/file.bin?uploadId=fake-upload-id&partNumber=1", "data")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("UploadPart nonexistent upload: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchUpload") {
		t.Errorf("expected NoSuchUpload in body, got:\n%s", w.Body.String())
	}
}

func TestS3_AbortMultipartUpload_NoSuchUpload(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "abort-err-bucket")

	// Abort nonexistent upload → 404 NoSuchUpload
	req := s3Req(t, http.MethodDelete, "/abort-err-bucket/file.bin?uploadId=fake-id")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("AbortMultipartUpload nonexistent: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchUpload") {
		t.Errorf("expected NoSuchUpload in body, got:\n%s", w.Body.String())
	}
}

func TestS3_CompleteMultipartUpload_NoSuchUpload(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "complete-err-bucket")

	// Complete with nonexistent uploadId → 404 NoSuchUpload
	completeXML := `<CompleteMultipartUpload><Part><PartNumber>1</PartNumber><ETag>"abc"</ETag></Part></CompleteMultipartUpload>`
	req := s3ReqWithBody(t, http.MethodPost,
		"/complete-err-bucket/file.bin?uploadId=fake-id", completeXML)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("CompleteMultipartUpload nonexistent: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchUpload") {
		t.Errorf("expected NoSuchUpload in body, got:\n%s", w.Body.String())
	}
}

func TestS3_CompleteMultipartUpload_InvalidPart(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "invpart-bucket")

	// Create upload
	req := s3Req(t, http.MethodPost, "/invpart-bucket/file.bin?uploads")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CreateMultipartUpload: expected 200, got %d", w.Code)
	}

	var initResult initMultipartUploadResult
	if err := xml.Unmarshal(w.Body.Bytes(), &initResult); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Upload only part 1
	req = s3ReqWithBody(t, http.MethodPut,
		fmt.Sprintf("/invpart-bucket/file.bin?uploadId=%s&partNumber=1", initResult.UploadId),
		"part1-data")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UploadPart: expected 200, got %d", w.Code)
	}

	// Complete with part 1 and nonexistent part 2 → 400 InvalidPart
	completeXML := fmt.Sprintf(`<CompleteMultipartUpload>
		<Part><PartNumber>1</PartNumber><ETag>"x"</ETag></Part>
		<Part><PartNumber>2</PartNumber><ETag>"y"</ETag></Part>
	</CompleteMultipartUpload>`)
	req = s3ReqWithBody(t, http.MethodPost,
		fmt.Sprintf("/invpart-bucket/file.bin?uploadId=%s", initResult.UploadId), completeXML)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("CompleteMultipartUpload invalid part: expected 400, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "InvalidPart") {
		t.Errorf("expected InvalidPart in body, got:\n%s", w.Body.String())
	}
}

func TestS3_CompleteMultipartUpload_MalformedXML(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "malxml-bucket")

	// Create upload
	req := s3Req(t, http.MethodPost, "/malxml-bucket/file.bin?uploads")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CreateMultipartUpload: expected 200, got %d", w.Code)
	}

	var initResult initMultipartUploadResult
	if err := xml.Unmarshal(w.Body.Bytes(), &initResult); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Complete with malformed XML → 400 MalformedXML
	req = s3ReqWithBody(t, http.MethodPost,
		fmt.Sprintf("/malxml-bucket/file.bin?uploadId=%s", initResult.UploadId), "not xml at all{{{")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("CompleteMultipartUpload malformed XML: expected 400, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "MalformedXML") {
		t.Errorf("expected MalformedXML in body, got:\n%s", w.Body.String())
	}
}

func TestS3_ListMultipartUploads_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	// LIST multipart uploads on nonexistent bucket → 404
	req := s3Req(t, http.MethodGet, "/ghost-mp-bucket?uploads")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("ListMultipartUploads nonexistent: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_ListMultipartUploads_Empty(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "empty-mp-bucket")

	// LIST multipart uploads with none pending → 200 with empty list
	req := s3Req(t, http.MethodGet, "/empty-mp-bucket?uploads")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListMultipartUploads empty: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	var result listMultipartUploadsResponse
	if err := xml.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if len(result.Uploads) != 0 {
		t.Errorf("expected 0 uploads, got %d", len(result.Uploads))
	}
}

func TestS3_ListMultipartUploads_MultipleUploads(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "multi-mp-bucket")

	// Create 3 multipart uploads
	for _, key := range []string{"file1.bin", "file2.bin", "file3.bin"} {
		req := s3Req(t, http.MethodPost, "/multi-mp-bucket/"+key+"?uploads")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("CreateMultipartUpload %s: expected 200, got %d", key, w.Code)
		}
	}

	// List all uploads
	req := s3Req(t, http.MethodGet, "/multi-mp-bucket?uploads")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListMultipartUploads: expected 200, got %d", w.Code)
	}

	var result listMultipartUploadsResponse
	if err := xml.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if len(result.Uploads) != 3 {
		t.Fatalf("expected 3 uploads, got %d", len(result.Uploads))
	}

	// Verify each upload has a unique ID
	ids := make(map[string]bool)
	for _, u := range result.Uploads {
		if u.UploadId == "" {
			t.Error("expected non-empty UploadId")
		}
		if ids[u.UploadId] {
			t.Errorf("duplicate UploadId: %s", u.UploadId)
		}
		ids[u.UploadId] = true
	}
}

func TestS3_UploadPart_InvalidPartNumber(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "badpart-bucket")

	// Create upload
	req := s3Req(t, http.MethodPost, "/badpart-bucket/file.bin?uploads")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CreateMultipartUpload: expected 200, got %d", w.Code)
	}

	var initResult initMultipartUploadResult
	if err := xml.Unmarshal(w.Body.Bytes(), &initResult); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Upload with non-numeric partNumber → 400 InvalidArgument
	req = s3ReqWithBody(t, http.MethodPut,
		fmt.Sprintf("/badpart-bucket/file.bin?uploadId=%s&partNumber=abc", initResult.UploadId),
		"data")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UploadPart bad partNumber: expected 400, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "InvalidArgument") {
		t.Errorf("expected InvalidArgument in body, got:\n%s", w.Body.String())
	}
}

// ---- Versioning Error Cases ----

func TestS3_PutBucketVersioning_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	body := `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Enabled</Status></VersioningConfiguration>`
	req := s3ReqWithBody(t, http.MethodPut, "/nonexistent-bucket?versioning", body)
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("PutBucketVersioning nonexistent: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_GetBucketVersioning_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	req := s3Req(t, http.MethodGet, "/nonexistent-bucket?versioning")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("GetBucketVersioning nonexistent: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_ListObjectVersions_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	req := s3Req(t, http.MethodGet, "/nonexistent-bucket?versions")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("ListObjectVersions nonexistent: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_PutBucketVersioning_MalformedXML(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "ver-malformed-bucket")

	// Malformed XML → 400
	req := s3ReqWithBody(t, http.MethodPut, "/ver-malformed-bucket?versioning", "not xml{{{")
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("PutBucketVersioning malformed: expected 400, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "MalformedXML") {
		t.Errorf("expected MalformedXML in body, got:\n%s", w.Body.String())
	}
}

func TestS3_PutBucketVersioning_InvalidStatus(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "ver-invalid-bucket")

	// Invalid status value → 400
	body := `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>InvalidValue</Status></VersioningConfiguration>`
	req := s3ReqWithBody(t, http.MethodPut, "/ver-invalid-bucket?versioning", body)
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("PutBucketVersioning invalid status: expected 400, got %d – %s",
			w.Code, w.Body.String())
	}
}

func TestS3_Versioning_SuspendedBehavior(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "susp-bucket")
	enableVersioning(t, handler, "susp-bucket")

	// Put a versioned object
	mustPutObject(t, handler, "susp-bucket", "file.txt", "text/plain", "v1")

	// Suspend versioning
	body := `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Suspended</Status></VersioningConfiguration>`
	req := s3ReqWithBody(t, http.MethodPut, "/susp-bucket?versioning", body)
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Suspend versioning: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	// Verify status is Suspended
	req = s3Req(t, http.MethodGet, "/susp-bucket?versioning")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET versioning: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<Status>Suspended</Status>") {
		t.Errorf("expected Suspended status, got: %s", w.Body.String())
	}
}

func TestS3_ListObjectVersions_WithPrefix(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "ver-prefix-bucket")
	enableVersioning(t, handler, "ver-prefix-bucket")

	mustPutObject(t, handler, "ver-prefix-bucket", "logs/a.txt", "text/plain", "log-a")
	mustPutObject(t, handler, "ver-prefix-bucket", "logs/b.txt", "text/plain", "log-b")
	mustPutObject(t, handler, "ver-prefix-bucket", "data/c.txt", "text/plain", "data-c")

	// List versions with prefix=logs/ → should only see logs
	req := s3Req(t, http.MethodGet, "/ver-prefix-bucket?versions&prefix=logs/")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListObjectVersions prefix: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "logs/a.txt") {
		t.Errorf("expected logs/a.txt in versions, got:\n%s", body)
	}
	if !strings.Contains(body, "logs/b.txt") {
		t.Errorf("expected logs/b.txt in versions, got:\n%s", body)
	}
	if strings.Contains(body, "data/c.txt") {
		t.Errorf("did not expect data/c.txt with prefix=logs/, got:\n%s", body)
	}
}

// ---- Bucket Policy Error Cases ----

func TestS3_GetBucketPolicy_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	req := s3Req(t, http.MethodGet, "/nonexistent-bucket?policy")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("GetBucketPolicy nonexistent bucket: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_GetBucketPolicy_NoPolicySet(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "nopol-bucket")

	// GET policy when none set → 404 NoSuchBucketPolicy
	req := s3Req(t, http.MethodGet, "/nopol-bucket?policy")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("GetBucketPolicy no policy: expected 404, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucketPolicy") {
		t.Errorf("expected NoSuchBucketPolicy in body, got:\n%s", w.Body.String())
	}
}

func TestS3_PutBucketPolicy_EmptyBody(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "empty-pol-bucket")

	// PUT policy with empty body → 400
	req := s3Req(t, http.MethodPut, "/empty-pol-bucket?policy")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("PutBucketPolicy empty: expected 400, got %d – %s", w.Code, w.Body.String())
	}
}

func TestS3_PutBucketPolicy_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	req := s3ReqWithBody(t, http.MethodPut, "/nonexistent-bucket?policy", `{"Version":"2012-10-17"}`)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("PutBucketPolicy nonexistent: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

func TestS3_DeleteBucketPolicy_NoSuchBucket(t *testing.T) {
	handler := newS3Gateway(t)

	req := s3Req(t, http.MethodDelete, "/nonexistent-bucket?policy")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("DeleteBucketPolicy nonexistent: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected NoSuchBucket in body, got:\n%s", w.Body.String())
	}
}

// ---- Versioned object operations ----

func TestS3_Versioning_PutReturnsVersionId(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "verid-bucket")
	enableVersioning(t, handler, "verid-bucket")

	// PUT object with versioning → response should have x-amz-version-id
	req := httptest.NewRequest(http.MethodPut, "/verid-bucket/versioned.txt", strings.NewReader("data"))
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT versioned: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	if w.Header().Get("x-amz-version-id") == "" {
		t.Error("expected x-amz-version-id header on versioned PUT")
	}
}

func TestS3_Versioning_NonVersionedNoVersionId(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "noverid-bucket")

	// PUT without versioning → no x-amz-version-id
	mustPutObject(t, handler, "noverid-bucket", "plain.txt", "text/plain", "data")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/noverid-bucket/plain.txt"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d", w.Code)
	}
	if w.Header().Get("x-amz-version-id") != "" {
		t.Error("did not expect x-amz-version-id on non-versioned object")
	}
}

func TestS3_ListObjectVersions_EmptyBucket(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "ver-empty-bucket")
	enableVersioning(t, handler, "ver-empty-bucket")

	req := s3Req(t, http.MethodGet, "/ver-empty-bucket?versions")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListObjectVersions empty: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "ListVersionsResult") {
		t.Errorf("expected ListVersionsResult in body, got:\n%s", body)
	}
	// Should have no <Version> elements
	if strings.Contains(body, "<Version>") {
		t.Errorf("expected no Version elements in empty bucket, got:\n%s", body)
	}
}

// ---- Object with nested keys ----

func TestS3_PutGetObject_NestedKey(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "nested-bucket")

	content := "deeply nested content"
	mustPutObject(t, handler, "nested-bucket", "a/b/c/d/deep.txt", "text/plain", content)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/nested-bucket/a/b/c/d/deep.txt"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET nested key: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	if w.Body.String() != content {
		t.Errorf("GET nested key: expected %q, got %q", content, w.Body.String())
	}
}

func TestS3_PutGetObject_EmptyBody(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "empty-body-bucket")

	// PUT with empty body (valid in S3 for directory markers)
	req := s3ReqWithBody(t, http.MethodPut, "/empty-body-bucket/marker/", "")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT empty body: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	// GET should return empty body
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/empty-body-bucket/marker/"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET empty body: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Errorf("GET empty body: expected empty response, got %q", w.Body.String())
	}
}

func TestS3_PutGetObject_LargeBody(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "large-bucket")

	// 1 MB object
	content := strings.Repeat("x", 1024*1024)
	mustPutObject(t, handler, "large-bucket", "big.bin", "application/octet-stream", content)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/large-bucket/big.bin"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET large: expected 200, got %d", w.Code)
	}
	if w.Body.Len() != 1024*1024 {
		t.Errorf("GET large: expected %d bytes, got %d", 1024*1024, w.Body.Len())
	}
}

// ---- CopyObject XML response structure ----

func TestS3_CopyObject_XMLResponse(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "copyxml-bucket")
	mustPutObject(t, handler, "copyxml-bucket", "src.txt", "text/plain", "source data")

	req := httptest.NewRequest(http.MethodPut, "/copyxml-bucket/dst.txt", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("x-amz-copy-source", "/copyxml-bucket/src.txt")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("COPY: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	type copyResult struct {
		XMLName      xml.Name `xml:"CopyObjectResult"`
		ETag         string   `xml:"ETag"`
		LastModified string   `xml:"LastModified"`
	}
	var result copyResult
	if err := xml.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse CopyObjectResult: %v\n%s", err, w.Body.String())
	}
	if result.ETag == "" {
		t.Error("expected non-empty ETag in CopyObjectResult")
	}
	if result.LastModified == "" {
		t.Error("expected non-empty LastModified in CopyObjectResult")
	}
}

// ---- ListParts error case ----

func TestS3_ListParts_NoSuchUpload(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "listparts-err-bucket")

	req := s3Req(t, http.MethodGet, "/listparts-err-bucket/file.bin?uploadId=nonexistent-id")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("ListParts nonexistent upload: expected 404, got %d – %s",
			w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "NoSuchUpload") {
		t.Errorf("expected NoSuchUpload in body, got:\n%s", w.Body.String())
	}
}

// ---- Multipart upload creates correct final object ----

func TestS3_MultipartUpload_ObjectContentType(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "mp-ct-bucket")

	// Create multipart upload
	req := s3Req(t, http.MethodPost, "/mp-ct-bucket/file.bin?uploads")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CreateMultipartUpload: expected 200, got %d", w.Code)
	}

	var initResult initMultipartUploadResult
	if err := xml.Unmarshal(w.Body.Bytes(), &initResult); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Upload 1 part
	req = s3ReqWithBody(t, http.MethodPut,
		fmt.Sprintf("/mp-ct-bucket/file.bin?uploadId=%s&partNumber=1", initResult.UploadId),
		"part-data")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UploadPart: expected 200, got %d", w.Code)
	}
	etag := w.Header().Get("ETag")

	// Complete
	completeXML := fmt.Sprintf(`<CompleteMultipartUpload><Part><PartNumber>1</PartNumber><ETag>%s</ETag></Part></CompleteMultipartUpload>`, etag)
	req = s3ReqWithBody(t, http.MethodPost,
		fmt.Sprintf("/mp-ct-bucket/file.bin?uploadId=%s", initResult.UploadId), completeXML)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Complete: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	// HEAD the completed object → should have default content type
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/mp-ct-bucket/file.bin"))
	if w.Code != http.StatusOK {
		t.Fatalf("HEAD: expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("expected Content-Type application/octet-stream, got %q", ct)
	}
}

// ---- CreateBucket then HEAD confirms existence ----

func TestS3_CreateBucket_ThenHeadConfirms(t *testing.T) {
	handler := newS3Gateway(t)

	// Create
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/confirm-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("PUT: expected 200, got %d", w.Code)
	}

	// HEAD immediately confirms
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/confirm-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("HEAD: expected 200, got %d", w.Code)
	}
}

// ---- Delete bucket after clearing objects ----

func TestS3_DeleteBucket_AfterClearingObjects(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "clearable-bucket")
	mustPutObject(t, handler, "clearable-bucket", "file.txt", "text/plain", "data")

	// Try delete (should fail)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/clearable-bucket"))
	if w.Code != http.StatusConflict {
		t.Fatalf("DELETE non-empty: expected 409, got %d", w.Code)
	}

	// Delete the object
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/clearable-bucket/file.txt"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE object: expected 204, got %d", w.Code)
	}

	// Now delete bucket (should succeed)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/clearable-bucket"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE empty bucket: expected 204, got %d – %s", w.Code, w.Body.String())
	}

	// HEAD confirms gone
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/clearable-bucket"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("HEAD after delete: expected 404, got %d", w.Code)
	}
}
