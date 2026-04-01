package resourcegroups_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/resourcegroups"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.ResourceGroupsService { return svc.New("123456789012", "us-east-1") }
func restCtx(method, path string, body map[string]any) *service.RequestContext {
	var b []byte; if body != nil { b, _ = json.Marshal(body) }
	return &service.RequestContext{Region: "us-east-1", AccountID: "123456789012", Body: b,
		RawRequest: httptest.NewRequest(method, path, nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

func TestRG_CreateAndGetGroup(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{
		"Name": "my-group", "Description": "Test group", "Tags": map[string]any{"env": "test"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "my-group", m["Group"].(map[string]any)["Name"])

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/groups/my-group", nil))
	gm := respJSON(t, getResp)
	assert.Equal(t, "my-group", gm["Group"].(map[string]any)["Name"])
}

func TestRG_ListGroups(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{"Name": "g1"}))
	s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{"Name": "g2"}))

	resp, _ := s.HandleRequest(restCtx(http.MethodGet, "/groups", nil))
	m := respJSON(t, resp)
	assert.Len(t, m["Groups"].([]any), 2)
}

func TestRG_UpdateGroup(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{"Name": "upd-g"}))
	resp, err := s.HandleRequest(restCtx(http.MethodPut, "/groups/upd-g", map[string]any{"Description": "Updated"}))
	require.NoError(t, err)
	assert.Equal(t, "Updated", respJSON(t, resp)["Group"].(map[string]any)["Description"])
}

func TestRG_DeleteGroup(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{"Name": "del-g"}))
	resp, err := s.HandleRequest(restCtx(http.MethodDelete, "/groups/del-g", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, err = s.HandleRequest(restCtx(http.MethodGet, "/groups/del-g", nil))
	require.Error(t, err)
}

func TestRG_GroupAndUngroupResources(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{"Name": "res-g"}))

	grpResp, _ := s.HandleRequest(restCtx(http.MethodPost, "/group-resources", map[string]any{
		"Group": "res-g", "ResourceArns": []string{"arn:a", "arn:b"},
	}))
	gm := respJSON(t, grpResp)
	assert.Len(t, gm["Succeeded"].([]any), 2)

	listResp, _ := s.HandleRequest(restCtx(http.MethodPost, "/list-group-resources", map[string]any{"Group": "res-g"}))
	assert.Len(t, respJSON(t, listResp)["Resources"].([]any), 2)

	ungResp, _ := s.HandleRequest(restCtx(http.MethodPost, "/ungroup-resources", map[string]any{
		"Group": "res-g", "ResourceArns": []string{"arn:a"},
	}))
	assert.Len(t, respJSON(t, ungResp)["Succeeded"].([]any), 1)
}

func TestRG_TagAndGetTags(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{
		"Name": "tag-g", "Tags": map[string]any{"env": "prod"},
	}))
	arn := respJSON(t, cr)["Group"].(map[string]any)["GroupArn"].(string)

	tagResp, _ := s.HandleRequest(restCtx(http.MethodPut, "/resources/"+arn+"/tags", map[string]any{
		"Tags": map[string]any{"team": "platform"},
	}))
	assert.Equal(t, http.StatusOK, tagResp.StatusCode)

	getResp, _ := s.HandleRequest(restCtx(http.MethodGet, "/resources/"+arn+"/tags", nil))
	tags := respJSON(t, getResp)["Tags"].(map[string]any)
	assert.Equal(t, "platform", tags["team"])
}

func TestRG_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/groups/nonexistent", nil))
	require.Error(t, err)
}

func TestRG_ListGroupResourcesNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/list-group-resources", map[string]any{"Group": "nonexistent"}))
	require.Error(t, err)
}

func TestRG_DuplicateGroup(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{"Name": "dup-g"}))
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{"Name": "dup-g"}))
	require.Error(t, err)
}

func TestRG_SearchResources(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPost, "/groups", map[string]any{"Name": "search-g"}))
	s.HandleRequest(restCtx(http.MethodPost, "/group-resources", map[string]any{
		"Group": "search-g", "ResourceArns": []string{"arn:x"},
	}))

	resp, _ := s.HandleRequest(restCtx(http.MethodPost, "/resources/search", map[string]any{}))
	m := respJSON(t, resp)
	assert.Len(t, m["ResourceIdentifiers"].([]any), 1)
}
