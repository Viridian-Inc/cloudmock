package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/gateway"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/integration"
	"github.com/neureaux/cloudmock/pkg/routing"
	ebsvc "github.com/neureaux/cloudmock/services/eventbridge"
	s3svc "github.com/neureaux/cloudmock/services/s3"
	snssvc "github.com/neureaux/cloudmock/services/sns"
	sqssvc "github.com/neureaux/cloudmock/services/sqs"
)

// setupCrossServiceStack creates a fully wired gateway with S3, SQS, SNS,
// and EventBridge services, plus the event bus and cross-service integrations.
func setupCrossServiceStack(t *testing.T) *httptest.Server {
	t.Helper()

	cfg := config.Default()
	cfg.IAM.Mode = "none" // skip IAM for integration tests

	store := iampkg.NewStore(cfg.AccountID)
	_ = store.InitRoot("ROOTKEY", "ROOTSECRET")
	engine := iampkg.NewEngine()

	bus := eventbus.NewBus()
	registry := routing.NewRegistry()

	// S3 with event bus
	registry.Register(s3svc.NewWithBus(bus))

	// SQS
	registry.Register(sqssvc.New(cfg.AccountID, cfg.Region))

	// SNS with locator (set after registry is populated)
	snsService := snssvc.New(cfg.AccountID, cfg.Region)
	registry.Register(snsService)

	// EventBridge with locator
	ebService := ebsvc.New(cfg.AccountID, cfg.Region)
	registry.Register(ebService)

	// Wire locators
	snsService.SetLocator(registry)
	ebService.SetLocator(registry)

	// Wire event bus integrations
	integration.WireIntegrations(bus, registry, cfg.AccountID, cfg.Region)

	gw := gateway.NewWithIAM(cfg, registry, store, engine)
	return httptest.NewServer(gw)
}

// doPost sends a form-encoded POST to the given server with the specified
// action and form values.
func doPost(t *testing.T, server *httptest.Server, svcName, action string, form url.Values) *http.Response {
	t.Helper()
	form.Set("Action", action)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Set the Authorization header so the gateway can route to the correct service.
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20240101/us-east-1/"+svcName+"/aws4_request, SignedHeaders=host;x-amz-date, Signature=deadbeef")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// doJSON sends a JSON POST with X-Amz-Target header (for JSON protocol services).
func doJSON(t *testing.T, server *httptest.Server, svcName, target string, body any) *http.Response {
	t.Helper()

	data, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/", strings.NewReader(string(data)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", target)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20240101/us-east-1/"+svcName+"/aws4_request, SignedHeaders=host;x-amz-date, Signature=deadbeef")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// readBody reads and closes a response body.
func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(data)
}

// TestCrossService_S3ToSQS verifies that putting an object in S3 delivers
// an event notification to an SQS queue named "s3-events-{bucket}".
func TestCrossService_S3ToSQS(t *testing.T) {
	server := setupCrossServiceStack(t)
	defer server.Close()

	// Step 1: Create S3 bucket "test-bucket"
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/test-bucket", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20240101/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-date, Signature=deadbeef")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 2: Create SQS queue "s3-events-test-bucket"
	form := url.Values{}
	resp = doPost(t, server, "sqs", "CreateQueue", form)
	// Need to re-create with proper form
	form = url.Values{"QueueName": {"s3-events-test-bucket"}}
	resp = doPost(t, server, "sqs", "CreateQueue", form)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "CreateQueue failed: %s", body)

	// Step 3: Put an object in S3
	objReq, _ := http.NewRequest(http.MethodPut, server.URL+"/test-bucket/hello.txt",
		strings.NewReader("Hello, World!"))
	objReq.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20240101/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-date, Signature=deadbeef")
	objReq.Header.Set("Content-Type", "text/plain")
	resp, err = http.DefaultClient.Do(objReq)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 4: Wait briefly for async event delivery
	time.Sleep(200 * time.Millisecond)

	// Step 5: Receive message from SQS
	form = url.Values{
		"QueueUrl":            {"http://sqs.us-east-1.localhost:4566/000000000000/s3-events-test-bucket"},
		"MaxNumberOfMessages": {"1"},
	}
	resp = doPost(t, server, "sqs", "ReceiveMessage", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify the message contains S3 event data
	assert.Contains(t, body, "test-bucket", "SQS message should reference the bucket")
	assert.Contains(t, body, "hello.txt", "SQS message should reference the object key")
	assert.Contains(t, body, "s3:ObjectCreated:Put", "SQS message should contain the event name")
}

