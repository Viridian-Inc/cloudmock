package config_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.ConfigService {
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

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// --- Config Rules ---

func TestConfig_PutConfigRule(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("PutConfigRule", map[string]any{
		"ConfigRule": map[string]any{
			"ConfigRuleName": "s3-bucket-versioning",
			"Source": map[string]any{
				"Owner":            "AWS",
				"SourceIdentifier": "S3_BUCKET_VERSIONING_ENABLED",
			},
		},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, "s3-bucket-versioning", m["ConfigRuleName"])
	assert.NotEmpty(t, m["ConfigRuleArn"])
	assert.NotEmpty(t, m["ConfigRuleId"])
	assert.Equal(t, "ACTIVE", m["ConfigRuleState"])
}

func TestConfig_DescribeConfigRules(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutConfigRule", map[string]any{
		"ConfigRule": map[string]any{"ConfigRuleName": "rule-1", "Source": map[string]any{"Owner": "AWS", "SourceIdentifier": "ID1"}},
	}))
	s.HandleRequest(jsonCtx("PutConfigRule", map[string]any{
		"ConfigRule": map[string]any{"ConfigRuleName": "rule-2", "Source": map[string]any{"Owner": "AWS", "SourceIdentifier": "ID2"}},
	}))

	resp, err := s.HandleRequest(jsonCtx("DescribeConfigRules", map[string]any{}))
	require.NoError(t, err)
	rules := respBody(t, resp)["ConfigRules"].([]any)
	assert.Len(t, rules, 2)

	// Filter by name
	resp, err = s.HandleRequest(jsonCtx("DescribeConfigRules", map[string]any{
		"ConfigRuleNames": []any{"rule-1"},
	}))
	require.NoError(t, err)
	rules = respBody(t, resp)["ConfigRules"].([]any)
	assert.Len(t, rules, 1)
}

func TestConfig_DeleteConfigRule(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutConfigRule", map[string]any{
		"ConfigRule": map[string]any{"ConfigRuleName": "delete-me", "Source": map[string]any{"Owner": "AWS", "SourceIdentifier": "ID"}},
	}))
	_, err := s.HandleRequest(jsonCtx("DeleteConfigRule", map[string]any{"ConfigRuleName": "delete-me"}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("DescribeConfigRules", map[string]any{}))
	rules := respBody(t, resp)["ConfigRules"].([]any)
	assert.Len(t, rules, 0)
}

func TestConfig_DeleteConfigRule_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteConfigRule", map[string]any{"ConfigRuleName": "nonexistent"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NoSuchConfigRuleException")
}

// --- Configuration Recorders ---

func TestConfig_PutConfigurationRecorder(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("PutConfigurationRecorder", map[string]any{
		"ConfigurationRecorder": map[string]any{
			"name":    "default",
			"roleARN": "arn:aws:iam::123456789012:role/config-role",
			"recordingGroup": map[string]any{
				"allSupported":               true,
				"includeGlobalResourceTypes": true,
			},
		},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("DescribeConfigurationRecorders", map[string]any{}))
	recs := respBody(t, resp)["ConfigurationRecorders"].([]any)
	assert.Len(t, recs, 1)
	rec := recs[0].(map[string]any)
	assert.Equal(t, "default", rec["name"])
}

func TestConfig_StartStopRecorder(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutConfigurationRecorder", map[string]any{
		"ConfigurationRecorder": map[string]any{"name": "default", "roleARN": "arn:role"},
	}))

	_, err := s.HandleRequest(jsonCtx("StartConfigurationRecorder", map[string]any{
		"ConfigurationRecorderName": "default",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("StopConfigurationRecorder", map[string]any{
		"ConfigurationRecorderName": "default",
	}))
	require.NoError(t, err)
}

func TestConfig_DeleteConfigurationRecorder(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutConfigurationRecorder", map[string]any{
		"ConfigurationRecorder": map[string]any{"name": "default", "roleARN": "arn:role"},
	}))
	_, err := s.HandleRequest(jsonCtx("DeleteConfigurationRecorder", map[string]any{
		"ConfigurationRecorderName": "default",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteConfigurationRecorder", map[string]any{
		"ConfigurationRecorderName": "default",
	}))
	require.Error(t, err)
}

// --- Delivery Channels ---

