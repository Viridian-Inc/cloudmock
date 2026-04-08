package dax_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	"github.com/Viridian-Inc/cloudmock/services/dax"
	dynamodbsvc "github.com/Viridian-Inc/cloudmock/services/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDataPlane(t *testing.T) (*dax.DataPlane, *dynamodbsvc.DynamoDBService) {
	t.Helper()
	ddbSvc := dynamodbsvc.New("123456789012", "us-east-1")
	daxSvc := dax.New("123456789012", "us-east-1")

	createTestTable(t, ddbSvc, "TestTable", "pk", "sk")
	putTestItem(t, ddbSvc, "TestTable", map[string]any{
		"pk": map[string]any{"S": "user1"}, "sk": map[string]any{"S": "profile"},
		"name": map[string]any{"S": "Alice"},
	})

	daxSvc.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName": "test-cluster", "NodeType": "dax.r4.large",
		"ReplicationFactor": 1, "IamRoleArn": "arn:aws:iam::123456789012:role/r",
	}))

	return dax.NewDataPlane(daxSvc, ddbSvc), ddbSvc
}

func createTestTable(t *testing.T, svc *dynamodbsvc.DynamoDBService, name, pk, sk string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"TableName": name,
		"KeySchema": []any{
			map[string]any{"AttributeName": pk, "KeyType": "HASH"},
			map[string]any{"AttributeName": sk, "KeyType": "RANGE"},
		},
		"AttributeDefinitions": []any{
			map[string]any{"AttributeName": pk, "AttributeType": "S"},
			map[string]any{"AttributeName": sk, "AttributeType": "S"},
		},
		"BillingMode": "PAY_PER_REQUEST",
	})
	ctx := &service.RequestContext{
		Action: "CreateTable", Body: body, Region: "us-east-1", AccountID: "123456789012",
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err := svc.HandleRequest(ctx)
	require.NoError(t, err)
}

func putTestItem(t *testing.T, svc *dynamodbsvc.DynamoDBService, table string, item map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"TableName": table, "Item": item})
	ctx := &service.RequestContext{
		Action: "PutItem", Body: body, Region: "us-east-1", AccountID: "123456789012",
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err := svc.HandleRequest(ctx)
	require.NoError(t, err)
}

func TestDataPlane_GetItemReadThrough(t *testing.T) {
	dp, _ := setupDataPlane(t)
	reqBody := `{"TableName":"TestTable","Key":{"pk":{"S":"user1"},"sk":{"S":"profile"}}}`

	// First call — miss
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req.Header.Set("X-Dax-Cluster", "test-cluster")
	w := httptest.NewRecorder()
	dp.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	item := resp["Item"].(map[string]any)
	assert.Equal(t, map[string]any{"S": "Alice"}, item["name"])

	// Second call — hit
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req2.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req2.Header.Set("X-Dax-Cluster", "test-cluster")
	w2 := httptest.NewRecorder()
	dp.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	stats := dp.ClusterStats("test-cluster")
	assert.Equal(t, int64(1), stats.ItemHits)
	assert.Equal(t, int64(1), stats.ItemMisses)
}

func TestDataPlane_PutItemInvalidatesCache(t *testing.T) {
	dp, _ := setupDataPlane(t)
	getBody := `{"TableName":"TestTable","Key":{"pk":{"S":"user1"},"sk":{"S":"profile"}}}`

	// Read into cache
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(getBody))
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req.Header.Set("X-Dax-Cluster", "test-cluster")
	w := httptest.NewRecorder()
	dp.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// PutItem with updated data
	putBody := `{"TableName":"TestTable","Item":{"pk":{"S":"user1"},"sk":{"S":"profile"},"name":{"S":"Bob"}}}`
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(putBody))
	req2.Header.Set("X-Amz-Target", "DynamoDB_20120810.PutItem")
	req2.Header.Set("X-Dax-Cluster", "test-cluster")
	w2 := httptest.NewRecorder()
	dp.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	// Read again — should miss (invalidated) and return Bob
	req3 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(getBody))
	req3.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req3.Header.Set("X-Dax-Cluster", "test-cluster")
	w3 := httptest.NewRecorder()
	dp.ServeHTTP(w3, req3)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &resp))
	item := resp["Item"].(map[string]any)
	assert.Equal(t, map[string]any{"S": "Bob"}, item["name"])

	stats := dp.ClusterStats("test-cluster")
	assert.True(t, stats.Invalidations >= 1)
	assert.True(t, stats.WriteThroughs >= 1)
}

func TestDataPlane_QueryReadThrough(t *testing.T) {
	dp, _ := setupDataPlane(t)
	queryBody := `{"TableName":"TestTable","KeyConditionExpression":"pk = :pk","ExpressionAttributeValues":{":pk":{"S":"user1"}}}`

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(queryBody))
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Query")
	req.Header.Set("X-Dax-Cluster", "test-cluster")
	w := httptest.NewRecorder()
	dp.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(queryBody))
	req2.Header.Set("X-Amz-Target", "DynamoDB_20120810.Query")
	req2.Header.Set("X-Dax-Cluster", "test-cluster")
	w2 := httptest.NewRecorder()
	dp.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	stats := dp.ClusterStats("test-cluster")
	assert.Equal(t, int64(1), stats.QueryHits)
	assert.Equal(t, int64(1), stats.QueryMisses)
}

func TestDataPlane_StatsEndpoint(t *testing.T) {
	dp, _ := setupDataPlane(t)

	getBody := `{"TableName":"TestTable","Key":{"pk":{"S":"user1"},"sk":{"S":"profile"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(getBody))
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req.Header.Set("X-Dax-Cluster", "test-cluster")
	w := httptest.NewRecorder()
	dp.ServeHTTP(w, req)

	statsReq := httptest.NewRequest(http.MethodGet, "/stats/test-cluster", nil)
	statsW := httptest.NewRecorder()
	dp.ServeHTTP(statsW, statsReq)
	assert.Equal(t, http.StatusOK, statsW.Code)

	var stats dax.CacheStats
	require.NoError(t, json.Unmarshal(statsW.Body.Bytes(), &stats))
	assert.Equal(t, int64(1), stats.ItemMisses)
	assert.Equal(t, int64(1), stats.ItemSize)
}
