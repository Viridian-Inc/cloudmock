package wafv2_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/wafv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.WAFv2Service {
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

func createWebACL(t *testing.T, s *svc.WAFv2Service, name string) map[string]any {
	t.Helper()
	return createWebACLScoped(t, s, name, "REGIONAL")
}

func createWebACLScoped(t *testing.T, s *svc.WAFv2Service, name, scope string) map[string]any {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          name,
		"Scope":         scope,
		"DefaultAction": map[string]any{"Allow": map[string]any{}},
		"VisibilityConfig": map[string]any{
			"SampledRequestsEnabled":   true,
			"CloudWatchMetricsEnabled": true,
			"MetricName":               name,
		},
	}))
	require.NoError(t, err)
	return respBody(t, resp)["Summary"].(map[string]any)
}

// ---- WebACL CRUD ----

func TestWAFv2_CreateWebACL(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "test-acl")
	assert.Equal(t, "test-acl", summary["Name"])
	assert.NotEmpty(t, summary["Id"])
	assert.NotEmpty(t, summary["ARN"])
	assert.NotEmpty(t, summary["LockToken"])
}

func TestWAFv2_CreateWebACL_ARNFormat(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "arn-acl")
	arn := summary["ARN"].(string)
	assert.Contains(t, arn, "arn:aws:wafv2:")
	assert.Contains(t, arn, "webacl/arn-acl/")
}

func TestWAFv2_CreateWebACL_Duplicate(t *testing.T) {
	s := newService()
	createWebACL(t, s, "dup-acl")
	_, err := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name": "dup-acl", "Scope": "REGIONAL", "DefaultAction": map[string]any{"Allow": map[string]any{}},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFDuplicateItemException")
}

func TestWAFv2_CreateWebACL_CloudfrontScope(t *testing.T) {
	s := newService()
	summary := createWebACLScoped(t, s, "cf-acl", "CLOUDFRONT")
	arn := summary["ARN"].(string)
	assert.Contains(t, arn, "global/webacl/")
}

