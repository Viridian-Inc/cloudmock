package verifiedpermissions_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/verifiedpermissions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.VerifiedPermissionsService {
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

func createPolicyStore(t *testing.T, s *svc.VerifiedPermissionsService) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreatePolicyStore", map[string]any{
		"description": "Test store",
		"validationSettings": map[string]any{"mode": "STRICT"},
	}))
	require.NoError(t, err)
	return respBody(t, resp)["policyStoreId"].(string)
}

func TestVP_CreatePolicyStore(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreatePolicyStore", map[string]any{
		"description": "My store",
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotEmpty(t, m["policyStoreId"])
	assert.NotEmpty(t, m["arn"])
	assert.NotEmpty(t, m["createdDate"])
}

func TestVP_GetPolicyStore(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("GetPolicyStore", map[string]any{
		"policyStoreId": storeID,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, storeID, m["policyStoreId"])
	assert.Equal(t, "Test store", m["description"])
}

func TestVP_GetPolicyStore_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetPolicyStore", map[string]any{
		"policyStoreId": "nonexistent",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestVP_ListPolicyStores(t *testing.T) {
	s := newService()
	createPolicyStore(t, s)
	createPolicyStore(t, s)
	resp, _ := s.HandleRequest(jsonCtx("ListPolicyStores", map[string]any{}))
	stores := respBody(t, resp)["policyStores"].([]any)
	assert.Len(t, stores, 2)
}

func TestVP_UpdatePolicyStore(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("UpdatePolicyStore", map[string]any{
		"policyStoreId": storeID,
		"description":   "Updated store",
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, storeID, m["policyStoreId"])
}

func TestVP_DeletePolicyStore(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	_, err := s.HandleRequest(jsonCtx("DeletePolicyStore", map[string]any{
		"policyStoreId": storeID,
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetPolicyStore", map[string]any{
		"policyStoreId": storeID,
	}))
	require.Error(t, err)
}

func TestVP_CreatePolicy(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyStoreId": storeID,
		"definition": map[string]any{
			"static": map[string]any{
				"statement": `permit(principal, action, resource);`,
			},
		},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotEmpty(t, m["policyId"])
	assert.Equal(t, "STATIC", m["policyType"])
}

func TestVP_GetPolicy(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, _ := s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyStoreId": storeID,
		"definition":    map[string]any{"static": map[string]any{"statement": "permit;"}},
	}))
	policyID := respBody(t, resp)["policyId"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetPolicy", map[string]any{
		"policyStoreId": storeID,
		"policyId":      policyID,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, policyID, m["policyId"])
	assert.NotNil(t, m["definition"])
}

func TestVP_ListPolicies(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyStoreId": storeID, "definition": map[string]any{"static": map[string]any{}},
	}))
	s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyStoreId": storeID, "definition": map[string]any{"static": map[string]any{}},
	}))
	resp, _ := s.HandleRequest(jsonCtx("ListPolicies", map[string]any{
		"policyStoreId": storeID,
	}))
	policies := respBody(t, resp)["policies"].([]any)
	assert.Len(t, policies, 2)
}

func TestVP_UpdatePolicy(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, _ := s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyStoreId": storeID, "definition": map[string]any{"static": map[string]any{}},
	}))
	policyID := respBody(t, resp)["policyId"].(string)

	resp, err := s.HandleRequest(jsonCtx("UpdatePolicy", map[string]any{
		"policyStoreId": storeID,
		"policyId":      policyID,
		"definition":    map[string]any{"static": map[string]any{"statement": "forbid;"}},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, policyID, m["policyId"])
}

func TestVP_DeletePolicy(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, _ := s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyStoreId": storeID, "definition": map[string]any{"static": map[string]any{}},
	}))
	policyID := respBody(t, resp)["policyId"].(string)

	_, err := s.HandleRequest(jsonCtx("DeletePolicy", map[string]any{
		"policyStoreId": storeID, "policyId": policyID,
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetPolicy", map[string]any{
		"policyStoreId": storeID, "policyId": policyID,
	}))
	require.Error(t, err)
}

