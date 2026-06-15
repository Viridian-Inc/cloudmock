package dynamodb_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---- PartiQL: ExecuteStatement ----

func TestDDB_ExecuteStatement_Insert(t *testing.T) {
	handler := newDDBGateway(t)

	// Create table first
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName":            "partiql-test",
		"KeySchema":            []map[string]string{{"AttributeName": "pk", "KeyType": "HASH"}},
		"AttributeDefinitions": []map[string]string{{"AttributeName": "pk", "AttributeType": "S"}},
		"BillingMode":          "PAY_PER_REQUEST",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: %d %s", w.Code, w.Body.String())
	}

	// INSERT via PartiQL
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ExecuteStatement", map[string]any{
		"Statement": `INSERT INTO "partiql-test" VALUE {'pk': 'user-1', 'name': 'Alice'}`,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteStatement INSERT: %d %s", w.Code, w.Body.String())
	}

	// SELECT via PartiQL — must return the item we just inserted.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ExecuteStatement", map[string]any{
		"Statement": `SELECT * FROM "partiql-test" WHERE pk = 'user-1'`,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteStatement SELECT: %d %s", w.Code, w.Body.String())
	}
	items := executeStatementItems(t, w)
	if len(items) != 1 {
		t.Fatalf("SELECT returned %d items, want 1: %s", len(items), w.Body.String())
	}
	if got := attrS(items[0], "pk"); got != "user-1" {
		t.Errorf("pk = %q, want user-1", got)
	}
	if got := attrS(items[0], "name"); got != "Alice" {
		t.Errorf("name = %q, want Alice", got)
	}

	// SELECT with a non-matching condition returns no items.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ExecuteStatement", map[string]any{
		"Statement": `SELECT * FROM "partiql-test" WHERE pk = 'nobody'`,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteStatement SELECT(no match): %d %s", w.Code, w.Body.String())
	}
	if items := executeStatementItems(t, w); len(items) != 0 {
		t.Errorf("non-matching SELECT returned %d items, want 0", len(items))
	}
}

func TestDDB_ExecuteStatement_Parameters(t *testing.T) {
	handler := newDDBGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName":            "pq-params",
		"KeySchema":            []map[string]string{{"AttributeName": "pk", "KeyType": "HASH"}},
		"AttributeDefinitions": []map[string]string{{"AttributeName": "pk", "AttributeType": "S"}},
		"BillingMode":          "PAY_PER_REQUEST",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: %d %s", w.Code, w.Body.String())
	}

	// INSERT using a ? placeholder bound to a typed Parameter.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ExecuteStatement", map[string]any{
		"Statement":  `INSERT INTO "pq-params" VALUE {'pk': ?}`,
		"Parameters": []any{map[string]any{"S": "p-1"}},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteStatement INSERT(param): %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ExecuteStatement", map[string]any{
		"Statement":  `SELECT * FROM "pq-params" WHERE pk = ?`,
		"Parameters": []any{map[string]any{"S": "p-1"}},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteStatement SELECT(param): %d %s", w.Code, w.Body.String())
	}
	if items := executeStatementItems(t, w); len(items) != 1 || attrS(items[0], "pk") != "p-1" {
		t.Errorf("parameterized SELECT mismatch: %s", w.Body.String())
	}
}

func TestDDB_ExecuteStatement_Unsupported(t *testing.T) {
	handler := newDDBGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ExecuteStatement", map[string]any{
		"Statement": `UPDATE "x" SET a = 1 WHERE pk = 'y'`,
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unsupported statement: status %d, want 400; %s", w.Code, w.Body.String())
	}
}

// executeStatementItems parses the Items array from an ExecuteStatement response.
func executeStatementItems(t *testing.T, w *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var resp struct {
		Items []map[string]any `json:"Items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse ExecuteStatement response: %v (%s)", err, w.Body.String())
	}
	return resp.Items
}

// attrS returns the string ("S") value of an attribute in a typed item.
func attrS(item map[string]any, attr string) string {
	av, ok := item[attr].(map[string]any)
	if !ok {
		return ""
	}
	s, _ := av["S"].(string)
	return s
}

// ---- Backups ----

func TestDDB_CreateBackup(t *testing.T) {
	handler := newDDBGateway(t)

	// Create table
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName":            "backup-test",
		"KeySchema":            []map[string]string{{"AttributeName": "pk", "KeyType": "HASH"}},
		"AttributeDefinitions": []map[string]string{{"AttributeName": "pk", "AttributeType": "S"}},
		"BillingMode":          "PAY_PER_REQUEST",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: %d %s", w.Code, w.Body.String())
	}

	// CreateBackup
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateBackup", map[string]any{
		"TableName":  "backup-test",
		"BackupName": "my-backup",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateBackup: %d %s", w.Code, w.Body.String())
	}

	// ListBackups
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ListBackups", map[string]any{
		"TableName": "backup-test",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ListBackups: %d %s", w.Code, w.Body.String())
	}
}

// ---- Global Tables ----

func TestDDB_GlobalTables(t *testing.T) {
	handler := newDDBGateway(t)

	// Create table
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName":            "global-test",
		"KeySchema":            []map[string]string{{"AttributeName": "pk", "KeyType": "HASH"}},
		"AttributeDefinitions": []map[string]string{{"AttributeName": "pk", "AttributeType": "S"}},
		"BillingMode":          "PAY_PER_REQUEST",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: %d %s", w.Code, w.Body.String())
	}

	// CreateGlobalTable
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateGlobalTable", map[string]any{
		"GlobalTableName": "global-test",
		"ReplicationGroup": []map[string]string{
			{"RegionName": "us-east-1"},
			{"RegionName": "eu-west-1"},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateGlobalTable: %d %s", w.Code, w.Body.String())
	}

	// DescribeGlobalTable
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DescribeGlobalTable", map[string]any{
		"GlobalTableName": "global-test",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeGlobalTable: %d %s", w.Code, w.Body.String())
	}

	// ListGlobalTables
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ListGlobalTables", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListGlobalTables: %d %s", w.Code, w.Body.String())
	}
}

// ---- Exports ----

func TestDDB_ExportTable(t *testing.T) {
	handler := newDDBGateway(t)

	// Create table
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName":            "export-test",
		"KeySchema":            []map[string]string{{"AttributeName": "pk", "KeyType": "HASH"}},
		"AttributeDefinitions": []map[string]string{{"AttributeName": "pk", "AttributeType": "S"}},
		"BillingMode":          "PAY_PER_REQUEST",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: %d %s", w.Code, w.Body.String())
	}

	// ExportTableToPointInTime
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ExportTableToPointInTime", map[string]any{
		"TableArn":     "arn:aws:dynamodb:us-east-1:000000000000:table/export-test",
		"S3Bucket":     "my-export-bucket",
		"ExportFormat": "DYNAMODB_JSON",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ExportTableToPointInTime: %d %s", w.Code, w.Body.String())
	}

	// ListExports
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ListExports", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListExports: %d %s", w.Code, w.Body.String())
	}
}
