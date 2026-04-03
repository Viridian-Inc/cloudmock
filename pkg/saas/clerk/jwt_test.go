package clerk

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
)

// testRSAKey is a key pair generated once for all JWT tests.
var testRSAKey *rsa.PrivateKey

func init() {
	var err error
	testRSAKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate test RSA key: " + err.Error())
	}
}

// startJWKSServer starts an httptest.Server that serves the public key as a JWKS endpoint.
func startJWKSServer(t *testing.T, kid string, pub *rsa.PublicKey) *httptest.Server {
	t.Helper()
	nBase64 := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eBytes := big.NewInt(int64(pub.E)).Bytes()
	eBase64 := base64.RawURLEncoding.EncodeToString(eBytes)

	jwks := jwksResponse{
		Keys: []jwkKey{
			{
				KID: kid,
				KTY: "RSA",
				ALG: "RS256",
				Use: "sig",
				N:   nBase64,
				E:   eBase64,
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// createTestJWT creates a signed RS256 JWT with the given claims and kid header.
func createTestJWT(t *testing.T, claims ClerkClaims, kid string, key *rsa.PrivateKey) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	return signed
}

func TestJWT_ValidToken(t *testing.T) {
	kid := "test-key-1"
	jwksSrv := startJWKSServer(t, kid, &testRSAKey.PublicKey)
	verifier := NewJWTVerifierWithURL(jwksSrv.URL, nil)

	claims := ClerkClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://clerk.example.com",
			Subject:   "user_abc",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		OrgID:   "org_123",
		OrgSlug: "acme",
		OrgRole: "admin",
		Email:   "alice@example.com",
	}

	tokenStr := createTestJWT(t, claims, kid, testRSAKey)
	got, err := verifier.VerifyToken(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}

	if got.OrgID != "org_123" {
		t.Errorf("OrgID = %q, want %q", got.OrgID, "org_123")
	}
	if got.OrgSlug != "acme" {
		t.Errorf("OrgSlug = %q, want %q", got.OrgSlug, "acme")
	}
	if got.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "alice@example.com")
	}
	if got.Subject != "user_abc" {
		t.Errorf("Subject = %q, want %q", got.Subject, "user_abc")
	}
}

func TestJWT_ExpiredToken(t *testing.T) {
	kid := "test-key-1"
	jwksSrv := startJWKSServer(t, kid, &testRSAKey.PublicKey)
	verifier := NewJWTVerifierWithURL(jwksSrv.URL, nil)

	claims := ClerkClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://clerk.example.com",
			Subject:   "user_abc",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // expired
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
		OrgID: "org_123",
		Email: "alice@example.com",
	}

	tokenStr := createTestJWT(t, claims, kid, testRSAKey)
	_, err := verifier.VerifyToken(context.Background(), tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestJWT_WrongSignature(t *testing.T) {
	kid := "test-key-1"
	jwksSrv := startJWKSServer(t, kid, &testRSAKey.PublicKey)
	verifier := NewJWTVerifierWithURL(jwksSrv.URL, nil)

	// Create a token signed with a DIFFERENT key.
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}

	claims := ClerkClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://clerk.example.com",
			Subject:   "user_abc",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		OrgID: "org_123",
	}

	tokenStr := createTestJWT(t, claims, kid, otherKey)
	_, err = verifier.VerifyToken(context.Background(), tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong signature, got nil")
	}
}

func TestJWT_ExtractClaims(t *testing.T) {
	kid := "test-key-1"
	jwksSrv := startJWKSServer(t, kid, &testRSAKey.PublicKey)
	verifier := NewJWTVerifierWithURL(jwksSrv.URL, nil)

	claims := ClerkClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://clerk.example.com",
			Subject:   "user_xyz",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		OrgID:   "org_456",
		OrgSlug: "mega-corp",
		OrgRole: "member",
		Email:   "bob@mega.com",
	}

	tokenStr := createTestJWT(t, claims, kid, testRSAKey)

	// Test VerifyToken extracts all claims.
	got, err := verifier.VerifyToken(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if got.OrgID != "org_456" {
		t.Errorf("OrgID = %q, want %q", got.OrgID, "org_456")
	}
	if got.OrgSlug != "mega-corp" {
		t.Errorf("OrgSlug = %q, want %q", got.OrgSlug, "mega-corp")
	}
	if got.OrgRole != "member" {
		t.Errorf("OrgRole = %q, want %q", got.OrgRole, "member")
	}
	if got.Email != "bob@mega.com" {
		t.Errorf("Email = %q, want %q", got.Email, "bob@mega.com")
	}

	// Test TenantID helper.
	tenantID, err := verifier.TenantID(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("TenantID: %v", err)
	}
	if tenantID != "org_456" {
		t.Errorf("TenantID = %q, want %q", tenantID, "org_456")
	}
}
