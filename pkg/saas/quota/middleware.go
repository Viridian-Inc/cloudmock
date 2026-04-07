// Package quota provides HTTP middleware that enforces per-tenant
// request quotas for the hosted SaaS tier.
package quota

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/saas/tenant"
)

// Middleware enforces request quotas per tenant. It reads the tenant
// ID from the X-Tenant-ID request header (set by an upstream auth
// middleware), looks up the tenant, and rejects requests that exceed
// the tenant's request_limit with a 429 status code.
//
// When usage reaches 80% of the limit an X-CloudMock-Usage-Warning
// header is added to the response.
type Middleware struct {
	tenants tenant.Store
	logger  *slog.Logger
}

// New creates a quota enforcement middleware.
func New(tenants tenant.Store) *Middleware {
	return &Middleware{
		tenants: tenants,
		logger:  slog.Default(),
	}
}

// Handler returns an http.Handler that wraps next with quota enforcement.
//
// For each request the middleware:
//  1. Extracts the tenant_id from the X-Tenant-ID header.
//  2. Looks up the tenant in the store.
//  3. If request_limit > 0 and request_count >= request_limit, returns 429.
//  4. If request_count >= request_limit * 0.8, adds X-CloudMock-Usage-Warning header.
//  5. Increments the tenant's request_count.
//  6. Calls next.
//
// Requests without a tenant ID header are passed through without quota
// enforcement (they may be unauthenticated admin/health endpoints).
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			// No tenant context — skip quota enforcement.
			next.ServeHTTP(w, r)
			return
		}

		t, err := m.tenants.Get(r.Context(), tenantID)
		if err != nil {
			// Tenant not found — let the request through (auth middleware
			// will handle unknown tenants separately).
			m.logger.Debug("quota: tenant not found, skipping enforcement",
				"tenant_id", tenantID,
				"error", err,
			)
			next.ServeHTTP(w, r)
			return
		}

		// Enforce quota if a limit is configured.
		if t.RequestLimit > 0 {
			if t.RequestCount >= t.RequestLimit {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-CloudMock-Usage-Warning", "quota_exceeded")
				w.Header().Set("Retry-After", "3600")
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintf(w, `{"error":"quota exceeded","request_count":%d,"request_limit":%d,"tier":"%s"}`,
					t.RequestCount, t.RequestLimit, t.Tier)

				m.logger.Warn("quota: request rejected, limit exceeded",
					"tenant_id", tenantID,
					"request_count", t.RequestCount,
					"request_limit", t.RequestLimit,
					"tier", t.Tier,
				)
				return
			}

			// Add warning header at 80% usage.
			threshold := int64(float64(t.RequestLimit) * 0.8)
			if t.RequestCount >= threshold {
				pct := float64(t.RequestCount) / float64(t.RequestLimit) * 100
				w.Header().Set("X-CloudMock-Usage-Warning",
					fmt.Sprintf("%.0f%% of quota used (%d/%d)", pct, t.RequestCount, t.RequestLimit))
			}
		}

		// Increment the request count.
		if err := m.tenants.IncrementRequestCount(r.Context(), tenantID); err != nil {
			m.logger.Error("quota: failed to increment request count",
				"tenant_id", tenantID,
				"error", err,
			)
			// Don't block the request on a counter failure.
		}

		next.ServeHTTP(w, r)
	})
}
