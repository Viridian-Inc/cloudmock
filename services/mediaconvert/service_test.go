package mediaconvert_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/mediaconvert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.MediaConvertService { return svc.New("123456789012", "us-east-1") }
func restCtx(method, path string, body map[string]any) *service.RequestContext {
	var b []byte; if body != nil { b, _ = json.Marshal(body) }
	return &service.RequestContext{Region: "us-east-1", AccountID: "123456789012", Body: b,
		RawRequest: httptest.NewRequest(method, path, nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

func TestMC_CreateAndGetJob(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobs", map[string]any{
		"role": "arn:aws:iam::123456789012:role/mc", "queue": "Default",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	jobID := respJSON(t, resp)["job"].(map[string]any)["id"].(string)

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/jobs/"+jobID, nil))
	job := respJSON(t, getResp)["job"].(map[string]any)
	assert.Equal(t, "SUBMITTED", job["status"])
}

func TestMC_ListJobs(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobs", map[string]any{"role": "r"}))
	s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobs", map[string]any{"role": "r"}))

	resp, _ := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/jobs", nil))
	assert.Len(t, respJSON(t, resp)["jobs"].([]any), 2)
}

func TestMC_CancelJob(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobs", map[string]any{"role": "r"}))
	jobID := respJSON(t, cr)["job"].(map[string]any)["id"].(string)

	resp, err := s.HandleRequest(restCtx(http.MethodDelete, "/2017-08-29/jobs/"+jobID, nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMC_JobTemplateCRUD(t *testing.T) {
	s := newService()
	cr, err := s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobTemplates", map[string]any{
		"name": "my-tmpl", "description": "test",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, cr.StatusCode)

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/jobTemplates/my-tmpl", nil))
	assert.Equal(t, "my-tmpl", respJSON(t, getResp)["jobTemplate"].(map[string]any)["name"])

	listResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/jobTemplates", nil))
	assert.Len(t, respJSON(t, listResp)["jobTemplates"].([]any), 1)

	delResp, _ := s.HandleRequest(restCtx(http.MethodDelete, "/2017-08-29/jobTemplates/my-tmpl", nil))
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)
}

func TestMC_PresetCRUD(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/presets", map[string]any{
		"name": "my-preset", "description": "test preset",
	}))

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/presets/my-preset", nil))
	assert.Equal(t, "my-preset", respJSON(t, getResp)["preset"].(map[string]any)["name"])

	s.HandleRequest(restCtx(http.MethodDelete, "/2017-08-29/presets/my-preset", nil))
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/presets/my-preset", nil))
	require.Error(t, err)
}

func TestMC_QueueCRUD(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/queues", map[string]any{
		"name": "custom-q", "description": "Custom queue",
	}))
	assert.Equal(t, http.StatusCreated, cr.StatusCode)

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/queues/custom-q", nil))
	q := respJSON(t, getResp)["queue"].(map[string]any)
	assert.Equal(t, "ACTIVE", q["status"])

	listResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/queues", nil))
	assert.GreaterOrEqual(t, len(respJSON(t, listResp)["queues"].([]any)), 2) // Default + custom

	s.HandleRequest(restCtx(http.MethodDelete, "/2017-08-29/queues/custom-q", nil))
}

func TestMC_JobNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/jobs/nonexistent", nil))
	require.Error(t, err)
}

func TestMC_DuplicateTemplate(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobTemplates", map[string]any{"name": "dup"}))
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobTemplates", map[string]any{"name": "dup"}))
	require.Error(t, err)
}

func TestMC_InvalidInputURI(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobs", map[string]any{
		"role": "arn:aws:iam::123456789012:role/mc", "queue": "Default",
		"settings": map[string]any{
			"inputs": []any{map[string]any{"fileInput": "http://example.com/video.mp4"}},
		},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "s3://")
}

func TestMC_ValidS3InputURI(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobs", map[string]any{
		"role": "arn:aws:iam::123456789012:role/mc", "queue": "Default",
		"settings": map[string]any{
			"inputs": []any{map[string]any{"fileInput": "s3://my-bucket/input.mp4"}},
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestMC_NonexistentQueue(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobs", map[string]any{
		"role": "arn:aws:iam::123456789012:role/mc", "queue": "nonexistent-queue",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Queue not found")
}

func TestMC_UpdateQueue(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/queues", map[string]any{
		"name": "updateable-q",
	}))

	resp, err := s.HandleRequest(restCtx(http.MethodPut, "/2017-08-29/queues/updateable-q", map[string]any{
		"description": "updated description",
	}))
	require.NoError(t, err)
	q := respJSON(t, resp)["queue"].(map[string]any)
	assert.Equal(t, "updateable-q", q["name"])
}

func TestMC_DeleteQueueNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodDelete, "/2017-08-29/queues/nonexistent", nil))
	require.Error(t, err)
}

func TestMC_TemplateNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/jobTemplates/nonexistent", nil))
	require.Error(t, err)
}

func TestMC_PresetNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/presets/nonexistent", nil))
	require.Error(t, err)
}

func TestMC_ListPresets(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/presets", map[string]any{"name": "p1"}))
	s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/presets", map[string]any{"name": "p2"}))

	resp, err := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/presets", nil))
	require.NoError(t, err)
	presets := respJSON(t, resp)["presets"].([]any)
	assert.Len(t, presets, 2)
}

func TestMC_ListJobTemplates(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobTemplates", map[string]any{"name": "tmpl-a"}))
	s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobTemplates", map[string]any{"name": "tmpl-b"}))

	resp, err := s.HandleRequest(restCtx(http.MethodGet, "/2017-08-29/jobTemplates", nil))
	require.NoError(t, err)
	templates := respJSON(t, resp)["jobTemplates"].([]any)
	assert.Len(t, templates, 2)
}

func TestMC_JobHasCreatedAt(t *testing.T) {
	s := newService()
	cr, err := s.HandleRequest(restCtx(http.MethodPost, "/2017-08-29/jobs", map[string]any{"role": "r"}))
	require.NoError(t, err)
	job := respJSON(t, cr)["job"].(map[string]any)
	assert.NotEmpty(t, job["createdAt"])
}
