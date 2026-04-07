package integration_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
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
	cfnsvc "github.com/Viridian-Inc/cloudmock/services/cloudformation"
	ebsvc "github.com/Viridian-Inc/cloudmock/services/eventbridge"
	lambdasvc "github.com/Viridian-Inc/cloudmock/services/lambda"
	s3svc "github.com/Viridian-Inc/cloudmock/services/s3"
	snssvc "github.com/Viridian-Inc/cloudmock/services/sns"
	sqssvc "github.com/Viridian-Inc/cloudmock/services/sqs"
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

// ---- Lambda invocation integration test ----

// nodeAvailable returns true if node is installed.
func nodeAvailable() bool {
	_, err := exec.LookPath("node")
	return err == nil
}

// createZipFile creates a zip archive in memory with the given filename and content.
func createZipFile(t *testing.T, filename, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(filename)
	require.NoError(t, err)
	_, err = f.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// setupLambdaStack creates a gateway with Lambda registered (for Lambda invocation tests).
func setupLambdaStack(t *testing.T) *httptest.Server {
	t.Helper()

	cfg := config.Default()
	cfg.IAM.Mode = "none"

	store := iampkg.NewStore(cfg.AccountID)
	_ = store.InitRoot("ROOTKEY", "ROOTSECRET")
	engine := iampkg.NewEngine()

	registry := routing.NewRegistry()

	// Lambda
	registry.Register(lambdasvc.New(cfg.AccountID, cfg.Region))

	gw := gateway.NewWithIAM(cfg, registry, store, engine)
	return httptest.NewServer(gw)
}

// doLambdaReq sends a REST request to the Lambda service.
func doLambdaReq(t *testing.T, server *httptest.Server, method, path, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20240101/us-east-1/lambda/aws4_request, SignedHeaders=host;x-amz-date, Signature=deadbeef")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// TestCrossService_LambdaInvocation verifies that a Lambda function can be
// created and invoked through the gateway, returning a correct response.
func TestCrossService_LambdaInvocation(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("node not available, skipping Lambda invocation integration test")
	}

	server := setupLambdaStack(t)
	defer server.Close()

	// Step 1: Create a Lambda function
	nodeCode := createZipFile(t, "index.js",
		`exports.handler = async (event) => {
			return {
				statusCode: 200,
				body: JSON.stringify({ message: "Hello from Lambda", input: event })
			};
		};`)
	b64Code := base64.StdEncoding.EncodeToString(nodeCode)

	createBody := `{
		"FunctionName": "integration-test-func",
		"Runtime": "nodejs20.x",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "index.handler",
		"Code": {"ZipFile": "` + b64Code + `"},
		"Description": "Integration test function",
		"Timeout": 10,
		"MemorySize": 128
	}`

	resp := doLambdaReq(t, server, http.MethodPost, "/2015-03-31/functions", createBody)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "CreateFunction should return 201: %s", body)

	var createResp map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &createResp))
	assert.Equal(t, "integration-test-func", createResp["FunctionName"])
	assert.Contains(t, createResp["FunctionArn"].(string), "integration-test-func")

	// Step 2: Invoke the Lambda function with a payload
	invokePayload := `{"key": "value", "action": "test"}`
	resp = doLambdaReq(t, server, http.MethodPost,
		"/2015-03-31/functions/integration-test-func/invocations", invokePayload)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Invoke should return 200: %s", body)

	// Step 3: Verify the response contains our input
	var invokeResp map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &invokeResp))
	assert.Equal(t, float64(200), invokeResp["statusCode"], "Lambda should return statusCode 200")

	// Parse the body field
	bodyStr, ok := invokeResp["body"].(string)
	require.True(t, ok, "body should be a string")
	var bodyParsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodyStr), &bodyParsed))
	assert.Equal(t, "Hello from Lambda", bodyParsed["message"])

	// The input should be echoed back
	input, ok := bodyParsed["input"].(map[string]any)
	require.True(t, ok, "input should be a map")
	assert.Equal(t, "value", input["key"])
	assert.Equal(t, "test", input["action"])

	// Step 4: Verify the function is listed
	resp = doLambdaReq(t, server, http.MethodGet, "/2015-03-31/functions", "")
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &listResp))
	functions := listResp["Functions"].([]any)
	assert.Len(t, functions, 1)
	assert.Equal(t, "integration-test-func", functions[0].(map[string]any)["FunctionName"])
}

// ---- CloudFormation provisioning integration test ----

