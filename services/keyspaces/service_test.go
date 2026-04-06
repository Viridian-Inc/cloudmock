package keyspaces_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	svc "github.com/neureaux/cloudmock/services/keyspaces"
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
	req.Header.Set("X-Amz-Target", "KeyspacesService."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/cassandra/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestCreateKeyspace(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateKeyspace", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateKeyspace: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateTable(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTable", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateType(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateType", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateType: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteKeyspace(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteKeyspace", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteKeyspace: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTable(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTable", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteType(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteType", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteType: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetKeyspace(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetKeyspace", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetKeyspace: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetTable(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetTable", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetTableAutoScalingSettings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetTableAutoScalingSettings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetTableAutoScalingSettings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetType(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetType", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetType: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListKeyspaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListKeyspaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListKeyspaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTables(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTables", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTables: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListTypes(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTypes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTypes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestRestoreTable(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "RestoreTable", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("RestoreTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdateKeyspace(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateKeyspace", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateKeyspace: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTable(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTable", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

