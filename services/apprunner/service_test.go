package apprunner_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/apprunner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.AppRunnerService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	b, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       b,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func decode(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// ---- Service tests ----

func TestAppRunner_CreateService(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"ServiceName": "my-app",
		"SourceConfiguration": map[string]any{
			"ImageRepository": map[string]any{
				"ImageIdentifier":     "public.ecr.aws/nginx/nginx:latest",
				"ImageRepositoryType": "ECR_PUBLIC",
			},
		},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	svcMap := m["Service"].(map[string]any)
	assert.Equal(t, "my-app", svcMap["ServiceName"])
	assert.Equal(t, "RUNNING", svcMap["Status"])
	assert.NotEmpty(t, svcMap["ServiceArn"])
}

func TestAppRunner_CreateService_Duplicate(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateService", map[string]any{"ServiceName": "dup-app"}))
	_, err := s.HandleRequest(jsonCtx("CreateService", map[string]any{"ServiceName": "dup-app"}))
	require.Error(t, err)
}

func TestAppRunner_DescribeService(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{"ServiceName": "desc-app"}))
	arn := decode(t, createResp)["Service"].(map[string]any)["ServiceArn"].(string)

	resp, err := s.HandleRequest(jsonCtx("DescribeService", map[string]any{"ServiceArn": arn}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "desc-app", m["Service"].(map[string]any)["ServiceName"])
}

func TestAppRunner_DescribeService_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeService", map[string]any{"ServiceArn": "arn:aws:apprunner:us-east-1:123:service/nope/nope"}))
	require.Error(t, err)
}

func TestAppRunner_ListServices(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateService", map[string]any{"ServiceName": "app-1"}))
	s.HandleRequest(jsonCtx("CreateService", map[string]any{"ServiceName": "app-2"}))

	resp, err := s.HandleRequest(jsonCtx("ListServices", nil))
	require.NoError(t, err)
	m := decode(t, resp)
	items := m["ServiceSummaryList"].([]any)
	assert.Len(t, items, 2)
}

func TestAppRunner_UpdateService(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{"ServiceName": "update-app"}))
	arn := decode(t, cr)["Service"].(map[string]any)["ServiceArn"].(string)

	resp, err := s.HandleRequest(jsonCtx("UpdateService", map[string]any{"ServiceArn": arn}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["OperationId"])
}

func TestAppRunner_DeleteService(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{"ServiceName": "del-app"}))
	arn := decode(t, cr)["Service"].(map[string]any)["ServiceArn"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteService", map[string]any{"ServiceArn": arn}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, err2 := s.HandleRequest(jsonCtx("DescribeService", map[string]any{"ServiceArn": arn}))
	require.Error(t, err2)
}

func TestAppRunner_PauseAndResumeService(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{"ServiceName": "pause-app"}))
	arn := decode(t, cr)["Service"].(map[string]any)["ServiceArn"].(string)

	pauseResp, err := s.HandleRequest(jsonCtx("PauseService", map[string]any{"ServiceArn": arn}))
	require.NoError(t, err)
	assert.Equal(t, "PAUSED", decode(t, pauseResp)["Service"].(map[string]any)["Status"])

	resumeResp, err := s.HandleRequest(jsonCtx("ResumeService", map[string]any{"ServiceArn": arn}))
	require.NoError(t, err)
	assert.Equal(t, "RUNNING", decode(t, resumeResp)["Service"].(map[string]any)["Status"])
}

// ---- Connection tests ----

func TestAppRunner_CreateAndDescribeConnection(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateConnection", map[string]any{
		"ConnectionName": "my-github",
		"ProviderType":   "GITHUB",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	conn := m["Connection"].(map[string]any)
	assert.Equal(t, "AVAILABLE", conn["Status"])
	arn := conn["ConnectionArn"].(string)

	descResp, err := s.HandleRequest(jsonCtx("DescribeConnection", map[string]any{"ConnectionArn": arn}))
	require.NoError(t, err)
	descConn := decode(t, descResp)["Connection"].(map[string]any)
	assert.Equal(t, "my-github", descConn["ConnectionName"])
}

// ---- AutoScalingConfiguration tests ----

