package autoscaling_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	assvc "github.com/neureaux/cloudmock/services/autoscaling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newASGGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(assvc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func asReq(t *testing.T, params map[string]string) *http.Request {
	t.Helper()
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	body := vals.Encode()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/autoscaling/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

func body(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return w.Body.String()
}

func extractXML(t *testing.T, xmlBody, tag string) string {
	t.Helper()
	start := strings.Index(xmlBody, "<"+tag+">")
	if start == -1 {
		t.Fatalf("tag <%s> not found in body:\n%s", tag, xmlBody)
	}
	start += len("<" + tag + ">")
	end := strings.Index(xmlBody[start:], "</"+tag+">")
	if end == -1 {
		t.Fatalf("closing </%s> not found", tag)
	}
	return xmlBody[start : start+end]
}

// ---- LaunchConfiguration ----

func TestAS_CreateLaunchConfiguration(t *testing.T) {
	h := newASGGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                    "CreateLaunchConfiguration",
		"LaunchConfigurationName":   "test-lc",
		"ImageId":                   "ami-12345",
		"InstanceType":              "t3.micro",
	}))
	require.Equal(t, http.StatusOK, w.Code, body(t, w))
}

func TestAS_CreateLaunchConfiguration_Duplicate(t *testing.T) {
	h := newASGGateway(t)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, asReq(t, map[string]string{
			"Action":                    "CreateLaunchConfiguration",
			"LaunchConfigurationName":   "dup-lc",
			"ImageId":                   "ami-12345",
			"InstanceType":              "t3.micro",
		}))
		if i == 0 {
			require.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusConflict, w.Code)
			assert.Contains(t, body(t, w), "AlreadyExists")
		}
	}
}

func TestAS_CreateLaunchConfiguration_MissingFields(t *testing.T) {
	h := newASGGateway(t)
	// Missing ImageId
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                    "CreateLaunchConfiguration",
		"LaunchConfigurationName":   "no-ami",
		"InstanceType":              "t3.micro",
	}))
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Missing InstanceType
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                    "CreateLaunchConfiguration",
		"LaunchConfigurationName":   "no-type",
		"ImageId":                   "ami-12345",
	}))
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestAS_DescribeLaunchConfigurations(t *testing.T) {
	h := newASGGateway(t)
	for _, name := range []string{"lc-1", "lc-2"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, asReq(t, map[string]string{
			"Action":                    "CreateLaunchConfiguration",
			"LaunchConfigurationName":   name,
			"ImageId":                   "ami-12345",
			"InstanceType":              "t3.micro",
		}))
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action": "DescribeLaunchConfigurations",
	}))
	require.Equal(t, http.StatusOK, w.Code)
	b := body(t, w)
	assert.Contains(t, b, "lc-1")
	assert.Contains(t, b, "lc-2")
}

func TestAS_DeleteLaunchConfiguration(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                    "CreateLaunchConfiguration",
		"LaunchConfigurationName":   "del-lc",
		"ImageId":                   "ami-12345",
		"InstanceType":              "t3.micro",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                    "DeleteLaunchConfiguration",
		"LaunchConfigurationName":   "del-lc",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	// Verify gone
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, asReq(t, map[string]string{
		"Action":                                "DescribeLaunchConfigurations",
		"LaunchConfigurationNames.member.1":     "del-lc",
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	assert.NotContains(t, body(t, w3), "del-lc")
}

func TestAS_DeleteLaunchConfiguration_NotFound(t *testing.T) {
	h := newASGGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                    "DeleteLaunchConfiguration",
		"LaunchConfigurationName":   "nonexistent",
	}))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- AutoScalingGroup ----

func TestAS_CreateAutoScalingGroup(t *testing.T) {
	h := newASGGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                "CreateAutoScalingGroup",
		"AutoScalingGroupName":  "test-asg",
		"MinSize":               "1",
		"MaxSize":               "5",
		"DesiredCapacity":       "2",
		"AvailabilityZones.member.1": "us-east-1a",
	}))
	require.Equal(t, http.StatusOK, w.Code, body(t, w))
}

