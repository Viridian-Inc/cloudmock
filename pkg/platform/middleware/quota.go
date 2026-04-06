package middleware

import (
	"context"
	"encoding/json"
	"net/http"
)

const freeRequestLimit = 1000

// UsageCounter reads and increments request counts per tenant.
type UsageCounter interface {
	GetCurrentPeriodCount(ctx context.Context, tenantID string) (int64, error)
	IncrementRequestCount(ctx context.Context, tenantID, appID string) error
}

// QuotaTenantLookup checks whether a tenant has a payment method on file.
type QuotaTenantLookup interface {
	HasPaymentMethod(ctx context.Context, tenantID string) (bool, error)
}

// Quota is the request-quota enforcement middleware.
type Quota struct {
	usage   UsageCounter
	tenants QuotaTenantLookup
}

// NewQuota creates a Quota middleware.
func NewQuota(usage UsageCounter, tenants QuotaTenantLookup) *Quota {
	return &Quota{usage: usage, tenants: tenants}
}

// Handler returns an http.Handler that enforces per-tenant request quotas.
// Requests with no auth context are passed through without quota checks.
func (q *Quota) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ac := AuthFromContext(r.Context())
		if ac == nil {
			next.ServeHTTP(w, r)
			return
		}

		count, err := q.usage.GetCurrentPeriodCount(r.Context(), ac.TenantID)
		if err != nil {
			http.Error(w, "failed to check quota", http.StatusInternalServerError)
			return
		}

		if count >= freeRequestLimit {
			hasPM, err := q.tenants.HasPaymentMethod(r.Context(), ac.TenantID)
			if err != nil {
				http.Error(w, "failed to check payment method", http.StatusInternalServerError)
				return
			}
			if !hasPM {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "free request limit exceeded; add a payment method to continue",
				})
				return
			}
			// Has payment method – Stripe meters it; pass through.
		}

		// Increment counter fire-and-forget.
		go func() {
			_ = q.usage.IncrementRequestCount(context.Background(), ac.TenantID, ac.AppID)
		}()

		next.ServeHTTP(w, r)
	})
}
