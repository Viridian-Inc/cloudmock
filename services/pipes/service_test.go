package pipes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/pipes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.PipesService {
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

func mustCreatePipe(t *testing.T, s *svc.PipesService, name string) map[string]any {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreatePipe", map[string]any{
		"Name":    name,
		"Source":  "arn:aws:sqs:us-east-1:123456789012:my-queue",
		"Target":  "arn:aws:lambda:us-east-1:123456789012:function:my-func",
		"RoleArn": "arn:aws:iam::123456789012:role/pipe-role",
	}))
	require.NoError(t, err)
	return decode(t, resp)
}

// ---- Create tests ----

func TestCreatePipe(t *testing.T) {
	s := newService()
	m := mustCreatePipe(t, s, "my-pipe")
	assert.Equal(t, "my-pipe", m["Name"])
	assert.Contains(t, m["Arn"].(string), "my-pipe")
	assert.Equal(t, "RUNNING", m["DesiredState"])
}

func TestCreatePipeLifecycleInstant(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "my-pipe")

	// With lifecycle delays disabled, pipe should be RUNNING instantly
	resp, err := s.HandleRequest(jsonCtx("DescribePipe", map[string]any{"Name": "my-pipe"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "RUNNING", m["CurrentState"])
}

func TestCreatePipeDuplicate(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "my-pipe")
	_, err := s.HandleRequest(jsonCtx("CreatePipe", map[string]any{
		"Name": "my-pipe", "Source": "arn:aws:sqs:us-east-1:123456789012:q",
		"Target": "arn:aws:lambda:us-east-1:123456789012:function:f",
		"RoleArn": "arn:aws:iam::123456789012:role/r",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ConflictException", awsErr.Code)
}

func TestCreatePipeMissingFields(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreatePipe", map[string]any{"Name": "my-pipe"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Code)
}

// ---- Describe tests ----

func TestDescribePipe(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "my-pipe")

	resp, err := s.HandleRequest(jsonCtx("DescribePipe", map[string]any{"Name": "my-pipe"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "my-pipe", m["Name"])
	assert.NotEmpty(t, m["CreationTime"])
}

func TestDescribePipeNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribePipe", map[string]any{"Name": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "NotFoundException", awsErr.Code)
}

// ---- List tests ----

func TestListPipes(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "pipe-1")
	mustCreatePipe(t, s, "pipe-2")

	resp, err := s.HandleRequest(jsonCtx("ListPipes", map[string]any{}))
	require.NoError(t, err)
	m := decode(t, resp)
	pipes := m["Pipes"].([]any)
	assert.Len(t, pipes, 2)
}

func TestListPipesWithFilter(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "alpha-pipe")
	mustCreatePipe(t, s, "beta-pipe")

	resp, err := s.HandleRequest(jsonCtx("ListPipes", map[string]any{"NamePrefix": "alpha"}))
	require.NoError(t, err)
	m := decode(t, resp)
	pipes := m["Pipes"].([]any)
	assert.Len(t, pipes, 1)
}

// ---- Update tests ----

func TestUpdatePipe(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "my-pipe")

	resp, err := s.HandleRequest(jsonCtx("UpdatePipe", map[string]any{
		"Name": "my-pipe", "Description": "updated",
		"RoleArn": "arn:aws:iam::123456789012:role/new-role",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "my-pipe", m["Name"])
}

func TestUpdatePipeNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("UpdatePipe", map[string]any{
		"Name": "nonexistent", "RoleArn": "arn:aws:iam::123456789012:role/r",
	}))
	require.Error(t, err)
}

// ---- Delete tests ----

func TestDeletePipe(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "my-pipe")

	resp, err := s.HandleRequest(jsonCtx("DeletePipe", map[string]any{"Name": "my-pipe"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "my-pipe", m["Name"])

	// Verify deleted
	_, err = s.HandleRequest(jsonCtx("DescribePipe", map[string]any{"Name": "my-pipe"}))
	require.Error(t, err)
}

func TestDeletePipeNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeletePipe", map[string]any{"Name": "nonexistent"}))
	require.Error(t, err)
}

// ---- Stop/Start lifecycle ----

func TestStopAndStartPipe(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "my-pipe")

	// Pipe should be RUNNING (instant lifecycle)
	descResp, _ := s.HandleRequest(jsonCtx("DescribePipe", map[string]any{"Name": "my-pipe"}))
	assert.Equal(t, "RUNNING", decode(t, descResp)["CurrentState"])

	// Stop it
	stopResp, err := s.HandleRequest(jsonCtx("StopPipe", map[string]any{"Name": "my-pipe"}))
	require.NoError(t, err)
	stopData := decode(t, stopResp)
	assert.Equal(t, "STOPPED", stopData["DesiredState"])

	// After instant transitions, it should be STOPPED
	descResp2, _ := s.HandleRequest(jsonCtx("DescribePipe", map[string]any{"Name": "my-pipe"}))
	assert.Equal(t, "STOPPED", decode(t, descResp2)["CurrentState"])

	// Start it again
	startResp, err := s.HandleRequest(jsonCtx("StartPipe", map[string]any{"Name": "my-pipe"}))
	require.NoError(t, err)
	startData := decode(t, startResp)
	assert.Equal(t, "RUNNING", startData["DesiredState"])
}

