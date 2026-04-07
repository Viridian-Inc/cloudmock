package scheduler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.SchedulerService {
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

// ---- Schedule tests ----

func TestCreateSchedule(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name":               "my-schedule",
		"ScheduleExpression": "rate(5 minutes)",
		"Target":             map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:my-func", "RoleArn": "arn:aws:iam::123456789012:role/scheduler-role"},
		"FlexibleTimeWindow": map[string]any{"Mode": "OFF"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := decode(t, resp)
	assert.Contains(t, m["ScheduleArn"].(string), "my-schedule")
}

func TestCreateScheduleMissingExpression(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{"Name": "my-schedule"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
}

func TestCreateScheduleDuplicate(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "my-schedule", "ScheduleExpression": "rate(5 minutes)",
	}))
	_, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "my-schedule", "ScheduleExpression": "rate(10 minutes)",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ConflictException", awsErr.Code)
}

func TestGetSchedule(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "my-schedule", "ScheduleExpression": "rate(5 minutes)",
	}))

	resp, err := s.HandleRequest(jsonCtx("GetSchedule", map[string]any{"Name": "my-schedule"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "my-schedule", m["Name"])
	assert.Equal(t, "rate(5 minutes)", m["ScheduleExpression"])
	assert.Equal(t, "ENABLED", m["State"])
	assert.Equal(t, "default", m["GroupName"])
}

func TestGetScheduleNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetSchedule", map[string]any{"Name": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

func TestListSchedules(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "sched-1", "ScheduleExpression": "rate(5 minutes)",
	}))
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "sched-2", "ScheduleExpression": "rate(10 minutes)",
	}))

	resp, err := s.HandleRequest(jsonCtx("ListSchedules", map[string]any{}))
	require.NoError(t, err)
	m := decode(t, resp)
	schedules := m["Schedules"].([]any)
	assert.Len(t, schedules, 2)
}

func TestUpdateSchedule(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "my-schedule", "ScheduleExpression": "rate(5 minutes)",
	}))

	resp, err := s.HandleRequest(jsonCtx("UpdateSchedule", map[string]any{
		"Name": "my-schedule", "Description": "updated",
		"ScheduleExpression": "rate(10 minutes)", "State": "DISABLED",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["ScheduleArn"])

	// Verify update
	getResp, _ := s.HandleRequest(jsonCtx("GetSchedule", map[string]any{"Name": "my-schedule"}))
	getData := decode(t, getResp)
	assert.Equal(t, "rate(10 minutes)", getData["ScheduleExpression"])
	assert.Equal(t, "DISABLED", getData["State"])
}

func TestDeleteSchedule(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "my-schedule", "ScheduleExpression": "rate(5 minutes)",
	}))

	_, err := s.HandleRequest(jsonCtx("DeleteSchedule", map[string]any{"Name": "my-schedule"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetSchedule", map[string]any{"Name": "my-schedule"}))
	require.Error(t, err)
}

func TestDeleteScheduleNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteSchedule", map[string]any{"Name": "nonexistent"}))
	require.Error(t, err)
}

// ---- Schedule Group tests ----

func TestCreateScheduleGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{"Name": "my-group"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Contains(t, m["ScheduleGroupArn"].(string), "my-group")
}

func TestCreateScheduleGroupDuplicate(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{"Name": "my-group"}))
	_, err := s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{"Name": "my-group"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ConflictException", awsErr.Code)
}

func TestGetScheduleGroup(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{"Name": "my-group"}))

	resp, err := s.HandleRequest(jsonCtx("GetScheduleGroup", map[string]any{"Name": "my-group"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "my-group", m["Name"])
	assert.Equal(t, "ACTIVE", m["State"])
}

func TestListScheduleGroups(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{"Name": "group-1"}))
	s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{"Name": "group-2"}))

	resp, err := s.HandleRequest(jsonCtx("ListScheduleGroups", map[string]any{}))
	require.NoError(t, err)
	m := decode(t, resp)
	groups := m["ScheduleGroups"].([]any)
	assert.GreaterOrEqual(t, len(groups), 3) // 2 created + 1 default
}

func TestDeleteScheduleGroup(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{"Name": "my-group"}))

	_, err := s.HandleRequest(jsonCtx("DeleteScheduleGroup", map[string]any{"Name": "my-group"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetScheduleGroup", map[string]any{"Name": "my-group"}))
	require.Error(t, err)
}

func TestDefaultGroupExists(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetScheduleGroup", map[string]any{"Name": "default"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "default", m["Name"])
}

// ---- Schedule in custom group ----

func TestScheduleInCustomGroup(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{"Name": "my-group"}))

	resp, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "my-schedule", "GroupName": "my-group",
		"ScheduleExpression": "rate(1 hour)",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Contains(t, m["ScheduleArn"].(string), "my-group")
}

