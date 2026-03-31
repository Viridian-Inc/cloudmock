package ecr_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	ecrsvc "github.com/neureaux/cloudmock/services/ecr"
)

// newECRGateway builds a full gateway stack with the ECR service registered and IAM disabled.
func newECRGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(ecrsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// ecrReq builds a JSON POST request targeting the ECR service via X-Amz-Target.
func ecrReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("ecrReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerRegistry_V20150921."+action)
	// Authorization header places "ecr" as the service in the credential scope
	// so the gateway router can detect "ecr" as the target service.
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ecr/aws4_request, SignedHeaders=host;x-amz-target, Signature=abc123")
	return req
}

// decodeJSON is a test helper that unmarshals JSON into a map.
func decodeJSON(t *testing.T, data string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("decodeJSON: %v\nbody: %s", err, data)
	}
	return m
}

// ---- Test 1: CreateRepository + DescribeRepositories ----

func TestECR_CreateAndDescribeRepositories(t *testing.T) {
	handler := newECRGateway(t)

	// Create a repository.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName":     "my-app",
		"imageTagMutability": "MUTABLE",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateRepository: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}

	mc := decodeJSON(t, wc.Body.String())
	repo, ok := mc["repository"].(map[string]any)
	if !ok {
		t.Fatalf("CreateRepository: missing repository in response\nbody: %s", wc.Body.String())
	}

	if repo["repositoryName"].(string) != "my-app" {
		t.Errorf("CreateRepository: expected repositoryName=my-app, got %q", repo["repositoryName"])
	}
	arn, _ := repo["repositoryArn"].(string)
	if !strings.Contains(arn, "my-app") {
		t.Errorf("CreateRepository: ARN %q does not contain repo name", arn)
	}
	uri, _ := repo["repositoryUri"].(string)
	if !strings.Contains(uri, "my-app") {
		t.Errorf("CreateRepository: URI %q does not contain repo name", uri)
	}
	if _, ok := repo["createdAt"]; !ok {
		t.Error("CreateRepository: missing createdAt in response")
	}

	// DescribeRepositories — all.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ecrReq(t, "DescribeRepositories", map[string]any{}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeRepositories: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	md := decodeJSON(t, wd.Body.String())
	repos, ok := md["repositories"].([]any)
	if !ok || len(repos) == 0 {
		t.Fatalf("DescribeRepositories: expected non-empty repositories\nbody: %s", wd.Body.String())
	}

	found := false
	for _, r := range repos {
		entry := r.(map[string]any)
		if entry["repositoryName"].(string) == "my-app" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DescribeRepositories: my-app not found in list")
	}

	// DescribeRepositories — by name.
	wdn := httptest.NewRecorder()
	handler.ServeHTTP(wdn, ecrReq(t, "DescribeRepositories", map[string]any{
		"repositoryNames": []string{"my-app"},
	}))
	if wdn.Code != http.StatusOK {
		t.Fatalf("DescribeRepositories by name: expected 200, got %d\nbody: %s", wdn.Code, wdn.Body.String())
	}
	mdn := decodeJSON(t, wdn.Body.String())
	reposN := mdn["repositories"].([]any)
	if len(reposN) != 1 {
		t.Errorf("DescribeRepositories by name: expected 1 repo, got %d", len(reposN))
	}
}

func TestECR_CreateRepository_AlreadyExists(t *testing.T) {
	handler := newECRGateway(t)

	// Create once.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName": "dup-repo",
	}))
	if w1.Code != http.StatusOK {
		t.Fatalf("first CreateRepository: %d %s", w1.Code, w1.Body.String())
	}

	// Create again — should fail.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName": "dup-repo",
	}))
	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate CreateRepository: expected 409, got %d\nbody: %s", w2.Code, w2.Body.String())
	}
}

// ---- Test 2: PutImage + ListImages ----