func TestWAFv2_GetWebACL(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "get-acl")
	resp, err := s.HandleRequest(jsonCtx("GetWebACL", map[string]any{
		"Name": "get-acl", "Scope": "REGIONAL", "Id": summary["Id"],
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	acl := m["WebACL"].(map[string]any)
	assert.Equal(t, "get-acl", acl["Name"])
	assert.NotEmpty(t, m["LockToken"])
}

func TestWAFv2_GetWebACL_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetWebACL", map[string]any{
		"Name": "nope", "Scope": "REGIONAL", "Id": "nonexistent",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFNonexistentItemException")
}

func TestWAFv2_ListWebACLs(t *testing.T) {
	s := newService()
	createWebACL(t, s, "acl-1")
	createWebACL(t, s, "acl-2")
	resp, err := s.HandleRequest(jsonCtx("ListWebACLs", map[string]any{"Scope": "REGIONAL"}))
	require.NoError(t, err)
	acls := respBody(t, resp)["WebACLs"].([]any)
	assert.Len(t, acls, 2)
}

func TestWAFv2_ListWebACLs_ScopeIsolation(t *testing.T) {
	s := newService()
	createWebACLScoped(t, s, "reg-acl", "REGIONAL")
	createWebACLScoped(t, s, "cf-acl", "CLOUDFRONT")

	// Regional only
	resp, _ := s.HandleRequest(jsonCtx("ListWebACLs", map[string]any{"Scope": "REGIONAL"}))
	acls := respBody(t, resp)["WebACLs"].([]any)
	assert.Len(t, acls, 1)

	// CloudFront only
	resp, _ = s.HandleRequest(jsonCtx("ListWebACLs", map[string]any{"Scope": "CLOUDFRONT"}))
	acls = respBody(t, resp)["WebACLs"].([]any)
	assert.Len(t, acls, 1)
}

func TestWAFv2_UpdateWebACL(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "update-acl")
	resp, err := s.HandleRequest(jsonCtx("UpdateWebACL", map[string]any{
		"Name": "update-acl", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken":   summary["LockToken"],
		"Description": "updated",
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotEmpty(t, m["NextLockToken"])
}

func TestWAFv2_UpdateWebACL_LockTokenMismatch(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "lock-acl")
	_, err := s.HandleRequest(jsonCtx("UpdateWebACL", map[string]any{
		"Name": "lock-acl", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken": "wrong-token",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFOptimisticLockException")
}

func TestWAFv2_UpdateWebACL_NewLockTokenUsable(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "lock2-acl")

	// First update
	resp, err := s.HandleRequest(jsonCtx("UpdateWebACL", map[string]any{
		"Name": "lock2-acl", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken": summary["LockToken"], "Description": "v2",
	}))
	require.NoError(t, err)
	newToken := respBody(t, resp)["NextLockToken"].(string)

	// Second update with new token
	_, err = s.HandleRequest(jsonCtx("UpdateWebACL", map[string]any{
		"Name": "lock2-acl", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken": newToken, "Description": "v3",
	}))
	require.NoError(t, err)
}

func TestWAFv2_DeleteWebACL(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "del-acl")
	_, err := s.HandleRequest(jsonCtx("DeleteWebACL", map[string]any{
		"Name": "del-acl", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken": summary["LockToken"],
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetWebACL", map[string]any{
		"Name": "del-acl", "Scope": "REGIONAL", "Id": summary["Id"],
	}))
	require.Error(t, err)
}

func TestWAFv2_DeleteWebACL_WithAssociations(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "assoc-del-acl")
	arn := summary["ARN"].(string)

	// Associate the ACL
	s.HandleRequest(jsonCtx("AssociateWebACL", map[string]any{
		"WebACLArn":   arn,
		"ResourceArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/alb/123",
	}))

	// Try to delete - should fail because it's associated
	_, err := s.HandleRequest(jsonCtx("DeleteWebACL", map[string]any{
		"Name": "assoc-del-acl", "Scope": "REGIONAL",
		"Id": summary["Id"], "LockToken": summary["LockToken"],
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFAssociatedItemException")
}

func TestWAFv2_WebACL_WithRules(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          "rules-acl",
		"Scope":         "REGIONAL",
		"DefaultAction": map[string]any{"Allow": map[string]any{}},
		"Rules": []any{
			map[string]any{
				"Name":     "rule1",
				"Priority": float64(1),
				"Action":   map[string]any{"Block": map[string]any{}},
				"Statement": map[string]any{
					"ByteMatchStatement": map[string]any{
						"SearchString":         "malicious",
						"PositionalConstraint": "CONTAINS",
						"FieldToMatch":         map[string]any{"UriPath": map[string]any{}},
					},
				},
				"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true},
			},
		},
		"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true, "MetricName": "rules"},
	}))
	require.NoError(t, err)
	summary := respBody(t, resp)["Summary"].(map[string]any)
	assert.NotEmpty(t, summary["Id"])
}

// ---- Rule Group CRUD ----

func TestWAFv2_RuleGroup_CRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateRuleGroup", map[string]any{
		"Name": "test-rg", "Scope": "REGIONAL", "Capacity": float64(100),
	}))
	require.NoError(t, err)
	summary := respBody(t, resp)["Summary"].(map[string]any)
	rgID := summary["Id"].(string)
	lockToken := summary["LockToken"].(string)

	// Get
	resp, err = s.HandleRequest(jsonCtx("GetRuleGroup", map[string]any{
		"Name": "test-rg", "Scope": "REGIONAL", "Id": rgID,
	}))
	require.NoError(t, err)
	rg := respBody(t, resp)["RuleGroup"].(map[string]any)
	assert.Equal(t, "test-rg", rg["Name"])

	// List
	resp, _ = s.HandleRequest(jsonCtx("ListRuleGroups", map[string]any{"Scope": "REGIONAL"}))
	assert.Len(t, respBody(t, resp)["RuleGroups"].([]any), 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteRuleGroup", map[string]any{
		"Name": "test-rg", "Scope": "REGIONAL", "Id": rgID, "LockToken": lockToken,
	}))
	require.NoError(t, err)
}

