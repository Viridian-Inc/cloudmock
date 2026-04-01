package airflow_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/airflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.AirflowService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		Params:     make(map[string]string),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

// ---- Test 1: CreateEnvironment ----

func TestCreateEnvironment(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "test-env",
		"SourceBucketArn":  "arn:aws:s3:::my-bucket",
		"DagS3Path":        "dags/",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/AirflowRole",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := respBody(t, resp)
	assert.Contains(t, body["Arn"].(string), "test-env")
}

// ---- Test 2: GetEnvironment ----

func TestGetEnvironment(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "get-env",
		"SourceBucketArn":  "arn:aws:s3:::my-bucket",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetEnvironment", map[string]any{"Name": "get-env"}))
	require.NoError(t, err)
	body := respBody(t, resp)
	env := body["Environment"].(map[string]any)
	assert.Equal(t, "get-env", env["Name"])
	assert.NotEmpty(t, env["Status"])
}

// ---- Test 3: ListEnvironments ----

func TestListEnvironments(t *testing.T) {
	s := newService()
	for _, name := range []string{"env-1", "env-2", "env-3"} {
		_, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
			"Name":             name,
			"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
		}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListEnvironments", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	envs := body["Environments"].([]any)
	assert.Len(t, envs, 3)
}

// ---- Test 4: DeleteEnvironment ----

func TestDeleteEnvironment(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "del-env",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteEnvironment", map[string]any{"Name": "del-env"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetEnvironment", map[string]any{"Name": "del-env"}))
	require.Error(t, err)
}

// ---- Test 5: UpdateEnvironment ----

func TestUpdateEnvironment(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "upd-env",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
		"MaxWorkers":       float64(10),
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("UpdateEnvironment", map[string]any{
		"Name":       "upd-env",
		"MaxWorkers": float64(20),
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Contains(t, body["Arn"].(string), "upd-env")

	getResp, err := s.HandleRequest(jsonCtx("GetEnvironment", map[string]any{"Name": "upd-env"}))
	require.NoError(t, err)
	env := respBody(t, getResp)["Environment"].(map[string]any)
	assert.Equal(t, float64(20), env["MaxWorkers"])
}

// ---- Test 6: Environment lifecycle CREATING -> AVAILABLE ----

func TestEnvironmentLifecycle(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "lc-env",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetEnvironment", map[string]any{"Name": "lc-env"}))
	require.NoError(t, err)
	env := respBody(t, resp)["Environment"].(map[string]any)
	assert.Contains(t, []string{"CREATING", "AVAILABLE"}, env["Status"])

	time.Sleep(3 * time.Second)
	resp2, err := s.HandleRequest(jsonCtx("GetEnvironment", map[string]any{"Name": "lc-env"}))
	require.NoError(t, err)
	env2 := respBody(t, resp2)["Environment"].(map[string]any)
	assert.Equal(t, "AVAILABLE", env2["Status"])
}

// ---- Test 7: CreateCliToken ----

func TestCreateCliToken(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "cli-env",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("CreateCliToken", map[string]any{"Name": "cli-env"}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["CliToken"])
	assert.NotEmpty(t, body["WebServerHostname"])
}

// ---- Test 8: CreateWebLoginToken ----

func TestCreateWebLoginToken(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "web-env",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("CreateWebLoginToken", map[string]any{"Name": "web-env"}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["WebToken"])
	assert.NotEmpty(t, body["WebServerHostname"])
}

// ---- Test 9: NotFound ----

func TestEnvironmentNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetEnvironment", map[string]any{"Name": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

// ---- Test 10: InvalidAction ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 11: Tagging ----

func TestTagging(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "tag-env",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
		"Tags":             map[string]any{"env": "test"},
	}))
	require.NoError(t, err)

	arn := "arn:aws:airflow:us-east-1:123456789012:environment/tag-env"

	_, err = s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags":        map[string]any{"team": "data"},
	}))
	require.NoError(t, err)

	listResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	tags := respBody(t, listResp)["Tags"].(map[string]any)
	assert.Len(t, tags, 2)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": arn,
		"tagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	listResp2, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	tags2 := respBody(t, listResp2)["Tags"].(map[string]any)
	assert.Len(t, tags2, 1)
}

// ---- Test 12: Duplicate environment ----

func TestDuplicateEnvironment(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "dup-env",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("CreateEnvironment", map[string]any{
		"Name":             "dup-env",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/Role",
	}))
	require.Error(t, err)
}

// ---- Test 13: Service Name and HealthCheck ----

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "airflow", s.Name())
	assert.NoError(t, s.HealthCheck())
}
