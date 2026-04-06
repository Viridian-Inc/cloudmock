package clerk

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware_NoToken_PassesThrough(t *testing.T) {
	// Requests without Authorization header should pass through (local mode).
	mw := NewAuthMiddleware(nil, nil)
	called := false
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/traces", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called for request without token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_PublicPaths_SkipAuth(t *testing.T) {
	mw := NewAuthMiddleware(nil, nil)

	publicPaths := []string{
		"/api/health",
		"/api/version",
		"/api/saas/config",
		"/api/webhooks/clerk",
		"/api/webhooks/stripe",
		"/api/stream",
	}

	for _, path := range publicPaths {
		t.Run(path, func(t *testing.T) {
			called := false
			handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Header.Set("Authorization", "Bearer invalid-token")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if !called {
				t.Errorf("expected handler to be called for public path %s", path)
			}
		})
	}
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	// Use a verifier with a bogus JWKS URL so token validation always fails.
	verifier := NewJWTVerifierWithURL("http://localhost:0/invalid", nil)
	mw := NewAuthMiddleware(verifier, nil)

	called := false
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/traces", nil)
	req.Header.Set("Authorization", "Bearer some.invalid.token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Error("expected handler NOT to be called for invalid token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_NonBearerToken_PassesThrough(t *testing.T) {
	mw := NewAuthMiddleware(nil, nil)
	called := false
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/traces", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called for non-Bearer token")
	}
}
