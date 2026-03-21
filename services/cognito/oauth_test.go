package cognito_test

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// cognitoOAuthReq builds a request to an OAuth/OIDC path with the cognito-idp credential scope
// so the gateway routes it to the Cognito service.
func cognitoOAuthReq(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/cognito-idp/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// setupPoolAndUser creates a pool, client, and confirmed user via the JSON API.
// Returns poolID, clientID.
func setupPoolAndUser(t *testing.T, handler http.Handler) (string, string) {
	t.Helper()

	// Create pool.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "CreateUserPool", map[string]interface{}{
		"PoolName": "oauth-test-pool",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPool: %d %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	poolID := m["UserPool"].(map[string]interface{})["Id"].(string)

	// Create client.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "CreateUserPoolClient", map[string]interface{}{
		"UserPoolId":        poolID,
		"ClientName":        "oauth-client",
		"GenerateSecret":    true,
		"ExplicitAuthFlows": []string{"ALLOW_USER_PASSWORD_AUTH"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPoolClient: %d %s", w.Code, w.Body.String())
	}
	m = decodeJSON(t, w.Body.String())
	clientID := m["UserPoolClient"].(map[string]interface{})["ClientId"].(string)

	// SignUp user.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "SignUp", map[string]interface{}{
		"ClientId": clientID,
		"Username": "oauthuser@example.com",
		"Password": "TestPass123!",
		"UserAttributes": []map[string]string{
			{"Name": "email", "Value": "oauthuser@example.com"},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("setup SignUp: %d %s", w.Code, w.Body.String())
	}

	// Confirm user.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "AdminConfirmSignUp", map[string]interface{}{
		"UserPoolId": poolID,
		"Username":   "oauthuser@example.com",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("setup AdminConfirmSignUp: %d %s", w.Code, w.Body.String())
	}

	return poolID, clientID
}

// ---- Test: JWKS endpoint returns valid JWK with RSA public key ----

