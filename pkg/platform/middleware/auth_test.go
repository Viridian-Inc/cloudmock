package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Viridian-Inc/cloudmock/pkg/platform/middleware"
	"github.com/Viridian-Inc/cloudmock/pkg/platform/model"
)

// ---------------------------------------------------------------------------
// Mock API key store
// ---------------------------------------------------------------------------

type mockKeyStore struct {
	key *model.APIKey
	err error
}

func (m *mockKeyStore) GetByPlaintext(_ context.Context, _ string) (*model.APIKey, error) {
	return m.key, m.err
}

func (m *mockKeyStore) TouchLastUsed(_ context.Context, _ string) error {
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// captureAuthHandler returns an http.Handler that writes 200 and stores the
// AuthContext from the request for inspection.
func captureAuthHandler(out **model.AuthContext) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*out = middleware.AuthFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

// ---------------------------------------------------------------------------
// TestAPIKeyAuth
// ---------------------------------------------------------------------------

func TestAPIKeyAuth(t *testing.T) {
	validKey := &model.APIKey{
		ID:       "key-1",
		TenantID: "tenant-1",
		AppID:    "app-1",
		Role:     "developer",
	}

	t.Run("valid key returns 200 with correct auth context", func(t *testing.T) {
		store := &mockKeyStore{key: validKey}
		var captured *model.AuthContext
		handler := middleware.NewAuth(nil, store).Handler(captureAuthHandler(&captured))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Api-Key", "cm_live_abc123")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if captured == nil {
			t.Fatal("expected auth context, got nil")
		}
		if captured.ActorType != "api_key" {
			t.Errorf("expected ActorType=api_key, got %q", captured.ActorType)
		}
		if captured.TenantID != "tenant-1" {
			t.Errorf("expected TenantID=tenant-1, got %q", captured.TenantID)
		}
		if captured.AppID != "app-1" {
			t.Errorf("expected AppID=app-1, got %q", captured.AppID)
		}
		if captured.Role != "developer" {
			t.Errorf("expected Role=developer, got %q", captured.Role)
		}
	})

	t.Run("missing auth returns 401", func(t *testing.T) {
		store := &mockKeyStore{key: validKey}
		var captured *model.AuthContext
		handler := middleware.NewAuth(nil, store).Handler(captureAuthHandler(&captured))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("invalid key returns 401", func(t *testing.T) {
		store := &mockKeyStore{err: errNotFound}
		var captured *model.AuthContext
		handler := middleware.NewAuth(nil, store).Handler(captureAuthHandler(&captured))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Api-Key", "cm_live_bad")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
	})
}

// sentinel error used by mock store
var errNotFound = &notFoundError{}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "not found" }

// ---------------------------------------------------------------------------
// TestClerkJWTAuth
// ---------------------------------------------------------------------------

func TestClerkJWTAuth(t *testing.T) {
	// Generate an RSA key pair for the test.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	const kid = "test-kid-1"

	// Serve a JWKS endpoint via httptest.
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nBytes := publicKey.N.Bytes()
		eVal := big.NewInt(int64(publicKey.E))
		eBytes := eVal.Bytes()

		jwks := map[string]any{
			"keys": []map[string]any{
				{
					"kid": kid,
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"n":   base64.RawURLEncoding.EncodeToString(nBytes),
					"e":   base64.RawURLEncoding.EncodeToString(eBytes),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	// Helper: sign a JWT with the test private key.
	signToken := func(claims jwt.Claims) string {
		t.Helper()
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tok.Header["kid"] = kid
		signed, err := tok.SignedString(privateKey)
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		return signed
	}

	verifier := middleware.NewClerkVerifier(jwksServer.URL)
	// mockKeyStore with no keys – JWT path does not use it.
	store := &mockKeyStore{}

	t.Run("valid JWT returns 200 with correct auth context", func(t *testing.T) {
		type clerkTestClaims struct {
			jwt.RegisteredClaims
			OrgID   string `json:"org_id"`
			OrgSlug string `json:"org_slug"`
			OrgRole string `json:"org_role"`
		}

		claims := clerkTestClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-abc",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			OrgID:   "org_123",
			OrgSlug: "my-org",
			OrgRole: "org:admin",
		}
		tokenStr := signToken(claims)

		var captured *model.AuthContext
		handler := middleware.NewAuth(verifier, store).Handler(captureAuthHandler(&captured))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
		}
		if captured == nil {
			t.Fatal("expected auth context, got nil")
		}
		if captured.ActorType != "user" {
			t.Errorf("expected ActorType=user, got %q", captured.ActorType)
		}
		if captured.ActorID != "user-abc" {
			t.Errorf("expected ActorID=user-abc, got %q", captured.ActorID)
		}
		if captured.TenantID != "org_123" {
			t.Errorf("expected TenantID=org_123, got %q", captured.TenantID)
		}
		if captured.Role != "admin" {
			t.Errorf("expected Role=admin, got %q", captured.Role)
		}
	})

	t.Run("expired JWT returns 401", func(t *testing.T) {
		type clerkTestClaims struct {
			jwt.RegisteredClaims
			OrgID   string `json:"org_id"`
			OrgSlug string `json:"org_slug"`
			OrgRole string `json:"org_role"`
		}

		claims := clerkTestClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-abc",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // expired
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
			OrgID:   "org_123",
			OrgSlug: "my-org",
			OrgRole: "org:developer",
		}
		tokenStr := signToken(claims)

		var captured *model.AuthContext
		handler := middleware.NewAuth(verifier, store).Handler(captureAuthHandler(&captured))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for expired token, got %d", rr.Code)
		}
	})
}
