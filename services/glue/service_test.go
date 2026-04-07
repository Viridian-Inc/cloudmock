package glue_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/glue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.GlueService {
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
	assert.Equal(t, "glue", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestCreateDatabase(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{
		"DatabaseInput": map[string]any{"Name": "testdb", "Description": "test"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateDatabaseDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "dup"}}))
	_, err := s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "dup"}}))
	require.Error(t, err)
}

func TestGetDatabase(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "mydb"}}))
	resp, err := s.HandleRequest(jsonCtx("GetDatabase", map[string]any{"Name": "mydb"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	db := m["Database"].(map[string]any)
	assert.Equal(t, "mydb", db["Name"])
}

func TestGetDatabaseNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetDatabase", map[string]any{"Name": "nope"}))
	require.Error(t, err)
}

func TestGetDatabases(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "db1"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "db2"}}))
	resp, err := s.HandleRequest(jsonCtx("GetDatabases", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	dbs := m["DatabaseList"].([]any)
	assert.Len(t, dbs, 2)
}

func TestDeleteDatabase(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "deldb"}}))
	resp, err := s.HandleRequest(jsonCtx("DeleteDatabase", map[string]any{"Name": "deldb"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_, err = s.HandleRequest(jsonCtx("GetDatabase", map[string]any{"Name": "deldb"}))
	require.Error(t, err)
}

func TestCreateTable(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "tbldb"}}))
	resp, err := s.HandleRequest(jsonCtx("CreateTable", map[string]any{
		"DatabaseName": "tbldb",
		"TableInput":   map[string]any{"Name": "tbl1", "StorageDescriptor": map[string]any{"Location": "s3://bucket/"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetTable(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "tbldb2"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{
		"DatabaseName": "tbldb2", "TableInput": map[string]any{"Name": "tbl1"},
	}))
	resp, err := s.HandleRequest(jsonCtx("GetTable", map[string]any{"DatabaseName": "tbldb2", "Name": "tbl1"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tbl := m["Table"].(map[string]any)
	assert.Equal(t, "tbl1", tbl["Name"])
}

func TestGetTables(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "tbldb3"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "tbldb3", "TableInput": map[string]any{"Name": "a"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "tbldb3", "TableInput": map[string]any{"Name": "b"}}))
	resp, err := s.HandleRequest(jsonCtx("GetTables", map[string]any{"DatabaseName": "tbldb3"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tbls := m["TableList"].([]any)
	assert.Len(t, tbls, 2)
}

func TestUpdateTable(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "upddb"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "upddb", "TableInput": map[string]any{"Name": "tbl"}}))
	resp, err := s.HandleRequest(jsonCtx("UpdateTable", map[string]any{
		"DatabaseName": "upddb", "TableInput": map[string]any{"Name": "tbl", "Description": "updated"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteTable(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "deldb2"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{"DatabaseName": "deldb2", "TableInput": map[string]any{"Name": "tbl"}}))
	resp, err := s.HandleRequest(jsonCtx("DeleteTable", map[string]any{"DatabaseName": "deldb2", "Name": "tbl"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateCrawler(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{
		"Name": "c1", "Role": "arn:aws:iam::123456789012:role/crawl",
		"DatabaseName": "db", "Targets": map[string]any{"S3Targets": []map[string]any{{"Path": "s3://bucket/"}}},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetCrawler(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{"Name": "gc1", "Role": "r"}))
	resp, err := s.HandleRequest(jsonCtx("GetCrawler", map[string]any{"Name": "gc1"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	c := m["Crawler"].(map[string]any)
	assert.Equal(t, "gc1", c["Name"])
	assert.Equal(t, "READY", c["State"])
}

func TestStartCrawler(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{"Name": "sc1", "Role": "r"}))
	resp, err := s.HandleRequest(jsonCtx("StartCrawler", map[string]any{"Name": "sc1"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// With default config (instant transitions), RUNNING transitions to READY immediately.
	// Just verify StartCrawler succeeded without error.
}

func TestStartCrawlerNotReady(t *testing.T) {
	s := newService()
	// Crawler that doesn't exist should fail
	_, err := s.HandleRequest(jsonCtx("StartCrawler", map[string]any{"Name": "nonexistent"}))
	require.Error(t, err)
}

func TestStopCrawlerNotRunning(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{"Name": "stc1", "Role": "r"}))
	// Crawler is in READY state, so stopping should fail
	_, err := s.HandleRequest(jsonCtx("StopCrawler", map[string]any{"Name": "stc1"}))
	require.Error(t, err)
}

func TestDeleteCrawler(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{"Name": "dc1", "Role": "r"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteCrawler", map[string]any{"Name": "dc1"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateJob(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateJob", map[string]any{
		"Name": "job1", "Role": "arn:aws:iam::123456789012:role/glue",
		"Command": map[string]any{"Name": "glueetl", "ScriptLocation": "s3://scripts/job.py"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "job1", m["Name"])
}

func TestGetJob(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateJob", map[string]any{"Name": "gj1", "Role": "r", "Command": map[string]any{"Name": "glueetl"}}))
	resp, err := s.HandleRequest(jsonCtx("GetJob", map[string]any{"JobName": "gj1"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	j := m["Job"].(map[string]any)
	assert.Equal(t, "gj1", j["Name"])
}

func TestStartJobRun(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateJob", map[string]any{"Name": "rj1", "Role": "r", "Command": map[string]any{"Name": "glueetl"}}))
	resp, err := s.HandleRequest(jsonCtx("StartJobRun", map[string]any{"JobName": "rj1"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotEmpty(t, m["JobRunId"])
}

func TestGetJobRun(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateJob", map[string]any{"Name": "gjr1", "Role": "r", "Command": map[string]any{"Name": "glueetl"}}))
	startResp, _ := s.HandleRequest(jsonCtx("StartJobRun", map[string]any{"JobName": "gjr1"}))
	m := respJSON(t, startResp)
	runID := m["JobRunId"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetJobRun", map[string]any{"JobName": "gjr1", "RunId": runID}))
	require.NoError(t, err)
	rm := respJSON(t, resp)
	jr := rm["JobRun"].(map[string]any)
	assert.Equal(t, runID, jr["Id"])
}

func TestCreateConnection(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateConnection", map[string]any{
		"ConnectionInput": map[string]any{"Name": "conn1", "ConnectionType": "JDBC"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTagging(t *testing.T) {
	s := newService()
	arn := "arn:aws:glue:us-east-1:123456789012:crawler/tc1"
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{"Name": "tc1", "Role": "r"}))

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": arn, "TagsToAdd": map[string]string{"env": "prod"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetTags", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tags := m["Tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": arn, "TagsToRemove": []string{"env"},
	}))
	require.NoError(t, err)
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

func TestStartCrawler_GeneratesTableSchema(t *testing.T) {
	s := newService()
	// Create a database first
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{
		"DatabaseInput": map[string]any{"Name": "crawl_db"},
	}))
	// Create a crawler with S3 target
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{
		"Name": "test-crawler", "Role": "arn:aws:iam::123456789012:role/crawler",
		"DatabaseName": "crawl_db",
		"Targets":      map[string]any{"S3Targets": []map[string]any{{"Path": "s3://my-bucket/data/"}}},
	}))

	// Start the crawler (no S3 service available, degrades gracefully)
	resp, err := s.HandleRequest(jsonCtx("StartCrawler", map[string]any{"Name": "test-crawler"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that tables were auto-generated in the database
	tablesResp, err := s.HandleRequest(jsonCtx("GetTables", map[string]any{"DatabaseName": "crawl_db"}))
	require.NoError(t, err)
	m := respJSON(t, tablesResp)
	tables := m["TableList"].([]any)
	assert.GreaterOrEqual(t, len(tables), 1)

	// Verify the auto-generated table has columns
	tbl := tables[0].(map[string]any)
	sd := tbl["StorageDescriptor"].(map[string]any)
	cols := sd["Columns"].([]any)
	assert.GreaterOrEqual(t, len(cols), 1)
}

func TestStartCrawler_LastCrawlMetadata(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{
		"DatabaseInput": map[string]any{"Name": "lc_db"},
	}))
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{
		"Name": "meta-crawler", "Role": "r", "DatabaseName": "lc_db",
		"Targets": map[string]any{"S3Targets": []map[string]any{{"Path": "s3://bucket/path/"}}},
	}))

	// Start the crawler
	_, err := s.HandleRequest(jsonCtx("StartCrawler", map[string]any{"Name": "meta-crawler"}))
	require.NoError(t, err)

	// Get crawler and check LastCrawl is populated
	resp, err := s.HandleRequest(jsonCtx("GetCrawler", map[string]any{"Name": "meta-crawler"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	crawler := m["Crawler"].(map[string]any)
	lastCrawl, ok := crawler["LastCrawl"].(map[string]any)
	require.True(t, ok, "LastCrawl should be present after running crawler")
	assert.Equal(t, "SUCCEEDED", lastCrawl["Status"])
	assert.NotZero(t, lastCrawl["TablesCreated"])
}

func TestStartCrawler_AutoCreatesDatabase(t *testing.T) {
	s := newService()
	// Create crawler for a database that doesn't exist yet
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{
		"Name": "autocreate-crawler", "Role": "r", "DatabaseName": "auto_db",
		"Targets": map[string]any{"S3Targets": []map[string]any{{"Path": "s3://bucket/data/"}}},
	}))

	_, err := s.HandleRequest(jsonCtx("StartCrawler", map[string]any{"Name": "autocreate-crawler"}))
	require.NoError(t, err)

	// The database should now exist
	resp, err := s.HandleRequest(jsonCtx("GetDatabase", map[string]any{"Name": "auto_db"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	db := m["Database"].(map[string]any)
	assert.Equal(t, "auto_db", db["Name"])
}

func TestUpdateDatabase(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "upd-db"}}))
	resp, err := s.HandleRequest(jsonCtx("UpdateDatabase", map[string]any{
		"Name":          "upd-db",
		"DatabaseInput": map[string]any{"Description": "updated desc"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateDatabaseNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("UpdateDatabase", map[string]any{
		"Name": "ghost-db", "DatabaseInput": map[string]any{"Description": "x"},
	}))
	require.Error(t, err)
}

func TestCreatePartition(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "part-db"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{
		"DatabaseName": "part-db", "TableInput": map[string]any{"Name": "part-tbl"},
	}))
	resp, err := s.HandleRequest(jsonCtx("CreatePartition", map[string]any{
		"DatabaseName": "part-db", "TableName": "part-tbl",
		"PartitionInput": map[string]any{"Values": []string{"2024", "01"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetPartitions(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "gpart-db"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{
		"DatabaseName": "gpart-db", "TableInput": map[string]any{"Name": "gpart-tbl"},
	}))
	_, _ = s.HandleRequest(jsonCtx("CreatePartition", map[string]any{
		"DatabaseName": "gpart-db", "TableName": "gpart-tbl",
		"PartitionInput": map[string]any{"Values": []string{"2024"}},
	}))
	_, _ = s.HandleRequest(jsonCtx("CreatePartition", map[string]any{
		"DatabaseName": "gpart-db", "TableName": "gpart-tbl",
		"PartitionInput": map[string]any{"Values": []string{"2023"}},
	}))
	resp, err := s.HandleRequest(jsonCtx("GetPartitions", map[string]any{"DatabaseName": "gpart-db", "TableName": "gpart-tbl"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	parts := m["Partitions"].([]any)
	assert.Len(t, parts, 2)
}

func TestBatchCreatePartition(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "bpart-db"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{
		"DatabaseName": "bpart-db", "TableInput": map[string]any{"Name": "bpart-tbl"},
	}))
	resp, err := s.HandleRequest(jsonCtx("BatchCreatePartition", map[string]any{
		"DatabaseName": "bpart-db", "TableName": "bpart-tbl",
		"PartitionInputList": []map[string]any{
			{"Values": []string{"2024", "01"}},
			{"Values": []string{"2024", "02"}},
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Verify partitions were created
	partResp, err := s.HandleRequest(jsonCtx("GetPartitions", map[string]any{"DatabaseName": "bpart-db", "TableName": "bpart-tbl"}))
	require.NoError(t, err)
	pm := respJSON(t, partResp)
	parts := pm["Partitions"].([]any)
	assert.Len(t, parts, 2)
}

func TestDeletePartition(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDatabase", map[string]any{"DatabaseInput": map[string]any{"Name": "dpart-db"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateTable", map[string]any{
		"DatabaseName": "dpart-db", "TableInput": map[string]any{"Name": "dpart-tbl"},
	}))
	_, _ = s.HandleRequest(jsonCtx("CreatePartition", map[string]any{
		"DatabaseName": "dpart-db", "TableName": "dpart-tbl",
		"PartitionInput": map[string]any{"Values": []string{"2024"}},
	}))
	resp, err := s.HandleRequest(jsonCtx("DeletePartition", map[string]any{
		"DatabaseName": "dpart-db", "TableName": "dpart-tbl", "PartitionValues": []string{"2024"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateCrawler(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{"Name": "upd-c", "Role": "r1"}))
	resp, err := s.HandleRequest(jsonCtx("UpdateCrawler", map[string]any{
		"Name": "upd-c", "Role": "r2", "Description": "new desc",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListCrawlers(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{"Name": "lc1", "Role": "r"}))
	_, _ = s.HandleRequest(jsonCtx("CreateCrawler", map[string]any{"Name": "lc2", "Role": "r"}))
	resp, err := s.HandleRequest(jsonCtx("ListCrawlers", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	names := m["CrawlerNames"].([]any)
	assert.Len(t, names, 2)
}

func TestUpdateJob(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateJob", map[string]any{"Name": "upd-job", "Role": "r", "Command": map[string]any{"Name": "glueetl"}}))
	resp, err := s.HandleRequest(jsonCtx("UpdateJob", map[string]any{
		"JobName": "upd-job", "JobUpdate": map[string]any{"Description": "updated", "MaxRetries": 2},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "upd-job", m["JobName"])
}

func TestGetJobs(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateJob", map[string]any{"Name": "gj-a", "Role": "r", "Command": map[string]any{"Name": "glueetl"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateJob", map[string]any{"Name": "gj-b", "Role": "r", "Command": map[string]any{"Name": "glueetl"}}))
	resp, err := s.HandleRequest(jsonCtx("GetJobs", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	jobs := m["Jobs"].([]any)
	assert.Len(t, jobs, 2)
}

func TestCreateTrigger(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateTrigger", map[string]any{
		"Name": "trig1", "Type": "SCHEDULED", "Schedule": "cron(0 * * * ? *)",
		"Actions": []map[string]any{{"JobName": "job1"}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "trig1", m["Name"])
}

func TestGetTrigger(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateTrigger", map[string]any{"Name": "gt1", "Type": "ON_DEMAND"}))
	resp, err := s.HandleRequest(jsonCtx("GetTrigger", map[string]any{"Name": "gt1"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	trig := m["Trigger"].(map[string]any)
	assert.Equal(t, "gt1", trig["Name"])
	assert.Equal(t, "CREATED", trig["State"])
}

func TestGetTriggers(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateTrigger", map[string]any{"Name": "t1", "Type": "ON_DEMAND"}))
	_, _ = s.HandleRequest(jsonCtx("CreateTrigger", map[string]any{"Name": "t2", "Type": "SCHEDULED"}))
	resp, err := s.HandleRequest(jsonCtx("GetTriggers", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	trigs := m["Triggers"].([]any)
	assert.Len(t, trigs, 2)
}

func TestUpdateTrigger(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateTrigger", map[string]any{"Name": "upd-t", "Type": "SCHEDULED"}))
	resp, err := s.HandleRequest(jsonCtx("UpdateTrigger", map[string]any{
		"Name": "upd-t", "TriggerUpdate": map[string]any{"Description": "updated"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "upd-t", m["Name"])
}

func TestDeleteTrigger(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateTrigger", map[string]any{"Name": "del-t", "Type": "ON_DEMAND"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteTrigger", map[string]any{"Name": "del-t"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_, err = s.HandleRequest(jsonCtx("GetTrigger", map[string]any{"Name": "del-t"}))
	require.Error(t, err)
}

func TestCreateClassifier(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateClassifier", map[string]any{
		"GrokClassifier": map[string]any{"Name": "cls1", "Classification": "json", "GrokPattern": "%{GREEDYDATA}"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetClassifier(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateClassifier", map[string]any{
		"GrokClassifier": map[string]any{"Name": "gcls1", "Classification": "csv"},
	}))
	resp, err := s.HandleRequest(jsonCtx("GetClassifier", map[string]any{"Name": "gcls1"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	cls := m["Classifier"].(map[string]any)
	assert.NotNil(t, cls["GrokClassifier"])
}

func TestGetClassifiers(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateClassifier", map[string]any{"GrokClassifier": map[string]any{"Name": "cls-a"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateClassifier", map[string]any{"GrokClassifier": map[string]any{"Name": "cls-b"}}))
	resp, err := s.HandleRequest(jsonCtx("GetClassifiers", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	clss := m["Classifiers"].([]any)
	assert.Len(t, clss, 2)
}

func TestDeleteClassifier(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateClassifier", map[string]any{"GrokClassifier": map[string]any{"Name": "del-cls"}}))
	resp, err := s.HandleRequest(jsonCtx("DeleteClassifier", map[string]any{"Name": "del-cls"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_, err = s.HandleRequest(jsonCtx("GetClassifier", map[string]any{"Name": "del-cls"}))
	require.Error(t, err)
}

func TestGetConnections(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateConnection", map[string]any{"ConnectionInput": map[string]any{"Name": "conn-a", "ConnectionType": "JDBC"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateConnection", map[string]any{"ConnectionInput": map[string]any{"Name": "conn-b", "ConnectionType": "KAFKA"}}))
	resp, err := s.HandleRequest(jsonCtx("GetConnections", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	conns := m["ConnectionList"].([]any)
	assert.Len(t, conns, 2)
}

func TestStartJobRun_NoLocator(t *testing.T) {
	// Job runs should work even without a locator (graceful degradation for logging)
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateJob", map[string]any{
		"Name": "log-job", "Role": "r", "Command": map[string]any{"Name": "glueetl"},
	}))

	resp, err := s.HandleRequest(jsonCtx("StartJobRun", map[string]any{"JobName": "log-job"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotEmpty(t, m["JobRunId"])
}
