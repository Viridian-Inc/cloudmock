package account_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.AccountService { return svc.New("123456789012", "us-east-1") }
func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{Action: action, Region: "us-east-1", AccountID: "123456789012", Body: bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

func TestAccount_GetContactInformation(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetContactInformation", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	ci := m["ContactInformation"].(map[string]any)
	assert.Equal(t, "CloudMock Admin", ci["FullName"])
}

func TestAccount_PutContactInformation(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutContactInformation", map[string]any{
		"ContactInformation": map[string]any{"FullName": "New Admin", "City": "Portland"},
	}))

	resp, _ := s.HandleRequest(jsonCtx("GetContactInformation", nil))
	ci := respJSON(t, resp)["ContactInformation"].(map[string]any)
	assert.Equal(t, "New Admin", ci["FullName"])
	assert.Equal(t, "Portland", ci["City"])
}

func TestAccount_AlternateContactCRUD(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("PutAlternateContact", map[string]any{
		"AlternateContactType": "BILLING", "Name": "Finance Team",
		"EmailAddress": "billing@example.com", "PhoneNumber": "+1-555-0101",
	}))

	resp, err := s.HandleRequest(jsonCtx("GetAlternateContact", map[string]any{"AlternateContactType": "BILLING"}))
	require.NoError(t, err)
	ac := respJSON(t, resp)["AlternateContact"].(map[string]any)
	assert.Equal(t, "Finance Team", ac["Name"])

	delResp, err := s.HandleRequest(jsonCtx("DeleteAlternateContact", map[string]any{"AlternateContactType": "BILLING"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)

	_, err = s.HandleRequest(jsonCtx("GetAlternateContact", map[string]any{"AlternateContactType": "BILLING"}))
	require.Error(t, err)
}

func TestAccount_ListRegions(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("ListRegions", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	regions := m["Regions"].([]any)
	assert.Greater(t, len(regions), 10)
}

func TestAccount_GetRegionOptStatus(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetRegionOptStatus", map[string]any{"RegionName": "us-east-1"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, "ENABLED", m["RegionOptStatus"])
}

func TestAccount_EnableDisableRegion(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("EnableRegion", map[string]any{"RegionName": "af-south-1"}))
	resp, _ := s.HandleRequest(jsonCtx("GetRegionOptStatus", map[string]any{"RegionName": "af-south-1"}))
	assert.Equal(t, "ENABLED", respJSON(t, resp)["RegionOptStatus"])

	s.HandleRequest(jsonCtx("DisableRegion", map[string]any{"RegionName": "af-south-1"}))
	resp2, _ := s.HandleRequest(jsonCtx("GetRegionOptStatus", map[string]any{"RegionName": "af-south-1"}))
	assert.Equal(t, "DISABLED", respJSON(t, resp2)["RegionOptStatus"])
}

func TestAccount_AlternateContactNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetAlternateContact", map[string]any{"AlternateContactType": "SECURITY"}))
	require.Error(t, err)
}

func TestAccount_DeleteNonexistentContact(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteAlternateContact", map[string]any{"AlternateContactType": "OPERATIONS"}))
	require.Error(t, err)
}

func TestAccount_InvalidEmailFormat(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("PutAlternateContact", map[string]any{
		"AlternateContactType": "BILLING",
		"Name":                 "Finance Team",
		"EmailAddress":         "not-an-email",
		"PhoneNumber":          "+15550100",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestAccount_InvalidPhoneFormat(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("PutAlternateContact", map[string]any{
		"AlternateContactType": "SECURITY",
		"Name":                 "Security Team",
		"EmailAddress":         "security@example.com",
		"PhoneNumber":          "12345",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PhoneNumber")
}

func TestAccount_InvalidContactType(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("PutAlternateContact", map[string]any{
		"AlternateContactType": "INVALID",
		"Name":                 "Team",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BILLING, OPERATIONS, SECURITY")
}

func TestAccount_InvalidRegionName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("EnableRegion", map[string]any{"RegionName": "us-fake-99"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid AWS region")
}

func TestAccount_ListRegionsHasDescriptions(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("ListRegions", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	regions := m["Regions"].([]any)
	// Find us-east-1 and check its description
	found := false
	for _, r := range regions {
		rm := r.(map[string]any)
		if rm["RegionName"] == "us-east-1" {
			assert.Equal(t, "US East (N. Virginia)", rm["RegionDescription"])
			found = true
			break
		}
	}
	assert.True(t, found, "us-east-1 should be in region list")
}

func TestAccount_PutContactInfoValidation(t *testing.T) {
	s := newService()
	// Missing FullName
	_, err := s.HandleRequest(jsonCtx("PutContactInformation", map[string]any{
		"ContactInformation": map[string]any{"City": "Portland"},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FullName")

	// Invalid phone
	_, err = s.HandleRequest(jsonCtx("PutContactInformation", map[string]any{
		"ContactInformation": map[string]any{"FullName": "Admin", "PhoneNumber": "abc"},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PhoneNumber")
}

func TestAccount_GetAlternateContactNotFound(t *testing.T) {
	s := newService()
	// Before any contact is set, GetAlternateContact should return not found
	_, err := s.HandleRequest(jsonCtx("GetAlternateContact", map[string]any{
		"AlternateContactType": "BILLING",
	}))
	require.Error(t, err)
}

func TestAccount_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("NonExistentAction", map[string]any{}))
	require.Error(t, err)
}
