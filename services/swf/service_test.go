package swf_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/swf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.SWFService {
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

func mustRegisterDomain(t *testing.T, s *svc.SWFService, name string) {
	t.Helper()
	_, err := s.HandleRequest(jsonCtx("RegisterDomain", map[string]any{
		"name": name, "workflowExecutionRetentionPeriodInDays": "30",
	}))
	require.NoError(t, err)
}

func mustRegisterWorkflowType(t *testing.T, s *svc.SWFService, domain, name, version string) {
	t.Helper()
	_, err := s.HandleRequest(jsonCtx("RegisterWorkflowType", map[string]any{
		"domain": domain, "name": name, "version": version,
		"defaultTaskList": map[string]any{"name": "default"},
	}))
	require.NoError(t, err)
}

// ---- Domain tests ----

func TestRegisterDomain(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("RegisterDomain", map[string]any{
		"name": "my-domain", "workflowExecutionRetentionPeriodInDays": "30",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRegisterDomainDuplicate(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	_, err := s.HandleRequest(jsonCtx("RegisterDomain", map[string]any{
		"name": "my-domain", "workflowExecutionRetentionPeriodInDays": "30",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "DomainAlreadyExistsFault", awsErr.Code)
}

func TestDescribeDomain(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")

	resp, err := s.HandleRequest(jsonCtx("DescribeDomain", map[string]any{"name": "my-domain"}))
	require.NoError(t, err)
	m := decode(t, resp)
	info := m["domainInfo"].(map[string]any)
	assert.Equal(t, "my-domain", info["name"])
	assert.Equal(t, "REGISTERED", info["status"])
}

func TestDescribeDomainNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeDomain", map[string]any{"name": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "UnknownResourceFault", awsErr.Code)
}

func TestListDomains(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "domain-1")
	mustRegisterDomain(t, s, "domain-2")

	resp, err := s.HandleRequest(jsonCtx("ListDomains", map[string]any{"registrationStatus": "REGISTERED"}))
	require.NoError(t, err)
	m := decode(t, resp)
	domains := m["domainInfos"].([]any)
	assert.Len(t, domains, 2)
}

func TestDeprecateDomain(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")

	_, err := s.HandleRequest(jsonCtx("DeprecateDomain", map[string]any{"name": "my-domain"}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("DescribeDomain", map[string]any{"name": "my-domain"}))
	m := decode(t, resp)
	info := m["domainInfo"].(map[string]any)
	assert.Equal(t, "DEPRECATED", info["status"])
}

// ---- Workflow Type tests ----

func TestRegisterWorkflowType(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")

	_, err := s.HandleRequest(jsonCtx("RegisterWorkflowType", map[string]any{
		"domain": "my-domain", "name": "my-workflow", "version": "1.0",
	}))
	require.NoError(t, err)
}

func TestRegisterWorkflowTypeDuplicate(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	_, err := s.HandleRequest(jsonCtx("RegisterWorkflowType", map[string]any{
		"domain": "my-domain", "name": "my-workflow", "version": "1.0",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "TypeAlreadyExistsFault", awsErr.Code)
}

func TestDescribeWorkflowType(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	resp, err := s.HandleRequest(jsonCtx("DescribeWorkflowType", map[string]any{
		"domain":       "my-domain",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	typeInfo := m["typeInfo"].(map[string]any)
	assert.Equal(t, "REGISTERED", typeInfo["status"])
}

func TestListWorkflowTypes(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "wf-1", "1.0")
	mustRegisterWorkflowType(t, s, "my-domain", "wf-2", "1.0")

	resp, err := s.HandleRequest(jsonCtx("ListWorkflowTypes", map[string]any{
		"domain": "my-domain", "registrationStatus": "REGISTERED",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	types := m["typeInfos"].([]any)
	assert.Len(t, types, 2)
}

func TestDeprecateWorkflowType(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	_, err := s.HandleRequest(jsonCtx("DeprecateWorkflowType", map[string]any{
		"domain":       "my-domain",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("DescribeWorkflowType", map[string]any{
		"domain": "my-domain", "workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
	}))
	typeInfo := decode(t, resp)["typeInfo"].(map[string]any)
	assert.Equal(t, "DEPRECATED", typeInfo["status"])
}

// ---- Activity Type tests ----

func TestRegisterActivityType(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")

	_, err := s.HandleRequest(jsonCtx("RegisterActivityType", map[string]any{
		"domain": "my-domain", "name": "my-activity", "version": "1.0",
	}))
	require.NoError(t, err)
}

func TestListActivityTypes(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	s.HandleRequest(jsonCtx("RegisterActivityType", map[string]any{
		"domain": "my-domain", "name": "act-1", "version": "1.0",
	}))
	s.HandleRequest(jsonCtx("RegisterActivityType", map[string]any{
		"domain": "my-domain", "name": "act-2", "version": "1.0",
	}))

	resp, err := s.HandleRequest(jsonCtx("ListActivityTypes", map[string]any{
		"domain": "my-domain", "registrationStatus": "REGISTERED",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	types := m["typeInfos"].([]any)
	assert.Len(t, types, 2)
}

// ---- Workflow Execution tests ----

func TestStartWorkflowExecution(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	resp, err := s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
		"taskList":     map[string]any{"name": "default"},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["runId"])
}

func TestStartWorkflowExecutionDuplicate(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
	}))
	_, err := s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "WorkflowExecutionAlreadyStartedFault", awsErr.Code)
}

func TestDescribeWorkflowExecution(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	startResp, _ := s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
	}))
	runID := decode(t, startResp)["runId"].(string)

	resp, err := s.HandleRequest(jsonCtx("DescribeWorkflowExecution", map[string]any{
		"domain": "my-domain", "execution": map[string]any{"workflowId": "exec-1", "runId": runID},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	info := m["executionInfo"].(map[string]any)
	assert.Equal(t, "OPEN", info["executionStatus"])
}

func TestListOpenWorkflowExecutions(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
	}))

	resp, err := s.HandleRequest(jsonCtx("ListOpenWorkflowExecutions", map[string]any{
		"domain":          "my-domain",
		"startTimeFilter": map[string]any{"oldestDate": 0},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	execs := m["executionInfos"].([]any)
	assert.Len(t, execs, 1)
}

func TestTerminateWorkflowExecution(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	startResp, _ := s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
	}))
	runID := decode(t, startResp)["runId"].(string)

	_, err := s.HandleRequest(jsonCtx("TerminateWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1", "runId": runID,
		"reason": "test termination",
	}))
	require.NoError(t, err)

	// Verify it's closed
	resp, _ := s.HandleRequest(jsonCtx("ListClosedWorkflowExecutions", map[string]any{
		"domain":          "my-domain",
		"startTimeFilter": map[string]any{"oldestDate": 0},
	}))
	execs := decode(t, resp)["executionInfos"].([]any)
	assert.Len(t, execs, 1)
	assert.Equal(t, "TERMINATED", execs[0].(map[string]any)["executionStatus"])
}

// ---- Decision Task tests ----

func TestPollForDecisionTask(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
		"taskList":     map[string]any{"name": "default"},
	}))

	resp, err := s.HandleRequest(jsonCtx("PollForDecisionTask", map[string]any{
		"domain": "my-domain", "taskList": map[string]any{"name": "default"},
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	assert.NotEmpty(t, m["taskToken"])
}

func TestRespondDecisionTaskCompleted(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	startResp, _ := s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
		"taskList":     map[string]any{"name": "default"},
	}))
	runID := decode(t, startResp)["runId"].(string)

	// Poll for the decision task
	pollResp, _ := s.HandleRequest(jsonCtx("PollForDecisionTask", map[string]any{
		"domain": "my-domain", "taskList": map[string]any{"name": "default"},
	}))
	taskToken := decode(t, pollResp)["taskToken"].(string)

	// Complete the workflow
	_, err := s.HandleRequest(jsonCtx("RespondDecisionTaskCompleted", map[string]any{
		"taskToken": taskToken,
		"decisions": []map[string]any{{"decisionType": "CompleteWorkflowExecution"}},
	}))
	require.NoError(t, err)

	// Verify it's completed
	descResp, _ := s.HandleRequest(jsonCtx("DescribeWorkflowExecution", map[string]any{
		"domain": "my-domain", "execution": map[string]any{"workflowId": "exec-1", "runId": runID},
	}))
	info := decode(t, descResp)["executionInfo"].(map[string]any)
	assert.Equal(t, "COMPLETED", info["executionStatus"])
}

// ---- Signal test ----

func TestSignalWorkflowExecution(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	mustRegisterWorkflowType(t, s, "my-domain", "my-workflow", "1.0")

	s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1",
		"workflowType": map[string]any{"name": "my-workflow", "version": "1.0"},
	}))

	_, err := s.HandleRequest(jsonCtx("SignalWorkflowExecution", map[string]any{
		"domain": "my-domain", "workflowId": "exec-1", "signalName": "my-signal",
	}))
	require.NoError(t, err)
}

// ---- Tagging ----

func TestTagDomain(t *testing.T) {
	s := newService()
	mustRegisterDomain(t, s, "my-domain")
	arn := "arn:aws:swf:us-east-1:123456789012:/domain/my-domain"

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"resourceArn": arn,
		"tags":        []map[string]any{{"key": "env", "value": "prod"}},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	require.NoError(t, err)
	m := decode(t, resp)
	tags := m["tags"].([]any)
	assert.Len(t, tags, 1)
}

func TestUntagDomain(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("RegisterDomain", map[string]any{
		"name": "my-domain", "workflowExecutionRetentionPeriodInDays": "30",
		"tags": []map[string]any{{"key": "env", "value": "prod"}, {"key": "team", "value": "alpha"}},
	}))
	arn := "arn:aws:swf:us-east-1:123456789012:/domain/my-domain"

	_, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []string{"team"},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	tags := decode(t, resp)["tags"].([]any)
	assert.Len(t, tags, 1)
}

// ---- Invalid action ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "UnknownOperationException", awsErr.Code)
}

