package tagging_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/tagging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.TaggingService { return svc.New("123456789012", "us-east-1") }

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action: action, Region: "us-east-1", AccountID: "123456789012", Body: bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestTagging_TagAndGetResources(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-123"},
		"Tags":            map[string]any{"env": "prod", "team": "platform"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotNil(t, m["FailedResourcesMap"])

	getResp, err := s.HandleRequest(jsonCtx("GetResources", map[string]any{}))
	require.NoError(t, err)
	gm := respJSON(t, getResp)
	resources := gm["ResourceTagMappingList"].([]any)
	assert.Len(t, resources, 1)
	assert.Equal(t, "arn:aws:ec2:us-east-1:123456789012:instance/i-123", resources[0].(map[string]any)["ResourceARN"])
}

func TestTagging_GetTagKeys(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:s3:::my-bucket"},
		"Tags":            map[string]any{"env": "dev", "project": "cloudmock"},
	}))

	resp, err := s.HandleRequest(jsonCtx("GetTagKeys", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	keys := m["TagKeys"].([]any)
	assert.Len(t, keys, 2)
}

func TestTagging_GetTagValues(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:s3:::bucket1"},
		"Tags":            map[string]any{"env": "staging"},
	}))

	resp, err := s.HandleRequest(jsonCtx("GetTagValues", map[string]any{"Key": "env"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	values := m["TagValues"].([]any)
	assert.Contains(t, values, "staging")
}

func TestTagging_UntagResources(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-456"},
		"Tags":            map[string]any{"env": "test", "team": "backend"},
	}))

	resp, err := s.HandleRequest(jsonCtx("UntagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-456"},
		"TagKeys":         []string{"team"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	keysResp, _ := s.HandleRequest(jsonCtx("GetTagKeys", nil))
	km := respJSON(t, keysResp)
	keys := km["TagKeys"].([]any)
	assert.Contains(t, keys, "env")
}

func TestTagging_FilterByTag(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:r1"}, "Tags": map[string]any{"env": "prod"},
	}))
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:r2"}, "Tags": map[string]any{"env": "dev"},
	}))

	resp, _ := s.HandleRequest(jsonCtx("GetResources", map[string]any{
		"TagFilters": []map[string]any{{"Key": "env", "Values": []string{"prod"}}},
	}))
	m := respJSON(t, resp)
	assert.Len(t, m["ResourceTagMappingList"].([]any), 1)
}

func TestTagging_MissingKeyForGetTagValues(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetTagValues", map[string]any{}))
	require.Error(t, err)
}

func TestTagging_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", nil))
	require.Error(t, err)
}

func TestTagging_MultipleResources(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:a", "arn:b", "arn:c"},
		"Tags":            map[string]any{"tier": "free"},
	}))

	resp, _ := s.HandleRequest(jsonCtx("GetResources", nil))
	m := respJSON(t, resp)
	assert.Len(t, m["ResourceTagMappingList"].([]any), 3)
}

func TestTagging_ReservedKeyPrefix(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:s3:::test"},
		"Tags":            map[string]any{"aws:internal": "blocked"},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved")
}

func TestTagging_EmptyARNList(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{},
		"Tags":            map[string]any{"env": "test"},
	}))
	require.Error(t, err)
}

func TestTagging_EmptyTags(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:s3:::test"},
		"Tags":            map[string]any{},
	}))
	require.Error(t, err)
}

func TestTagging_ResourceTypeFilter(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-1"},
		"Tags":            map[string]any{"env": "prod"},
	}))
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:s3:::my-bucket"},
		"Tags":            map[string]any{"env": "prod"},
	}))

	// Filter by ec2 only
	resp, _ := s.HandleRequest(jsonCtx("GetResources", map[string]any{
		"ResourceTypeFilters": []string{"ec2"},
	}))
	m := respJSON(t, resp)
	resources := m["ResourceTagMappingList"].([]any)
	assert.Len(t, resources, 1)
	assert.Contains(t, resources[0].(map[string]any)["ResourceARN"], "ec2")
}

func TestTagging_GetComplianceSummary(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:s3:::test"},
		"Tags":            map[string]any{"env": "prod"},
	}))
	resp, err := s.HandleRequest(jsonCtx("GetComplianceSummary", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	summaryList := m["SummaryList"].([]any)
	assert.Len(t, summaryList, 1)
}

func TestTagging_SetLocator(t *testing.T) {
	s := newService()
	// Just verify SetLocator doesn't panic with nil
	s.SetLocator(nil)
}

func TestTagging_GetTagValuesEmpty(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetTagValues", map[string]any{"Key": "nonexistent"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	values := m["TagValues"].([]any)
	assert.Len(t, values, 0)
}

func TestTagging_UntagResourcesMultiple(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("TagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:s3:::bucket-a", "arn:aws:s3:::bucket-b"},
		"Tags":            map[string]any{"env": "test"},
	}))

	resp, err := s.HandleRequest(jsonCtx("UntagResources", map[string]any{
		"ResourceARNList": []string{"arn:aws:s3:::bucket-a"},
		"TagKeys":         []string{"env"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotNil(t, m["FailedResourcesMap"])
}