func TestConfig_PutDeliveryChannel(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("PutDeliveryChannel", map[string]any{
		"DeliveryChannel": map[string]any{
			"name":         "default",
			"s3BucketName": "config-bucket",
		},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("DescribeDeliveryChannels", map[string]any{}))
	channels := respBody(t, resp)["DeliveryChannels"].([]any)
	assert.Len(t, channels, 1)
}

func TestConfig_PutDeliveryChannel_MissingBucket(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("PutDeliveryChannel", map[string]any{
		"DeliveryChannel": map[string]any{"name": "default"},
	}))
	require.Error(t, err)
}

func TestConfig_DeleteDeliveryChannel(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutDeliveryChannel", map[string]any{
		"DeliveryChannel": map[string]any{"name": "default", "s3BucketName": "bucket"},
	}))
	_, err := s.HandleRequest(jsonCtx("DeleteDeliveryChannel", map[string]any{
		"DeliveryChannelName": "default",
	}))
	require.NoError(t, err)
}

// --- Conformance Packs ---

func TestConfig_PutConformancePack(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("PutConformancePack", map[string]any{
		"ConformancePackName": "my-pack",
		"DeliveryS3Bucket":   "pack-bucket",
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotEmpty(t, m["ConformancePackArn"])
}

func TestConfig_DescribeConformancePacks(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutConformancePack", map[string]any{
		"ConformancePackName": "pack-1", "DeliveryS3Bucket": "b",
	}))
	s.HandleRequest(jsonCtx("PutConformancePack", map[string]any{
		"ConformancePackName": "pack-2", "DeliveryS3Bucket": "b",
	}))
	resp, _ := s.HandleRequest(jsonCtx("DescribeConformancePacks", map[string]any{}))
	packs := respBody(t, resp)["ConformancePackDetails"].([]any)
	assert.Len(t, packs, 2)
}

func TestConfig_DeleteConformancePack(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutConformancePack", map[string]any{
		"ConformancePackName": "del-pack", "DeliveryS3Bucket": "b",
	}))
	_, err := s.HandleRequest(jsonCtx("DeleteConformancePack", map[string]any{
		"ConformancePackName": "del-pack",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteConformancePack", map[string]any{
		"ConformancePackName": "del-pack",
	}))
	require.Error(t, err)
}

// --- Evaluations / Compliance ---

func TestConfig_PutEvaluations(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("PutEvaluations", map[string]any{
		"ResultToken": "my-rule",
		"Evaluations": []any{
			map[string]any{
				"ComplianceResourceType": "AWS::S3::Bucket",
				"ComplianceResourceId":   "my-bucket",
				"ComplianceType":         "COMPLIANT",
			},
		},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotNil(t, m["FailedEvaluations"])
}

func TestConfig_GetComplianceDetailsByConfigRule(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetComplianceDetailsByConfigRule", map[string]any{
		"ConfigRuleName": "some-rule",
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotNil(t, m["EvaluationResults"])
}

func TestConfig_DescribeComplianceByConfigRule(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutConfigRule", map[string]any{
		"ConfigRule": map[string]any{"ConfigRuleName": "compliance-rule", "Source": map[string]any{"Owner": "AWS", "SourceIdentifier": "ID"}},
	}))
	resp, err := s.HandleRequest(jsonCtx("DescribeComplianceByConfigRule", map[string]any{}))
	require.NoError(t, err)
	m := respBody(t, resp)
	items := m["ComplianceByConfigRules"].([]any)
	assert.Len(t, items, 1)
}

func TestConfig_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

func TestConfig_GetResourceConfigHistory(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetResourceConfigHistory", map[string]any{}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotNil(t, m["configurationItems"])
}

// --- Behavioral Tests ---

func newServiceWithBus() (*svc.ConfigService, *eventbus.Bus) {
	bus := eventbus.NewBus()
	s := svc.NewWithBus("123456789012", "us-east-1", bus)
	return s, bus
}

func TestConfig_Recorder_TracksConfigItems(t *testing.T) {
	s, bus := newServiceWithBus()

	// Create recorder and start it
	s.HandleRequest(jsonCtx("PutConfigurationRecorder", map[string]any{
		"ConfigurationRecorder": map[string]any{
			"name":    "default",
			"roleARN": "arn:aws:iam::123456789012:role/config-role",
			"recordingGroup": map[string]any{
				"allSupported": true,
			},
		},
	}))

	_, err := s.HandleRequest(jsonCtx("StartConfigurationRecorder", map[string]any{
		"ConfigurationRecorderName": "default",
	}))
	require.NoError(t, err)

	// Publish a resource change event
	bus.PublishSync(&eventbus.Event{
		Source: "s3",
		Type:   "s3:ApiCall:CreateBucket",
		Detail: map[string]any{
			"resourceType": "AWS::S3::Bucket",
			"resourceId":   "my-test-bucket",
			"resourceName": "my-test-bucket",
		},
		Time:      time.Now().UTC(),
		Region:    "us-east-1",
		AccountID: "123456789012",
	})

	// Query config history
	resp, err := s.HandleRequest(jsonCtx("GetResourceConfigHistory", map[string]any{
		"resourceType": "AWS::S3::Bucket",
		"resourceId":   "my-test-bucket",
	}))
	require.NoError(t, err)
	items := respBody(t, resp)["configurationItems"].([]any)
	assert.Len(t, items, 1)
	item := items[0].(map[string]any)
	assert.Equal(t, "AWS::S3::Bucket", item["resourceType"])
	assert.Equal(t, "my-test-bucket", item["resourceId"])
}