func TestWAFv2_RuleGroup_Update(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateRuleGroup", map[string]any{
		"Name": "upd-rg", "Scope": "REGIONAL", "Capacity": float64(50),
	}))
	require.NoError(t, err)
	summary := respBody(t, resp)["Summary"].(map[string]any)

	resp, err = s.HandleRequest(jsonCtx("UpdateRuleGroup", map[string]any{
		"Name": "upd-rg", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken": summary["LockToken"], "Description": "updated rg",
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, respBody(t, resp)["NextLockToken"])
}

func TestWAFv2_RuleGroup_LockTokenEnforced(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateRuleGroup", map[string]any{
		"Name": "lock-rg", "Scope": "REGIONAL", "Capacity": float64(100),
	}))
	summary := respBody(t, resp)["Summary"].(map[string]any)

	_, err := s.HandleRequest(jsonCtx("DeleteRuleGroup", map[string]any{
		"Name": "lock-rg", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken": "wrong-token",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFOptimisticLockException")
}

func TestWAFv2_RuleGroup_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetRuleGroup", map[string]any{
		"Name": "nope", "Scope": "REGIONAL", "Id": "nonexistent",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFNonexistentItemException")
}

// ---- IP Set CRUD ----

func TestWAFv2_IPSet_CRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateIPSet", map[string]any{
		"Name": "test-ipset", "Scope": "REGIONAL",
		"IPAddressVersion": "IPV4", "Addresses": []any{"10.0.0.0/8"},
	}))
	require.NoError(t, err)
	summary := respBody(t, resp)["Summary"].(map[string]any)
	ipID := summary["Id"].(string)

	// Get
	resp, _ = s.HandleRequest(jsonCtx("GetIPSet", map[string]any{
		"Name": "test-ipset", "Scope": "REGIONAL", "Id": ipID,
	}))
	ipSet := respBody(t, resp)["IPSet"].(map[string]any)
	assert.Equal(t, "IPV4", ipSet["IPAddressVersion"])
	addrs := ipSet["Addresses"].([]any)
	assert.Len(t, addrs, 1)

	// List
	resp, _ = s.HandleRequest(jsonCtx("ListIPSets", map[string]any{"Scope": "REGIONAL"}))
	assert.Len(t, respBody(t, resp)["IPSets"].([]any), 1)
}

func TestWAFv2_IPSet_Update(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateIPSet", map[string]any{
		"Name": "upd-ipset", "Scope": "REGIONAL",
		"IPAddressVersion": "IPV4", "Addresses": []any{"10.0.0.0/8"},
	}))
	require.NoError(t, err)
	summary := respBody(t, resp)["Summary"].(map[string]any)

	resp, err = s.HandleRequest(jsonCtx("UpdateIPSet", map[string]any{
		"Name": "upd-ipset", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken": summary["LockToken"],
		"Addresses": []any{"192.168.0.0/16", "172.16.0.0/12"},
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, respBody(t, resp)["NextLockToken"])
}

func TestWAFv2_IPSet_IPv6(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateIPSet", map[string]any{
		"Name": "ipv6-set", "Scope": "REGIONAL",
		"IPAddressVersion": "IPV6",
		"Addresses":        []any{"2001:db8::/32"},
	}))
	require.NoError(t, err)
	summary := respBody(t, resp)["Summary"].(map[string]any)

	resp, _ = s.HandleRequest(jsonCtx("GetIPSet", map[string]any{
		"Name": "ipv6-set", "Scope": "REGIONAL", "Id": summary["Id"],
	}))
	ipSet := respBody(t, resp)["IPSet"].(map[string]any)
	assert.Equal(t, "IPV6", ipSet["IPAddressVersion"])
}

func TestWAFv2_IPSet_Delete(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateIPSet", map[string]any{
		"Name": "del-ipset", "Scope": "REGIONAL", "IPAddressVersion": "IPV4", "Addresses": []any{},
	}))
	summary := respBody(t, resp)["Summary"].(map[string]any)

	_, err := s.HandleRequest(jsonCtx("DeleteIPSet", map[string]any{
		"Name": "del-ipset", "Scope": "REGIONAL",
		"Id": summary["Id"], "LockToken": summary["LockToken"],
	}))
	require.NoError(t, err)
}

func TestWAFv2_IPSet_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetIPSet", map[string]any{
		"Name": "nope", "Scope": "REGIONAL", "Id": "nonexistent",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFNonexistentItemException")
}

// ---- Regex Pattern Set CRUD ----

