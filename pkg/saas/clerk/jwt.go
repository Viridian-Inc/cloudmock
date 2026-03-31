package clerk

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSCacheDuration is how long fetched JWKS keys are cached.
const JWKSCacheDuration = 1 * time.Hour

// ClerkClaims are the JWT claims extracted from a Clerk session token.
type ClerkClaims struct {
	jwt.RegisteredClaims
	OrgID   string `json:"org_id"`
	OrgSlug string `json:"org_slug"`
	OrgRole string `json:"org_role"`
	Email   string `json:"email"`
}

// JWTVerifier verifies Clerk-issued JWTs using the JWKS endpoint.
type JWTVerifier struct {
	jwksURL    string
	httpClient *http.Client
	logger     *slog.Logger

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// jwksResponse is the JSON structure returned by the JWKS endpoint.
type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

// jwkKey is a single key from the JWKS endpoint.
type jwkKey struct {
	KID string `json:"kid"`
	KTY string `json:"kty"`
	ALG string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewJWTVerifier creates a verifier that fetches keys from the Clerk JWKS endpoint.
// clerkDomain should be the Clerk frontend API domain (e.g. "clerk.example.com"
// or the full Clerk instance domain like "abc123.clerk.accounts.dev").
func NewJWTVerifier(clerkDomain string, logger *slog.Logger) *JWTVerifier {
	if logger == nil {
		logger = slog.Default()
	}

	// Normalize domain — strip protocol if present.
	domain := clerkDomain
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")

	return &JWTVerifier{
		jwksURL:    "https://" + domain + "/.well-known/jwks.json",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		keys:       make(map[string]*rsa.PublicKey),
	}
}

// NewJWTVerifierWithURL creates a verifier with a custom JWKS URL (useful for testing).
func NewJWTVerifierWithURL(jwksURL string, logger *slog.Logger) *JWTVerifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &JWTVerifier{
		jwksURL:    jwksURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		keys:       make(map[string]*rsa.PublicKey),
	}
}

// VerifyToken verifies a Clerk JWT and returns the parsed claims.
// The token is expected to be an RS256-signed JWT.
func (v *JWTVerifier) VerifyToken(ctx context.Context, tokenString string) (*ClerkClaims, error) {
	claims := &ClerkClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is RSA.
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("missing kid in token header")
		}

		key, err := v.getKey(ctx, kid)
		if err != nil {
			return nil, fmt.Errorf("get signing key: %w", err)
		}
		return key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// TenantID extracts the org_id from a verified Clerk token, suitable
// for use as a tenant identifier.
func (v *JWTVerifier) TenantID(ctx context.Context, tokenString string) (string, error) {
	claims, err := v.VerifyToken(ctx, tokenString)
	if err != nil {
		return "", err
	}
	if claims.OrgID == "" {
		return "", fmt.Errorf("no org_id in token claims")
	}
	return claims.OrgID, nil
}

// getKey returns the RSA public key for the given kid, fetching from JWKS if needed.
func (v *JWTVerifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Try cache first.
	v.mu.RLock()
	if key, ok := v.keys[kid]; ok && time.Since(v.fetchedAt) < JWKSCacheDuration {
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	// Fetch fresh keys.
	if err := v.fetchJWKS(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %q not found in JWKS", kid)
	}
	return key, nil
}

// fetchJWKS fetches and parses the JWKS endpoint, updating the key cache.
func (v *JWTVerifier) fetchJWKS(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Double-check: another goroutine may have fetched while we waited for the lock.
	if time.Since(v.fetchedAt) < JWKSCacheDuration && len(v.keys) > 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read JWKS response: %w", err)
	}

	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("parse JWKS: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.KTY != "RSA" {
			continue
		}

		pubKey, err := parseRSAPublicKey(k)
		if err != nil {
			v.logger.Warn("clerk jwt: skipping malformed JWK",
				"kid", k.KID,
				"error", err,
			)
			continue
		}
		newKeys[k.KID] = pubKey
	}

	if len(newKeys) == 0 {
		return fmt.Errorf("no usable RSA keys in JWKS response")
	}

	v.keys = newKeys
	v.fetchedAt = time.Now()
	v.logger.Debug("clerk jwt: refreshed JWKS cache", "key_count", len(newKeys))
	return nil
}

// parseRSAPublicKey converts a JWK key to an *rsa.PublicKey.
func parseRSAPublicKey(k jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}
