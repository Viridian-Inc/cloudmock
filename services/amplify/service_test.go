package amplify_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/amplify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.AmplifyService {
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

func mustCreateApp(t *testing.T, s *svc.AmplifyService, name string) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateApp", map[string]any{"name": name}))
	require.NoError(t, err)
	app := decode(t, resp)["app"].(map[string]any)
	return app["appId"].(string)
}

func mustCreateBranch(t *testing.T, s *svc.AmplifyService, appID, branchName string) {
	t.Helper()
	_, err := s.HandleRequest(jsonCtx("CreateBranch", map[string]any{
		"appId": appID, "branchName": branchName,
	}))
	require.NoError(t, err)
}

// ---- App tests ----

func TestCreateApp(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateApp", map[string]any{
		"name": "my-app", "description": "test app",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	app := decode(t, resp)["app"].(map[string]any)
	assert.Equal(t, "my-app", app["name"])
	assert.NotEmpty(t, app["appId"])
	assert.NotEmpty(t, app["appArn"])
	assert.Equal(t, "WEB", app["platform"])
	assert.NotEmpty(t, app["defaultDomain"])
}

func TestCreateAppMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateApp", map[string]any{}))
	require.Error(t, err)
}

func TestGetApp(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	resp, err := s.HandleRequest(jsonCtx("GetApp", map[string]any{"appId": appID}))
	require.NoError(t, err)
	app := decode(t, resp)["app"].(map[string]any)
	assert.Equal(t, "my-app", app["name"])
}

func TestGetAppNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetApp", map[string]any{"appId": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "NotFoundException", awsErr.Code)
}

func TestListApps(t *testing.T) {
	s := newService()
	mustCreateApp(t, s, "app-1")
	mustCreateApp(t, s, "app-2")

	resp, err := s.HandleRequest(jsonCtx("ListApps", map[string]any{}))
	require.NoError(t, err)
	apps := decode(t, resp)["apps"].([]any)
	assert.Len(t, apps, 2)
}

func TestUpdateApp(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	resp, err := s.HandleRequest(jsonCtx("UpdateApp", map[string]any{
		"appId": appID, "name": "updated-app", "description": "updated",
	}))
	require.NoError(t, err)
	app := decode(t, resp)["app"].(map[string]any)
	assert.Equal(t, "updated-app", app["name"])
}

func TestDeleteApp(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	resp, err := s.HandleRequest(jsonCtx("DeleteApp", map[string]any{"appId": appID}))
	require.NoError(t, err)
	app := decode(t, resp)["app"].(map[string]any)
	assert.Equal(t, "my-app", app["name"])

	_, err = s.HandleRequest(jsonCtx("GetApp", map[string]any{"appId": appID}))
	require.Error(t, err)
}

// ---- Branch tests ----

func TestCreateBranch(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	resp, err := s.HandleRequest(jsonCtx("CreateBranch", map[string]any{
		"appId": appID, "branchName": "main", "stage": "PRODUCTION",
	}))
	require.NoError(t, err)
	branch := decode(t, resp)["branch"].(map[string]any)
	assert.Equal(t, "main", branch["branchName"])
	assert.Equal(t, "PRODUCTION", branch["stage"])
	assert.NotEmpty(t, branch["branchArn"])
}

func TestCreateBranchDefaultStage(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	resp, _ := s.HandleRequest(jsonCtx("CreateBranch", map[string]any{
		"appId": appID, "branchName": "dev",
	}))
	branch := decode(t, resp)["branch"].(map[string]any)
	assert.Equal(t, "NONE", branch["stage"])
}

func TestGetBranch(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	mustCreateBranch(t, s, appID, "main")

	resp, err := s.HandleRequest(jsonCtx("GetBranch", map[string]any{
		"appId": appID, "branchName": "main",
	}))
	require.NoError(t, err)
	branch := decode(t, resp)["branch"].(map[string]any)
	assert.Equal(t, "main", branch["branchName"])
}

func TestListBranches(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	mustCreateBranch(t, s, appID, "main")
	mustCreateBranch(t, s, appID, "develop")

	resp, err := s.HandleRequest(jsonCtx("ListBranches", map[string]any{"appId": appID}))
	require.NoError(t, err)
	branches := decode(t, resp)["branches"].([]any)
	assert.Len(t, branches, 2)
}

func TestUpdateBranch(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	mustCreateBranch(t, s, appID, "main")

	resp, err := s.HandleRequest(jsonCtx("UpdateBranch", map[string]any{
		"appId": appID, "branchName": "main", "stage": "PRODUCTION",
	}))
	require.NoError(t, err)
	branch := decode(t, resp)["branch"].(map[string]any)
	assert.Equal(t, "PRODUCTION", branch["stage"])
}

func TestDeleteBranch(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	mustCreateBranch(t, s, appID, "main")

	resp, err := s.HandleRequest(jsonCtx("DeleteBranch", map[string]any{
		"appId": appID, "branchName": "main",
	}))
	require.NoError(t, err)
	branch := decode(t, resp)["branch"].(map[string]any)
	assert.Equal(t, "main", branch["branchName"])

	_, err = s.HandleRequest(jsonCtx("GetBranch", map[string]any{
		"appId": appID, "branchName": "main",
	}))
	require.Error(t, err)
}

