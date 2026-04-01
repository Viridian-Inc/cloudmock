package kinesisanalytics_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/kinesisanalytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.KinesisAnalyticsService {
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
	assert.Equal(t, "kinesisanalytics", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestCreateApplication(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{
		"ApplicationName": "app1", "RuntimeEnvironment": "FLINK-1_15", "ServiceExecutionRole": "arn:aws:iam::123456789012:role/ka",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	detail := m["ApplicationDetail"].(map[string]any)
	assert.Equal(t, "app1", detail["ApplicationName"])
	assert.NotEmpty(t, detail["ApplicationARN"])
}

func TestCreateApplicationDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "dup"}))
	_, err := s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "dup"}))
	require.Error(t, err)
}

func TestDescribeApplication(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "desc-app"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeApplication", map[string]any{"ApplicationName": "desc-app"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "desc-app", m["ApplicationDetail"].(map[string]any)["ApplicationName"])
}

func TestDescribeApplicationNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeApplication", map[string]any{"ApplicationName": "nope"}))
	require.Error(t, err)
}

func TestListApplications(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "la1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "la2"}))
	resp, err := s.HandleRequest(jsonCtx("ListApplications", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	apps := m["ApplicationSummaries"].([]any)
	assert.Len(t, apps, 2)
}

func TestDeleteApplication(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "del-app"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteApplication", map[string]any{"ApplicationName": "del-app"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_, err = s.HandleRequest(jsonCtx("DescribeApplication", map[string]any{"ApplicationName": "del-app"}))
	require.Error(t, err)
}

func TestUpdateApplication(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "upd-app"}))
	resp, err := s.HandleRequest(jsonCtx("UpdateApplication", map[string]any{
		"ApplicationName": "upd-app", "ApplicationDescription": "updated desc",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	detail := m["ApplicationDetail"].(map[string]any)
	assert.Equal(t, "updated desc", detail["ApplicationDescription"])
	assert.Equal(t, float64(2), detail["ApplicationVersionId"])
}

func TestStartApplication(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "start-app"}))
	resp, err := s.HandleRequest(jsonCtx("StartApplication", map[string]any{"ApplicationName": "start-app"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestStopApplicationNotRunning(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "stop-nr"}))
	_, err := s.HandleRequest(jsonCtx("StopApplication", map[string]any{"ApplicationName": "stop-nr"}))
	require.Error(t, err) // not in RUNNING state
}

func TestAddApplicationInput(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "inp-app"}))
	resp, err := s.HandleRequest(jsonCtx("AddApplicationInput", map[string]any{
		"ApplicationName": "inp-app",
		"Input": map[string]any{
			"NamePrefix": "SOURCE_SQL_STREAM",
			"InputSchema": map[string]any{
				"RecordFormat":  map[string]any{"RecordFormatType": "JSON"},
				"RecordColumns": []map[string]any{{"Name": "col1", "SqlType": "VARCHAR(64)"}},
			},
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddApplicationOutput(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "out-app"}))
	resp, err := s.HandleRequest(jsonCtx("AddApplicationOutput", map[string]any{
		"ApplicationName": "out-app",
		"Output": map[string]any{
			"Name":              "DESTINATION_SQL_STREAM",
			"DestinationSchema": map[string]any{"RecordFormatType": "JSON"},
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateApplicationSnapshot(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "snap-app"}))
	resp, err := s.HandleRequest(jsonCtx("CreateApplicationSnapshot", map[string]any{
		"ApplicationName": "snap-app", "SnapshotName": "snap1",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListApplicationSnapshots(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "lsnap-app"}))
	_, _ = s.HandleRequest(jsonCtx("CreateApplicationSnapshot", map[string]any{"ApplicationName": "lsnap-app", "SnapshotName": "s1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateApplicationSnapshot", map[string]any{"ApplicationName": "lsnap-app", "SnapshotName": "s2"}))
	resp, err := s.HandleRequest(jsonCtx("ListApplicationSnapshots", map[string]any{"ApplicationName": "lsnap-app"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	snaps := m["SnapshotSummaries"].([]any)
	assert.Len(t, snaps, 2)
}

func TestDeleteApplicationSnapshot(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "dsnap-app"}))
	_, _ = s.HandleRequest(jsonCtx("CreateApplicationSnapshot", map[string]any{"ApplicationName": "dsnap-app", "SnapshotName": "ds1"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteApplicationSnapshot", map[string]any{"ApplicationName": "dsnap-app", "SnapshotName": "ds1"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTagResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{"ApplicationName": "tag-app"}))
	arn := "arn:aws:kinesisanalytics:us-east-1:123456789012:application/tag-app"
	resp, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": arn, "Tags": []map[string]string{{"Key": "env", "Value": "test"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListTagsForResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateApplication", map[string]any{
		"ApplicationName": "ltag-app", "Tags": []map[string]string{{"Key": "k1", "Value": "v1"}},
	}))
	arn := "arn:aws:kinesisanalytics:us-east-1:123456789012:application/ltag-app"
	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": arn}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tags := m["Tags"].([]any)
	assert.Len(t, tags, 1)
}

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}
