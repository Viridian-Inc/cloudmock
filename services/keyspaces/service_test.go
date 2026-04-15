package keyspaces_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/keyspaces"
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
	req.Header.Set("X-Amz-Target", "KeyspacesService."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/cassandra/aws4_request, SignedHeaders=host, Signature=abc")
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

func simpleSchema() map[string]any {
	return map[string]any{
		"allColumns": []map[string]any{
			{"name": "id", "type": "uuid"},
			{"name": "value", "type": "text"},
		},
		"partitionKeys": []map[string]any{
			{"name": "id"},
		},
	}
}

func TestKeyspaceLifecycle(t *testing.T) {
	h := newGateway(t)

	if w := doCall(t, h, "CreateKeyspace", map[string]any{
		"keyspaceName": "my_ks",
		"replicationSpecification": map[string]any{
			"replicationStrategy": "SINGLE_REGION",
		},
	}); w.Code != http.StatusOK {
		t.Fatalf("CreateKeyspace: want 200, got %d: %s", w.Code, w.Body.String())
	}

	w := doCall(t, h, "GetKeyspace", map[string]any{"keyspaceName": "my_ks"})
	if w.Code != http.StatusOK {
		t.Fatalf("GetKeyspace: want 200, got %d", w.Code)
	}

	w = doCall(t, h, "ListKeyspaces", nil)
	var listed struct {
		Keyspaces []struct {
			KeyspaceName string `json:"keyspaceName"`
		} `json:"keyspaces"`
	}
	decode(t, w, &listed)
	if len(listed.Keyspaces) != 1 || listed.Keyspaces[0].KeyspaceName != "my_ks" {
		t.Fatalf("unexpected: %+v", listed)
	}

	if w := doCall(t, h, "UpdateKeyspace", map[string]any{
		"keyspaceName": "my_ks",
		"replicationSpecification": map[string]any{
			"replicationStrategy": "MULTI_REGION",
			"regionList":          []string{"us-east-1", "us-west-2"},
		},
	}); w.Code != http.StatusOK {
		t.Fatalf("UpdateKeyspace: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "DeleteKeyspace", map[string]any{"keyspaceName": "my_ks"}); w.Code != http.StatusOK {
		t.Fatalf("DeleteKeyspace: want 200, got %d", w.Code)
	}
	if w := doCall(t, h, "GetKeyspace", map[string]any{"keyspaceName": "my_ks"}); w.Code != http.StatusNotFound {
		t.Fatalf("GetKeyspace after delete: want 404, got %d", w.Code)
	}
}

func TestTableLifecycle(t *testing.T) {
	h := newGateway(t)

	doCall(t, h, "CreateKeyspace", map[string]any{"keyspaceName": "ks1"})

	w := doCall(t, h, "CreateTable", map[string]any{
		"keyspaceName":     "ks1",
		"tableName":        "t1",
		"schemaDefinition": simpleSchema(),
		"capacitySpecification": map[string]any{
			"throughputMode": "PAY_PER_REQUEST",
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: want 200, got %d: %s", w.Code, w.Body.String())
	}

	if w := doCall(t, h, "GetTable", map[string]any{"keyspaceName": "ks1", "tableName": "t1"}); w.Code != http.StatusOK {
		t.Fatalf("GetTable: want 200, got %d", w.Code)
	}

	w = doCall(t, h, "ListTables", map[string]any{"keyspaceName": "ks1"})
	var listed struct {
		Tables []struct {
			TableName string `json:"tableName"`
		} `json:"tables"`
	}
	decode(t, w, &listed)
	if len(listed.Tables) != 1 || listed.Tables[0].TableName != "t1" {
		t.Fatalf("unexpected tables: %+v", listed)
	}

	if w := doCall(t, h, "GetTableAutoScalingSettings", map[string]any{"keyspaceName": "ks1", "tableName": "t1"}); w.Code != http.StatusOK {
		t.Fatalf("GetTableAutoScalingSettings: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "UpdateTable", map[string]any{
		"keyspaceName":      "ks1",
		"tableName":         "t1",
		"defaultTimeToLive": 60,
	}); w.Code != http.StatusOK {
		t.Fatalf("UpdateTable: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "RestoreTable", map[string]any{
		"sourceKeyspaceName": "ks1",
		"sourceTableName":    "t1",
		"targetKeyspaceName": "ks1",
		"targetTableName":    "t1_restored",
	}); w.Code != http.StatusOK {
		t.Fatalf("RestoreTable: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "DeleteTable", map[string]any{"keyspaceName": "ks1", "tableName": "t1"}); w.Code != http.StatusOK {
		t.Fatalf("DeleteTable: want 200, got %d", w.Code)
	}
}

func TestTypeLifecycle(t *testing.T) {
	h := newGateway(t)
	doCall(t, h, "CreateKeyspace", map[string]any{"keyspaceName": "ks"})

	w := doCall(t, h, "CreateType", map[string]any{
		"keyspaceName": "ks",
		"typeName":     "address",
		"fieldDefinitions": []map[string]any{
			{"name": "street", "type": "text"},
			{"name": "zip", "type": "text"},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("CreateType: want 200, got %d: %s", w.Code, w.Body.String())
	}

	if w := doCall(t, h, "GetType", map[string]any{"keyspaceName": "ks", "typeName": "address"}); w.Code != http.StatusOK {
		t.Fatalf("GetType: want 200, got %d", w.Code)
	}

	w = doCall(t, h, "ListTypes", map[string]any{"keyspaceName": "ks"})
	var listed struct {
		Types []string `json:"types"`
	}
	decode(t, w, &listed)
	if len(listed.Types) != 1 || listed.Types[0] != "address" {
		t.Fatalf("unexpected types: %+v", listed)
	}

	if w := doCall(t, h, "DeleteType", map[string]any{"keyspaceName": "ks", "typeName": "address"}); w.Code != http.StatusOK {
		t.Fatalf("DeleteType: want 200, got %d", w.Code)
	}
}

func TestTagsLifecycle(t *testing.T) {
	h := newGateway(t)
	arn := "arn:aws:cassandra:us-east-1:000000000000:/keyspace/ks"

	if w := doCall(t, h, "TagResource", map[string]any{
		"resourceArn": arn,
		"tags": []map[string]any{
			{"key": "env", "value": "dev"},
		},
	}); w.Code != http.StatusOK {
		t.Fatalf("TagResource: want 200, got %d", w.Code)
	}

	w := doCall(t, h, "ListTagsForResource", map[string]any{"resourceArn": arn})
	var listed struct {
		Tags []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"tags"`
	}
	decode(t, w, &listed)
	if len(listed.Tags) != 1 || listed.Tags[0].Value != "dev" {
		t.Fatalf("unexpected tags: %+v", listed)
	}

	if w := doCall(t, h, "UntagResource", map[string]any{
		"resourceArn": arn,
		"tags": []map[string]any{
			{"key": "env"},
		},
	}); w.Code != http.StatusOK {
		t.Fatalf("UntagResource: want 200, got %d", w.Code)
	}
}