// setupCFNStack creates a gateway with CloudFormation + S3 + SQS registered,
// with CloudFormation wired to the service locator for real provisioning.
func setupCFNStack(t *testing.T) *httptest.Server {
	t.Helper()

	cfg := config.Default()
	cfg.IAM.Mode = "none"

	store := iampkg.NewStore(cfg.AccountID)
	_ = store.InitRoot("ROOTKEY", "ROOTSECRET")
	engine := iampkg.NewEngine()

	registry := routing.NewRegistry()

	// Register S3 and SQS (resources that CloudFormation will provision)
	registry.Register(s3svc.New())
	registry.Register(sqssvc.New(cfg.AccountID, cfg.Region))

	// CloudFormation with locator (so it can provision real resources)
	cfnService := cfnsvc.New(cfg.AccountID, cfg.Region)
	cfnService.SetLocator(registry)
	registry.Register(cfnService)

	gw := gateway.NewWithIAM(cfg, registry, store, engine)
	return httptest.NewServer(gw)
}

// doCFNPost sends a form-encoded POST to the CloudFormation service.
func doCFNPost(t *testing.T, server *httptest.Server, action string, form url.Values) *http.Response {
	t.Helper()
	form.Set("Action", action)
	form.Set("Version", "2010-05-15")

	req, err := http.NewRequest(http.MethodPost, server.URL+"/", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20240101/us-east-1/cloudformation/aws4_request, SignedHeaders=host;x-amz-date, Signature=deadbeef")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// doS3Req sends an S3 request to the given server.
func doS3Req(t *testing.T, server *httptest.Server, method, path string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, server.URL+path, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20240101/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-date, Signature=deadbeef")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// TestCrossService_CloudFormationProvisioning verifies that CreateStack with
// a template containing S3 and SQS resources actually provisions those resources.
func TestCrossService_CloudFormationProvisioning(t *testing.T) {
	server := setupCFNStack(t)
	defer server.Close()

	// Step 1: CreateStack with a template that defines an S3 bucket
	template := `{
		"Description": "Integration test stack",
		"Resources": {
			"TestBucket": {
				"Type": "AWS::S3::Bucket",
				"Properties": {
					"BucketName": "cfn-integration-test-bucket"
				}
			},
			"TestQueue": {
				"Type": "AWS::SQS::Queue",
				"Properties": {
					"QueueName": "cfn-integration-test-queue"
				}
			}
		}
	}`

	form := url.Values{
		"StackName":    {"integration-test-stack"},
		"TemplateBody": {template},
	}
	resp := doCFNPost(t, server, "CreateStack", form)
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "CreateStack should return 200: %s", body)

	// Parse the response to get StackId
	var createResp struct {
		Result struct {
			StackId string `xml:"StackId"`
		} `xml:"CreateStackResult"`
	}
	require.NoError(t, xml.Unmarshal([]byte(body), &createResp))
	assert.NotEmpty(t, createResp.Result.StackId, "StackId should not be empty")
	assert.Contains(t, createResp.Result.StackId, "integration-test-stack")

	// Step 2: DescribeStacks to verify stack is CREATE_COMPLETE
	form = url.Values{"StackName": {"integration-test-stack"}}
	resp = doCFNPost(t, server, "DescribeStacks", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "CREATE_COMPLETE", "Stack should be in CREATE_COMPLETE state")
	assert.Contains(t, body, "integration-test-stack")

	// Step 3: Verify the S3 bucket was actually provisioned via S3 API
	resp = doS3Req(t, server, http.MethodHead, "/cfn-integration-test-bucket")
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"HeadBucket should return 200 for CFN-provisioned bucket")

	// Step 4: Verify the SQS queue was provisioned via SQS API
	form = url.Values{
		"QueueName": {"cfn-integration-test-queue"},
	}
	resp = doPost(t, server, "sqs", "GetQueueUrl", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "GetQueueUrl should return 200: %s", body)
	assert.Contains(t, body, "cfn-integration-test-queue",
		"SQS queue should exist after CFN provisioning")

	// Step 5: DescribeStackResources to verify resource metadata
	form = url.Values{"StackName": {"integration-test-stack"}}
	resp = doCFNPost(t, server, "DescribeStackResources", form)
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "AWS::S3::Bucket", "Resources should include S3 bucket type")
	assert.Contains(t, body, "AWS::SQS::Queue", "Resources should include SQS queue type")
	assert.Contains(t, body, "CREATE_COMPLETE", "Resources should be in CREATE_COMPLETE state")
}