func TestStopPipeNotRunning(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "my-pipe")

	// Stop it first
	s.HandleRequest(jsonCtx("StopPipe", map[string]any{"Name": "my-pipe"}))

	// Try to stop again — not in RUNNING state
	_, err := s.HandleRequest(jsonCtx("StopPipe", map[string]any{"Name": "my-pipe"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ConflictException", awsErr.Code)
}

// ---- Tagging ----

func TestTagPipe(t *testing.T) {
	s := newService()
	createData := mustCreatePipe(t, s, "my-pipe")
	arn := createData["Arn"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags":        map[string]any{"env": "prod"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	tags := decode(t, resp)["Tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
}

func TestUntagPipe(t *testing.T) {
	s := newService()
	createData := mustCreatePipe(t, s, "my-pipe")
	arn := createData["Arn"].(string)

	s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags":        map[string]any{"env": "prod", "team": "alpha"},
	}))

	_, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []string{"team"},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	tags := decode(t, resp)["Tags"].(map[string]any)
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

// ---- Behavioral: Source polling and target forwarding ----

func TestPipePollingWithLocator(t *testing.T) {
	invoked := false
	mockSQS := &mockTargetService{
		name: "sqs",
		handleFn: func(ctx *service.RequestContext) (*service.Response, error) {
			if ctx.Action == "ReceiveMessage" {
				return &service.Response{StatusCode: 200, Body: map[string]any{
					"Messages": []any{map[string]any{"Body": "hello", "MessageId": "msg-1"}},
				}, Format: service.FormatJSON}, nil
			}
			return &service.Response{StatusCode: 200, Body: map[string]any{}, Format: service.FormatJSON}, nil
		},
	}
	mockLambda := &mockTargetService{
		name: "lambda",
		handleFn: func(ctx *service.RequestContext) (*service.Response, error) {
			assert.Equal(t, "Invoke", ctx.Action)
			invoked = true
			return &service.Response{StatusCode: 200, Body: map[string]any{}, Format: service.FormatJSON}, nil
		},
	}
	locator := &mockLocator{services: map[string]service.Service{"sqs": mockSQS, "lambda": mockLambda}}
	s := svc.NewWithLocator("123456789012", "us-east-1", locator)

	_, err := s.HandleRequest(jsonCtx("CreatePipe", map[string]any{
		"Name":    "poll-pipe",
		"Source":  "arn:aws:sqs:us-east-1:123456789012:my-queue",
		"Target":  "arn:aws:lambda:us-east-1:123456789012:function:my-func",
		"RoleArn": "arn:aws:iam::123456789012:role/pipe-role",
	}))
	require.NoError(t, err)

	// Wait for polling goroutine
	time.Sleep(100 * time.Millisecond)

	assert.True(t, invoked, "Lambda target should have been invoked with SQS message")
}

func TestPipeWithoutLocatorGraceful(t *testing.T) {
	// Without locator, pipe should still create and transition to RUNNING.
	s := newService()
	mustCreatePipe(t, s, "no-locator-pipe")

	resp, err := s.HandleRequest(jsonCtx("DescribePipe", map[string]any{"Name": "no-locator-pipe"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "RUNNING", m["CurrentState"])
}

func TestPipeStartStopPolling(t *testing.T) {
	pollCount := 0
	mockSQS := &mockTargetService{
		name: "sqs",
		handleFn: func(ctx *service.RequestContext) (*service.Response, error) {
			if ctx.Action == "ReceiveMessage" {
				pollCount++
			}
			return &service.Response{StatusCode: 200, Body: map[string]any{"Messages": []any{}}, Format: service.FormatJSON}, nil
		},
	}
	locator := &mockLocator{services: map[string]service.Service{"sqs": mockSQS}}
	s := svc.NewWithLocator("123456789012", "us-east-1", locator)

	_, err := s.HandleRequest(jsonCtx("CreatePipe", map[string]any{
		"Name":    "lifecycle-pipe",
		"Source":  "arn:aws:sqs:us-east-1:123456789012:my-queue",
		"Target":  "arn:aws:lambda:us-east-1:123456789012:function:my-func",
		"RoleArn": "arn:aws:iam::123456789012:role/pipe-role",
	}))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.GreaterOrEqual(t, pollCount, 1, "Should have polled at least once")

	// Stop the pipe
	_, err = s.HandleRequest(jsonCtx("StopPipe", map[string]any{"Name": "lifecycle-pipe"}))
	require.NoError(t, err)

	// Start it again
	_, err = s.HandleRequest(jsonCtx("StartPipe", map[string]any{"Name": "lifecycle-pipe"}))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.GreaterOrEqual(t, pollCount, 2, "Should have polled again after restart")
}

// ---- Additional coverage tests ----

func TestCreatePipeWithDesiredStateStopped(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreatePipe", map[string]any{
		"Name":         "stopped-pipe",
		"Source":       "arn:aws:sqs:us-east-1:123456789012:my-queue",
		"Target":       "arn:aws:lambda:us-east-1:123456789012:function:my-func",
		"RoleArn":      "arn:aws:iam::123456789012:role/pipe-role",
		"DesiredState": "STOPPED",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "STOPPED", m["DesiredState"])
}

func TestCreatePipeWithDescription(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreatePipe", map[string]any{
		"Name":        "desc-pipe",
		"Description": "A test pipe with description",
		"Source":      "arn:aws:sqs:us-east-1:123456789012:my-queue",
		"Target":      "arn:aws:lambda:us-east-1:123456789012:function:my-func",
		"RoleArn":     "arn:aws:iam::123456789012:role/pipe-role",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "A test pipe with description", m["Description"])
}

func TestCreatePipeWithEnrichment(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreatePipe", map[string]any{
		"Name":       "enrich-pipe",
		"Source":     "arn:aws:sqs:us-east-1:123456789012:my-queue",
		"Target":     "arn:aws:lambda:us-east-1:123456789012:function:my-func",
		"RoleArn":    "arn:aws:iam::123456789012:role/pipe-role",
		"Enrichment": "arn:aws:lambda:us-east-1:123456789012:function:enrichment-func",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "enrich-pipe", m["Name"])
}