func TestWAFv2_RegexPatternSet_CRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateRegexPatternSet", map[string]any{
		"Name": "test-rps", "Scope": "REGIONAL",
		"RegularExpressionList": []any{
			map[string]any{"RegexString": "^/admin"},
		},
	}))
	require.NoError(t, err)
	summary := respBody(t, resp)["Summary"].(map[string]any)
	rpsID := summary["Id"].(string)

	// Get
	resp, _ = s.HandleRequest(jsonCtx("GetRegexPatternSet", map[string]any{
		"Name": "test-rps", "Scope": "REGIONAL", "Id": rpsID,
	}))
	rps := respBody(t, resp)["RegexPatternSet"].(map[string]any)
	assert.Equal(t, "test-rps", rps["Name"])
	regexList := rps["RegularExpressionList"].([]any)
	assert.Len(t, regexList, 1)

	// List
	resp, _ = s.HandleRequest(jsonCtx("ListRegexPatternSets", map[string]any{"Scope": "REGIONAL"}))
	assert.Len(t, respBody(t, resp)["RegexPatternSets"].([]any), 1)
}

func TestWAFv2_RegexPatternSet_Update(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateRegexPatternSet", map[string]any{
		"Name": "upd-rps", "Scope": "REGIONAL",
		"RegularExpressionList": []any{
			map[string]any{"RegexString": "^/old"},
		},
	}))
	require.NoError(t, err)
	summary := respBody(t, resp)["Summary"].(map[string]any)

	resp, err = s.HandleRequest(jsonCtx("UpdateRegexPatternSet", map[string]any{
		"Name": "upd-rps", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken": summary["LockToken"],
		"RegularExpressionList": []any{
			map[string]any{"RegexString": "^/new1"},
			map[string]any{"RegexString": "^/new2"},
		},
	}))
	require.NoError(t, err)
	newToken := respBody(t, resp)["NextLockToken"].(string)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, summary["LockToken"], newToken)
}

func TestWAFv2_RegexPatternSet_Update_LockTokenMismatch(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateRegexPatternSet", map[string]any{
		"Name": "lock-rps", "Scope": "REGIONAL", "RegularExpressionList": []any{},
	}))
	summary := respBody(t, resp)["Summary"].(map[string]any)

	_, err := s.HandleRequest(jsonCtx("UpdateRegexPatternSet", map[string]any{
		"Name": "lock-rps", "Scope": "REGIONAL", "Id": summary["Id"],
		"LockToken": "wrong", "RegularExpressionList": []any{},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFOptimisticLockException")
}

func TestWAFv2_RegexPatternSet_Delete(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateRegexPatternSet", map[string]any{
		"Name": "del-rps", "Scope": "REGIONAL", "RegularExpressionList": []any{},
	}))
	summary := respBody(t, resp)["Summary"].(map[string]any)

	_, err := s.HandleRequest(jsonCtx("DeleteRegexPatternSet", map[string]any{
		"Name": "del-rps", "Scope": "REGIONAL",
		"Id": summary["Id"], "LockToken": summary["LockToken"],
	}))
	require.NoError(t, err)
}

func TestWAFv2_RegexPatternSet_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetRegexPatternSet", map[string]any{
		"Name": "nope", "Scope": "REGIONAL", "Id": "nonexistent",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFNonexistentItemException")
}

// ---- Web ACL resource associations ----

func TestWAFv2_AssociateWebACL(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "assoc-acl")
	arn := summary["ARN"].(string)
	resourceArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/123"

	_, err := s.HandleRequest(jsonCtx("AssociateWebACL", map[string]any{
		"WebACLArn":   arn,
		"ResourceArn": resourceArn,
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetWebACLForResource", map[string]any{
		"ResourceArn": resourceArn,
	}))
	require.NoError(t, err)
	acl := respBody(t, resp)["WebACL"].(map[string]any)
	assert.Equal(t, "assoc-acl", acl["Name"])

	_, err = s.HandleRequest(jsonCtx("DisassociateWebACL", map[string]any{
		"ResourceArn": resourceArn,
	}))
	require.NoError(t, err)
}

func TestWAFv2_GetWebACLForResource_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetWebACLForResource", map[string]any{
		"ResourceArn": "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/nope/999",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFNonexistentItemException")
}

