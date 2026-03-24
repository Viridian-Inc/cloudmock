package dynamodb_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// createStreamTable creates a table with streams enabled.
func createStreamTable(t *testing.T, handler http.Handler, tableName string) {
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
		"StreamSpecification": map[string]any{
			"StreamEnabled":  true,
			"StreamViewType": "NEW_AND_OLD_IMAGES",
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("createStreamTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDDB_CreateTable_WithStream_DescribeTable(t *testing.T) {
	handler := newDDBGateway(t)
	createStreamTable(t, handler, "stream-table")

	// DescribeTable should show StreamSpecification
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DescribeTable", map[string]any{
		"TableName": "stream-table",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTable: expected 200, got %d", w.Code)
	}

	result := decodeJSON(t, w.Body.String())
	table := result["Table"].(map[string]any)

	spec, ok := table["StreamSpecification"].(map[string]any)
	if !ok {
		t.Fatal("expected StreamSpecification in DescribeTable response")
	}
	if spec["StreamEnabled"] != true {
		t.Error("expected StreamEnabled = true")
	}
	if spec["StreamViewType"] != "NEW_AND_OLD_IMAGES" {
		t.Errorf("expected StreamViewType = NEW_AND_OLD_IMAGES, got %v", spec["StreamViewType"])
	}
	if table["LatestStreamArn"] == nil || table["LatestStreamArn"] == "" {
		t.Error("expected LatestStreamArn to be set")
	}
}

func TestDDB_Stream_PutItem_GetRecords_INSERT(t *testing.T) {
	handler := newDDBGateway(t)
	createStreamTable(t, handler, "ins-stream")

	// Get the stream ARN from DescribeTable
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DescribeTable", map[string]any{
		"TableName": "ins-stream",
	}))
	table := decodeJSON(t, w.Body.String())["Table"].(map[string]any)
	streamARN := table["LatestStreamArn"].(string)

	// DescribeStream
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DescribeStream", map[string]any{
		"StreamArn": streamARN,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeStream: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	streamDesc := decodeJSON(t, w.Body.String())["StreamDescription"].(map[string]any)
	shards := streamDesc["Shards"].([]any)
	if len(shards) == 0 {
		t.Fatal("expected at least one shard")
	}
	shardId := shards[0].(map[string]any)["ShardId"].(string)

	// GetShardIterator
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetShardIterator", map[string]any{
		"StreamArn":         streamARN,
		"ShardId":           shardId,
		"ShardIteratorType": "TRIM_HORIZON",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetShardIterator: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	iterator := decodeJSON(t, w.Body.String())["ShardIterator"].(string)

	// PutItem
	putTestItem(t, handler, "ins-stream", map[string]any{
		"pk": map[string]any{"S": "user1"},
		"sk": map[string]any{"S": "profile"},
	})

	// GetRecords
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetRecords", map[string]any{
		"ShardIterator": iterator,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetRecords: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	recordsResp := decodeJSON(t, w.Body.String())
	records := recordsResp["Records"].([]any)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	rec := records[0].(map[string]any)
	if rec["eventName"] != "INSERT" {
		t.Errorf("expected INSERT event, got %v", rec["eventName"])
	}
	if rec["NewImage"] == nil {
		t.Error("expected NewImage in INSERT record")
	}
}

func TestDDB_Stream_UpdateItem_MODIFY(t *testing.T) {
	handler := newDDBGateway(t)
	createStreamTable(t, handler, "mod-stream")

	// Put initial item
	putTestItem(t, handler, "mod-stream", map[string]any{
		"pk":   map[string]any{"S": "user1"},
		"sk":   map[string]any{"S": "profile"},
		"name": map[string]any{"S": "Alice"},
	})

	// Get stream ARN and iterator (after the INSERT)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DescribeTable", map[string]any{
		"TableName": "mod-stream",
	}))
	table := decodeJSON(t, w.Body.String())["Table"].(map[string]any)
	streamARN := table["LatestStreamArn"].(string)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DescribeStream", map[string]any{
		"StreamArn": streamARN,
	}))
	shards := decodeJSON(t, w.Body.String())["StreamDescription"].(map[string]any)["Shards"].([]any)
	shardId := shards[0].(map[string]any)["ShardId"].(string)

	// Get iterator at LATEST to skip INSERT
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetShardIterator", map[string]any{
		"StreamArn":         streamARN,
		"ShardId":           shardId,
		"ShardIteratorType": "LATEST",
	}))
	iterator := decodeJSON(t, w.Body.String())["ShardIterator"].(string)

	// UpdateItem
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "UpdateItem", map[string]any{
		"TableName": "mod-stream",
		"Key": map[string]any{
			"pk": map[string]any{"S": "user1"},
			"sk": map[string]any{"S": "profile"},
		},
		"UpdateExpression":          "SET #n = :val",
		"ExpressionAttributeNames":  map[string]string{"#n": "name"},
		"ExpressionAttributeValues": map[string]any{":val": map[string]any{"S": "Bob"}},
		"ReturnValues":              "ALL_NEW",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateItem: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	// GetRecords should show MODIFY
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetRecords", map[string]any{
		"ShardIterator": iterator,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetRecords: expected 200, got %d – %s", w.Code, w.Body.String())
	}
	records := decodeJSON(t, w.Body.String())["Records"].([]any)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	rec := records[0].(map[string]any)
	if rec["eventName"] != "MODIFY" {
		t.Errorf("expected MODIFY event, got %v", rec["eventName"])
	}
	if rec["OldImage"] == nil {
		t.Error("expected OldImage in MODIFY record")
	}
	if rec["NewImage"] == nil {
		t.Error("expected NewImage in MODIFY record")
	}
}

