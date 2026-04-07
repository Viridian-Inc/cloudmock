package pinpoint_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/pinpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.PinpointService { return svc.New("123456789012", "us-east-1") }
func restCtx(method, path string, body map[string]any) *service.RequestContext {
	var b []byte; if body != nil { b, _ = json.Marshal(body) }
	return &service.RequestContext{Region: "us-east-1", AccountID: "123456789012", Body: b,
		RawRequest: httptest.NewRequest(method, path, nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

func createApp(t *testing.T, s *svc.PinpointService, name string) string {
	resp, err := s.HandleRequest(restCtx(http.MethodPost, "/v1/apps", map[string]any{
		"CreateApplicationRequest": map[string]any{"Name": name},
	}))
	require.NoError(t, err)
	return respJSON(t, resp)["ApplicationResponse"].(map[string]any)["Id"].(string)
}

func TestPinpoint_CreateAndGetApp(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "test-app")
	assert.NotEmpty(t, appID)

	resp, _ := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID, nil))
	m := respJSON(t, resp)
	assert.Equal(t, "test-app", m["ApplicationResponse"].(map[string]any)["Name"])
}

func TestPinpoint_ListApps(t *testing.T) {
	s := newService()
	createApp(t, s, "a1"); createApp(t, s, "a2")
	resp, _ := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps", nil))
	m := respJSON(t, resp)
	assert.Len(t, m["ApplicationsResponse"].(map[string]any)["Item"].([]any), 2)
}

func TestPinpoint_DeleteApp(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "del-app")
	resp, err := s.HandleRequest(restCtx(http.MethodDelete, "/v1/apps/"+appID, nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, err = s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID, nil))
	require.Error(t, err)
}

func TestPinpoint_SegmentCRUD(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "seg-app")

	segResp, err := s.HandleRequest(restCtx(http.MethodPost, "/v1/apps/"+appID+"/segments", map[string]any{
		"WriteSegmentRequest": map[string]any{"Name": "my-segment"},
	}))
	require.NoError(t, err)
	segID := respJSON(t, segResp)["SegmentResponse"].(map[string]any)["Id"].(string)

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/segments/"+segID, nil))
	assert.Equal(t, "my-segment", respJSON(t, getResp)["SegmentResponse"].(map[string]any)["Name"])

	listResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/segments", nil))
	assert.Len(t, respJSON(t, listResp)["SegmentsResponse"].(map[string]any)["Item"].([]any), 1)

	delResp, err := s.HandleRequest(restCtx(http.MethodDelete, "/v1/apps/"+appID+"/segments/"+segID, nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)
}

func TestPinpoint_CampaignCRUD(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "camp-app")

	campResp, err := s.HandleRequest(restCtx(http.MethodPost, "/v1/apps/"+appID+"/campaigns", map[string]any{
		"WriteCampaignRequest": map[string]any{"Name": "my-campaign", "SegmentId": "seg-1"},
	}))
	require.NoError(t, err)
	cm := respJSON(t, campResp)
	campID := cm["CampaignResponse"].(map[string]any)["Id"].(string)
	assert.Equal(t, "SCHEDULED", cm["CampaignResponse"].(map[string]any)["State"].(map[string]any)["CampaignStatus"])

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/campaigns/"+campID, nil))
	assert.Equal(t, "my-campaign", respJSON(t, getResp)["CampaignResponse"].(map[string]any)["Name"])

	s.HandleRequest(restCtx(http.MethodDelete, "/v1/apps/"+appID+"/campaigns/"+campID, nil))
}

func TestPinpoint_JourneyLifecycle(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "journey-app")

	jResp, err := s.HandleRequest(restCtx(http.MethodPost, "/v1/apps/"+appID+"/journeys", map[string]any{
		"WriteJourneyRequest": map[string]any{"Name": "my-journey"},
	}))
	require.NoError(t, err)
	jID := respJSON(t, jResp)["JourneyResponse"].(map[string]any)["Id"].(string)

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/journeys/"+jID, nil))
	assert.Equal(t, "DRAFT", respJSON(t, getResp)["JourneyResponse"].(map[string]any)["State"])

	listResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/journeys", nil))
	assert.Len(t, respJSON(t, listResp)["JourneysResponse"].(map[string]any)["Item"].([]any), 1)
}

