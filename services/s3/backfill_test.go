package s3_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---- Bucket Tagging (TDD: tests written first) ----

func TestS3_BucketTagging(t *testing.T) {
	handler := newS3Gateway(t)

	// Create bucket
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/tag-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("create bucket: %d", w.Code)
	}

	// PUT tagging
	taggingXML := `<Tagging><TagSet><Tag><Key>env</Key><Value>dev</Value></Tag><Tag><Key>team</Key><Value>platform</Value></Tag></TagSet></Tagging>`
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/tag-bucket?tagging", taggingXML))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("PUT tagging: expected 200/204, got %d body: %s", w.Code, w.Body.String())
	}

	// GET tagging
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/tag-bucket?tagging"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET tagging: %d body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "env") || !strings.Contains(body, "dev") {
		t.Errorf("GET tagging: expected tags in response, got: %s", body)
	}

	// DELETE tagging
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/tag-bucket?tagging"))
	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("DELETE tagging: %d", w.Code)
	}

	// GET tagging after delete — should be empty
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/tag-bucket?tagging"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET tagging after delete: %d", w.Code)
	}
}

// ---- Object Tagging ----

func TestS3_ObjectTagging(t *testing.T) {
	handler := newS3Gateway(t)

	// Create bucket + object
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/obj-tag-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("create bucket: %d", w.Code)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/obj-tag-bucket/myfile.txt", "hello"))
	if w.Code != http.StatusOK {
		t.Fatalf("put object: %d", w.Code)
	}

	// PUT object tagging
	taggingXML := `<Tagging><TagSet><Tag><Key>classification</Key><Value>public</Value></Tag></TagSet></Tagging>`
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/obj-tag-bucket/myfile.txt?tagging", taggingXML))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("PUT object tagging: %d body: %s", w.Code, w.Body.String())
	}

	// GET object tagging
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/obj-tag-bucket/myfile.txt?tagging"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET object tagging: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "classification") {
		t.Errorf("GET object tagging: expected tags, got: %s", w.Body.String())
	}
}

// ---- Bucket CORS ----

func TestS3_BucketCORS(t *testing.T) {
	handler := newS3Gateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/cors-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("create bucket: %d", w.Code)
	}

	// PUT CORS
	corsXML := `<CORSConfiguration><CORSRule><AllowedOrigin>*</AllowedOrigin><AllowedMethod>GET</AllowedMethod><AllowedMethod>PUT</AllowedMethod></CORSRule></CORSConfiguration>`
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/cors-bucket?cors", corsXML))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("PUT CORS: %d body: %s", w.Code, w.Body.String())
	}

	// GET CORS
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/cors-bucket?cors"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET CORS: %d body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "AllowedOrigin") {
		t.Errorf("GET CORS: expected CORS config, got: %s", w.Body.String())
	}

	// DELETE CORS
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/cors-bucket?cors"))
	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("DELETE CORS: %d", w.Code)
	}
}

// ---- Bucket Lifecycle ----

func TestS3_BucketLifecycle(t *testing.T) {
	handler := newS3Gateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/lifecycle-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("create bucket: %d", w.Code)
	}

	// PUT lifecycle
	lifecycleXML := `<LifecycleConfiguration><Rule><ID>expire-old</ID><Status>Enabled</Status><Expiration><Days>90</Days></Expiration></Rule></LifecycleConfiguration>`
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/lifecycle-bucket?lifecycle", lifecycleXML))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("PUT lifecycle: %d body: %s", w.Code, w.Body.String())
	}

	// GET lifecycle
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/lifecycle-bucket?lifecycle"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET lifecycle: %d body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "expire-old") {
		t.Errorf("GET lifecycle: expected rule, got: %s", w.Body.String())
	}
}

// ---- Bucket Notification ----

func TestS3_BucketNotification(t *testing.T) {
	handler := newS3Gateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/notif-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("create bucket: %d", w.Code)
	}

	// PUT notification
	notifXML := `<NotificationConfiguration><TopicConfiguration><Topic>arn:aws:sns:us-east-1:000000000000:my-topic</Topic><Event>s3:ObjectCreated:*</Event></TopicConfiguration></NotificationConfiguration>`
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/notif-bucket?notification", notifXML))
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("PUT notification: %d body: %s", w.Code, w.Body.String())
	}

	// GET notification
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/notif-bucket?notification"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET notification: %d body: %s", w.Code, w.Body.String())
	}
}

// ---- Bucket ACL ----

func TestS3_BucketACL(t *testing.T) {
	handler := newS3Gateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodPut, "/acl-bucket"))
	if w.Code != http.StatusOK {
		t.Fatalf("create bucket: %d", w.Code)
	}

	// GET ACL (should return default)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/acl-bucket?acl"))
	if w.Code != http.StatusOK {
		t.Fatalf("GET ACL: %d body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Owner") || !strings.Contains(body, "FULL_CONTROL") {
		t.Errorf("GET ACL: expected owner with FULL_CONTROL, got: %s", body)
	}
}

// Suppress unused import warnings
var _ = xml.Header