func TestECR_PutImageAndListImages(t *testing.T) {
	handler := newECRGateway(t)

	// Create repo.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName": "img-repo",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateRepository: %d %s", wc.Code, wc.Body.String())
	}

	// Push an image.
	manifest := `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json"}`
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, ecrReq(t, "PutImage", map[string]any{
		"repositoryName": "img-repo",
		"imageManifest":  manifest,
		"imageTag":       "latest",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutImage: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	mp := decodeJSON(t, wp.Body.String())
	img, ok := mp["image"].(map[string]any)
	if !ok {
		t.Fatalf("PutImage: missing image in response\nbody: %s", wp.Body.String())
	}
	imgID := img["imageId"].(map[string]any)
	digest, _ := imgID["imageDigest"].(string)
	if !strings.HasPrefix(digest, "sha256:") {
		t.Errorf("PutImage: expected sha256 digest, got %q", digest)
	}
	if imgID["imageTag"].(string) != "latest" {
		t.Errorf("PutImage: expected imageTag=latest, got %q", imgID["imageTag"])
	}

	// ListImages.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ecrReq(t, "ListImages", map[string]any{
		"repositoryName": "img-repo",
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListImages: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}

	ml := decodeJSON(t, wl.Body.String())
	ids, ok := ml["imageIds"].([]any)
	if !ok || len(ids) == 0 {
		t.Fatalf("ListImages: expected non-empty imageIds\nbody: %s", wl.Body.String())
	}
	first := ids[0].(map[string]any)
	if first["imageDigest"].(string) != digest {
		t.Errorf("ListImages: expected digest %q, got %q", digest, first["imageDigest"])
	}
}

// ---- Test 3: BatchGetImage ----

func TestECR_BatchGetImage(t *testing.T) {
	handler := newECRGateway(t)

	// Setup.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName": "batch-repo",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateRepository: %d %s", wc.Code, wc.Body.String())
	}

	manifest := `{"schemaVersion":2,"config":{"digest":"sha256:abc"}}`
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, ecrReq(t, "PutImage", map[string]any{
		"repositoryName": "batch-repo",
		"imageManifest":  manifest,
		"imageTag":       "v1",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("setup PutImage: %d %s", wp.Code, wp.Body.String())
	}
	mPut := decodeJSON(t, wp.Body.String())
	digest := mPut["image"].(map[string]any)["imageId"].(map[string]any)["imageDigest"].(string)

	// BatchGetImage by digest.
	wb := httptest.NewRecorder()
	handler.ServeHTTP(wb, ecrReq(t, "BatchGetImage", map[string]any{
		"repositoryName": "batch-repo",
		"imageIds": []map[string]string{
			{"imageDigest": digest},
		},
	}))
	if wb.Code != http.StatusOK {
		t.Fatalf("BatchGetImage: expected 200, got %d\nbody: %s", wb.Code, wb.Body.String())
	}

	mb := decodeJSON(t, wb.Body.String())
	images, ok := mb["images"].([]any)
	if !ok || len(images) == 0 {
		t.Fatalf("BatchGetImage: expected non-empty images\nbody: %s", wb.Body.String())
	}
	gotImg := images[0].(map[string]any)
	if gotImg["imageManifest"].(string) != manifest {
		t.Errorf("BatchGetImage: manifest mismatch")
	}
	if gotImg["repositoryName"].(string) != "batch-repo" {
		t.Errorf("BatchGetImage: repositoryName mismatch")
	}

	// BatchGetImage by tag.
	wbt := httptest.NewRecorder()
	handler.ServeHTTP(wbt, ecrReq(t, "BatchGetImage", map[string]any{
		"repositoryName": "batch-repo",
		"imageIds": []map[string]string{
			{"imageTag": "v1"},
		},
	}))
	if wbt.Code != http.StatusOK {
		t.Fatalf("BatchGetImage by tag: expected 200, got %d\nbody: %s", wbt.Code, wbt.Body.String())
	}
	mbt := decodeJSON(t, wbt.Body.String())
	if imgs := mbt["images"].([]any); len(imgs) == 0 {
		t.Error("BatchGetImage by tag: expected image in result")
	}

	// BatchGetImage — not found.
	wbf := httptest.NewRecorder()
	handler.ServeHTTP(wbf, ecrReq(t, "BatchGetImage", map[string]any{
		"repositoryName": "batch-repo",
		"imageIds": []map[string]string{
			{"imageTag": "nonexistent"},
		},
	}))
	if wbf.Code != http.StatusOK {
		t.Fatalf("BatchGetImage not-found: expected 200, got %d\nbody: %s", wbf.Code, wbf.Body.String())
	}
	mbf := decodeJSON(t, wbf.Body.String())
	failures, ok := mbf["failures"].([]any)
	if !ok || len(failures) == 0 {
		t.Error("BatchGetImage not-found: expected failure entry")
	}
}

// ---- Test 4: BatchDeleteImage ----

