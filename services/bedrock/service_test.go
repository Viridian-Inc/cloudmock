package bedrock_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/bedrock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.BedrockService {
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

// ---- Test 1: CreateModelCustomizationJob ----

func TestCreateModelCustomizationJob(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateModelCustomizationJob", map[string]any{
		"jobName":             "cust-job-1",
		"baseModelIdentifier": "amazon.titan-text-express-v1",
		"customModelName":     "my-custom-model",
		"roleArn":             "arn:aws:iam::123456789012:role/Role",
		"trainingDataConfig":  map[string]any{"s3Uri": "s3://bucket/training"},
		"outputDataConfig":    map[string]any{"s3Uri": "s3://bucket/output"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := respBody(t, resp)
	assert.Contains(t, body["jobArn"].(string), "cust-job-1")
}

// ---- Test 2: GetModelCustomizationJob ----

func TestGetModelCustomizationJob(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateModelCustomizationJob", map[string]any{
		"jobName":             "cust-get",
		"baseModelIdentifier": "amazon.titan-text-express-v1",
		"customModelName":     "custom-get",
		"roleArn":             "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetModelCustomizationJob", map[string]any{
		"jobIdentifier": "cust-get",
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "cust-get", body["jobName"])
	assert.Contains(t, []string{"InProgress", "Completed"}, body["status"])
}

// ---- Test 3: ListModelCustomizationJobs ----

func TestListModelCustomizationJobs(t *testing.T) {
	s := newService()
	for _, name := range []string{"cj-1", "cj-2"} {
		_, err := s.HandleRequest(jsonCtx("CreateModelCustomizationJob", map[string]any{
			"jobName":             name,
			"baseModelIdentifier": "amazon.titan-text-express-v1",
			"customModelName":     name + "-model",
			"roleArn":             "arn:aws:iam::123456789012:role/Role",
		}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListModelCustomizationJobs", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	summaries := body["modelCustomizationJobSummaries"].([]any)
	assert.Len(t, summaries, 2)
}

// ---- Test 4: StopModelCustomizationJob ----

func TestStopModelCustomizationJob(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateModelCustomizationJob", map[string]any{
		"jobName":             "cj-stop",
		"baseModelIdentifier": "amazon.titan-text-express-v1",
		"customModelName":     "stop-model",
		"roleArn":             "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("StopModelCustomizationJob", map[string]any{
		"jobIdentifier": "cj-stop",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ---- Test 5: Customization job lifecycle InProgress -> Completed ----

func TestCustomizationJobLifecycle(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateModelCustomizationJob", map[string]any{
		"jobName":             "cj-lc",
		"baseModelIdentifier": "amazon.titan-text-express-v1",
		"customModelName":     "lc-model",
		"roleArn":             "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	resp, err := s.HandleRequest(jsonCtx("GetModelCustomizationJob", map[string]any{"jobIdentifier": "cj-lc"}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "Completed", body["status"])
	assert.NotNil(t, body["endTime"])
}

// ---- Test 6: CreateProvisionedModelThroughput ----

func TestCreateProvisionedModelThroughput(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateProvisionedModelThroughput", map[string]any{
		"provisionedModelName": "prov-1",
		"modelId":              "amazon.titan-text-express-v1",
		"modelUnits":           float64(1),
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body := respBody(t, resp)
	assert.Contains(t, body["provisionedModelArn"].(string), "prov-1")
}

// ---- Test 7: GetProvisionedModelThroughput ----

func TestGetProvisionedModelThroughput(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateProvisionedModelThroughput", map[string]any{
		"provisionedModelName": "prov-get",
		"modelId":              "amazon.titan-text-express-v1",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetProvisionedModelThroughput", map[string]any{
		"provisionedModelId": "prov-get",
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "prov-get", body["provisionedModelName"])
}

// ---- Test 8: ListProvisionedModelThroughputs ----

func TestListProvisionedModelThroughputs(t *testing.T) {
	s := newService()
	for _, name := range []string{"pm-1", "pm-2"} {
		_, err := s.HandleRequest(jsonCtx("CreateProvisionedModelThroughput", map[string]any{
			"provisionedModelName": name,
			"modelId":              "amazon.titan-text-express-v1",
		}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListProvisionedModelThroughputs", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	summaries := body["provisionedModelSummaries"].([]any)
	assert.Len(t, summaries, 2)
}

// ---- Test 9: UpdateProvisionedModelThroughput ----

func TestUpdateProvisionedModelThroughput(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateProvisionedModelThroughput", map[string]any{
		"provisionedModelName": "prov-upd",
		"modelId":              "amazon.titan-text-express-v1",
		"modelUnits":           float64(1),
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("UpdateProvisionedModelThroughput", map[string]any{
		"provisionedModelId": "prov-upd",
		"desiredModelUnits":  float64(2),
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ---- Test 10: DeleteProvisionedModelThroughput ----

func TestDeleteProvisionedModelThroughput(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateProvisionedModelThroughput", map[string]any{
		"provisionedModelName": "prov-del",
		"modelId":              "amazon.titan-text-express-v1",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteProvisionedModelThroughput", map[string]any{
		"provisionedModelId": "prov-del",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetProvisionedModelThroughput", map[string]any{
		"provisionedModelId": "prov-del",
	}))
	require.Error(t, err)
}

// ---- Test 11: ListFoundationModels ----

func TestListFoundationModels(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("ListFoundationModels", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	summaries := body["modelSummaries"].([]any)
	assert.GreaterOrEqual(t, len(summaries), 5)

	// Verify Anthropic model is present
	found := false
	for _, m := range summaries {
		model := m.(map[string]any)
		if model["provider"] == "Anthropic" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected Anthropic model in foundation models")
}

// ---- Test 12: GetFoundationModel ----

func TestGetFoundationModel(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetFoundationModel", map[string]any{
		"modelIdentifier": "anthropic.claude-3-5-sonnet-20241022-v2:0",
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	details := body["modelDetails"].(map[string]any)
	assert.Equal(t, "Anthropic", details["provider"])
	assert.Equal(t, "Claude 3.5 Sonnet v2", details["modelName"])
}

// ---- Test 13: GetFoundationModel NotFound ----

func TestGetFoundationModelNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetFoundationModel", map[string]any{
		"modelIdentifier": "nonexistent-model",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

// ---- Test 14: InvalidAction ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 15: Tagging ----

func TestTagging(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateModelCustomizationJob", map[string]any{
		"jobName":             "cj-tag",
		"baseModelIdentifier": "amazon.titan-text-express-v1",
		"customModelName":     "tag-model",
		"roleArn":             "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)
	arn := "arn:aws:bedrock:us-east-1:123456789012:model-customization-job/cj-tag"

	_, err = s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"resourceARN": arn,
		"tags":        []any{map[string]any{"key": "env", "value": "test"}},
	}))
	require.NoError(t, err)

	listResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceARN": arn}))
	require.NoError(t, err)
	tags := respBody(t, listResp)["tags"].([]any)
	assert.Len(t, tags, 1)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"resourceARN": arn,
		"tagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	listResp2, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceARN": arn}))
	require.NoError(t, err)
	tags2 := respBody(t, listResp2)["tags"].([]any)
	assert.Len(t, tags2, 0)
}

// ---- Test 16: Service Name and HealthCheck ----

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "bedrock", s.Name())
	assert.NoError(t, s.HealthCheck())
}

// ---- Test 17: Duplicate customization job ----

func TestDuplicateCustomizationJob(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateModelCustomizationJob", map[string]any{
		"jobName":             "cj-dup",
		"baseModelIdentifier": "amazon.titan-text-express-v1",
		"customModelName":     "dup-model",
		"roleArn":             "arn:aws:iam::123456789012:role/Role",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("CreateModelCustomizationJob", map[string]any{
		"jobName":             "cj-dup",
		"baseModelIdentifier": "amazon.titan-text-express-v1",
		"customModelName":     "dup-model2",
		"roleArn":             "arn:aws:iam::123456789012:role/Role",
	}))
	require.Error(t, err)
}

// ---- Enrichment tests ----

func TestInvokeModel(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("InvokeModel", map[string]any{
		"modelId": "anthropic.claude-3-5-sonnet-20241022-v2:0",
		"body":    `{"prompt": "What is 2+2?"}`,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotEmpty(t, m["body"])
	assert.Equal(t, "application/json", m["contentType"])

	// Parse the body to verify mock response structure.
	var body map[string]any
	err = json.Unmarshal([]byte(m["body"].(string)), &body)
	require.NoError(t, err)
	assert.Contains(t, body["generated_text"], "mock response")
	assert.NotNil(t, body["usage"])
}

func TestInvokeModel_InvalidModel(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("InvokeModel", map[string]any{
		"modelId": "nonexistent-model",
		"body":    `{"prompt": "test"}`,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Could not resolve")
}

func TestInvokeModel_MissingModelId(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("InvokeModel", map[string]any{
		"body": `{"prompt": "test"}`,
	}))
	require.Error(t, err)
}

func TestCreateAndApplyGuardrail(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateGuardrail", map[string]any{
		"name":                    "content-filter",
		"description":             "Blocks harmful content",
		"blockedInputMessaging":   "Input blocked by guardrail.",
		"blockedOutputsMessaging": "Output blocked by guardrail.",
	}))
	require.NoError(t, err)
	cm := respBody(t, createResp)
	guardrailID := cm["guardrailId"].(string)
	assert.NotEmpty(t, guardrailID)
	assert.NotEmpty(t, cm["guardrailArn"])
	assert.Equal(t, "DRAFT", cm["version"])

	// Apply guardrail.
	applyResp, err := s.HandleRequest(jsonCtx("ApplyGuardrail", map[string]any{
		"guardrailIdentifier": guardrailID,
		"source":              "INPUT",
		"content":             "Hello world",
	}))
	require.NoError(t, err)
	am := respBody(t, applyResp)
	assert.Equal(t, "NONE", am["action"])
	assert.NotNil(t, am["assessments"])
}

func TestApplyGuardrail_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("ApplyGuardrail", map[string]any{
		"guardrailIdentifier": "nonexistent",
		"content":             "test",
	}))
	require.Error(t, err)
}
