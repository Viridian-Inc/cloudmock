package stripe

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/saas/tenant"
)

// urlRewriteTransport rewrites request URLs to point to a test server,
// allowing us to intercept calls to the Stripe API constant base URL.
type urlRewriteTransport struct {
	base  string
	inner http.RoundTripper
}

func (t *urlRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.base[len("http://"):]
	return t.inner.RoundTrip(req)
}

func newTestReporter(t *testing.T, store tenant.Store, serverURL string) *UsageReporter {
	t.Helper()
	return &UsageReporter{
		tenants: store,
		apiKey:  "sk_test_key",
		httpClient: &http.Client{
			Transport: &urlRewriteTransport{base: serverURL, inner: http.DefaultTransport},
		},
		logger: slog.Default(),
	}
}

func TestMeter_ReportsUsage(t *testing.T) {
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	tn := &tenant.Tenant{
		ClerkOrgID:       "org_meter_1",
		Name:             "Metered Tenant",
		Slug:             "metered",
		StripeCustomerID: "cus_meter_abc",
		Tier:             "pro",
		Status:           "active",
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	rec := &tenant.UsageRecord{
		TenantID:     tn.ID,
		PeriodStart:  time.Now().Add(-1 * time.Hour),
		PeriodEnd:    time.Now(),
		RequestCount: 500,
	}
	if err := store.RecordUsage(ctx, rec); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	var receivedEvent meterEvent
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &receivedEvent); err != nil {
			t.Errorf("unmarshal meter event: %v", err)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer sk_test_key" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer sk_test_key")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"mevt_1"}`))
	}))
	defer mockServer.Close()

	reporter := newTestReporter(t, store, mockServer.URL)

	if err := reporter.ReportUsage(ctx); err != nil {
		t.Fatalf("ReportUsage: %v", err)
	}

	if receivedEvent.EventName != "api_requests" {
		t.Errorf("EventName = %q, want %q", receivedEvent.EventName, "api_requests")
	}
	if receivedEvent.Payload.StripeCustomerID != "cus_meter_abc" {
		t.Errorf("StripeCustomerID = %q, want %q", receivedEvent.Payload.StripeCustomerID, "cus_meter_abc")
	}
	if receivedEvent.Payload.Value != "500" {
		t.Errorf("Value = %q, want %q", receivedEvent.Payload.Value, "500")
	}

	// Verify record was marked as reported.
	unreported, err := store.GetUnreportedUsage(ctx)
	if err != nil {
		t.Fatalf("GetUnreportedUsage: %v", err)
	}
	if len(unreported) != 0 {
		t.Errorf("expected 0 unreported records, got %d", len(unreported))
	}
}

func TestMeter_SkipsFreeTier(t *testing.T) {
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	// Create a tenant WITHOUT a Stripe customer ID (free tier).
	tn := &tenant.Tenant{
		ClerkOrgID: "org_free_1",
		Name:       "Free Tenant",
		Slug:       "free-tenant",
		Tier:       "free",
		Status:     "active",
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	rec := &tenant.UsageRecord{
		TenantID:     tn.ID,
		PeriodStart:  time.Now().Add(-1 * time.Hour),
		PeriodEnd:    time.Now(),
		RequestCount: 100,
	}
	if err := store.RecordUsage(ctx, rec); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	apiCalled := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"mevt_1"}`))
	}))
	defer mockServer.Close()

	reporter := newTestReporter(t, store, mockServer.URL)

	err := reporter.ReportUsage(ctx)
	if err != nil {
		t.Fatalf("ReportUsage: %v", err)
	}

	if apiCalled {
		t.Error("Stripe API was called for a free-tier tenant with no customer ID")
	}
}

func TestMeter_MarksReported(t *testing.T) {
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	tn := &tenant.Tenant{
		ClerkOrgID:       "org_mark_1",
		Name:             "Mark Test",
		Slug:             "mark-test",
		StripeCustomerID: "cus_mark_abc",
		Tier:             "pro",
		Status:           "active",
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	rec := &tenant.UsageRecord{
		TenantID:     tn.ID,
		PeriodStart:  time.Now().Add(-1 * time.Hour),
		PeriodEnd:    time.Now(),
		RequestCount: 42,
	}
	if err := store.RecordUsage(ctx, rec); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	// Confirm it's unreported.
	unreported, _ := store.GetUnreportedUsage(ctx)
	if len(unreported) != 1 {
		t.Fatalf("expected 1 unreported record, got %d", len(unreported))
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"mevt_1"}`))
	}))
	defer mockServer.Close()

	reporter := newTestReporter(t, store, mockServer.URL)

	if err := reporter.ReportUsage(ctx); err != nil {
		t.Fatalf("ReportUsage: %v", err)
	}

	// Verify it was marked as reported.
	unreported, _ = store.GetUnreportedUsage(ctx)
	if len(unreported) != 0 {
		t.Errorf("expected 0 unreported records after reporting, got %d", len(unreported))
	}
}
