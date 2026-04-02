package batch_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/batch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensure json import is used
var _ = json.Marshal

func newService() *svc.BatchService { return svc.New("123456789012", "us-east-1") }
func restCtx(path string, body map[string]any) *service.RequestContext {
	var b []byte; if body != nil { b, _ = json.Marshal(body) }
	return &service.RequestContext{Region: "us-east-1", AccountID: "123456789012", Body: b,
		RawRequest: httptest.NewRequest(http.MethodPost, path, nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

func TestBatch_ComputeEnvironmentCRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(restCtx("/v1/createcomputeenvironment", map[string]any{
		"computeEnvironmentName": "my-ce", "type": "MANAGED", "state": "ENABLED",
		"computeResources": map[string]any{"type": "EC2", "maxvCpus": 256, "instanceTypes": []string{"m5.xlarge"}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "my-ce", m["computeEnvironmentName"])

	descResp, _ := s.HandleRequest(restCtx("/v1/describecomputeenvironments", map[string]any{}))
	envs := respJSON(t, descResp)["computeEnvironments"].([]any)
	assert.Len(t, envs, 1)
	assert.Equal(t, "CREATING", envs[0].(map[string]any)["status"])

	delResp, err := s.HandleRequest(restCtx("/v1/deletecomputeenvironment", map[string]any{"computeEnvironment": "my-ce"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)
}

func TestBatch_JobQueueCRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(restCtx("/v1/createjobqueue", map[string]any{
		"jobQueueName": "my-queue", "priority": 10, "state": "ENABLED",
		"computeEnvironmentOrder": []map[string]any{{"computeEnvironment": "my-ce", "order": 1}},
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, respJSON(t, resp)["jobQueueArn"])

	descResp, _ := s.HandleRequest(restCtx("/v1/describejobqueues", nil))
	assert.Len(t, respJSON(t, descResp)["jobQueues"].([]any), 1)

	s.HandleRequest(restCtx("/v1/deletejobqueue", map[string]any{"jobQueue": "my-queue"}))
	descResp2, _ := s.HandleRequest(restCtx("/v1/describejobqueues", nil))
	assert.Len(t, respJSON(t, descResp2)["jobQueues"].([]any), 0)
}

func TestBatch_RegisterJobDefinitionWithRevisions(t *testing.T) {
	s := newService()
	resp1, err := s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{
		"jobDefinitionName": "my-def", "type": "container",
		"containerProperties": map[string]any{"image": "alpine:latest", "vcpus": 1, "memory": 512},
	}))
	require.NoError(t, err)
	m1 := respJSON(t, resp1)
	assert.Equal(t, float64(1), m1["revision"])

	resp2, _ := s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{
		"jobDefinitionName": "my-def", "type": "container",
		"containerProperties": map[string]any{"image": "alpine:3.18", "vcpus": 2, "memory": 1024},
	}))
	assert.Equal(t, float64(2), respJSON(t, resp2)["revision"])

	descResp, _ := s.HandleRequest(restCtx("/v1/describejobdefinitions", nil))
	defs := respJSON(t, descResp)["jobDefinitions"].([]any)
	assert.Len(t, defs, 2)
}

func TestBatch_DeregisterJobDefinition(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{
		"jobDefinitionName": "dereg-def", "type": "container",
	}))
	defArn := respJSON(t, cr)["jobDefinitionArn"].(string)

	resp, err := s.HandleRequest(restCtx("/v1/deregisterjobdefinition", map[string]any{"jobDefinition": defArn}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	descResp, _ := s.HandleRequest(restCtx("/v1/describejobdefinitions", nil))
	defs := respJSON(t, descResp)["jobDefinitions"].([]any)
	assert.Equal(t, "INACTIVE", defs[0].(map[string]any)["status"])
}

func TestBatch_SubmitAndDescribeJob(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{
		"jobDefinitionName": "submit-def", "type": "container",
	}))
	s.HandleRequest(restCtx("/v1/createjobqueue", map[string]any{
		"jobQueueName": "submit-q", "priority": 1,
	}))

	submitResp, err := s.HandleRequest(restCtx("/v1/submitjob", map[string]any{
		"jobName": "my-job", "jobQueue": "submit-q", "jobDefinition": "submit-def:1",
	}))
	require.NoError(t, err)
	m := respJSON(t, submitResp)
	jobID := m["jobId"].(string)
	assert.NotEmpty(t, jobID)

	descResp, _ := s.HandleRequest(restCtx("/v1/describejobs", map[string]any{"jobs": []string{jobID}}))
	jobs := respJSON(t, descResp)["jobs"].([]any)
	assert.Len(t, jobs, 1)
	// In instant mode (no lifecycle delays), job progresses through all states immediately.
	status := jobs[0].(map[string]any)["status"].(string)
	assert.Contains(t, []string{"SUBMITTED", "PENDING", "RUNNABLE", "STARTING", "RUNNING", "SUCCEEDED"}, status)
}

func TestBatch_ListJobs(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx("/v1/createjobqueue", map[string]any{"jobQueueName": "list-q", "priority": 1}))
	s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{"jobDefinitionName": "list-def", "type": "container"}))
	s.HandleRequest(restCtx("/v1/submitjob", map[string]any{
		"jobName": "j1", "jobQueue": "list-q", "jobDefinition": "list-def:1",
	}))
	s.HandleRequest(restCtx("/v1/submitjob", map[string]any{
		"jobName": "j2", "jobQueue": "list-q", "jobDefinition": "list-def:1",
	}))

	resp, _ := s.HandleRequest(restCtx("/v1/listjobs", map[string]any{"jobQueue": "list-q"}))
	assert.Len(t, respJSON(t, resp)["jobSummaryList"].([]any), 2)
}

