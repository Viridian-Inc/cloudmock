package efs_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/efs"
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
	req.Header.Set("X-Amz-Target", "elasticfilesystem."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/elasticfilesystem/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestCreateAccessPoint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateAccessPoint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateAccessPoint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateFileSystem(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateFileSystem", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateFileSystem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateMountTarget(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateMountTarget", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateMountTarget: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateReplicationConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateReplicationConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateReplicationConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateTags(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTags", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTags: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAccessPoint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteAccessPoint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteAccessPoint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFileSystem(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteFileSystem", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteFileSystem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFileSystemPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteFileSystemPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteFileSystemPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteMountTarget(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteMountTarget", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteMountTarget: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteReplicationConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteReplicationConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteReplicationConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTags(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTags", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTags: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAccessPoints(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAccessPoints", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAccessPoints: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAccountPreferences(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAccountPreferences", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAccountPreferences: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeBackupPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeBackupPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeBackupPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeFileSystemPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeFileSystemPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeFileSystemPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeFileSystems(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeFileSystems", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeFileSystems: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeLifecycleConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeLifecycleConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeLifecycleConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeMountTargetSecurityGroups(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeMountTargetSecurityGroups", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeMountTargetSecurityGroups: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeMountTargets(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeMountTargets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeMountTargets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeReplicationConfigurations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeReplicationConfigurations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeReplicationConfigurations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTags(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTags", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTags: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestModifyMountTargetSecurityGroups(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ModifyMountTargetSecurityGroups", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ModifyMountTargetSecurityGroups: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutAccountPreferences(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutAccountPreferences", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutAccountPreferences: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutBackupPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutBackupPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutBackupPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutFileSystemPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutFileSystemPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutFileSystemPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutLifecycleConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutLifecycleConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutLifecycleConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdateFileSystem(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFileSystem", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFileSystem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateFileSystemProtection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFileSystemProtection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFileSystemProtection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