func TestWAFv2_DisassociateWebACL_Idempotent(t *testing.T) {
	s := newService()
	// Disassociating a resource that's not associated should not error
	_, err := s.HandleRequest(jsonCtx("DisassociateWebACL", map[string]any{
		"ResourceArn": "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/notassoc/999",
	}))
	require.NoError(t, err)
}

// ---- Tagging ----

func TestWAFv2_Tagging(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "tag-acl")
	arn := summary["ARN"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": arn,
		"Tags":        []any{map[string]any{"Key": "env", "Value": "prod"}},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{
		"ResourceARN": arn,
	}))
	info := respBody(t, resp)["TagInfoForResource"].(map[string]any)
	tags := info["TagList"].([]any)
	assert.Len(t, tags, 1)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceARN": arn,
		"TagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": arn}))
	info = respBody(t, resp)["TagInfoForResource"].(map[string]any)
	tags = info["TagList"].([]any)
	assert.Len(t, tags, 0)
}

func TestWAFv2_Tagging_IPSet(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateIPSet", map[string]any{
		"Name": "tag-ipset", "Scope": "REGIONAL", "IPAddressVersion": "IPV4", "Addresses": []any{},
	}))
	arn := respBody(t, resp)["Summary"].(map[string]any)["ARN"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": arn,
		"Tags":        []any{map[string]any{"Key": "team", "Value": "security"}},
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": arn}))
	info := respBody(t, resp)["TagInfoForResource"].(map[string]any)
	tags := info["TagList"].([]any)
	assert.Len(t, tags, 1)
	tag := tags[0].(map[string]any)
	assert.Equal(t, "team", tag["Key"])
	assert.Equal(t, "security", tag["Value"])
}

func TestWAFv2_Tagging_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": "arn:aws:wafv2:us-east-1:123:regional/webacl/nonexistent/id",
		"Tags":        []any{map[string]any{"Key": "k", "Value": "v"}},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFNonexistentItemException")
}

// ---- Logging ----

func TestWAFv2_LoggingConfig(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "log-acl")
	arn := summary["ARN"].(string)

	_, err := s.HandleRequest(jsonCtx("PutLoggingConfiguration", map[string]any{
		"LoggingConfiguration": map[string]any{
			"ResourceArn":           arn,
			"LogDestinationConfigs": []any{"arn:aws:firehose:us-east-1:123456789012:deliverystream/aws-waf-logs"},
		},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("GetLoggingConfiguration", map[string]any{
		"ResourceArn": arn,
	}))
	m := respBody(t, resp)
	assert.NotNil(t, m["LoggingConfiguration"])

	_, err = s.HandleRequest(jsonCtx("DeleteLoggingConfiguration", map[string]any{
		"ResourceArn": arn,
	}))
	require.NoError(t, err)
}

// ---- Request evaluation (behavioral) ----

func TestWAFv2_CheckRequest_IPSetBlock(t *testing.T) {
	s := newService()

	// Create an IP set with a blocked range
	resp, _ := s.HandleRequest(jsonCtx("CreateIPSet", map[string]any{
		"Name": "blocked-ips", "Scope": "REGIONAL",
		"IPAddressVersion": "IPV4", "Addresses": []any{"10.0.0.0/8"},
	}))
	ipSetSummary := respBody(t, resp)["Summary"].(map[string]any)
	ipSetARN := ipSetSummary["ARN"].(string)

	// Create a Web ACL with a rule that blocks IPs in the set
	resp, err := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          "ip-block-acl",
		"Scope":         "REGIONAL",
		"DefaultAction": map[string]any{"Allow": map[string]any{}},
		"Rules": []any{
			map[string]any{
				"Name":     "block-bad-ips",
				"Priority": float64(1),
				"Action":   map[string]any{"Block": map[string]any{}},
				"Statement": map[string]any{
					"IPSetReferenceStatement": map[string]any{"ARN": ipSetARN},
				},
				"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true},
			},
		},
		"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true, "MetricName": "test"},
	}))
	require.NoError(t, err)
	aclSummary := respBody(t, resp)["Summary"].(map[string]any)
	aclID := aclSummary["Id"].(string)

	// Check a blocked IP
	resp, err = s.HandleRequest(jsonCtx("CheckRequest", map[string]any{
		"WebACLId": aclID,
		"IP":       "10.1.2.3",
		"URI":      "/test",
	}))
	require.NoError(t, err)
	result := respBody(t, resp)
	assert.Equal(t, "BLOCK", result["Action"])
	assert.Equal(t, "block-bad-ips", result["RuleName"])

	// Check an allowed IP
	resp, err = s.HandleRequest(jsonCtx("CheckRequest", map[string]any{
		"WebACLId": aclID,
		"IP":       "192.168.1.1",
		"URI":      "/test",
	}))
	require.NoError(t, err)
	result = respBody(t, resp)
	assert.Equal(t, "ALLOW", result["Action"])
	assert.Equal(t, "Default", result["RuleName"])
}