// ---- Tagging ----

func TestTagSchedule(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "my-schedule", "ScheduleExpression": "rate(5 minutes)",
	}))
	schedARN := decode(t, createResp)["ScheduleArn"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": schedARN,
		"Tags":        map[string]any{"env": "prod"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": schedARN}))
	require.NoError(t, err)
	m := decode(t, resp)
	tags := m["Tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
}

func TestUntagSchedule(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "my-schedule", "ScheduleExpression": "rate(5 minutes)",
		"Tags": map[string]any{"env": "prod", "team": "alpha"},
	}))
	schedARN := decode(t, createResp)["ScheduleArn"].(string)

	_, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": schedARN,
		"TagKeys":     []string{"team"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": schedARN}))
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

// ---- Behavioral: Rate expression parsing ----

func TestParseRateExpression(t *testing.T) {
	s := newService()
	// Schedule with rate expression should track invocation even without locator.
	resp, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name":               "rate-schedule",
		"ScheduleExpression": "rate(5 minutes)",
		"Target":             map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:my-func", "RoleArn": "arn:aws:iam::123456789012:role/r"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Give goroutine time to fire
	time.Sleep(50 * time.Millisecond)

	// GetSchedule should show invocation count (attempted but failed due to no locator)
	getResp, err := s.HandleRequest(jsonCtx("GetSchedule", map[string]any{"Name": "rate-schedule"}))
	require.NoError(t, err)
	m := decode(t, getResp)
	// InvocationCount should be >= 1 (attempted)
	assert.GreaterOrEqual(t, int(m["InvocationCount"].(float64)), 1)
}

func TestScheduleWithLocator(t *testing.T) {
	// Create a mock lambda that records invocations
	invoked := false
	mockLambda := &mockTargetService{
		name: "lambda",
		handleFn: func(ctx *service.RequestContext) (*service.Response, error) {
			invoked = true
			return &service.Response{StatusCode: http.StatusOK, Body: map[string]any{"StatusCode": 200}, Format: service.FormatJSON}, nil
		},
	}

	locator := &mockLocator{services: map[string]service.Service{"lambda": mockLambda}}
	s := svc.NewWithLocator("123456789012", "us-east-1", locator)

	_, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name":               "invoke-schedule",
		"ScheduleExpression": "rate(1 minute)",
		"Target":             map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:my-func", "RoleArn": "arn:aws:iam::123456789012:role/r", "Input": `{"key":"value"}`},
	}))
	require.NoError(t, err)

	// Give goroutine time to fire
	time.Sleep(50 * time.Millisecond)

	assert.True(t, invoked, "Lambda target should have been invoked")
}

func TestScheduleOneTimeExpression(t *testing.T) {
	invoked := false
	mockLambda := &mockTargetService{
		name: "lambda",
		handleFn: func(ctx *service.RequestContext) (*service.Response, error) {
			invoked = true
			return &service.Response{StatusCode: http.StatusOK, Body: map[string]any{}, Format: service.FormatJSON}, nil
		},
	}
	locator := &mockLocator{services: map[string]service.Service{"lambda": mockLambda}}
	s := svc.NewWithLocator("123456789012", "us-east-1", locator)

	_, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name":               "onetime-schedule",
		"ScheduleExpression": "at(2025-01-01T00:00:00)",
		"Target":             map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:my-func", "RoleArn": "arn:aws:iam::123456789012:role/r"},
	}))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.True(t, invoked, "One-time schedule should fire immediately in mock mode")
}

func TestScheduleSQSTarget(t *testing.T) {
	invoked := false
	mockSQS := &mockTargetService{
		name: "sqs",
		handleFn: func(ctx *service.RequestContext) (*service.Response, error) {
			assert.Equal(t, "SendMessage", ctx.Action)
			invoked = true
			return &service.Response{StatusCode: http.StatusOK, Body: map[string]any{}, Format: service.FormatJSON}, nil
		},
	}
	locator := &mockLocator{services: map[string]service.Service{"sqs": mockSQS}}
	s := svc.NewWithLocator("123456789012", "us-east-1", locator)

	_, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name":               "sqs-schedule",
		"ScheduleExpression": "rate(5 minutes)",
		"Target":             map[string]any{"Arn": "arn:aws:sqs:us-east-1:123456789012:my-queue", "RoleArn": "arn:aws:iam::123456789012:role/r", "Input": "hello"},
	}))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.True(t, invoked, "SQS target should have been invoked")
}

