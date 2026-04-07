package cognito

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const loginPageHTML = `<!DOCTYPE html>
<html>
<head><title>CloudMock - Sign In</title></head>
<body>
  <h2>Sign In</h2>
  <form method="POST" action="/login">
    <input type="hidden" name="client_id" value="%s">
    <input type="hidden" name="redirect_uri" value="%s">
    <input type="hidden" name="response_type" value="%s">
    <input type="hidden" name="state" value="%s">
    <label>Username: <input type="text" name="username"></label><br>
    <label>Password: <input type="password" name="password"></label><br>
    <button type="submit">Sign In</button>
  </form>
</body>
</html>`

// handleLoginPage serves the hosted UI login form.
func (s *CognitoService) handleLoginPage(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	responseType := r.URL.Query().Get("response_type")
	state := r.URL.Query().Get("state")

	html := fmt.Sprintf(loginPageHTML, clientID, redirectURI, responseType, state)
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(html),
		RawContentType: "text/html; charset=utf-8",
	}, nil
}

// handleLoginSubmit processes the login form submission.
func (s *CognitoService) handleLoginSubmit(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	// The gateway already consumed r.Body into ctx.Body, so restore it for ParseForm.
	r.Body = io.NopCloser(bytes.NewReader(ctx.Body))
	if err := r.ParseForm(); err != nil {
		return oauthError("invalid_request", "Could not parse form.", http.StatusBadRequest)
	}

	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	responseType := r.FormValue("response_type")
	state := r.FormValue("state")
	username := r.FormValue("username")
	password := r.FormValue("password")

	if clientID == "" || redirectURI == "" || username == "" || password == "" {
		return oauthError("invalid_request",
			"client_id, redirect_uri, username, and password are required.", http.StatusBadRequest)
	}

	// Look up the pool and validate credentials.
	pool, awsErr := s.store.findPoolByClientID(clientID)
	if awsErr != nil {
		return oauthError("invalid_client", "Unknown client.", http.StatusBadRequest)
	}

	s.store.mu.RLock()
	user, ok := pool.Users[username]
	poolID := pool.Id
	s.store.mu.RUnlock()

	if !ok || !checkPassword(user.PasswordHash, password) {
		return oauthError("invalid_grant", "Incorrect username or password.", http.StatusUnauthorized)
	}

	if responseType == "code" {
		// Generate auth code and redirect.
		entry := &authCodeEntry{
			UserPoolID:  poolID,
			ClientID:    clientID,
			Username:    username,
			Sub:         user.Sub,
			RedirectURI: redirectURI,
			CreatedAt:   time.Now().UTC(),
		}
		code := s.authCodes.Put(entry)

		redirectURL, err := url.Parse(redirectURI)
		if err != nil {
			return oauthError("invalid_request", "Invalid redirect_uri.", http.StatusBadRequest)
		}
		q := redirectURL.Query()
		q.Set("code", code)
		if state != "" {
			q.Set("state", state)
		}
		redirectURL.RawQuery = q.Encode()

		return &service.Response{
			StatusCode: http.StatusFound,
			Headers: map[string]string{
				"Location": redirectURL.String(),
			},
		}, nil
	}

	// Implicit flow (response_type=token) — return tokens in fragment.
	tokens, err := s.generateTokens(pool.Id, clientID, user)
	if err != nil {
		return oauthError("server_error", "Failed to generate tokens.", http.StatusInternalServerError)
	}

	redirectURL, parseErr := url.Parse(redirectURI)
	if parseErr != nil {
		return oauthError("invalid_request", "Invalid redirect_uri.", http.StatusBadRequest)
	}
	fragment := fmt.Sprintf("access_token=%s&id_token=%s&token_type=Bearer&expires_in=3600",
		tokens.AccessToken, tokens.IDToken)
	if state != "" {
		fragment += "&state=" + state
	}
	redirectURL.Fragment = fragment

	return &service.Response{
		StatusCode: http.StatusFound,
		Headers: map[string]string{
			"Location": redirectURL.String(),
		},
	}, nil
}
