package opensearch_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/opensearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.OpenSearchService {
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

func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestServiceName(t *testing.T) {
	assert.Equal(t, "opensearch", newService().Name())
}

func TestHealthCheck(t *testing.T) {
	assert.NoError(t, newService().HealthCheck())
}

func TestCreateDomain(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateDomain", map[string]any{
		"DomainName": "test-domain", "EngineVersion": "OpenSearch_2.11",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	ds := m["DomainStatus"].(map[string]any)
	assert.Equal(t, "test-domain", ds["DomainName"])
	assert.NotEmpty(t, ds["Endpoint"])
	assert.True(t, ds["Created"].(bool))
}

func TestCreateDomainDuplicate(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "dup"}))
	_, err := s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "dup"}))
	require.Error(t, err)
}

func TestDescribeDomain(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "desc-dom"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeDomain", map[string]any{"DomainName": "desc-dom"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	ds := m["DomainStatus"].(map[string]any)
	assert.Equal(t, "desc-dom", ds["DomainName"])
}

func TestDescribeDomainNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeDomain", map[string]any{"DomainName": "nope"}))
	require.Error(t, err)
}

func TestListDomainNames(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "d1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "d2"}))
	resp, err := s.HandleRequest(jsonCtx("ListDomainNames", map[string]any{}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	names := m["DomainNames"].([]any)
	assert.Len(t, names, 2)
}

func TestDeleteDomain(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "del-dom"}))
	resp, err := s.HandleRequest(jsonCtx("DeleteDomain", map[string]any{"DomainName": "del-dom"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	ds := m["DomainStatus"].(map[string]any)
	assert.True(t, ds["Deleted"].(bool))
}

func TestDeleteDomainNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteDomain", map[string]any{"DomainName": "ghost"}))
	require.Error(t, err)
}

func TestUpdateDomainConfig(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "upd-dom"}))
	resp, err := s.HandleRequest(jsonCtx("UpdateDomainConfig", map[string]any{
		"DomainName":    "upd-dom",
		"ClusterConfig": map[string]any{"InstanceType": "r6g.2xlarge.search", "InstanceCount": 3},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDescribeDomainConfig(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "cfg-dom"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeDomainConfig", map[string]any{"DomainName": "cfg-dom"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddTags(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "tag-dom"}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/tag-dom"
	resp, err := s.HandleRequest(jsonCtx("AddTags", map[string]any{
		"ARN": arn, "TagList": []map[string]string{{"Key": "env", "Value": "prod"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListTags(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{
		"DomainName": "ltag-dom", "TagList": []map[string]string{{"Key": "team", "Value": "data"}},
	}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/ltag-dom"
	resp, err := s.HandleRequest(jsonCtx("ListTags", map[string]any{"ARN": arn}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tags := m["TagList"].([]any)
	assert.Len(t, tags, 1)
}

func TestRemoveTags(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{
		"DomainName": "rmtag-dom", "TagList": []map[string]string{{"Key": "rm", "Value": "me"}},
	}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/rmtag-dom"
	resp, err := s.HandleRequest(jsonCtx("RemoveTags", map[string]any{"ARN": arn, "TagKeys": []string{"rm"}}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpgradeDomain(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "upg-dom"}))
	resp, err := s.HandleRequest(jsonCtx("UpgradeDomain", map[string]any{
		"DomainName": "upg-dom", "TargetVersion": "OpenSearch_2.13",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "SUCCEEDED", m["StepStatus"])
}

func TestGetUpgradeStatus(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "gupg-dom"}))
	_, _ = s.HandleRequest(jsonCtx("UpgradeDomain", map[string]any{"DomainName": "gupg-dom", "TargetVersion": "OpenSearch_2.13"}))
	resp, err := s.HandleRequest(jsonCtx("GetUpgradeStatus", map[string]any{"DomainName": "gupg-dom"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "SUCCEEDED", m["StepStatus"])
}

func TestGetUpgradeStatusNoUpgrade(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "noupg"}))
	resp, err := s.HandleRequest(jsonCtx("GetUpgradeStatus", map[string]any{"DomainName": "noupg"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "NOT_ELIGIBLE", m["StepStatus"])
}

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Behavioral tests ----

func TestIndexAndSearchDocuments(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "search-dom"}))

	// Index a document
	resp, err := s.HandleRequest(jsonCtx("IndexDocument", map[string]any{
		"DomainName": "search-dom", "Index": "products", "DocumentId": "doc1",
		"Document": map[string]any{"name": "Widget", "price": 9.99, "category": "tools"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "doc1", m["_id"])
	assert.Equal(t, "created", m["result"])

	// Index another document
	_, _ = s.HandleRequest(jsonCtx("IndexDocument", map[string]any{
		"DomainName": "search-dom", "Index": "products", "DocumentId": "doc2",
		"Document": map[string]any{"name": "Gadget", "price": 19.99, "category": "electronics"},
	}))

	// Search all documents
	resp, err = s.HandleRequest(jsonCtx("Search", map[string]any{
		"DomainName": "search-dom", "Index": "products",
	}))
	require.NoError(t, err)
	m = respJSON(t, resp)
	hits := m["hits"].(map[string]any)
	total := hits["total"].(map[string]any)
	assert.Equal(t, float64(2), total["value"])

	// Search with match query
	resp, err = s.HandleRequest(jsonCtx("Search", map[string]any{
		"DomainName": "search-dom", "Index": "products",
		"Query": map[string]any{"match": map[string]any{"category": "tools"}},
	}))
	require.NoError(t, err)
	m = respJSON(t, resp)
	hits = m["hits"].(map[string]any)
	total = hits["total"].(map[string]any)
	assert.Equal(t, float64(1), total["value"])
	hitList := hits["hits"].([]any)
	doc := hitList[0].(map[string]any)
	source := doc["_source"].(map[string]any)
	assert.Equal(t, "Widget", source["name"])
}

func TestIndexDocument_DomainNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("IndexDocument", map[string]any{
		"DomainName": "nonexistent", "Index": "test", "Document": map[string]any{},
	}))
	require.Error(t, err)
}

func TestClusterHealth_SingleNode(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{
		"DomainName":    "health-dom",
		"ClusterConfig": map[string]any{"InstanceCount": 1},
	}))
	resp, err := s.HandleRequest(jsonCtx("ClusterHealth", map[string]any{"DomainName": "health-dom"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "yellow", m["status"])
}

func TestClusterHealth_MultiNode(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{
		"DomainName":    "green-dom",
		"ClusterConfig": map[string]any{"InstanceCount": 3},
	}))
	resp, err := s.HandleRequest(jsonCtx("ClusterHealth", map[string]any{"DomainName": "green-dom"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "green", m["status"])
}

// ---- DescribeDomains ----

func TestDescribeDomains(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "dd-dom1"}))
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "dd-dom2"}))

	resp, err := s.HandleRequest(jsonCtx("DescribeDomains", map[string]any{
		"DomainNames": []any{"dd-dom1", "dd-dom2"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := respJSON(t, resp)
	list := m["DomainStatusList"].([]any)
	assert.Len(t, list, 2)
}

func TestDescribeDomains_Partial(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "partial-dom1"}))

	resp, err := s.HandleRequest(jsonCtx("DescribeDomains", map[string]any{
		"DomainNames": []any{"partial-dom1", "nonexistent-dom"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	list := m["DomainStatusList"].([]any)
	assert.Len(t, list, 1)
}

// ---- GetCompatibleVersions ----

func TestGetCompatibleVersions_All(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetCompatibleVersions", map[string]any{}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := respJSON(t, resp)
	list := m["CompatibleVersions"].([]any)
	assert.Greater(t, len(list), 0)
}

func TestGetCompatibleVersions_ForDomain(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{
		"DomainName": "compat-dom", "EngineVersion": "OpenSearch_2.7",
	}))

	resp, err := s.HandleRequest(jsonCtx("GetCompatibleVersions", map[string]any{
		"DomainName": "compat-dom",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	list := m["CompatibleVersions"].([]any)
	require.Len(t, list, 1)
	cv := list[0].(map[string]any)
	assert.Equal(t, "OpenSearch_2.7", cv["SourceVersion"])
}

// ---- VPC Endpoints ----

func TestCreateVpcEndpoint(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "vpc-dom"}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/vpc-dom"

	resp, err := s.HandleRequest(jsonCtx("CreateVpcEndpoint", map[string]any{
		"DomainArn": arn,
		"VpcOptions": map[string]any{
			"VPCId":            "vpc-12345",
			"SubnetIds":        []any{"subnet-aaa"},
			"SecurityGroupIds": []any{"sg-bbb"},
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := respJSON(t, resp)
	ep := m["VpcEndpoint"].(map[string]any)
	assert.NotEmpty(t, ep["VpcEndpointId"])
	assert.Equal(t, arn, ep["DomainArn"])
}

func TestCreateVpcEndpoint_MissingDomainArn(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateVpcEndpoint", map[string]any{}))
	require.Error(t, err)
}

func TestDescribeVpcEndpoints(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "dvpc-dom"}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/dvpc-dom"

	createResp, _ := s.HandleRequest(jsonCtx("CreateVpcEndpoint", map[string]any{
		"DomainArn":  arn,
		"VpcOptions": map[string]any{"VPCId": "vpc-abc"},
	}))
	cm := respJSON(t, createResp)
	epID := cm["VpcEndpoint"].(map[string]any)["VpcEndpointId"].(string)

	resp, err := s.HandleRequest(jsonCtx("DescribeVpcEndpoints", map[string]any{
		"VpcEndpointIds": []any{epID},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	endpoints := m["VpcEndpoints"].([]any)
	assert.Len(t, endpoints, 1)
}

func TestListVpcEndpoints(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "lvpc-dom"}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/lvpc-dom"

	for i := 0; i < 2; i++ {
		_, _ = s.HandleRequest(jsonCtx("CreateVpcEndpoint", map[string]any{
			"DomainArn":  arn,
			"VpcOptions": map[string]any{"VPCId": "vpc-abc"},
		}))
	}

	resp, err := s.HandleRequest(jsonCtx("ListVpcEndpoints", map[string]any{
		"DomainArn": arn,
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	endpoints := m["VpcEndpoints"].([]any)
	assert.Len(t, endpoints, 2)
}

func TestDeleteVpcEndpoint(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "delvpc-dom"}))
	arn := "arn:aws:es:us-east-1:123456789012:domain/delvpc-dom"

	createResp, _ := s.HandleRequest(jsonCtx("CreateVpcEndpoint", map[string]any{
		"DomainArn":  arn,
		"VpcOptions": map[string]any{},
	}))
	cm := respJSON(t, createResp)
	epID := cm["VpcEndpoint"].(map[string]any)["VpcEndpointId"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteVpcEndpoint", map[string]any{
		"VpcEndpointId": epID,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteVpcEndpoint_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteVpcEndpoint", map[string]any{
		"VpcEndpointId": "nonexistent-endpoint",
	}))
	require.Error(t, err)
}

func TestDomainEndpointFormat(t *testing.T) {
	s := newService()
	_, _ = s.HandleRequest(jsonCtx("CreateDomain", map[string]any{"DomainName": "ep-dom"}))
	resp, err := s.HandleRequest(jsonCtx("DescribeDomain", map[string]any{"DomainName": "ep-dom"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	ds := m["DomainStatus"].(map[string]any)
	endpoint := ds["Endpoint"].(string)
	assert.Contains(t, endpoint, "ep-dom")
	assert.Contains(t, endpoint, "us-east-1")
	assert.Contains(t, endpoint, "es.amazonaws.com")
}