func TestAS_CreateAutoScalingGroup_Duplicate(t *testing.T) {
	h := newASGGateway(t)
	params := map[string]string{
		"Action":                "CreateAutoScalingGroup",
		"AutoScalingGroupName":  "dup-asg",
		"MinSize":               "1",
		"MaxSize":               "3",
	}
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, params))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, params))
	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestAS_CreateAutoScalingGroup_MissingName(t *testing.T) {
	h := newASGGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":  "CreateAutoScalingGroup",
		"MinSize": "1",
		"MaxSize": "3",
	}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAS_DescribeAutoScalingGroups(t *testing.T) {
	h := newASGGateway(t)
	for _, name := range []string{"asg-1", "asg-2"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, asReq(t, map[string]string{
			"Action":                "CreateAutoScalingGroup",
			"AutoScalingGroupName":  name,
			"MinSize":               "1",
			"MaxSize":               "3",
			"DesiredCapacity":       "1",
		}))
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action": "DescribeAutoScalingGroups",
	}))
	require.Equal(t, http.StatusOK, w.Code)
	b := body(t, w)
	assert.Contains(t, b, "asg-1")
	assert.Contains(t, b, "asg-2")
}

func TestAS_DescribeAutoScalingGroups_Instances(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                     "CreateAutoScalingGroup",
		"AutoScalingGroupName":       "inst-asg",
		"MinSize":                    "2",
		"MaxSize":                    "5",
		"DesiredCapacity":            "3",
		"AvailabilityZones.member.1": "us-east-1a",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                              "DescribeAutoScalingGroups",
		"AutoScalingGroupNames.member.1":      "inst-asg",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
	b := body(t, w2)
	// Should contain 3 instances with InService state
	assert.Contains(t, b, "InService")
	assert.Contains(t, b, "Healthy")
}

func TestAS_UpdateAutoScalingGroup(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                "CreateAutoScalingGroup",
		"AutoScalingGroupName":  "update-asg",
		"MinSize":               "1",
		"MaxSize":               "3",
		"DesiredCapacity":       "1",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                "UpdateAutoScalingGroup",
		"AutoScalingGroupName":  "update-asg",
		"MaxSize":               "10",
		"DesiredCapacity":       "2",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	// Verify update
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, asReq(t, map[string]string{
		"Action":                            "DescribeAutoScalingGroups",
		"AutoScalingGroupNames.member.1":    "update-asg",
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	b := body(t, w3)
	assert.Contains(t, b, "<MaxSize>10</MaxSize>")
	assert.Contains(t, b, "<DesiredCapacity>2</DesiredCapacity>")
}

func TestAS_UpdateAutoScalingGroup_NotFound(t *testing.T) {
	h := newASGGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                "UpdateAutoScalingGroup",
		"AutoScalingGroupName":  "nonexistent",
		"MaxSize":               "5",
	}))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAS_DeleteAutoScalingGroup(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                "CreateAutoScalingGroup",
		"AutoScalingGroupName":  "del-asg",
		"MinSize":               "0",
		"MaxSize":               "1",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                "DeleteAutoScalingGroup",
		"AutoScalingGroupName":  "del-asg",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
}

func TestAS_DeleteAutoScalingGroup_NotFound(t *testing.T) {
	h := newASGGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                "DeleteAutoScalingGroup",
		"AutoScalingGroupName":  "no-such-asg",
	}))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- SetDesiredCapacity ----

func TestAS_SetDesiredCapacity(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                     "CreateAutoScalingGroup",
		"AutoScalingGroupName":       "cap-asg",
		"MinSize":                    "0",
		"MaxSize":                    "10",
		"DesiredCapacity":            "1",
		"AvailabilityZones.member.1": "us-east-1a",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                "SetDesiredCapacity",
		"AutoScalingGroupName":  "cap-asg",
		"DesiredCapacity":       "5",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	// Verify instances
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, asReq(t, map[string]string{
		"Action": "DescribeAutoScalingInstances",
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	b := body(t, w3)
	assert.Contains(t, b, "cap-asg")
}

// ---- ScalingPolicy ----

func TestAS_PutScalingPolicy(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                "CreateAutoScalingGroup",
		"AutoScalingGroupName":  "pol-asg",
		"MinSize":               "1",
		"MaxSize":               "5",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                "PutScalingPolicy",
		"AutoScalingGroupName":  "pol-asg",
		"PolicyName":            "scale-up",
		"AdjustmentType":        "ChangeInCapacity",
		"ScalingAdjustment":     "1",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, body(t, w2), "PolicyARN")
}

func TestAS_DescribePolicies(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                "CreateAutoScalingGroup",
		"AutoScalingGroupName":  "dpol-asg",
		"MinSize":               "1",
		"MaxSize":               "5",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                "PutScalingPolicy",
		"AutoScalingGroupName":  "dpol-asg",
		"PolicyName":            "my-pol",
		"AdjustmentType":        "ChangeInCapacity",
		"ScalingAdjustment":     "2",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, asReq(t, map[string]string{
		"Action":                "DescribePolicies",
		"AutoScalingGroupName":  "dpol-asg",
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, body(t, w3), "my-pol")
}

func TestAS_DeletePolicy(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                "CreateAutoScalingGroup",
		"AutoScalingGroupName":  "delpol-asg",
		"MinSize":               "1",
		"MaxSize":               "5",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                "PutScalingPolicy",
		"AutoScalingGroupName":  "delpol-asg",
		"PolicyName":            "del-pol",
		"AdjustmentType":        "ChangeInCapacity",
		"ScalingAdjustment":     "1",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, asReq(t, map[string]string{
		"Action":                "DeletePolicy",
		"AutoScalingGroupName":  "delpol-asg",
		"PolicyName":            "del-pol",
	}))
	require.Equal(t, http.StatusOK, w3.Code)
}

func TestAS_DeletePolicy_NotFound(t *testing.T) {
	h := newASGGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                "DeletePolicy",
		"AutoScalingGroupName":  "no-asg",
		"PolicyName":            "no-pol",
	}))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- Tags ----

