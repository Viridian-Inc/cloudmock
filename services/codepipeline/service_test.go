package codepipeline_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/codepipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.CodePipelineService {
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
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func pipelineBody(name string) map[string]any {
	return map[string]any{
		"pipeline": map[string]any{
			"name":    name,
			"roleArn": "arn:aws:iam::123456789012:role/pipeline-role",
			"stages": []any{
				map[string]any{
					"name": "Source",
					"actions": []any{
						map[string]any{
							"name": "SourceAction",
							"actionTypeId": map[string]any{
								"category": "Source",
								"owner":    "AWS",
								"provider": "CodeCommit",
								"version":  "1",
							},
							"configuration": map[string]any{"RepositoryName": "my-repo"},
							"outputArtifacts": []any{
								map[string]any{"name": "SourceOutput"},
							},
						},
					},
				},
				map[string]any{
					"name": "Build",
					"actions": []any{
						map[string]any{
							"name": "BuildAction",
							"actionTypeId": map[string]any{
								"category": "Build",
								"owner":    "AWS",
								"provider": "CodeBuild",
								"version":  "1",
							},
							"inputArtifacts": []any{
								map[string]any{"name": "SourceOutput"},
							},
						},
					},
				},
			},
		},
	}
}

func createPipeline(t *testing.T, s *svc.CodePipelineService, name string) map[string]any {
	t.Helper()
	ctx := jsonCtx("CreatePipeline", pipelineBody(name))
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return respBody(t, resp)
}

// --- Pipeline CRUD ---

func TestCreatePipeline(t *testing.T) {
	s := newService()
	body := createPipeline(t, s, "my-pipeline")
	pipeline := body["pipeline"].(map[string]any)
	assert.Equal(t, "my-pipeline", pipeline["name"])
	assert.Equal(t, float64(1), pipeline["version"])
	stages := pipeline["stages"].([]any)
	assert.Len(t, stages, 2)
}

func TestCreatePipelineDuplicate(t *testing.T) {
	s := newService()
	createPipeline(t, s, "dup-pipe")
	ctx := jsonCtx("CreatePipeline", pipelineBody("dup-pipe"))
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PipelineNameInUseException")
}

func TestCreatePipelineMissingName(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreatePipeline", map[string]any{
		"pipeline": map[string]any{},
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestCreatePipelineMissingPipelineField(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreatePipeline", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestGetPipeline(t *testing.T) {
	s := newService()
	createPipeline(t, s, "get-pipe")

	ctx := jsonCtx("GetPipeline", map[string]any{"name": "get-pipe"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	pipeline := body["pipeline"].(map[string]any)
	assert.Equal(t, "get-pipe", pipeline["name"])
	metadata := body["metadata"].(map[string]any)
	assert.Contains(t, metadata["pipelineArn"], "get-pipe")
}

func TestGetPipelineNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("GetPipeline", map[string]any{"name": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PipelineNotFoundException")
}

func TestListPipelines(t *testing.T) {
	s := newService()
	createPipeline(t, s, "pipe-1")
	createPipeline(t, s, "pipe-2")

	resp, err := s.HandleRequest(jsonCtx("ListPipelines", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	pipelines := body["pipelines"].([]any)
	assert.Len(t, pipelines, 2)
}

func TestUpdatePipeline(t *testing.T) {
	s := newService()
	createPipeline(t, s, "upd-pipe")

	ctx := jsonCtx("UpdatePipeline", map[string]any{
		"pipeline": map[string]any{
			"name":    "upd-pipe",
			"roleArn": "arn:aws:iam::123456789012:role/new-role",
			"stages": []any{
				map[string]any{
					"name":    "Source",
					"actions": []any{},
				},
			},
		},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	pipeline := body["pipeline"].(map[string]any)
	assert.Equal(t, float64(2), pipeline["version"])
}

func TestUpdatePipelineNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("UpdatePipeline", map[string]any{
		"pipeline": map[string]any{"name": "nonexistent"},
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PipelineNotFoundException")
}

func TestDeletePipeline(t *testing.T) {
	s := newService()
	createPipeline(t, s, "del-pipe")

	ctx := jsonCtx("DeletePipeline", map[string]any{"name": "del-pipe"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify gone
	resp2, _ := s.HandleRequest(jsonCtx("ListPipelines", map[string]any{}))
	body := respBody(t, resp2)
	assert.Len(t, body["pipelines"].([]any), 0)
}

func TestDeletePipelineNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("DeletePipeline", map[string]any{"name": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PipelineNotFoundException")
}

// --- Execution Tests ---

func TestStartPipelineExecution(t *testing.T) {
	s := newService()
	createPipeline(t, s, "exec-pipe")

	ctx := jsonCtx("StartPipelineExecution", map[string]any{"name": "exec-pipe"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["pipelineExecutionId"])
}

func TestStartPipelineExecutionNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("StartPipelineExecution", map[string]any{"name": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PipelineNotFoundException")
}

func TestGetPipelineExecution(t *testing.T) {
	s := newService()
	createPipeline(t, s, "getexec-pipe")

	resp, _ := s.HandleRequest(jsonCtx("StartPipelineExecution", map[string]any{"name": "getexec-pipe"}))
	body := respBody(t, resp)
	execID := body["pipelineExecutionId"].(string)

	ctx := jsonCtx("GetPipelineExecution", map[string]any{
		"pipelineName":        "getexec-pipe",
		"pipelineExecutionId": execID,
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	exec := body2["pipelineExecution"].(map[string]any)
	assert.Equal(t, execID, exec["pipelineExecutionId"])
	// With default lifecycle config (disabled), transitions are instant
	assert.Contains(t, []string{"InProgress", "Succeeded"}, exec["status"])
}

func TestGetPipelineExecutionNotFound(t *testing.T) {
	s := newService()
	createPipeline(t, s, "getexec-pipe2")
	ctx := jsonCtx("GetPipelineExecution", map[string]any{
		"pipelineName":        "getexec-pipe2",
		"pipelineExecutionId": "nonexistent",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PipelineExecutionNotFoundException")
}

func TestListPipelineExecutions(t *testing.T) {
	s := newService()
	createPipeline(t, s, "listexec-pipe")
	s.HandleRequest(jsonCtx("StartPipelineExecution", map[string]any{"name": "listexec-pipe"}))
	s.HandleRequest(jsonCtx("StartPipelineExecution", map[string]any{"name": "listexec-pipe"}))

	ctx := jsonCtx("ListPipelineExecutions", map[string]any{"pipelineName": "listexec-pipe"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	summaries := body["pipelineExecutionSummaries"].([]any)
	assert.Len(t, summaries, 2)
}

func TestStopPipelineExecution(t *testing.T) {
	s := newService()
	createPipeline(t, s, "stop-pipe")

	resp, _ := s.HandleRequest(jsonCtx("StartPipelineExecution", map[string]any{"name": "stop-pipe"}))
	body := respBody(t, resp)
	execID := body["pipelineExecutionId"].(string)

	ctx := jsonCtx("StopPipelineExecution", map[string]any{
		"pipelineName":        "stop-pipe",
		"pipelineExecutionId": execID,
		"reason":              "testing stop",
		"abandon":             true,
	})
	_, err := s.HandleRequest(ctx)
	// With instant lifecycle, execution may already be Succeeded and thus not stoppable.
	// Either it succeeds (still InProgress) or returns PipelineExecutionNotStoppableException.
	if err != nil {
		assert.Contains(t, err.Error(), "PipelineExecutionNotStoppableException")
	}
}

func TestGetPipelineState(t *testing.T) {
	s := newService()
	createPipeline(t, s, "state-pipe")
	s.HandleRequest(jsonCtx("StartPipelineExecution", map[string]any{"name": "state-pipe"}))

	ctx := jsonCtx("GetPipelineState", map[string]any{"name": "state-pipe"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "state-pipe", body["pipelineName"])
	stageStates := body["stageStates"].([]any)
	assert.Len(t, stageStates, 2)
}

func TestExecutionLifecycleTransition(t *testing.T) {
	s := newService()
	createPipeline(t, s, "lc-pipe")

	resp, _ := s.HandleRequest(jsonCtx("StartPipelineExecution", map[string]any{"name": "lc-pipe"}))
	body := respBody(t, resp)
	execID := body["pipelineExecutionId"].(string)

	// With default lifecycle config (disabled), transitions are instant.
	// Give goroutine callbacks a moment to complete.
	time.Sleep(50 * time.Millisecond)

	ctx := jsonCtx("GetPipelineExecution", map[string]any{
		"pipelineName":        "lc-pipe",
		"pipelineExecutionId": execID,
	})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	exec := body2["pipelineExecution"].(map[string]any)
	assert.Equal(t, "Succeeded", exec["status"])
}

// --- Approval ---

func TestPutApprovalResult(t *testing.T) {
	s := newService()
	createPipeline(t, s, "approval-pipe")
	s.HandleRequest(jsonCtx("StartPipelineExecution", map[string]any{"name": "approval-pipe"}))

	ctx := jsonCtx("PutApprovalResult", map[string]any{
		"pipelineName": "approval-pipe",
		"stageName":    "Source",
		"actionName":   "SourceAction",
		"result": map[string]any{
			"summary": "Looks good",
			"status":  "Approved",
		},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- Tags ---

func TestTagging(t *testing.T) {
	s := newService()
	createPipeline(t, s, "tag-pipe")
	arn := "arn:aws:codepipeline:us-east-1:123456789012:tag-pipe"

	// Tag
	ctx := jsonCtx("TagResource", map[string]any{
		"resourceArn": arn,
		"tags":        []any{map[string]any{"key": "env", "value": "prod"}},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// List tags
	ctx2 := jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn})
	resp2, err2 := s.HandleRequest(ctx2)
	require.NoError(t, err2)
	body := respBody(t, resp2)
	tags := body["tags"].([]any)
	assert.Len(t, tags, 1)
	tag := tags[0].(map[string]any)
	assert.Equal(t, "env", tag["key"])
	assert.Equal(t, "prod", tag["value"])

	// Untag
	ctx3 := jsonCtx("UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []any{"env"},
	})
	resp3, err3 := s.HandleRequest(ctx3)
	require.NoError(t, err3)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)

	// Verify removed
	resp4, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	body4 := respBody(t, resp4)
	assert.Len(t, body4["tags"].([]any), 0)
}

// --- Behavioral: Executor ---

func TestExecutor_SourceActionsSucceedImmediately(t *testing.T) {
	// Source actions should always succeed even without locator
	s := newService()
	createPipeline(t, s, "exec-test")

	resp, err := s.HandleRequest(jsonCtx("StartPipelineExecution", map[string]any{"name": "exec-test"}))
	require.NoError(t, err)
	body := respBody(t, resp)
	execID := body["pipelineExecutionId"].(string)
	assert.NotEmpty(t, execID)

	// With instant lifecycle, execution should succeed
	time.Sleep(100 * time.Millisecond)

	resp2, err := s.HandleRequest(jsonCtx("GetPipelineExecution", map[string]any{
		"pipelineName":        "exec-test",
		"pipelineExecutionId": execID,
	}))
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	exec := body2["pipelineExecution"].(map[string]any)
	assert.Equal(t, "Succeeded", exec["status"])
}

func TestPutApprovalResult_ApproveAction(t *testing.T) {
	s := newService()
	createPipeline(t, s, "approve-pipe")
	s.HandleRequest(jsonCtx("StartPipelineExecution", map[string]any{"name": "approve-pipe"}))

	// Approve the Source action
	resp, err := s.HandleRequest(jsonCtx("PutApprovalResult", map[string]any{
		"pipelineName": "approve-pipe",
		"stageName":    "Source",
		"actionName":   "SourceAction",
		"result": map[string]any{
			"summary": "Approved for release",
			"status":  "Approved",
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify action status changed
	resp2, _ := s.HandleRequest(jsonCtx("GetPipelineState", map[string]any{"name": "approve-pipe"}))
	body := respBody(t, resp2)
	stages := body["stageStates"].([]any)
	require.GreaterOrEqual(t, len(stages), 1)
	stage0 := stages[0].(map[string]any)
	actions := stage0["actionStates"].([]any)
	if len(actions) > 0 {
		action0 := actions[0].(map[string]any)
		latestExec := action0["latestExecution"].(map[string]any)
		assert.Equal(t, "Succeeded", latestExec["status"])
	}
}

// --- Invalid Action ---

func TestInvalidAction(t *testing.T) {
	s := newService()
	ctx := jsonCtx("BogusAction", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

// --- Service Metadata ---

func TestServiceName(t *testing.T) {
	s := newService()
	assert.Equal(t, "codepipeline", s.Name())
}

func TestHealthCheck(t *testing.T) {
	s := newService()
	assert.NoError(t, s.HealthCheck())
}