func TestPinpoint_Endpoint(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "ep-app")

	putResp, err := s.HandleRequest(restCtx(http.MethodPut, "/v1/apps/"+appID+"/endpoints/ep-1", map[string]any{
		"EndpointRequest": map[string]any{"ChannelType": "EMAIL", "Address": "user@example.com"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, putResp.StatusCode)

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/endpoints/ep-1", nil))
	ep := respJSON(t, getResp)["EndpointResponse"].(map[string]any)
	assert.Equal(t, "EMAIL", ep["ChannelType"])
}

func TestPinpoint_EndpointInvalidChannelType(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "ch-app")

	_, err := s.HandleRequest(restCtx(http.MethodPut, "/v1/apps/"+appID+"/endpoints/ep-bad", map[string]any{
		"EndpointRequest": map[string]any{"ChannelType": "INVALID_CHANNEL", "Address": "x"},
	}))
	require.Error(t, err)
}

func TestPinpoint_CampaignInitialState(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "state-app")
	campResp, err := s.HandleRequest(restCtx(http.MethodPost, "/v1/apps/"+appID+"/campaigns", map[string]any{
		"WriteCampaignRequest": map[string]any{"Name": "state-camp", "SegmentId": "seg-1"},
	}))
	require.NoError(t, err)
	cm := respJSON(t, campResp)
	state := cm["CampaignResponse"].(map[string]any)["State"].(map[string]any)
	assert.Equal(t, "SCHEDULED", state["CampaignStatus"])
}

func TestPinpoint_AppNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/nonexistent", nil))
	require.Error(t, err)
}

func TestPinpoint_GetCampaigns(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "camp-list-app")
	s.HandleRequest(restCtx(http.MethodPost, "/v1/apps/"+appID+"/campaigns", map[string]any{
		"WriteCampaignRequest": map[string]any{"Name": "camp-1", "SegmentId": "s1"},
	}))
	s.HandleRequest(restCtx(http.MethodPost, "/v1/apps/"+appID+"/campaigns", map[string]any{
		"WriteCampaignRequest": map[string]any{"Name": "camp-2", "SegmentId": "s2"},
	}))

	resp, err := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/campaigns", nil))
	require.NoError(t, err)
	items := respJSON(t, resp)["CampaignsResponse"].(map[string]any)["Item"].([]any)
	assert.Len(t, items, 2)
}

func TestPinpoint_GetSegments(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "seg-list-app")
	s.HandleRequest(restCtx(http.MethodPost, "/v1/apps/"+appID+"/segments", map[string]any{
		"WriteSegmentRequest": map[string]any{"Name": "seg-A"},
	}))
	s.HandleRequest(restCtx(http.MethodPost, "/v1/apps/"+appID+"/segments", map[string]any{
		"WriteSegmentRequest": map[string]any{"Name": "seg-B"},
	}))

	resp, err := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/segments", nil))
	require.NoError(t, err)
	items := respJSON(t, resp)["SegmentsResponse"].(map[string]any)["Item"].([]any)
	assert.Len(t, items, 2)
}

func TestPinpoint_DeleteCampaign(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "del-camp-app")
	campResp, err := s.HandleRequest(restCtx(http.MethodPost, "/v1/apps/"+appID+"/campaigns", map[string]any{
		"WriteCampaignRequest": map[string]any{"Name": "to-delete", "SegmentId": "s1"},
	}))
	require.NoError(t, err)
	campID := respJSON(t, campResp)["CampaignResponse"].(map[string]any)["Id"].(string)

	_, err = s.HandleRequest(restCtx(http.MethodDelete, "/v1/apps/"+appID+"/campaigns/"+campID, nil))
	require.NoError(t, err)

	_, err = s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/campaigns/"+campID, nil))
	require.Error(t, err)
}

func TestPinpoint_SegmentNotFound(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "notfound-app")
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/segments/nonexistent", nil))
	require.Error(t, err)
}

func TestPinpoint_CampaignNotFound(t *testing.T) {
	s := newService()
	appID := createApp(t, s, "notfound-app2")
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/v1/apps/"+appID+"/campaigns/nonexistent", nil))
	require.Error(t, err)
}

func TestPinpoint_CreateAppMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/v1/apps", map[string]any{
		"CreateApplicationRequest": map[string]any{},
	}))
	require.Error(t, err)
}