// ---- Missing required fields ----

func TestRegisterDomainMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("RegisterDomain", map[string]any{}))
	require.Error(t, err)
}

func TestStartWorkflowMissingFields(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain": "my-domain",
	}))
	require.Error(t, err)
}

// ---- Behavioral: Workflow history events ----

func TestWorkflowExecutionHistory(t *testing.T) {
	s := newService()

	// Setup
	s.HandleRequest(jsonCtx("RegisterDomain", map[string]any{
		"name": "hist-domain", "workflowExecutionRetentionPeriodInDays": "7",
	}))
	s.HandleRequest(jsonCtx("RegisterWorkflowType", map[string]any{
		"domain": "hist-domain", "name": "hist-wf", "version": "1.0",
		"defaultTaskList": map[string]any{"name": "hist-tasks"},
	}))

	// Start workflow
	resp, err := s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain":     "hist-domain",
		"workflowId": "hist-exec",
		"workflowType": map[string]any{"name": "hist-wf", "version": "1.0"},
		"input":      "test-input",
	}))
	require.NoError(t, err)
	m := decode(t, resp)
	runID := m["runId"].(string)

	// Get history
	histResp, err := s.HandleRequest(jsonCtx("GetWorkflowExecutionHistory", map[string]any{
		"domain": "hist-domain",
		"execution": map[string]any{"workflowId": "hist-exec", "runId": runID},
	}))
	require.NoError(t, err)
	histData := decode(t, histResp)
	events := histData["events"].([]any)

	// Should have at least WorkflowExecutionStarted and DecisionTaskScheduled
	assert.GreaterOrEqual(t, len(events), 2)
	assert.Equal(t, "WorkflowExecutionStarted", events[0].(map[string]any)["eventType"])
	assert.Equal(t, "DecisionTaskScheduled", events[1].(map[string]any)["eventType"])
}

