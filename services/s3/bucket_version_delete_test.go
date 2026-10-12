package s3_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

type bucketDeleteVersion struct {
	Key       string `xml:"Key"`
	VersionID string `xml:"VersionId"`
	IsLatest  bool   `xml:"IsLatest"`
}

type bucketDeleteVersions struct {
	Versions []bucketDeleteVersion `xml:"Version"`
	Markers  []bucketDeleteVersion `xml:"DeleteMarker"`
}

func listBucketDeleteVersions(t *testing.T, handler http.Handler, bucket string) bucketDeleteVersions {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/"+bucket+"?versions"))
	if w.Code != http.StatusOK {
		t.Fatalf("list versions: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result bucketDeleteVersions
	if err := xml.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode versions: %v", err)
	}
	return result
}

func TestS3_DeleteBucketRetainedVersions(t *testing.T) {
	for _, tc := range []struct {
		name          string
		bodies        []string
		deleteCurrent bool
	}{
		{name: "noncurrent-versions-and-marker", bodies: []string{"first body", "second body"}, deleteCurrent: true},
		{name: "delete-marker-only", deleteCurrent: true},
		{name: "object-version-only", bodies: []string{"retained body"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newS3Gateway(t)
			bucket := "retained-" + tc.name
			mustCreateBucket(t, handler, bucket)
			enableVersioning(t, handler, bucket)
			bodiesByVersion := make(map[string]string)
			for _, body := range tc.bodies {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/"+bucket+"/file.txt", body))
				if w.Code != http.StatusOK || w.Header().Get("x-amz-version-id") == "" {
					t.Fatalf("put version: status=%d version=%q body=%s", w.Code, w.Header().Get("x-amz-version-id"), w.Body.String())
				}
				bodiesByVersion[w.Header().Get("x-amz-version-id")] = body
			}
			if tc.deleteCurrent {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+bucket+"/file.txt"))
				if w.Code != http.StatusNoContent || w.Header().Get("x-amz-delete-marker") != "true" {
					t.Fatalf("create delete marker: status=%d marker=%q body=%s", w.Code, w.Header().Get("x-amz-delete-marker"), w.Body.String())
				}
			}
			before := listBucketDeleteVersions(t, handler, bucket)
			if len(before.Versions) != len(tc.bodies) || len(before.Markers) != boolCount(tc.deleteCurrent) {
				t.Fatalf("unexpected initial versions: %+v", before)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+bucket))
			var failure struct {
				Code string `xml:"Code"`
			}
			if err := xml.Unmarshal(w.Body.Bytes(), &failure); w.Code != http.StatusConflict || err != nil || failure.Code != "BucketNotEmpty" {
				t.Fatalf("delete retained bucket: expected 409 BucketNotEmpty, got %d: %s", w.Code, w.Body.String())
			}
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/"+bucket))
			if w.Code != http.StatusOK {
				t.Fatalf("rejected deletion removed the bucket: HEAD returned %d", w.Code)
			}
			if after := listBucketDeleteVersions(t, handler, bucket); !reflect.DeepEqual(after, before) {
				t.Fatalf("rejected deletion changed versions: before=%+v after=%+v", before, after)
			}
			for versionID, body := range bodiesByVersion {
				w = httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/"+bucket+"/file.txt?versionId="+url.QueryEscape(versionID)))
				if w.Code != http.StatusOK || w.Body.String() != body {
					t.Errorf("retained version %s: expected body %q, got status=%d body=%q", versionID, body, w.Code, w.Body.String())
				}
			}
		})
	}
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestS3_DeleteBucketAfterRemovingAllVersions(t *testing.T) {
	handler := newS3Gateway(t)
	const bucket = "removed-all-versions"
	mustCreateBucket(t, handler, bucket)
	enableVersioning(t, handler, bucket)
	mustPutObject(t, handler, bucket, "file.txt", "text/plain", "first body")
	mustPutObject(t, handler, bucket, "file.txt", "text/plain", "second body")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+bucket+"/file.txt"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete current object: got %d: %s", w.Code, w.Body.String())
	}
	retained := listBucketDeleteVersions(t, handler, bucket)
	if len(retained.Versions) != 2 || len(retained.Markers) != 1 {
		t.Fatalf("expected two versions and one delete marker, got %+v", retained)
	}
	for _, version := range append(retained.Versions, retained.Markers...) {
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+bucket+"/"+version.Key+"?versionId="+url.QueryEscape(version.VersionID)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("delete version %s: got %d: %s", version.VersionID, w.Code, w.Body.String())
		}
	}
	if remaining := listBucketDeleteVersions(t, handler, bucket); len(remaining.Versions) != 0 || len(remaining.Markers) != 0 {
		t.Fatalf("explicit version deletion left retained entries: %+v", remaining)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+bucket))
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete emptied versioned bucket: expected 204, got %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, s3Req(t, http.MethodHead, "/"+bucket))
	if w.Code != http.StatusNotFound {
		t.Fatalf("deleted bucket: expected HEAD 404, got %d", w.Code)
	}
}

