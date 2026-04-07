package cognito

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// authCodeEntry stores an authorization code pending exchange.
type authCodeEntry struct {
	UserPoolID  string
	ClientID    string
	Username    string
	Sub         string
	RedirectURI string
	CreatedAt   time.Time
}

// authCodeStore holds pending authorization codes.
type authCodeStore struct {
	mu    sync.Mutex
	codes map[string]*authCodeEntry
}

func newAuthCodeStore() *authCodeStore {
	return &authCodeStore{
		codes: make(map[string]*authCodeEntry),
	}
}

// Put stores an auth code entry and returns the code.
func (s *authCodeStore) Put(entry *authCodeEntry) string {
	code := newUUID()
	s.mu.Lock()
	s.codes[code] = entry
	s.mu.Unlock()
	return code
}

// Take retrieves and removes an auth code entry. Returns nil if not found or expired.
func (s *authCodeStore) Take(code string) *authCodeEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.codes[code]
	if !ok {
		return nil
	}
	delete(s.codes, code)
	// Expire after 5 minutes.
	if time.Since(entry.CreatedAt) > 5*time.Minute {
		return nil
	}
	return entry
}

// handleOAuth routes OAuth/OIDC REST-style requests.
func (s *CognitoService) handleOAuth(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, "/.well-known/openid-configuration"):
		return s.handleOIDCDiscovery(ctx, path)
	case strings.HasSuffix(path, "/.well-known/jwks.json"):
		return s.handleJWKS(ctx)
	case path == "/oauth2/token" && r.Method == http.MethodPost:
		return s.handleTokenEndpoint(ctx)
	case path == "/oauth2/authorize" && r.Method == http.MethodGet:
		return s.handleAuthorize(ctx)
	case path == "/oauth2/userInfo" && r.Method == http.MethodGet:
		return s.handleUserInfo(ctx)
	case path == "/login" && r.Method == http.MethodGet:
		return s.handleLoginPage(ctx)
	case path == "/login" && r.Method == http.MethodPost:
		return s.handleLoginSubmit(ctx)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidRequest",
				"Unknown OAuth/OIDC endpoint: "+path,
				http.StatusNotFound)
	}
}

// handleOIDCDiscovery returns the OIDC discovery document for a user pool.
func (s *CognitoService) handleOIDCDiscovery(ctx *service.RequestContext, path string) (*service.Response, error) {
	// Extract pool ID from path: /{poolId}/.well-known/openid-configuration
	// If no pool ID in path (e.g. bare /.well-known/openid-configuration),
	// fall back to the first available user pool for local dev convenience.
	poolID := extractPoolIDFromPath(path)
	if poolID == "" {
		pools := s.store.ListUserPools(1)
		if len(pools) > 0 {
			poolID = pools[0].Id
		}
	}
	if poolID == "" {
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("ResourceNotFoundException",
				"No user pools exist. Create one first.", http.StatusNotFound)
	}

	// Verify pool exists.
	if _, awsErr := s.store.GetUserPool(poolID); awsErr != nil {
		return &service.Response{Format: service.FormatJSON}, awsErr
	}

	baseURL := s.baseURL(ctx)
	issuer := fmt.Sprintf("%s/%s", baseURL, poolID)

	doc := map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                baseURL + "/oauth2/authorize",
		"token_endpoint":                        baseURL + "/oauth2/token",
		"userinfo_endpoint":                     baseURL + "/oauth2/userInfo",
		"jwks_uri":                              fmt.Sprintf("%s/%s/.well-known/jwks.json", baseURL, poolID),
		"response_types_supported":              []string{"code", "token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
	}

	body, _ := json.Marshal(doc)
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        body,
		RawContentType: "application/json",
	}, nil
}

// handleJWKS returns the JWKS (public key set) for JWT verification.
func (s *CognitoService) handleJWKS(ctx *service.RequestContext) (*service.Response, error) {
	jwks := map[string]any{
		"keys": []any{s.keys.JWK()},
	}
	body, _ := json.Marshal(jwks)
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        body,
		RawContentType: "application/json",
	}, nil
}

