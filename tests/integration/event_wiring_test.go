package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/eventbus"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
	"github.com/Viridian-Inc/cloudmock/pkg/integration"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	cwsvc "github.com/Viridian-Inc/cloudmock/services/cloudwatch"
	ebsvc "github.com/Viridian-Inc/cloudmock/services/eventbridge"
	lambdasvc "github.com/Viridian-Inc/cloudmock/services/lambda"
	s3svc "github.com/Viridian-Inc/cloudmock/services/s3"
	snssvc "github.com/Viridian-Inc/cloudmock/services/sns"
	sqssvc "github.com/Viridian-Inc/cloudmock/services/sqs"
)

// setupEventWiringStack creates a fully wired gateway with all services needed
// for event wiring tests (S3, SQS, SNS, EventBridge, Lambda, CloudWatch).
func setupEventWiringStack(t *testing.T) (*httptest.Server, *lambdasvc.LambdaService) {
	t.Helper()

	cfg := config.Default()
	cfg.IAM.Mode = "none"

	store := iampkg.NewStore(cfg.AccountID)
	_ = store.InitRoot("ROOTKEY", "ROOTSECRET")
	engine := iampkg.NewEngine()

	bus := eventbus.NewBus()
	registry := routing.NewRegistry()

	registry.Register(s3svc.NewWithBus(bus))
	registry.Register(sqssvc.New(cfg.AccountID, cfg.Region))

	snsService := snssvc.New(cfg.AccountID, cfg.Region)
	registry.Register(snsService)

	ebService := ebsvc.New(cfg.AccountID, cfg.Region)
	registry.Register(ebService)

	lambdaService := lambdasvc.New(cfg.AccountID, cfg.Region)
	registry.Register(lambdaService)

	cwService := cwsvc.New(cfg.AccountID, cfg.Region)
	registry.Register(cwService)

	// Wire locators.
	snsService.SetLocator(registry)
	ebService.SetLocator(registry)
	lambdaService.SetLocator(registry)
	cwService.SetLocator(registry)

	integration.WireIntegrations(bus, registry, cfg.AccountID, cfg.Region)

	gw := gateway.NewWithIAM(cfg, registry, store, engine)
	return httptest.NewServer(gw), lambdaService
}

