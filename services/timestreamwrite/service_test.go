package timestreamwrite_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/timestreamwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.TimestreamWriteService {
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
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestServiceName(t *testing.T) {
	assert.Equal(t, "timestream-write", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestCreateDatabase(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "testdb"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	db := m["Database"].(map[string]any)
	assert.Equal(t, "testdb", db["DatabaseName"])
	assert.NotEmpty(t, db["Arn"])
}

func TestCreateDatabaseDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "dup"}))
	_, err := s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "dup"}))
	require.Error(t, err)
}

func TestDescribeDatabase(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "descdb"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeDatabase", map[string]any{"DatabaseName": "descdb"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "descdb", m["Database"].(map[string]any)["DatabaseName"])
}

func TestDescribeDatabaseNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeDatabase", map[string]any{"DatabaseName": "nope"}))
	require.Error(t, err)
}

func TestListDatabases(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "db1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "db2"}))
	resp, err := s.HandleRequest(jsonCtx("ListDatabases", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	dbs := m["Databases"].([]any)
	assert.Len(t, dbs, 2)
}

func TestUpdateDatabase(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "upddb"}))
	resp, err := s.HandleRequest(jsonCtx("UpdateDatabase", map[string]any{
		"DatabaseName": "upddb", "KmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/new-key",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "arn:aws:kms:us-east-1:123456789012:key/new-key", m["Database"].(map[string]any)["KmsKeyId"])
}

func TestDeleteDatabase(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "deldb"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteDatabase", map[string]any{"DatabaseName": "deldb"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_, err = s.HandleRequest(jsonCtx("DescribeDatabase", map[string]any{"DatabaseName": "deldb"}))
	require.Error(t, err)
}

func TestCreateTable(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "tbldb"}))
	resp, err := s.HandleRequest(jsonCtx("CreateTable", map[string]any{
		"DatabaseName": "tbldb", "TableName": "metrics",
		"RetentionProperties": map[string]any{"MemoryStoreRetentionPeriodInHours": 24, "MagneticStoreRetentionPeriodInDays": 365},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tbl := m["Table"].(map[string]any)
	assert.Equal(t, "metrics", tbl["TableName"])
}

func TestDescribeTable(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "dtbldb"}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "dtbldb", "TableName": "t1"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeTable", map[string]any{"DatabaseName": "dtbldb", "TableName": "t1"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "t1", m["Table"].(map[string]any)["TableName"])
}

func TestListTables(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "ltbldb"}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "ltbldb", "TableName": "t1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "ltbldb", "TableName": "t2"}))
	resp, err := s.HandleRequest(jsonCtx("ListTables", map[string]any{"DatabaseName": "ltbldb"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tbls := m["Tables"].([]any)
	assert.Len(t, tbls, 2)
}

func TestUpdateTable(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "utbldb"}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "utbldb", "TableName": "ut1"}))
	resp, err := s.HandleRequest(jsonCtx("UpdateTable", map[string]any{
		"DatabaseName": "utbldb", "TableName": "ut1",
		"RetentionProperties": map[string]any{"MemoryStoreRetentionPeriodInHours": 48, "MagneticStoreRetentionPeriodInDays": 730},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteTable(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "dtdb"}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "dtdb", "TableName": "dt1"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteTable", map[string]any{"DatabaseName": "dtdb", "TableName": "dt1"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestWriteRecords(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "wrdb"}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "wrdb", "TableName": "wrt1"}))
	resp, err := s.HandleRequest(jsonCtx("WriteRecords", map[string]any{
		"DatabaseName": "wrdb", "TableName": "wrt1",
		"Records": []map[string]any{{
			"Dimensions":       []map[string]string{{"Name": "host", "Value": "web-01"}},
			"MeasureName":      "cpu_utilization",
			"MeasureValue":     "85.5",
			"MeasureValueType": "DOUBLE",
			"Time":             "1609459200000",
			"TimeUnit":         "MILLISECONDS",
		}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	ingested := m["RecordsIngested"].(map[string]any)
	assert.Equal(t, float64(1), ingested["Total"])
}

func TestDescribeEndpoints(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("DescribeEndpoints", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	eps := m["Endpoints"].([]any)
	assert.Len(t, eps, 2) // ingest + query endpoints
}

func TestTagResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "tagdb"}))
	arn := "arn:aws:timestream:us-east-1:123456789012:database/tagdb"
	resp, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": arn, "Tags": []map[string]string{{"Key": "env", "Value": "prod"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListTagsForResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{
		"DatabaseName": "ltagdb", "Tags": []map[string]string{{"Key": "k", "Value": "v"}},
	}))
	arn := "arn:aws:timestream:us-east-1:123456789012:database/ltagdb"
	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": arn}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tags := m["Tags"].([]any)
	assert.Len(t, tags, 1)
}

func TestUntagResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{
		"DatabaseName": "untagdb", "Tags": []map[string]string{{"Key": "rm", "Value": "me"}},
	}))
	arn := "arn:aws:timestream:us-east-1:123456789012:database/untagdb"
	resp, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceARN": arn, "TagKeys": []string{"rm"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Behavioral tests ----

func TestQuery_BasicTimeRange(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "qdb"}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{
		"DatabaseName": "qdb", "TableName": "metrics",
	}))

	// Write records at different times
	_, _ = s.HandleRequest(jsonCtx("WriteRecords", map[string]any{
		"DatabaseName": "qdb", "TableName": "metrics",
		"Records": []map[string]any{
			{
				"Dimensions":       []map[string]string{{"Name": "host", "Value": "web-01"}},
				"MeasureName":      "cpu",
				"MeasureValue":     "85.5",
				"MeasureValueType": "DOUBLE",
				"Time":             "1609459200000",
				"TimeUnit":         "MILLISECONDS",
			},
			{
				"Dimensions":       []map[string]string{{"Name": "host", "Value": "web-02"}},
				"MeasureName":      "cpu",
				"MeasureValue":     "42.0",
				"MeasureValueType": "DOUBLE",
				"Time":             "1609459300000",
				"TimeUnit":         "MILLISECONDS",
			},
			{
				"Dimensions":       []map[string]string{{"Name": "host", "Value": "web-03"}},
				"MeasureName":      "cpu",
				"MeasureValue":     "99.9",
				"MeasureValueType": "DOUBLE",
				"Time":             "1609459400000",
				"TimeUnit":         "MILLISECONDS",
			},
		},
	}))

	// Query all records
	resp, err := s.HandleRequest(jsonCtx("Query", map[string]any{
		"QueryString": `SELECT * FROM "qdb"."metrics"`,
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	rows := m["Rows"].([]any)
	assert.Len(t, rows, 3)
	assert.NotEmpty(t, m["QueryId"])
	assert.NotNil(t, m["ColumnInfo"])

	// Query with time range filter
	resp, err = s.HandleRequest(jsonCtx("Query", map[string]any{
		"QueryString": `SELECT * FROM "qdb"."metrics" WHERE time BETWEEN '1609459200000' AND '1609459300000'`,
	}))
	require.NoError(t, err)
	m = respJSON(t, resp)
	rows = m["Rows"].([]any)
	assert.Len(t, rows, 2) // Only first two records in range
}

func TestQuery_TableNotFound(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "nodb"}))
	_, err := s.HandleRequest(jsonCtx("Query", map[string]any{
		"QueryString": `SELECT * FROM "nodb"."nonexistent"`,
	}))
	require.Error(t, err)
}

func TestDescribeEndpoints_IngestAndQuery(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("DescribeEndpoints", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	eps := m["Endpoints"].([]any)
	assert.Len(t, eps, 2)
	ep0 := eps[0].(map[string]any)
	ep1 := eps[1].(map[string]any)
	assert.Contains(t, ep0["Address"], "ingest.timestream")
	assert.Contains(t, ep1["Address"], "query.timestream")
}

func TestRetentionProperties_Tracked(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseName": "retdb"}))
	resp, err := s.HandleRequest(jsonCtx("CreateTable", map[string]any{
		"DatabaseName": "retdb", "TableName": "ret_tbl",
		"RetentionProperties": map[string]any{
			"MemoryStoreRetentionPeriodInHours":  48,
			"MagneticStoreRetentionPeriodInDays": 365,
		},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tbl := m["Table"].(map[string]any)
	ret := tbl["RetentionProperties"].(map[string]any)
	assert.Equal(t, float64(48), ret["MemoryStoreRetentionPeriodInHours"])
	assert.Equal(t, float64(365), ret["MagneticStoreRetentionPeriodInDays"])

	// Update retention
	resp, err = s.HandleRequest(jsonCtx("UpdateTable", map[string]any{
		"DatabaseName": "retdb", "TableName": "ret_tbl",
		"RetentionProperties": map[string]any{
			"MemoryStoreRetentionPeriodInHours":  72,
			"MagneticStoreRetentionPeriodInDays": 730,
		},
	}))
	require.NoError(t, err)
	m = respJSON(t, resp)
	tbl = m["Table"].(map[string]any)
	ret = tbl["RetentionProperties"].(map[string]any)
	assert.Equal(t, float64(72), ret["MemoryStoreRetentionPeriodInHours"])
	assert.Equal(t, float64(730), ret["MagneticStoreRetentionPeriodInDays"])
}
