package redshift_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/redshift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.RedshiftService {
	return svc.New("123456789012", "us-east-1")
}

func queryCtx(action string, params map[string]string) *service.RequestContext {
	vals := url.Values{}
	vals.Set("Action", action)
	for k, v := range params {
		vals.Set(k, v)
	}
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       []byte(vals.Encode()),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func TestServiceName(t *testing.T) {
	assert.Equal(t, "redshift", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestCreateCluster(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "test-cluster", "NodeType": "dc2.large", "MasterUsername": "admin",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateClusterDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "dup"}))
	_, err := s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "dup"}))
	require.Error(t, err)
}

func TestDescribeClusters(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "c1"}))
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "c2"}))
	resp, err := s.HandleRequest(queryCtx("DescribeClusters", map[string]string{}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeClusterByID(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "specific"}))
	resp, err := s.HandleRequest(queryCtx("DescribeClusters", map[string]string{"ClusterIdentifier": "specific"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("DescribeClusters", map[string]string{"ClusterIdentifier": "ghost"}))
	require.Error(t, err)
}

func TestModifyCluster(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "mod-c", "NodeType": "dc2.large"}))
	resp, err := s.HandleRequest(queryCtx("ModifyCluster", map[string]string{
		"ClusterIdentifier": "mod-c", "NodeType": "ra3.xlplus", "NumberOfNodes": "4",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteCluster(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "del-c"}))
	resp, err := s.HandleRequest(queryCtx("DeleteCluster", map[string]string{"ClusterIdentifier": "del-c"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("DeleteCluster", map[string]string{"ClusterIdentifier": "nope"}))
	require.Error(t, err)
}

func TestRebootCluster(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "reboot-c"}))
	resp, err := s.HandleRequest(queryCtx("RebootCluster", map[string]string{"ClusterIdentifier": "reboot-c"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateClusterSnapshot(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "snap-c"}))
	resp, err := s.HandleRequest(queryCtx("CreateClusterSnapshot", map[string]string{
		"SnapshotIdentifier": "snap-1", "ClusterIdentifier": "snap-c",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeClusterSnapshots(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "dsnap-c"}))
	_, _ = s.HandleRequest(queryCtx("CreateClusterSnapshot", map[string]string{"SnapshotIdentifier": "s1", "ClusterIdentifier": "dsnap-c"}))
	resp, err := s.HandleRequest(queryCtx("DescribeClusterSnapshots", map[string]string{"ClusterIdentifier": "dsnap-c"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteClusterSnapshot(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "ds-c"}))
	_, _ = s.HandleRequest(queryCtx("CreateClusterSnapshot", map[string]string{"SnapshotIdentifier": "ds1", "ClusterIdentifier": "ds-c"}))
	resp, err := s.HandleRequest(queryCtx("DeleteClusterSnapshot", map[string]string{"SnapshotIdentifier": "ds1"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRestoreFromClusterSnapshot(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "orig"}))
	_, _ = s.HandleRequest(queryCtx("CreateClusterSnapshot", map[string]string{"SnapshotIdentifier": "restore-snap", "ClusterIdentifier": "orig"}))
	resp, err := s.HandleRequest(queryCtx("RestoreFromClusterSnapshot", map[string]string{
		"ClusterIdentifier": "restored", "SnapshotIdentifier": "restore-snap",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateClusterSubnetGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(queryCtx("CreateClusterSubnetGroup", map[string]string{
		"ClusterSubnetGroupName": "sg1", "Description": "test",
		"SubnetIds.SubnetIdentifier.1": "subnet-abc",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteClusterSubnetGroup(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateClusterSubnetGroup", map[string]string{"ClusterSubnetGroupName": "dsg"}))
	resp, err := s.HandleRequest(queryCtx("DeleteClusterSubnetGroup", map[string]string{"ClusterSubnetGroupName": "dsg"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateClusterParameterGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(queryCtx("CreateClusterParameterGroup", map[string]string{
		"ParameterGroupName": "pg1", "ParameterGroupFamily": "redshift-1.0", "Description": "test",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteClusterParameterGroup(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateClusterParameterGroup", map[string]string{"ParameterGroupName": "dpg"}))
	resp, err := s.HandleRequest(queryCtx("DeleteClusterParameterGroup", map[string]string{"ParameterGroupName": "dpg"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClusterTagging(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{"ClusterIdentifier": "tag-c"}))
	// Get the ARN
	arn := "arn:aws:redshift:us-east-1:123456789012:cluster:tag-c"
	resp, err := s.HandleRequest(queryCtx("CreateTags", map[string]string{
		"ResourceName": arn, "Tags.Tag.1.Key": "env", "Tags.Tag.1.Value": "prod",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = s.HandleRequest(queryCtx("DescribeTags", map[string]string{"ResourceName": arn}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = s.HandleRequest(queryCtx("DeleteTags", map[string]string{
		"ResourceName": arn, "TagKeys.TagKey.1": "env",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("BogusAction", map[string]string{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Behavioral tests ----

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

func TestExecuteStatement(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "exec-cluster", "NodeType": "dc2.large", "MasterUsername": "admin",
	}))

	resp, err := s.HandleRequest(jsonCtx("ExecuteStatement", map[string]any{
		"ClusterIdentifier": "exec-cluster",
		"Database":          "dev",
		"Sql":               "SELECT 1",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotEmpty(t, m["Id"])
	assert.Equal(t, "FINISHED", m["Status"])
}

func TestExecuteStatement_InvalidSQL(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "bad-sql-cluster", "NodeType": "dc2.large",
	}))

	resp, err := s.HandleRequest(jsonCtx("ExecuteStatement", map[string]any{
		"ClusterIdentifier": "bad-sql-cluster",
		"Sql":               "INVALID GIBBERISH",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "FAILED", m["Status"])
}

func TestExecuteStatement_ClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("ExecuteStatement", map[string]any{
		"ClusterIdentifier": "nonexistent",
		"Sql":               "SELECT 1",
	}))
	require.Error(t, err)
}

func TestDescribeStatement(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "desc-stmt-cluster", "NodeType": "dc2.large",
	}))
	execResp, _ := s.HandleRequest(jsonCtx("ExecuteStatement", map[string]any{
		"ClusterIdentifier": "desc-stmt-cluster", "Sql": "SELECT id, name FROM t",
	}))
	em := respJSON(t, execResp)
	stmtID := em["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("DescribeStatement", map[string]any{"Id": stmtID}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "FINISHED", m["Status"])
	assert.Equal(t, float64(5), m["ResultRows"])
}

func TestGetStatementResult(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "result-cluster", "NodeType": "dc2.large",
	}))
	execResp, _ := s.HandleRequest(jsonCtx("ExecuteStatement", map[string]any{
		"ClusterIdentifier": "result-cluster", "Sql": "SELECT id, name FROM t",
	}))
	em := respJSON(t, execResp)
	stmtID := em["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetStatementResult", map[string]any{"Id": stmtID}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, float64(5), m["TotalNumRows"])
	records := m["Records"].([]any)
	assert.Len(t, records, 5)
	colMeta := m["ColumnMetadata"].([]any)
	assert.Len(t, colMeta, 2)
}

// ---- PauseCluster / ResumeCluster ----

func TestPauseCluster(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "pause-cluster", "NodeType": "dc2.large",
	}))
	resp, err := s.HandleRequest(queryCtx("PauseCluster", map[string]string{
		"ClusterIdentifier": "pause-cluster",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPauseCluster_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("PauseCluster", map[string]string{
		"ClusterIdentifier": "nonexistent",
	}))
	require.Error(t, err)
}

func TestResumeCluster(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "resume-cluster", "NodeType": "dc2.large",
	}))
	_, _ = s.HandleRequest(queryCtx("PauseCluster", map[string]string{
		"ClusterIdentifier": "resume-cluster",
	}))
	resp, err := s.HandleRequest(queryCtx("ResumeCluster", map[string]string{
		"ClusterIdentifier": "resume-cluster",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestResumeCluster_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("ResumeCluster", map[string]string{
		"ClusterIdentifier": "nonexistent",
	}))
	require.Error(t, err)
}

func TestPauseResumeCycle(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "cycle-cluster", "NodeType": "dc2.large",
	}))

	// Pause
	resp, err := s.HandleRequest(queryCtx("PauseCluster", map[string]string{
		"ClusterIdentifier": "cycle-cluster",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Resume
	resp, err = s.HandleRequest(queryCtx("ResumeCluster", map[string]string{
		"ClusterIdentifier": "cycle-cluster",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddTagsToResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "tag-res-cluster", "NodeType": "dc2.large",
	}))
	arn := "arn:aws:redshift:us-east-1:123456789012:cluster:tag-res-cluster"
	resp, err := s.HandleRequest(queryCtx("AddTagsToResource", map[string]string{
		"ResourceName": arn, "Tags.Tag.1.Key": "env", "Tags.Tag.1.Value": "prod",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRemoveTagsFromResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "untag-res-cluster", "NodeType": "dc2.large",
	}))
	arn := "arn:aws:redshift:us-east-1:123456789012:cluster:untag-res-cluster"
	_, _ = s.HandleRequest(queryCtx("AddTagsToResource", map[string]string{
		"ResourceName": arn, "Tags.Tag.1.Key": "rm", "Tags.Tag.1.Value": "me",
	}))
	resp, err := s.HandleRequest(queryCtx("RemoveTagsFromResource", map[string]string{
		"ResourceName": arn, "TagKeys.TagKey.1": "rm",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRestoreFromSnapshot_CopiesSchema(t *testing.T) {
	s := newService()
	// Create cluster, create a table via ExecuteStatement, snapshot, restore
	_, _ = s.HandleRequest(queryCtx("CreateCluster", map[string]string{
		"ClusterIdentifier": "schema-cluster", "NodeType": "dc2.large",
	}))
	// Create a table
	_, _ = s.HandleRequest(jsonCtx("ExecuteStatement", map[string]any{
		"ClusterIdentifier": "schema-cluster",
		"Sql":               "CREATE TABLE users (id int, name varchar)",
	}))
	// Snapshot
	_, _ = s.HandleRequest(queryCtx("CreateClusterSnapshot", map[string]string{
		"SnapshotIdentifier": "schema-snap", "ClusterIdentifier": "schema-cluster",
	}))
	// Restore
	resp, err := s.HandleRequest(queryCtx("RestoreFromClusterSnapshot", map[string]string{
		"ClusterIdentifier": "restored-schema", "SnapshotIdentifier": "schema-snap",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// The restored cluster should have the same schema — SELECT from users should succeed
	execResp, err := s.HandleRequest(jsonCtx("ExecuteStatement", map[string]any{
		"ClusterIdentifier": "restored-schema",
		"Sql":               "SELECT id FROM users",
	}))
	require.NoError(t, err)
	em := respJSON(t, execResp)
	assert.Equal(t, "FINISHED", em["Status"])
}
