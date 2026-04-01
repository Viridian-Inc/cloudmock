package athena_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/athena"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.AthenaService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestServiceName(t *testing.T) {
	s := newService()
	assert.Equal(t, "athena", s.Name())
}

func TestHealthCheck(t *testing.T) {
	s := newService()
	assert.NoError(t, s.HealthCheck())
}

func TestCreateWorkGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{
		"Name": "test-wg", "Description": "test workgroup",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateWorkGroupDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{"Name": "dup-wg"}))
	_, err := s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{"Name": "dup-wg"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "AlreadyExists", awsErr.Code)
}

func TestCreateWorkGroupMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{}))
	require.Error(t, err)
}

func TestGetWorkGroup(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{"Name": "my-wg", "Description": "desc"}))
	resp, err := s.HandleRequest(jsonCtx("GetWorkGroup", map[string]any{"WorkGroup": "my-wg"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := respJSON(t, resp)
	wg := m["WorkGroup"].(map[string]any)
	assert.Equal(t, "my-wg", wg["Name"])
	assert.Equal(t, "ENABLED", wg["State"])
}

func TestGetWorkGroupNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetWorkGroup", map[string]any{"WorkGroup": "nonexistent"}))
	require.Error(t, err)
}

func TestListWorkGroups(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{"Name": "wg1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{"Name": "wg2"}))
	resp, err := s.HandleRequest(jsonCtx("ListWorkGroups", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	wgs := m["WorkGroups"].([]any)
	assert.GreaterOrEqual(t, len(wgs), 3) // primary + wg1 + wg2
}

func TestUpdateWorkGroup(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{"Name": "upd-wg"}))
	resp, err := s.HandleRequest(jsonCtx("UpdateWorkGroup", map[string]any{
		"WorkGroup": "upd-wg", "Description": "updated", "State": "DISABLED",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	getResp, err := s.HandleRequest(jsonCtx("GetWorkGroup", map[string]any{"WorkGroup": "upd-wg"}))
	require.NoError(t, err)
	m := respJSON(t, getResp)
	wg := m["WorkGroup"].(map[string]any)
	assert.Equal(t, "DISABLED", wg["State"])
	assert.Equal(t, "updated", wg["Description"])
}

func TestDeleteWorkGroup(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{"Name": "del-wg"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteWorkGroup", map[string]any{"WorkGroup": "del-wg"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, err = s.HandleRequest(jsonCtx("GetWorkGroup", map[string]any{"WorkGroup": "del-wg"}))
	require.Error(t, err)
}

func TestDeleteWorkGroupNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteWorkGroup", map[string]any{"WorkGroup": "ghost"}))
	require.Error(t, err)
}

func TestCreateNamedQuery(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateNamedQuery", map[string]any{
		"Name": "q1", "Database": "default", "QueryString": "SELECT 1",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotEmpty(t, m["NamedQueryId"])
}

func TestGetNamedQuery(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateNamedQuery", map[string]any{
		"Name": "q1", "Database": "default", "QueryString": "SELECT 1",
	}))
	m := respJSON(t, createResp)
	id := m["NamedQueryId"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetNamedQuery", map[string]any{"NamedQueryId": id}))
	require.NoError(t, err)
	rm := respJSON(t, resp)
	nq := rm["NamedQuery"].(map[string]any)
	assert.Equal(t, "q1", nq["Name"])
	assert.Equal(t, "default", nq["Database"])
}

func TestListNamedQueries(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateNamedQuery", map[string]any{"Name": "q1", "Database": "db", "QueryString": "SELECT 1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateNamedQuery", map[string]any{"Name": "q2", "Database": "db", "QueryString": "SELECT 2"}))
	resp, err := s.HandleRequest(jsonCtx("ListNamedQueries", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	ids := m["NamedQueryIds"].([]any)
	assert.Len(t, ids, 2)
}

func TestDeleteNamedQuery(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateNamedQuery", map[string]any{
		"Name": "q1", "Database": "db", "QueryString": "SELECT 1",
	}))
	m := respJSON(t, createResp)
	id := m["NamedQueryId"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteNamedQuery", map[string]any{"NamedQueryId": id}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, err = s.HandleRequest(jsonCtx("GetNamedQuery", map[string]any{"NamedQueryId": id}))
	require.Error(t, err)
}

func TestStartQueryExecution(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "SELECT * FROM t",
		"QueryExecutionContext": map[string]any{"Database": "default"},
		"ResultConfiguration":  map[string]any{"OutputLocation": "s3://bucket/output/"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotEmpty(t, m["QueryExecutionId"])
}

func TestGetQueryExecution(t *testing.T) {
	s := newService()
	startResp, _ := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "SELECT 1",
	}))
	m := respJSON(t, startResp)
	qeID := m["QueryExecutionId"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetQueryExecution", map[string]any{"QueryExecutionId": qeID}))
	require.NoError(t, err)
	rm := respJSON(t, resp)
	qe := rm["QueryExecution"].(map[string]any)
	status := qe["Status"].(map[string]any)
	assert.Equal(t, "SUCCEEDED", status["State"])
}

func TestQueryExecutionLifecycle(t *testing.T) {
	s := newService()
	// Query immediately transitions to SUCCEEDED in mock
	startResp, _ := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{"QueryString": "SELECT 1"}))
	m := respJSON(t, startResp)
	qeID := m["QueryExecutionId"].(string)

	getResp, err := s.HandleRequest(jsonCtx("GetQueryExecution", map[string]any{"QueryExecutionId": qeID}))
	require.NoError(t, err)
	rm := respJSON(t, getResp)
	qe := rm["QueryExecution"].(map[string]any)
	status := qe["Status"].(map[string]any)
	assert.Equal(t, "SUCCEEDED", status["State"])
	assert.NotNil(t, status["CompletionDateTime"])
}