func TestECR_BatchDeleteImage(t *testing.T) {
	handler := newECRGateway(t)

	// Setup.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName": "del-repo",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateRepository: %d %s", wc.Code, wc.Body.String())
	}

	manifest1 := `{"schemaVersion":2,"tag":"v1"}`
	manifest2 := `{"schemaVersion":2,"tag":"v2"}`

	for _, m := range []struct{ manifest, tag string }{
		{manifest1, "v1"},
		{manifest2, "v2"},
	} {
		wp := httptest.NewRecorder()
		handler.ServeHTTP(wp, ecrReq(t, "PutImage", map[string]any{
			"repositoryName": "del-repo",
			"imageManifest":  m.manifest,
			"imageTag":       m.tag,
		}))
		if wp.Code != http.StatusOK {
			t.Fatalf("setup PutImage %s: %d %s", m.tag, wp.Code, wp.Body.String())
		}
	}

	// Verify 2 images exist.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ecrReq(t, "ListImages", map[string]any{
		"repositoryName": "del-repo",
	}))
	ml := decodeJSON(t, wl.Body.String())
	if len(ml["imageIds"].([]any)) != 2 {
		t.Fatalf("setup: expected 2 images, got %d", len(ml["imageIds"].([]any)))
	}

	// Delete by tag.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ecrReq(t, "BatchDeleteImage", map[string]any{
		"repositoryName": "del-repo",
		"imageIds": []map[string]string{
			{"imageTag": "v1"},
		},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("BatchDeleteImage: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	md := decodeJSON(t, wd.Body.String())
	deleted, ok := md["imageIds"].([]any)
	if !ok || len(deleted) == 0 {
		t.Fatal("BatchDeleteImage: expected deleted imageIds in response")
	}
	if md["failures"] != nil {
		if failures := md["failures"].([]any); len(failures) > 0 {
			t.Errorf("BatchDeleteImage: unexpected failures: %v", failures)
		}
	}

	// Verify only 1 image remains.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, ecrReq(t, "ListImages", map[string]any{
		"repositoryName": "del-repo",
	}))
	ml2 := decodeJSON(t, wl2.Body.String())
	if len(ml2["imageIds"].([]any)) != 1 {
		t.Errorf("after BatchDeleteImage: expected 1 image, got %d", len(ml2["imageIds"].([]any)))
	}
}

// ---- Test 5: DeleteRepository ----

func TestECR_DeleteRepository(t *testing.T) {
	handler := newECRGateway(t)

	// Create repo.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName": "to-delete",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateRepository: %d %s", wc.Code, wc.Body.String())
	}

	// Delete empty repo.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, ecrReq(t, "DeleteRepository", map[string]any{
		"repositoryName": "to-delete",
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteRepository: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}

	mdel := decodeJSON(t, wdel.Body.String())
	repo := mdel["repository"].(map[string]any)
	if repo["repositoryName"].(string) != "to-delete" {
		t.Errorf("DeleteRepository: expected repositoryName=to-delete in response")
	}

	// Verify it no longer exists.
	wd2 := httptest.NewRecorder()
	handler.ServeHTTP(wd2, ecrReq(t, "DescribeRepositories", map[string]any{
		"repositoryNames": []string{"to-delete"},
	}))
	if wd2.Code != http.StatusBadRequest {
		t.Errorf("DescribeRepositories deleted repo: expected 400, got %d", wd2.Code)
	}
}