func TestDDB_Stream_DeleteItem_REMOVE(t *testing.T) {
	handler := newDDBGateway(t)
	createStreamTable(t, handler, "rem-stream")

	// Put item
	putTestItem(t, handler, "rem-stream", map[string]any{
		"pk": map[string]any{"S": "user1"},
		"sk": map[string]any{"S": "profile"},
	})

	// Get stream ARN and LATEST iterator
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DescribeTable", map[string]any{
		"TableName": "rem-stream",
	}))
	table := decodeJSON(t, w.Body.String())["Table"].(map[string]any)
	streamARN := table["LatestStreamArn"].(string)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DescribeStream", map[string]any{
		"StreamArn": streamARN,
	}))
	shards := decodeJSON(t, w.Body.String())["StreamDescription"].(map[string]any)["Shards"].([]any)
	shardId := shards[0].(map[string]any)["ShardId"].(string)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetShardIterator", map[string]any{
		"StreamArn":         streamARN,
		"ShardId":           shardId,
		"ShardIteratorType": "LATEST",
	}))
	iterator := decodeJSON(t, w.Body.String())["ShardIterator"].(string)

	// DeleteItem
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DeleteItem", map[string]any{
		"TableName": "rem-stream",
		"Key": map[string]any{
			"pk": map[string]any{"S": "user1"},
			"sk": map[string]any{"S": "profile"},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteItem: expected 200, got %d", w.Code)
	}

	// GetRecords should show REMOVE
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetRecords", map[string]any{
		"ShardIterator": iterator,
	}))
	records := decodeJSON(t, w.Body.String())["Records"].([]any)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	rec := records[0].(map[string]any)
	if rec["eventName"] != "REMOVE" {
		t.Errorf("expected REMOVE event, got %v", rec["eventName"])
	}
	if rec["OldImage"] == nil {
		t.Error("expected OldImage in REMOVE record")
	}
}

func TestDDB_TTL_UpdateAndDescribe(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "ttl-table")

	// UpdateTimeToLive
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "UpdateTimeToLive", map[string]any{
		"TableName": "ttl-table",
		"TimeToLiveSpecification": map[string]any{
			"AttributeName": "expiresAt",
			"Enabled":       true,
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTimeToLive: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	// DescribeTimeToLive
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "DescribeTimeToLive", map[string]any{
		"TableName": "ttl-table",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTimeToLive: expected 200, got %d – %s", w.Code, w.Body.String())
	}

	result := decodeJSON(t, w.Body.String())
	desc := result["TimeToLiveDescription"].(map[string]any)
	if desc["TimeToLiveStatus"] != "ENABLED" {
		t.Errorf("expected ENABLED, got %v", desc["TimeToLiveStatus"])
	}
	if desc["AttributeName"] != "expiresAt" {
		t.Errorf("expected expiresAt, got %v", desc["AttributeName"])
	}
}

func TestDDB_TTL_ExpiredItemDeleted(t *testing.T) {
	handler := newDDBGateway(t)
	createTestTable(t, handler, "ttl-exp-table")

	// Enable TTL
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "UpdateTimeToLive", map[string]any{
		"TableName": "ttl-exp-table",
		"TimeToLiveSpecification": map[string]any{
			"AttributeName": "ttl",
			"Enabled":       true,
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTimeToLive: expected 200, got %d", w.Code)
	}

	// Put item with TTL in the past
	pastTTL := fmt.Sprintf("%d", time.Now().Unix()-100)
	putTestItem(t, handler, "ttl-exp-table", map[string]any{
		"pk":  map[string]any{"S": "expired"},
		"sk":  map[string]any{"S": "item"},
		"ttl": map[string]any{"N": pastTTL},
	})

	// Put item with TTL far in the future (should survive)
	futureTTL := fmt.Sprintf("%d", time.Now().Unix()+100000)
	putTestItem(t, handler, "ttl-exp-table", map[string]any{
		"pk":  map[string]any{"S": "alive"},
		"sk":  map[string]any{"S": "item"},
		"ttl": map[string]any{"N": futureTTL},
	})

	// Wait for TTL reaper (runs every 5 seconds).
	time.Sleep(7 * time.Second)

	// GetItem for expired item → should be gone
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetItem", map[string]any{
		"TableName": "ttl-exp-table",
		"Key": map[string]any{
			"pk": map[string]any{"S": "expired"},
			"sk": map[string]any{"S": "item"},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetItem expired: expected 200, got %d", w.Code)
	}
	result := decodeJSON(t, w.Body.String())
	if result["Item"] != nil {
		t.Error("expected expired item to be deleted by TTL reaper")
	}

	// GetItem for alive item → should still exist
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ddbReq(t, "GetItem", map[string]any{
		"TableName": "ttl-exp-table",
		"Key": map[string]any{
			"pk": map[string]any{"S": "alive"},
			"sk": map[string]any{"S": "item"},
		},
	}))
	result = decodeJSON(t, w.Body.String())
	if result["Item"] == nil {
		t.Error("expected alive item to still exist")
	}
}
