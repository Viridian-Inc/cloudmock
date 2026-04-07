package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Viridian-Inc/cloudmock/pkg/platform/model"
)

type contextKey string

const authContextKey contextKey = "auth"

// APIKeyLookup is the interface for looking up API keys by plaintext.
type APIKeyLookup interface {
	GetByPlaintext(ctx context.Context, plaintext string) (*model.APIKey, error)
	TouchLastUsed(ctx context.Context, id string) error
}

// Auth is the authentication middleware.
type Auth struct {
	clerk *ClerkVerifier
	keys  APIKeyLookup
}

// NewAuth creates a new Auth middleware.
func NewAuth(clerk *ClerkVerifier, keys APIKeyLookup) *Auth {
	return &Auth{clerk: clerk, keys: keys}
}

// AuthFromContext extracts the AuthContext from ctx. Returns nil if not set.
func AuthFromContext(ctx context.Context) *model.AuthContext {
	v, _ := ctx.Value(authContextKey).(*model.AuthContext)
	return v
}

// WithAuthContext sets the AuthContext in ctx. Used by handlers and tests.
func WithAuthContext(ctx context.Context, ac *model.AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, ac)
}

// Handler returns an http.Handler that authenticates requests via API key or Clerk JWT.
func (a *Auth) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Try X-Api-Key header first.
		if apiKey := r.Header.Get("X-Api-Key"); apiKey != "" {
			key, err := a.keys.GetByPlaintext(r.Context(), apiKey)
			if err != nil {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}
			ac := &model.AuthContext{
				TenantID:  key.TenantID,
				ActorID:   key.ID,
				ActorType: "api_key",
				Role:      key.Role,
				AppID:     key.AppID,
			}
			// Fire-and-forget: update last used timestamp.
			go func() {
				_ = a.keys.TouchLastUsed(context.Background(), key.ID)
			}()
			next.ServeHTTP(w, r.WithContext(WithAuthContext(r.Context(), ac)))
			return
		}

		// 2. Try Authorization: Bearer <jwt>.
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := a.clerk.Verify(r.Context(), tokenString)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			role := mapClerkRole(claims.OrgRole)
			ac := &model.AuthContext{
				TenantID:  claims.OrgID,
				ActorID:   claims.Subject,
				ActorType: "user",
				Role:      role,
				AppID:     "",
			}
			next.ServeHTTP(w, r.WithContext(WithAuthContext(r.Context(), ac)))
			return
		}

		// 3. No credentials provided.
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// mapClerkRole maps Clerk org roles to platform roles.
func mapClerkRole(orgRole string) string {
	switch orgRole {
	case "org:admin":
		return "admin"
	case "org:developer":
		return "developer"
	case "org:viewer":
		return "viewer"
	default:
		return "viewer"
	}
}

// -----------------------------------------------------------------------------
// ClerkVerifier – validates Clerk-issued JWTs using JWKS.
// -----------------------------------------------------------------------------

type clerkClaims struct {
	jwt.RegisteredClaims
	OrgID   string `json:"org_id"`
	OrgSlug string `json:"org_slug"`
	OrgRole string `json:"org_role"`
}

// jwksKey is a single key entry from a JWKS response.
type jwksKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

// ClerkVerifier verifies Clerk JWTs by fetching and caching JWKS keys.
type ClerkVerifier struct {
	jwksURL  string
	mu       sync.RWMutex
	keys     map[string]*rsa.PublicKey
	fetchedAt time.Time
	cacheTTL time.Duration
}

// NewClerkVerifier creates a ClerkVerifier that fetches keys from jwksURL.
func NewClerkVerifier(jwksURL string) *ClerkVerifier {
	return &ClerkVerifier{
		jwksURL:  jwksURL,
		keys:     make(map[string]*rsa.PublicKey),
		cacheTTL: time.Hour,
	}
}

// Verify parses and validates a JWT string, returning the claims on success.
func (v *ClerkVerifier) Verify(ctx context.Context, tokenString string) (*clerkClaims, error) {
	claims := &clerkClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		return v.getKey(ctx, kid)
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token invalid")
	}
	return claims, nil
}

// getKey returns the RSA public key for the given kid, fetching JWKS if needed.
func (v *ClerkVerifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Fast path: read lock.
	v.mu.RLock()
	key, ok := v.keys[kid]
	expired := time.Since(v.fetchedAt) > v.cacheTTL
	v.mu.RUnlock()

	if ok && !expired {
		return key, nil
	}

	// Slow path: refresh JWKS.
	if err := v.fetchJWKS(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("key not found for kid %q", kid)
	}
	return key, nil
}

// fetchJWKS fetches the JWKS endpoint and updates the key cache.
func (v *ClerkVerifier) fetchJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("build jwks request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := jwkToRSA(k)
		if err != nil {
			return fmt.Errorf("parse jwk kid=%s: %w", k.Kid, err)
		}
		newKeys[k.Kid] = pub
	}

	v.mu.Lock()
	v.keys = newKeys
	v.fetchedAt = time.Now()
	v.mu.Unlock()

	return nil
}

// jwkToRSA converts a JWKS key entry to an *rsa.PublicKey.
func jwkToRSA(k jwksKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}