func TestGetQueryResults(t *testing.T) {
	s := newService()
	startResp, _ := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{"QueryString": "SELECT 1"}))
	m := respJSON(t, startResp)
	qeID := m["QueryExecutionId"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetQueryResults", map[string]any{"QueryExecutionId": qeID}))
	require.NoError(t, err)
	rm := respJSON(t, resp)
	assert.NotNil(t, rm["ResultSet"])
}

func TestListQueryExecutions(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{"QueryString": "SELECT 1"}))
	_, _ = s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{"QueryString": "SELECT 2"}))
	resp, err := s.HandleRequest(jsonCtx("ListQueryExecutions", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	ids := m["QueryExecutionIds"].([]any)
	assert.Len(t, ids, 2)
}

func TestStopQueryExecution(t *testing.T) {
	s := newService()
	startResp, _ := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{"QueryString": "SELECT 1"}))
	m := respJSON(t, startResp)
	qeID := m["QueryExecutionId"].(string)

	// Already SUCCEEDED, so stop won't change state but should succeed
	resp, err := s.HandleRequest(jsonCtx("StopQueryExecution", map[string]any{"QueryExecutionId": qeID}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTagResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{"Name": "tag-wg"}))
	arn := "arn:aws:athena:us-east-1:123456789012:workgroup/tag-wg"

	resp, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": arn,
		"Tags":        []map[string]string{{"Key": "env", "Value": "test"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUntagResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateWorkGroup", map[string]any{
		"Name": "untag-wg",
		"Tags": []map[string]string{{"Key": "env", "Value": "test"}},
	}))
	arn := "arn:aws:athena:us-east-1:123456789012:workgroup/untag-wg"

	resp, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceARN": arn, "TagKeys": []string{"env"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

func TestDefaultPrimaryWorkGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetWorkGroup", map[string]any{"WorkGroup": "primary"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	wg := m["WorkGroup"].(map[string]any)
	assert.Equal(t, "primary", wg["Name"])
	assert.Equal(t, "ENABLED", wg["State"])
}

// ---- Behavioral tests ----

func TestStartQueryExecution_SQLParsing(t *testing.T) {
	s := newService()
	// Valid SELECT query should succeed
	resp, err := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "SELECT id, name FROM users WHERE id = 1",
		"QueryExecutionContext": map[string]any{"Database": "default"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	qeID := m["QueryExecutionId"].(string)
	assert.NotEmpty(t, qeID)

	// Check the query succeeded
	getResp, err := s.HandleRequest(jsonCtx("GetQueryExecution", map[string]any{"QueryExecutionId": qeID}))
	require.NoError(t, err)
	gm := respJSON(t, getResp)
	qe := gm["QueryExecution"].(map[string]any)
	status := qe["Status"].(map[string]any)
	assert.Equal(t, "SUCCEEDED", status["State"])

	// Check DataScannedInBytes is tracked
	stats := qe["Statistics"].(map[string]any)
	assert.Greater(t, stats["DataScannedInBytes"].(float64), float64(0))
}

func TestStartQueryExecution_InvalidSQL(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "",
	}))
	// Empty query string is rejected at handler level
	require.Error(t, err)
	_ = resp
}

func TestStartQueryExecution_BadSQLSyntax(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "INVALID GIBBERISH QUERY",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	qeID := m["QueryExecutionId"].(string)

	getResp, err := s.HandleRequest(jsonCtx("GetQueryExecution", map[string]any{"QueryExecutionId": qeID}))
	require.NoError(t, err)
	gm := respJSON(t, getResp)
	qe := gm["QueryExecution"].(map[string]any)
	status := qe["Status"].(map[string]any)
	assert.Equal(t, "FAILED", status["State"])
	assert.Contains(t, status["StateChangeReason"], "SYNTAX_ERROR")
}

