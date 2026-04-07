package docdb_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/docdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.DocDBService {
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
	assert.Equal(t, "docdb", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestCreateDBCluster(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "test-cluster", "Engine": "docdb",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateDBClusterDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "dup"}))
	_, err := s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "dup"}))
	require.Error(t, err)
}

func TestDescribeDBClusters(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "c1"}))
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "c2"}))
	resp, err := s.HandleRequest(queryCtx("DescribeDBClusters", map[string]string{}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeDBClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("DescribeDBClusters", map[string]string{"DBClusterIdentifier": "ghost"}))
	require.Error(t, err)
}

func TestModifyDBCluster(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "mod-c"}))
	resp, err := s.HandleRequest(queryCtx("ModifyDBCluster", map[string]string{
		"DBClusterIdentifier": "mod-c", "EngineVersion": "5.0.0",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteDBCluster(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "del-c"}))
	resp, err := s.HandleRequest(queryCtx("DeleteDBCluster", map[string]string{"DBClusterIdentifier": "del-c"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteDBClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("DeleteDBCluster", map[string]string{"DBClusterIdentifier": "nope"}))
	require.Error(t, err)
}

func TestCreateDBInstance(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "inst-c"}))
	resp, err := s.HandleRequest(queryCtx("CreateDBInstance", map[string]string{
		"DBInstanceIdentifier": "inst-1", "DBClusterIdentifier": "inst-c",
		"DBInstanceClass": "db.r5.large", "Engine": "docdb",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeDBInstances(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBInstance", map[string]string{"DBInstanceIdentifier": "di1", "DBInstanceClass": "db.r5.large"}))
	resp, err := s.HandleRequest(queryCtx("DescribeDBInstances", map[string]string{}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeDBInstanceNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("DescribeDBInstances", map[string]string{"DBInstanceIdentifier": "ghost"}))
	require.Error(t, err)
}

func TestModifyDBInstance(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBInstance", map[string]string{"DBInstanceIdentifier": "mod-i", "DBInstanceClass": "db.r5.large"}))
	resp, err := s.HandleRequest(queryCtx("ModifyDBInstance", map[string]string{
		"DBInstanceIdentifier": "mod-i", "DBInstanceClass": "db.r5.2xlarge",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteDBInstance(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBInstance", map[string]string{"DBInstanceIdentifier": "del-i"}))
	resp, err := s.HandleRequest(queryCtx("DeleteDBInstance", map[string]string{"DBInstanceIdentifier": "del-i"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateDBClusterSnapshot(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "snap-c"}))
	resp, err := s.HandleRequest(queryCtx("CreateDBClusterSnapshot", map[string]string{
		"DBClusterSnapshotIdentifier": "snap-1", "DBClusterIdentifier": "snap-c",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteDBClusterSnapshot(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "dsnapc"}))
	_, _ = s.HandleRequest(queryCtx("CreateDBClusterSnapshot", map[string]string{
		"DBClusterSnapshotIdentifier": "delsn", "DBClusterIdentifier": "dsnapc",
	}))
	resp, err := s.HandleRequest(queryCtx("DeleteDBClusterSnapshot", map[string]string{"DBClusterSnapshotIdentifier": "delsn"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateDBSubnetGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(queryCtx("CreateDBSubnetGroup", map[string]string{
		"DBSubnetGroupName": "sg1", "DBSubnetGroupDescription": "test", "SubnetIds.member.1": "subnet-abc",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteDBSubnetGroup(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBSubnetGroup", map[string]string{"DBSubnetGroupName": "dsg"}))
	resp, err := s.HandleRequest(queryCtx("DeleteDBSubnetGroup", map[string]string{"DBSubnetGroupName": "dsg"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddTagsToResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "tag-c"}))
	arn := "arn:aws:rds:us-east-1:123456789012:cluster:tag-c"
	resp, err := s.HandleRequest(queryCtx("AddTagsToResource", map[string]string{
		"ResourceName": arn, "Tags.member.1.Key": "env", "Tags.member.1.Value": "prod",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListTagsForResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "ltag-c"}))
	arn := "arn:aws:rds:us-east-1:123456789012:cluster:ltag-c"
	_, _ = s.HandleRequest(queryCtx("AddTagsToResource", map[string]string{
		"ResourceName": arn, "Tags.member.1.Key": "team", "Tags.member.1.Value": "data",
	}))
	resp, err := s.HandleRequest(queryCtx("ListTagsForResource", map[string]string{"ResourceName": arn}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRemoveTagsFromResource(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{"DBClusterIdentifier": "rmtag-c"}))
	arn := "arn:aws:rds:us-east-1:123456789012:cluster:rmtag-c"
	_, _ = s.HandleRequest(queryCtx("AddTagsToResource", map[string]string{
		"ResourceName": arn, "Tags.member.1.Key": "rm", "Tags.member.1.Value": "me",
	}))
	resp, err := s.HandleRequest(queryCtx("RemoveTagsFromResource", map[string]string{
		"ResourceName": arn, "TagKeys.member.1": "rm",
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

// ---- Behavioral: Instance roles (writer vs reader) ----

func TestDocDBInstanceRoles(t *testing.T) {
	s := newService()

	_, err := s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "role-cluster",
	}))
	require.NoError(t, err)

	// First instance is writer
	_, err = s.HandleRequest(queryCtx("CreateDBInstance", map[string]string{
		"DBInstanceIdentifier": "inst-writer",
		"DBClusterIdentifier":  "role-cluster",
		"DBInstanceClass":      "db.r5.large",
	}))
	require.NoError(t, err)

	// Second is reader
	_, err = s.HandleRequest(queryCtx("CreateDBInstance", map[string]string{
		"DBInstanceIdentifier": "inst-reader",
		"DBClusterIdentifier":  "role-cluster",
		"DBInstanceClass":      "db.r5.large",
	}))
	require.NoError(t, err)

	// Both should exist
	resp, err := s.HandleRequest(queryCtx("DescribeDBInstances", map[string]string{}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ---- Behavioral: DocumentDB uses port 27017 ----

func TestDocDBClusterEndpointPort(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(queryCtx("CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "port-cluster",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
