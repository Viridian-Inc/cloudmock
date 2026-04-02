package serverlessrepo_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/serverlessrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.ServerlessRepoService { return svc.New("123456789012", "us-east-1") }
func restCtx(method, path string, body map[string]any) *service.RequestContext {
	var b []byte; if body != nil { b, _ = json.Marshal(body) }
	return &service.RequestContext{Region: "us-east-1", AccountID: "123456789012", Body: b,
		RawRequest: httptest.NewRequest(method, path, nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

func createApp(t *testing.T, s *svc.ServerlessRepoService) string {
	resp, err := s.HandleRequest(restCtx(http.MethodPost, "/applications", map[string]any{
		"Name": "my-app", "Author": "cloudmock", "Description": "Test app", "SemanticVersion": "1.0.0",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	return respJSON(t, resp)["ApplicationId"].(string)
}

func TestSR_CreateAndGetApplication(t *testing.T) {
	s := newService()
	appID := createApp(t, s)

	getResp, err := s.HandleRequest(restCtx(http.MethodGet, "/applications/"+appID, nil))
	require.NoError(t, err)
	m := respJSON(t, getResp)
	assert.Equal(t, "my-app", m["Name"])
	assert.Equal(t, "cloudmock", m["Author"])
}

func TestSR_ListApplications(t *testing.T) {
	s := newService()
	createApp(t, s); createApp(t, s)
	resp, _ := s.HandleRequest(restCtx(http.MethodGet, "/applications", nil))
	assert.Len(t, respJSON(t, resp)["Applications"].([]any), 2)
}

func TestSR_DeleteApplication(t *testing.T) {
	s := newService()
	appID := createApp(t, s)
	resp, err := s.HandleRequest(restCtx(http.MethodDelete, "/applications/"+appID, nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	_, err = s.HandleRequest(restCtx(http.MethodGet, "/applications/"+appID, nil))
	require.Error(t, err)
}

func TestSR_CreateApplicationVersion(t *testing.T) {
	s := newService()
	appID := createApp(t, s)

	verResp, err := s.HandleRequest(restCtx(http.MethodPut, "/applications/"+appID+"/versions/2.0.0", map[string]any{
		"SourceCodeUrl": "https://github.com/example",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, verResp.StatusCode)
	assert.Equal(t, "2.0.0", respJSON(t, verResp)["SemanticVersion"])
}

func TestSR_ListVersions(t *testing.T) {
	s := newService()
	appID := createApp(t, s)
	s.HandleRequest(restCtx(http.MethodPut, "/applications/"+appID+"/versions/2.0.0", map[string]any{}))

	resp, _ := s.HandleRequest(restCtx(http.MethodGet, "/applications/"+appID+"/versions", nil))
	versions := respJSON(t, resp)["Versions"].([]any)
	assert.GreaterOrEqual(t, len(versions), 2) // 1.0.0 + 2.0.0
}

func TestSR_CreateChangeSet(t *testing.T) {
	s := newService()
	appID := createApp(t, s)

	csResp, err := s.HandleRequest(restCtx(http.MethodPost, "/applications/"+appID+"/changesets", map[string]any{
		"SemanticVersion": "1.0.0", "StackName": "my-stack",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, csResp.StatusCode)
	m := respJSON(t, csResp)
	assert.NotEmpty(t, m["ChangeSetId"])
	assert.NotEmpty(t, m["StackId"])
}

func TestSR_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/applications/nonexistent", nil))
	require.Error(t, err)
}

func TestSR_MissingRequiredFields(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/applications", map[string]any{"Name": "no-author"}))
	require.Error(t, err)
}

func TestSR_InvalidSemver(t *testing.T) {
	s := newService()
	// Create app first
	cr, _ := s.HandleRequest(restCtx(http.MethodPost, "/applications", map[string]any{
		"Name": "semver-app", "Author": "Test", "Description": "desc",
	}))
	appID := respJSON(t, cr)["ApplicationId"].(string)

	// Create version with invalid semver
	_, err := s.HandleRequest(restCtx(http.MethodPut, "/applications/"+appID+"/versions/not-a-version", nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid semantic version")
}

func TestSR_ValidSemver(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(restCtx(http.MethodPost, "/applications", map[string]any{
		"Name": "sv-app", "Author": "Test", "Description": "desc", "SemanticVersion": "1.0.0",
	}))
	appID := respJSON(t, cr)["ApplicationId"].(string)

	// Create another valid version
	resp, err := s.HandleRequest(restCtx(http.MethodPut, "/applications/"+appID+"/versions/2.0.0", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestSR_DuplicateVersion(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(restCtx(http.MethodPost, "/applications", map[string]any{
		"Name": "dup-ver-app", "Author": "Test", "Description": "desc", "SemanticVersion": "1.0.0",
	}))
	appID := respJSON(t, cr)["ApplicationId"].(string)

	// Try to create same version again
	_, err := s.HandleRequest(restCtx(http.MethodPut, "/applications/"+appID+"/versions/1.0.0", nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestSR_UpdateApplication(t *testing.T) {
	s := newService()
	appID := createApp(t, s)

	resp, err := s.HandleRequest(restCtx(http.MethodPatch, "/applications/"+appID, map[string]any{
		"Description": "Updated description",
		"Author":      "Updated Author",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "Updated description", m["Description"])
	assert.Equal(t, "Updated Author", m["Author"])
}

func TestSR_UpdateApplicationNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodPatch, "/applications/nonexistent", map[string]any{
		"Description": "update",
	}))
	require.Error(t, err)
}

func TestSR_ApplicationHasID(t *testing.T) {
	s := newService()
	cr, err := s.HandleRequest(restCtx(http.MethodPost, "/applications", map[string]any{
		"Name": "arn-app", "Author": "test", "Description": "test", "SemanticVersion": "1.0.0",
	}))
	require.NoError(t, err)
	m := respJSON(t, cr)
	assert.NotEmpty(t, m["ApplicationId"])
	// ApplicationId is the short ID (e.g. app-000000000001)
	assert.Contains(t, m["ApplicationId"].(string), "app-")
}

func TestSR_DeleteAppNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodDelete, "/applications/nonexistent", nil))
	require.Error(t, err)
}

func TestSR_ListApplicationsEmpty(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(restCtx(http.MethodGet, "/applications", nil))
	require.NoError(t, err)
	apps := respJSON(t, resp)["Applications"].([]any)
	assert.Len(t, apps, 0)
}

func TestSR_VersionHasTimestamp(t *testing.T) {
	s := newService()
	appID := createApp(t, s)

	verResp, err := s.HandleRequest(restCtx(http.MethodPut, "/applications/"+appID+"/versions/3.0.0", nil))
	require.NoError(t, err)
	m := respJSON(t, verResp)
	assert.NotEmpty(t, m["CreationTime"])
}
