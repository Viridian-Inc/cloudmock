package dax_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/dax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.DAXService { return svc.New("123456789012", "us-east-1") }

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

func decode(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func mustCreateCluster(t *testing.T, s *svc.DAXService, name string) map[string]any {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName":       name,
		"NodeType":          "dax.r4.large",
		"ReplicationFactor": 1,
		"IamRoleArn":        "arn:aws:iam::123456789012:role/dax-role",
	}))
	require.NoError(t, err)
	return decode(t, resp)["Cluster"].(map[string]any)
}

// ---- Cluster tests ----

func TestDAX_CreateCluster(t *testing.T) {
	s := newService()
	cluster := mustCreateCluster(t, s, "my-cluster")
	assert.Equal(t, "my-cluster", cluster["ClusterName"])
	assert.NotEmpty(t, cluster["ClusterArn"])
	assert.Contains(t, cluster["ClusterArn"].(string), "arn:aws:dax:us-east-1:123456789012:cache/my-cluster")
	assert.NotEmpty(t, cluster["Status"])
	assert.NotNil(t, cluster["ClusterDiscoveryEndpoint"])
}

func TestDAX_CreateClusterMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"NodeType": "dax.r4.large", "IamRoleArn": "arn:aws:iam::123456789012:role/dax",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
}

func TestDAX_CreateClusterMissingIamRole(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName": "no-role", "NodeType": "dax.r4.large",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
}

func TestDAX_CreateDuplicateCluster(t *testing.T) {
	s := newService()
	mustCreateCluster(t, s, "dup-cluster")
	_, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName": "dup-cluster", "NodeType": "dax.r4.large",
		"IamRoleArn": "arn:aws:iam::123456789012:role/dax",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ClusterAlreadyExistsFault", awsErr.Code)
}

func TestDAX_DescribeClusters(t *testing.T) {
	s := newService()
	mustCreateCluster(t, s, "cluster-1")
	mustCreateCluster(t, s, "cluster-2")

	resp, err := s.HandleRequest(jsonCtx("DescribeClusters", map[string]any{}))
	require.NoError(t, err)
	clusters := decode(t, resp)["Clusters"].([]any)
	assert.Len(t, clusters, 2)
}

func TestDAX_DescribeClustersByName(t *testing.T) {
	s := newService()
	mustCreateCluster(t, s, "c1")
	mustCreateCluster(t, s, "c2")

	resp, err := s.HandleRequest(jsonCtx("DescribeClusters", map[string]any{
		"ClusterNames": []string{"c1"},
	}))
	require.NoError(t, err)
	clusters := decode(t, resp)["Clusters"].([]any)
	assert.Len(t, clusters, 1)
	assert.Equal(t, "c1", clusters[0].(map[string]any)["ClusterName"])
}

func TestDAX_UpdateCluster(t *testing.T) {
	s := newService()
	mustCreateCluster(t, s, "upd-cluster")

	resp, err := s.HandleRequest(jsonCtx("UpdateCluster", map[string]any{
		"ClusterName": "upd-cluster",
		"Description": "Updated description",
	}))
	require.NoError(t, err)
	cluster := decode(t, resp)["Cluster"].(map[string]any)
	assert.Equal(t, "Updated description", cluster["Description"])
}

func TestDAX_UpdateClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("UpdateCluster", map[string]any{
		"ClusterName": "nonexistent", "Description": "update",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ClusterNotFoundFault", awsErr.Code)
}

func TestDAX_DeleteCluster(t *testing.T) {
	s := newService()
	mustCreateCluster(t, s, "del-cluster")

	resp, err := s.HandleRequest(jsonCtx("DeleteCluster", map[string]any{
		"ClusterName": "del-cluster",
	}))
	require.NoError(t, err)
	cluster := decode(t, resp)["Cluster"].(map[string]any)
	assert.Equal(t, "deleting", cluster["Status"])

	// Verify it's gone
	listResp, _ := s.HandleRequest(jsonCtx("DescribeClusters", map[string]any{}))
	clusters := decode(t, listResp)["Clusters"].([]any)
	assert.Len(t, clusters, 0)
}

func TestDAX_DeleteClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteCluster", map[string]any{"ClusterName": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ClusterNotFoundFault", awsErr.Code)
}