func TestS3_DeleteBucketEmptyCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name          string
		versioned     bool
		putThenDelete bool
	}{
		{name: "empty-unversioned"},
		{name: "empty-versioned", versioned: true},
		{name: "emptied-unversioned", putThenDelete: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newS3Gateway(t)
			mustCreateBucket(t, handler, tc.name)
			if tc.versioned {
				enableVersioning(t, handler, tc.name)
			}
			if tc.putThenDelete {
				mustPutObject(t, handler, tc.name, "file.txt", "text/plain", "temporary")
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+tc.name+"/file.txt"))
				if w.Code != http.StatusNoContent {
					t.Fatalf("delete unversioned object: expected 204, got %d", w.Code)
				}
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+tc.name))
			if w.Code != http.StatusNoContent {
				t.Fatalf("delete empty bucket: expected 204, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestS3_DeleteObjectVersionLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name             string
		target           string
		createMarker     bool
		expectedBody     string
		expectedVersions int
	}{
		{name: "nonlatest-version", target: "first", expectedBody: "second body", expectedVersions: 1},
		{name: "latest-version-restores-previous", target: "second", expectedBody: "first body", expectedVersions: 1},
		{name: "delete-marker-restores-previous", target: "marker", createMarker: true, expectedBody: "second body", expectedVersions: 2},
		{name: "nonexistent-version-is-idempotent", target: "missing", expectedBody: "second body", expectedVersions: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newS3Gateway(t)
			const bucket = "delete-version-lifecycle"
			mustCreateBucket(t, handler, bucket)
			enableVersioning(t, handler, bucket)
			ids := map[string]string{"missing": "missing-version-id"}
			for _, version := range []string{"first", "second"} {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/"+bucket+"/file.txt", version+" body"))
				ids[version] = w.Header().Get("x-amz-version-id")
				if w.Code != http.StatusOK || ids[version] == "" {
					t.Fatalf("put %s version: status=%d version=%q", version, w.Code, ids[version])
				}
			}
			if tc.createMarker {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+bucket+"/file.txt"))
				ids["marker"] = w.Header().Get("x-amz-version-id")
				if w.Code != http.StatusNoContent || ids["marker"] == "" {
					t.Fatalf("create marker: status=%d version=%q", w.Code, ids["marker"])
				}
			}
			deletePath := "/" + bucket + "/file.txt?versionId=" + url.QueryEscape(ids[tc.target])
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodDelete, deletePath))
			if w.Code != http.StatusNoContent || w.Header().Get("x-amz-version-id") != ids[tc.target] {
				t.Fatalf("delete requested version: status=%d version=%q, want %q", w.Code, w.Header().Get("x-amz-version-id"), ids[tc.target])
			}
			if tc.createMarker && w.Header().Get("x-amz-delete-marker") != "true" {
				t.Error("deleting a delete marker must return x-amz-delete-marker: true")
			}
			remaining := listBucketDeleteVersions(t, handler, bucket)
			if len(remaining.Versions) != tc.expectedVersions || len(remaining.Markers) != 0 {
				t.Fatalf("unexpected retained entries after deletion: %+v", remaining)
			}
			latestCount := 0
			for _, version := range remaining.Versions {
				if version.IsLatest {
					latestCount++
				}
			}
			if latestCount != 1 {
				t.Errorf("expected exactly one latest version, got %d: %+v", latestCount, remaining)
			}
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodGet, "/"+bucket+"/file.txt"))
			if w.Code != http.StatusOK || w.Body.String() != tc.expectedBody {
				t.Fatalf("current object after version deletion: expected %q, got status=%d body=%q", tc.expectedBody, w.Code, w.Body.String())
			}
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodGet, deletePath))
			if w.Code != http.StatusNotFound {
				t.Errorf("deleted version still readable: expected 404, got %d", w.Code)
			}
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodDelete, deletePath))
			if w.Code != http.StatusNoContent {
				t.Fatalf("repeated version deletion: expected 204, got %d", w.Code)
			}
			if afterRepeat := listBucketDeleteVersions(t, handler, bucket); !reflect.DeepEqual(afterRepeat, remaining) {
				t.Fatalf("repeated version deletion changed retained entries: before=%+v after=%+v", remaining, afterRepeat)
			}
		})
	}
}

