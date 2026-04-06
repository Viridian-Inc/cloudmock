package dynamodb_test

import (
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

	// SELECT via PartiQL
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ExecuteStatement", map[string]any{
		"Statement": `SELECT * FROM "partiql-test" WHERE pk = 'user-1'`,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteStatement SELECT: %d %s", w.Code, w.Body.String())
	}
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
