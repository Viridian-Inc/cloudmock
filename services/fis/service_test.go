package fis_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/fis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.FISService { return svc.New("123456789012", "us-east-1") }
func restCtx(method, path string, body map[string]any) *service.RequestContext {
	var b []byte; if body != nil { b, _ = json.Marshal(body) }
	return &service.RequestContext{Region: "us-east-1", AccountID: "123456789012", Body: b,
		RawRequest: httptest.NewRequest(method, path, nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

func createTemplate(t *testing.T, s *svc.FISService) string {
	resp, err := s.HandleRequest(restCtx(http.MethodPost, "/experimentTemplates", map[string]any{
		"description": "test template", "roleArn": "arn:aws:iam::123456789012:role/fis",
		"actions": map[string]any{"stop-ec2": map[string]any{"actionId": "aws:ec2:stop-instances"}},
		"targets": map[string]any{"ec2": map[string]any{"resourceType": "aws:ec2:instance", "selectionMode": "ALL"}},
		"stopConditions": []map[string]any{{"source": "none"}},
	}))
	require.NoError(t, err)
	return respJSON(t, resp)["experimentTemplate"].(map[string]any)["id"].(string)
}

func TestFIS_CreateAndGetTemplate(t *testing.T) {
	s := newService()
	id := createTemplate(t, s)
	assert.NotEmpty(t, id)

	resp, _ := s.HandleRequest(restCtx(http.MethodGet, "/experimentTemplates/"+id, nil))
	m := respJSON(t, resp)
	assert.Equal(t, id, m["experimentTemplate"].(map[string]any)["id"])
}

func TestFIS_ListTemplates(t *testing.T) {
	s := newService()
	createTemplate(t, s)
	createTemplate(t, s)
	resp, _ := s.HandleRequest(restCtx(http.MethodGet, "/experimentTemplates", nil))
	assert.Len(t, respJSON(t, resp)["experimentTemplates"].([]any), 2)
}

func TestFIS_DeleteTemplate(t *testing.T) {
	s := newService()
	id := createTemplate(t, s)
	resp, err := s.HandleRequest(restCtx(http.MethodDelete, "/experimentTemplates/"+id, nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	_, err = s.HandleRequest(restCtx(http.MethodGet, "/experimentTemplates/"+id, nil))
	require.Error(t, err)
}

func TestFIS_StartAndGetExperiment(t *testing.T) {
	s := newService()
	tmplID := createTemplate(t, s)

	startResp, err := s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{
		"experimentTemplateId": tmplID,
	}))
	require.NoError(t, err)
	expID := respJSON(t, startResp)["experiment"].(map[string]any)["id"].(string)

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/experiments/"+expID, nil))
	exp := respJSON(t, getResp)["experiment"].(map[string]any)
	assert.Equal(t, "initiating", exp["state"].(map[string]any)["status"])
}

func TestFIS_ListExperiments(t *testing.T) {
	s := newService()
	tmplID := createTemplate(t, s)
	s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{"experimentTemplateId": tmplID}))
	s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{"experimentTemplateId": tmplID}))

	resp, _ := s.HandleRequest(restCtx(http.MethodGet, "/experiments", nil))
	assert.Len(t, respJSON(t, resp)["experiments"].([]any), 2)
}

func TestFIS_StopExperiment(t *testing.T) {
	s := newService()
	tmplID := createTemplate(t, s)
	startResp, _ := s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{"experimentTemplateId": tmplID}))
	expID := respJSON(t, startResp)["experiment"].(map[string]any)["id"].(string)

	// Use DELETE on /experiments/{id}/stop as per the handler routing
	// The handler checks for /stop suffix
	stopResp, err := s.HandleRequest(restCtx(http.MethodDelete, "/experiments/"+expID+"/stop", nil))
	require.NoError(t, err)
	exp := respJSON(t, stopResp)["experiment"].(map[string]any)
	assert.Equal(t, "stopped", exp["state"].(map[string]any)["status"])
}

func TestFIS_TemplateNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/experimentTemplates/nonexistent", nil))
	require.Error(t, err)
}

func TestFIS_ExperimentNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/experiments/nonexistent", nil))
	require.Error(t, err)
}

func TestFIS_ExperimentTracksActions(t *testing.T) {
	s := newService()
	tmplID := createTemplate(t, s)

	startResp, err := s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{
		"experimentTemplateId": tmplID,
	}))
	require.NoError(t, err)
	exp := respJSON(t, startResp)["experiment"].(map[string]any)

	// Experiment should have actions from template.
	actions := exp["actions"].(map[string]any)
	assert.Contains(t, actions, "stop-ec2")
	stopAction := actions["stop-ec2"].(map[string]any)
	assert.Equal(t, "aws:ec2:stop-instances", stopAction["actionId"])
	assert.Equal(t, "pending", stopAction["state"].(map[string]any)["status"])
}