func TestAppRunner_AutoScalingConfiguration_CRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateAutoScalingConfiguration", map[string]any{
		"AutoScalingConfigurationName": "my-asc",
		"MinSize":                      1,
		"MaxSize":                      10,
		"MaxConcurrency":               100,
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	asc := m["AutoScalingConfiguration"].(map[string]any)
	assert.Equal(t, "my-asc", asc["AutoScalingConfigurationName"])
	assert.Equal(t, float64(1), asc["AutoScalingConfigurationRevision"])
	arn := asc["AutoScalingConfigurationArn"].(string)

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeAutoScalingConfiguration", map[string]any{
		"AutoScalingConfigurationArn": arn,
	}))
	require.NoError(t, err)
	descASC := decode(t, descResp)["AutoScalingConfiguration"].(map[string]any)
	assert.Equal(t, "my-asc", descASC["AutoScalingConfigurationName"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListAutoScalingConfigurations", nil))
	require.NoError(t, err)
	items := decode(t, listResp)["AutoScalingConfigurationSummaryList"].([]any)
	assert.Len(t, items, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteAutoScalingConfiguration", map[string]any{
		"AutoScalingConfigurationArn": arn,
	}))
	require.NoError(t, err)
	_, err = s.HandleRequest(jsonCtx("DescribeAutoScalingConfiguration", map[string]any{
		"AutoScalingConfigurationArn": arn,
	}))
	require.Error(t, err)
}

func TestAppRunner_AutoScalingConfiguration_Revisions(t *testing.T) {
	s := newService()
	r1, _ := s.HandleRequest(jsonCtx("CreateAutoScalingConfiguration", map[string]any{
		"AutoScalingConfigurationName": "rev-asc", "MinSize": 1, "MaxSize": 5,
	}))
	r2, _ := s.HandleRequest(jsonCtx("CreateAutoScalingConfiguration", map[string]any{
		"AutoScalingConfigurationName": "rev-asc", "MinSize": 2, "MaxSize": 10,
	}))
	rev1 := decode(t, r1)["AutoScalingConfiguration"].(map[string]any)["AutoScalingConfigurationRevision"].(float64)
	rev2 := decode(t, r2)["AutoScalingConfiguration"].(map[string]any)["AutoScalingConfigurationRevision"].(float64)
	assert.Equal(t, float64(1), rev1)
	assert.Equal(t, float64(2), rev2)
}

// ---- VpcConnector tests ----

func TestAppRunner_VpcConnector_CRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateVpcConnector", map[string]any{
		"VpcConnectorName": "my-vc",
		"Subnets":          []string{"subnet-1", "subnet-2"},
		"SecurityGroups":   []string{"sg-1"},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	vc := m["VpcConnector"].(map[string]any)
	assert.Equal(t, "my-vc", vc["VpcConnectorName"])
	arn := vc["VpcConnectorArn"].(string)

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeVpcConnector", map[string]any{"VpcConnectorArn": arn}))
	require.NoError(t, err)
	descVC := decode(t, descResp)["VpcConnector"].(map[string]any)
	assert.Equal(t, "ACTIVE", descVC["Status"])

	// List
	listResp, _ := s.HandleRequest(jsonCtx("ListVpcConnectors", nil))
	items := decode(t, listResp)["VpcConnectors"].([]any)
	assert.Len(t, items, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteVpcConnector", map[string]any{"VpcConnectorArn": arn}))
	require.NoError(t, err)
	_, err = s.HandleRequest(jsonCtx("DescribeVpcConnector", map[string]any{"VpcConnectorArn": arn}))
	require.Error(t, err)
}

// ---- Tag tests ----

func TestAppRunner_Tags(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateService", map[string]any{
		"ServiceName": "tagged-app",
		"Tags":        map[string]string{"env": "test"},
	}))
	arn := decode(t, cr)["Service"].(map[string]any)["ServiceArn"].(string)

	// Tag resource
	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags":        map[string]string{"owner": "team-a"},
	}))
	require.NoError(t, err)

	// List tags
	listResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	tags := decode(t, listResp)["Tags"].(map[string]any)
	assert.Equal(t, "team-a", tags["owner"])

	// Untag
	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []string{"owner"},
	}))
	require.NoError(t, err)
	listResp2, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	tags2 := decode(t, listResp2)["Tags"].(map[string]any)
	_, hasOwner := tags2["owner"]
	assert.False(t, hasOwner)
}

func TestAppRunner_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("NonExistentAction", nil))
	require.Error(t, err)
}

func TestAppRunner_ServiceURL(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateService", map[string]any{"ServiceName": "url-test"}))
	require.NoError(t, err)
	svcMap := decode(t, resp)["Service"].(map[string]any)
	url := svcMap["ServiceUrl"].(string)
	assert.Contains(t, url, "awsapprunner.com")
}