// handleTokenEndpoint handles POST /oauth2/token.
func (s *CognitoService) handleTokenEndpoint(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest

	// The gateway already consumed r.Body into ctx.Body, so restore it for ParseForm.
	r.Body = io.NopCloser(bytes.NewReader(ctx.Body))
	if err := r.ParseForm(); err != nil {
		return oauthError("invalid_request", "Could not parse form body.", http.StatusBadRequest)
	}

	grantType := r.FormValue("grant_type")
	switch grantType {
	case "authorization_code":
		return s.handleAuthCodeGrant(ctx)
	case "refresh_token":
		return s.handleRefreshTokenGrant(ctx)
	case "client_credentials":
		return s.handleClientCredentialsGrant(ctx)
	default:
		return oauthError("unsupported_grant_type",
			fmt.Sprintf("Unsupported grant_type: %s", grantType), http.StatusBadRequest)
	}
}

// handleAuthCodeGrant exchanges an authorization code for tokens.
func (s *CognitoService) handleAuthCodeGrant(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")

	if code == "" || clientID == "" {
		return oauthError("invalid_request", "code and client_id are required.", http.StatusBadRequest)
	}

	entry := s.authCodes.Take(code)
	if entry == nil {
		return oauthError("invalid_grant", "Authorization code is invalid or expired.", http.StatusBadRequest)
	}

	if entry.ClientID != clientID {
		return oauthError("invalid_grant", "client_id does not match.", http.StatusBadRequest)
	}
	if redirectURI != "" && entry.RedirectURI != redirectURI {
		return oauthError("invalid_grant", "redirect_uri does not match.", http.StatusBadRequest)
	}

	// Look up the user.
	pool, awsErr := s.store.GetUserPool(entry.UserPoolID)
	if awsErr != nil {
		return oauthError("server_error", "User pool not found.", http.StatusInternalServerError)
	}

	s.store.mu.RLock()
	user, ok := pool.Users[entry.Username]
	s.store.mu.RUnlock()
	if !ok {
		return oauthError("server_error", "User not found.", http.StatusInternalServerError)
	}

	tokens, err := s.generateTokens(entry.UserPoolID, clientID, user)
	if err != nil {
		return oauthError("server_error", "Failed to generate tokens.", http.StatusInternalServerError)
	}

	return oauthTokenResponse(tokens)
}

// handleRefreshTokenGrant refreshes an access/id token pair.
func (s *CognitoService) handleRefreshTokenGrant(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	refreshToken := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")

	if refreshToken == "" || clientID == "" {
		return oauthError("invalid_request", "refresh_token and client_id are required.", http.StatusBadRequest)
	}

	// Decode the refresh token to extract sub and pool info.
	claims, err := decodeJWTPayload(refreshToken)
	if err != nil {
		return oauthError("invalid_grant", "Invalid refresh token.", http.StatusBadRequest)
	}

	sub, _ := claims["sub"].(string)
	iss, _ := claims["iss"].(string)
	tokenUse, _ := claims["token_use"].(string)
	if sub == "" || iss == "" || tokenUse != "refresh" {
		return oauthError("invalid_grant", "Invalid refresh token claims.", http.StatusBadRequest)
	}

	// Extract pool ID from issuer.
	poolID := poolIDFromIssuer(iss)
	if poolID == "" {
		return oauthError("invalid_grant", "Cannot determine user pool from refresh token.", http.StatusBadRequest)
	}

	// Find the user by sub.
	pool, awsErr := s.store.GetUserPool(poolID)
	if awsErr != nil {
		return oauthError("invalid_grant", "User pool not found.", http.StatusBadRequest)
	}

	s.store.mu.RLock()
	var user *User
	for _, u := range pool.Users {
		if u.Sub == sub {
			user = u
			break
		}
	}
	s.store.mu.RUnlock()

	if user == nil {
		return oauthError("invalid_grant", "User not found.", http.StatusBadRequest)
	}

	tokens, err := s.generateTokens(poolID, clientID, user)
	if err != nil {
		return oauthError("server_error", "Failed to generate tokens.", http.StatusInternalServerError)
	}

	return oauthTokenResponse(tokens)
}