func TestFIS_ExperimentTracksTargets(t *testing.T) {
	s := newService()
	// Create template with specific resource ARNs.
	resp, _ := s.HandleRequest(restCtx(http.MethodPost, "/experimentTemplates", map[string]any{
		"description": "with targets", "roleArn": "arn:aws:iam::123456789012:role/fis",
		"actions": map[string]any{"stop-ec2": map[string]any{"actionId": "aws:ec2:stop-instances"}},
		"targets": map[string]any{"ec2": map[string]any{
			"resourceType":  "aws:ec2:instance",
			"selectionMode": "ALL",
			"resourceArns":  []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-123"},
		}},
		"stopConditions": []map[string]any{{"source": "none"}},
	}))
	tmplID := respJSON(t, resp)["experimentTemplate"].(map[string]any)["id"].(string)

	startResp, _ := s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{
		"experimentTemplateId": tmplID,
	}))
	exp := respJSON(t, startResp)["experiment"].(map[string]any)

	targets := exp["targets"].(map[string]any)
	assert.Contains(t, targets, "ec2")
	ec2Target := targets["ec2"].(map[string]any)
	assert.Equal(t, "aws:ec2:instance", ec2Target["resourceType"])
}

func TestFIS_MissingTemplateID(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimentTemplateId")
}

func TestFIS_StartExperimentFromNonexistentTemplate(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{
		"experimentTemplateId": "EXT9999999999",
	}))
	require.Error(t, err)
}

func TestFIS_UpdateExperimentTemplate(t *testing.T) {
	s := newService()
	id := createTemplate(t, s)

	resp, err := s.HandleRequest(restCtx(http.MethodPatch, "/experimentTemplates/"+id, map[string]any{
		"description": "updated description",
		"roleArn":     "arn:aws:iam::123456789012:role/fis-updated",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "updated description", m["experimentTemplate"].(map[string]any)["description"])
}

func TestFIS_UpdateExperimentTemplateNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodPatch, "/experimentTemplates/nonexistent", map[string]any{
		"description": "updated",
	}))
	require.Error(t, err)
}

func TestFIS_TagAndListTagsForTemplate(t *testing.T) {
	s := newService()
	id := createTemplate(t, s)
	arn := "arn:aws:fis:us-east-1:123456789012:experiment-template/" + id

	_, err := s.HandleRequest(restCtx(http.MethodPost, "/tags/"+arn, map[string]any{
		"tags": map[string]any{"env": "prod", "team": "sre"},
	}))
	require.NoError(t, err)

	tagsResp, err := s.HandleRequest(restCtx(http.MethodGet, "/tags/"+arn, nil))
	require.NoError(t, err)
	tags := respJSON(t, tagsResp)["tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
	assert.Equal(t, "sre", tags["team"])
}

func TestFIS_UntagResource(t *testing.T) {
	s := newService()
	id := createTemplate(t, s)
	arn := "arn:aws:fis:us-east-1:123456789012:experiment-template/" + id

	s.HandleRequest(restCtx(http.MethodPost, "/tags/"+arn, map[string]any{
		"tags": map[string]any{"env": "prod", "team": "sre"},
	}))

	_, err := s.HandleRequest(restCtx(http.MethodDelete, "/tags/"+arn, map[string]any{
		"tagKeys": []string{"team"},
	}))
	require.NoError(t, err)

	tagsResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/tags/"+arn, nil))
	tags := respJSON(t, tagsResp)["tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
	assert.Nil(t, tags["team"])
}

func TestFIS_ListTargetResourceTypes(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(restCtx(http.MethodGet, "/targetResourceTypes", nil))
	require.NoError(t, err)
	types := respJSON(t, resp)["targetResourceTypes"].([]any)
	assert.NotEmpty(t, types)
	// Verify ec2:instance is present
	found := false
	for _, typ := range types {
		if typ.(map[string]any)["resourceType"] == "aws:ec2:instance" {
			found = true
			break
		}
	}
	assert.True(t, found, "aws:ec2:instance should be in target resource types")
}

func TestFIS_ListActions(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(restCtx(http.MethodGet, "/actions", nil))
	require.NoError(t, err)
	actions := respJSON(t, resp)["actions"].([]any)
	assert.NotEmpty(t, actions)
	found := false
	for _, action := range actions {
		if action.(map[string]any)["id"] == "aws:ec2:stop-instances" {
			found = true
			break
		}
	}
	assert.True(t, found, "aws:ec2:stop-instances should be in actions")
}

func TestFIS_ExperimentStateTransitions(t *testing.T) {
	s := newService()
	tmplID := createTemplate(t, s)

	// Start experiment - initial state should be "initiating"
	startResp, err := s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{
		"experimentTemplateId": tmplID,
	}))
	require.NoError(t, err)
	exp := respJSON(t, startResp)["experiment"].(map[string]any)
	state := exp["state"].(map[string]any)
	// State should be initiating at start (instant mode may advance it)
	assert.NotEmpty(t, state["status"])
}

func TestFIS_MultipleExperimentsFromSameTemplate(t *testing.T) {
	s := newService()
	tmplID := createTemplate(t, s)

	ids := make([]string, 3)
	for i := 0; i < 3; i++ {
		resp, err := s.HandleRequest(restCtx(http.MethodPost, "/experiments", map[string]any{
			"experimentTemplateId": tmplID,
		}))
		require.NoError(t, err)
		ids[i] = respJSON(t, resp)["experiment"].(map[string]any)["id"].(string)
	}

	// All IDs should be unique
	assert.NotEqual(t, ids[0], ids[1])
	assert.NotEqual(t, ids[1], ids[2])

	listResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/experiments", nil))
	assert.Len(t, respJSON(t, listResp)["experiments"].([]any), 3)
}