func TestScheduleSNSTarget(t *testing.T) {
	invoked := false
	mockSNS := &mockTargetService{
		name: "sns",
		handleFn: func(ctx *service.RequestContext) (*service.Response, error) {
			assert.Equal(t, "Publish", ctx.Action)
			invoked = true
			return &service.Response{StatusCode: http.StatusOK, Body: map[string]any{}, Format: service.FormatJSON}, nil
		},
	}
	locator := &mockLocator{services: map[string]service.Service{"sns": mockSNS}}
	s := svc.NewWithLocator("123456789012", "us-east-1", locator)

	_, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name":               "sns-schedule",
		"ScheduleExpression": "rate(5 minutes)",
		"Target":             map[string]any{"Arn": "arn:aws:sns:us-east-1:123456789012:my-topic", "RoleArn": "arn:aws:iam::123456789012:role/r", "Input": "hello"},
	}))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.True(t, invoked, "SNS target should have been invoked")
}

func TestScheduleDisabledDoesNotFire(t *testing.T) {
	invoked := false
	mockLambda := &mockTargetService{
		name: "lambda",
		handleFn: func(ctx *service.RequestContext) (*service.Response, error) {
			invoked = true
			return &service.Response{StatusCode: http.StatusOK, Body: map[string]any{}, Format: service.FormatJSON}, nil
		},
	}
	locator := &mockLocator{services: map[string]service.Service{"lambda": mockLambda}}
	s := svc.NewWithLocator("123456789012", "us-east-1", locator)

	_, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name":               "disabled-schedule",
		"ScheduleExpression": "rate(5 minutes)",
		"State":              "DISABLED",
		"Target":             map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:my-func", "RoleArn": "arn:aws:iam::123456789012:role/r"},
	}))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.False(t, invoked, "Disabled schedule should not fire")
}

func TestScheduleCronExpression(t *testing.T) {
	invoked := false
	mockLambda := &mockTargetService{
		name: "lambda",
		handleFn: func(ctx *service.RequestContext) (*service.Response, error) {
			invoked = true
			return &service.Response{StatusCode: http.StatusOK, Body: map[string]any{}, Format: service.FormatJSON}, nil
		},
	}
	locator := &mockLocator{services: map[string]service.Service{"lambda": mockLambda}}
	s := svc.NewWithLocator("123456789012", "us-east-1", locator)

	_, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name":               "cron-schedule",
		"ScheduleExpression": "cron(0 12 * * ? *)",
		"Target":             map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:my-func", "RoleArn": "arn:aws:iam::123456789012:role/r"},
	}))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.True(t, invoked, "Cron schedule should fire once immediately in mock mode")
}

// ---- Additional coverage tests ----

func TestListSchedulesWithStateFilter(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "enabled-sched", "ScheduleExpression": "rate(5 minutes)", "State": "ENABLED",
		"Target": map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:f", "RoleArn": "arn:aws:iam::123456789012:role/r"},
		"FlexibleTimeWindow": map[string]any{"Mode": "OFF"},
	}))
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "disabled-sched", "ScheduleExpression": "rate(10 minutes)", "State": "DISABLED",
		"Target": map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:f", "RoleArn": "arn:aws:iam::123456789012:role/r"},
		"FlexibleTimeWindow": map[string]any{"Mode": "OFF"},
	}))

	resp, err := s.HandleRequest(jsonCtx("ListSchedules", map[string]any{"State": "ENABLED"}))
	require.NoError(t, err)
	m := decode(t, resp)
	schedules := m["Schedules"].([]any)
	assert.Len(t, schedules, 1)
}

func TestListSchedulesNamePrefixFilter(t *testing.T) {
	s := newService()
	for _, name := range []string{"prod-1", "prod-2", "dev-1"} {
		s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
			"Name": name, "ScheduleExpression": "rate(5 minutes)",
			"Target": map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:f", "RoleArn": "arn:aws:iam::123456789012:role/r"},
			"FlexibleTimeWindow": map[string]any{"Mode": "OFF"},
		}))
	}

	resp, err := s.HandleRequest(jsonCtx("ListSchedules", map[string]any{"NamePrefix": "prod"}))
	require.NoError(t, err)
	schedules := decode(t, resp)["Schedules"].([]any)
	assert.Len(t, schedules, 2)
}

