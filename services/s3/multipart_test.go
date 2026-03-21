package s3_test

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// initMultipartUploadResult is used for parsing the XML response.
type initMultipartUploadResult struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadId string   `xml:"UploadId"`
}

type listMultipartUploadsResponse struct {
	XMLName xml.Name       `xml:"ListMultipartUploadsResult"`
	Bucket  string         `xml:"Bucket"`
	Uploads []uploadEntry  `xml:"Upload"`
}

type uploadEntry struct {
	Key      string `xml:"Key"`
	UploadId string `xml:"UploadId"`
}

type listPartsResponse struct {
	XMLName  xml.Name    `xml:"ListPartsResult"`
	Bucket   string      `xml:"Bucket"`
	Key      string      `xml:"Key"`
	UploadId string      `xml:"UploadId"`
	Parts    []partEntry `xml:"Part"`
}

type partEntry struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
	Size       int64  `xml:"Size"`
}

// TestS3_MultipartUpload_FullCycle tests create → upload parts → complete → get object.
func TestS3_MultipartUpload_FullCycle(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "mp-bucket")

	// 1. Create multipart upload: POST /mp-bucket/big-file.bin?uploads
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mp-bucket/big-file.bin?uploads", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("CreateMultipartUpload: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	var initResult initMultipartUploadResult
	if err := xml.Unmarshal(w.Body.Bytes(), &initResult); err != nil {
		t.Fatalf("Failed to parse InitiateMultipartUploadResult: %v", err)
	}
	if initResult.UploadId == "" {
		t.Fatal("CreateMultipartUpload: expected non-empty UploadId")
	}
	if initResult.Bucket != "mp-bucket" {
		t.Errorf("CreateMultipartUpload: expected Bucket mp-bucket, got %s", initResult.Bucket)
	}
	if initResult.Key != "big-file.bin" {
		t.Errorf("CreateMultipartUpload: expected Key big-file.bin, got %s", initResult.Key)
	}

	uploadId := initResult.UploadId

	// 2. Upload 3 parts.
	parts := []string{"AAAA", "BBBB", "CCCC"}
	etags := make([]string, 3)
	for i, content := range parts {
		partNum := i + 1
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPut,
			fmt.Sprintf("/mp-bucket/big-file.bin?uploadId=%s&partNumber=%d", uploadId, partNum),
			strings.NewReader(content))
		req.Header.Set("Authorization",
			"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("UploadPart %d: expected 200, got %d – %s", partNum, w.Code, w.Body.String())
		}
		etag := w.Header().Get("ETag")
		if etag == "" {
			t.Fatalf("UploadPart %d: expected non-empty ETag", partNum)
		}
		etags[i] = etag
	}

	// 3. Complete multipart upload.
	completeXML := fmt.Sprintf(`<CompleteMultipartUpload>
		<Part><PartNumber>1</PartNumber><ETag>%s</ETag></Part>
		<Part><PartNumber>2</PartNumber><ETag>%s</ETag></Part>
		<Part><PartNumber>3</PartNumber><ETag>%s</ETag></Part>
	</CompleteMultipartUpload>`, etags[0], etags[1], etags[2])

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/mp-bucket/big-file.bin?uploadId=%s", uploadId),
		strings.NewReader(completeXML))
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("CompleteMultipartUpload: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "CompleteMultipartUploadResult") {
		t.Errorf("CompleteMultipartUpload: expected CompleteMultipartUploadResult in body\n%s", w.Body.String())
	}

	// 4. GET the completed object – should be concatenated.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/mp-bucket/big-file.bin"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	expected := "AAAABBBBCCCC"
	if got := w.Body.String(); got != expected {
		t.Errorf("GET: expected %q, got %q", expected, got)
	}
}

