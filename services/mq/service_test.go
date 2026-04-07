package mq_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/mq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.MQService {
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

func createBroker(t *testing.T, s *svc.MQService, name string) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateBroker", map[string]any{
		"brokerName": name,
		"engineType": "ACTIVEMQ",
		"users": []any{
			map[string]any{"username": "admin", "consoleAccess": true},
		},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	return body["brokerId"].(string)
}

func createConfiguration(t *testing.T, s *svc.MQService, name string) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateConfiguration", map[string]any{
		"name":          name,
		"engineType":    "ACTIVEMQ",
		"engineVersion": "5.17.6",
	}))
	require.NoError(t, err)
	return respBody(t, resp)["id"].(string)
}

// ---- Test 1: CreateBroker ----

func TestCreateBroker(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateBroker", map[string]any{
		"brokerName": "test-broker",
		"engineType": "ACTIVEMQ",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["brokerId"])
	assert.NotEmpty(t, body["brokerArn"])
}

// ---- Test 2: DescribeBroker ----

func TestDescribeBroker(t *testing.T) {
	s := newService()
	brokerId := createBroker(t, s, "desc-broker")

	resp, err := s.HandleRequest(jsonCtx("DescribeBroker", map[string]any{"brokerId": brokerId}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "desc-broker", body["brokerName"])
	assert.Equal(t, "ACTIVEMQ", body["engineType"])
}

// ---- Test 3: ListBrokers ----

func TestListBrokers(t *testing.T) {
	s := newService()
	createBroker(t, s, "broker-1")
	createBroker(t, s, "broker-2")

	resp, err := s.HandleRequest(jsonCtx("ListBrokers", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	summaries := body["brokerSummaries"].([]any)
	assert.Len(t, summaries, 2)
}

// ---- Test 4: DeleteBroker ----

func TestDeleteBroker(t *testing.T) {
	s := newService()
	brokerId := createBroker(t, s, "del-broker")

	_, err := s.HandleRequest(jsonCtx("DeleteBroker", map[string]any{"brokerId": brokerId}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DescribeBroker", map[string]any{"brokerId": brokerId}))
	require.Error(t, err)
}

// ---- Test 5: UpdateBroker ----

func TestUpdateBroker(t *testing.T) {
	s := newService()
	brokerId := createBroker(t, s, "upd-broker")

	resp, err := s.HandleRequest(jsonCtx("UpdateBroker", map[string]any{
		"brokerId":         brokerId,
		"hostInstanceType": "mq.m5.xlarge",
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, brokerId, body["brokerId"])

	descResp, err := s.HandleRequest(jsonCtx("DescribeBroker", map[string]any{"brokerId": brokerId}))
	require.NoError(t, err)
	descBody := respBody(t, descResp)
	assert.Equal(t, "mq.m5.xlarge", descBody["hostInstanceType"])
}

// ---- Test 6: Broker lifecycle CREATION_IN_PROGRESS -> RUNNING ----

func TestBrokerLifecycle(t *testing.T) {
	s := newService()
	brokerId := createBroker(t, s, "lc-broker")

	resp, err := s.HandleRequest(jsonCtx("DescribeBroker", map[string]any{"brokerId": brokerId}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Contains(t, []string{"CREATION_IN_PROGRESS", "RUNNING"}, body["brokerState"])

	time.Sleep(3 * time.Second)
	resp2, err := s.HandleRequest(jsonCtx("DescribeBroker", map[string]any{"brokerId": brokerId}))
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	assert.Equal(t, "RUNNING", body2["brokerState"])
}

// ---- Test 7: RebootBroker ----

func TestRebootBroker(t *testing.T) {
	s := newService()
	brokerId := createBroker(t, s, "reboot-broker")

	_, err := s.HandleRequest(jsonCtx("RebootBroker", map[string]any{"brokerId": brokerId}))
	require.NoError(t, err)
}

// ---- Test 8: Configuration CRUD ----

func TestConfigurationCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateConfiguration", map[string]any{
		"name":          "test-config",
		"description":   "Test MQ config",
		"engineType":    "ACTIVEMQ",
		"engineVersion": "5.17.6",
	}))
	require.NoError(t, err)
	body := respBody(t, createResp)
	configId := body["id"].(string)

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeConfiguration", map[string]any{"configurationId": configId}))
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
		"configurationId": configId,
		"description":     "Updated config",
	}))
	require.NoError(t, err)
	updBody := respBody(t, updResp)
	rev := updBody["latestRevision"].(map[string]any)
	assert.Equal(t, float64(2), rev["revisionId"])
}

// ---- Test 9: User CRUD ----

func TestUserCRUD(t *testing.T) {
	s := newService()
	brokerId := createBroker(t, s, "user-broker")

	// Create user
	_, err := s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"brokerId":      brokerId,
		"username":      "testuser",
		"consoleAccess": true,
		"groups":        []any{"admins"},
	}))
	require.NoError(t, err)

	// Describe user
	descResp, err := s.HandleRequest(jsonCtx("DescribeUser", map[string]any{
		"brokerId": brokerId,
		"username": "testuser",
	}))
	require.NoError(t, err)
	descBody := respBody(t, descResp)
	assert.Equal(t, "testuser", descBody["username"])
	assert.Equal(t, true, descBody["consoleAccess"])

	// List users
	listResp, err := s.HandleRequest(jsonCtx("ListUsers", map[string]any{"brokerId": brokerId}))
	require.NoError(t, err)
	listBody := respBody(t, listResp)
	users := listBody["users"].([]any)
	assert.GreaterOrEqual(t, len(users), 1)

	// Update user
	_, err = s.HandleRequest(jsonCtx("UpdateUser", map[string]any{
		"brokerId":      brokerId,
		"username":      "testuser",
		"consoleAccess": false,
	}))
	require.NoError(t, err)

	// Delete user
	_, err = s.HandleRequest(jsonCtx("DeleteUser", map[string]any{
		"brokerId": brokerId,
		"username": "testuser",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DescribeUser", map[string]any{
		"brokerId": brokerId,
		"username": "testuser",
	}))
	require.Error(t, err)
}