func TestListPipesCurrentStateFilter(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "pipe-a")
	mustCreatePipe(t, s, "pipe-b")

	resp, err := s.HandleRequest(jsonCtx("ListPipes", map[string]any{
		"CurrentState": "RUNNING",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	pipes := m["Pipes"].([]any)
	// With instant lifecycle all should be RUNNING
	assert.Len(t, pipes, 2)
}

func TestUpdatePipeDesiredState(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "state-pipe")

	resp, err := s.HandleRequest(jsonCtx("UpdatePipe", map[string]any{
		"Name":         "state-pipe",
		"DesiredState": "STOPPED",
		"RoleArn":      "arn:aws:iam::123456789012:role/pipe-role",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "state-pipe", m["Name"])
}

func TestListTagsForPipeNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{
		"ResourceArn": "arn:aws:pipes:us-east-1:123456789012:pipe/nonexistent",
	}))
	require.Error(t, err)
}

func TestPipeARNContainsName(t *testing.T) {
	s := newService()
	m := mustCreatePipe(t, s, "arn-test-pipe")
	assert.Contains(t, m["Arn"].(string), "arn-test-pipe")
	assert.Contains(t, m["Arn"].(string), "arn:aws:pipes:")
}

func TestDeletePipeReturnsState(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "del-state-pipe")

	resp, err := s.HandleRequest(jsonCtx("DeletePipe", map[string]any{"Name": "del-state-pipe"}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "DELETING", m["CurrentState"])
}

func TestUpdatePipeTarget(t *testing.T) {
	s := newService()
	mustCreatePipe(t, s, "target-pipe")

	newTarget := "arn:aws:lambda:us-east-1:123456789012:function:new-func"
	resp, err := s.HandleRequest(jsonCtx("UpdatePipe", map[string]any{
		"Name":    "target-pipe",
		"Target":  newTarget,
		"RoleArn": "arn:aws:iam::123456789012:role/pipe-role",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.Equal(t, "target-pipe", m["Name"])

	// Verify target updated
	descResp, _ := s.HandleRequest(jsonCtx("DescribePipe", map[string]any{"Name": "target-pipe"}))
	descM := decode(t, descResp)
	assert.Equal(t, newTarget, descM["Target"])
}

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "pipes", s.Name())
	assert.NoError(t, s.HealthCheck())
}

// ---- Test helpers for behavioral tests ----

type mockTargetService struct {
	name     string
	handleFn func(ctx *service.RequestContext) (*service.Response, error)
}

func (m *mockTargetService) Name() string             { return m.name }
func (m *mockTargetService) Actions() []service.Action { return nil }
func (m *mockTargetService) HealthCheck() error        { return nil }
func (m *mockTargetService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return m.handleFn(ctx)
}

type mockLocator struct {
	services map[string]service.Service
}

func (l *mockLocator) Lookup(name string) (service.Service, error) {
	sv, ok := l.services[name]
	if !ok {
		return nil, fmt.Errorf("service %q not found", name)
	}
	return sv, nil
}
