package lakeformation_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/lakeformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.LakeFormationService {
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
	assert.Equal(t, "lakeformation", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestRegisterResource(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("RegisterResource", map[string]any{
		"ResourceArn": "arn:aws:s3:::my-bucket", "RoleArn": "arn:aws:iam::123456789012:role/lf",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRegisterResourceDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("RegisterResource", map[string]any{"ResourceArn": "arn:aws:s3:::dup"}))
	_, err := s.HandleRequest(jsonCtx("RegisterResource", map[string]any{"ResourceArn": "arn:aws:s3:::dup"}))
	require.Error(t, err)
}

func TestDeregisterResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("RegisterResource", map[string]any{"ResourceArn": "arn:aws:s3:::dereg"}))
	resp, err := s.HandleRequest(jsonCtx("DeregisterResource", map[string]any{"ResourceArn": "arn:aws:s3:::dereg"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeregisterResourceNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeregisterResource", map[string]any{"ResourceArn": "arn:aws:s3:::nope"}))
	require.Error(t, err)
}

func TestListResources(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("RegisterResource", map[string]any{"ResourceArn": "arn:aws:s3:::r1"}))
	_, _ = s.HandleRequest(jsonCtx("RegisterResource", map[string]any{"ResourceArn": "arn:aws:s3:::r2"}))
	resp, err := s.HandleRequest(jsonCtx("ListResources", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	list := m["ResourceInfoList"].([]any)
	assert.Len(t, list, 2)
}

func TestGrantPermissions(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GrantPermissions", map[string]any{
		"Principal":   map[string]any{"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/analyst"},
		"Resource":    map[string]any{"Database": map[string]any{"Name": "mydb"}},
		"Permissions": []string{"SELECT", "DESCRIBE"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListPermissions(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("GrantPermissions", map[string]any{
		"Principal":   map[string]any{"DataLakePrincipalIdentifier": "role1"},
		"Resource":    map[string]any{"Database": map[string]any{"Name": "db1"}},
		"Permissions": []string{"ALL"},
	}))
	resp, err := s.HandleRequest(jsonCtx("ListPermissions", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	perms := m["PrincipalResourcePermissions"].([]any)
	assert.Len(t, perms, 1)
}

func TestRevokePermissions(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("GrantPermissions", map[string]any{
		"Principal":   map[string]any{"DataLakePrincipalIdentifier": "role2"},
		"Resource":    map[string]any{"Database": map[string]any{"Name": "db2"}},
		"Permissions": []string{"ALL"},
	}))
	resp, err := s.HandleRequest(jsonCtx("RevokePermissions", map[string]any{
		"Principal": map[string]any{"DataLakePrincipalIdentifier": "role2"},
		"Resource":  map[string]any{"Database": map[string]any{"Name": "db2"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	listResp, _ := s.HandleRequest(jsonCtx("ListPermissions", map[string]any{
		"Principal": map[string]any{"DataLakePrincipalIdentifier": "role2"},
	}))
	m := respJSON(t, listResp)
	perms := m["PrincipalResourcePermissions"].([]any)
	assert.Len(t, perms, 0)
}

func TestGetDataLakeSettings(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetDataLakeSettings", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotNil(t, m["DataLakeSettings"])
}

func TestPutDataLakeSettings(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("PutDataLakeSettings", map[string]any{
		"DataLakeSettings": map[string]any{
			"DataLakeAdmins": []map[string]any{{"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/admin"}},
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateLFTag(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateLFTag", map[string]any{
		"TagKey": "env", "TagValues": []string{"dev", "staging", "prod"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateLFTagDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateLFTag", map[string]any{"TagKey": "dup", "TagValues": []string{"a"}}))
	_, err := s.HandleRequest(jsonCtx("CreateLFTag", map[string]any{"TagKey": "dup", "TagValues": []string{"b"}}))
	require.Error(t, err)
}

func TestGetLFTag(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateLFTag", map[string]any{"TagKey": "team", "TagValues": []string{"eng", "data"}}))
	resp, err := s.HandleRequest(jsonCtx("GetLFTag", map[string]any{"TagKey": "team"}))
	require.NoError(t, err)
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, "team", m["TagKey"])
}

func TestListLFTags(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateLFTag", map[string]any{"TagKey": "t1", "TagValues": []string{"a"}}))
	_, _ = s.HandleRequest(jsonCtx("CreateLFTag", map[string]any{"TagKey": "t2", "TagValues": []string{"b"}}))
	resp, err := s.HandleRequest(jsonCtx("ListLFTags", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tags := m["LFTags"].([]any)
	assert.Len(t, tags, 2)
}

func TestUpdateLFTag(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateLFTag", map[string]any{"TagKey": "upd", "TagValues": []string{"a", "b"}}))
	resp, err := s.HandleRequest(jsonCtx("UpdateLFTag", map[string]any{
		"TagKey": "upd", "TagValuesToAdd": []string{"c"}, "TagValuesToDelete": []string{"a"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteLFTag(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateLFTag", map[string]any{"TagKey": "del", "TagValues": []string{"x"}}))
	resp, err := s.HandleRequest(jsonCtx("DeleteLFTag", map[string]any{"TagKey": "del"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddLFTagsToResource(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("AddLFTagsToResource", map[string]any{
		"Resource": map[string]any{"Database": map[string]any{"Name": "mydb"}},
		"LFTags":   []map[string]any{{"TagKey": "env", "TagValues": []string{"prod"}}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotNil(t, m["Failures"])
}

func TestGetResourceLFTags(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("AddLFTagsToResource", map[string]any{
		"Resource": map[string]any{"Database": map[string]any{"Name": "tagged-db"}},
		"LFTags":   []map[string]any{{"TagKey": "env", "TagValues": []string{"dev"}}},
	}))
	resp, err := s.HandleRequest(jsonCtx("GetResourceLFTags", map[string]any{
		"Resource": map[string]any{"Database": map[string]any{"Name": "tagged-db"}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tags := m["LFTagOnDatabase"].([]any)
	assert.Len(t, tags, 1)
}

func TestGrantPermissionsMissingPrincipal(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GrantPermissions", map[string]any{
		"Principal":   map[string]any{},
		"Resource":    map[string]any{"Database": map[string]any{"Name": "mydb"}},
		"Permissions": []string{"SELECT"},
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
}

func TestGrantPermissionsMissingResource(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GrantPermissions", map[string]any{
		"Principal":   map[string]any{"DataLakePrincipalIdentifier": "role1"},
		"Resource":    map[string]any{},
		"Permissions": []string{"SELECT"},
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
}

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("Bogus", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}
