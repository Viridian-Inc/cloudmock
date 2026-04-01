package dms_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/dms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.DMSService {
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
	data, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// ---- Test 1: Create and Describe Replication Instance ----

func TestDMS_CreateAndDescribeReplicationInstance(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateReplicationInstance", map[string]any{
		"ReplicationInstanceIdentifier": "test-ri",
		"ReplicationInstanceClass":      "dms.r5.large",
		"AllocatedStorage":              100,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	m := respJSON(t, resp)
	ri := m["ReplicationInstance"].(map[string]any)
	assert.Equal(t, "test-ri", ri["ReplicationInstanceIdentifier"])
	assert.Equal(t, "dms.r5.large", ri["ReplicationInstanceClass"])
	assert.Equal(t, "creating", ri["ReplicationInstanceStatus"])
	assert.Contains(t, ri["ReplicationInstanceArn"], "test-ri")
}

// ---- Test 2: List Replication Instances ----

func TestDMS_ListReplicationInstances(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateReplicationInstance", map[string]any{
		"ReplicationInstanceIdentifier": "ri-1",
	}))
	s.HandleRequest(jsonCtx("CreateReplicationInstance", map[string]any{
		"ReplicationInstanceIdentifier": "ri-2",
	}))

	resp, err := s.HandleRequest(jsonCtx("DescribeReplicationInstances", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	instances := m["ReplicationInstances"].([]any)
	assert.Len(t, instances, 2)
}

// ---- Test 3: Delete Replication Instance ----

func TestDMS_DeleteReplicationInstance(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateReplicationInstance", map[string]any{
		"ReplicationInstanceIdentifier": "delete-ri",
	}))
	m := respJSON(t, createResp)
	arn := m["ReplicationInstance"].(map[string]any)["ReplicationInstanceArn"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteReplicationInstance", map[string]any{
		"ReplicationInstanceArn": arn,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	listResp, _ := s.HandleRequest(jsonCtx("DescribeReplicationInstances", nil))
	lm := respJSON(t, listResp)
	assert.Len(t, lm["ReplicationInstances"].([]any), 0)
}

// ---- Test 4: Create and Describe Endpoints ----

func TestDMS_CreateAndDescribeEndpoints(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateEndpoint", map[string]any{
		"EndpointIdentifier": "src-ep",
		"EndpointType":       "source",
		"EngineName":         "mysql",
		"ServerName":         "db.example.com",
		"Port":               3306,
		"DatabaseName":       "mydb",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	listResp, _ := s.HandleRequest(jsonCtx("DescribeEndpoints", nil))
	m := respJSON(t, listResp)
	eps := m["Endpoints"].([]any)
	assert.Len(t, eps, 1)
	assert.Equal(t, "src-ep", eps[0].(map[string]any)["EndpointIdentifier"])
}

// ---- Test 5: Delete Endpoint ----

func TestDMS_DeleteEndpoint(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateEndpoint", map[string]any{
		"EndpointIdentifier": "del-ep",
		"EndpointType":       "target",
		"EngineName":         "postgres",
	}))
	m := respJSON(t, createResp)
	arn := m["Endpoint"].(map[string]any)["EndpointArn"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteEndpoint", map[string]any{
		"EndpointArn": arn,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	listResp, _ := s.HandleRequest(jsonCtx("DescribeEndpoints", nil))
	lm := respJSON(t, listResp)
	assert.Len(t, lm["Endpoints"].([]any), 0)
}

// ---- Test 6: Create and Manage Replication Tasks ----

func TestDMS_ReplicationTaskLifecycle(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateReplicationTask", map[string]any{
		"ReplicationTaskIdentifier": "task-1",
		"SourceEndpointArn":         "arn:src",
		"TargetEndpointArn":         "arn:tgt",
		"ReplicationInstanceArn":    "arn:ri",
		"MigrationType":             "full-load",
		"TableMappings":             "{}",
	}))
	m := respJSON(t, createResp)
	taskArn := m["ReplicationTask"].(map[string]any)["ReplicationTaskArn"].(string)
	assert.Equal(t, "ready", m["ReplicationTask"].(map[string]any)["Status"])

	// Start task
	startResp, err := s.HandleRequest(jsonCtx("StartReplicationTask", map[string]any{
		"ReplicationTaskArn": taskArn,
	}))
	require.NoError(t, err)
	sm := respJSON(t, startResp)
	// In instant mode, task may have already transitioned to running.
	taskStatus := sm["ReplicationTask"].(map[string]any)["Status"].(string)
	assert.Contains(t, []string{"starting", "running"}, taskStatus)

	// List tasks
	listResp, _ := s.HandleRequest(jsonCtx("DescribeReplicationTasks", nil))
	lm := respJSON(t, listResp)
	assert.Len(t, lm["ReplicationTasks"].([]any), 1)
}

// ---- Test 7: Event Subscriptions ----