func TestStartQueryExecution_SchemaValidation(t *testing.T) {
	s := newService()
	// Register a known schema
	s.RegisterSchema("mydb", "users", []string{"id", "name", "email"})

	// Query referencing a missing table should fail
	resp, err := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "SELECT id FROM nonexistent_table",
		"QueryExecutionContext": map[string]any{"Database": "mydb"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	qeID := m["QueryExecutionId"].(string)

	getResp, err := s.HandleRequest(jsonCtx("GetQueryExecution", map[string]any{"QueryExecutionId": qeID}))
	require.NoError(t, err)
	gm := respJSON(t, getResp)
	qe := gm["QueryExecution"].(map[string]any)
	status := qe["Status"].(map[string]any)
	assert.Equal(t, "FAILED", status["State"])
	assert.Contains(t, status["StateChangeReason"], "SEMANTIC_ERROR")
	assert.Contains(t, status["StateChangeReason"], "nonexistent_table")
}

func TestStartQueryExecution_SchemaValidationColumnMissing(t *testing.T) {
	s := newService()
	s.RegisterSchema("mydb", "users", []string{"id", "name", "email"})

	// Query with a valid table but missing column
	resp, err := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "SELECT id, nonexistent_col FROM users",
		"QueryExecutionContext": map[string]any{"Database": "mydb"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	qeID := m["QueryExecutionId"].(string)

	getResp, err := s.HandleRequest(jsonCtx("GetQueryExecution", map[string]any{"QueryExecutionId": qeID}))
	require.NoError(t, err)
	gm := respJSON(t, getResp)
	qe := gm["QueryExecution"].(map[string]any)
	status := qe["Status"].(map[string]any)
	assert.Equal(t, "FAILED", status["State"])
	assert.Contains(t, status["StateChangeReason"], "nonexistent_col")
}

func TestGetQueryResults_MockData(t *testing.T) {
	s := newService()

	// Execute a SELECT query
	startResp, err := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "SELECT id, name FROM users",
	}))
	require.NoError(t, err)
	m := respJSON(t, startResp)
	qeID := m["QueryExecutionId"].(string)

	// Get results
	resp, err := s.HandleRequest(jsonCtx("GetQueryResults", map[string]any{"QueryExecutionId": qeID}))
	require.NoError(t, err)
	rm := respJSON(t, resp)
	rs := rm["ResultSet"].(map[string]any)

	// Should have rows (header + 5-10 data rows)
	rows := rs["Rows"].([]any)
	assert.GreaterOrEqual(t, len(rows), 6)  // 1 header + 5 min data rows
	assert.LessOrEqual(t, len(rows), 11)    // 1 header + 10 max data rows

	// Should have column metadata
	meta := rs["ResultSetMetadata"].(map[string]any)
	colInfo := meta["ColumnInfo"].([]any)
	assert.Len(t, colInfo, 2)

	// Verify column names
	col0 := colInfo[0].(map[string]any)
	col1 := colInfo[1].(map[string]any)
	assert.Equal(t, "id", col0["Name"])
	assert.Equal(t, "name", col1["Name"])
}

func TestGetQueryResults_FailedQuery(t *testing.T) {
	s := newService()

	// Execute a bad query that will FAIL
	startResp, err := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "INVALID SQL",
	}))
	require.NoError(t, err)
	m := respJSON(t, startResp)
	qeID := m["QueryExecutionId"].(string)

	// Getting results of a FAILED query should error
	_, err = s.HandleRequest(jsonCtx("GetQueryResults", map[string]any{"QueryExecutionId": qeID}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidRequestException", awsErr.Code)
}

func TestStartQueryExecution_DataScannedInBytes(t *testing.T) {
	s := newService()

	// Simple query
	resp1, _ := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "SELECT 1",
	}))
	m1 := respJSON(t, resp1)
	id1 := m1["QueryExecutionId"].(string)

	// More complex query with tables
	resp2, _ := s.HandleRequest(jsonCtx("StartQueryExecution", map[string]any{
		"QueryString": "SELECT a, b, c FROM table1 JOIN table2",
	}))
	m2 := respJSON(t, resp2)
	id2 := m2["QueryExecutionId"].(string)

	get1, _ := s.HandleRequest(jsonCtx("GetQueryExecution", map[string]any{"QueryExecutionId": id1}))
	get2, _ := s.HandleRequest(jsonCtx("GetQueryExecution", map[string]any{"QueryExecutionId": id2}))
	gm1 := respJSON(t, get1)
	gm2 := respJSON(t, get2)

	stats1 := gm1["QueryExecution"].(map[string]any)["Statistics"].(map[string]any)
	stats2 := gm2["QueryExecution"].(map[string]any)["Statistics"].(map[string]any)

	// More complex query should scan more data
	assert.Greater(t, stats2["DataScannedInBytes"].(float64), stats1["DataScannedInBytes"].(float64))
}