func TestDAX_IncreaseReplicationFactor(t *testing.T) {
	s := newService()
	mustCreateCluster(t, s, "rf-cluster")

	resp, err := s.HandleRequest(jsonCtx("IncreaseReplicationFactor", map[string]any{
		"ClusterName":          "rf-cluster",
		"NewReplicationFactor": 3,
	}))
	require.NoError(t, err)
	cluster := decode(t, resp)["Cluster"].(map[string]any)
	assert.Equal(t, float64(3), cluster["ReplicationFactor"])
}

func TestDAX_DecreaseReplicationFactor(t *testing.T) {
	s := newService()
	// Create cluster with RF=3
	s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName":       "rf3-cluster",
		"NodeType":          "dax.r4.large",
		"ReplicationFactor": 3,
		"IamRoleArn":        "arn:aws:iam::123456789012:role/dax",
	}))

	resp, err := s.HandleRequest(jsonCtx("DecreaseReplicationFactor", map[string]any{
		"ClusterName":          "rf3-cluster",
		"NewReplicationFactor": 1,
	}))
	require.NoError(t, err)
	cluster := decode(t, resp)["Cluster"].(map[string]any)
	assert.Equal(t, float64(1), cluster["ReplicationFactor"])
}

// ---- Subnet Group tests ----

func TestDAX_CreateAndDescribeSubnetGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateSubnetGroup", map[string]any{
		"SubnetGroupName": "my-subnet-group",
		"Description":     "Test subnet group",
		"SubnetIds":       []string{"subnet-aaa", "subnet-bbb"},
	}))
	require.NoError(t, err)
	sg := decode(t, resp)["SubnetGroup"].(map[string]any)
	assert.Equal(t, "my-subnet-group", sg["SubnetGroupName"])

	listResp, err := s.HandleRequest(jsonCtx("DescribeSubnetGroups", map[string]any{}))
	require.NoError(t, err)
	groups := decode(t, listResp)["SubnetGroups"].([]any)
	assert.Len(t, groups, 1)
}

func TestDAX_DeleteSubnetGroup(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSubnetGroup", map[string]any{
		"SubnetGroupName": "del-sg",
	}))

	_, err := s.HandleRequest(jsonCtx("DeleteSubnetGroup", map[string]any{
		"SubnetGroupName": "del-sg",
	}))
	require.NoError(t, err)

	// Verify it's gone
	listResp, _ := s.HandleRequest(jsonCtx("DescribeSubnetGroups", map[string]any{}))
	groups := decode(t, listResp)["SubnetGroups"].([]any)
	assert.Len(t, groups, 0)
}

func TestDAX_DeleteSubnetGroupNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteSubnetGroup", map[string]any{"SubnetGroupName": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "SubnetGroupNotFoundFault", awsErr.Code)
}

// ---- Parameter Group tests ----

func TestDAX_CreateAndDescribeParameterGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateParameterGroup", map[string]any{
		"ParameterGroupName": "my-param-group",
		"Description":        "Test parameter group",
	}))
	require.NoError(t, err)
	pg := decode(t, resp)["ParameterGroup"].(map[string]any)
	assert.Equal(t, "my-param-group", pg["ParameterGroupName"])

	listResp, err := s.HandleRequest(jsonCtx("DescribeParameterGroups", map[string]any{}))
	require.NoError(t, err)
	groups := decode(t, listResp)["ParameterGroups"].([]any)
	assert.Len(t, groups, 1)
}

