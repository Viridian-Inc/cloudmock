package applicationautoscaling_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/applicationautoscaling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.ApplicationAutoScalingService {
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

func decode(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// ---- Scalable Target tests ----

func TestRegisterScalableTarget(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace":  "ecs",
		"ResourceId":        "service/my-cluster/my-service",
		"ScalableDimension": "ecs:service:DesiredCount",
		"MinCapacity":       1,
		"MaxCapacity":       10,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRegisterScalableTargetMissingFields(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
}

func TestDescribeScalableTargets(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs", "ResourceId": "service/my-cluster/svc-1",
		"ScalableDimension": "ecs:service:DesiredCount", "MinCapacity": 1, "MaxCapacity": 10,
	}))
	s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs", "ResourceId": "service/my-cluster/svc-2",
		"ScalableDimension": "ecs:service:DesiredCount", "MinCapacity": 2, "MaxCapacity": 20,
	}))

	resp, err := s.HandleRequest(jsonCtx("DescribeScalableTargets", map[string]any{
		"ServiceNamespace": "ecs",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	targets := m["ScalableTargets"].([]any)
	assert.Len(t, targets, 2)
}

func TestDescribeScalableTargetsWithResourceFilter(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs", "ResourceId": "service/my-cluster/svc-1",
		"ScalableDimension": "ecs:service:DesiredCount", "MinCapacity": 1, "MaxCapacity": 10,
	}))
	s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs", "ResourceId": "service/my-cluster/svc-2",
		"ScalableDimension": "ecs:service:DesiredCount", "MinCapacity": 2, "MaxCapacity": 20,
	}))

	resp, err := s.HandleRequest(jsonCtx("DescribeScalableTargets", map[string]any{
		"ServiceNamespace": "ecs",
		"ResourceIds":      []string{"service/my-cluster/svc-1"},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	targets := m["ScalableTargets"].([]any)
	assert.Len(t, targets, 1)
}

func TestDeregisterScalableTarget(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs", "ResourceId": "service/my-cluster/svc-1",
		"ScalableDimension": "ecs:service:DesiredCount", "MinCapacity": 1, "MaxCapacity": 10,
	}))

	resp, err := s.HandleRequest(jsonCtx("DeregisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs", "ResourceId": "service/my-cluster/svc-1",
		"ScalableDimension": "ecs:service:DesiredCount",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify gone
	descResp, err := s.HandleRequest(jsonCtx("DescribeScalableTargets", map[string]any{
		"ServiceNamespace": "ecs",
	}))
	require.NoError(t, err)
	m := decode(t, descResp)
	targets := m["ScalableTargets"].([]any)
	assert.Len(t, targets, 0)
}

func TestDeregisterScalableTargetNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeregisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs", "ResourceId": "nonexistent",
		"ScalableDimension": "ecs:service:DesiredCount",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ObjectNotFoundException", awsErr.Code)
}

// ---- Scaling Policy tests ----