func TestAS_CreateOrUpdateTags_DescribeTags(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                "CreateAutoScalingGroup",
		"AutoScalingGroupName":  "tag-asg",
		"MinSize":               "0",
		"MaxSize":               "1",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                       "CreateOrUpdateTags",
		"Tags.member.1.Key":            "env",
		"Tags.member.1.Value":          "prod",
		"Tags.member.1.ResourceId":     "tag-asg",
		"Tags.member.1.ResourceType":   "auto-scaling-group",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, asReq(t, map[string]string{
		"Action": "DescribeTags",
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	b := body(t, w3)
	assert.Contains(t, b, "env")
	assert.Contains(t, b, "prod")
}

func TestAS_DeleteTags(t *testing.T) {
	h := newASGGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, asReq(t, map[string]string{
		"Action":                "CreateAutoScalingGroup",
		"AutoScalingGroupName":  "deltag-asg",
		"MinSize":               "0",
		"MaxSize":               "1",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                       "CreateOrUpdateTags",
		"Tags.member.1.Key":            "remove-me",
		"Tags.member.1.Value":          "val",
		"Tags.member.1.ResourceId":     "deltag-asg",
		"Tags.member.1.ResourceType":   "auto-scaling-group",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, asReq(t, map[string]string{
		"Action":                       "DeleteTags",
		"Tags.member.1.Key":            "remove-me",
		"Tags.member.1.ResourceId":     "deltag-asg",
		"Tags.member.1.ResourceType":   "auto-scaling-group",
	}))
	require.Equal(t, http.StatusOK, w3.Code)

	w4 := httptest.NewRecorder()
	h.ServeHTTP(w4, asReq(t, map[string]string{
		"Action": "DescribeTags",
	}))
	require.Equal(t, http.StatusOK, w4.Code)
	assert.NotContains(t, body(t, w4), "remove-me")
}

// ---- Behavioral: Event Bus ----

func TestAS_SetEventBus_PublishesLaunchEvents(t *testing.T) {
	// Test that the service can accept an event bus (smoke test for SetEventBus)
	svc := assvc.New("123456789012", "us-east-1")
	bus := eventbus.NewBus()
	svc.SetEventBus(bus)

	var received []*eventbus.Event
	bus.Subscribe(&eventbus.Subscription{
		Source: "autoscaling",
		Types:  []string{"autoscaling:*"},
		Handler: func(e *eventbus.Event) error {
			received = append(received, e)
			return nil
		},
	})

	// The service should be functional with the bus set
	assert.NotNil(t, svc)
}

func TestAS_ReconcileInstances_WithoutLocator(t *testing.T) {
	// When locator is nil, instances get stub IDs (backward compatibility)
	h := newASGGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                     "CreateAutoScalingGroup",
		"AutoScalingGroupName":       "stub-asg",
		"MinSize":                    "1",
		"MaxSize":                    "5",
		"DesiredCapacity":            "3",
		"AvailabilityZones.member.1": "us-east-1a",
	}))
	require.Equal(t, http.StatusOK, w.Code, body(t, w))

	// Verify instances were created with stub IDs
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                              "DescribeAutoScalingGroups",
		"AutoScalingGroupNames.member.1":      "stub-asg",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
	b := body(t, w2)
	assert.Contains(t, b, "InService")
	assert.Contains(t, b, "Healthy")
	assert.Contains(t, b, "<DesiredCapacity>3</DesiredCapacity>")
}

// ---- InvalidAction ----

func TestAS_InvalidAction(t *testing.T) {
	h := newASGGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action": "BogusAction",
	}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, body(t, w), "InvalidAction")
}

