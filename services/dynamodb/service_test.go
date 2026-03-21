package dynamodb_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	ddbsvc "github.com/neureaux/cloudmock/services/dynamodb"
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
func ddbReq(t *testing.T, action string, body interface{}) *http.Request {
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
func decodeJSON(t *testing.T, data string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("decodeJSON: %v\nbody: %s", err, data)
	}
	return m
}

// createTestTable is a helper that creates a table with pk (HASH) and sk (RANGE) keys.
func createTestTable(t *testing.T, handler http.Handler, tableName string) {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]interface{}{
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
func putTestItem(t *testing.T, handler http.Handler, tableName string, item map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "PutItem", map[string]interface{}{
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
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]interface{}{
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
	td, ok := m["TableDescription"].(map[string]interface{})
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
	handler.ServeHTTP(w2, ddbReq(t, "DescribeTable", map[string]interface{}{
		"TableName": "Users",
	}))

	if w2.Code != http.StatusOK {
		t.Fatalf("DescribeTable: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	m2 := decodeJSON(t, w2.Body.String())
	table, ok := m2["Table"].(map[string]interface{})
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
	handler.ServeHTTP(w, ddbReq(t, "CreateTable", map[string]interface{}{
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
	putTestItem(t, handler, "Items", map[string]interface{}{
		"pk":   map[string]interface{}{"S": "user1"},
		"sk":   map[string]interface{}{"S": "profile"},
		"name": map[string]interface{}{"S": "Alice"},
		"age":  map[string]interface{}{"N": "30"},
	})

	// Get the item.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetItem", map[string]interface{}{
		"TableName": "Items",
		"Key": map[string]interface{}{
			"pk": map[string]interface{}{"S": "user1"},
			"sk": map[string]interface{}{"S": "profile"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("GetItem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	item, ok := m["Item"].(map[string]interface{})
	if !ok {
		t.Fatalf("GetItem: missing Item\nbody: %s", w.Body.String())
	}

	name := item["name"].(map[string]interface{})
	if name["S"] != "Alice" {
		t.Errorf("GetItem: expected name.S=Alice, got %v", name["S"])
	}

	age := item["age"].(map[string]interface{})
	if age["N"] != "30" {
		t.Errorf("GetItem: expected age.N=30, got %v", age["N"])
	}
}

func TestDDB_GetItem_NotFound(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "Items2")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetItem", map[string]interface{}{
		"TableName": "Items2",
		"Key": map[string]interface{}{
			"pk": map[string]interface{}{"S": "nonexistent"},
			"sk": map[string]interface{}{"S": "nope"},
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

	putTestItem(t, handler, "DelTest", map[string]interface{}{
		"pk": map[string]interface{}{"S": "u1"},
		"sk": map[string]interface{}{"S": "s1"},
	})

	// Delete the item.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DeleteItem", map[string]interface{}{
		"TableName": "DelTest",
		"Key": map[string]interface{}{
			"pk": map[string]interface{}{"S": "u1"},
			"sk": map[string]interface{}{"S": "s1"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("DeleteItem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify it's gone.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "GetItem", map[string]interface{}{
		"TableName": "DelTest",
		"Key": map[string]interface{}{
			"pk": map[string]interface{}{"S": "u1"},
			"sk": map[string]interface{}{"S": "s1"},
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

	putTestItem(t, handler, "UpdTest", map[string]interface{}{
		"pk":   map[string]interface{}{"S": "u1"},
		"sk":   map[string]interface{}{"S": "profile"},
		"name": map[string]interface{}{"S": "Alice"},
		"age":  map[string]interface{}{"N": "25"},
	})

	// Update with SET expression.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "UpdateItem", map[string]interface{}{
		"TableName": "UpdTest",
		"Key": map[string]interface{}{
			"pk": map[string]interface{}{"S": "u1"},
			"sk": map[string]interface{}{"S": "profile"},
		},
		"UpdateExpression": "SET #n = :name, #a = :age",
		"ExpressionAttributeNames": map[string]string{
			"#n": "name",
			"#a": "age",
		},
		"ExpressionAttributeValues": map[string]interface{}{
			":name": map[string]interface{}{"S": "Bob"},
			":age":  map[string]interface{}{"N": "30"},
		},
		"ReturnValues": "ALL_NEW",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("UpdateItem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	attrs, ok := m["Attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("UpdateItem: missing Attributes\nbody: %s", w.Body.String())
	}

	name := attrs["name"].(map[string]interface{})
	if name["S"] != "Bob" {
		t.Errorf("UpdateItem: expected name.S=Bob, got %v", name["S"])
	}

	age := attrs["age"].(map[string]interface{})
	if age["N"] != "30" {
		t.Errorf("UpdateItem: expected age.N=30, got %v", age["N"])
	}

	// Verify via GetItem.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "GetItem", map[string]interface{}{
		"TableName": "UpdTest",
		"Key": map[string]interface{}{
			"pk": map[string]interface{}{"S": "u1"},
			"sk": map[string]interface{}{"S": "profile"},
		},
	}))
	m2 := decodeJSON(t, w2.Body.String())
	item := m2["Item"].(map[string]interface{})
	gotName := item["name"].(map[string]interface{})
	if gotName["S"] != "Bob" {
		t.Errorf("UpdateItem verify: expected name.S=Bob, got %v", gotName["S"])
	}
}

func TestDDB_UpdateItem_REMOVE(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "RemoveTest")

	putTestItem(t, handler, "RemoveTest", map[string]interface{}{
		"pk":     map[string]interface{}{"S": "u1"},
		"sk":     map[string]interface{}{"S": "data"},
		"field1": map[string]interface{}{"S": "val1"},
		"field2": map[string]interface{}{"S": "val2"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "UpdateItem", map[string]interface{}{
		"TableName": "RemoveTest",
		"Key": map[string]interface{}{
			"pk": map[string]interface{}{"S": "u1"},
			"sk": map[string]interface{}{"S": "data"},
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
	attrs := m["Attributes"].(map[string]interface{})
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
		putTestItem(t, handler, "QueryTest", map[string]interface{}{
			"pk":    map[string]interface{}{"S": "user1"},
			"sk":    map[string]interface{}{"S": sk},
			"value": map[string]interface{}{"S": "val-" + sk},
		})
	}

	// Also insert an item for a different partition key.
	putTestItem(t, handler, "QueryTest", map[string]interface{}{
		"pk":    map[string]interface{}{"S": "user2"},
		"sk":    map[string]interface{}{"S": "x"},
		"value": map[string]interface{}{"S": "other"},
	})

	// Query for pk = user1 AND sk begins_with b.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Query", map[string]interface{}{
		"TableName":              "QueryTest",
		"KeyConditionExpression": "#pk = :pk AND begins_with(#sk, :prefix)",
		"ExpressionAttributeNames": map[string]string{
			"#pk": "pk",
			"#sk": "sk",
		},
		"ExpressionAttributeValues": map[string]interface{}{
			":pk":     map[string]interface{}{"S": "user1"},
			":prefix": map[string]interface{}{"S": "b"},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("Query: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	items := m["Items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("Query: expected 1 item, got %d\nbody: %s", len(items), w.Body.String())
	}

	item := items[0].(map[string]interface{})
	sk := item["sk"].(map[string]interface{})
	if sk["S"] != "b" {
		t.Errorf("Query: expected sk.S=b, got %v", sk["S"])
	}
}

func TestDDB_Query_EqualityOnly(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "QEqTest")

	putTestItem(t, handler, "QEqTest", map[string]interface{}{
		"pk": map[string]interface{}{"S": "u1"},
		"sk": map[string]interface{}{"S": "s1"},
	})
	putTestItem(t, handler, "QEqTest", map[string]interface{}{
		"pk": map[string]interface{}{"S": "u1"},
		"sk": map[string]interface{}{"S": "s2"},
	})
	putTestItem(t, handler, "QEqTest", map[string]interface{}{
		"pk": map[string]interface{}{"S": "u2"},
		"sk": map[string]interface{}{"S": "s1"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Query", map[string]interface{}{
		"TableName":              "QEqTest",
		"KeyConditionExpression": "pk = :pk",
		"ExpressionAttributeValues": map[string]interface{}{
			":pk": map[string]interface{}{"S": "u1"},
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
		putTestItem(t, handler, "BetweenTest", map[string]interface{}{
			"pk": map[string]interface{}{"S": "p1"},
			"sk": map[string]interface{}{"S": sk},
		})
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Query", map[string]interface{}{
		"TableName":              "BetweenTest",
		"KeyConditionExpression": "pk = :pk AND sk BETWEEN :lo AND :hi",
		"ExpressionAttributeValues": map[string]interface{}{
			":pk": map[string]interface{}{"S": "p1"},
			":lo": map[string]interface{}{"S": "050"},
			":hi": map[string]interface{}{"S": "150"},
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
		putTestItem(t, handler, "SortTest", map[string]interface{}{
			"pk": map[string]interface{}{"S": "p1"},
			"sk": map[string]interface{}{"S": sk},
		})
	}

	// Forward order.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Query", map[string]interface{}{
		"TableName":              "SortTest",
		"KeyConditionExpression": "pk = :pk",
		"ExpressionAttributeValues": map[string]interface{}{
			":pk": map[string]interface{}{"S": "p1"},
		},
		"ScanIndexForward": true,
	}))

	m := decodeJSON(t, w.Body.String())
	items := m["Items"].([]interface{})
	if len(items) != 3 {
		t.Fatalf("Query sort: expected 3 items, got %d", len(items))
	}

	firstSK := items[0].(map[string]interface{})["sk"].(map[string]interface{})["S"]
	lastSK := items[2].(map[string]interface{})["sk"].(map[string]interface{})["S"]
	if firstSK != "a" || lastSK != "c" {
		t.Errorf("Query sort forward: expected a..c, got %v..%v", firstSK, lastSK)
	}

	// Reverse order.
	w2 := httptest.NewRecorder()
	scanForward := false
	_ = scanForward
	handler.ServeHTTP(w2, ddbReq(t, "Query", map[string]interface{}{
		"TableName":              "SortTest",
		"KeyConditionExpression": "pk = :pk",
		"ExpressionAttributeValues": map[string]interface{}{
			":pk": map[string]interface{}{"S": "p1"},
		},
		"ScanIndexForward": false,
	}))

	m2 := decodeJSON(t, w2.Body.String())
	items2 := m2["Items"].([]interface{})
	firstSK2 := items2[0].(map[string]interface{})["sk"].(map[string]interface{})["S"]
	lastSK2 := items2[2].(map[string]interface{})["sk"].(map[string]interface{})["S"]
	if firstSK2 != "c" || lastSK2 != "a" {
		t.Errorf("Query sort reverse: expected c..a, got %v..%v", firstSK2, lastSK2)
	}
}

// ---- Test 6: Scan returns all items ----

func TestDDB_Scan(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "ScanTest")

	for i := 0; i < 5; i++ {
		putTestItem(t, handler, "ScanTest", map[string]interface{}{
			"pk": map[string]interface{}{"S": "user"},
			"sk": map[string]interface{}{"S": string(rune('a' + i))},
		})
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Scan", map[string]interface{}{
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

	putTestItem(t, handler, "ScanFilter", map[string]interface{}{
		"pk":     map[string]interface{}{"S": "u1"},
		"sk":     map[string]interface{}{"S": "s1"},
		"status": map[string]interface{}{"S": "active"},
	})
	putTestItem(t, handler, "ScanFilter", map[string]interface{}{
		"pk":     map[string]interface{}{"S": "u2"},
		"sk":     map[string]interface{}{"S": "s1"},
		"status": map[string]interface{}{"S": "inactive"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "Scan", map[string]interface{}{
		"TableName":        "ScanFilter",
		"FilterExpression": "#s = :status",
		"ExpressionAttributeNames": map[string]string{
			"#s": "status",
		},
		"ExpressionAttributeValues": map[string]interface{}{
			":status": map[string]interface{}{"S": "active"},
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
	handler.ServeHTTP(w, ddbReq(t, "DeleteTable", map[string]interface{}{
		"TableName": "ToDelete",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	td := m["TableDescription"].(map[string]interface{})
	if td["TableStatus"] != "DELETING" {
		t.Errorf("DeleteTable: expected TableStatus=DELETING, got %v", td["TableStatus"])
	}

	// Verify table is gone.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "DescribeTable", map[string]interface{}{
		"TableName": "ToDelete",
	}))
	if w2.Code != http.StatusBadRequest {
		t.Errorf("DeleteTable verify: expected 400 (ResourceNotFoundException), got %d", w2.Code)
	}
}

func TestDDB_DeleteTable_NotFound(t *testing.T) {
	handler := newDDBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DeleteTable", map[string]interface{}{
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
	names, ok := m["TableNames"].([]interface{})
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
	handler.ServeHTTP(w, ddbReq(t, "BatchWriteItem", map[string]interface{}{
		"RequestItems": map[string]interface{}{
			"BatchTest": []map[string]interface{}{
				{"PutRequest": map[string]interface{}{
					"Item": map[string]interface{}{
						"pk": map[string]interface{}{"S": "b1"},
						"sk": map[string]interface{}{"S": "s1"},
					},
				}},
				{"PutRequest": map[string]interface{}{
					"Item": map[string]interface{}{
						"pk": map[string]interface{}{"S": "b2"},
						"sk": map[string]interface{}{"S": "s1"},
					},
				}},
				{"PutRequest": map[string]interface{}{
					"Item": map[string]interface{}{
						"pk": map[string]interface{}{"S": "b3"},
						"sk": map[string]interface{}{"S": "s1"},
					},
				}},
			},
		},
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("BatchWriteItem: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	unprocessed := m["UnprocessedItems"].(map[string]interface{})
	if len(unprocessed) != 0 {
		t.Errorf("BatchWriteItem: expected 0 unprocessed items, got %d", len(unprocessed))
	}

	// BatchGetItem: get 2 of the 3 items.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ddbReq(t, "BatchGetItem", map[string]interface{}{
		"RequestItems": map[string]interface{}{
			"BatchTest": map[string]interface{}{
				"Keys": []map[string]interface{}{
					{
						"pk": map[string]interface{}{"S": "b1"},
						"sk": map[string]interface{}{"S": "s1"},
					},
					{
						"pk": map[string]interface{}{"S": "b3"},
						"sk": map[string]interface{}{"S": "s1"},
					},
				},
			},
		},
	}))

	if w2.Code != http.StatusOK {
		t.Fatalf("BatchGetItem: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	m2 := decodeJSON(t, w2.Body.String())
	responses := m2["Responses"].(map[string]interface{})
	batchItems := responses["BatchTest"].([]interface{})
	if len(batchItems) != 2 {
		t.Errorf("BatchGetItem: expected 2 items, got %d", len(batchItems))
	}
}

func TestDDB_BatchWriteItem_Delete(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "BWDel")

	putTestItem(t, handler, "BWDel", map[string]interface{}{
		"pk": map[string]interface{}{"S": "d1"},
		"sk": map[string]interface{}{"S": "s1"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "BatchWriteItem", map[string]interface{}{
		"RequestItems": map[string]interface{}{
			"BWDel": []map[string]interface{}{
				{"DeleteRequest": map[string]interface{}{
					"Key": map[string]interface{}{
						"pk": map[string]interface{}{"S": "d1"},
						"sk": map[string]interface{}{"S": "s1"},
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
	handler.ServeHTTP(w2, ddbReq(t, "GetItem", map[string]interface{}{
		"TableName": "BWDel",
		"Key": map[string]interface{}{
			"pk": map[string]interface{}{"S": "d1"},
			"sk": map[string]interface{}{"S": "s1"},
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

	putTestItem(t, handler, "ProjTest", map[string]interface{}{
		"pk":    map[string]interface{}{"S": "u1"},
		"sk":    map[string]interface{}{"S": "s1"},
		"name":  map[string]interface{}{"S": "Alice"},
		"email": map[string]interface{}{"S": "alice@example.com"},
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetItem", map[string]interface{}{
		"TableName": "ProjTest",
		"Key": map[string]interface{}{
			"pk": map[string]interface{}{"S": "u1"},
			"sk": map[string]interface{}{"S": "s1"},
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
	item := m["Item"].(map[string]interface{})
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