// TestCrossService_SNSToSQS verifies that publishing to an SNS topic with an
// SQS subscription delivers the message to the SQS queue.
func TestCrossService_SNSToSQS(t *testing.T) {
	server := setupCrossServiceStack(t)
	defer server.Close()

	// Step 1: Create SQS queue
	form := url.Values{"QueueName": {"sns-target-queue"}}
	resp := doPost(t, server, "sqs", "CreateQueue", form)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "CreateQueue failed: %s", body)

	// Step 2: Create SNS topic
	form = url.Values{"Name": {"test-topic"}}
	resp = doPost(t, server, "sns", "CreateTopic", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "CreateTopic failed: %s", body)

	// Extract topic ARN
	topicArn := "arn:aws:sns:us-east-1:000000000000:test-topic"

	// Step 3: Subscribe SQS queue to SNS topic
	sqsArn := "arn:aws:sqs:us-east-1:000000000000:sns-target-queue"
	form = url.Values{
		"TopicArn": {topicArn},
		"Protocol": {"sqs"},
		"Endpoint": {sqsArn},
	}
	resp = doPost(t, server, "sns", "Subscribe", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Subscribe failed: %s", body)

	// Step 4: Publish a message to the SNS topic
	form = url.Values{
		"TopicArn": {topicArn},
		"Message":  {"Hello from SNS!"},
		"Subject":  {"Test Subject"},
	}
	resp = doPost(t, server, "sns", "Publish", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Publish failed: %s", body)

	// Step 5: Receive message from SQS queue
	form = url.Values{
		"QueueUrl":            {"http://sqs.us-east-1.localhost:4566/000000000000/sns-target-queue"},
		"MaxNumberOfMessages": {"1"},
	}
	resp = doPost(t, server, "sqs", "ReceiveMessage", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify the message contains the SNS notification
	assert.Contains(t, body, "Hello from SNS!", "SQS message should contain the SNS message")
	assert.Contains(t, body, topicArn, "SQS message should reference the topic ARN")
}

// TestCrossService_EventBridgeToSQS verifies that PutEvents with a matching
// rule delivers events to an SQS queue target.
func TestCrossService_EventBridgeToSQS(t *testing.T) {
	server := setupCrossServiceStack(t)
	defer server.Close()

	// Step 1: Create SQS queue
	form := url.Values{"QueueName": {"eb-target-queue"}}
	resp := doPost(t, server, "sqs", "CreateQueue", form)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "CreateQueue failed: %s", body)

	// Step 2: Create EventBridge rule matching source "custom.app"
	putRuleReq := map[string]any{
		"Name":         "test-rule",
		"EventPattern": `{"source":["custom.app"]}`,
		"State":        "ENABLED",
	}
	resp = doJSON(t, server, "events", "AWSEvents.PutRule", putRuleReq)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PutRule failed: %s", body)

	// Step 3: Add SQS queue as a target
	putTargetsReq := map[string]any{
		"Rule": "test-rule",
		"Targets": []map[string]string{
			{
				"Id":  "sqs-target",
				"Arn": "arn:aws:sqs:us-east-1:000000000000:eb-target-queue",
			},
		},
	}
	resp = doJSON(t, server, "events", "AWSEvents.PutTargets", putTargetsReq)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PutTargets failed: %s", body)

	// Step 4: PutEvents with source "custom.app"
	putEventsReq := map[string]any{
		"Entries": []map[string]any{
			{
				"Source":     "custom.app",
				"DetailType": "MyEvent",
				"Detail":     `{"key":"value","action":"test"}`,
			},
		},
	}
	resp = doJSON(t, server, "events", "AWSEvents.PutEvents", putEventsReq)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PutEvents failed: %s", body)

	// Step 5: Receive message from SQS queue
	form = url.Values{
		"QueueUrl":            {"http://sqs.us-east-1.localhost:4566/000000000000/eb-target-queue"},
		"MaxNumberOfMessages": {"1"},
	}
	resp = doPost(t, server, "sqs", "ReceiveMessage", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify the message contains the event data
	assert.Contains(t, body, "custom.app", "SQS message should contain the event source")
	assert.Contains(t, body, "MyEvent", "SQS message should contain the detail type")
	assert.Contains(t, body, "test", "SQS message should contain the event detail")
}

// TestCrossService_EventBridgeNoMatch verifies that events not matching a rule
// are not delivered to targets.
func TestCrossService_EventBridgeNoMatch(t *testing.T) {
	server := setupCrossServiceStack(t)
	defer server.Close()

	// Create SQS queue
	form := url.Values{"QueueName": {"eb-nomatch-queue"}}
	resp := doPost(t, server, "sqs", "CreateQueue", form)
	readBody(t, resp)

	// Create rule matching source "specific.app"
	putRuleReq := map[string]any{
		"Name":         "specific-rule",
		"EventPattern": `{"source":["specific.app"]}`,
		"State":        "ENABLED",
	}
	resp = doJSON(t, server, "events", "AWSEvents.PutRule", putRuleReq)
	readBody(t, resp)

	// Add target
	putTargetsReq := map[string]any{
		"Rule": "specific-rule",
		"Targets": []map[string]string{
			{
				"Id":  "sqs-target",
				"Arn": "arn:aws:sqs:us-east-1:000000000000:eb-nomatch-queue",
			},
		},
	}
	resp = doJSON(t, server, "events", "AWSEvents.PutTargets", putTargetsReq)
	readBody(t, resp)

	// PutEvents with a DIFFERENT source
	putEventsReq := map[string]any{
		"Entries": []map[string]any{
			{
				"Source":     "other.app",
				"DetailType": "SomeEvent",
				"Detail":     `{"key":"value"}`,
			},
		},
	}
	resp = doJSON(t, server, "events", "AWSEvents.PutEvents", putEventsReq)
	readBody(t, resp)

	// Verify no message in SQS
	form = url.Values{
		"QueueUrl":            {"http://sqs.us-east-1.localhost:4566/000000000000/eb-nomatch-queue"},
		"MaxNumberOfMessages": {"1"},
	}
	resp = doPost(t, server, "sqs", "ReceiveMessage", form)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// The response should have no messages (empty result).
	assert.NotContains(t, body, "other.app", "No message should be delivered for non-matching source")
}