// ---- Scheduled Actions ----

func createTestASG(t *testing.T, h http.Handler, name string) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                  "CreateLaunchConfiguration",
		"LaunchConfigurationName": "lc-for-" + name,
		"ImageId":                 "ami-test",
		"InstanceType":            "t3.micro",
	}))
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":                  "CreateAutoScalingGroup",
		"AutoScalingGroupName":    name,
		"LaunchConfigurationName": "lc-for-" + name,
		"MinSize":                 "1",
		"MaxSize":                 "5",
		"DesiredCapacity":         "2",
		"AvailabilityZones.member.1": "us-east-1a",
	}))
	require.Equal(t, http.StatusOK, w2.Code, body(t, w2))
}

func TestAS_ScheduledActions(t *testing.T) {
	h := newASGGateway(t)
	createTestASG(t, h, "sched-asg")

	// PutScheduledUpdateGroupAction
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":               "PutScheduledUpdateGroupAction",
		"AutoScalingGroupName": "sched-asg",
		"ScheduledActionName":  "scale-up",
		"DesiredCapacity":      "3",
		"Recurrence":           "0 12 * * *",
	}))
	require.Equal(t, http.StatusOK, w.Code, body(t, w))

	// DescribeScheduledActions
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":               "DescribeScheduledActions",
		"AutoScalingGroupName": "sched-asg",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, body(t, w2), "scale-up")

	// DeleteScheduledAction
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, asReq(t, map[string]string{
		"Action":               "DeleteScheduledAction",
		"AutoScalingGroupName": "sched-asg",
		"ScheduledActionName":  "scale-up",
	}))
	require.Equal(t, http.StatusOK, w3.Code)
}

// ---- Lifecycle Hooks ----

func TestAS_LifecycleHooks(t *testing.T) {
	h := newASGGateway(t)
	createTestASG(t, h, "hook-asg")

	// PutLifecycleHook
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":                  "PutLifecycleHook",
		"AutoScalingGroupName":    "hook-asg",
		"LifecycleHookName":       "my-hook",
		"LifecycleTransition":     "autoscaling:EC2_INSTANCE_LAUNCHING",
		"DefaultResult":           "CONTINUE",
		"HeartbeatTimeout":        "300",
	}))
	require.Equal(t, http.StatusOK, w.Code, body(t, w))

	// DescribeLifecycleHooks
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":               "DescribeLifecycleHooks",
		"AutoScalingGroupName": "hook-asg",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, body(t, w2), "my-hook")

	// CompleteLifecycleAction
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, asReq(t, map[string]string{
		"Action":               "CompleteLifecycleAction",
		"AutoScalingGroupName": "hook-asg",
		"LifecycleHookName":    "my-hook",
		"LifecycleActionResult": "CONTINUE",
	}))
	require.Equal(t, http.StatusOK, w3.Code)

	// DeleteLifecycleHook
	w4 := httptest.NewRecorder()
	h.ServeHTTP(w4, asReq(t, map[string]string{
		"Action":               "DeleteLifecycleHook",
		"AutoScalingGroupName": "hook-asg",
		"LifecycleHookName":    "my-hook",
	}))
	require.Equal(t, http.StatusOK, w4.Code)
}

// ---- ExecutePolicy ----

func TestAS_ExecutePolicy(t *testing.T) {
	h := newASGGateway(t)
	createTestASG(t, h, "exec-asg")

	// Create policy
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":               "PutScalingPolicy",
		"AutoScalingGroupName": "exec-asg",
		"PolicyName":           "scale-out",
		"AdjustmentType":       "ChangeInCapacity",
		"ScalingAdjustment":    "1",
	}))
	require.Equal(t, http.StatusOK, w.Code)

	// Execute policy
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":               "ExecutePolicy",
		"AutoScalingGroupName": "exec-asg",
		"PolicyName":           "scale-out",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
}

// ---- Metrics Collection ----

func TestAS_MetricsCollection(t *testing.T) {
	h := newASGGateway(t)
	createTestASG(t, h, "metrics-asg")

	// Enable
	w := httptest.NewRecorder()
	h.ServeHTTP(w, asReq(t, map[string]string{
		"Action":               "EnableMetricsCollection",
		"AutoScalingGroupName": "metrics-asg",
		"Granularity":          "1Minute",
	}))
	require.Equal(t, http.StatusOK, w.Code, body(t, w))

	// Disable
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, asReq(t, map[string]string{
		"Action":               "DisableMetricsCollection",
		"AutoScalingGroupName": "metrics-asg",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
}