// TestEventWiring_SNSToLambda verifies that publishing to an SNS topic with a
// Lambda subscription invokes the Lambda function with the SNS event payload.
func TestEventWiring_SNSToLambda(t *testing.T) {
	server, lambdaSvc := setupEventWiringStack(t)
	defer server.Close()

	// Step 1: Create a Lambda function (no code — returns mock response).
	funcReq := map[string]any{
		"FunctionName": "sns-handler",
		"Runtime":      "nodejs18.x",
		"Handler":      "index.handler",
		"Role":         "arn:aws:iam::000000000000:role/test",
		"Code": map[string]string{
			"ZipFile": "UEsDBBQAAAAAACdrdVwkks6nYAAAAGAAAAAIAAAAaW5kZXguanNleHBvcnRzLmhhbmRsZXIgPSBhc3luYyAoZXZlbnQpID0+IHsgcmV0dXJuIHsgc3RhdHVzQ29kZTogMjAwLCBib2R5OiBKU09OLnN0cmluZ2lmeShldmVudCkgfTsgfTtQSwECFAMUAAAAAAAna3VcJJLOp2AAAABgAAAACAAAAAAAAAAAAAAAgAEAAAAAaW5kZXguanNQSwUGAAAAAAEAAQA2AAAAhgAAAAAA", // minimal valid base64 (empty-ish zip)
		},
	}
	resp := doLambdaREST(t, server, http.MethodPost, "/2015-03-31/functions", funcReq)
	body := readBody(t, resp)
	// Function creation may succeed or return mock — either way the function exists.
	assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
		"CreateFunction: status=%d body=%s", resp.StatusCode, body)

	// Step 2: Create SNS topic.
	form := url.Values{"Name": {"lambda-test-topic"}}
	resp = doPost(t, server, "sns", "CreateTopic", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "CreateTopic failed: %s", body)

	topicArn := "arn:aws:sns:us-east-1:000000000000:lambda-test-topic"
	lambdaArn := "arn:aws:lambda:us-east-1:000000000000:function:sns-handler"

	// Step 3: Subscribe Lambda function to SNS topic.
	form = url.Values{
		"TopicArn": {topicArn},
		"Protocol": {"lambda"},
		"Endpoint": {lambdaArn},
	}
	resp = doPost(t, server, "sns", "Subscribe", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Subscribe failed: %s", body)

	// Step 4: Publish a message.
	form = url.Values{
		"TopicArn": {topicArn},
		"Message":  {"Hello Lambda from SNS!"},
		"Subject":  {"Test Subject"},
	}
	resp = doPost(t, server, "sns", "Publish", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Publish failed: %s", body)

	// Step 5: Check Lambda logs for invocation.
	time.Sleep(200 * time.Millisecond)
	logs := lambdaSvc.Logs().Recent("sns-handler", 20)
	// The function should have been invoked (START log entry).
	found := false
	for _, entry := range logs {
		if strings.Contains(entry.Message, "START RequestId") {
			found = true
			break
		}
	}
	assert.True(t, found, "Lambda function sns-handler should have been invoked via SNS; logs: %v", logs)
}

// TestEventWiring_EventBridgeToLambda verifies that PutEvents with a matching
// rule and Lambda target invokes the function.
func TestEventWiring_EventBridgeToLambda(t *testing.T) {
	server, lambdaSvc := setupEventWiringStack(t)
	defer server.Close()

	// Step 1: Create a Lambda function.
	funcReq := map[string]any{
		"FunctionName": "eb-handler",
		"Runtime":      "nodejs18.x",
		"Handler":      "index.handler",
		"Role":         "arn:aws:iam::000000000000:role/test",
		"Code": map[string]string{
			"ZipFile": "UEsDBBQAAAAAACdrdVwkks6nYAAAAGAAAAAIAAAAaW5kZXguanNleHBvcnRzLmhhbmRsZXIgPSBhc3luYyAoZXZlbnQpID0+IHsgcmV0dXJuIHsgc3RhdHVzQ29kZTogMjAwLCBib2R5OiBKU09OLnN0cmluZ2lmeShldmVudCkgfTsgfTtQSwECFAMUAAAAAAAna3VcJJLOp2AAAABgAAAACAAAAAAAAAAAAAAAgAEAAAAAaW5kZXguanNQSwUGAAAAAAEAAQA2AAAAhgAAAAAA",
		},
	}
	resp := doLambdaREST(t, server, http.MethodPost, "/2015-03-31/functions", funcReq)
	readBody(t, resp)

	// Step 2: Create EventBridge rule.
	putRuleReq := map[string]any{
		"Name":         "lambda-rule",
		"EventPattern": `{"source":["my.app"]}`,
		"State":        "ENABLED",
	}
	resp = doJSON(t, server, "events", "AWSEvents.PutRule", putRuleReq)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PutRule failed: %s", body)

	// Step 3: Add Lambda as target.
	putTargetsReq := map[string]any{
		"Rule": "lambda-rule",
		"Targets": []map[string]string{
			{
				"Id":  "lambda-target",
				"Arn": "arn:aws:lambda:us-east-1:000000000000:function:eb-handler",
			},
		},
	}
	resp = doJSON(t, server, "events", "AWSEvents.PutTargets", putTargetsReq)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PutTargets failed: %s", body)

	// Step 4: PutEvents.
	putEventsReq := map[string]any{
		"Entries": []map[string]any{
			{
				"Source":     "my.app",
				"DetailType": "TestEvent",
				"Detail":     `{"action":"test-eb-lambda"}`,
			},
		},
	}
	resp = doJSON(t, server, "events", "AWSEvents.PutEvents", putEventsReq)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PutEvents failed: %s", body)

	// Step 5: Check Lambda logs.
	time.Sleep(200 * time.Millisecond)
	logs := lambdaSvc.Logs().Recent("eb-handler", 20)
	found := false
	for _, entry := range logs {
		if strings.Contains(entry.Message, "START RequestId") {
			found = true
			break
		}
	}
	assert.True(t, found, "Lambda function eb-handler should have been invoked via EventBridge; logs: %v", logs)
}

// TestEventWiring_CloudWatchAlarmToSNS verifies that setting an alarm state to ALARM
// publishes a notification to the SNS topic listed in AlarmActions.
func TestEventWiring_CloudWatchAlarmToSNS(t *testing.T) {
	server, _ := setupEventWiringStack(t)
	defer server.Close()

	// Step 1: Create SQS queue (to receive SNS messages).
	form := url.Values{"QueueName": {"alarm-target-queue"}}
	resp := doPost(t, server, "sqs", "CreateQueue", form)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "CreateQueue failed: %s", body)

	// Step 2: Create SNS topic.
	form = url.Values{"Name": {"alarm-topic"}}
	resp = doPost(t, server, "sns", "CreateTopic", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "CreateTopic failed: %s", body)

	// Step 3: Subscribe SQS queue to SNS topic.
	topicArn := "arn:aws:sns:us-east-1:000000000000:alarm-topic"
	sqsArn := "arn:aws:sqs:us-east-1:000000000000:alarm-target-queue"
	form = url.Values{
		"TopicArn": {topicArn},
		"Protocol": {"sqs"},
		"Endpoint": {sqsArn},
	}
	resp = doPost(t, server, "sns", "Subscribe", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Subscribe failed: %s", body)

	// Step 4: Create CloudWatch alarm with SNS action.
	form = url.Values{
		"AlarmName":          {"test-alarm"},
		"Namespace":          {"AWS/EC2"},
		"MetricName":         {"CPUUtilization"},
		"ComparisonOperator": {"GreaterThanThreshold"},
		"Threshold":          {"80"},
		"EvaluationPeriods":  {"1"},
		"Period":             {"300"},
		"Statistic":          {"Average"},
		"AlarmActions.member.1": {topicArn},
	}
	resp = doPost(t, server, "monitoring", "PutMetricAlarm", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "PutMetricAlarm failed: %s", body)

	// Step 5: Set alarm state to ALARM.
	form = url.Values{
		"AlarmName":   {"test-alarm"},
		"StateValue":  {"ALARM"},
		"StateReason": {"Test alarm trigger"},
	}
	resp = doPost(t, server, "monitoring", "SetAlarmState", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "SetAlarmState failed: %s", body)

	// Step 6: Receive message from SQS queue.
	form = url.Values{
		"QueueUrl":            {"http://sqs.us-east-1.localhost:4566/000000000000/alarm-target-queue"},
		"MaxNumberOfMessages": {"1"},
	}
	resp = doPost(t, server, "sqs", "ReceiveMessage", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify the message contains alarm notification data.
	assert.Contains(t, body, "test-alarm", "SQS message should contain the alarm name")
	assert.Contains(t, body, "ALARM", "SQS message should contain the new state value")
}

// doLambdaREST sends a JSON request to a Lambda REST endpoint.
func doLambdaREST(t *testing.T, server *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()

	var bodyReader *strings.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = strings.NewReader(string(data))
	} else {
		bodyReader = strings.NewReader("")
	}

	req, err := http.NewRequest(method, server.URL+path, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20240101/us-east-1/lambda/aws4_request, SignedHeaders=host;x-amz-date, Signature=deadbeef")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}