// ---- Behavioral: Decision completes workflow and records events ----

func TestDecisionCompletesWorkflowHistory(t *testing.T) {
	s := newService()

	s.HandleRequest(jsonCtx("RegisterDomain", map[string]any{
		"name": "dec-domain", "workflowExecutionRetentionPeriodInDays": "7",
	}))
	s.HandleRequest(jsonCtx("RegisterWorkflowType", map[string]any{
		"domain": "dec-domain", "name": "dec-wf", "version": "1.0",
		"defaultTaskList": map[string]any{"name": "dec-tasks"},
	}))

	startResp, _ := s.HandleRequest(jsonCtx("StartWorkflowExecution", map[string]any{
		"domain":     "dec-domain",
		"workflowId": "dec-exec",
		"workflowType": map[string]any{"name": "dec-wf", "version": "1.0"},
	}))
	runID := decode(t, startResp)["runId"].(string)

	// Poll for decision task
	pollResp, _ := s.HandleRequest(jsonCtx("PollForDecisionTask", map[string]any{
		"domain":   "dec-domain",
		"taskList": map[string]any{"name": "dec-tasks"},
	}))
	pollData := decode(t, pollResp)
	taskToken := pollData["taskToken"].(string)

	// Complete the workflow via decision
	_, err := s.HandleRequest(jsonCtx("RespondDecisionTaskCompleted", map[string]any{
		"taskToken": taskToken,
		"decisions": []any{
			map[string]any{"decisionType": "CompleteWorkflowExecution"},
		},
	}))
	require.NoError(t, err)

	// Verify execution is completed
	descResp, _ := s.HandleRequest(jsonCtx("DescribeWorkflowExecution", map[string]any{
		"domain":    "dec-domain",
		"execution": map[string]any{"workflowId": "dec-exec", "runId": runID},
	}))
	descData := decode(t, descResp)
	assert.Equal(t, "COMPLETED", descData["executionInfo"].(map[string]any)["executionStatus"])

	// Check history has completion event
	histResp, _ := s.HandleRequest(jsonCtx("GetWorkflowExecutionHistory", map[string]any{
		"domain": "dec-domain",
		"execution": map[string]any{"workflowId": "dec-exec", "runId": runID},
	}))
	histData := decode(t, histResp)
	events := histData["events"].([]any)

	// Should have: Started, DecisionTaskScheduled, DecisionTaskCompleted, WorkflowExecutionCompleted
	assert.GreaterOrEqual(t, len(events), 4)
	lastEvent := events[len(events)-1].(map[string]any)
	assert.Equal(t, "WorkflowExecutionCompleted", lastEvent["eventType"])
}
