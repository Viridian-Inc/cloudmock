package ssoadmin_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/ssoadmin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.SSOAdminService {
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

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func getInstanceArn(t *testing.T, s *svc.SSOAdminService) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("ListInstances", map[string]any{}))
	require.NoError(t, err)
	instances := respBody(t, resp)["Instances"].([]any)
	require.NotEmpty(t, instances)
	return instances[0].(map[string]any)["InstanceArn"].(string)
}

func createPermissionSet(t *testing.T, s *svc.SSOAdminService, instanceArn, name string) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreatePermissionSet", map[string]any{
		"InstanceArn": instanceArn,
		"Name":        name,
		"Description": "Test permission set",
	}))
	require.NoError(t, err)
	return respBody(t, resp)["PermissionSet"].(map[string]any)["PermissionSetArn"].(string)
}

func TestSSOAdmin_ListInstances(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("ListInstances", map[string]any{}))
	require.NoError(t, err)
	instances := respBody(t, resp)["Instances"].([]any)
	assert.Len(t, instances, 1)
	inst := instances[0].(map[string]any)
	assert.Equal(t, "ACTIVE", inst["Status"])
	assert.NotEmpty(t, inst["IdentityStoreId"])
}

func TestSSOAdmin_DescribeInstance(t *testing.T) {
	s := newService()
	arn := getInstanceArn(t, s)
	resp, err := s.HandleRequest(jsonCtx("DescribeInstance", map[string]any{
		"InstanceArn": arn,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, arn, m["InstanceArn"])
	assert.Equal(t, "ACTIVE", m["Status"])
}

func TestSSOAdmin_DescribeInstance_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeInstance", map[string]any{
		"InstanceArn": "arn:aws:sso:::123456789012:instance/ssoins-nonexistent",
	}))
	require.Error(t, err)
}

func TestSSOAdmin_CreatePermissionSet(t *testing.T) {
	s := newService()
	arn := getInstanceArn(t, s)
	psArn := createPermissionSet(t, s, arn, "AdminAccess")
	assert.Contains(t, psArn, "/ps-")
}