func TestPutScalingPolicy(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("PutScalingPolicy", map[string]any{
		"PolicyName": "my-policy", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
		"PolicyType": "TargetTrackingScaling",
		"TargetTrackingScalingPolicyConfiguration": map[string]any{"TargetValue": 50.0},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["PolicyARN"])
}

func TestDescribeScalingPolicies(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutScalingPolicy", map[string]any{
		"PolicyName": "policy-1", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
	}))
	s.HandleRequest(jsonCtx("PutScalingPolicy", map[string]any{
		"PolicyName": "policy-2", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
	}))

	resp, err := s.HandleRequest(jsonCtx("DescribeScalingPolicies", map[string]any{
		"ServiceNamespace": "ecs",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	policies := m["ScalingPolicies"].([]any)
	assert.Len(t, policies, 2)
}

func TestDeleteScalingPolicy(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutScalingPolicy", map[string]any{
		"PolicyName": "my-policy", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
	}))

	resp, err := s.HandleRequest(jsonCtx("DeleteScalingPolicy", map[string]any{
		"PolicyName": "my-policy", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteScalingPolicyNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteScalingPolicy", map[string]any{
		"PolicyName": "nonexistent", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ObjectNotFoundException", awsErr.Code)
}

// ---- Scheduled Action tests ----

func TestPutScheduledAction(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("PutScheduledAction", map[string]any{
		"ScheduledActionName": "my-action", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
		"Schedule": "cron(0 9 * * ? *)",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeScheduledActions(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutScheduledAction", map[string]any{
		"ScheduledActionName": "action-1", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
		"Schedule": "cron(0 9 * * ? *)",
	}))
	s.HandleRequest(jsonCtx("PutScheduledAction", map[string]any{
		"ScheduledActionName": "action-2", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
		"Schedule": "cron(0 18 * * ? *)",
	}))

	resp, err := s.HandleRequest(jsonCtx("DescribeScheduledActions", map[string]any{
		"ServiceNamespace": "ecs",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	actions := m["ScheduledActions"].([]any)
	assert.Len(t, actions, 2)
}

func TestDeleteScheduledAction(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutScheduledAction", map[string]any{
		"ScheduledActionName": "my-action", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
		"Schedule": "cron(0 9 * * ? *)",
	}))

	_, err := s.HandleRequest(jsonCtx("DeleteScheduledAction", map[string]any{
		"ScheduledActionName": "my-action", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
	}))
	require.NoError(t, err)
}

func TestDeleteScheduledActionNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteScheduledAction", map[string]any{
		"ScheduledActionName": "nonexistent", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ObjectNotFoundException", awsErr.Code)
}

// ---- Tagging tests ----

func TestTagScalingPolicy(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("PutScalingPolicy", map[string]any{
		"PolicyName": "my-policy", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
	}))
	policyARN := decode(t, createResp)["PolicyARN"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": policyARN,
		"Tags":        map[string]any{"env": "prod"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": policyARN}))
	require.NoError(t, err)
	m := decode(t, resp)
	tags := m["Tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
}

func TestUntagResource(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("PutScalingPolicy", map[string]any{
		"PolicyName": "my-policy", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
	}))
	policyARN := decode(t, createResp)["PolicyARN"].(string)

	s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": policyARN,
		"Tags":        map[string]any{"env": "prod", "team": "alpha"},
	}))

	_, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceARN": policyARN,
		"TagKeys":     []string{"team"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": policyARN}))
	require.NoError(t, err)
	m := decode(t, resp)
	tags := m["Tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
	assert.Nil(t, tags["team"])
}

// ---- Invalid action ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- ServiceNamespace validation ----

func TestRegisterScalableTargetInvalidNamespace(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace":  "invalid-namespace",
		"ResourceId":        "service/my-cluster/my-service",
		"ScalableDimension": "ecs:service:DesiredCount",
		"MinCapacity":       1,
		"MaxCapacity":       10,
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
	assert.Contains(t, awsErr.Message, "ServiceNamespace")
}

func TestPutScalingPolicyTargetTrackingRequiresTargetValue(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("PutScalingPolicy", map[string]any{
		"PolicyName": "my-policy", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
		"PolicyType": "TargetTrackingScaling",
		"TargetTrackingScalingPolicyConfiguration": map[string]any{
			"PredefinedMetricSpecification": map[string]any{
				"PredefinedMetricType": "ECSServiceAverageCPUUtilization",
			},
		},
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
	assert.Contains(t, awsErr.Message, "TargetValue")
}

func TestPutScalingPolicyTargetTrackingWithTargetValue(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("PutScalingPolicy", map[string]any{
		"PolicyName": "my-policy", "ServiceNamespace": "ecs",
		"ResourceId": "service/my-cluster/svc-1", "ScalableDimension": "ecs:service:DesiredCount",
		"PolicyType": "TargetTrackingScaling",
		"TargetTrackingScalingPolicyConfiguration": map[string]any{
			"TargetValue": 50.0,
		},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["PolicyARN"])
}

// ---- Update (re-register) scalable target ----

func TestUpdateScalableTarget(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs", "ResourceId": "service/my-cluster/svc-1",
		"ScalableDimension": "ecs:service:DesiredCount", "MinCapacity": 1, "MaxCapacity": 10,
	}))

	// Re-register with updated capacity
	s.HandleRequest(jsonCtx("RegisterScalableTarget", map[string]any{
		"ServiceNamespace": "ecs", "ResourceId": "service/my-cluster/svc-1",
		"ScalableDimension": "ecs:service:DesiredCount", "MinCapacity": 5, "MaxCapacity": 50,
	}))

	resp, err := s.HandleRequest(jsonCtx("DescribeScalableTargets", map[string]any{
		"ServiceNamespace": "ecs",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	targets := m["ScalableTargets"].([]any)
	assert.Len(t, targets, 1)
	target := targets[0].(map[string]any)
	assert.Equal(t, float64(5), target["MinCapacity"])
	assert.Equal(t, float64(50), target["MaxCapacity"])
}
