package wafregional_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/wafregional"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.WAFRegionalService {
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
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func createWebACL(t *testing.T, s *svc.WAFRegionalService, name string) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          name,
		"MetricName":    name + "Metric",
		"DefaultAction": map[string]any{"Type": "ALLOW"},
		"ChangeToken":   "token",
	}))
	require.NoError(t, err)
	return respBody(t, resp)["WebACL"].(map[string]any)["WebACLId"].(string)
}

func TestWAFRegional_CreateWebACL(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          "test-acl",
		"MetricName":    "testMetric",
		"DefaultAction": map[string]any{"Type": "BLOCK"},
		"ChangeToken":   "ct",
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	acl := m["WebACL"].(map[string]any)
	assert.Equal(t, "test-acl", acl["Name"])
	assert.NotEmpty(t, acl["WebACLId"])
	assert.NotEmpty(t, acl["WebACLArn"])
	assert.NotEmpty(t, m["ChangeToken"])
}

func TestWAFRegional_GetWebACL(t *testing.T) {
	s := newService()
	id := createWebACL(t, s, "get-acl")
	resp, err := s.HandleRequest(jsonCtx("GetWebACL", map[string]any{"WebACLId": id}))
	require.NoError(t, err)
	acl := respBody(t, resp)["WebACL"].(map[string]any)
	assert.Equal(t, "get-acl", acl["Name"])
}

func TestWAFRegional_GetWebACL_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetWebACL", map[string]any{"WebACLId": "nonexistent"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFNonexistentItemException")
}

func TestWAFRegional_ListWebACLs(t *testing.T) {
	s := newService()
	createWebACL(t, s, "acl-1")
	createWebACL(t, s, "acl-2")
	resp, _ := s.HandleRequest(jsonCtx("ListWebACLs", map[string]any{}))
	acls := respBody(t, resp)["WebACLs"].([]any)
	assert.Len(t, acls, 2)
}

func TestWAFRegional_DeleteWebACL(t *testing.T) {
	s := newService()
	id := createWebACL(t, s, "del-acl")
	_, err := s.HandleRequest(jsonCtx("DeleteWebACL", map[string]any{
		"WebACLId": id, "ChangeToken": "ct",
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetWebACL", map[string]any{"WebACLId": id}))
	require.Error(t, err)
}

func TestWAFRegional_CreateRule(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateRule", map[string]any{
		"Name": "test-rule", "MetricName": "testRuleMetric", "ChangeToken": "ct",
	}))
	require.NoError(t, err)
	rule := respBody(t, resp)["Rule"].(map[string]any)
	assert.Equal(t, "test-rule", rule["Name"])
	assert.NotEmpty(t, rule["RuleId"])
}

func TestWAFRegional_GetRule(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateRule", map[string]any{
		"Name": "get-rule", "MetricName": "m", "ChangeToken": "ct",
	}))
	ruleID := respBody(t, resp)["Rule"].(map[string]any)["RuleId"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetRule", map[string]any{"RuleId": ruleID}))
	require.NoError(t, err)
	rule := respBody(t, resp)["Rule"].(map[string]any)
	assert.Equal(t, "get-rule", rule["Name"])
}

func TestWAFRegional_ListRules(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateRule", map[string]any{"Name": "r1", "MetricName": "m1", "ChangeToken": "ct"}))
	s.HandleRequest(jsonCtx("CreateRule", map[string]any{"Name": "r2", "MetricName": "m2", "ChangeToken": "ct"}))
	resp, _ := s.HandleRequest(jsonCtx("ListRules", map[string]any{}))
	rules := respBody(t, resp)["Rules"].([]any)
	assert.Len(t, rules, 2)
}

func TestWAFRegional_DeleteRule(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateRule", map[string]any{
		"Name": "del-rule", "MetricName": "m", "ChangeToken": "ct",
	}))
	ruleID := respBody(t, resp)["Rule"].(map[string]any)["RuleId"].(string)
	_, err := s.HandleRequest(jsonCtx("DeleteRule", map[string]any{
		"RuleId": ruleID, "ChangeToken": "ct",
	}))
	require.NoError(t, err)
}