func TestScheduleARNFormat(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "arn-sched", "ScheduleExpression": "rate(5 minutes)",
		"Target": map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:f", "RoleArn": "arn:aws:iam::123456789012:role/r"},
		"FlexibleTimeWindow": map[string]any{"Mode": "OFF"},
	}))
	require.NoError(t, err)
	body := decode(t, resp)
	assert.Contains(t, body["ScheduleArn"].(string), "arn:aws:scheduler:")
	assert.Contains(t, body["ScheduleArn"].(string), "arn-sched")
}

func TestScheduleGroupARNFormat(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{
		"Name": "arn-group",
	}))
	require.NoError(t, err)
	body := decode(t, resp)
	assert.Contains(t, body["ScheduleGroupArn"].(string), "arn:aws:scheduler:")
	assert.Contains(t, body["ScheduleGroupArn"].(string), "arn-group")
}

func TestUpdateScheduleExpression(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "upd-expr-sched", "ScheduleExpression": "rate(5 minutes)",
		"Target": map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:f", "RoleArn": "arn:aws:iam::123456789012:role/r"},
		"FlexibleTimeWindow": map[string]any{"Mode": "OFF"},
	}))

	_, err := s.HandleRequest(jsonCtx("UpdateSchedule", map[string]any{
		"Name": "upd-expr-sched", "ScheduleExpression": "rate(10 minutes)",
		"Target": map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:f", "RoleArn": "arn:aws:iam::123456789012:role/r"},
		"FlexibleTimeWindow": map[string]any{"Mode": "OFF"},
	}))
	require.NoError(t, err)

	// Verify via GetSchedule
	getResp, err := s.HandleRequest(jsonCtx("GetSchedule", map[string]any{"Name": "upd-expr-sched"}))
	require.NoError(t, err)
	getBody := decode(t, getResp)
	assert.Equal(t, "rate(10 minutes)", getBody["ScheduleExpression"])
}

func TestDeleteScheduleGroupWithSchedules(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateScheduleGroup", map[string]any{"Name": "del-group"}))
	s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name": "del-group-sched", "GroupName": "del-group", "ScheduleExpression": "rate(5 minutes)",
		"Target": map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:f", "RoleArn": "arn:aws:iam::123456789012:role/r"},
		"FlexibleTimeWindow": map[string]any{"Mode": "OFF"},
	}))

	// Delete group should also delete its schedules
	_, err := s.HandleRequest(jsonCtx("DeleteScheduleGroup", map[string]any{"Name": "del-group"}))
	require.NoError(t, err)

	// Group should be gone
	_, err = s.HandleRequest(jsonCtx("GetScheduleGroup", map[string]any{"Name": "del-group"}))
	require.Error(t, err)
}

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "scheduler", s.Name())
	assert.NoError(t, s.HealthCheck())
}

func TestScheduleFlexibleTimeWindow(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateSchedule", map[string]any{
		"Name":               "flex-sched",
		"ScheduleExpression": "rate(15 minutes)",
		"FlexibleTimeWindow": map[string]any{
			"Mode":                   "FLEXIBLE",
			"MaximumWindowInMinutes": float64(10),
		},
		"Target": map[string]any{"Arn": "arn:aws:lambda:us-east-1:123456789012:function:f", "RoleArn": "arn:aws:iam::123456789012:role/r"},
	}))
	require.NoError(t, err)
	body := decode(t, resp)
	assert.NotEmpty(t, body["ScheduleArn"])

	// Get and verify flexible time window
	getResp, err := s.HandleRequest(jsonCtx("GetSchedule", map[string]any{"Name": "flex-sched"}))
	require.NoError(t, err)
	getBody := decode(t, getResp)
	ftw := getBody["FlexibleTimeWindow"].(map[string]any)
	assert.Equal(t, "FLEXIBLE", ftw["Mode"])
	assert.Equal(t, float64(10), ftw["MaximumWindowInMinutes"])
}

// ---- Test helpers for behavioral tests ----

type mockTargetService struct {
	name     string
	handleFn func(ctx *service.RequestContext) (*service.Response, error)
}

func (m *mockTargetService) Name() string                 { return m.name }
func (m *mockTargetService) Actions() []service.Action     { return nil }
func (m *mockTargetService) HealthCheck() error            { return nil }
func (m *mockTargetService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return m.handleFn(ctx)
}

type mockLocator struct {
	services map[string]service.Service
}

func (l *mockLocator) Lookup(name string) (service.Service, error) {
	svc, ok := l.services[name]
	if !ok {
		return nil, fmt.Errorf("service %q not found", name)
	}
	return svc, nil
}
