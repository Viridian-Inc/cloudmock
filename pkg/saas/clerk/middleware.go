package clerk

import (
	"log/slog"
	"net/http"
	"strings"
)

// AuthMiddleware extracts Clerk JWT tokens from the Authorization header,
// verifies them, and sets the X-Tenant-ID header for downstream middleware
// (quota enforcement, tenant-scoped data isolation).
//
// Requests without a Bearer token are passed through — they may be
// unauthenticated endpoints (health, webhooks) or local-mode requests.
type AuthMiddleware struct {
	verifier *JWTVerifier
	logger   *slog.Logger
}

// NewAuthMiddleware creates a Clerk auth middleware.
func NewAuthMiddleware(verifier *JWTVerifier, logger *slog.Logger) *AuthMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthMiddleware{verifier: verifier, logger: logger}
}

// Handler wraps next with Clerk JWT verification. It extracts the org_id
// claim from verified tokens and sets it as the X-Tenant-ID header so that
// quota and tenant-scoping middleware can identify the tenant.
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health, webhook, and public endpoints.
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No token — allow through (local mode or unauthenticated).
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			// Not a Bearer token — pass through.
			next.ServeHTTP(w, r)
			return
		}

		claims, err := m.verifier.VerifyToken(r.Context(), token)
		if err != nil {
			m.logger.Debug("clerk auth: invalid token", "error", err)
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Set tenant ID from org_id for downstream middleware.
		if claims.OrgID != "" {
			r.Header.Set("X-Tenant-ID", claims.OrgID)
		}

		// Set user context headers for audit logging.
		if claims.Email != "" {
			r.Header.Set("X-User-Email", claims.Email)
		}
		if claims.Subject != "" {
			r.Header.Set("X-User-ID", claims.Subject)
		}
		if claims.OrgRole != "" {
			r.Header.Set("X-Org-Role", claims.OrgRole)
		}

		next.ServeHTTP(w, r)
	})
}

// isPublicPath returns true for endpoints that should not require auth.
func isPublicPath(path string) bool {
	switch {
	case path == "/api/health":
		return true
	case path == "/api/version":
		return true
	case path == "/api/saas/config":
		return true
	case strings.HasPrefix(path, "/api/webhooks/"):
		return true
	case path == "/api/stream":
		return true
	default:
		return false
	}
}