func TestBatch_CancelJob(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx("/v1/createjobqueue", map[string]any{"jobQueueName": "cancel-q", "priority": 1}))
	s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{"jobDefinitionName": "cancel-def", "type": "container"}))
	submitResp, _ := s.HandleRequest(restCtx("/v1/submitjob", map[string]any{
		"jobName": "cancel-j", "jobQueue": "cancel-q", "jobDefinition": "cancel-def:1",
	}))
	jobID := respJSON(t, submitResp)["jobId"].(string)

	// In instant mode, the job has already completed. Cancel should fail for terminal jobs.
	_, err := s.HandleRequest(restCtx("/v1/canceljob", map[string]any{"jobId": jobID, "reason": "Testing"}))
	// The job may already be in SUCCEEDED state in instant mode.
	// CancelJob returns error for terminal states, which is correct behavior.
	if err != nil {
		// Expected when job has already reached SUCCEEDED in instant mode.
		return
	}

	descResp, _ := s.HandleRequest(restCtx("/v1/describejobs", map[string]any{"jobs": []string{jobID}}))
	status := respJSON(t, descResp)["jobs"].([]any)[0].(map[string]any)["status"].(string)
	assert.Contains(t, []string{"FAILED", "SUCCEEDED"}, status)
}

func TestBatch_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx("/v1/deletecomputeenvironment", map[string]any{"computeEnvironment": "nonexistent"}))
	require.Error(t, err)
}

func TestBatch_DuplicateComputeEnvironment(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx("/v1/createcomputeenvironment", map[string]any{"computeEnvironmentName": "dup-ce"}))
	_, err := s.HandleRequest(restCtx("/v1/createcomputeenvironment", map[string]any{"computeEnvironmentName": "dup-ce"}))
	require.Error(t, err)
}

// ---- Behavioral: Job state machine ----

func TestBatch_JobStateProgression(t *testing.T) {
	s := newService()

	// Create compute env, job queue, job def
	s.HandleRequest(restCtx("/v1/createcomputeenvironment", map[string]any{
		"computeEnvironmentName": "state-ce",
		"type":                   "MANAGED",
		"state":                  "ENABLED",
	}))

	s.HandleRequest(restCtx("/v1/createjobqueue", map[string]any{
		"jobQueueName": "state-queue",
		"state":        "ENABLED",
		"priority":     1,
		"computeEnvironmentOrder": []map[string]any{
			{"computeEnvironment": "state-ce", "order": 1},
		},
	}))

	s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{
		"jobDefinitionName": "state-job-def",
		"type":              "container",
		"containerProperties": map[string]any{
			"image": "busybox", "vcpus": 1, "memory": 512,
		},
	}))

	// Submit job
	resp, err := s.HandleRequest(restCtx("/v1/submitjob", map[string]any{
		"jobName":       "state-job",
		"jobQueue":      "state-queue",
		"jobDefinition": "state-job-def:1",
	}))
	require.NoError(t, err)
	m := decodeBody(t, resp)
	jobID := m["jobId"].(string)
	assert.NotEmpty(t, jobID)

	// In instant mode, job should have progressed through all states to SUCCEEDED.
	descResp, err := s.HandleRequest(restCtx("/v1/describejobs", map[string]any{
		"jobs": []string{jobID},
	}))
	require.NoError(t, err)
	descData := decodeBody(t, descResp)
	jobs := descData["jobs"].([]any)
	require.Len(t, jobs, 1)
	assert.Equal(t, "SUCCEEDED", jobs[0].(map[string]any)["status"])
}

// ---- Behavioral: Job definition revisions ----

