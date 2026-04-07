package ecrpublic_test

import (
	"bytes"
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

func svcReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "SpencerFrontendService."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ecr-public/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestBatchCheckLayerAvailability(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchCheckLayerAvailability", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchCheckLayerAvailability: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDeleteImage(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDeleteImage", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDeleteImage: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCompleteLayerUpload(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CompleteLayerUpload", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CompleteLayerUpload: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateRepository(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateRepository", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateRepository: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteRepository(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteRepository", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteRepository: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteRepositoryPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteRepositoryPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteRepositoryPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeImageTags(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeImageTags", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeImageTags: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeImages(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeImages", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeImages: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeRegistries(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeRegistries", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeRegistries: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeRepositories(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeRepositories", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeRepositories: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetAuthorizationToken(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetAuthorizationToken", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetAuthorizationToken: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetRegistryCatalogData(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetRegistryCatalogData", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetRegistryCatalogData: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetRepositoryCatalogData(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetRepositoryCatalogData", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetRepositoryCatalogData: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetRepositoryPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetRepositoryPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetRepositoryPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestInitiateLayerUpload(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "InitiateLayerUpload", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("InitiateLayerUpload: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTagsForResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTagsForResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutImage(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutImage", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutImage: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutRegistryCatalogData(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutRegistryCatalogData", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutRegistryCatalogData: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutRepositoryCatalogData(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutRepositoryCatalogData", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutRepositoryCatalogData: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSetRepositoryPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SetRepositoryPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SetRepositoryPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestTagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "TagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUntagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UntagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUploadLayerPart(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UploadLayerPart", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UploadLayerPart: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