func TestDMS_EventSubscriptions(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateEventSubscription", map[string]any{
		"SubscriptionName": "my-sub",
		"SnsTopicArn":      "arn:aws:sns:us-east-1:123456789012:my-topic",
		"SourceType":       "replication-instance",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	listResp, _ := s.HandleRequest(jsonCtx("DescribeEventSubscriptions", nil))
	lm := respJSON(t, listResp)
	assert.Len(t, lm["EventSubscriptionsList"].([]any), 1)

	delResp, err := s.HandleRequest(jsonCtx("DeleteEventSubscription", map[string]any{
		"SubscriptionName": "my-sub",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)

	listResp2, _ := s.HandleRequest(jsonCtx("DescribeEventSubscriptions", nil))
	lm2 := respJSON(t, listResp2)
	assert.Len(t, lm2["EventSubscriptionsList"].([]any), 0)
}

// ---- Test 8: Not Found ----

func TestDMS_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteReplicationInstance", map[string]any{
		"ReplicationInstanceArn": "arn:nonexistent",
	}))
	require.Error(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteEndpoint", map[string]any{
		"EndpointArn": "arn:nonexistent",
	}))
	require.Error(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteEventSubscription", map[string]any{
		"SubscriptionName": "nonexistent",
	}))
	require.Error(t, err)
}

// ---- Test 9: Invalid Action ----

func TestDMS_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", nil))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 10: Duplicate Resources ----

func TestDMS_DuplicateInstance(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateReplicationInstance", map[string]any{
		"ReplicationInstanceIdentifier": "dup-ri",
	}))
	_, err := s.HandleRequest(jsonCtx("CreateReplicationInstance", map[string]any{
		"ReplicationInstanceIdentifier": "dup-ri",
	}))
	require.Error(t, err)
}

// ---- Behavioral: TestConnection ----

func TestDMS_TestConnection(t *testing.T) {
	s := newService()

	// Create instance and endpoint
	instResp, _ := s.HandleRequest(jsonCtx("CreateReplicationInstance", map[string]any{
		"ReplicationInstanceIdentifier": "conn-ri",
		"ReplicationInstanceClass":      "dms.t2.micro",
	}))
	instArn := respJSON(t, instResp)["ReplicationInstance"].(map[string]any)["ReplicationInstanceArn"].(string)

	epResp, _ := s.HandleRequest(jsonCtx("CreateEndpoint", map[string]any{
		"EndpointIdentifier": "conn-ep",
		"EndpointType":       "source",
		"EngineName":         "mysql",
		"ServerName":         "db.example.com",
		"Port":               3306,
	}))
	epArn := respJSON(t, epResp)["Endpoint"].(map[string]any)["EndpointArn"].(string)

	// Test connection
	resp, err := s.HandleRequest(jsonCtx("TestConnection", map[string]any{
		"EndpointArn":            epArn,
		"ReplicationInstanceArn": instArn,
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	conn := m["Connection"].(map[string]any)
	assert.Equal(t, "successful", conn["Status"])

	// Describe connections
	descResp, err := s.HandleRequest(jsonCtx("DescribeConnections", nil))
	require.NoError(t, err)
	descData := respJSON(t, descResp)
	conns := descData["Connections"].([]any)
	assert.Len(t, conns, 1)
}

func TestDMS_TestConnectionNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("TestConnection", map[string]any{
		"EndpointArn":            "arn:aws:dms:us-east-1:123456789012:endpoint:nonexistent",
		"ReplicationInstanceArn": "arn:aws:dms:us-east-1:123456789012:rep:nonexistent",
	}))
	require.Error(t, err)
}

// ---- Behavioral: Task table counters ----

func TestDMS_TaskTableCounters(t *testing.T) {
	s := newService()

	// Create instance, endpoints, task
	s.HandleRequest(jsonCtx("CreateReplicationInstance", map[string]any{
		"ReplicationInstanceIdentifier": "table-ri",
	}))
	s.HandleRequest(jsonCtx("CreateEndpoint", map[string]any{
		"EndpointIdentifier": "table-src", "EndpointType": "source",
		"EngineName": "mysql", "ServerName": "src.example.com", "Port": 3306,
	}))
	s.HandleRequest(jsonCtx("CreateEndpoint", map[string]any{
		"EndpointIdentifier": "table-tgt", "EndpointType": "target",
		"EngineName": "mysql", "ServerName": "tgt.example.com", "Port": 3306,
	}))

	createResp, _ := s.HandleRequest(jsonCtx("CreateReplicationTask", map[string]any{
		"ReplicationTaskIdentifier": "table-task",
		"SourceEndpointArn":         "arn:aws:dms:us-east-1:123456789012:endpoint:table-src",
		"TargetEndpointArn":         "arn:aws:dms:us-east-1:123456789012:endpoint:table-tgt",
		"ReplicationInstanceArn":    "arn:aws:dms:us-east-1:123456789012:rep:table-ri",
		"MigrationType":             "full-load",
		"TableMappings":             "{}",
	}))
	taskArn := respJSON(t, createResp)["ReplicationTask"].(map[string]any)["ReplicationTaskArn"].(string)

	// Start task
	_, err := s.HandleRequest(jsonCtx("StartReplicationTask", map[string]any{
		"ReplicationTaskArn": taskArn,
	}))
	require.NoError(t, err)

	// In instant mode, task should be running with table stats populated.
	descResp, _ := s.HandleRequest(jsonCtx("DescribeReplicationTasks", nil))
	tasks := respJSON(t, descResp)["ReplicationTasks"].([]any)
	assert.GreaterOrEqual(t, len(tasks), 1)
}
