package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/platform/middleware"
	"github.com/neureaux/cloudmock/pkg/platform/model"
)

// ---------------------------------------------------------------------------
// Mock usage counter
// ---------------------------------------------------------------------------

type mockUsageCounter struct {
	count int64
	err   error
}

func (m *mockUsageCounter) GetCurrentPeriodCount(_ context.Context, _ string) (int64, error) {
	return m.count, m.err
}

func (m *mockUsageCounter) IncrementRequestCount(_ context.Context, _, _ string) error {
	return nil
}

// ---------------------------------------------------------------------------
// Mock tenant lookup
// ---------------------------------------------------------------------------

type mockTenantLookup struct {
	hasPM bool
	err   error
}

func (m *mockTenantLookup) HasPaymentMethod(_ context.Context, _ string) (bool, error) {
	return m.hasPM, m.err
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func requestWithAuth(tenantID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ac := &model.AuthContext{
		TenantID:  tenantID,
		ActorID:   "actor-1",
		ActorType: "api_key",
		Role:      "developer",
		AppID:     "app-1",
	}
	return req.WithContext(middleware.WithAuthContext(req.Context(), ac))
}

func TestQuotaMiddleware(t *testing.T) {
	t.Run("under limit passes with 200", func(t *testing.T) {
		usage := &mockUsageCounter{count: 500}
		tenants := &mockTenantLookup{hasPM: false}
		q := middleware.NewQuota(usage, tenants)

		rr := httptest.NewRecorder()
		q.Handler(okHandler()).ServeHTTP(rr, requestWithAuth("tenant-1"))

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("over limit without payment method returns 429", func(t *testing.T) {
		usage := &mockUsageCounter{count: 1000}
		tenants := &mockTenantLookup{hasPM: false}
		q := middleware.NewQuota(usage, tenants)

		rr := httptest.NewRecorder()
		q.Handler(okHandler()).ServeHTTP(rr, requestWithAuth("tenant-1"))

		if rr.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429, got %d", rr.Code)
		}
	})

	t.Run("over limit with payment method passes with 200", func(t *testing.T) {
		usage := &mockUsageCounter{count: 5000}
		tenants := &mockTenantLookup{hasPM: true}
		q := middleware.NewQuota(usage, tenants)

		rr := httptest.NewRecorder()
		q.Handler(okHandler()).ServeHTTP(rr, requestWithAuth("tenant-1"))

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("no auth context passes through without quota check", func(t *testing.T) {
		usage := &mockUsageCounter{count: 99999}
		tenants := &mockTenantLookup{hasPM: false}
		q := middleware.NewQuota(usage, tenants)

		req := httptest.NewRequest(http.MethodGet, "/", nil) // no auth context
		rr := httptest.NewRecorder()
		q.Handler(okHandler()).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})
}
