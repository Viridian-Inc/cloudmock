package cognito_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Helper: create a user pool and return its Id. Every IdP test needs
// a pool to anchor the provider against.
func createPoolForIdpTest(t *testing.T, handler http.Handler) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "idp-test-pool",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateUserPool setup: status %d, body %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w.Body.String())
	pool, ok := resp["UserPool"].(map[string]any)
	if !ok {
		t.Fatalf("CreateUserPool setup: no UserPool in response: %s", w.Body.String())
	}
	poolID, _ := pool["Id"].(string)
	if poolID == "" {
		t.Fatalf("CreateUserPool setup: empty pool ID: %s", w.Body.String())
	}
	return poolID
}

func TestIdp_CreateDescribeUpdateDelete_SAML(t *testing.T) {
	handler := newCognitoGateway(t)
	poolID := createPoolForIdpTest(t, handler)

	// Create — SAML with MetadataURL + attribute mapping
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, cognitoReq(t, "CreateIdentityProvider", map[string]any{
		"UserPoolId":   poolID,
		"ProviderName": "sp-en-acme-saml",
		"ProviderType": "SAML",
		"ProviderDetails": map[string]string{
			"MetadataURL": "https://adfs.acme.edu/FederationMetadata/2007-06/FederationMetadata.xml",
			"IDPSignout":  "true",
		},
		"AttributeMapping": map[string]string{
			"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
		},
		"IdpIdentifiers": []string{"acme.edu"},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateIdentityProvider: status %d, body %s", wc.Code, wc.Body.String())
	}
	cr := decodeJSON(t, wc.Body.String())
	idp, ok := cr["IdentityProvider"].(map[string]any)
	if !ok {
		t.Fatalf("CreateIdentityProvider: no IdentityProvider in response: %s", wc.Body.String())
	}
	if idp["ProviderName"].(string) != "sp-en-acme-saml" {
		t.Errorf("unexpected ProviderName: %v", idp["ProviderName"])
	}
	if idp["ProviderType"].(string) != "SAML" {
		t.Errorf("unexpected ProviderType: %v", idp["ProviderType"])
	}
	details := idp["ProviderDetails"].(map[string]any)
	if details["MetadataURL"].(string) == "" {
		t.Errorf("ProviderDetails.MetadataURL lost in round-trip: %v", details)
	}
	if _, ok := idp["CreationDate"]; !ok {
		t.Error("Create response missing CreationDate")
	}
	if _, ok := idp["LastModifiedDate"]; !ok {
		t.Error("Create response missing LastModifiedDate")
	}

	// Describe — should return the same thing including ProviderDetails
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, cognitoReq(t, "DescribeIdentityProvider", map[string]any{
		"UserPoolId":   poolID,
		"ProviderName": "sp-en-acme-saml",
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeIdentityProvider: status %d, body %s", wd.Code, wd.Body.String())
	}
	dr := decodeJSON(t, wd.Body.String())
	descIdp := dr["IdentityProvider"].(map[string]any)
	if descIdp["ProviderType"].(string) != "SAML" {
		t.Errorf("Describe ProviderType mismatch: %v", descIdp["ProviderType"])
	}

	// Update — change only AttributeMapping; ProviderDetails should stay
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, cognitoReq(t, "UpdateIdentityProvider", map[string]any{
		"UserPoolId":   poolID,
		"ProviderName": "sp-en-acme-saml",
		"AttributeMapping": map[string]string{
			"email":      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"given_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
		},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UpdateIdentityProvider: status %d, body %s", wu.Code, wu.Body.String())
	}
	updIdp := decodeJSON(t, wu.Body.String())["IdentityProvider"].(map[string]any)
	am := updIdp["AttributeMapping"].(map[string]any)
	if _, ok := am["given_name"]; !ok {
		t.Errorf("Update didn't add given_name to AttributeMapping: %v", am)
	}
	// ProviderDetails preserved (update did not pass it)
	updDetails := updIdp["ProviderDetails"].(map[string]any)
	if updDetails["MetadataURL"].(string) == "" {
		t.Errorf("Update cleared ProviderDetails — should preserve when omitted")
	}
	// LastModifiedDate bumped
	orig := idp["LastModifiedDate"].(float64)
	now := updIdp["LastModifiedDate"].(float64)
	if now < orig {
		t.Errorf("Update didn't bump LastModifiedDate: orig=%v now=%v", orig, now)
	}

	// Delete
	wx := httptest.NewRecorder()
	handler.ServeHTTP(wx, cognitoReq(t, "DeleteIdentityProvider", map[string]any{
		"UserPoolId":   poolID,
		"ProviderName": "sp-en-acme-saml",
	}))
	if wx.Code != http.StatusOK {
		t.Fatalf("DeleteIdentityProvider: status %d, body %s", wx.Code, wx.Body.String())
	}

	// Describe again — ResourceNotFoundException
	wm := httptest.NewRecorder()
	handler.ServeHTTP(wm, cognitoReq(t, "DescribeIdentityProvider", map[string]any{
		"UserPoolId":   poolID,
		"ProviderName": "sp-en-acme-saml",
	}))
	if wm.Code == http.StatusOK {
		t.Errorf("DescribeIdentityProvider after delete: expected error, got 200: %s", wm.Body.String())
	}
}

func TestIdp_Create_OIDC(t *testing.T) {
	handler := newCognitoGateway(t)
	poolID := createPoolForIdpTest(t, handler)

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, cognitoReq(t, "CreateIdentityProvider", map[string]any{
		"UserPoolId":   poolID,
		"ProviderName": "okta-company",
		"ProviderType": "OIDC",
		"ProviderDetails": map[string]string{
			"client_id":                   "okta-client-123",
			"client_secret":               "secret-abc",
			"attributes_request_method":   "GET",
			"oidc_issuer":                 "https://company.okta.com",
			"authorize_scopes":            "openid profile email",
			"authorize_url":               "https://company.okta.com/oauth2/v1/authorize",
			"token_url":                   "https://company.okta.com/oauth2/v1/token",
			"attributes_url":              "https://company.okta.com/oauth2/v1/userinfo",
			"jwks_uri":                    "https://company.okta.com/oauth2/v1/keys",
		},
		"AttributeMapping": map[string]string{
			"email": "email",
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateIdentityProvider OIDC: status %d, body %s", wc.Code, wc.Body.String())
	}
	cr := decodeJSON(t, wc.Body.String())
	idp := cr["IdentityProvider"].(map[string]any)
	if idp["ProviderType"].(string) != "OIDC" {
		t.Errorf("ProviderType: %v", idp["ProviderType"])
	}
	details := idp["ProviderDetails"].(map[string]any)
	if details["oidc_issuer"].(string) != "https://company.okta.com" {
		t.Errorf("OIDC issuer lost: %v", details)
	}
}

func TestIdp_Create_DuplicateProviderException(t *testing.T) {
	handler := newCognitoGateway(t)
	poolID := createPoolForIdpTest(t, handler)

	// First create — ok
	body := map[string]any{
		"UserPoolId":   poolID,
		"ProviderName": "dup-test",
		"ProviderType": "SAML",
		"ProviderDetails": map[string]string{
			"MetadataURL": "https://idp.example.com/metadata.xml",
		},
	}
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, cognitoReq(t, "CreateIdentityProvider", body))
	if w1.Code != http.StatusOK {
		t.Fatalf("first create: %d %s", w1.Code, w1.Body.String())
	}

	// Second create with the same name — DuplicateProviderException
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, cognitoReq(t, "CreateIdentityProvider", body))
	if w2.Code == http.StatusOK {
		t.Fatalf("second create should fail, got 200: %s", w2.Body.String())
	}
	resp := decodeJSON(t, w2.Body.String())
	if resp["__type"] != "DuplicateProviderException" {
		t.Errorf("expected DuplicateProviderException, got %v\nfull: %s",
			resp["__type"], w2.Body.String())
	}
}

func TestIdp_Create_InvalidProviderType(t *testing.T) {
	handler := newCognitoGateway(t)
	poolID := createPoolForIdpTest(t, handler)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "CreateIdentityProvider", map[string]any{
		"UserPoolId":   poolID,
		"ProviderName": "weird",
		"ProviderType": "LDAP", // not in the AWS enum
		"ProviderDetails": map[string]string{
			"foo": "bar",
		},
	}))
	if w.Code == http.StatusOK {
		t.Fatalf("expected InvalidParameterException, got 200: %s", w.Body.String())
	}
	resp := decodeJSON(t, w.Body.String())
	if resp["__type"] != "InvalidParameterException" {
		t.Errorf("expected InvalidParameterException, got %v", resp["__type"])
	}
}

func TestIdp_List_SortedAndRespectsMaxResults(t *testing.T) {
	handler := newCognitoGateway(t)
	poolID := createPoolForIdpTest(t, handler)

	// Create 3 in non-sorted order
	for _, name := range []string{"zeta", "alpha", "mu"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, cognitoReq(t, "CreateIdentityProvider", map[string]any{
			"UserPoolId":   poolID,
			"ProviderName": name,
			"ProviderType": "SAML",
			"ProviderDetails": map[string]string{
				"MetadataURL": "https://idp.example.com/" + name + ".xml",
			},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("create %s: %d %s", name, w.Code, w.Body.String())
		}
	}

	// List — sorted alphabetically
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, cognitoReq(t, "ListIdentityProviders", map[string]any{
		"UserPoolId": poolID,
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("list: %d %s", wl.Code, wl.Body.String())
	}
	providers := decodeJSON(t, wl.Body.String())["Providers"].([]any)
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(providers))
	}
	names := []string{
		providers[0].(map[string]any)["ProviderName"].(string),
		providers[1].(map[string]any)["ProviderName"].(string),
		providers[2].(map[string]any)["ProviderName"].(string),
	}
	if names[0] != "alpha" || names[1] != "mu" || names[2] != "zeta" {
		t.Errorf("not sorted: %v", names)
	}
	// ProviderDetails should NOT leak in List — only summaries
	if _, ok := providers[0].(map[string]any)["ProviderDetails"]; ok {
		t.Errorf("List leaked ProviderDetails — should be summary only")
	}

	// MaxResults clamp
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, cognitoReq(t, "ListIdentityProviders", map[string]any{
		"UserPoolId": poolID,
		"MaxResults": 2,
	}))
	if wl2.Code != http.StatusOK {
		t.Fatalf("list maxResults=2: %d %s", wl2.Code, wl2.Body.String())
	}
	capped := decodeJSON(t, wl2.Body.String())["Providers"].([]any)
	if len(capped) != 2 {
		t.Errorf("MaxResults=2 didn't clamp, got %d", len(capped))
	}
}

func TestIdp_PoolNotFound(t *testing.T) {
	handler := newCognitoGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "CreateIdentityProvider", map[string]any{
		"UserPoolId":   "us-east-1_DOES_NOT_EXIST",
		"ProviderName": "ghost",
		"ProviderType": "SAML",
		"ProviderDetails": map[string]string{
			"MetadataURL": "https://ghost.example.com/metadata.xml",
		},
	}))
	if w.Code == http.StatusOK {
		t.Fatalf("expected ResourceNotFoundException on missing pool, got 200: %s", w.Body.String())
	}
	resp := decodeJSON(t, w.Body.String())
	if resp["__type"] != "ResourceNotFoundException" {
		t.Errorf("expected ResourceNotFoundException, got %v", resp["__type"])
	}
}
