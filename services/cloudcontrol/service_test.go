package cloudcontrol_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/cloudcontrol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.CloudControlService { return svc.New("123456789012", "us-east-1") }

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action: action, Region: "us-east-1", AccountID: "123456789012", Body: bodyBytes,
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

func TestCloudControl_CreateAndGetResource(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "my-bucket", "DesiredState": `{"BucketName":"my-bucket"}`,
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	pe := m["ProgressEvent"].(map[string]any)
	assert.Equal(t, "PENDING", pe["OperationStatus"])
	assert.NotEmpty(t, pe["RequestToken"])

	getResp, err := s.HandleRequest(jsonCtx("GetResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "my-bucket",
	}))
	require.NoError(t, err)
	gm := respJSON(t, getResp)
	assert.Equal(t, "my-bucket", gm["ResourceDescription"].(map[string]any)["Identifier"])
}

func TestCloudControl_ListResources(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::EC2::Instance", "Identifier": "i-1", "DesiredState": "{}",
	}))
	s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::EC2::Instance", "Identifier": "i-2", "DesiredState": "{}",
	}))

	resp, _ := s.HandleRequest(jsonCtx("ListResources", map[string]any{"TypeName": "AWS::EC2::Instance"}))
	m := respJSON(t, resp)
	assert.Len(t, m["ResourceDescriptions"].([]any), 2)
}

func TestCloudControl_UpdateResource(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "upd-bucket", "DesiredState": "{}",
	}))

	resp, err := s.HandleRequest(jsonCtx("UpdateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "upd-bucket", "PatchDocument": `[{"op":"add"}]`,
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "UPDATE", m["ProgressEvent"].(map[string]any)["OperationType"])
}

func TestCloudControl_DeleteResource(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "del-bucket", "DesiredState": "{}",
	}))

	resp, err := s.HandleRequest(jsonCtx("DeleteResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "del-bucket",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "DELETE", m["ProgressEvent"].(map[string]any)["OperationType"])

	_, err = s.HandleRequest(jsonCtx("GetResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "del-bucket",
	}))
	require.Error(t, err)
}

func TestCloudControl_RequestStatus(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "status-bucket", "DesiredState": "{}",
	}))
	token := respJSON(t, createResp)["ProgressEvent"].(map[string]any)["RequestToken"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetResourceRequestStatus", map[string]any{"RequestToken": token}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotEmpty(t, m["ProgressEvent"].(map[string]any)["OperationStatus"])
}

func TestCloudControl_ListResourceRequests(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "req-1", "DesiredState": "{}",
	}))
	s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "req-2", "DesiredState": "{}",
	}))

	resp, _ := s.HandleRequest(jsonCtx("ListResourceRequests", nil))
	m := respJSON(t, resp)
	assert.GreaterOrEqual(t, len(m["ResourceRequestStatusSummaries"].([]any)), 2)
}

func TestCloudControl_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "nonexistent",
	}))
	require.Error(t, err)
}

func TestCloudControl_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("Bogus", nil))
	require.Error(t, err)
}

func TestCloudControl_DuplicateResource(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "dup", "DesiredState": "{}",
	}))
	_, err := s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "dup", "DesiredState": "{}",
	}))
	require.Error(t, err)
}

func TestCloudControl_ResourceTypeMapping(t *testing.T) {
	// Verify resource type to service mapping exists for known types
	assert.Equal(t, "s3", svc.ResourceTypeToService["AWS::S3::Bucket"])
	assert.Equal(t, "dynamodb", svc.ResourceTypeToService["AWS::DynamoDB::Table"])
	assert.Equal(t, "sqs", svc.ResourceTypeToService["AWS::SQS::Queue"])
	assert.Equal(t, "sns", svc.ResourceTypeToService["AWS::SNS::Topic"])
}

func TestCloudControl_CreateWithoutLocator(t *testing.T) {
	// Without a locator, create should still work with generic fallback.
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "Identifier": "no-locator-bucket", "DesiredState": `{"BucketName":"no-locator-bucket"}`,
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	pe := m["ProgressEvent"].(map[string]any)
	assert.Equal(t, "PENDING", pe["OperationStatus"])
}

func TestCloudControl_CreateUnknownTypeWithLocator(t *testing.T) {
	// Unknown resource type should fall back to generic behavior even with locator.
	s := newService()
	s.SetLocator(&noopLocator{})
	resp, err := s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::Custom::Widget", "Identifier": "w-1", "DesiredState": "{}",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "PENDING", m["ProgressEvent"].(map[string]any)["OperationStatus"])
}

func TestCloudControl_MissingTypeName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"Identifier": "x", "DesiredState": "{}",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TypeName")
}

func TestCloudControl_MissingIdentifier(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateResource", map[string]any{
		"TypeName": "AWS::S3::Bucket", "DesiredState": "{}",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Identifier")
}

// noopLocator is a test locator that returns errors for all lookups.
type noopLocator struct{}

func (n *noopLocator) Lookup(name string) (service.Service, error) {
	return nil, fmt.Errorf("service %q not available", name)
}

func TestCloudControl_ListResourcesMissingTypeName(t *testing.T) {
	s := newService()
	s.SetLocator(&noopLocator{})
	_, err := s.HandleRequest(jsonCtx("ListResources", map[string]any{}))
	require.Error(t, err)
}

func TestCloudControl_InvalidAction2(t *testing.T) {
	s := newService()
	s.SetLocator(&noopLocator{})
	_, err := s.HandleRequest(jsonCtx("NonExistentAction2", map[string]any{}))
	require.Error(t, err)
}