func TestS3_DeleteObjectVersionWhileSuspended(t *testing.T) {
	for _, newerUnversioned := range []bool{false, true} {
		name := "cleanup-retained-versions"
		if newerUnversioned {
			name = "preserve-newer-unversioned-object"
		}
		t.Run(name, func(t *testing.T) {
			handler := newS3Gateway(t)
			const bucket = "suspended-version-cleanup"
			const path = "/" + bucket + "/file.txt"
			mustCreateBucket(t, handler, bucket)
			enableVersioning(t, handler, bucket)
			mustPutObject(t, handler, bucket, "file.txt", "text/plain", "old version")
			mustPutObject(t, handler, bucket, "file.txt", "text/plain", "new version")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodDelete, path))
			if w.Code != http.StatusNoContent {
				t.Fatalf("create marker: expected 204, got %d", w.Code)
			}
			retained := listBucketDeleteVersions(t, handler, bucket)
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/"+bucket+"?versioning",
				`<VersioningConfiguration><Status>Suspended</Status></VersioningConfiguration>`))
			if w.Code != http.StatusOK {
				t.Fatalf("suspend versioning: expected 200, got %d: %s", w.Code, w.Body.String())
			}
			if newerUnversioned {
				mustPutObject(t, handler, bucket, "file.txt", "text/plain", "newer unversioned body")
			}
			// Remove the latest marker first, then the former current version,
			// exercising restoration while an unversioned current object may exist.
			for _, entry := range append(retained.Markers, retained.Versions...) {
				w = httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, http.MethodDelete, path+"?versionId="+url.QueryEscape(entry.VersionID)))
				if w.Code != http.StatusNoContent {
					t.Fatalf("delete retained entry while suspended: expected 204, got %d: %s", w.Code, w.Body.String())
				}
				if newerUnversioned {
					w = httptest.NewRecorder()
					handler.ServeHTTP(w, s3Req(t, http.MethodGet, path))
					if w.Code != http.StatusOK || w.Body.String() != "newer unversioned body" || w.Header().Get("x-amz-version-id") != "" {
						t.Fatalf("historical deletion changed newer unversioned object: status=%d version=%q body=%q", w.Code, w.Header().Get("x-amz-version-id"), w.Body.String())
					}
				}
			}
			if remaining := listBucketDeleteVersions(t, handler, bucket); len(remaining.Versions) != 0 || len(remaining.Markers) != 0 {
				t.Fatalf("suspended cleanup left retained entries: %+v", remaining)
			}
			if newerUnversioned {
				w = httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+bucket))
				if w.Code != http.StatusConflict {
					t.Fatalf("bucket with newer unversioned object: expected 409, got %d", w.Code)
				}
				w = httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, http.MethodDelete, path))
				if w.Code != http.StatusNoContent {
					t.Fatalf("delete newer unversioned object: expected 204, got %d", w.Code)
				}
			}
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+bucket))
			if w.Code != http.StatusNoContent {
				t.Fatalf("delete emptied suspended bucket: expected 204, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestS3_DeleteObjectVersionRestoresHeaders(t *testing.T) {
	for _, target := range []string{"latest-version", "delete-marker"} {
		t.Run(target, func(t *testing.T) {
			handler := newS3Gateway(t)
			const bucket = "restored-version-headers"
			const path = "/" + bucket + "/file.json"
			const body = `{"retained":"original body"}`
			mustCreateBucket(t, handler, bucket)
			enableVersioning(t, handler, bucket)
			mustPutObject(t, handler, bucket, "file.json", "application/json", body)
			before := httptest.NewRecorder()
			handler.ServeHTTP(before, s3Req(t, http.MethodGet, path))
			if before.Code != http.StatusOK {
				t.Fatalf("read original version: expected 200, got %d", before.Code)
			}
			w := httptest.NewRecorder()
			if target == "latest-version" {
				req := s3ReqWithBody(t, http.MethodPut, path, "replacement text")
				req.Header.Set("Content-Type", "text/plain")
				handler.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Fatalf("put replacement: expected 200, got %d", w.Code)
				}
			} else {
				handler.ServeHTTP(w, s3Req(t, http.MethodDelete, path))
				if w.Code != http.StatusNoContent {
					t.Fatalf("create marker: expected 204, got %d", w.Code)
				}
			}
			versionID := w.Header().Get("x-amz-version-id")
			if versionID == "" {
				t.Fatal("missing replacement version or marker ID")
			}
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodDelete, path+"?versionId="+url.QueryEscape(versionID)))
			if w.Code != http.StatusNoContent {
				t.Fatalf("delete replacement: expected 204, got %d", w.Code)
			}
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				w = httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, method, path))
				if w.Code != http.StatusOK {
					t.Fatalf("%s restored version: expected 200, got %d", method, w.Code)
				}
				for _, header := range []string{"Content-Type", "Content-Length", "ETag", "Last-Modified"} {
					if got, want := w.Header().Get(header), before.Header().Get(header); got != want {
						t.Errorf("%s restored %s: got %q, want %q", method, header, got, want)
					}
				}
				if method == http.MethodGet && (w.Body.String() != body || w.Header().Get("x-amz-version-id") != before.Header().Get("x-amz-version-id")) {
					t.Errorf("restored version identity/body mismatch: version=%q body=%q", w.Header().Get("x-amz-version-id"), w.Body.String())
				}
			}
		})
	}
}