// ---- Test 10: Broker NotFound ----

func TestBrokerNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeBroker", map[string]any{
		"brokerId": "nonexistent-id",
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
	resp, err := s.HandleRequest(jsonCtx("CreateBroker", map[string]any{
		"brokerName": "tag-broker",
		"tags":       map[string]any{"env": "test"},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	brokerArn := body["brokerArn"].(string)

	_, err = s.HandleRequest(jsonCtx("CreateTags", map[string]any{
		"resourceArn": brokerArn,
		"tags":        map[string]any{"team": "data"},
	}))
	require.NoError(t, err)

	listResp, err := s.HandleRequest(jsonCtx("ListTags", map[string]any{"resourceArn": brokerArn}))
	require.NoError(t, err)
	tags := respBody(t, listResp)["tags"].(map[string]any)
	assert.Len(t, tags, 2)

	_, err = s.HandleRequest(jsonCtx("DeleteTags", map[string]any{
		"resourceArn": brokerArn,
		"tagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	listResp2, err := s.HandleRequest(jsonCtx("ListTags", map[string]any{"resourceArn": brokerArn}))
	require.NoError(t, err)
	tags2 := respBody(t, listResp2)["tags"].(map[string]any)
	assert.Len(t, tags2, 1)
}

// ---- Test 13: Service Name and HealthCheck ----

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "mq", s.Name())
	assert.NoError(t, s.HealthCheck())
}

// ---- Test 14: DescribeConfigurationRevision ----

func TestDescribeConfigurationRevision(t *testing.T) {
	s := newService()
	configId := createConfiguration(t, s, "rev-config")

	resp, err := s.HandleRequest(jsonCtx("DescribeConfigurationRevision", map[string]any{
		"configurationId":       configId,
		"configurationRevision": float64(1),
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, configId, body["configurationId"])
	assert.Equal(t, float64(1), body["configurationRevision"])
}

// ---- Test 15: ListConfigurationRevisions ----

func TestListConfigurationRevisions(t *testing.T) {
	s := newService()
	configId := createConfiguration(t, s, "list-rev-config")

	// Update to create a second revision
	_, err := s.HandleRequest(jsonCtx("UpdateConfiguration", map[string]any{
		"configurationId": configId,
		"description":     "second revision",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListConfigurationRevisions", map[string]any{
		"configurationId": configId,
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	revisions := body["revisions"].([]any)
	assert.Len(t, revisions, 2)
	assert.Equal(t, float64(2), body["maxResults"])
}

// ---- Test 16: CreateBroker missing name ----

func TestCreateBrokerMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateBroker", map[string]any{"engineType": "ACTIVEMQ"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
}

// ---- Test 17: RabbitMQ engine type ----

func TestCreateRabbitMQBroker(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateBroker", map[string]any{
		"brokerName": "rabbit-broker",
		"engineType": "RABBITMQ",
		"deploymentMode": "SINGLE_INSTANCE",
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["brokerId"])

	descResp, _ := s.HandleRequest(jsonCtx("DescribeBroker", map[string]any{
		"brokerId": body["brokerId"],
	}))
	descBody := respBody(t, descResp)
	assert.Equal(t, "RABBITMQ", descBody["engineType"])
}

// ---- Test 18: Create duplicate user ----

func TestCreateDuplicateUser(t *testing.T) {
	s := newService()
	brokerId := createBroker(t, s, "dup-user-broker")

	_, err := s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"brokerId": brokerId,
		"username": "alice",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"brokerId": brokerId,
		"username": "alice",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ConflictException", awsErr.Code)
}

// ---- Test 19: Configuration not found ----

func TestConfigurationNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeConfiguration", map[string]any{
		"configurationId": "nonexistent-config",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "NotFoundException", awsErr.Code)
}

// ---- Test 20: Broker ARN format ----

func TestBrokerARNFormat(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateBroker", map[string]any{
		"brokerName": "arn-broker",
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	arn := body["brokerArn"].(string)
	assert.Contains(t, arn, "arn:aws:mq:")
	assert.Contains(t, arn, "broker")
}
