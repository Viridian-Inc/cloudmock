package codebuild_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/codebuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.CodeBuildService {
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

func createProject(t *testing.T, s *svc.CodeBuildService, name string) map[string]any {
	t.Helper()
	ctx := jsonCtx("CreateProject", map[string]any{
		"name":        name,
		"description": "test project",
		"serviceRole": "arn:aws:iam::123456789012:role/codebuild-role",
		"source":      map[string]any{"type": "CODECOMMIT", "location": "https://git-codecommit.us-east-1.amazonaws.com/v1/repos/my-repo"},
		"artifacts":   map[string]any{"type": "S3", "location": "my-bucket"},
		"environment": map[string]any{"type": "LINUX_CONTAINER", "image": "aws/codebuild/standard:5.0", "computeType": "BUILD_GENERAL1_SMALL"},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return respBody(t, resp)
}

// --- Project Tests ---

func TestCreateProject(t *testing.T) {
	s := newService()
	body := createProject(t, s, "my-project")
	project := body["project"].(map[string]any)
	assert.Equal(t, "my-project", project["name"])
	assert.Contains(t, project["arn"], "arn:aws:codebuild:us-east-1:123456789012:project/my-project")
	assert.Equal(t, "test project", project["description"])
	assert.Equal(t, "CODECOMMIT", project["source"].(map[string]any)["type"])
}

func TestCreateProjectDuplicate(t *testing.T) {
	s := newService()
	createProject(t, s, "dup-project")
	ctx := jsonCtx("CreateProject", map[string]any{"name": "dup-project"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceAlreadyExistsException")
}

func TestCreateProjectMissingName(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateProject", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestBatchGetProjects(t *testing.T) {
	s := newService()
	createProject(t, s, "proj-a")
	createProject(t, s, "proj-b")

	ctx := jsonCtx("BatchGetProjects", map[string]any{
		"names": []any{"proj-a", "proj-b", "proj-nonexistent"},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)

	projects := body["projects"].([]any)
	assert.Len(t, projects, 2)
	notFound := body["projectsNotFound"].([]any)
	assert.Len(t, notFound, 1)
	assert.Equal(t, "proj-nonexistent", notFound[0])
}

func TestListProjects(t *testing.T) {
	s := newService()
	createProject(t, s, "proj-1")
	createProject(t, s, "proj-2")

	ctx := jsonCtx("ListProjects", map[string]any{})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	projects := body["projects"].([]any)
	assert.Len(t, projects, 2)
}

func TestUpdateProject(t *testing.T) {
	s := newService()
	createProject(t, s, "update-proj")

	ctx := jsonCtx("UpdateProject", map[string]any{
		"name":        "update-proj",
		"description": "updated desc",
		"serviceRole": "arn:aws:iam::123456789012:role/new-role",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	project := body["project"].(map[string]any)
	assert.Equal(t, "updated desc", project["description"])
	assert.Equal(t, "arn:aws:iam::123456789012:role/new-role", project["serviceRole"])
}

func TestUpdateProjectNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("UpdateProject", map[string]any{"name": "nonexistent"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestDeleteProject(t *testing.T) {
	s := newService()
	createProject(t, s, "del-proj")

	ctx := jsonCtx("DeleteProject", map[string]any{"name": "del-proj"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify gone
	ctx2 := jsonCtx("ListProjects", map[string]any{})
	resp2, err2 := s.HandleRequest(ctx2)
	require.NoError(t, err2)
	body := respBody(t, resp2)
	assert.Len(t, body["projects"].([]any), 0)
}

func TestDeleteProjectNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("DeleteProject", map[string]any{"name": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

// --- Build Tests ---

func TestStartBuild(t *testing.T) {
	s := newService()
	createProject(t, s, "build-proj")

	ctx := jsonCtx("StartBuild", map[string]any{"projectName": "build-proj"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	build := body["build"].(map[string]any)
	assert.Equal(t, "build-proj", build["projectName"])
	assert.Equal(t, "SUBMITTED", build["buildStatus"])
	assert.NotEmpty(t, build["id"])
	assert.NotEmpty(t, build["arn"])
	assert.Equal(t, float64(1), build["buildNumber"])
}

func TestStartBuildProjectNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("StartBuild", map[string]any{"projectName": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestStartBuildMissingProjectName(t *testing.T) {
	s := newService()
	ctx := jsonCtx("StartBuild", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestBatchGetBuilds(t *testing.T) {
	s := newService()
	createProject(t, s, "bg-proj")

	// Start a build
	ctx := jsonCtx("StartBuild", map[string]any{"projectName": "bg-proj"})
	resp, _ := s.HandleRequest(ctx)
	body := respBody(t, resp)
	buildID := body["build"].(map[string]any)["id"].(string)

	ctx2 := jsonCtx("BatchGetBuilds", map[string]any{"ids": []any{buildID, "nonexistent-id"}})
	resp2, err := s.HandleRequest(ctx2)
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	builds := body2["builds"].([]any)
	assert.Len(t, builds, 1)
	notFound := body2["buildsNotFound"].([]any)
	assert.Len(t, notFound, 1)
}

func TestListBuildsForProject(t *testing.T) {
	s := newService()
	createProject(t, s, "list-build-proj")

	s.HandleRequest(jsonCtx("StartBuild", map[string]any{"projectName": "list-build-proj"}))
	s.HandleRequest(jsonCtx("StartBuild", map[string]any{"projectName": "list-build-proj"}))

	ctx := jsonCtx("ListBuildsForProject", map[string]any{"projectName": "list-build-proj"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	ids := body["ids"].([]any)
	assert.Len(t, ids, 2)
}

func TestStopBuild(t *testing.T) {
	s := newService()
	createProject(t, s, "stop-proj")

	resp, _ := s.HandleRequest(jsonCtx("StartBuild", map[string]any{"projectName": "stop-proj"}))
	body := respBody(t, resp)
	buildID := body["build"].(map[string]any)["id"].(string)

	ctx := jsonCtx("StopBuild", map[string]any{"id": buildID})
	resp2, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	build := body2["build"].(map[string]any)
	assert.Equal(t, "STOPPED", build["buildStatus"])
	assert.Equal(t, "COMPLETED", build["currentPhase"])
}

func TestStopBuildNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("StopBuild", map[string]any{"id": "nonexistent"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestBuildLifecycleTransition(t *testing.T) {
	s := newService()
	createProject(t, s, "lifecycle-proj")

	resp, _ := s.HandleRequest(jsonCtx("StartBuild", map[string]any{"projectName": "lifecycle-proj"}))
	body := respBody(t, resp)
	buildID := body["build"].(map[string]any)["id"].(string)

	// With default lifecycle config (disabled), transitions are instant.
	// Give goroutine callbacks a moment to complete.
	time.Sleep(50 * time.Millisecond)

	resp2, err := s.HandleRequest(jsonCtx("BatchGetBuilds", map[string]any{"ids": []any{buildID}}))
	require.NoError(t, err)
	body2 := respBody(t, resp2)
	builds := body2["builds"].([]any)
	require.Len(t, builds, 1)
	status := builds[0].(map[string]any)["buildStatus"].(string)
	assert.Equal(t, "SUCCEEDED", status)
}

// --- Report Group Tests ---

func TestCreateReportGroup(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateReportGroup", map[string]any{
		"name": "my-report-group",
		"type": "TEST",
		"exportConfig": map[string]any{
			"exportConfigType": "S3",
			"s3Destination":    map[string]any{"bucket": "my-bucket", "path": "reports/"},
		},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	rg := body["reportGroup"].(map[string]any)
	assert.Equal(t, "my-report-group", rg["name"])
	assert.Equal(t, "TEST", rg["type"])
	assert.Contains(t, rg["arn"], "report-group/my-report-group")
}

func TestCreateReportGroupMissingName(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateReportGroup", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestBatchGetReportGroups(t *testing.T) {
	s := newService()
	// Create a report group and get its name
	ctx := jsonCtx("CreateReportGroup", map[string]any{"name": "rg-1", "type": "TEST"})
	s.HandleRequest(ctx)

	ctx2 := jsonCtx("BatchGetReportGroups", map[string]any{
		"reportGroupArns": []any{"rg-1", "nonexistent"},
	})
	resp, err := s.HandleRequest(ctx2)
	require.NoError(t, err)
	body := respBody(t, resp)
	groups := body["reportGroups"].([]any)
	assert.Len(t, groups, 1)
	notFound := body["reportGroupsNotFound"].([]any)
	assert.Len(t, notFound, 1)
}

func TestListReportGroups(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateReportGroup", map[string]any{"name": "rg-a"}))
	s.HandleRequest(jsonCtx("CreateReportGroup", map[string]any{"name": "rg-b"}))

	resp, err := s.HandleRequest(jsonCtx("ListReportGroups", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	groups := body["reportGroups"].([]any)
	assert.Len(t, groups, 2)
}

func TestDeleteReportGroup(t *testing.T) {
	s := newService()
	rResp, _ := s.HandleRequest(jsonCtx("CreateReportGroup", map[string]any{"name": "del-rg"}))
	rBody := respBody(t, rResp)
	arn := rBody["reportGroup"].(map[string]any)["arn"].(string)

	ctx := jsonCtx("DeleteReportGroup", map[string]any{"arn": arn})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify gone
	resp2, _ := s.HandleRequest(jsonCtx("ListReportGroups", map[string]any{}))
	body := respBody(t, resp2)
	assert.Len(t, body["reportGroups"].([]any), 0)
}

func TestDeleteReportGroupNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("DeleteReportGroup", map[string]any{"arn": "arn:aws:codebuild:us-east-1:123456789012:report-group/nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
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
	assert.Equal(t, "codebuild", s.Name())
}

func TestHealthCheck(t *testing.T) {
	s := newService()
	assert.NoError(t, s.HealthCheck())
}

func TestProjectDefaultTimeout(t *testing.T) {
	s := newService()
	body := createProject(t, s, "timeout-proj")
	project := body["project"].(map[string]any)
	assert.Equal(t, float64(60), project["timeoutInMinutes"])
}

func TestBuildPhases_Present(t *testing.T) {
	s := newService()
	createProject(t, s, "phases-proj")

	resp, _ := s.HandleRequest(jsonCtx("StartBuild", map[string]any{"projectName": "phases-proj"}))
	body := respBody(t, resp)
	buildID := body["build"].(map[string]any)["id"].(string)

	// Wait for lifecycle to complete (goroutine callbacks)
	var build map[string]any
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		resp2, err := s.HandleRequest(jsonCtx("BatchGetBuilds", map[string]any{"ids": []any{buildID}}))
		require.NoError(t, err)
		body2 := respBody(t, resp2)
		builds := body2["builds"].([]any)
		require.Len(t, builds, 1)
		build = builds[0].(map[string]any)
		if build["buildStatus"] == "SUCCEEDED" {
			break
		}
	}

	assert.Equal(t, "SUCCEEDED", build["buildStatus"])
	// Check that phases are present
	phases, ok := build["phases"].([]any)
	assert.True(t, ok, "phases should be present in completed build")
	if ok {
		assert.GreaterOrEqual(t, len(phases), 10, "should have all build phases")

		// Verify first phase is SUBMITTED
		if len(phases) > 0 {
			first := phases[0].(map[string]any)
			assert.Equal(t, "SUBMITTED", first["phaseType"])
			assert.Equal(t, "SUCCEEDED", first["phaseStatus"])
		}

		// Verify last phase is COMPLETED
		if len(phases) > 0 {
			last := phases[len(phases)-1].(map[string]any)
			assert.Equal(t, "COMPLETED", last["phaseType"])
		}
	}
}

func TestBuildNumberIncrements(t *testing.T) {
	s := newService()
	createProject(t, s, "incr-proj")

	resp1, _ := s.HandleRequest(jsonCtx("StartBuild", map[string]any{"projectName": "incr-proj"}))
	body1 := respBody(t, resp1)
	assert.Equal(t, float64(1), body1["build"].(map[string]any)["buildNumber"])

	resp2, _ := s.HandleRequest(jsonCtx("StartBuild", map[string]any{"projectName": "incr-proj"}))
	body2 := respBody(t, resp2)
	assert.Equal(t, float64(2), body2["build"].(map[string]any)["buildNumber"])
}