func TestS3_DeleteNullObjectVersion(t *testing.T) {
	for _, tc := range []struct {
		name           string
		retained       bool
		retainedMarker bool
	}{
		{name: "unversioned"},
		{name: "suspended-restores-version", retained: true},
		{name: "suspended-preserves-marker", retained: true, retainedMarker: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newS3Gateway(t)
			const bucket = "delete-null-version"
			const path = "/" + bucket + "/file.txt"
			mustCreateBucket(t, handler, bucket)
			if tc.retained {
				enableVersioning(t, handler, bucket)
				mustPutObject(t, handler, bucket, "file.txt", "text/plain", "retained numbered version")
				if tc.retainedMarker {
					w := httptest.NewRecorder()
					handler.ServeHTTP(w, s3Req(t, http.MethodDelete, path))
					if w.Code != http.StatusNoContent {
						t.Fatalf("create retained marker: expected 204, got %d", w.Code)
					}
				}
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, s3ReqWithBody(t, http.MethodPut, "/"+bucket+"?versioning",
					`<VersioningConfiguration><Status>Suspended</Status></VersioningConfiguration>`))
				if w.Code != http.StatusOK {
					t.Fatalf("suspend versioning: expected 200, got %d", w.Code)
				}
			}
			retained := listBucketDeleteVersions(t, handler, bucket)
			mustPutObject(t, handler, bucket, "file.txt", "text/plain", "current null version")
			for attempt := 0; attempt < 2; attempt++ {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, http.MethodDelete, path+"?versionId=null"))
				if w.Code != http.StatusNoContent || w.Header().Get("x-amz-version-id") != "null" {
					t.Fatalf("delete null attempt %d: status=%d version=%q", attempt, w.Code, w.Header().Get("x-amz-version-id"))
				}
				if remaining := listBucketDeleteVersions(t, handler, bucket); !reflect.DeepEqual(remaining, retained) {
					t.Fatalf("null deletion changed retained numbered history: before=%+v after=%+v", retained, remaining)
				}
				w = httptest.NewRecorder()
				handler.ServeHTTP(w, s3Req(t, http.MethodGet, path))
				if tc.retained && !tc.retainedMarker {
					if w.Code != http.StatusOK || w.Body.String() != "retained numbered version" || w.Header().Get("x-amz-version-id") != retained.Versions[0].VersionID {
						t.Fatalf("null deletion did not restore retained version: status=%d version=%q body=%q", w.Code, w.Header().Get("x-amz-version-id"), w.Body.String())
					}
				} else if w.Code != http.StatusNotFound {
					t.Fatalf("expected no visible object after null deletion, got status=%d body=%q", w.Code, w.Body.String())
				}
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, s3Req(t, http.MethodDelete, "/"+bucket))
			expectedStatus := http.StatusNoContent
			if tc.retained {
				expectedStatus = http.StatusConflict
			}
			if w.Code != expectedStatus {
				t.Fatalf("delete bucket after null cleanup: expected %d, got %d: %s", expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}