// ---- Domain Association tests ----

func TestCreateDomainAssociation(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	resp, err := s.HandleRequest(jsonCtx("CreateDomainAssociation", map[string]any{
		"appId": appID, "domainName": "example.com",
		"subDomainSettings": []map[string]any{{"prefix": "", "branchName": "main"}},
	}))
	require.NoError(t, err)
	domain := decode(t, resp)["domainAssociation"].(map[string]any)
	assert.Equal(t, "example.com", domain["domainName"])
	assert.Equal(t, "CREATING", domain["domainStatus"])
}

func TestListDomainAssociations(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	s.HandleRequest(jsonCtx("CreateDomainAssociation", map[string]any{
		"appId": appID, "domainName": "example.com",
	}))
	s.HandleRequest(jsonCtx("CreateDomainAssociation", map[string]any{
		"appId": appID, "domainName": "other.com",
	}))

	resp, err := s.HandleRequest(jsonCtx("ListDomainAssociations", map[string]any{"appId": appID}))
	require.NoError(t, err)
	domains := decode(t, resp)["domainAssociations"].([]any)
	assert.Len(t, domains, 2)
}

func TestDeleteDomainAssociation(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	s.HandleRequest(jsonCtx("CreateDomainAssociation", map[string]any{
		"appId": appID, "domainName": "example.com",
	}))

	_, err := s.HandleRequest(jsonCtx("DeleteDomainAssociation", map[string]any{
		"appId": appID, "domainName": "example.com",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetDomainAssociation", map[string]any{
		"appId": appID, "domainName": "example.com",
	}))
	require.Error(t, err)
}

// ---- Webhook tests ----

func TestCreateWebhook(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	resp, err := s.HandleRequest(jsonCtx("CreateWebhook", map[string]any{
		"appId": appID, "branchName": "main", "description": "test webhook",
	}))
	require.NoError(t, err)
	wh := decode(t, resp)["webhook"].(map[string]any)
	assert.NotEmpty(t, wh["webhookId"])
	assert.NotEmpty(t, wh["webhookUrl"])
	assert.Equal(t, "main", wh["branchName"])
}

func TestListWebhooks(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	s.HandleRequest(jsonCtx("CreateWebhook", map[string]any{"appId": appID, "branchName": "main"}))
	s.HandleRequest(jsonCtx("CreateWebhook", map[string]any{"appId": appID, "branchName": "dev"}))

	resp, err := s.HandleRequest(jsonCtx("ListWebhooks", map[string]any{"appId": appID}))
	require.NoError(t, err)
	webhooks := decode(t, resp)["webhooks"].([]any)
	assert.Len(t, webhooks, 2)
}

func TestDeleteWebhook(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	createResp, _ := s.HandleRequest(jsonCtx("CreateWebhook", map[string]any{
		"appId": appID, "branchName": "main",
	}))
	whID := decode(t, createResp)["webhook"].(map[string]any)["webhookId"].(string)

	_, err := s.HandleRequest(jsonCtx("DeleteWebhook", map[string]any{"webhookId": whID}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetWebhook", map[string]any{"webhookId": whID}))
	require.Error(t, err)
}

// ---- Job lifecycle tests ----

func TestStartJob(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	mustCreateBranch(t, s, appID, "main")

	resp, err := s.HandleRequest(jsonCtx("StartJob", map[string]any{
		"appId": appID, "branchName": "main", "jobType": "RELEASE",
		"commitId": "abc123", "commitMessage": "initial commit",
	}))
	require.NoError(t, err)
	summary := decode(t, resp)["jobSummary"].(map[string]any)
	assert.NotEmpty(t, summary["jobId"])
	assert.Equal(t, "RELEASE", summary["jobType"])
}

func TestJobLifecycleInstant(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	mustCreateBranch(t, s, appID, "main")

	startResp, _ := s.HandleRequest(jsonCtx("StartJob", map[string]any{
		"appId": appID, "branchName": "main",
	}))
	jobID := decode(t, startResp)["jobSummary"].(map[string]any)["jobId"].(string)

	// With lifecycle delays disabled, job transitions PENDING -> RUNNING -> SUCCEED instantly
	getResp, err := s.HandleRequest(jsonCtx("GetJob", map[string]any{
		"appId": appID, "branchName": "main", "jobId": jobID,
	}))
	require.NoError(t, err)
	job := decode(t, getResp)["job"].(map[string]any)
	summary := job["summary"].(map[string]any)
	assert.Equal(t, "SUCCEED", summary["status"])
}

func TestListJobs(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	mustCreateBranch(t, s, appID, "main")

	s.HandleRequest(jsonCtx("StartJob", map[string]any{"appId": appID, "branchName": "main"}))
	s.HandleRequest(jsonCtx("StartJob", map[string]any{"appId": appID, "branchName": "main"}))

	resp, err := s.HandleRequest(jsonCtx("ListJobs", map[string]any{
		"appId": appID, "branchName": "main",
	}))
	require.NoError(t, err)
	jobs := decode(t, resp)["jobSummaries"].([]any)
	assert.Len(t, jobs, 2)
}

func TestStopJobNotFound(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	mustCreateBranch(t, s, appID, "main")

	_, err := s.HandleRequest(jsonCtx("StopJob", map[string]any{
		"appId": appID, "branchName": "main", "jobId": "99999",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "NotFoundException", awsErr.Code)
}

// ---- Tagging ----

func TestTagApp(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateApp", map[string]any{"name": "my-app"}))
	app := decode(t, resp)["app"].(map[string]any)
	arn := app["appArn"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"resourceArn": arn,
		"tags":        map[string]any{"env": "prod"},
	}))
	require.NoError(t, err)

	tagsResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	require.NoError(t, err)
	tags := decode(t, tagsResp)["tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
}

func TestUntagApp(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateApp", map[string]any{
		"name": "my-app", "tags": map[string]any{"env": "prod", "team": "alpha"},
	}))
	app := decode(t, resp)["app"].(map[string]any)
	arn := app["appArn"].(string)

	_, err := s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []string{"team"},
	}))
	require.NoError(t, err)

	tagsResp, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	tags := decode(t, tagsResp)["tags"].(map[string]any)
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

