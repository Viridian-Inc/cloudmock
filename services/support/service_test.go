package support_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/support"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.SupportService { return svc.New("123456789012", "us-east-1") }

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

func TestSupport_CreateAndDescribeCases(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateCase", map[string]any{
		"subject": "Test issue", "serviceCode": "amazon-ec2", "severityCode": "normal",
		"communicationBody": "My EC2 instance is down",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	caseID := m["caseId"].(string)
	assert.NotEmpty(t, caseID)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeCases", map[string]any{}))
	dm := respJSON(t, descResp)
	cases := dm["cases"].([]any)
	assert.Len(t, cases, 1)
	assert.Equal(t, "opened", cases[0].(map[string]any)["status"])
}

func TestSupport_ResolveCase(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateCase", map[string]any{"subject": "Resolve me", "communicationBody": "Please resolve"}))
	caseID := respJSON(t, cr)["caseId"].(string)

	resp, err := s.HandleRequest(jsonCtx("ResolveCase", map[string]any{"caseId": caseID}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "resolved", m["finalCaseStatus"])

	// Resolved cases hidden by default
	descResp, _ := s.HandleRequest(jsonCtx("DescribeCases", map[string]any{}))
	assert.Len(t, respJSON(t, descResp)["cases"].([]any), 0)

	// Include resolved
	descResp2, _ := s.HandleRequest(jsonCtx("DescribeCases", map[string]any{"includeResolvedCases": true}))
	assert.Len(t, respJSON(t, descResp2)["cases"].([]any), 1)
}

func TestSupport_Communications(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateCase", map[string]any{
		"subject": "Comms test", "communicationBody": "First message",
	}))
	caseID := respJSON(t, cr)["caseId"].(string)

	addResp, err := s.HandleRequest(jsonCtx("AddCommunicationToCase", map[string]any{
		"caseId": caseID, "communicationBody": "Follow-up message",
	}))
	require.NoError(t, err)
	assert.Equal(t, true, respJSON(t, addResp)["result"])

	commsResp, _ := s.HandleRequest(jsonCtx("DescribeCommunications", map[string]any{"caseId": caseID}))
	comms := respJSON(t, commsResp)["communications"].([]any)
	assert.Len(t, comms, 2)
}

func TestSupport_TrustedAdvisorChecks(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("DescribeTrustedAdvisorChecks", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	checks := m["checks"].([]any)
	assert.Greater(t, len(checks), 0)

	checkID := checks[0].(map[string]any)["id"].(string)
	resultResp, err := s.HandleRequest(jsonCtx("DescribeTrustedAdvisorCheckResult", map[string]any{"checkId": checkID}))
	require.NoError(t, err)
	rm := respJSON(t, resultResp)
	assert.NotEmpty(t, rm["result"].(map[string]any)["status"])
}

func TestSupport_RefreshTrustedAdvisorCheck(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("RefreshTrustedAdvisorCheck", map[string]any{"checkId": "Pfx0RwqBli"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "enqueued", m["status"].(map[string]any)["status"])
}

func TestSupport_DescribeServices(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("DescribeServices", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Greater(t, len(m["services"].([]any)), 0)
}

func TestSupport_DescribeSeverityLevels(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("DescribeSeverityLevels", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Len(t, m["severityLevels"].([]any), 5)
}

func TestSupport_CaseNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("ResolveCase", map[string]any{"caseId": "nonexistent"}))
	require.Error(t, err)
}

func TestSupport_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", nil))
	require.Error(t, err)
}

func TestSupport_InvalidSeverityCode(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateCase", map[string]any{
		"subject": "Test", "severityCode": "mega-urgent", "communicationBody": "Help",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid severity code")
}

func TestSupport_MissingCommunicationBody(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateCase", map[string]any{
		"subject": "Test", "severityCode": "normal",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "communicationBody")
}

func TestSupport_TrustedAdvisorSecurityCheck(t *testing.T) {
	s := newService()
	// Security check: Pfx0RwqBli
	resp, err := s.HandleRequest(jsonCtx("DescribeTrustedAdvisorCheckResult", map[string]any{
		"checkId": "Pfx0RwqBli",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	result := m["result"].(map[string]any)
	assert.Equal(t, "error", result["status"])
	flagged := result["flaggedResources"].([]any)
	assert.Greater(t, len(flagged), 0)
	// Security flagged resources should have security-related fields
	first := flagged[0].(map[string]any)
	assert.NotEmpty(t, first["protocol"])
}

func TestSupport_TrustedAdvisorCostCheck(t *testing.T) {
	s := newService()
	// Cost check: DAvU7jPFbW (Low Utilization EC2)
	resp, err := s.HandleRequest(jsonCtx("DescribeTrustedAdvisorCheckResult", map[string]any{
		"checkId": "DAvU7jPFbW",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	result := m["result"].(map[string]any)
	assert.Equal(t, "warning", result["status"])
	flagged := result["flaggedResources"].([]any)
	assert.Equal(t, 3, len(flagged))
	first := flagged[0].(map[string]any)
	assert.NotEmpty(t, first["estimatedMonthlySavings"])
}

func TestSupport_AllFiveSeverityLevels(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("DescribeSeverityLevels", nil))
	m := respJSON(t, resp)
	levels := m["severityLevels"].([]any)
	assert.Len(t, levels, 5)
	codes := make([]string, 0, 5)
	for _, l := range levels {
		codes = append(codes, l.(map[string]any)["code"].(string))
	}
	assert.Contains(t, codes, "low")
	assert.Contains(t, codes, "normal")
	assert.Contains(t, codes, "high")
	assert.Contains(t, codes, "urgent")
	assert.Contains(t, codes, "critical")
}

func TestSupport_CreateCaseMissingSubject(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateCase", map[string]any{
		"communicationBody": "body only, no subject",
	}))
	require.Error(t, err)
}

func TestSupport_InvalidAction2(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("NonExistentAction2", map[string]any{}))
	require.Error(t, err)
}
