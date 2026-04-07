package dynamodb_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	ddbsvc "github.com/Viridian-Inc/cloudmock/services/dynamodb"
)

// newDDBGateway builds a full gateway stack with the DynamoDB service registered and IAM disabled.
func newDDBGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(ddbsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// ddbReq builds a JSON POST request targeting the DynamoDB service via X-Amz-Target.
func ddbReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("ddbReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc123")
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

// createTestTable is a helper that creates a table with pk (HASH) and sk (RANGE) keys.
func createTestTable(t *testing.T, handler http.Handler, tableName string) {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName": tableName,
		"KeySchema": []map[string]string{
			{"AttributeName": "pk", "KeyType": "HASH"},
			{"AttributeName": "sk", "KeyType": "RANGE"},
		},
		"AttributeDefinitions": []map[string]string{
			{"AttributeName": "pk", "AttributeType": "S"},
			{"AttributeName": "sk", "AttributeType": "S"},
		},
		"BillingMode": "PAY_PER_REQUEST",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("createTestTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// putTestItem is a helper that puts an item into a table.
func putTestItem(t *testing.T, handler http.Handler, tableName string, item map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "PutItem", map[string]any{
		"TableName": tableName,
		"Item":      item,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("putTestItem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 1: CreateTable + DescribeTable ----

func TestDDB_CreateTable_DescribeTable(t *testing.T) {
	handler := newDDBGateway(t)

	// Create table.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName": "Users",
		"KeySchema": []map[string]string{
			{"AttributeName": "userId", "KeyType": "HASH"},
		},
		"AttributeDefinitions": []map[string]string{
			{"AttributeName": "userId", "AttributeType": "S"},
		},
		"BillingMode": "PAY_PER_REQUEST",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	td, ok := m["TableDescription"].(map[string]any)
	if !ok {
		t.Fatalf("CreateTable: missing TableDescription\nbody: %s", w.Body.String())
	}
	if td["TableName"] != "Users" {
		t.Errorf("CreateTable: expected TableName=Users, got %v", td["TableName"])
	}
	if td["TableStatus"] != "ACTIVE" {
		t.Errorf("CreateTable: expected TableStatus=ACTIVE, got %v", td["TableStatus"])
	}
	if td["TableArn"] == nil || td["TableArn"] == "" {
		t.Errorf("CreateTable: missing TableArn")
	}

	// Describe table.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "DescribeTable", map[string]any{
		"TableName": "Users",
	}))

	if w2.Code != http.StatusOK {
		t.Fatalf("DescribeTable: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	m2 := decodeJSON(t, w2.Body.String())
	table, ok := m2["Table"].(map[string]any)
	if !ok {
		t.Fatalf("DescribeTable: missing Table\nbody: %s", w2.Body.String())
	}
	if table["TableName"] != "Users" {
		t.Errorf("DescribeTable: expected TableName=Users, got %v", table["TableName"])
	}
}

func TestDDB_CreateTable_AlreadyExists(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "Dupes")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName": "Dupes",
		"KeySchema": []map[string]string{
			{"AttributeName": "pk", "KeyType": "HASH"},
		},
		"AttributeDefinitions": []map[string]string{
			{"AttributeName": "pk", "AttributeType": "S"},
		},
	}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateTable duplicate: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 2: PutItem + GetItem round-trip ----

func TestDDB_PutItem_GetItem(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "Items")

	// Put an item.
	putTestItem(t, handler, "Items", map[string]any{
		"pk":   map[string]any{"S": "user1"},
		"sk":   map[string]any{"S": "profile"},
		"name": map[string]any{"S": "Alice"},
		"age":  map[string]any{"N": "30"},
	})

	// Get the item.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetItem", map[string]any{
		"TableName": "Items",
		"Key": map[string]any{
			"pk": map[string]any{"S": "user1"},
			"sk": map[string]any{"S": "profile"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("GetItem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	item, ok := m["Item"].(map[string]any)
	if !ok {
		t.Fatalf("GetItem: missing Item\nbody: %s", w.Body.String())
	}

	name := item["name"].(map[string]any)
	if name["S"] != "Alice" {
		t.Errorf("GetItem: expected name.S=Alice, got %v", name["S"])
	}

	age := item["age"].(map[string]any)
	if age["N"] != "30" {
		t.Errorf("GetItem: expected age.N=30, got %v", age["N"])
	}
}

func TestDDB_GetItem_NotFound(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "Items2")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetItem", map[string]any{
		"TableName": "Items2",
		"Key": map[string]any{
			"pk": map[string]any{"S": "nonexistent"},
			"sk": map[string]any{"S": "nope"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("GetItem not found: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Item should be absent or null.
	m := decodeJSON(t, w.Body.String())
	if m["Item"] != nil {
		t.Errorf("GetItem not found: expected no Item, got %v", m["Item"])
	}
}

// ---- Test 3: DeleteItem ----

func TestDDB_DeleteItem(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "DelTest")

	putTestItem(t, handler, "DelTest", map[string]any{
		"pk": map[string]any{"S": "u1"},
		"sk": map[string]any{"S": "s1"},
	})

	// Delete the item.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DeleteItem", map[string]any{
		"TableName": "DelTest",
		"Key": map[string]any{
			"pk": map[string]any{"S": "u1"},
			"sk": map[string]any{"S": "s1"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("DeleteItem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify it's gone.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "GetItem", map[string]any{
		"TableName": "DelTest",
		"Key": map[string]any{
			"pk": map[string]any{"S": "u1"},
			"sk": map[string]any{"S": "s1"},
		},
	}))
	m := decodeJSON(t, w2.Body.String())
	if m["Item"] != nil {
		t.Errorf("DeleteItem: item still exists after delete")
	}
}

// ---- Test 4: UpdateItem with SET expression ----

func TestDDB_UpdateItem_SET(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "UpdTest")

	putTestItem(t, handler, "UpdTest", map[string]any{
		"pk":   map[string]any{"S": "u1"},
		"sk":   map[string]any{"S": "profile"},
		"name": map[string]any{"S": "Alice"},
		"age":  map[string]any{"N": "25"},
	})

	// Update with SET expression.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "UpdateItem", map[string]any{
		"TableName": "UpdTest",
		"Key": map[string]any{
			"pk": map[string]any{"S": "u1"},
			"sk": map[string]any{"S": "profile"},
		},
		"UpdateExpression": "SET #n = :name, #a = :age",
		"ExpressionAttributeNames": map[string]string{
			"#n": "name",
			"#a": "age",
		},
		"ExpressionAttributeValues": map[string]any{
			":name": map[string]any{"S": "Bob"},
			":age":  map[string]any{"N": "30"},
		},
		"ReturnValues": "ALL_NEW",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("UpdateItem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	attrs, ok := m["Attributes"].(map[string]any)
	if !ok {
		t.Fatalf("UpdateItem: missing Attributes\nbody: %s", w.Body.String())
	}

	name := attrs["name"].(map[string]any)
	if name["S"] != "Bob" {
		t.Errorf("UpdateItem: expected name.S=Bob, got %v", name["S"])
	}

	age := attrs["age"].(map[string]any)
	if age["N"] != "30" {
		t.Errorf("UpdateItem: expected age.N=30, got %v", age["N"])
	}

	// Verify via GetItem.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "GetItem", map[string]any{
		"TableName": "UpdTest",
		"Key": map[string]any{
			"pk": map[string]any{"S": "u1"},
			"sk": map[string]any{"S": "profile"},
		},
	}))
	m2 := decodeJSON(t, w2.Body.String())
	item := m2["Item"].(map[string]any)
	gotName := item["name"].(map[string]any)
	if gotName["S"] != "Bob" {
		t.Errorf("UpdateItem verify: expected name.S=Bob, got %v", gotName["S"])
	}
}

func TestDDB_UpdateItem_REMOVE(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "RemoveTest")

	putTestItem(t, handler, "RemoveTest", map[string]any{
		"pk":     map[string]any{"S": "u1"},
		"sk":     map[string]any{"S": "data"},
		"field1": map[string]any{"S": "val1"},
		"field2": map[string]any{"S": "val2"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "UpdateItem", map[string]any{
		"TableName": "RemoveTest",
		"Key": map[string]any{
			"pk": map[string]any{"S": "u1"},
			"sk": map[string]any{"S": "data"},
		},
		"UpdateExpression": "REMOVE #f",
		"ExpressionAttributeNames": map[string]string{
			"#f": "field1",
		},
		"ReturnValues": "ALL_NEW",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("UpdateItem REMOVE: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	attrs := m["Attributes"].(map[string]any)
	if _, exists := attrs["field1"]; exists {
		t.Errorf("UpdateItem REMOVE: field1 should have been removed")
	}
	if _, exists := attrs["field2"]; !exists {
		t.Errorf("UpdateItem REMOVE: field2 should still exist")
	}
}

// ---- Test 5: Query with KeyConditionExpression ----

func TestDDB_Query_KeyCondition(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "QueryTest")

	// Insert items with different sort keys.
	for _, sk := range []string{"a", "b", "c", "d"} {
		putTestItem(t, handler, "QueryTest", map[string]any{
			"pk":    map[string]any{"S": "user1"},
			"sk":    map[string]any{"S": sk},
			"value": map[string]any{"S": "val-" + sk},
		})
	}

	// Also insert an item for a different partition key.
	putTestItem(t, handler, "QueryTest", map[string]any{
		"pk":    map[string]any{"S": "user2"},
		"sk":    map[string]any{"S": "x"},
		"value": map[string]any{"S": "other"},
	})

	// Query for pk = user1 AND sk begins_with b.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Query", map[string]any{
		"TableName":              "QueryTest",
		"KeyConditionExpression": "#pk = :pk AND begins_with(#sk, :prefix)",
		"ExpressionAttributeNames": map[string]string{
			"#pk": "pk",
			"#sk": "sk",
		},
		"ExpressionAttributeValues": map[string]any{
			":pk":     map[string]any{"S": "user1"},
			":prefix": map[string]any{"S": "b"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("Query: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	items := m["Items"].([]any)
	if len(items) != 1 {
		t.Fatalf("Query: expected 1 item, got %d\nbody: %s", len(items), w.Body.String())
	}

	item := items[0].(map[string]any)
	sk := item["sk"].(map[string]any)
	if sk["S"] != "b" {
		t.Errorf("Query: expected sk.S=b, got %v", sk["S"])
	}
}

func TestDDB_Query_EqualityOnly(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "QEqTest")

	putTestItem(t, handler, "QEqTest", map[string]any{
		"pk": map[string]any{"S": "u1"},
		"sk": map[string]any{"S": "s1"},
	})
	putTestItem(t, handler, "QEqTest", map[string]any{
		"pk": map[string]any{"S": "u1"},
		"sk": map[string]any{"S": "s2"},
	})
	putTestItem(t, handler, "QEqTest", map[string]any{
		"pk": map[string]any{"S": "u2"},
		"sk": map[string]any{"S": "s1"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Query", map[string]any{
		"TableName":              "QEqTest",
		"KeyConditionExpression": "pk = :pk",
		"ExpressionAttributeValues": map[string]any{
			":pk": map[string]any{"S": "u1"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("Query equality: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	count := int(m["Count"].(float64))
	if count != 2 {
		t.Errorf("Query equality: expected Count=2, got %d", count)
	}
}

func TestDDB_Query_Between(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "BetweenTest")

	for _, sk := range []string{"001", "050", "100", "150", "200"} {
		putTestItem(t, handler, "BetweenTest", map[string]any{
			"pk": map[string]any{"S": "p1"},
			"sk": map[string]any{"S": sk},
		})
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Query", map[string]any{
		"TableName":              "BetweenTest",
		"KeyConditionExpression": "pk = :pk AND sk BETWEEN :lo AND :hi",
		"ExpressionAttributeValues": map[string]any{
			":pk": map[string]any{"S": "p1"},
			":lo": map[string]any{"S": "050"},
			":hi": map[string]any{"S": "150"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("Query BETWEEN: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	count := int(m["Count"].(float64))
	if count != 3 {
		t.Errorf("Query BETWEEN: expected Count=3 (050, 100, 150), got %d", count)
	}
}

func TestDDB_Query_SortOrder(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "SortTest")

	for _, sk := range []string{"c", "a", "b"} {
		putTestItem(t, handler, "SortTest", map[string]any{
			"pk": map[string]any{"S": "p1"},
			"sk": map[string]any{"S": sk},
		})
	}

	// Forward order.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Query", map[string]any{
		"TableName":              "SortTest",
		"KeyConditionExpression": "pk = :pk",
		"ExpressionAttributeValues": map[string]any{
			":pk": map[string]any{"S": "p1"},
		},
		"ScanIndexForward": true,
	}))

	m := decodeJSON(t, w.Body.String())
	items := m["Items"].([]any)
	if len(items) != 3 {
		t.Fatalf("Query sort: expected 3 items, got %d", len(items))
	}

	firstSK := items[0].(map[string]any)["sk"].(map[string]any)["S"]
	lastSK := items[2].(map[string]any)["sk"].(map[string]any)["S"]
	if firstSK != "a" || lastSK != "c" {
		t.Errorf("Query sort forward: expected a..c, got %v..%v", firstSK, lastSK)
	}

	// Reverse order.
	w2 := httptest.NewRecorder()
	scanForward := false
	_ = scanForward
	handler.ServeHTTP(w2, ddbReq(t, "Query", map[string]any{
		"TableName":              "SortTest",
		"KeyConditionExpression": "pk = :pk",
		"ExpressionAttributeValues": map[string]any{
			":pk": map[string]any{"S": "p1"},
		},
		"ScanIndexForward": false,
	}))

	m2 := decodeJSON(t, w2.Body.String())
	items2 := m2["Items"].([]any)
	firstSK2 := items2[0].(map[string]any)["sk"].(map[string]any)["S"]
	lastSK2 := items2[2].(map[string]any)["sk"].(map[string]any)["S"]
	if firstSK2 != "c" || lastSK2 != "a" {
		t.Errorf("Query sort reverse: expected c..a, got %v..%v", firstSK2, lastSK2)
	}
}

// ---- Test 6: Scan returns all items ----

func TestDDB_Scan(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "ScanTest")

	for i := 0; i < 5; i++ {
		putTestItem(t, handler, "ScanTest", map[string]any{
			"pk": map[string]any{"S": "user"},
			"sk": map[string]any{"S": string(rune('a' + i))},
		})
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Scan", map[string]any{
		"TableName": "ScanTest",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("Scan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	count := int(m["Count"].(float64))
	if count != 5 {
		t.Errorf("Scan: expected Count=5, got %d", count)
	}

	scanned := int(m["ScannedCount"].(float64))
	if scanned != 5 {
		t.Errorf("Scan: expected ScannedCount=5, got %d", scanned)
	}
}

func TestDDB_Scan_WithFilter(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "ScanFilter")

	putTestItem(t, handler, "ScanFilter", map[string]any{
		"pk":     map[string]any{"S": "u1"},
		"sk":     map[string]any{"S": "s1"},
		"status": map[string]any{"S": "active"},
	})
	putTestItem(t, handler, "ScanFilter", map[string]any{
		"pk":     map[string]any{"S": "u2"},
		"sk":     map[string]any{"S": "s1"},
		"status": map[string]any{"S": "inactive"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Scan", map[string]any{
		"TableName":        "ScanFilter",
		"FilterExpression": "#s = :status",
		"ExpressionAttributeNames": map[string]string{
			"#s": "status",
		},
		"ExpressionAttributeValues": map[string]any{
			":status": map[string]any{"S": "active"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("Scan with filter: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	count := int(m["Count"].(float64))
	if count != 1 {
		t.Errorf("Scan with filter: expected Count=1, got %d", count)
	}
}

// ---- Test 7: DeleteTable ----

func TestDDB_DeleteTable(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "ToDelete")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DeleteTable", map[string]any{
		"TableName": "ToDelete",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	td := m["TableDescription"].(map[string]any)
	if td["TableStatus"] != "DELETING" {
		t.Errorf("DeleteTable: expected TableStatus=DELETING, got %v", td["TableStatus"])
	}

	// Verify table is gone.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "DescribeTable", map[string]any{
		"TableName": "ToDelete",
	}))
	if w2.Code != http.StatusBadRequest {
		t.Errorf("DeleteTable verify: expected 400 (ResourceNotFoundException), got %d", w2.Code)
	}
}

func TestDDB_DeleteTable_NotFound(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DeleteTable", map[string]any{
		"TableName": "NonExistent",
	}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("DeleteTable not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 8: ListTables ----

func TestDDB_ListTables(t *testing.T) {
	handler := newDDBGateway(t)

	// Create two tables.
	createTestTable(t, handler, "Alpha")
	createTestTable(t, handler, "Beta")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "ListTables", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("ListTables: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	names, ok := m["TableNames"].([]any)
	if !ok {
		t.Fatalf("ListTables: missing TableNames\nbody: %s", w.Body.String())
	}

	if len(names) < 2 {
		t.Errorf("ListTables: expected at least 2 tables, got %d", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n.(string)] = true
	}
	if !nameSet["Alpha"] || !nameSet["Beta"] {
		t.Errorf("ListTables: missing expected tables, got %v", names)
	}
}

// ---- Test 9: BatchWriteItem + BatchGetItem ----

func TestDDB_BatchWriteItem_BatchGetItem(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "BatchTest")

	// BatchWriteItem: put 3 items.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "BatchWriteItem", map[string]any{
		"RequestItems": map[string]any{
			"BatchTest": []map[string]any{
				{"PutRequest": map[string]any{
					"Item": map[string]any{
						"pk": map[string]any{"S": "b1"},
						"sk": map[string]any{"S": "s1"},
					},
				}},
				{"PutRequest": map[string]any{
					"Item": map[string]any{
						"pk": map[string]any{"S": "b2"},
						"sk": map[string]any{"S": "s1"},
					},
				}},
				{"PutRequest": map[string]any{
					"Item": map[string]any{
						"pk": map[string]any{"S": "b3"},
						"sk": map[string]any{"S": "s1"},
					},
				}},
			},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("BatchWriteItem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	unprocessed := m["UnprocessedItems"].(map[string]any)
	if len(unprocessed) != 0 {
		t.Errorf("BatchWriteItem: expected 0 unprocessed items, got %d", len(unprocessed))
	}

	// BatchGetItem: get 2 of the 3 items.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "BatchGetItem", map[string]any{
		"RequestItems": map[string]any{
			"BatchTest": map[string]any{
				"Keys": []map[string]any{
					{
						"pk": map[string]any{"S": "b1"},
						"sk": map[string]any{"S": "s1"},
					},
					{
						"pk": map[string]any{"S": "b3"},
						"sk": map[string]any{"S": "s1"},
					},
				},
			},
		},
	}))

	if w2.Code != http.StatusOK {
		t.Fatalf("BatchGetItem: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	m2 := decodeJSON(t, w2.Body.String())
	responses := m2["Responses"].(map[string]any)
	batchItems := responses["BatchTest"].([]any)
	if len(batchItems) != 2 {
		t.Errorf("BatchGetItem: expected 2 items, got %d", len(batchItems))
	}
}

func TestDDB_BatchWriteItem_Delete(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "BWDel")

	putTestItem(t, handler, "BWDel", map[string]any{
		"pk": map[string]any{"S": "d1"},
		"sk": map[string]any{"S": "s1"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "BatchWriteItem", map[string]any{
		"RequestItems": map[string]any{
			"BWDel": []map[string]any{
				{"DeleteRequest": map[string]any{
					"Key": map[string]any{
						"pk": map[string]any{"S": "d1"},
						"sk": map[string]any{"S": "s1"},
					},
				}},
			},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("BatchWriteItem delete: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify deleted.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "GetItem", map[string]any{
		"TableName": "BWDel",
		"Key": map[string]any{
			"pk": map[string]any{"S": "d1"},
			"sk": map[string]any{"S": "s1"},
		},
	}))
	m := decodeJSON(t, w2.Body.String())
	if m["Item"] != nil {
		t.Errorf("BatchWriteItem delete: item still exists")
	}
}

// ---- Projection Expression ----

func TestDDB_GetItem_Projection(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "ProjTest")

	putTestItem(t, handler, "ProjTest", map[string]any{
		"pk":    map[string]any{"S": "u1"},
		"sk":    map[string]any{"S": "s1"},
		"name":  map[string]any{"S": "Alice"},
		"email": map[string]any{"S": "alice@example.com"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetItem", map[string]any{
		"TableName": "ProjTest",
		"Key": map[string]any{
			"pk": map[string]any{"S": "u1"},
			"sk": map[string]any{"S": "s1"},
		},
		"ProjectionExpression": "#n",
		"ExpressionAttributeNames": map[string]string{
			"#n": "name",
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("GetItem projection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	item := m["Item"].(map[string]any)
	if _, exists := item["name"]; !exists {
		t.Error("GetItem projection: name should exist")
	}
	if _, exists := item["email"]; exists {
		t.Error("GetItem projection: email should not exist")
	}
}

// ---- Unknown Action ----

func TestDDB_UnknownAction(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- GSI Tests ----

func TestDDB_CreateTable_WithGSI_DescribeTable(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName": "GSITable",
		"KeySchema": []map[string]string{
			{"AttributeName": "pk", "KeyType": "HASH"},
			{"AttributeName": "sk", "KeyType": "RANGE"},
		},
		"AttributeDefinitions": []map[string]string{
			{"AttributeName": "pk", "AttributeType": "S"},
			{"AttributeName": "sk", "AttributeType": "S"},
			{"AttributeName": "gsiPk", "AttributeType": "S"},
			{"AttributeName": "gsiSk", "AttributeType": "S"},
		},
		"BillingMode": "PAY_PER_REQUEST",
		"GlobalSecondaryIndexes": []map[string]any{
			{
				"IndexName": "gsi-index",
				"KeySchema": []map[string]string{
					{"AttributeName": "gsiPk", "KeyType": "HASH"},
					{"AttributeName": "gsiSk", "KeyType": "RANGE"},
				},
				"Projection": map[string]string{
					"ProjectionType": "ALL",
				},
			},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable with GSI: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// DescribeTable should show the GSI.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "DescribeTable", map[string]any{
		"TableName": "GSITable",
	}))

	if w2.Code != http.StatusOK {
		t.Fatalf("DescribeTable: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	m := decodeJSON(t, w2.Body.String())
	table := m["Table"].(map[string]any)
	gsis, ok := table["GlobalSecondaryIndexes"].([]any)
	if !ok || len(gsis) != 1 {
		t.Fatalf("DescribeTable: expected 1 GSI, got %v", table["GlobalSecondaryIndexes"])
	}
	gsi := gsis[0].(map[string]any)
	if gsi["IndexName"] != "gsi-index" {
		t.Errorf("DescribeTable: expected IndexName=gsi-index, got %v", gsi["IndexName"])
	}
	if gsi["IndexStatus"] != "ACTIVE" {
		t.Errorf("DescribeTable: expected IndexStatus=ACTIVE, got %v", gsi["IndexStatus"])
	}
}

func TestDDB_Query_GSI(t *testing.T) {
	handler := newDDBGateway(t)

	// Create table with GSI.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName": "GSIQuery",
		"KeySchema": []map[string]string{
			{"AttributeName": "pk", "KeyType": "HASH"},
			{"AttributeName": "sk", "KeyType": "RANGE"},
		},
		"AttributeDefinitions": []map[string]string{
			{"AttributeName": "pk", "AttributeType": "S"},
			{"AttributeName": "sk", "AttributeType": "S"},
			{"AttributeName": "status", "AttributeType": "S"},
			{"AttributeName": "createdAt", "AttributeType": "S"},
		},
		"BillingMode": "PAY_PER_REQUEST",
		"GlobalSecondaryIndexes": []map[string]any{
			{
				"IndexName": "status-createdAt-index",
				"KeySchema": []map[string]string{
					{"AttributeName": "status", "KeyType": "HASH"},
					{"AttributeName": "createdAt", "KeyType": "RANGE"},
				},
				"Projection": map[string]string{
					"ProjectionType": "ALL",
				},
			},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Put items with different statuses.
	putTestItem(t, handler, "GSIQuery", map[string]any{
		"pk":        map[string]any{"S": "order1"},
		"sk":        map[string]any{"S": "detail"},
		"status":    map[string]any{"S": "active"},
		"createdAt": map[string]any{"S": "2024-01-01"},
	})
	putTestItem(t, handler, "GSIQuery", map[string]any{
		"pk":        map[string]any{"S": "order2"},
		"sk":        map[string]any{"S": "detail"},
		"status":    map[string]any{"S": "active"},
		"createdAt": map[string]any{"S": "2024-01-02"},
	})
	putTestItem(t, handler, "GSIQuery", map[string]any{
		"pk":        map[string]any{"S": "order3"},
		"sk":        map[string]any{"S": "detail"},
		"status":    map[string]any{"S": "completed"},
		"createdAt": map[string]any{"S": "2024-01-03"},
	})

	// Query GSI for active orders.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "Query", map[string]any{
		"TableName":              "GSIQuery",
		"IndexName":              "status-createdAt-index",
		"KeyConditionExpression": "#s = :status",
		"ExpressionAttributeNames": map[string]string{
			"#s": "status",
		},
		"ExpressionAttributeValues": map[string]any{
			":status": map[string]any{"S": "active"},
		},
	}))

	if w2.Code != http.StatusOK {
		t.Fatalf("Query GSI: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	m := decodeJSON(t, w2.Body.String())
	count := int(m["Count"].(float64))
	if count != 2 {
		t.Errorf("Query GSI: expected 2 active items, got %d\nbody: %s", count, w2.Body.String())
	}
}

func TestDDB_Query_GSI_WithSortKeyCondition(t *testing.T) {
	handler := newDDBGateway(t)

	// Create table with GSI.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName": "GSISortQuery",
		"KeySchema": []map[string]string{
			{"AttributeName": "pk", "KeyType": "HASH"},
			{"AttributeName": "sk", "KeyType": "RANGE"},
		},
		"AttributeDefinitions": []map[string]string{
			{"AttributeName": "pk", "AttributeType": "S"},
			{"AttributeName": "sk", "AttributeType": "S"},
			{"AttributeName": "category", "AttributeType": "S"},
			{"AttributeName": "price", "AttributeType": "N"},
		},
		"BillingMode": "PAY_PER_REQUEST",
		"GlobalSecondaryIndexes": []map[string]any{
			{
				"IndexName": "category-price-index",
				"KeySchema": []map[string]string{
					{"AttributeName": "category", "KeyType": "HASH"},
					{"AttributeName": "price", "KeyType": "RANGE"},
				},
				"Projection": map[string]string{
					"ProjectionType": "ALL",
				},
			},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Put items.
	putTestItem(t, handler, "GSISortQuery", map[string]any{
		"pk":       map[string]any{"S": "p1"},
		"sk":       map[string]any{"S": "s1"},
		"category": map[string]any{"S": "electronics"},
		"price":    map[string]any{"N": "100"},
	})
	putTestItem(t, handler, "GSISortQuery", map[string]any{
		"pk":       map[string]any{"S": "p2"},
		"sk":       map[string]any{"S": "s1"},
		"category": map[string]any{"S": "electronics"},
		"price":    map[string]any{"N": "200"},
	})
	putTestItem(t, handler, "GSISortQuery", map[string]any{
		"pk":       map[string]any{"S": "p3"},
		"sk":       map[string]any{"S": "s1"},
		"category": map[string]any{"S": "electronics"},
		"price":    map[string]any{"N": "50"},
	})

	// Query GSI with sort key condition: price > 75.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "Query", map[string]any{
		"TableName":              "GSISortQuery",
		"IndexName":              "category-price-index",
		"KeyConditionExpression": "category = :cat AND price > :minPrice",
		"ExpressionAttributeValues": map[string]any{
			":cat":      map[string]any{"S": "electronics"},
			":minPrice": map[string]any{"N": "75"},
		},
	}))

	if w2.Code != http.StatusOK {
		t.Fatalf("Query GSI sort key: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	m := decodeJSON(t, w2.Body.String())
	count := int(m["Count"].(float64))
	if count != 2 {
		t.Errorf("Query GSI sort key: expected 2 items (price > 75), got %d", count)
	}

	// Verify items are sorted by price ascending (default).
	items := m["Items"].([]any)
	price0 := items[0].(map[string]any)["price"].(map[string]any)["N"]
	price1 := items[1].(map[string]any)["price"].(map[string]any)["N"]
	if price0 != "100" || price1 != "200" {
		t.Errorf("Query GSI sort key: expected prices 100, 200, got %v, %v", price0, price1)
	}
}

// ---- Transaction Tests ----

func TestDDB_TransactWriteItems_PutAndDelete(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "TxTable1")
	createTestTable(t, handler, "TxTable2")

	// Pre-populate an item to delete.
	putTestItem(t, handler, "TxTable2", map[string]any{
		"pk":   map[string]any{"S": "del1"},
		"sk":   map[string]any{"S": "s1"},
		"data": map[string]any{"S": "to-be-deleted"},
	})

	// TransactWriteItems: Put into TxTable1, Delete from TxTable2.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "TransactWriteItems", map[string]any{
		"TransactItems": []map[string]any{
			{
				"Put": map[string]any{
					"TableName": "TxTable1",
					"Item": map[string]any{
						"pk":   map[string]any{"S": "new1"},
						"sk":   map[string]any{"S": "s1"},
						"data": map[string]any{"S": "created-in-tx"},
					},
				},
			},
			{
				"Delete": map[string]any{
					"TableName": "TxTable2",
					"Key": map[string]any{
						"pk": map[string]any{"S": "del1"},
						"sk": map[string]any{"S": "s1"},
					},
				},
			},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("TransactWriteItems: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify put succeeded.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "GetItem", map[string]any{
		"TableName": "TxTable1",
		"Key": map[string]any{
			"pk": map[string]any{"S": "new1"},
			"sk": map[string]any{"S": "s1"},
		},
	}))
	m := decodeJSON(t, w2.Body.String())
	if m["Item"] == nil {
		t.Error("TransactWriteItems: put item not found")
	}

	// Verify delete succeeded.
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, ddbReq(t, "GetItem", map[string]any{
		"TableName": "TxTable2",
		"Key": map[string]any{
			"pk": map[string]any{"S": "del1"},
			"sk": map[string]any{"S": "s1"},
		},
	}))
	m2 := decodeJSON(t, w3.Body.String())
	if m2["Item"] != nil {
		t.Error("TransactWriteItems: deleted item still exists")
	}
}

func TestDDB_TransactWriteItems_ConditionCheckFailure(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "TxCondTable")

	// Put an item with status=active.
	putTestItem(t, handler, "TxCondTable", map[string]any{
		"pk":     map[string]any{"S": "item1"},
		"sk":     map[string]any{"S": "s1"},
		"status": map[string]any{"S": "active"},
	})

	// TransactWriteItems with ConditionCheck that will fail (expects status=completed).
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "TransactWriteItems", map[string]any{
		"TransactItems": []map[string]any{
			{
				"ConditionCheck": map[string]any{
					"TableName": "TxCondTable",
					"Key": map[string]any{
						"pk": map[string]any{"S": "item1"},
						"sk": map[string]any{"S": "s1"},
					},
					"ConditionExpression": "#s = :expected",
					"ExpressionAttributeNames": map[string]string{
						"#s": "status",
					},
					"ExpressionAttributeValues": map[string]any{
						":expected": map[string]any{"S": "completed"},
					},
				},
			},
			{
				"Put": map[string]any{
					"TableName": "TxCondTable",
					"Item": map[string]any{
						"pk":     map[string]any{"S": "item2"},
						"sk":     map[string]any{"S": "s1"},
						"status": map[string]any{"S": "new"},
					},
				},
			},
		},
	}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("TransactWriteItems condition failure: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	errType, _ := m["__type"].(string)
	if errType != "TransactionCanceledException" {
		t.Errorf("TransactWriteItems: expected TransactionCanceledException, got %v", errType)
	}

	// Verify the Put was rolled back (item2 should not exist).
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "GetItem", map[string]any{
		"TableName": "TxCondTable",
		"Key": map[string]any{
			"pk": map[string]any{"S": "item2"},
			"sk": map[string]any{"S": "s1"},
		},
	}))
	m2 := decodeJSON(t, w2.Body.String())
	if m2["Item"] != nil {
		t.Error("TransactWriteItems: put should have been rolled back but item2 exists")
	}
}

func TestDDB_TransactGetItems(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "TxGetTable1")
	createTestTable(t, handler, "TxGetTable2")

	putTestItem(t, handler, "TxGetTable1", map[string]any{
		"pk":   map[string]any{"S": "a1"},
		"sk":   map[string]any{"S": "s1"},
		"name": map[string]any{"S": "Alice"},
	})
	putTestItem(t, handler, "TxGetTable2", map[string]any{
		"pk":   map[string]any{"S": "b1"},
		"sk":   map[string]any{"S": "s1"},
		"name": map[string]any{"S": "Bob"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "TransactGetItems", map[string]any{
		"TransactItems": []map[string]any{
			{
				"Get": map[string]any{
					"TableName": "TxGetTable1",
					"Key": map[string]any{
						"pk": map[string]any{"S": "a1"},
						"sk": map[string]any{"S": "s1"},
					},
				},
			},
			{
				"Get": map[string]any{
					"TableName": "TxGetTable2",
					"Key": map[string]any{
						"pk": map[string]any{"S": "b1"},
						"sk": map[string]any{"S": "s1"},
					},
				},
			},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("TransactGetItems: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	responses, ok := m["Responses"].([]any)
	if !ok || len(responses) != 2 {
		t.Fatalf("TransactGetItems: expected 2 responses, got %v", m["Responses"])
	}

	r0 := responses[0].(map[string]any)
	item0 := r0["Item"].(map[string]any)
	name0 := item0["name"].(map[string]any)["S"]
	if name0 != "Alice" {
		t.Errorf("TransactGetItems: expected first item name=Alice, got %v", name0)
	}

	r1 := responses[1].(map[string]any)
	item1 := r1["Item"].(map[string]any)
	name1 := item1["name"].(map[string]any)["S"]
	if name1 != "Bob" {
		t.Errorf("TransactGetItems: expected second item name=Bob, got %v", name1)
	}
}

// ---- Verification: ResourceNotFoundException — PutItem on nonexistent table ----

func TestDDB_PutItem_ResourceNotFoundException(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "PutItem", map[string]any{
		"TableName": "NonExistentTable",
		"Item": map[string]any{
			"pk": map[string]any{"S": "test"},
			"sk": map[string]any{"S": "test"},
		},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PutItem nonexistent table: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	errType, _ := m["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("PutItem nonexistent table: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Verification: ResourceNotFoundException — GetItem on nonexistent table ----

func TestDDB_GetItem_ResourceNotFoundException(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetItem", map[string]any{
		"TableName": "NonExistentTable",
		"Key": map[string]any{
			"pk": map[string]any{"S": "test"},
			"sk": map[string]any{"S": "test"},
		},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("GetItem nonexistent table: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	errType, _ := m["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("GetItem nonexistent table: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Verification: ResourceNotFoundException — Query on nonexistent table ----

func TestDDB_Query_ResourceNotFoundException(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Query", map[string]any{
		"TableName":              "NonExistentTable",
		"KeyConditionExpression": "pk = :pk",
		"ExpressionAttributeValues": map[string]any{
			":pk": map[string]any{"S": "test"},
		},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Query nonexistent table: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	errType, _ := m["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("Query nonexistent table: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Verification: ResourceNotFoundException — Scan on nonexistent table ----

func TestDDB_Scan_ResourceNotFoundException(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Scan", map[string]any{
		"TableName": "NonExistentTable",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Scan nonexistent table: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	errType, _ := m["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("Scan nonexistent table: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Verification: ResourceNotFoundException — DeleteItem on nonexistent table ----

func TestDDB_DeleteItem_ResourceNotFoundException(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DeleteItem", map[string]any{
		"TableName": "NonExistentTable",
		"Key": map[string]any{
			"pk": map[string]any{"S": "test"},
			"sk": map[string]any{"S": "test"},
		},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DeleteItem nonexistent table: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	errType, _ := m["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("DeleteItem nonexistent table: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Verification: ResourceNotFoundException — UpdateItem on nonexistent table ----

func TestDDB_UpdateItem_ResourceNotFoundException(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "UpdateItem", map[string]any{
		"TableName": "NonExistentTable",
		"Key": map[string]any{
			"pk": map[string]any{"S": "test"},
			"sk": map[string]any{"S": "test"},
		},
		"UpdateExpression": "SET #n = :v",
		"ExpressionAttributeNames": map[string]string{
			"#n": "name",
		},
		"ExpressionAttributeValues": map[string]any{
			":v": map[string]any{"S": "test"},
		},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UpdateItem nonexistent table: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	errType, _ := m["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("UpdateItem nonexistent table: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Verification: ValidationException — CreateTable missing key schema ----

func TestDDB_CreateTable_ValidationException(t *testing.T) {
	handler := newDDBGateway(t)

	// Missing KeySchema entirely.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]any{
		"TableName": "bad-table",
		"AttributeDefinitions": []map[string]string{
			{"AttributeName": "pk", "AttributeType": "S"},
		},
		"BillingMode": "PAY_PER_REQUEST",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateTable missing KeySchema: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	errType, _ := m["__type"].(string)
	if errType != "ValidationException" {
		t.Errorf("CreateTable validation: expected ValidationException, got %q", errType)
	}
}

// ---- Verification: ResourceNotFoundException — TransactWriteItems on nonexistent table ----

func TestDDB_TransactWriteItems_ResourceNotFoundException(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "TransactWriteItems", map[string]any{
		"TransactItems": []map[string]any{
			{
				"Put": map[string]any{
					"TableName": "NonExistentTxTable",
					"Item": map[string]any{
						"pk": map[string]any{"S": "test"},
						"sk": map[string]any{"S": "test"},
					},
				},
			},
		},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("TransactWriteItems nonexistent table: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
