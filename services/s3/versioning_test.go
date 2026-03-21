package s3_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// enableVersioning enables versioning on a bucket.
func enableVersioning(t *testing.T, handler http.Handler, bucket string) {
	t.Helper()
	body := `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Enabled</Status></VersioningConfiguration>`
	req := httptest.NewRequest(http.MethodPut, "/"+bucket+"?versioning", strings.NewReader(body))
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT ?versioning: expected 200, got %d – %s", w.Code, w.Body.String())
	}
}

func TestS3_PutGetBucketVersioning(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "ver-bucket")

	// GET versioning before enabling → empty Status
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ver-bucket?versioning", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET ?versioning: expected 200, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "<Status>Enabled</Status>") {
		t.Error("expected versioning to not be enabled yet")
	}

	// Enable versioning
	enableVersioning(t, handler, "ver-bucket")

	// GET versioning after enabling → Enabled
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ver-bucket?versioning", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET ?versioning: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<Status>Enabled</Status>") {
		t.Errorf("expected Enabled in versioning response, got: %s", w.Body.String())
	}
}

func TestS3_Versioning_PutSameKeyTwice_ListVersions(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "vbucket")
	enableVersioning(t, handler, "vbucket")

	// Put same key twice
	mustPutObject(t, handler, "vbucket", "doc.txt", "text/plain", "version1")
	mustPutObject(t, handler, "vbucket", "doc.txt", "text/plain", "version2")

	// ListObjectVersions
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/vbucket?versions", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET ?versions: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Should contain 2 <Version> elements
	type listVersionsResult struct {
		XMLName  xml.Name `xml:"ListVersionsResult"`
		Versions []struct {
			Key       string `xml:"Key"`
			VersionId string `xml:"VersionId"`
			IsLatest  bool   `xml:"IsLatest"`
		} `xml:"Version"`
	}
	var result listVersionsResult
	if err := xml.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("failed to parse ListVersionsResult: %v\nbody: %s", err, body)
	}
	if len(result.Versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(result.Versions))
	}

	// First version should be latest
	if !result.Versions[0].IsLatest {
		t.Error("expected first version to be latest")
	}
	if result.Versions[1].IsLatest {
		t.Error("expected second version to NOT be latest")
	}

	// VersionIds should be different
	if result.Versions[0].VersionId == result.Versions[1].VersionId {
		t.Error("expected different version IDs")
	}
}

func TestS3_Versioning_GetObjectWithVersionId(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "vget-bucket")
	enableVersioning(t, handler, "vget-bucket")

	// Put two versions
	mustPutObject(t, handler, "vget-bucket", "file.txt", "text/plain", "v1-content")
	mustPutObject(t, handler, "vget-bucket", "file.txt", "text/plain", "v2-content")

	// Get latest (without versionId) → v2-content
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/vget-bucket/file.txt"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET latest: expected 200, got %d", w.Code)
	}
	if w.Body.String() != "v2-content" {
		t.Errorf("GET latest: expected v2-content, got %q", w.Body.String())
	}

	// List versions to get the first version's ID
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/vget-bucket?versions", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)

	type listVersionsResult struct {
		XMLName  xml.Name `xml:"ListVersionsResult"`
		Versions []struct {
			Key       string `xml:"Key"`
			VersionId string `xml:"VersionId"`
			IsLatest  bool   `xml:"IsLatest"`
		} `xml:"Version"`
	}
	var result listVersionsResult
	if err := xml.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse versions: %v", err)
	}

	// Find the non-latest version
	var olderVersionId string
	for _, v := range result.Versions {
		if !v.IsLatest {
			olderVersionId = v.VersionId
			break
		}
	}
	if olderVersionId == "" {
		t.Fatal("could not find older version")
	}

	// Get with specific versionId → v1-content
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/vget-bucket/file.txt?versionId="+olderVersionId, nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET versionId: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "v1-content" {
		t.Errorf("GET versionId: expected v1-content, got %q", w.Body.String())
	}
}

func TestS3_Versioning_DeleteCreatesMarker(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "vdel-bucket")
	enableVersioning(t, handler, "vdel-bucket")

	mustPutObject(t, handler, "vdel-bucket", "data.txt", "text/plain", "content")

	// DELETE with versioning → should create delete marker
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/vdel-bucket/data.txt"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE: expected 204, got %d", w.Code)
	}
	if w.Header().Get("x-amz-delete-marker") != "true" {
		t.Error("expected x-amz-delete-marker: true header")
	}
	if w.Header().Get("x-amz-version-id") == "" {
		t.Error("expected x-amz-version-id header on delete marker")
	}

	// GET should now return 404
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/vdel-bucket/data.txt"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET after delete: expected 404, got %d", w.Code)
	}

	// ListObjectVersions should show both the version AND the delete marker
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/vdel-bucket?versions", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "<Version>") {
		t.Error("expected <Version> in ListVersionsResult")
	}
	if !strings.Contains(body, "<DeleteMarker>") {
		t.Error("expected <DeleteMarker> in ListVersionsResult")
	}
}

func TestS3_BucketPolicy_PutGetDelete(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "pol-bucket")

	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::pol-bucket/*"}]}`

	// PUT policy
	req := httptest.NewRequest(http.MethodPut, "/pol-bucket?policy", strings.NewReader(policy))
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("PUT policy: expected 204, got %d – %s", w.Code, w.Body.String())
	}

	// GET policy
	req = httptest.NewRequest(http.MethodGet, "/pol-bucket?policy", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET policy: expected 200, got %d", w.Code)
	}
	if w.Body.String() != policy {
		t.Errorf("GET policy: expected %q, got %q", policy, w.Body.String())
	}

	// DELETE policy
	req = httptest.NewRequest(http.MethodDelete, "/pol-bucket?policy", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE policy: expected 204, got %d", w.Code)
	}

	// GET policy after delete → 404
	req = httptest.NewRequest(http.MethodGet, "/pol-bucket?policy", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET policy after delete: expected 404, got %d", w.Code)
	}
}
