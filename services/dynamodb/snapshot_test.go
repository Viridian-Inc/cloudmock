package dynamodb_test

import (
	"encoding/json"
	"testing"

	ddbsvc "github.com/neureaux/cloudmock/services/dynamodb"
)

const (
	ddbTestAccount = "123456789012"
	ddbTestRegion  = "us-east-1"
)

func TestDynamoDB_ExportState_Empty(t *testing.T) {
	svc := ddbsvc.New(ddbTestAccount, ddbTestRegion)

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("ExportState returned invalid JSON: %s", raw)
	}

	var state struct {
		Tables []any `json:"tables"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(state.Tables) != 0 {
		t.Errorf("expected empty tables, got %d", len(state.Tables))
	}
}

func TestDynamoDB_ExportState_WithTableAndItems(t *testing.T) {
	svc := ddbsvc.New(ddbTestAccount, ddbTestRegion)

	seed := json.RawMessage(`{"tables":[{"name":"orders","key_schema":[{"AttributeName":"id","KeyType":"HASH"}],"attribute_definitions":[{"AttributeName":"id","AttributeType":"S"}],"billing_mode":"PAY_PER_REQUEST","items":[{"id":{"S":"order-1"},"total":{"N":"99"}},{"id":{"S":"order-2"},"total":{"N":"42"}}]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		Tables []struct {
			Name        string `json:"name"`
			BillingMode string `json:"billing_mode"`
			KeySchema   []struct {
				AttributeName string `json:"AttributeName"`
				KeyType       string `json:"KeyType"`
			} `json:"key_schema"`
			Items []any `json:"items"`
		} `json:"tables"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.Tables) == 0 {
		t.Fatal("expected tables in export")
	}
	tbl := state.Tables[0]
	if tbl.Name != "orders" {
		t.Errorf("expected table 'orders', got %q", tbl.Name)
	}
	if tbl.BillingMode != "PAY_PER_REQUEST" {
		t.Errorf("expected PAY_PER_REQUEST, got %q", tbl.BillingMode)
	}
	if len(tbl.KeySchema) == 0 {
		t.Error("expected key schema in export")
	}
	if tbl.KeySchema[0].AttributeName != "id" {
		t.Errorf("expected key 'id', got %q", tbl.KeySchema[0].AttributeName)
	}
	if len(tbl.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(tbl.Items))
	}
}

func TestDynamoDB_ImportState_RestoresTableSchema(t *testing.T) {
	svc := ddbsvc.New(ddbTestAccount, ddbTestRegion)

	data := json.RawMessage(`{"tables":[{"name":"sessions","key_schema":[{"AttributeName":"session_id","KeyType":"HASH"},{"AttributeName":"user_id","KeyType":"RANGE"}],"attribute_definitions":[{"AttributeName":"session_id","AttributeType":"S"},{"AttributeName":"user_id","AttributeType":"S"}],"billing_mode":"PAY_PER_REQUEST","items":[]}]}`)
	if err := svc.ImportState(data); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	names := svc.GetTableNames()
	found := false
	for _, n := range names {
		if n == "sessions" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("table 'sessions' not restored, got: %v", names)
	}
}

func TestDynamoDB_ImportState_EmptyDoesNotCrash(t *testing.T) {
	svc := ddbsvc.New(ddbTestAccount, ddbTestRegion)

	if err := svc.ImportState(json.RawMessage(`{"tables":[]}`)); err != nil {
		t.Fatalf("ImportState with empty tables: %v", err)
	}
	if len(svc.GetTableNames()) != 0 {
		t.Error("expected no tables after importing empty state")
	}
}

func TestDynamoDB_RoundTrip_PreservesItems(t *testing.T) {
	svc := ddbsvc.New(ddbTestAccount, ddbTestRegion)

	seed := json.RawMessage(`{"tables":[{"name":"products","key_schema":[{"AttributeName":"sku","KeyType":"HASH"}],"attribute_definitions":[{"AttributeName":"sku","AttributeType":"S"}],"billing_mode":"PAY_PER_REQUEST","items":[{"sku":{"S":"ABC"},"price":{"N":"10"}},{"sku":{"S":"XYZ"},"price":{"N":"20"}}]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	svc2 := ddbsvc.New(ddbTestAccount, ddbTestRegion)
	if err := svc2.ImportState(raw); err != nil {
		t.Fatalf("ImportState (svc2): %v", err)
	}

	names := svc2.GetTableNames()
	found := false
	for _, n := range names {
		if n == "products" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("table 'products' not restored: %v", names)
	}
}

func TestDynamoDB_RoundTrip_MultipleTables(t *testing.T) {
	svc := ddbsvc.New(ddbTestAccount, ddbTestRegion)

	seed := json.RawMessage(`{"tables":[
		{"name":"table-a","key_schema":[{"AttributeName":"pk","KeyType":"HASH"}],"attribute_definitions":[{"AttributeName":"pk","AttributeType":"S"}],"billing_mode":"PAY_PER_REQUEST","items":[]},
		{"name":"table-b","key_schema":[{"AttributeName":"id","KeyType":"HASH"}],"attribute_definitions":[{"AttributeName":"id","AttributeType":"N"}],"billing_mode":"PAY_PER_REQUEST","items":[]}
	]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	svc2 := ddbsvc.New(ddbTestAccount, ddbTestRegion)
	svc2.ImportState(raw)

	names := svc2.GetTableNames()
	for _, expected := range []string{"table-a", "table-b"} {
		found := false
		for _, n := range names {
			if n == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("table %q not restored, got: %v", expected, names)
		}
	}
}
