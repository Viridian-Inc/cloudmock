package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var testSecret = []byte("test-secret-key")

func TestMiddleware_ValidJWT(t *testing.T) {
	user := &User{ID: "u1", Email: "a@b.com", Role: RoleAdmin}
	token, err := GenerateToken(user, testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	var gotUser *User
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(testSecret)(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if gotUser == nil {
		t.Fatal("expected user in context")
	}
	if gotUser.ID != "u1" || gotUser.Email != "a@b.com" || gotUser.Role != RoleAdmin {
		t.Fatalf("unexpected user: %+v", gotUser)
	}
}

func TestMiddleware_MissingToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	handler := Middleware(testSecret)(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	handler := Middleware(testSecret)(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddleware_ExpiredToken(t *testing.T) {
	user := &User{ID: "u1", Email: "a@b.com", Role: RoleViewer}
	token, err := GenerateToken(user, testSecret, -time.Hour) // already expired
	if err != nil {
		t.Fatal(err)
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	handler := Middleware(testSecret)(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddleware_HealthSkipsAuth(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(testSecret)(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !called {
		t.Fatal("expected inner handler to be called for health endpoint")
	}
}

func TestRequireRole_AdminPass(t *testing.T) {
	user := &User{ID: "u1", Email: "a@b.com", Role: RoleAdmin}
	token, err := GenerateToken(user, testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(testSecret)(RequireRole(RoleAdmin)(inner))
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !called {
		t.Fatal("expected handler to be called for admin")
	}
}

func TestRequireRole_ViewerForbidden(t *testing.T) {
	user := &User{ID: "u2", Email: "v@b.com", Role: RoleViewer}
	token, err := GenerateToken(user, testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for viewer")
	})

	handler := Middleware(testSecret)(RequireRole(RoleAdmin)(inner))
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestGenerateToken_Roundtrip(t *testing.T) {
	user := &User{
		ID:       "u1",
		Email:    "test@example.com",
		Role:     RoleEditor,
		TenantID: "t1",
	}
	token, err := GenerateToken(user, testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}
