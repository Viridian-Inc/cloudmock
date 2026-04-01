package es_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/es"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.ESService {
	return svc.New("123456789012", "us-east-1")
}

func queryCtx(action string, params map[string]string) *service.RequestContext {
	vals := url.Values{}
	vals.Set("Action", action)
	for k, v := range params {
		vals.Set(k, v)
	}
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       []byte(vals.Encode()),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func TestServiceName(t *testing.T) {
	assert.Equal(t, "es", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestCreateElasticsearchDomain(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{
		"DomainName": "test-domain", "ElasticsearchVersion": "7.10",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateElasticsearchDomainDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "dup"}))
	_, err := s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "dup"}))
	require.Error(t, err)
}

func TestCreateElasticsearchDomainMissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{}))
	require.Error(t, err)
}

func TestDescribeElasticsearchDomain(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "desc-dom"}))
	resp, err := s.HandleRequest(queryCtx("DescribeElasticsearchDomain", map[string]string{"DomainName": "desc-dom"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeElasticsearchDomainNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("DescribeElasticsearchDomain", map[string]string{"DomainName": "nope"}))
	require.Error(t, err)
}

func TestListDomainNames(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "d1"}))
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "d2"}))
	resp, err := s.HandleRequest(queryCtx("ListDomainNames", map[string]string{}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteElasticsearchDomain(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "del-dom"}))
	resp, err := s.HandleRequest(queryCtx("DeleteElasticsearchDomain", map[string]string{"DomainName": "del-dom"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteElasticsearchDomainNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("DeleteElasticsearchDomain", map[string]string{"DomainName": "ghost"}))
	require.Error(t, err)
}

func TestUpdateElasticsearchDomainConfig(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "upd-dom"}))
	resp, err := s.HandleRequest(queryCtx("UpdateElasticsearchDomainConfig", map[string]string{
		"DomainName": "upd-dom", "ElasticsearchClusterConfig.InstanceType": "r6g.2xlarge.elasticsearch",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeElasticsearchDomainConfig(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "cfg-dom"}))
	resp, err := s.HandleRequest(queryCtx("DescribeElasticsearchDomainConfig", map[string]string{"DomainName": "cfg-dom"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddTags(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "tag-dom"}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/tag-dom"
	resp, err := s.HandleRequest(queryCtx("AddTags", map[string]string{
		"ARN": arn, "Tags.member.1.Key": "env", "Tags.member.1.Value": "prod",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListTags(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "ltag-dom"}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/ltag-dom"
	_, _ = s.HandleRequest(queryCtx("AddTags", map[string]string{
		"ARN": arn, "Tags.member.1.Key": "team", "Tags.member.1.Value": "data",
	}))
	resp, err := s.HandleRequest(queryCtx("ListTags", map[string]string{"ARN": arn}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRemoveTags(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "rmtag-dom"}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/rmtag-dom"
	_, _ = s.HandleRequest(queryCtx("AddTags", map[string]string{
		"ARN": arn, "Tags.member.1.Key": "rm", "Tags.member.1.Value": "me",
	}))
	resp, err := s.HandleRequest(queryCtx("RemoveTags", map[string]string{
		"ARN": arn, "TagKeys.member.1": "rm",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(queryCtx("BogusAction", map[string]string{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Behavioral tests ----

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

func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestESIndexAndSearch(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "es-search"}))

	// Index document
	resp, err := s.HandleRequest(jsonCtx("IndexDocument", map[string]any{
		"DomainName": "es-search", "Index": "logs", "DocumentId": "1",
		"Document": map[string]any{"level": "error", "message": "disk full"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "1", m["_id"])

	// Index another
	_, _ = s.HandleRequest(jsonCtx("IndexDocument", map[string]any{
		"DomainName": "es-search", "Index": "logs", "DocumentId": "2",
		"Document": map[string]any{"level": "info", "message": "started"},
	}))

	// Search with match
	resp, err = s.HandleRequest(jsonCtx("Search", map[string]any{
		"DomainName": "es-search", "Index": "logs",
		"Query": map[string]any{"match": map[string]any{"level": "error"}},
	}))
	require.NoError(t, err)
	m = respJSON(t, resp)
	hits := m["hits"].(map[string]any)
	total := hits["total"].(map[string]any)
	assert.Equal(t, float64(1), total["value"])
}

func TestESClusterHealth(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{
		"DomainName": "health-es", "ElasticsearchClusterConfig.InstanceCount": "3",
	}))
	resp, err := s.HandleRequest(jsonCtx("ClusterHealth", map[string]any{"DomainName": "health-es"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "green", m["status"])
}

func TestESDomainEndpoint(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(queryCtx("CreateElasticsearchDomain", map[string]string{"DomainName": "ep-es"}))
	resp, err := s.HandleRequest(queryCtx("DescribeElasticsearchDomain", map[string]string{"DomainName": "ep-es"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
