package route53resolver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/route53resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.Route53ResolverService { return svc.New("123456789012", "us-east-1") }
func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{Action: action, Region: "us-east-1", AccountID: "123456789012", Body: bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

func TestR53R_CreateAndGetEndpoint(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateResolverEndpoint", map[string]any{
		"Name": "my-ep", "Direction": "INBOUND", "SecurityGroupIds": []string{"sg-123"},
		"IpAddresses": []map[string]any{{"SubnetId": "subnet-1"}, {"SubnetId": "subnet-2"}},
	}))
	require.NoError(t, err)
	epID := respJSON(t, resp)["ResolverEndpoint"].(map[string]any)["Id"].(string)

	getResp, _ := s.HandleRequest(jsonCtx("GetResolverEndpoint", map[string]any{"ResolverEndpointId": epID}))
	ep := respJSON(t, getResp)["ResolverEndpoint"].(map[string]any)
	assert.Equal(t, "CREATING", ep["Status"])
	assert.Equal(t, "INBOUND", ep["Direction"])
}

func TestR53R_ListEndpoints(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateResolverEndpoint", map[string]any{"Direction": "INBOUND"}))
	s.HandleRequest(jsonCtx("CreateResolverEndpoint", map[string]any{"Direction": "OUTBOUND"}))

	resp, _ := s.HandleRequest(jsonCtx("ListResolverEndpoints", nil))
	assert.Len(t, respJSON(t, resp)["ResolverEndpoints"].([]any), 2)
}

func TestR53R_DeleteEndpoint(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateResolverEndpoint", map[string]any{"Direction": "INBOUND"}))
	epID := respJSON(t, cr)["ResolverEndpoint"].(map[string]any)["Id"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteResolverEndpoint", map[string]any{"ResolverEndpointId": epID}))
	require.NoError(t, err)
	assert.Equal(t, "DELETING", respJSON(t, resp)["ResolverEndpoint"].(map[string]any)["Status"])
}

func TestR53R_RuleCRUD(t *testing.T) {
	s := newService()
	ruleResp, err := s.HandleRequest(jsonCtx("CreateResolverRule", map[string]any{
		"Name": "fwd-rule", "DomainName": "example.com.", "RuleType": "FORWARD",
		"TargetIps": []map[string]any{{"Ip": "10.0.0.1", "Port": 53}},
	}))
	require.NoError(t, err)
	ruleID := respJSON(t, ruleResp)["ResolverRule"].(map[string]any)["Id"].(string)

	getResp, _ := s.HandleRequest(jsonCtx("GetResolverRule", map[string]any{"ResolverRuleId": ruleID}))
	assert.Equal(t, "example.com.", respJSON(t, getResp)["ResolverRule"].(map[string]any)["DomainName"])

	listResp, _ := s.HandleRequest(jsonCtx("ListResolverRules", nil))
	assert.Len(t, respJSON(t, listResp)["ResolverRules"].([]any), 1)

	delResp, _ := s.HandleRequest(jsonCtx("DeleteResolverRule", map[string]any{"ResolverRuleId": ruleID}))
	assert.Equal(t, http.StatusOK, delResp.StatusCode)
}

func TestR53R_AssociateAndDisassociateRule(t *testing.T) {
	s := newService()
	rr, _ := s.HandleRequest(jsonCtx("CreateResolverRule", map[string]any{
		"DomainName": "test.com.", "RuleType": "FORWARD",
	}))
	ruleID := respJSON(t, rr)["ResolverRule"].(map[string]any)["Id"].(string)

	assocResp, err := s.HandleRequest(jsonCtx("AssociateResolverRule", map[string]any{
		"ResolverRuleId": ruleID, "VPCId": "vpc-123", "Name": "my-assoc",
	}))
	require.NoError(t, err)
	assocID := respJSON(t, assocResp)["ResolverRuleAssociation"].(map[string]any)["Id"].(string)

	getResp, _ := s.HandleRequest(jsonCtx("GetResolverRuleAssociation", map[string]any{"ResolverRuleAssociationId": assocID}))
	assert.Equal(t, "COMPLETE", respJSON(t, getResp)["ResolverRuleAssociation"].(map[string]any)["Status"])

	listResp, _ := s.HandleRequest(jsonCtx("ListResolverRuleAssociations", nil))
	assert.Len(t, respJSON(t, listResp)["ResolverRuleAssociations"].([]any), 1)

	disResp, _ := s.HandleRequest(jsonCtx("DisassociateResolverRule", map[string]any{"ResolverRuleAssociationId": assocID}))
	assert.Equal(t, "DELETING", respJSON(t, disResp)["ResolverRuleAssociation"].(map[string]any)["Status"])
}

func TestR53R_QueryLogConfigCRUD(t *testing.T) {
	s := newService()
	cr, err := s.HandleRequest(jsonCtx("CreateResolverQueryLogConfig", map[string]any{
		"Name": "my-log", "DestinationArn": "arn:aws:s3:::my-bucket",
	}))
	require.NoError(t, err)
	configID := respJSON(t, cr)["ResolverQueryLogConfig"].(map[string]any)["Id"].(string)

	getResp, _ := s.HandleRequest(jsonCtx("GetResolverQueryLogConfig", map[string]any{"ResolverQueryLogConfigId": configID}))
	assert.Equal(t, "CREATED", respJSON(t, getResp)["ResolverQueryLogConfig"].(map[string]any)["Status"])

	listResp, _ := s.HandleRequest(jsonCtx("ListResolverQueryLogConfigs", nil))
	assert.Len(t, respJSON(t, listResp)["ResolverQueryLogConfigs"].([]any), 1)

	delResp, _ := s.HandleRequest(jsonCtx("DeleteResolverQueryLogConfig", map[string]any{"ResolverQueryLogConfigId": configID}))
	assert.Equal(t, "DELETING", respJSON(t, delResp)["ResolverQueryLogConfig"].(map[string]any)["Status"])
}

func TestR53R_InvalidTargetIP(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateResolverRule", map[string]any{
		"Name": "bad-ip-rule", "DomainName": "example.com.", "RuleType": "FORWARD",
		"TargetIps": []map[string]any{{"Ip": "not-an-ip", "Port": 53}},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target IP")
}

func TestR53R_ValidIPv4TargetIP(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateResolverRule", map[string]any{
		"Name": "good-ip-rule", "DomainName": "example.com.", "RuleType": "FORWARD",
		"TargetIps": []map[string]any{{"Ip": "192.168.1.1", "Port": 53}},
	}))
	require.NoError(t, err)
	rule := respJSON(t, resp)["ResolverRule"].(map[string]any)
	assert.Equal(t, "COMPLETE", rule["Status"])
}

func TestR53R_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetResolverEndpoint", map[string]any{"ResolverEndpointId": "nonexistent"}))
	require.Error(t, err)
}

func TestR53R_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("Bogus", nil))
	require.Error(t, err)
}
