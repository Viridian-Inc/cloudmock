package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/admin"
	"github.com/Viridian-Inc/cloudmock/pkg/auth"
	authmemory "github.com/Viridian-Inc/cloudmock/pkg/auth/memory"
	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var registrationSecret = []byte("registration-integration-test-secret")

func newRegistrationAPI(t *testing.T, authEnabled, withUserStore bool) (http.Handler, *authmemory.Store) {
	t.Helper()
	cfg := config.Default()
	cfg.Auth.Enabled = authEnabled
	cfg.Auth.Secret = string(registrationSecret)
	api := admin.New(cfg, routing.NewRegistry(), gateway.NewRequestLog(10), gateway.NewRequestStats())
	store := authmemory.NewStore()
	if withUserStore {
		api.SetUserStore(store)
		api.SetAuthSecret(registrationSecret)
	}
	var handler http.Handler = api
	// Match cmd/gateway's conditional JWT wrapper. The stock gateway also
	// attaches a user store only when auth is enabled; embedded APIs may opt in.
	if cfg.Auth.Enabled {
		handler = auth.Middleware([]byte(cfg.Auth.Secret))(handler)
	}
	return handler, store
}

func registrationRequest(t *testing.T, handler http.Handler, method, path, authorization string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func registrationBearer(t *testing.T, role string, secret []byte, expiry time.Duration) string {
	t.Helper()
	token, err := auth.GenerateToken(&auth.User{ID: "caller", Email: "caller@example.test", Role: role}, secret, expiry)
	require.NoError(t, err)
	return "Bearer " + token
}

func TestAuthRegistrationAdminCreatesUserThroughMiddleware(t *testing.T) {
	for _, role := range []string{"", auth.RoleEditor} {
		name := role
		if name == "" {
			name = "default-viewer"
		}
		t.Run(name, func(t *testing.T) {
			handler, store := newRegistrationAPI(t, true, true)
			const email = "new-user@example.test"
			const password = "registration-test-password"
			w := registrationRequest(t, handler, http.MethodPost, "/api/auth/register",
				registrationBearer(t, auth.RoleAdmin, registrationSecret, time.Hour),
				map[string]string{"email": email, "password": password, "name": "New User", "role": role})
			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			var created auth.User
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
			require.NotEmpty(t, created.ID)
			expectedRole := role
			if expectedRole == "" {
				expectedRole = auth.RoleViewer
			}
			assert.Equal(t, email, created.Email)
			assert.Equal(t, "New User", created.Name)
			assert.Equal(t, expectedRole, created.Role)
			assert.NotContains(t, w.Body.String(), password)

			stored, err := store.GetByEmail(context.Background(), email)
			require.NoError(t, err)
			assert.Equal(t, created.ID, stored.ID)
			assert.Equal(t, expectedRole, stored.Role)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)))
			users, err := store.List(context.Background())
			require.NoError(t, err)
			require.Len(t, users, 1)

			// Newly persisted credentials must work at the public login endpoint,
			// and its returned JWT must authenticate through the same middleware.
			w = registrationRequest(t, handler, http.MethodPost, "/api/auth/login", "",
				map[string]string{"email": email, "password": password})
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			var login struct {
				Token string    `json:"token"`
				User  auth.User `json:"user"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &login))
			require.NotEmpty(t, login.Token)
			assert.Equal(t, created.ID, login.User.ID)
			w = registrationRequest(t, handler, http.MethodGet, "/api/auth/me", "Bearer "+login.Token, nil)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			var authenticated auth.User
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &authenticated))
			assert.Equal(t, created.ID, authenticated.ID)
			assert.Equal(t, expectedRole, authenticated.Role)
		})
	}
}

func TestAuthRegistrationRejectsUnauthorizedCallersWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name          string
		authorization string
		status        int
	}{
		{name: "viewer", authorization: registrationBearer(t, auth.RoleViewer, registrationSecret, time.Hour), status: http.StatusForbidden},
		{name: "editor", authorization: registrationBearer(t, auth.RoleEditor, registrationSecret, time.Hour), status: http.StatusForbidden},
		{name: "missing", status: http.StatusUnauthorized},
		{name: "malformed-token", authorization: "Bearer not-a-jwt", status: http.StatusUnauthorized},
		{name: "wrong-signature", authorization: registrationBearer(t, auth.RoleAdmin, []byte("wrong-secret"), time.Hour), status: http.StatusUnauthorized},
		{name: "expired", authorization: registrationBearer(t, auth.RoleAdmin, registrationSecret, -time.Hour), status: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, store := newRegistrationAPI(t, true, true)
			const email = "must-not-be-created@example.test"
			w := registrationRequest(t, handler, http.MethodPost, "/api/auth/register", tc.authorization,
				map[string]string{"email": email, "password": "registration-test-password", "name": "Denied User", "role": auth.RoleAdmin})
			assert.Equal(t, tc.status, w.Code, w.Body.String())
			users, err := store.List(context.Background())
			require.NoError(t, err)
			assert.Empty(t, users, "rejected registration must not write a user")
			_, err = store.GetByEmail(context.Background(), email)
			assert.Error(t, err, "rejected registration must not persist credentials")
		})
	}
}

func TestAuthRegistrationPublicEndpointsRemainAccessible(t *testing.T) {
	handler, store := newRegistrationAPI(t, true, true)
	const password = "existing-user-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	user := &auth.User{Email: "existing@example.test", Name: "Existing User", Role: auth.RoleAdmin, PasswordHash: string(hash)}
	require.NoError(t, store.Create(context.Background(), user))
	for _, authorization := range []string{"", "Bearer invalid-public-endpoint-token"} {
		w := registrationRequest(t, handler, http.MethodGet, "/api/health", authorization, nil)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		w = registrationRequest(t, handler, http.MethodPost, "/api/auth/login", authorization,
			map[string]string{"email": user.Email, "password": password})
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var login struct {
			Token string    `json:"token"`
			User  auth.User `json:"user"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &login))
		assert.NotEmpty(t, login.Token)
		assert.Equal(t, user.ID, login.User.ID)
	}
}

func TestAuthRegistrationDisabledBehavior(t *testing.T) {
	t.Run("stock-gateway-without-user-store", func(t *testing.T) {
		handler, _ := newRegistrationAPI(t, false, false)
		for _, path := range []string{"/api/auth/register", "/api/auth/login"} {
			w := registrationRequest(t, handler, http.MethodPost, path, "",
				map[string]string{"email": "unused@example.test", "password": "password", "name": "Unused"})
			assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), "auth not enabled")
		}
		for _, path := range []string{"/api/health", "/api/services"} {
			w := registrationRequest(t, handler, http.MethodGet, path, "", nil)
			assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		}
	})
	t.Run("embedded-api-with-user-store", func(t *testing.T) {
		handler, store := newRegistrationAPI(t, false, true)
		const email = "embedded@example.test"
		w := registrationRequest(t, handler, http.MethodPost, "/api/auth/register", "",
			map[string]string{"email": email, "password": "embedded-password", "name": "Embedded User"})
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		stored, err := store.GetByEmail(context.Background(), email)
		require.NoError(t, err)
		assert.Equal(t, auth.RoleViewer, stored.Role)
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("embedded-password")))
	})
}