func TestWAFv2_CheckRequest_RegexPatternBlock(t *testing.T) {
	s := newService()

	// Create regex pattern set
	resp, _ := s.HandleRequest(jsonCtx("CreateRegexPatternSet", map[string]any{
		"Name": "sql-patterns", "Scope": "REGIONAL",
		"RegularExpressionList": []any{
			map[string]any{"RegexString": "(?i)(union|select|insert|delete|drop).*"},
		},
	}))
	regexSummary := respBody(t, resp)["Summary"].(map[string]any)
	regexARN := regexSummary["ARN"].(string)

	// Create Web ACL with regex rule
	resp, _ = s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          "regex-acl",
		"Scope":         "REGIONAL",
		"DefaultAction": map[string]any{"Allow": map[string]any{}},
		"Rules": []any{
			map[string]any{
				"Name":     "block-sqli",
				"Priority": float64(1),
				"Action":   map[string]any{"Block": map[string]any{}},
				"Statement": map[string]any{
					"RegexPatternSetReferenceStatement": map[string]any{
						"ARN":          regexARN,
						"FieldToMatch": map[string]any{"UriPath": map[string]any{}},
					},
				},
				"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true},
			},
		},
		"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true, "MetricName": "regex"},
	}))
	aclID := respBody(t, resp)["Summary"].(map[string]any)["Id"].(string)

	// Check SQLi-like URI
	resp, _ = s.HandleRequest(jsonCtx("CheckRequest", map[string]any{
		"WebACLId": aclID,
		"IP":       "1.2.3.4",
		"URI":      "/api?q=SELECT * FROM users",
	}))
	result := respBody(t, resp)
	assert.Equal(t, "BLOCK", result["Action"])

	// Check normal URI
	resp, _ = s.HandleRequest(jsonCtx("CheckRequest", map[string]any{
		"WebACLId": aclID,
		"IP":       "1.2.3.4",
		"URI":      "/api/users",
	}))
	result = respBody(t, resp)
	assert.Equal(t, "ALLOW", result["Action"])
}

func TestWAFv2_CheckRequest_ByteMatch(t *testing.T) {
	s := newService()

	resp, _ := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          "bytematch-acl",
		"Scope":         "REGIONAL",
		"DefaultAction": map[string]any{"Allow": map[string]any{}},
		"Rules": []any{
			map[string]any{
				"Name":     "block-admin",
				"Priority": float64(1),
				"Action":   map[string]any{"Block": map[string]any{}},
				"Statement": map[string]any{
					"ByteMatchStatement": map[string]any{
						"SearchString":         "/admin",
						"PositionalConstraint": "STARTS_WITH",
						"FieldToMatch":         map[string]any{"UriPath": map[string]any{}},
					},
				},
				"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true},
			},
		},
		"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true, "MetricName": "bytematch"},
	}))
	aclID := respBody(t, resp)["Summary"].(map[string]any)["Id"].(string)

	// Should block /admin path
	resp, _ = s.HandleRequest(jsonCtx("CheckRequest", map[string]any{
		"WebACLId": aclID, "IP": "1.2.3.4", "URI": "/admin/settings",
	}))
	assert.Equal(t, "BLOCK", respBody(t, resp)["Action"])

	// Should allow /api path
	resp, _ = s.HandleRequest(jsonCtx("CheckRequest", map[string]any{
		"WebACLId": aclID, "IP": "1.2.3.4", "URI": "/api/data",
	}))
	assert.Equal(t, "ALLOW", respBody(t, resp)["Action"])
}

