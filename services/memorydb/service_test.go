package memorydb_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/memorydb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.MemoryDBService {
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
	assert.Equal(t, "memorydb", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestCreateCluster(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName": "test-cluster", "NodeType": "db.r6g.large", "ACLName": "open-access",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	c := m["Cluster"].(map[string]any)
	assert.Equal(t, "test-cluster", c["Name"])
	assert.NotEmpty(t, c["ARN"])
}

func TestCreateClusterDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "dup"}))
	_, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "dup"}))
	require.Error(t, err)
}

func TestDescribeClusters(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "dc1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "dc2"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeClusters", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	clusters := m["Clusters"].([]any)
	assert.Len(t, clusters, 2)
}

func TestDescribeClustersSpecific(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "spec"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeClusters", map[string]any{"ClusterName": "spec"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	clusters := m["Clusters"].([]any)
	assert.Len(t, clusters, 1)
}

func TestDescribeClustersNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeClusters", map[string]any{"ClusterName": "ghost"}))
	require.Error(t, err)
}

func TestUpdateCluster(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "upd-c"}))
	resp, err := s.HandleRequest(jsonCtx("UpdateCluster", map[string]any{
		"ClusterName": "upd-c", "NodeType": "db.r6g.2xlarge",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	c := m["Cluster"].(map[string]any)
	assert.Equal(t, "db.r6g.2xlarge", c["NodeType"])
}

func TestDeleteCluster(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "del-c"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteCluster", map[string]any{"ClusterName": "del-c"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	c := m["Cluster"].(map[string]any)
	assert.Equal(t, "deleting", c["Status"])
}

func TestDeleteClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteCluster", map[string]any{"ClusterName": "nope"}))
	require.Error(t, err)
}

func TestCreateACL(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateACL", map[string]any{
		"ACLName": "my-acl", "UserNames": []string{"user1"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	acl := m["ACL"].(map[string]any)
	assert.Equal(t, "my-acl", acl["Name"])
	assert.Equal(t, "active", acl["Status"])
}

func TestDescribeACLs(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateACL", map[string]any{"ACLName": "a1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateACL", map[string]any{"ACLName": "a2"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeACLs", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	acls := m["ACLs"].([]any)
	assert.Len(t, acls, 2)
}

func TestUpdateACL(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateACL", map[string]any{"ACLName": "upd-acl", "UserNames": []string{"u1"}}))
	resp, err := s.HandleRequest(jsonCtx("UpdateACL", map[string]any{
		"ACLName": "upd-acl", "UserNamesToAdd": []string{"u2"}, "UserNamesToRemove": []string{"u1"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	acl := m["ACL"].(map[string]any)
	users := acl["UserNames"].([]any)
	assert.Contains(t, users, "u2")
}

func TestDeleteACL(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateACL", map[string]any{"ACLName": "del-acl"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteACL", map[string]any{"ACLName": "del-acl"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateUser(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"UserName": "testuser", "AccessString": "on ~* &* +@all",
		"AuthenticationMode": map[string]any{"Type": "password", "Passwords": []string{"pass123!"}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	u := m["User"].(map[string]any)
	assert.Equal(t, "testuser", u["Name"])
	assert.Equal(t, "active", u["Status"])
}

func TestDescribeUsers(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateUser", map[string]any{"UserName": "u1", "AccessString": "on"}))
	_, _ = s.HandleRequest(jsonCtx("CreateUser", map[string]any{"UserName": "u2", "AccessString": "on"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeUsers", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	users := m["Users"].([]any)
	assert.Len(t, users, 2)
}

func TestUpdateUser(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateUser", map[string]any{"UserName": "upd-u", "AccessString": "on ~*"}))
	resp, err := s.HandleRequest(jsonCtx("UpdateUser", map[string]any{
		"UserName": "upd-u", "AccessString": "on ~app* &* +@read",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	u := m["User"].(map[string]any)
	assert.Equal(t, "on ~app* &* +@read", u["AccessString"])
}

func TestDeleteUser(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateUser", map[string]any{"UserName": "del-u", "AccessString": "on"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteUser", map[string]any{"UserName": "del-u"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateSubnetGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateSubnetGroup", map[string]any{
		"SubnetGroupName": "sg1", "Description": "test", "SubnetIds": []string{"subnet-abc"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	sg := m["SubnetGroup"].(map[string]any)
	assert.Equal(t, "sg1", sg["Name"])
}

func TestDeleteSubnetGroup(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateSubnetGroup", map[string]any{"SubnetGroupName": "dsg", "SubnetIds": []string{"s1"}}))
	resp, err := s.HandleRequest(jsonCtx("DeleteSubnetGroup", map[string]any{"SubnetGroupName": "dsg"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateParameterGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateParameterGroup", map[string]any{
		"ParameterGroupName": "pg1", "Family": "memorydb_redis7", "Description": "test",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	pg := m["ParameterGroup"].(map[string]any)
	assert.Equal(t, "pg1", pg["Name"])
}

func TestDeleteParameterGroup(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateParameterGroup", map[string]any{"ParameterGroupName": "dpg", "Family": "memorydb_redis7"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteParameterGroup", map[string]any{"ParameterGroupName": "dpg"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateSnapshot(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "snap-c"}))
	resp, err := s.HandleRequest(jsonCtx("CreateSnapshot", map[string]any{
		"SnapshotName": "snap1", "ClusterName": "snap-c",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	snap := m["Snapshot"].(map[string]any)
	assert.Equal(t, "snap1", snap["Name"])
	assert.Equal(t, "available", snap["Status"])
}

func TestDescribeSnapshots(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "dsnap-c"}))
	_, _ = s.HandleRequest(jsonCtx("CreateSnapshot", map[string]any{"SnapshotName": "s1", "ClusterName": "dsnap-c"}))
	_, _ = s.HandleRequest(jsonCtx("CreateSnapshot", map[string]any{"SnapshotName": "s2", "ClusterName": "dsnap-c"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeSnapshots", map[string]any{"ClusterName": "dsnap-c"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	snaps := m["Snapshots"].([]any)
	assert.Len(t, snaps, 2)
}

func TestDeleteSnapshot(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "delsnapc"}))
	_, _ = s.HandleRequest(jsonCtx("CreateSnapshot", map[string]any{"SnapshotName": "ds1", "ClusterName": "delsnapc"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteSnapshot", map[string]any{"SnapshotName": "ds1"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTagResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{"ClusterName": "tag-c"}))
	arn := "arn:aws:memorydb:us-east-1:123456789012:cluster/tag-c"
	resp, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": arn, "Tags": []map[string]string{{"Key": "env", "Value": "prod"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListTags(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName": "ltag-c", "Tags": []map[string]string{{"Key": "k", "Value": "v"}},
	}))
	arn := "arn:aws:memorydb:us-east-1:123456789012:cluster/ltag-c"
	resp, err := s.HandleRequest(jsonCtx("ListTags", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tags := m["TagList"].([]any)
	assert.Len(t, tags, 1)
}

func TestUntagResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName": "untag-c", "Tags": []map[string]string{{"Key": "rm", "Value": "me"}},
	}))
	arn := "arn:aws:memorydb:us-east-1:123456789012:cluster/untag-c"
	resp, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": arn, "TagKeys": []string{"rm"},
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
