package lakeformation_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/lakeformation"
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

func TestBatchGrantPermissions(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("BatchGrantPermissions", map[string]any{
		"Entries": []map[string]any{
			{
				"Id":          "entry-1",
				"Principal":   map[string]any{"DataLakePrincipalIdentifier": "role1"},
				"Resource":    map[string]any{"Database": map[string]any{"Name": "db1"}},
				"Permissions": []string{"SELECT"},
			},
			{
				"Id":          "entry-2",
				"Principal":   map[string]any{"DataLakePrincipalIdentifier": "role2"},
				"Resource":    map[string]any{"Database": map[string]any{"Name": "db2"}},
				"Permissions": []string{"ALL"},
			},
		},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotNil(t, m["Failures"])

	// Verify permissions were granted
	listResp, _ := s.HandleRequest(jsonCtx("ListPermissions", map[string]any{}))
	lm := respJSON(t, listResp)
	perms := lm["PrincipalResourcePermissions"].([]any)
	assert.Len(t, perms, 2)
}

func TestBatchRevokePermissions(t *testing.T) {
	s := newService()
	// First grant permissions
	_, _ = s.HandleRequest(jsonCtx("BatchGrantPermissions", map[string]any{
		"Entries": []map[string]any{
			{
				"Id":          "e1",
				"Principal":   map[string]any{"DataLakePrincipalIdentifier": "brole1"},
				"Resource":    map[string]any{"Database": map[string]any{"Name": "bdb1"}},
				"Permissions": []string{"SELECT"},
			},
			{
				"Id":          "e2",
				"Principal":   map[string]any{"DataLakePrincipalIdentifier": "brole2"},
				"Resource":    map[string]any{"Database": map[string]any{"Name": "bdb2"}},
				"Permissions": []string{"ALL"},
			},
		},
	}))

	// Revoke one
	resp, err := s.HandleRequest(jsonCtx("BatchRevokePermissions", map[string]any{
		"Entries": []map[string]any{
			{
				"Id":        "e1",
				"Principal": map[string]any{"DataLakePrincipalIdentifier": "brole1"},
				"Resource":  map[string]any{"Database": map[string]any{"Name": "bdb1"}},
			},
		},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotNil(t, m["Failures"])

	// Verify only one remains
	listResp, _ := s.HandleRequest(jsonCtx("ListPermissions", map[string]any{}))
	lm := respJSON(t, listResp)
	perms := lm["PrincipalResourcePermissions"].([]any)
	assert.Len(t, perms, 1)
}

func TestDescribeResource(t *testing.T) {
	s := newService()
	arn := "arn:aws:s3:::my-describe-bucket"
	_, _ = s.HandleRequest(jsonCtx("RegisterResource", map[string]any{
		"ResourceArn": arn, "RoleArn": "arn:aws:iam::123456789012:role/lf",
	}))

	resp, err := s.HandleRequest(jsonCtx("DescribeResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	info := m["ResourceInfo"].(map[string]any)
	assert.Equal(t, arn, info["ResourceArn"])
}

func TestDescribeResourceNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeResource", map[string]any{"ResourceArn": "arn:aws:s3:::nonexistent"}))
	require.Error(t, err)
}

func TestRemoveLFTagsFromResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("AddLFTagsToResource", map[string]any{
		"Resource": map[string]any{"Database": map[string]any{"Name": "rm-db"}},
		"LFTags":   []map[string]any{{"TagKey": "k1", "TagValues": []string{"v1"}}, {"TagKey": "k2", "TagValues": []string{"v2"}}},
	}))

	resp, err := s.HandleRequest(jsonCtx("RemoveLFTagsFromResource", map[string]any{
		"Resource": map[string]any{"Database": map[string]any{"Name": "rm-db"}},
		"LFTags":   []map[string]any{{"TagKey": "k1"}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotNil(t, m["Failures"])
}

func TestPutAndGetDataLakeSettings(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("PutDataLakeSettings", map[string]any{
		"DataLakeSettings": map[string]any{
			"DataLakeAdmins": []map[string]any{
				{"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/admin1"},
				{"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/admin2"},
			},
		},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetDataLakeSettings", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	settings := m["DataLakeSettings"].(map[string]any)
	admins := settings["DataLakeAdmins"].([]any)
	assert.Len(t, admins, 2)
}

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("Bogus", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}