func TestSSOAdmin_CreatePermissionSet_Duplicate(t *testing.T) {
	s := newService()
	arn := getInstanceArn(t, s)
	createPermissionSet(t, s, arn, "DupPS")
	_, err := s.HandleRequest(jsonCtx("CreatePermissionSet", map[string]any{
		"InstanceArn": arn, "Name": "DupPS",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ConflictException")
}

func TestSSOAdmin_DescribePermissionSet(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	psArn := createPermissionSet(t, s, instArn, "DescribePS")
	resp, err := s.HandleRequest(jsonCtx("DescribePermissionSet", map[string]any{
		"InstanceArn":      instArn,
		"PermissionSetArn": psArn,
	}))
	require.NoError(t, err)
	ps := respBody(t, resp)["PermissionSet"].(map[string]any)
	assert.Equal(t, "DescribePS", ps["Name"])
	assert.Equal(t, "PT1H", ps["SessionDuration"])
}

func TestSSOAdmin_ListPermissionSets(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	createPermissionSet(t, s, instArn, "PS1")
	createPermissionSet(t, s, instArn, "PS2")
	resp, _ := s.HandleRequest(jsonCtx("ListPermissionSets", map[string]any{
		"InstanceArn": instArn,
	}))
	arns := respBody(t, resp)["PermissionSets"].([]any)
	assert.Len(t, arns, 2)
}

func TestSSOAdmin_UpdatePermissionSet(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	psArn := createPermissionSet(t, s, instArn, "UpdatePS")
	_, err := s.HandleRequest(jsonCtx("UpdatePermissionSet", map[string]any{
		"InstanceArn":      instArn,
		"PermissionSetArn": psArn,
		"Description":      "Updated description",
		"SessionDuration":  "PT2H",
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("DescribePermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
	}))
	ps := respBody(t, resp)["PermissionSet"].(map[string]any)
	assert.Equal(t, "Updated description", ps["Description"])
	assert.Equal(t, "PT2H", ps["SessionDuration"])
}

func TestSSOAdmin_DeletePermissionSet(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	psArn := createPermissionSet(t, s, instArn, "DeletePS")
	_, err := s.HandleRequest(jsonCtx("DeletePermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DescribePermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
	}))
	require.Error(t, err)
}

func TestSSOAdmin_AccountAssignment(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	psArn := createPermissionSet(t, s, instArn, "AssignPS")

	resp, err := s.HandleRequest(jsonCtx("CreateAccountAssignment", map[string]any{
		"InstanceArn":      instArn,
		"PermissionSetArn": psArn,
		"TargetId":         "111111111111",
		"TargetType":       "AWS_ACCOUNT",
		"PrincipalId":      "user-123",
		"PrincipalType":    "USER",
	}))
	require.NoError(t, err)
	status := respBody(t, resp)["AccountAssignmentCreationStatus"].(map[string]any)
	assert.Equal(t, "SUCCEEDED", status["Status"])

	// List
	resp, _ = s.HandleRequest(jsonCtx("ListAccountAssignments", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn, "AccountId": "111111111111",
	}))
	assignments := respBody(t, resp)["AccountAssignments"].([]any)
	assert.Len(t, assignments, 1)

	// Delete
	resp, err = s.HandleRequest(jsonCtx("DeleteAccountAssignment", map[string]any{
		"InstanceArn":      instArn,
		"PermissionSetArn": psArn,
		"TargetId":         "111111111111",
		"TargetType":       "AWS_ACCOUNT",
		"PrincipalId":      "user-123",
		"PrincipalType":    "USER",
	}))
	require.NoError(t, err)
}

func TestSSOAdmin_ManagedPolicy(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	psArn := createPermissionSet(t, s, instArn, "ManagedPS")

	_, err := s.HandleRequest(jsonCtx("AttachManagedPolicyToPermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
		"ManagedPolicyArn": "arn:aws:iam::aws:policy/ReadOnlyAccess",
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("ListManagedPoliciesInPermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
	}))
	policies := respBody(t, resp)["AttachedManagedPolicies"].([]any)
	assert.Len(t, policies, 1)

	// Detach
	_, err = s.HandleRequest(jsonCtx("DetachManagedPolicyFromPermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
		"ManagedPolicyArn": "arn:aws:iam::aws:policy/ReadOnlyAccess",
	}))
	require.NoError(t, err)
}

func TestSSOAdmin_InlinePolicy(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	psArn := createPermissionSet(t, s, instArn, "InlinePS")

	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`
	_, err := s.HandleRequest(jsonCtx("PutInlinePolicyToPermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
		"InlinePolicy": policy,
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("GetInlinePolicyForPermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
	}))
	m := respBody(t, resp)
	assert.Equal(t, policy, m["InlinePolicy"])

	_, err = s.HandleRequest(jsonCtx("DeleteInlinePolicyFromPermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("GetInlinePolicyForPermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": psArn,
	}))
	m = respBody(t, resp)
	assert.Empty(t, m["InlinePolicy"])
}

func TestSSOAdmin_Tagging(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	psArn := createPermissionSet(t, s, instArn, "TagPS")

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": psArn,
		"Tags":        []any{map[string]any{"Key": "env", "Value": "prod"}},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{
		"ResourceArn": psArn,
	}))
	tags := respBody(t, resp)["Tags"].([]any)
	assert.Len(t, tags, 1)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": psArn,
		"TagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": psArn}))
	tags = respBody(t, resp)["Tags"].([]any)
	assert.Len(t, tags, 0)
}

func TestSSOAdmin_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

func TestSSOAdmin_InvalidSessionDuration(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	_, err := s.HandleRequest(jsonCtx("CreatePermissionSet", map[string]any{
		"InstanceArn":     instArn,
		"Name":            "BadDurationPS",
		"SessionDuration": "not-iso-8601",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationException")
}

func TestSSOAdmin_InvalidAccountID(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	psArn := createPermissionSet(t, s, instArn, "AccountIDPS")
	_, err := s.HandleRequest(jsonCtx("CreateAccountAssignment", map[string]any{
		"InstanceArn":      instArn,
		"PermissionSetArn": psArn,
		"TargetId":         "not-12-digits",
		"TargetType":       "AWS_ACCOUNT",
		"PrincipalId":      "user-1",
		"PrincipalType":    "USER",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationException")
}

func TestSSOAdmin_PermissionSet_NotFound(t *testing.T) {
	s := newService()
	instArn := getInstanceArn(t, s)
	_, err := s.HandleRequest(jsonCtx("DescribePermissionSet", map[string]any{
		"InstanceArn": instArn, "PermissionSetArn": "arn:nonexistent",
	}))
	require.Error(t, err)
}
