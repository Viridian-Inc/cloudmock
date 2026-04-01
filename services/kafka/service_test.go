package kafka_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.KafkaService {
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

func createCluster(t *testing.T, s *svc.KafkaService, name string) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"clusterName":         name,
		"kafkaVersion":        "3.5.1",
		"numberOfBrokerNodes": float64(3),
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	return body["clusterArn"].(string)
}

// ---- Test 1: CreateCluster ----

func TestCreateCluster(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"clusterName":         "test-cluster",
		"kafkaVersion":        "3.5.1",
		"numberOfBrokerNodes": float64(3),
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["clusterArn"])
	assert.Contains(t, []string{"CREATING", "ACTIVE"}, body["state"])
}

// ---- Test 2: DescribeCluster ----

func TestDescribeCluster(t *testing.T) {
	s := newService()
	arn := createCluster(t, s, "desc-cluster")

	resp, err := s.HandleRequest(jsonCtx("DescribeCluster", map[string]any{"clusterArn": arn}))
	require.NoError(t, err)
	body := respBody(t, resp)
	info := body["clusterInfo"].(map[string]any)
	assert.Equal(t, "desc-cluster", info["clusterName"])
}

// ---- Test 3: ListClusters ----

func TestListClusters(t *testing.T) {
	s := newService()
	createCluster(t, s, "list-1")
	createCluster(t, s, "list-2")

	resp, err := s.HandleRequest(jsonCtx("ListClusters", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	clusters := body["clusterInfoList"].([]any)
	assert.Len(t, clusters, 2)
}

// ---- Test 4: DeleteCluster ----

func TestDeleteCluster(t *testing.T) {
	s := newService()
	arn := createCluster(t, s, "del-cluster")

	resp, err := s.HandleRequest(jsonCtx("DeleteCluster", map[string]any{"clusterArn": arn}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "DELETING", body["state"])

	_, err = s.HandleRequest(jsonCtx("DescribeCluster", map[string]any{"clusterArn": arn}))
	require.Error(t, err)
}

// ---- Test 5: Cluster lifecycle CREATING -> ACTIVE ----

func TestClusterLifecycle(t *testing.T) {
	s := newService()
	arn := createCluster(t, s, "lc-cluster")

	resp, err := s.HandleRequest(jsonCtx("DescribeCluster", map[string]any{"clusterArn": arn}))
	require.NoError(t, err)
	info := respBody(t, resp)["clusterInfo"].(map[string]any)
	assert.Contains(t, []string{"CREATING", "ACTIVE"}, info["state"])

	time.Sleep(3 * time.Second)
	resp2, err := s.HandleRequest(jsonCtx("DescribeCluster", map[string]any{"clusterArn": arn}))
	require.NoError(t, err)
	info2 := respBody(t, resp2)["clusterInfo"].(map[string]any)
	assert.Equal(t, "ACTIVE", info2["state"])
}

// ---- Test 6: UpdateBrokerCount ----

func TestUpdateBrokerCount(t *testing.T) {
	s := newService()
	arn := createCluster(t, s, "upd-broker")

	resp, err := s.HandleRequest(jsonCtx("UpdateBrokerCount", map[string]any{
		"clusterArn":                arn,
		"targetNumberOfBrokerNodes": float64(6),
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["clusterOperationArn"])
}

// ---- Test 7: GetBootstrapBrokers ----

func TestGetBootstrapBrokers(t *testing.T) {
	s := newService()
	arn := createCluster(t, s, "bootstrap-cluster")

	resp, err := s.HandleRequest(jsonCtx("GetBootstrapBrokers", map[string]any{"clusterArn": arn}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["bootstrapBrokerString"])
}

// ---- Test 8: ListNodes ----

func TestListNodes(t *testing.T) {
	s := newService()
	arn := createCluster(t, s, "nodes-cluster")

	resp, err := s.HandleRequest(jsonCtx("ListNodes", map[string]any{"clusterArn": arn}))
	require.NoError(t, err)
	body := respBody(t, resp)
	nodes := body["nodeInfoList"].([]any)
	assert.Len(t, nodes, 3)
}

// ---- Test 9: Configuration CRUD ----

func TestConfigurationCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateConfiguration", map[string]any{
		"name":             "test-config",
		"description":      "Test configuration",
		"kafkaVersion":     "3.5.1",
		"serverProperties": "auto.create.topics.enable=true",
	}))
	require.NoError(t, err)
	body := respBody(t, createResp)
	configArn := body["arn"].(string)

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeConfiguration", map[string]any{"arn": configArn}))
	require.NoError(t, err)
	descBody := respBody(t, descResp)
	assert.Equal(t, "test-config", descBody["name"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListConfigurations", map[string]any{}))
	require.NoError(t, err)
	listBody := respBody(t, listResp)
	configs := listBody["configurations"].([]any)
	assert.Len(t, configs, 1)

	// Update
	updResp, err := s.HandleRequest(jsonCtx("UpdateConfiguration", map[string]any{
		"arn":              configArn,
		"description":      "Updated",
		"serverProperties": "auto.create.topics.enable=false",
	}))
	require.NoError(t, err)
	updBody := respBody(t, updResp)
	rev := updBody["latestRevision"].(map[string]any)
	assert.Equal(t, float64(2), rev["revision"])

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteConfiguration", map[string]any{"arn": configArn}))
	require.NoError(t, err)
}

// ---- Test 10: Cluster NotFound ----

func TestClusterNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeCluster", map[string]any{
		"clusterArn": "arn:aws:kafka:us-east-1:123456789012:cluster/nonexistent/fake-uuid",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "NotFoundException", awsErr.Code)
}

// ---- Test 11: InvalidAction ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 12: Tagging ----

func TestTagging(t *testing.T) {
	s := newService()
	arn := createCluster(t, s, "tag-cluster")

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"resourceArn": arn,
		"tags":        map[string]any{"env": "test", "team": "data"},
	}))
	require.NoError(t, err)

	listResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	require.NoError(t, err)
	tags := respBody(t, listResp)["tags"].(map[string]any)
	assert.Len(t, tags, 2)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	listResp2, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	require.NoError(t, err)
	tags2 := respBody(t, listResp2)["tags"].(map[string]any)
	assert.Len(t, tags2, 1)
}

// ---- Test 13: Duplicate cluster ----

func TestDuplicateCluster(t *testing.T) {
	s := newService()
	createCluster(t, s, "dup-cluster")
	_, err := s.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"clusterName": "dup-cluster",
	}))
	require.Error(t, err)
}

// ---- Test 14: RebootBroker ----

func TestRebootBroker(t *testing.T) {
	s := newService()
	arn := createCluster(t, s, "reboot-cluster")

	resp, err := s.HandleRequest(jsonCtx("RebootBroker", map[string]any{
		"clusterArn": arn,
		"brokerIds":  []any{"1"},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["clusterOperationArn"])
}

// ---- Test 15: Service Name and HealthCheck ----

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "kafka", s.Name())
	assert.NoError(t, s.HealthCheck())
}
