package quota

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/saas/tenant"
)

// okHandler is a simple handler that writes 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
})

func TestQuota_UnderLimit(t *testing.T) {
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	tn := &tenant.Tenant{
		ClerkOrgID:   "org_1",
		Name:         "Under Limit",
		Slug:         "under-limit",
		Tier:         "pro",
		Status:       "active",
		RequestCount: 10,
		RequestLimit: 100000,
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	mw := New(store)
	handler := mw.Handler(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Tenant-ID", tn.ID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "ok")
	}

	// No warning header when well under limit.
	if w := rr.Header().Get("X-CloudMock-Usage-Warning"); w != "" {
		t.Errorf("unexpected usage warning header: %q", w)
	}
}

func TestQuota_At80Percent(t *testing.T) {
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	tn := &tenant.Tenant{
		ClerkOrgID:   "org_2",
		Name:         "At 80 Percent",
		Slug:         "at-80",
		Tier:         "pro",
		Status:       "active",
		RequestCount: 850, // 85% of 1000
		RequestLimit: 1000,
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	mw := New(store)
	handler := mw.Handler(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Tenant-ID", tn.ID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	warning := rr.Header().Get("X-CloudMock-Usage-Warning")
	if warning == "" {
		t.Fatal("expected X-CloudMock-Usage-Warning header, got empty")
	}
	if !strings.Contains(warning, "85%") {
		t.Errorf("warning = %q, expected to contain '85%%'", warning)
	}
}

func TestQuota_AtLimit(t *testing.T) {
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	tn := &tenant.Tenant{
		ClerkOrgID:   "org_3",
		Name:         "At Limit",
		Slug:         "at-limit",
		Tier:         "free",
		Status:       "active",
		RequestCount: 1000, // Exactly at the limit
		RequestLimit: 1000,
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	mw := New(store)
	handler := mw.Handler(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Tenant-ID", tn.ID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTooManyRequests)
	}

	// Body should include quota info.
	if !strings.Contains(rr.Body.String(), "quota exceeded") {
		t.Errorf("body = %q, expected to contain 'quota exceeded'", rr.Body.String())
	}

	retryAfter := rr.Header().Get("Retry-After")
	if retryAfter != "3600" {
		t.Errorf("Retry-After = %q, want %q", retryAfter, "3600")
	}
}

func TestQuota_MissingTenantID(t *testing.T) {
	store := tenant.NewMemoryStore()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	mw := New(store)
	handler := mw.Handler(next)

	// No X-Tenant-ID header.
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !nextCalled {
		t.Error("next handler was not called for request without tenant ID")
	}
}

func TestQuota_IncrementCount(t *testing.T) {
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	tn := &tenant.Tenant{
		ClerkOrgID:   "org_5",
		Name:         "Increment",
		Slug:         "increment",
		Tier:         "pro",
		Status:       "active",
		RequestCount: 0,
		RequestLimit: 100000,
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	mw := New(store)
	handler := mw.Handler(okHandler)

	// Make 3 requests.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("X-Tenant-ID", tn.ID)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, rr.Code, http.StatusOK)
		}
	}

	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RequestCount != 3 {
		t.Errorf("RequestCount = %d, want 3 after 3 requests", got.RequestCount)
	}
}