func TestWAFRegional_IPSet_CRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateIPSet", map[string]any{
		"Name": "test-ipset", "ChangeToken": "ct",
	}))
	require.NoError(t, err)
	ipSetID := respBody(t, resp)["IPSet"].(map[string]any)["IPSetId"].(string)

	resp, _ = s.HandleRequest(jsonCtx("GetIPSet", map[string]any{"IPSetId": ipSetID}))
	ipSet := respBody(t, resp)["IPSet"].(map[string]any)
	assert.Equal(t, "test-ipset", ipSet["Name"])

	resp, _ = s.HandleRequest(jsonCtx("ListIPSets", map[string]any{}))
	assert.Len(t, respBody(t, resp)["IPSets"].([]any), 1)

	_, err = s.HandleRequest(jsonCtx("DeleteIPSet", map[string]any{
		"IPSetId": ipSetID, "ChangeToken": "ct",
	}))
	require.NoError(t, err)
}

func TestWAFRegional_ByteMatchSet_CRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateByteMatchSet", map[string]any{
		"Name": "test-bms", "ChangeToken": "ct",
	}))
	require.NoError(t, err)
	bmsID := respBody(t, resp)["ByteMatchSet"].(map[string]any)["ByteMatchSetId"].(string)

	resp, _ = s.HandleRequest(jsonCtx("GetByteMatchSet", map[string]any{"ByteMatchSetId": bmsID}))
	bms := respBody(t, resp)["ByteMatchSet"].(map[string]any)
	assert.Equal(t, "test-bms", bms["Name"])

	resp, _ = s.HandleRequest(jsonCtx("ListByteMatchSets", map[string]any{}))
	assert.Len(t, respBody(t, resp)["ByteMatchSets"].([]any), 1)

	_, err = s.HandleRequest(jsonCtx("DeleteByteMatchSet", map[string]any{
		"ByteMatchSetId": bmsID, "ChangeToken": "ct",
	}))
	require.NoError(t, err)
}

func TestWAFRegional_AssociateWebACL(t *testing.T) {
	s := newService()
	id := createWebACL(t, s, "assoc-acl")
	resourceArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/123"

	_, err := s.HandleRequest(jsonCtx("AssociateWebACL", map[string]any{
		"WebACLId":    id,
		"ResourceArn": resourceArn,
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetWebACLForResource", map[string]any{
		"ResourceArn": resourceArn,
	}))
	require.NoError(t, err)
	summary := respBody(t, resp)["WebACLSummary"].(map[string]any)
	assert.Equal(t, id, summary["WebACLId"])

	_, err = s.HandleRequest(jsonCtx("DisassociateWebACL", map[string]any{
		"ResourceArn": resourceArn,
	}))
	require.NoError(t, err)
}

func TestWAFRegional_UpdateWebACL_AddRule(t *testing.T) {
	s := newService()
	aclID := createWebACL(t, s, "update-acl")
	resp, _ := s.HandleRequest(jsonCtx("CreateRule", map[string]any{
		"Name": "my-rule", "MetricName": "m", "ChangeToken": "ct",
	}))
	ruleID := respBody(t, resp)["Rule"].(map[string]any)["RuleId"].(string)

	_, err := s.HandleRequest(jsonCtx("UpdateWebACL", map[string]any{
		"WebACLId":    aclID,
		"ChangeToken": "ct",
		"Updates": []any{
			map[string]any{
				"Action": "INSERT",
				"ActivatedRule": map[string]any{
					"RuleId":   ruleID,
					"Priority": float64(1),
					"Action":   map[string]any{"Type": "BLOCK"},
				},
			},
		},
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("GetWebACL", map[string]any{"WebACLId": aclID}))
	rules := respBody(t, resp)["WebACL"].(map[string]any)["Rules"].([]any)
	assert.Len(t, rules, 1)
}

func TestWAFRegional_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}