func TestConfig_Recorder_StopStopsTracking(t *testing.T) {
	s, bus := newServiceWithBus()

	s.HandleRequest(jsonCtx("PutConfigurationRecorder", map[string]any{
		"ConfigurationRecorder": map[string]any{"name": "default", "roleARN": "arn:role"},
	}))
	s.HandleRequest(jsonCtx("StartConfigurationRecorder", map[string]any{
		"ConfigurationRecorderName": "default",
	}))
	s.HandleRequest(jsonCtx("StopConfigurationRecorder", map[string]any{
		"ConfigurationRecorderName": "default",
	}))

	// Events after stop should not be recorded
	bus.PublishSync(&eventbus.Event{
		Source: "ec2", Type: "ec2:ApiCall:RunInstances",
		Detail: map[string]any{"resourceType": "AWS::EC2::Instance", "resourceId": "i-123"},
		Time:   time.Now().UTC(),
	})

	resp, _ := s.HandleRequest(jsonCtx("GetResourceConfigHistory", map[string]any{
		"resourceType": "AWS::EC2::Instance",
		"resourceId":   "i-123",
	}))
	items := respBody(t, resp)["configurationItems"].([]any)
	assert.Len(t, items, 0)
}

func TestConfig_Evaluations_ComplianceFlow(t *testing.T) {
	s := newService()

	// Create a config rule
	s.HandleRequest(jsonCtx("PutConfigRule", map[string]any{
		"ConfigRule": map[string]any{
			"ConfigRuleName": "s3-versioning",
			"Source":         map[string]any{"Owner": "AWS", "SourceIdentifier": "S3_BUCKET_VERSIONING_ENABLED"},
		},
	}))

	// Put evaluations for the rule
	_, err := s.HandleRequest(jsonCtx("PutEvaluations", map[string]any{
		"ResultToken": "s3-versioning",
		"Evaluations": []any{
			map[string]any{
				"ComplianceResourceType": "AWS::S3::Bucket",
				"ComplianceResourceId":   "compliant-bucket",
				"ComplianceType":         "COMPLIANT",
			},
			map[string]any{
				"ComplianceResourceType": "AWS::S3::Bucket",
				"ComplianceResourceId":   "bad-bucket",
				"ComplianceType":         "NON_COMPLIANT",
				"Annotation":             "Versioning not enabled",
			},
		},
	}))
	require.NoError(t, err)

	// Get compliance details
	resp, err := s.HandleRequest(jsonCtx("GetComplianceDetailsByConfigRule", map[string]any{
		"ConfigRuleName": "s3-versioning",
	}))
	require.NoError(t, err)
	results := respBody(t, resp)["EvaluationResults"].([]any)
	assert.Len(t, results, 2)

	// Describe compliance by rule
	resp, err = s.HandleRequest(jsonCtx("DescribeComplianceByConfigRule", map[string]any{}))
	require.NoError(t, err)
	compliance := respBody(t, resp)["ComplianceByConfigRules"].([]any)
	assert.Len(t, compliance, 1)
	rule := compliance[0].(map[string]any)
	comp := rule["Compliance"].(map[string]any)
	assert.Equal(t, "NON_COMPLIANT", comp["ComplianceType"])
}

func TestConfig_NoBus_DegradeGracefully(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutConfigurationRecorder", map[string]any{
		"ConfigurationRecorder": map[string]any{"name": "default", "roleARN": "arn:role"},
	}))
	_, err := s.HandleRequest(jsonCtx("StartConfigurationRecorder", map[string]any{
		"ConfigurationRecorderName": "default",
	}))
	require.NoError(t, err)
	_, err = s.HandleRequest(jsonCtx("StopConfigurationRecorder", map[string]any{
		"ConfigurationRecorderName": "default",
	}))
	require.NoError(t, err)
}
