package wafv2_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/wafv2"
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
	resp, err := s.HandleRequest(jsonCtx("CreateWebACL", map[string]any{
		"Name":          name,
		"Scope":         "REGIONAL",
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

func TestWAFv2_CreateWebACL(t *testing.T) {
	s := newService()
	summary := createWebACL(t, s, "test-acl")
	assert.Equal(t, "test-acl", summary["Name"])
	assert.NotEmpty(t, summary["Id"])
	assert.NotEmpty(t, summary["ARN"])
	assert.NotEmpty(t, summary["LockToken"])
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

func TestWAFv2_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

// --- Behavioral Tests ---

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