func TestOAuth_JWKS(t *testing.T) {
	handler := newCognitoGateway(t)
	poolID, _ := setupPoolAndUser(t, handler)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoOAuthReq(t, http.MethodGet, "/"+poolID+"/.well-known/jwks.json", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("JWKS: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &jwks); err != nil {
		t.Fatalf("JWKS: unmarshal: %v", err)
	}
	if len(jwks.Keys) != 1 {
		t.Fatalf("JWKS: expected 1 key, got %d", len(jwks.Keys))
	}

	key := jwks.Keys[0]
	if key.Kty != "RSA" {
		t.Errorf("JWKS: expected kty=RSA, got %q", key.Kty)
	}
	if key.Alg != "RS256" {
		t.Errorf("JWKS: expected alg=RS256, got %q", key.Alg)
	}
	if key.Use != "sig" {
		t.Errorf("JWKS: expected use=sig, got %q", key.Use)
	}
	if key.Kid == "" {
		t.Error("JWKS: kid is empty")
	}
	if key.N == "" {
		t.Error("JWKS: n (modulus) is empty")
	}
	if key.E == "" {
		t.Error("JWKS: e (exponent) is empty")
	}
}

// ---- Test: OIDC discovery returns valid document ----

func TestOAuth_OIDCDiscovery(t *testing.T) {
	handler := newCognitoGateway(t)
	poolID, _ := setupPoolAndUser(t, handler)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoOAuthReq(t, http.MethodGet, "/"+poolID+"/.well-known/openid-configuration", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("OIDC discovery: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("OIDC discovery: unmarshal: %v", err)
	}

	issuer, _ := doc["issuer"].(string)
	if !strings.Contains(issuer, poolID) {
		t.Errorf("OIDC discovery: issuer %q does not contain pool ID %q", issuer, poolID)
	}

	required := []string{
		"authorization_endpoint", "token_endpoint", "userinfo_endpoint",
		"jwks_uri", "response_types_supported", "subject_types_supported",
		"id_token_signing_alg_values_supported", "scopes_supported",
	}
	for _, field := range required {
		if doc[field] == nil {
			t.Errorf("OIDC discovery: missing field %q", field)
		}
	}

	jwksURI, _ := doc["jwks_uri"].(string)
	if !strings.Contains(jwksURI, poolID) || !strings.HasSuffix(jwksURI, "/jwks.json") {
		t.Errorf("OIDC discovery: jwks_uri %q is incorrect", jwksURI)
	}
}

// ---- Test: InitiateAuth returns verifiable JWT ----

func TestOAuth_InitiateAuth_VerifiableJWT(t *testing.T) {
	handler := newCognitoGateway(t)
	poolID, clientID := setupPoolAndUser(t, handler)

	// Authenticate.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "InitiateAuth", map[string]interface{}{
		"AuthFlow": "USER_PASSWORD_AUTH",
		"ClientId": clientID,
		"AuthParameters": map[string]string{
			"USERNAME": "oauthuser@example.com",
			"PASSWORD": "TestPass123!",
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("InitiateAuth: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	authResult := m["AuthenticationResult"].(map[string]interface{})
	accessToken := authResult["AccessToken"].(string)
	idToken := authResult["IdToken"].(string)

	// Fetch JWKS to get the public key.
	wJWKS := httptest.NewRecorder()
	handler.ServeHTTP(wJWKS, cognitoOAuthReq(t, http.MethodGet, "/"+poolID+"/.well-known/jwks.json", ""))
	if wJWKS.Code != http.StatusOK {
		t.Fatalf("JWKS: expected 200, got %d", wJWKS.Code)
	}

	pubKey := parseRSAPublicKeyFromJWKS(t, wJWKS.Body.Bytes())

	// Verify access token signature.
	verifyJWTSignature(t, accessToken, pubKey, "AccessToken")

	// Verify id token signature.
	verifyJWTSignature(t, idToken, pubKey, "IdToken")

	// Verify access token claims.
	accessClaims := decodeJWTClaims(t, accessToken)
	if accessClaims["token_use"] != "access" {
		t.Errorf("AccessToken: expected token_use=access, got %v", accessClaims["token_use"])
	}
	if accessClaims["client_id"] != clientID {
		t.Errorf("AccessToken: expected client_id=%s, got %v", clientID, accessClaims["client_id"])
	}
	if accessClaims["scope"] == nil {
		t.Error("AccessToken: missing scope claim")
	}

	// Verify id token claims.
	idClaims := decodeJWTClaims(t, idToken)
	if idClaims["token_use"] != "id" {
		t.Errorf("IdToken: expected token_use=id, got %v", idClaims["token_use"])
	}
	if idClaims["aud"] != clientID {
		t.Errorf("IdToken: expected aud=%s, got %v", clientID, idClaims["aud"])
	}
	if idClaims["email"] != "oauthuser@example.com" {
		t.Errorf("IdToken: expected email=oauthuser@example.com, got %v", idClaims["email"])
	}
	if idClaims["cognito:username"] != "oauthuser@example.com" {
		t.Errorf("IdToken: expected cognito:username=oauthuser@example.com, got %v", idClaims["cognito:username"])
	}
}

// ---- Test: Token endpoint with authorization_code grant ----

func TestOAuth_TokenEndpoint_AuthorizationCode(t *testing.T) {
	handler := newCognitoGateway(t)
	poolID, clientID := setupPoolAndUser(t, handler)

	// Submit login form to get an auth code.
	formData := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {"https://example.com/callback"},
		"response_type": {"code"},
		"username":      {"oauthuser@example.com"},
		"password":      {"TestPass123!"},
	}
	wLogin := httptest.NewRecorder()
	handler.ServeHTTP(wLogin, cognitoOAuthReq(t, http.MethodPost, "/login", formData.Encode()))
	if wLogin.Code != http.StatusFound {
		t.Fatalf("Login: expected 302, got %d\nbody: %s", wLogin.Code, wLogin.Body.String())
	}

	// Extract code from redirect Location header.
	location := wLogin.Header().Get("Location")
	if location == "" {
		t.Fatal("Login: missing Location header")
	}
	redirectURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("Login: parse Location: %v", err)
	}
	code := redirectURL.Query().Get("code")
	if code == "" {
		t.Fatalf("Login: missing code in redirect URL: %s", location)
	}

	// Exchange code for tokens.
	tokenForm := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"client_id":    {clientID},
		"redirect_uri": {"https://example.com/callback"},
	}
	wToken := httptest.NewRecorder()
	handler.ServeHTTP(wToken, cognitoOAuthReq(t, http.MethodPost, "/oauth2/token", tokenForm.Encode()))
	if wToken.Code != http.StatusOK {
		t.Fatalf("Token endpoint: expected 200, got %d\nbody: %s", wToken.Code, wToken.Body.String())
	}

	var tokenResp map[string]interface{}
	if err := json.Unmarshal(wToken.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("Token endpoint: unmarshal: %v", err)
	}

	if tokenResp["access_token"] == nil || tokenResp["access_token"].(string) == "" {
		t.Error("Token endpoint: missing access_token")
	}
	if tokenResp["id_token"] == nil || tokenResp["id_token"].(string) == "" {
		t.Error("Token endpoint: missing id_token")
	}
	if tokenResp["refresh_token"] == nil || tokenResp["refresh_token"].(string) == "" {
		t.Error("Token endpoint: missing refresh_token")
	}
	if tokenResp["token_type"] != "Bearer" {
		t.Errorf("Token endpoint: expected token_type=Bearer, got %v", tokenResp["token_type"])
	}
	if tokenResp["expires_in"] == nil || tokenResp["expires_in"].(float64) != 3600 {
		t.Errorf("Token endpoint: expected expires_in=3600, got %v", tokenResp["expires_in"])
	}

	// Verify the access token signature.
	wJWKS := httptest.NewRecorder()
	handler.ServeHTTP(wJWKS, cognitoOAuthReq(t, http.MethodGet, "/"+poolID+"/.well-known/jwks.json", ""))
	pubKey := parseRSAPublicKeyFromJWKS(t, wJWKS.Body.Bytes())
	verifyJWTSignature(t, tokenResp["access_token"].(string), pubKey, "AuthCode AccessToken")
}

// ---- Test: Token endpoint with refresh_token grant ----

func TestOAuth_TokenEndpoint_RefreshToken(t *testing.T) {
	handler := newCognitoGateway(t)
	_, clientID := setupPoolAndUser(t, handler)

	// Get initial tokens via InitiateAuth.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "InitiateAuth", map[string]interface{}{
		"AuthFlow": "USER_PASSWORD_AUTH",
		"ClientId": clientID,
		"AuthParameters": map[string]string{
			"USERNAME": "oauthuser@example.com",
			"PASSWORD": "TestPass123!",
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("InitiateAuth: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	refreshToken := m["AuthenticationResult"].(map[string]interface{})["RefreshToken"].(string)

	// Refresh tokens.
	refreshForm := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}
	wRefresh := httptest.NewRecorder()
	handler.ServeHTTP(wRefresh, cognitoOAuthReq(t, http.MethodPost, "/oauth2/token", refreshForm.Encode()))
	if wRefresh.Code != http.StatusOK {
		t.Fatalf("Refresh token: expected 200, got %d\nbody: %s", wRefresh.Code, wRefresh.Body.String())
	}

	var tokenResp map[string]interface{}
	if err := json.Unmarshal(wRefresh.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("Refresh token: unmarshal: %v", err)
	}

	if tokenResp["access_token"] == nil || tokenResp["access_token"].(string) == "" {
		t.Error("Refresh token: missing access_token")
	}
	if tokenResp["id_token"] == nil || tokenResp["id_token"].(string) == "" {
		t.Error("Refresh token: missing id_token")
	}
	if tokenResp["token_type"] != "Bearer" {
		t.Errorf("Refresh token: expected token_type=Bearer, got %v", tokenResp["token_type"])
	}
}

// ---- helpers ----

// parseRSAPublicKeyFromJWKS extracts the first RSA public key from a JWKS response body.
func parseRSAPublicKeyFromJWKS(t *testing.T, body []byte) *rsa.PublicKey {
	t.Helper()

	var jwks struct {
		Keys []struct {
			N string `json:"n"`
			E string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		t.Fatalf("parseRSAPublicKeyFromJWKS: %v", err)
	}
	if len(jwks.Keys) == 0 {
		t.Fatal("parseRSAPublicKeyFromJWKS: no keys")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwks.Keys[0].N)
	if err != nil {
		t.Fatalf("decode N: %v", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwks.Keys[0].E)
	if err != nil {
		t.Fatalf("decode E: %v", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}
}

// verifyJWTSignature verifies that a JWT's RS256 signature is valid with the given public key.
func verifyJWTSignature(t *testing.T, token string, pubKey *rsa.PublicKey, label string) {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("%s: expected 3 JWT parts, got %d", label, len(parts))
	}

	signingInput := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("%s: decode signature: %v", label, err)
	}

	hash := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes); err != nil {
		t.Errorf("%s: signature verification failed: %v", label, err)
	}
}

// decodeJWTClaims decodes the payload of a JWT without verifying the signature.
func decodeJWTClaims(t *testing.T, token string) map[string]interface{} {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}
	return claims
}