// handleClientCredentialsGrant handles the client_credentials flow.
func (s *CognitoService) handleClientCredentialsGrant(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	scope := r.FormValue("scope")

	if clientID == "" || clientSecret == "" {
		return oauthError("invalid_client", "client_id and client_secret are required.", http.StatusBadRequest)
	}

	// Verify client exists and secret matches.
	pool, awsErr := s.store.findPoolByClientID(clientID)
	if awsErr != nil {
		return oauthError("invalid_client", "Unknown client.", http.StatusBadRequest)
	}

	s.store.mu.RLock()
	client := pool.Clients[clientID]
	s.store.mu.RUnlock()

	if client.ClientSecret == "" || client.ClientSecret != clientSecret {
		return oauthError("invalid_client", "Invalid client credentials.", http.StatusUnauthorized)
	}

	if scope == "" {
		scope = "openid"
	}

	now := time.Now().UTC()
	iss := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", s.store.region, pool.Id)

	accessClaims := map[string]any{
		"sub":       clientID,
		"iss":       iss,
		"client_id": clientID,
		"token_use": "access",
		"scope":     scope,
		"auth_time": now.Unix(),
		"iat":       now.Unix(),
		"exp":       now.Add(time.Hour).Unix(),
	}

	accessToken, err := signJWT(s.keys, accessClaims)
	if err != nil {
		return oauthError("server_error", "Failed to generate token.", http.StatusInternalServerError)
	}

	body, _ := json.Marshal(map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        body,
		RawContentType: "application/json",
	}, nil
}

// handleAuthorize handles GET /oauth2/authorize — redirects to login or returns an auth code.
func (s *CognitoService) handleAuthorize(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	responseType := r.URL.Query().Get("response_type")
	state := r.URL.Query().Get("state")

	if clientID == "" || redirectURI == "" || responseType == "" {
		return oauthError("invalid_request",
			"client_id, redirect_uri, and response_type are required.", http.StatusBadRequest)
	}

	// Redirect to login page with parameters.
	baseURL := s.baseURL(ctx)
	loginURL := fmt.Sprintf("%s/login?client_id=%s&redirect_uri=%s&response_type=%s",
		baseURL, clientID, redirectURI, responseType)
	if state != "" {
		loginURL += "&state=" + state
	}

	return &service.Response{
		StatusCode: http.StatusFound,
		Headers: map[string]string{
			"Location": loginURL,
		},
	}, nil
}

// handleUserInfo handles GET /oauth2/userInfo — returns user attributes for the access token.
func (s *CognitoService) handleUserInfo(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	authHeader := r.Header.Get("Authorization")
	var accessToken string
	if strings.HasPrefix(authHeader, "Bearer ") {
		accessToken = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if accessToken == "" {
		return oauthError("invalid_token", "Missing or invalid Bearer token.", http.StatusUnauthorized)
	}

	claims, err := decodeJWTPayload(accessToken)
	if err != nil {
		return oauthError("invalid_token", "Could not decode access token.", http.StatusUnauthorized)
	}

	sub, _ := claims["sub"].(string)
	iss, _ := claims["iss"].(string)
	username, _ := claims["username"].(string)

	if sub == "" || iss == "" {
		return oauthError("invalid_token", "Invalid token claims.", http.StatusUnauthorized)
	}

	poolID := poolIDFromIssuer(iss)
	if poolID == "" {
		return oauthError("invalid_token", "Cannot determine user pool.", http.StatusUnauthorized)
	}

	pool, awsErr := s.store.GetUserPool(poolID)
	if awsErr != nil {
		return oauthError("invalid_token", "User pool not found.", http.StatusUnauthorized)
	}

	s.store.mu.RLock()
	user, ok := pool.Users[username]
	s.store.mu.RUnlock()
	if !ok {
		return oauthError("invalid_token", "User not found.", http.StatusUnauthorized)
	}

	info := map[string]any{
		"sub":      user.Sub,
		"username": user.Username,
	}
	for k, v := range user.Attributes {
		if k != "sub" {
			info[k] = v
		}
	}

	body, _ := json.Marshal(info)
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        body,
		RawContentType: "application/json",
	}, nil
}