func TestVP_Schema(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("PutSchema", map[string]any{
		"policyStoreId": storeID,
		"definition":    map[string]any{"cedarJson": `{"namespace":{}}`},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotNil(t, m["namespaces"])

	resp, err = s.HandleRequest(jsonCtx("GetSchema", map[string]any{
		"policyStoreId": storeID,
	}))
	require.NoError(t, err)
	m = respBody(t, resp)
	assert.Equal(t, `{"namespace":{}}`, m["schema"])
}

func TestVP_IsAuthorized(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("IsAuthorized", map[string]any{
		"policyStoreId": storeID,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, "ALLOW", m["decision"])
}

func TestVP_IsAuthorizedWithToken(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("IsAuthorizedWithToken", map[string]any{
		"policyStoreId": storeID,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, "ALLOW", m["decision"])
}

func TestVP_PolicyTemplate_CRUD(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("CreatePolicyTemplate", map[string]any{
		"policyStoreId": storeID,
		"description":   "Template",
		"statement":     "permit(principal == ?principal, action, resource);",
	}))
	require.NoError(t, err)
	templateID := respBody(t, resp)["policyTemplateId"].(string)

	resp, _ = s.HandleRequest(jsonCtx("GetPolicyTemplate", map[string]any{
		"policyStoreId": storeID, "policyTemplateId": templateID,
	}))
	m := respBody(t, resp)
	assert.Equal(t, "Template", m["description"])

	resp, _ = s.HandleRequest(jsonCtx("ListPolicyTemplates", map[string]any{
		"policyStoreId": storeID,
	}))
	templates := respBody(t, resp)["policyTemplates"].([]any)
	assert.Len(t, templates, 1)

	_, err = s.HandleRequest(jsonCtx("DeletePolicyTemplate", map[string]any{
		"policyStoreId": storeID, "policyTemplateId": templateID,
	}))
	require.NoError(t, err)
}

func TestVP_IdentitySource_CRUD(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("CreateIdentitySource", map[string]any{
		"policyStoreId":       storeID,
		"principalEntityType": "User",
		"configuration": map[string]any{
			"cognitoUserPoolConfiguration": map[string]any{
				"userPoolArn": "arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_abc",
			},
		},
	}))
	require.NoError(t, err)
	isID := respBody(t, resp)["identitySourceId"].(string)

	resp, _ = s.HandleRequest(jsonCtx("GetIdentitySource", map[string]any{
		"policyStoreId": storeID, "identitySourceId": isID,
	}))
	m := respBody(t, resp)
	assert.Equal(t, "User", m["principalEntityType"])

	resp, _ = s.HandleRequest(jsonCtx("ListIdentitySources", map[string]any{
		"policyStoreId": storeID,
	}))
	sources := respBody(t, resp)["identitySources"].([]any)
	assert.Len(t, sources, 1)

	_, err = s.HandleRequest(jsonCtx("DeleteIdentitySource", map[string]any{
		"policyStoreId": storeID, "identitySourceId": isID,
	}))
	require.NoError(t, err)
}

func TestVP_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

func TestVP_CedarPolicyValidation_InvalidStatement(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	_, err := s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyStoreId": storeID,
		"definition": map[string]any{
			"static": map[string]any{
				"statement": `invalid_keyword(principal, action, resource);`,
			},
		},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationException")
}

func TestVP_CedarPolicyValidation_Permit(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyStoreId": storeID,
		"definition": map[string]any{
			"static": map[string]any{
				"statement": `permit(principal, action, resource);`,
			},
		},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, "STATIC", m["policyType"])
}

func TestVP_CedarPolicyValidation_Forbid(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	resp, err := s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyStoreId": storeID,
		"definition": map[string]any{
			"static": map[string]any{
				"statement": `forbid(principal, action, resource);`,
			},
		},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, "STATIC", m["policyType"])
}

func TestVP_GetPolicy_NotFound(t *testing.T) {
	s := newService()
	storeID := createPolicyStore(t, s)
	_, err := s.HandleRequest(jsonCtx("GetPolicy", map[string]any{
		"policyStoreId": storeID, "policyId": "nonexistent",
	}))
	require.Error(t, err)
}