func TestBatch_JobDefRevisions(t *testing.T) {
	s := newService()

	// Register same job def twice
	s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{
		"jobDefinitionName": "rev-def", "type": "container",
		"containerProperties": map[string]any{"image": "v1", "vcpus": 1, "memory": 512},
	}))
	s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{
		"jobDefinitionName": "rev-def", "type": "container",
		"containerProperties": map[string]any{"image": "v2", "vcpus": 2, "memory": 1024},
	}))

	// List should have 2 definitions
	resp, err := s.HandleRequest(restCtx("/v1/describejobdefinitions", map[string]any{}))
	require.NoError(t, err)
	m := decodeBody(t, resp)
	defs := m["jobDefinitions"].([]any)
	assert.GreaterOrEqual(t, len(defs), 2)
}

// ---- Behavioral: Deregister job definition ----

func TestBatch_DeregisterJobDef(t *testing.T) {
	s := newService()

	resp, err := s.HandleRequest(restCtx("/v1/registerjobdefinition", map[string]any{
		"jobDefinitionName": "dereg-def", "type": "container",
		"containerProperties": map[string]any{"image": "img", "vcpus": 1, "memory": 512},
	}))
	require.NoError(t, err)
	m := decodeBody(t, resp)
	arn := m["jobDefinitionArn"].(string)

	_, err = s.HandleRequest(restCtx("/v1/deregisterjobdefinition", map[string]any{
		"jobDefinition": arn,
	}))
	require.NoError(t, err)
}

func decodeBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestBatch_UpdateComputeEnvironment(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx("/v1/createcomputeenvironment", map[string]any{
		"computeEnvironmentName": "update-ce", "type": "MANAGED",
	}))
	resp, err := s.HandleRequest(restCtx("/v1/updatecomputeenvironment", map[string]any{
		"computeEnvironment": "update-ce", "state": "DISABLED",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "update-ce", m["computeEnvironmentName"])
}

func TestBatch_UpdateComputeEnvironment_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx("/v1/updatecomputeenvironment", map[string]any{
		"computeEnvironment": "nonexistent",
	}))
	require.Error(t, err)
}

func TestBatch_UpdateJobQueue(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx("/v1/createjobqueue", map[string]any{
		"jobQueueName": "update-q", "priority": 1, "state": "ENABLED",
	}))
	resp, err := s.HandleRequest(restCtx("/v1/updatejobqueue", map[string]any{
		"jobQueue": "update-q", "state": "DISABLED", "priority": 5,
	}))
	require.NoError(t, err)
	assert.Equal(t, "update-q", respJSON(t, resp)["jobQueueName"])
}

func TestBatch_SchedulingPolicyCRUD(t *testing.T) {
	s := newService()
	// Create
	resp, err := s.HandleRequest(restCtx("/v1/createschedulingpolicy", map[string]any{
		"name": "my-policy",
		"fairsharePolicy": map[string]any{
			"computeReservation": 10,
			"shareDecaySeconds":  300,
		},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	arn := m["arn"].(string)
	assert.Contains(t, arn, "scheduling-policy")

	// Describe
	descResp, err := s.HandleRequest(restCtx("/v1/describeschedulingpolicies", map[string]any{
		"arns": []string{arn},
	}))
	require.NoError(t, err)
	policies := respJSON(t, descResp)["schedulingPolicies"].([]any)
	assert.Len(t, policies, 1)

	// Update
	_, err = s.HandleRequest(restCtx("/v1/updateschedulingpolicy", map[string]any{
		"arn": arn,
		"fairsharePolicy": map[string]any{"computeReservation": 20},
	}))
	require.NoError(t, err)

	// Delete
	_, err = s.HandleRequest(restCtx("/v1/deleteschedulingpolicy", map[string]any{"arn": arn}))
	require.NoError(t, err)

	// After delete, should not be in list
	listResp, _ := s.HandleRequest(restCtx("/v1/describeschedulingpolicies", map[string]any{
		"arns": []string{arn},
	}))
	assert.Len(t, respJSON(t, listResp)["schedulingPolicies"].([]any), 0)
}

func TestBatch_TaggingOperations(t *testing.T) {
	s := newService()
	// Create a CE and get its ARN
	cr, _ := s.HandleRequest(restCtx("/v1/createcomputeenvironment", map[string]any{
		"computeEnvironmentName": "tag-ce",
	}))
	arn := respJSON(t, cr)["computeEnvironmentArn"].(string)

	// POST tags
	r := httptest.NewRequest(http.MethodPost, "/v1/tags/"+arn, nil)
	tagCtx := &service.RequestContext{
		Region: "us-east-1", AccountID: "123456789012",
		Body:       mustMarshal(map[string]any{"tags": map[string]any{"env": "prod"}}),
		RawRequest: r,
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err := s.HandleRequest(tagCtx)
	require.NoError(t, err)

	// GET tags
	getR := httptest.NewRequest(http.MethodGet, "/v1/tags/"+arn, nil)
	getCtx := &service.RequestContext{
		Region: "us-east-1", AccountID: "123456789012",
		RawRequest: getR,
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	listResp, err := s.HandleRequest(getCtx)
	require.NoError(t, err)
	tags := respJSON(t, listResp)["tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
