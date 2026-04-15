package ecrpublic_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/ecrpublic"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func doCall(t *testing.T, h http.Handler, action string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body == nil {
		data = []byte("{}")
	} else {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "SpencerFrontendService."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ecr-public/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, w.Body.String())
	}
}

func TestRepositoryLifecycle(t *testing.T) {
	h := newGateway(t)

	if w := doCall(t, h, "CreateRepository", map[string]any{
		"repositoryName": "myrepo",
		"catalogData": map[string]any{
			"description": "demo",
		},
	}); w.Code != http.StatusOK {
		t.Fatalf("CreateRepository: want 200, got %d: %s", w.Code, w.Body.String())
	}

	w := doCall(t, h, "DescribeRepositories", nil)
	var listed struct {
		Repositories []struct {
			RepositoryName string
		}
	}
	decode(t, w, &listed)
	if len(listed.Repositories) != 1 || listed.Repositories[0].RepositoryName != "myrepo" {
		t.Fatalf("unexpected: %+v", listed)
	}

	if w := doCall(t, h, "GetRepositoryCatalogData", map[string]any{"repositoryName": "myrepo"}); w.Code != http.StatusOK {
		t.Fatalf("GetRepositoryCatalogData: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "DeleteRepository", map[string]any{"repositoryName": "myrepo"}); w.Code != http.StatusOK {
		t.Fatalf("DeleteRepository: want 200, got %d", w.Code)
	}
}

func TestImageLifecycle(t *testing.T) {
	h := newGateway(t)
	doCall(t, h, "CreateRepository", map[string]any{"repositoryName": "img"})

	if w := doCall(t, h, "PutImage", map[string]any{
		"repositoryName": "img",
		"imageManifest":  `{"schemaVersion":2}`,
		"imageTag":       "latest",
	}); w.Code != http.StatusOK {
		t.Fatalf("PutImage: want 200, got %d: %s", w.Code, w.Body.String())
	}

	w := doCall(t, h, "DescribeImages", map[string]any{"repositoryName": "img"})
	var desc struct {
		ImageDetails []struct {
			ImageDigest string
			ImageTags   []string
		}
	}
	decode(t, w, &desc)
	if len(desc.ImageDetails) != 1 || len(desc.ImageDetails[0].ImageTags) != 1 {
		t.Fatalf("unexpected describe: %+v", desc)
	}

	w = doCall(t, h, "DescribeImageTags", map[string]any{"repositoryName": "img"})
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeImageTags: want 200, got %d", w.Code)
	}
}

func TestRegistryAndAuth(t *testing.T) {
	h := newGateway(t)
	if w := doCall(t, h, "DescribeRegistries", nil); w.Code != http.StatusOK {
		t.Fatalf("DescribeRegistries: want 200, got %d: %s", w.Code, w.Body.String())
	}
	if w := doCall(t, h, "GetAuthorizationToken", nil); w.Code != http.StatusOK {
		t.Fatalf("GetAuthorizationToken: want 200, got %d", w.Code)
	}
	if w := doCall(t, h, "GetRegistryCatalogData", nil); w.Code != http.StatusOK {
		t.Fatalf("GetRegistryCatalogData: want 200, got %d", w.Code)
	}
	if w := doCall(t, h, "PutRegistryCatalogData", map[string]any{"displayName": "custom"}); w.Code != http.StatusOK {
		t.Fatalf("PutRegistryCatalogData: want 200, got %d", w.Code)
	}
}

func TestLayerUploads(t *testing.T) {
	h := newGateway(t)
	doCall(t, h, "CreateRepository", map[string]any{"repositoryName": "layers"})

	w := doCall(t, h, "InitiateLayerUpload", map[string]any{"repositoryName": "layers"})
	var init struct {
		UploadID string `json:"uploadId"`
	}
	decode(t, w, &init)
	if init.UploadID == "" {
		t.Fatalf("expected uploadId")
	}

	part := base64.StdEncoding.EncodeToString([]byte("layer-data"))
	if w := doCall(t, h, "UploadLayerPart", map[string]any{
		"repositoryName": "layers",
		"uploadId":       init.UploadID,
		"layerPartBlob":  part,
	}); w.Code != http.StatusOK {
		t.Fatalf("UploadLayerPart: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "CompleteLayerUpload", map[string]any{
		"repositoryName": "layers",
		"uploadId":       init.UploadID,
		"layerDigests":   []string{"sha256:abc"},
	}); w.Code != http.StatusOK {
		t.Fatalf("CompleteLayerUpload: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "BatchCheckLayerAvailability", map[string]any{
		"repositoryName": "layers",
		"layerDigests":   []string{"sha256:abc"},
	}); w.Code != http.StatusOK {
		t.Fatalf("BatchCheckLayerAvailability: want 200, got %d", w.Code)
	}
}

func TestRepositoryPolicy(t *testing.T) {
	h := newGateway(t)
	doCall(t, h, "CreateRepository", map[string]any{"repositoryName": "p"})

	if w := doCall(t, h, "SetRepositoryPolicy", map[string]any{
		"repositoryName": "p",
		"policyText":     `{"Version":"2012-10-17","Statement":[]}`,
	}); w.Code != http.StatusOK {
		t.Fatalf("SetRepositoryPolicy: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "GetRepositoryPolicy", map[string]any{"repositoryName": "p"}); w.Code != http.StatusOK {
		t.Fatalf("GetRepositoryPolicy: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "DeleteRepositoryPolicy", map[string]any{"repositoryName": "p"}); w.Code != http.StatusOK {
		t.Fatalf("DeleteRepositoryPolicy: want 200, got %d", w.Code)
	}
}

func TestTagging(t *testing.T) {
	h := newGateway(t)
	arn := "arn:aws:ecr-public::000000000000:repository/foo"
	if w := doCall(t, h, "TagResource", map[string]any{
		"resourceArn": arn,
		"tags": []map[string]any{
			{"Key": "env", "Value": "dev"},
		},
	}); w.Code != http.StatusOK {
		t.Fatalf("TagResource: want 200, got %d", w.Code)
	}

	w := doCall(t, h, "ListTagsForResource", map[string]any{"resourceArn": arn})
	var listed struct {
		Tags []struct{ Key, Value string }
	}
	decode(t, w, &listed)
	if len(listed.Tags) != 1 || listed.Tags[0].Value != "dev" {
		t.Fatalf("unexpected tags: %+v", listed)
	}
}