func TestWAFv2_CheckRequest_DefaultBlock(t *testing.T) {
	s := newService()

	resp, _ := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          "default-block-acl",
		"Scope":         "REGIONAL",
		"DefaultAction": map[string]any{"Block": map[string]any{}},
		"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true, "MetricName": "block"},
	}))
	aclID := respBody(t, resp)["Summary"].(map[string]any)["Id"].(string)

	// With no rules, default action should block
	resp, _ = s.HandleRequest(jsonCtx("CheckRequest", map[string]any{
		"WebACLId": aclID,
		"IP":       "1.2.3.4",
		"URI":      "/anything",
	}))
	result := respBody(t, resp)
	assert.Equal(t, "BLOCK", result["Action"])
}

func TestWAFv2_CheckRequest_UnknownACL(t *testing.T) {
	s := newService()
	// Unknown ACL should default to ALLOW
	resp, err := s.HandleRequest(jsonCtx("CheckRequest", map[string]any{
		"WebACLId": "unknown-acl-id",
		"IP":       "1.2.3.4",
		"URI":      "/test",
	}))
	require.NoError(t, err)
	assert.Equal(t, "ALLOW", respBody(t, resp)["Action"])
}

// ---- Sampled requests ----

func TestWAFv2_GetSampledRequests(t *testing.T) {
	s := newService()

	resp, _ := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          "sample-acl",
		"Scope":         "REGIONAL",
		"DefaultAction": map[string]any{"Allow": map[string]any{}},
		"VisibilityConfig": map[string]any{"SampledRequestsEnabled": true, "MetricName": "sample"},
	}))
	aclSummary := respBody(t, resp)["Summary"].(map[string]any)
	aclID := aclSummary["Id"].(string)
	aclARN := aclSummary["ARN"].(string)

	// Send some requests through CheckRequest
	for i := 0; i < 3; i++ {
		s.HandleRequest(jsonCtx("CheckRequest", map[string]any{
			"WebACLId": aclID,
			"IP":       "1.2.3.4",
			"URI":      "/test",
		}))
	}

	// Get sampled requests
	resp, err := s.HandleRequest(jsonCtx("GetSampledRequests", map[string]any{
		"WebAclArn": aclARN,
		"MaxItems":  float64(10),
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	samples := m["SampledRequests"].([]any)
	assert.Len(t, samples, 3)
}

// ---- Misc ----

func TestWAFv2_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

func TestWAFv2_InvalidJSON(t *testing.T) {
	s := newService()
	ctx := &service.RequestContext{
		Action:     "CreateWebACL",
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       []byte("{not valid json"),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	}
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WAFInvalidParameterException")
}

func TestWAFv2_WebACL_CapacityTracked(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "capacity-acl")
	resp, err := s.HandleRequest(jsonCtx("GetWebACL", map[string]any{
		"Name": "capacity-acl", "Scope": "REGIONAL", "Id": summary["Id"],
	}))
	require.NoError(t, err)
	acl := respBody(t, resp)["WebACL"].(map[string]any)
	capacity, ok := acl["Capacity"].(float64)
	assert.True(t, ok, "Capacity should be present")
	assert.True(t, capacity >= 0, "Capacity should be non-negative")
}

func TestWAFv2_RuleGroup_ScopeIsolation(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateRuleGroup", map[string]any{
		"Name": "reg-rg", "Scope": "REGIONAL", "Capacity": float64(100),
	}))
	s.HandleRequest(jsonCtx("CreateRuleGroup", map[string]any{
		"Name": "cf-rg", "Scope": "CLOUDFRONT", "Capacity": float64(100),
	}))

	resp, _ := s.HandleRequest(jsonCtx("ListRuleGroups", map[string]any{"Scope": "REGIONAL"}))
	assert.Len(t, respBody(t, resp)["RuleGroups"].([]any), 1)

	resp, _ = s.HandleRequest(jsonCtx("ListRuleGroups", map[string]any{"Scope": "CLOUDFRONT"}))
	assert.Len(t, respBody(t, resp)["RuleGroups"].([]any), 1)
}