func TestDAX_UpdateParameterGroup(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateParameterGroup", map[string]any{
		"ParameterGroupName": "upd-pg",
	}))

	resp, err := s.HandleRequest(jsonCtx("UpdateParameterGroup", map[string]any{
		"ParameterGroupName": "upd-pg",
		"ParameterNameValues": []map[string]string{
			{"ParameterName": "query-ttl-millis", "ParameterValue": "600000"},
		},
	}))
	require.NoError(t, err)
	pg := decode(t, resp)["ParameterGroup"].(map[string]any)
	assert.Equal(t, "upd-pg", pg["ParameterGroupName"])

	// Describe parameters to verify update
	paramsResp, err := s.HandleRequest(jsonCtx("DescribeParameters", map[string]any{
		"ParameterGroupName": "upd-pg",
	}))
	require.NoError(t, err)
	params := decode(t, paramsResp)["Parameters"].([]any)
	found := false
	for _, p := range params {
		pm := p.(map[string]any)
		if pm["ParameterName"] == "query-ttl-millis" && pm["ParameterValue"] == "600000" {
			found = true
		}
	}
	assert.True(t, found, "updated parameter should be present")
}

func TestDAX_DeleteParameterGroup(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateParameterGroup", map[string]any{
		"ParameterGroupName": "del-pg",
	}))

	_, err := s.HandleRequest(jsonCtx("DeleteParameterGroup", map[string]any{
		"ParameterGroupName": "del-pg",
	}))
	require.NoError(t, err)
}

func TestDAX_DescribeDefaultParameters(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("DescribeDefaultParameters", map[string]any{}))
	require.NoError(t, err)
	params := decode(t, resp)["Parameters"].([]any)
	assert.NotEmpty(t, params)
	// Verify expected default params
	names := make([]string, 0, len(params))
	for _, p := range params {
		names = append(names, p.(map[string]any)["ParameterName"].(string))
	}
	assert.Contains(t, names, "query-ttl-millis")
	assert.Contains(t, names, "record-ttl-millis")
}

// ---- Tagging tests ----

func TestDAX_TagAndListTags(t *testing.T) {
	s := newService()
	cluster := mustCreateCluster(t, s, "tag-cluster")
	arn := cluster["ClusterArn"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceName": arn,
		"Tags": []map[string]string{
			{"Key": "env", "Value": "prod"},
			{"Key": "team", "Value": "platform"},
		},
	}))
	require.NoError(t, err)

	tagsResp, err := s.HandleRequest(jsonCtx("ListTags", map[string]any{
		"ResourceName": arn,
	}))
	require.NoError(t, err)
	tags := decode(t, tagsResp)["Tags"].([]any)
	assert.Len(t, tags, 2)
}

func TestDAX_UntagResource(t *testing.T) {
	s := newService()
	cluster := mustCreateCluster(t, s, "untag-cluster")
	arn := cluster["ClusterArn"].(string)

	s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceName": arn,
		"Tags": []map[string]string{
			{"Key": "env", "Value": "prod"},
			{"Key": "team", "Value": "sre"},
		},
	}))

	_, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceName": arn,
		"TagKeys":      []string{"team"},
	}))
	require.NoError(t, err)

	tagsResp, _ := s.HandleRequest(jsonCtx("ListTags", map[string]any{"ResourceName": arn}))
	tags := decode(t, tagsResp)["Tags"].([]any)
	assert.Len(t, tags, 1)
	assert.Equal(t, "env", tags[0].(map[string]any)["Key"])
}

func TestDAX_TagResourceNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceName": "arn:aws:dax:us-east-1:123456789012:cache/nonexistent",
		"Tags":         []map[string]string{{"Key": "x", "Value": "y"}},
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidARNFault", awsErr.Code)
}

// ---- SSE test ----

func TestDAX_ClusterWithSSE(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName":       "sse-cluster",
		"NodeType":          "dax.r4.large",
		"ReplicationFactor": 1,
		"IamRoleArn":        "arn:aws:iam::123456789012:role/dax",
		"SSESpecification":  map[string]any{"Enabled": true},
	}))
	require.NoError(t, err)
	cluster := decode(t, resp)["Cluster"].(map[string]any)
	sse := cluster["SSEDescription"].(map[string]any)
	assert.Equal(t, "ENABLED", sse["Status"])
}

// ---- Invalid action test ----

func TestDAX_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("NonExistentAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}