// generateTokens creates access, id, and refresh tokens for a user.
func (s *CognitoService) generateTokens(poolID, clientID string, user *User) (*oauthTokens, error) {
	now := time.Now().UTC()
	iss := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", s.store.region, poolID)

	accessClaims := map[string]any{
		"sub":       user.Sub,
		"iss":       iss,
		"client_id": clientID,
		"token_use": "access",
		"scope":     "openid email profile",
		"auth_time": now.Unix(),
		"iat":       now.Unix(),
		"exp":       now.Add(time.Hour).Unix(),
		"username":  user.Username,
	}

	idClaims := map[string]any{
		"sub":               user.Sub,
		"iss":               iss,
		"aud":               clientID,
		"token_use":         "id",
		"auth_time":         now.Unix(),
		"iat":               now.Unix(),
		"exp":               now.Add(time.Hour).Unix(),
		"email":             user.Attributes["email"],
		"email_verified":    true,
		"cognito:username":  user.Username,
	}

	refreshClaims := map[string]any{
		"sub":       user.Sub,
		"iss":       iss,
		"token_use": "refresh",
		"iat":       now.Unix(),
		"exp":       now.Add(30 * 24 * time.Hour).Unix(),
	}

	accessToken, err := signJWT(s.keys, accessClaims)
	if err != nil {
		return nil, err
	}
	idToken, err := signJWT(s.keys, idClaims)
	if err != nil {
		return nil, err
	}
	refreshToken, err := signJWT(s.keys, refreshClaims)
	if err != nil {
		return nil, err
	}

	return &oauthTokens{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}, nil
}

// oauthTokens holds the tokens for an OAuth token response.
type oauthTokens struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresIn    int
	TokenType    string
}

// oauthTokenResponse creates a standard OAuth token response.
func oauthTokenResponse(tokens *oauthTokens) (*service.Response, error) {
	body, _ := json.Marshal(map[string]any{
		"access_token":  tokens.AccessToken,
		"id_token":      tokens.IDToken,
		"refresh_token": tokens.RefreshToken,
		"token_type":    tokens.TokenType,
		"expires_in":    tokens.ExpiresIn,
	})
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        body,
		RawContentType: "application/json",
	}, nil
}

// oauthError returns an OAuth-style error response.
func oauthError(errorCode, description string, status int) (*service.Response, error) {
	body, _ := json.Marshal(map[string]string{
		"error":             errorCode,
		"error_description": description,
	})
	return &service.Response{
		StatusCode:     status,
		RawBody:        body,
		RawContentType: "application/json",
	}, nil
}

// baseURL returns the base URL for this service using the request's Host header.
func (s *CognitoService) baseURL(ctx *service.RequestContext) string {
	r := ctx.RawRequest
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:4566"
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

// extractPoolIDFromPath extracts the pool ID from a path like /{poolId}/.well-known/...
func extractPoolIDFromPath(path string) string {
	// Path format: /{poolId}/.well-known/openid-configuration or /{poolId}/.well-known/jwks.json
	path = strings.TrimPrefix(path, "/")
	idx := strings.Index(path, "/")
	if idx < 0 {
		return ""
	}
	candidate := path[:idx]
	// Pool IDs look like "us-east-1_aBcDeFgHi" — skip non-pool-ID path segments
	if strings.HasPrefix(candidate, ".") || !strings.Contains(candidate, "_") {
		return ""
	}
	return candidate
}

// poolIDFromIssuer extracts the pool ID from an issuer URL.
// e.g. "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_Abc123" → "us-east-1_Abc123"
func poolIDFromIssuer(iss string) string {
	idx := strings.LastIndex(iss, "/")
	if idx < 0 {
		return ""
	}
	return iss[idx+1:]
}

// decodeJWTPayload decodes the payload section of a JWT without verifying the signature.
func decodeJWTPayload(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT: expected 3 parts, got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}
	return claims, nil
}