func TestECR_DeleteRepository_NotEmptyWithoutForce(t *testing.T) {
	handler := newECRGateway(t)

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName": "nonempty-repo",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup: %d %s", wc.Code, wc.Body.String())
	}

	// Push an image.
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, ecrReq(t, "PutImage", map[string]any{
		"repositoryName": "nonempty-repo",
		"imageManifest":  `{"schemaVersion":2}`,
		"imageTag":       "v1",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("setup PutImage: %d %s", wp.Code, wp.Body.String())
	}

	// Attempt delete without force — should fail.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ecrReq(t, "DeleteRepository", map[string]any{
		"repositoryName": "nonempty-repo",
		"force":          false,
	}))
	if wd.Code != http.StatusConflict {
		t.Fatalf("DeleteRepository non-empty no-force: expected 409, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Delete with force — should succeed.
	wdf := httptest.NewRecorder()
	handler.ServeHTTP(wdf, ecrReq(t, "DeleteRepository", map[string]any{
		"repositoryName": "nonempty-repo",
		"force":          true,
	}))
	if wdf.Code != http.StatusOK {
		t.Fatalf("DeleteRepository force: expected 200, got %d\nbody: %s", wdf.Code, wdf.Body.String())
	}
}

// ---- Test 6: GetAuthorizationToken ----

func TestECR_GetAuthorizationToken(t *testing.T) {
	handler := newECRGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecrReq(t, "GetAuthorizationToken", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetAuthorizationToken: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	authData, ok := m["authorizationData"].([]any)
	if !ok || len(authData) == 0 {
		t.Fatalf("GetAuthorizationToken: missing authorizationData\nbody: %s", w.Body.String())
	}

	entry := authData[0].(map[string]any)
	token, _ := entry["authorizationToken"].(string)
	if token == "" {
		t.Error("GetAuthorizationToken: missing authorizationToken")
	}
	if _, ok := entry["expiresAt"]; !ok {
		t.Error("GetAuthorizationToken: missing expiresAt")
	}
	proxyEndpoint, _ := entry["proxyEndpoint"].(string)
	if !strings.Contains(proxyEndpoint, "dkr.ecr") {
		t.Errorf("GetAuthorizationToken: unexpected proxyEndpoint %q", proxyEndpoint)
	}
}

// ---- TagResource / UntagResource / ListTagsForResource ----

func TestECR_TagOperations(t *testing.T) {
	handler := newECRGateway(t)

	// Create repo.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName": "tag-repo",
		"tags": []map[string]string{
			{"Key": "env", "Value": "test"},
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateRepository: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	repoARN := mc["repository"].(map[string]any)["repositoryArn"].(string)

	// TagResource.
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, ecrReq(t, "TagResource", map[string]any{
		"resourceArn": repoARN,
		"tags": []map[string]string{
			{"Key": "team", "Value": "platform"},
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// ListTagsForResource.
	wlt := httptest.NewRecorder()
	handler.ServeHTTP(wlt, ecrReq(t, "ListTagsForResource", map[string]any{
		"resourceArn": repoARN,
	}))
	if wlt.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", wlt.Code, wlt.Body.String())
	}
	mlt := decodeJSON(t, wlt.Body.String())
	tags, ok := mlt["tags"].([]any)
	if !ok {
		t.Fatalf("ListTagsForResource: missing tags\nbody: %s", wlt.Body.String())
	}
	tagMap := make(map[string]string)
	for _, tg := range tags {
		entry := tg.(map[string]any)
		tagMap[entry["Key"].(string)] = entry["Value"].(string)
	}
	if tagMap["team"] != "platform" {
		t.Errorf("ListTagsForResource: expected team=platform, got %q", tagMap["team"])
	}

	// UntagResource.
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, ecrReq(t, "UntagResource", map[string]any{
		"resourceArn": repoARN,
		"tagKeys":     []string{"team"},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}

	// Verify tag removed.
	wlt2 := httptest.NewRecorder()
	handler.ServeHTTP(wlt2, ecrReq(t, "ListTagsForResource", map[string]any{
		"resourceArn": repoARN,
	}))
	mlt2 := decodeJSON(t, wlt2.Body.String())
	tags2 := mlt2["tags"].([]any)
	for _, tg := range tags2 {
		entry := tg.(map[string]any)
		if entry["Key"].(string) == "team" {
			t.Error("UntagResource: team tag should have been removed")
		}
	}
}

// ---- DescribeImageScanFindings stub ----

func TestECR_DescribeImageScanFindings(t *testing.T) {
	handler := newECRGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecrReq(t, "DescribeImageScanFindings", map[string]any{
		"repositoryName": "any-repo",
		"imageId":        map[string]string{"imageTag": "latest"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeImageScanFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	status, ok := m["imageScanStatus"].(map[string]any)
	if !ok {
		t.Fatalf("DescribeImageScanFindings: missing imageScanStatus\nbody: %s", w.Body.String())
	}
	if status["status"].(string) != "COMPLETE" {
		t.Errorf("DescribeImageScanFindings: expected status=COMPLETE, got %q", status["status"])
	}
}

// ---- Test: RepositoryNotFoundException — DescribeRepositories by name ----

func TestECR_DescribeRepositories_NotFound(t *testing.T) {
	handler := newECRGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecrReq(t, "DescribeRepositories", map[string]any{
		"repositoryNames": []string{"nonexistent-repo"},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DescribeRepositories not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "RepositoryNotFoundException") {
		t.Errorf("DescribeRepositories not found: expected RepositoryNotFoundException\nbody: %s", body)
	}
}

// ---- Test: RepositoryNotFoundException — ListImages on missing repo ----

func TestECR_ListImages_RepositoryNotFound(t *testing.T) {
	handler := newECRGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecrReq(t, "ListImages", map[string]any{
		"repositoryName": "no-such-repo",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("ListImages repo not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "RepositoryNotFoundException") {
		t.Errorf("ListImages repo not found: expected RepositoryNotFoundException\nbody: %s", body)
	}
}

// ---- Test: PutImage — RepositoryNotFoundException ----

func TestECR_PutImage_RepositoryNotFound(t *testing.T) {
	handler := newECRGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecrReq(t, "PutImage", map[string]any{
		"repositoryName": "no-such-repo",
		"imageManifest":  `{"schemaVersion":2}`,
		"imageTag":       "latest",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PutImage repo not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "RepositoryNotFoundException") {
		t.Errorf("PutImage repo not found: expected RepositoryNotFoundException\nbody: %s", body)
	}
}

// ---- Test: ImageTagAlreadyExistsException — immutable tag ----

func TestECR_PutImage_ImageTagAlreadyExists_Immutable(t *testing.T) {
	handler := newECRGateway(t)

	// Create repo with IMMUTABLE tag mutability.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecrReq(t, "CreateRepository", map[string]any{
		"repositoryName":     "immutable-repo",
		"imageTagMutability": "IMMUTABLE",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateRepository: %d %s", wc.Code, wc.Body.String())
	}

	// Push first image with tag.
	wp1 := httptest.NewRecorder()
	handler.ServeHTTP(wp1, ecrReq(t, "PutImage", map[string]any{
		"repositoryName": "immutable-repo",
		"imageManifest":  `{"schemaVersion":2,"unique":"first"}`,
		"imageTag":       "v1",
	}))
	if wp1.Code != http.StatusOK {
		t.Fatalf("first PutImage: %d %s", wp1.Code, wp1.Body.String())
	}

	// Push different image with same tag — should fail for IMMUTABLE.
	wp2 := httptest.NewRecorder()
	handler.ServeHTTP(wp2, ecrReq(t, "PutImage", map[string]any{
		"repositoryName": "immutable-repo",
		"imageManifest":  `{"schemaVersion":2,"unique":"second"}`,
		"imageTag":       "v1",
	}))
	if wp2.Code != http.StatusConflict {
		t.Fatalf("duplicate tag immutable: expected 409, got %d\nbody: %s", wp2.Code, wp2.Body.String())
	}
	body := wp2.Body.String()
	if !strings.Contains(body, "ImageTagAlreadyExistsException") {
		t.Errorf("duplicate tag immutable: expected ImageTagAlreadyExistsException\nbody: %s", body)
	}
}

// ---- Test: DeleteRepository — RepositoryNotFoundException ----

func TestECR_DeleteRepository_NotFound(t *testing.T) {
	handler := newECRGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecrReq(t, "DeleteRepository", map[string]any{
		"repositoryName": "no-such-repo",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DeleteRepository not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "RepositoryNotFoundException") {
		t.Errorf("DeleteRepository not found: expected RepositoryNotFoundException\nbody: %s", body)
	}
}

// ---- Test: GetAuthorizationToken — response structure ----

func TestECR_GetAuthorizationToken_Structure(t *testing.T) {
	handler := newECRGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecrReq(t, "GetAuthorizationToken", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetAuthorizationToken: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	authData, ok := m["authorizationData"].([]any)
	if !ok || len(authData) == 0 {
		t.Fatalf("GetAuthorizationToken: missing authorizationData")
	}

	entry := authData[0].(map[string]any)
	token, _ := entry["authorizationToken"].(string)
	if token == "" {
		t.Error("GetAuthorizationToken: empty token")
	}

	// Token should be base64-encoded.
	if !strings.Contains(token, "=") && len(token) < 10 {
		t.Errorf("GetAuthorizationToken: token does not look base64-encoded: %q", token)
	}
}

// ---- Unknown action ----

func TestECR_UnknownAction(t *testing.T) {
	handler := newECRGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecrReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