// ---- Not found cases ----

func TestStartJobBranchNotFound(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	_, err := s.HandleRequest(jsonCtx("StartJob", map[string]any{
		"appId": appID, "branchName": "nonexistent",
	}))
	require.Error(t, err)
}

func TestDeleteAppNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteApp", map[string]any{"appId": "nonexistent"}))
	require.Error(t, err)
}

// ---- BackendEnvironment tests ----

func TestCreateBackendEnvironment(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	resp, err := s.HandleRequest(jsonCtx("CreateBackendEnvironment", map[string]any{
		"appId":               appID,
		"environmentName":     "staging",
		"deploymentArtifacts": "amplify-builds",
		"stackName":           "amplify-backend-staging",
	}))
	require.NoError(t, err)
	be := decode(t, resp)["backendEnvironment"].(map[string]any)
	assert.Equal(t, "staging", be["environmentName"])
	assert.Equal(t, "amplify-builds", be["deploymentArtifacts"])
	assert.NotEmpty(t, be["backendEnvironmentArn"])
}

func TestGetBackendEnvironment(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")
	s.HandleRequest(jsonCtx("CreateBackendEnvironment", map[string]any{
		"appId": appID, "environmentName": "prod",
	}))

	resp, err := s.HandleRequest(jsonCtx("GetBackendEnvironment", map[string]any{
		"appId": appID, "environmentName": "prod",
	}))
	require.NoError(t, err)
	be := decode(t, resp)["backendEnvironment"].(map[string]any)
	assert.Equal(t, "prod", be["environmentName"])
}

func TestGetBackendEnvironmentNotFound(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	_, err := s.HandleRequest(jsonCtx("GetBackendEnvironment", map[string]any{
		"appId": appID, "environmentName": "nonexistent",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "NotFoundException", awsErr.Code)
}

func TestListBackendEnvironments(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	s.HandleRequest(jsonCtx("CreateBackendEnvironment", map[string]any{
		"appId": appID, "environmentName": "dev",
	}))
	s.HandleRequest(jsonCtx("CreateBackendEnvironment", map[string]any{
		"appId": appID, "environmentName": "prod",
	}))

	resp, err := s.HandleRequest(jsonCtx("ListBackendEnvironments", map[string]any{"appId": appID}))
	require.NoError(t, err)
	backends := decode(t, resp)["backendEnvironments"].([]any)
	assert.Len(t, backends, 2)
}

func TestDeleteBackendEnvironment(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	s.HandleRequest(jsonCtx("CreateBackendEnvironment", map[string]any{
		"appId": appID, "environmentName": "temp",
	}))

	resp, err := s.HandleRequest(jsonCtx("DeleteBackendEnvironment", map[string]any{
		"appId": appID, "environmentName": "temp",
	}))
	require.NoError(t, err)
	be := decode(t, resp)["backendEnvironment"].(map[string]any)
	assert.Equal(t, "temp", be["environmentName"])

	_, err = s.HandleRequest(jsonCtx("GetBackendEnvironment", map[string]any{
		"appId": appID, "environmentName": "temp",
	}))
	require.Error(t, err)
}

func TestDeleteBackendEnvironmentNotFound(t *testing.T) {
	s := newService()
	appID := mustCreateApp(t, s, "my-app")

	_, err := s.HandleRequest(jsonCtx("DeleteBackendEnvironment", map[string]any{
		"appId": appID, "environmentName": "nonexistent",
	}))
	require.Error(t, err)
}

func TestCreateBackendEnvironmentAppNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateBackendEnvironment", map[string]any{
		"appId": "nonexistent", "environmentName": "prod",
	}))
	require.Error(t, err)
}