// TestS3_AbortMultipartUpload tests that aborting removes the upload from listings.
func TestS3_AbortMultipartUpload(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "abort-bucket")

	// Create a multipart upload.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/abort-bucket/temp-file.bin?uploads", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CreateMultipartUpload: expected 200, got %d", w.Code)
	}

	var initResult initMultipartUploadResult
	if err := xml.Unmarshal(w.Body.Bytes(), &initResult); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	uploadId := initResult.UploadId

	// Abort the upload: DELETE /abort-bucket/temp-file.bin?uploadId=X
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/abort-bucket/temp-file.bin?uploadId=%s", uploadId), nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("AbortMultipartUpload: expected 204, got %d – %s", w.Code, w.Body.String())
	}

	// ListMultipartUploads should show empty.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/abort-bucket?uploads", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListMultipartUploads: expected 200, got %d", w.Code)
	}

	var listResult listMultipartUploadsResponse
	if err := xml.Unmarshal(w.Body.Bytes(), &listResult); err != nil {
		t.Fatalf("Failed to parse ListMultipartUploadsResult: %v", err)
	}
	if len(listResult.Uploads) != 0 {
		t.Errorf("ListMultipartUploads: expected 0 uploads, got %d", len(listResult.Uploads))
	}
}

// TestS3_ListParts tests that uploaded parts are returned correctly.
func TestS3_ListParts(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "parts-bucket")

	// Create upload.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/parts-bucket/partfile.bin?uploads", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CreateMultipartUpload: expected 200, got %d", w.Code)
	}

	var initResult initMultipartUploadResult
	if err := xml.Unmarshal(w.Body.Bytes(), &initResult); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	uploadId := initResult.UploadId

	// Upload 2 parts.
	for i, content := range []string{"part-one", "part-two"} {
		partNum := i + 1
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPut,
			fmt.Sprintf("/parts-bucket/partfile.bin?uploadId=%s&partNumber=%d", uploadId, partNum),
			strings.NewReader(content))
		req.Header.Set("Authorization",
			"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("UploadPart %d: expected 200, got %d", partNum, w.Code)
		}
	}

	// List parts: GET /parts-bucket/partfile.bin?uploadId=X
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/parts-bucket/partfile.bin?uploadId=%s", uploadId), nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListParts: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	var partsResult listPartsResponse
	if err := xml.Unmarshal(w.Body.Bytes(), &partsResult); err != nil {
		t.Fatalf("Failed to parse ListPartsResult: %v", err)
	}
	if len(partsResult.Parts) != 2 {
		t.Fatalf("ListParts: expected 2 parts, got %d", len(partsResult.Parts))
	}
	if partsResult.Parts[0].PartNumber != 1 {
		t.Errorf("ListParts: expected part 1, got %d", partsResult.Parts[0].PartNumber)
	}
	if partsResult.Parts[1].PartNumber != 2 {
		t.Errorf("ListParts: expected part 2, got %d", partsResult.Parts[1].PartNumber)
	}
	if partsResult.Parts[0].Size != int64(len("part-one")) {
		t.Errorf("ListParts: expected part 1 size %d, got %d", len("part-one"), partsResult.Parts[0].Size)
	}
}

// TestS3_PresignedURL_GetObject verifies that presigned URL requests (with
// X-Amz-Algorithm query param) are accepted and route to the correct handler.
func TestS3_PresignedURL_GetObject(t *testing.T) {
	handler := newS3Gateway(t)
	mustCreateBucket(t, handler, "presigned-bucket")
	mustPutObject(t, handler, "presigned-bucket", "secret.txt", "text/plain", "presigned content")

	// GET with presigned URL query params (no Authorization header).
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/presigned-bucket/secret.txt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request&X-Amz-Signature=fake&X-Amz-Date=20240101T000000Z&X-Amz-Expires=3600&X-Amz-SignedHeaders=host",
		nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Presigned GET: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != "presigned content" {
		t.Errorf("Presigned GET: expected %q, got %q", "presigned content", got)
	}
}
