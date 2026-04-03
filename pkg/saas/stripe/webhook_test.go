package stripe

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/saas/tenant"
)

const testStripeSecret = "whsec_test_stripe_secret_key"

// signStripe generates a valid Stripe-Signature header for the given body.
func signStripe(body []byte, secret string) string {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := ts + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%s,v1=%s", ts, sig)
}

// newStripeRequest creates a signed POST request for the Stripe webhook.
func newStripeRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", signStripe(body, testStripeSecret))
	return req
}

// createTestTenant creates a tenant in the store for Stripe tests.
func createTestTenant(t *testing.T, store *tenant.MemoryStore, id, customerID, tier string) *tenant.Tenant {
	t.Helper()
	tn := &tenant.Tenant{
		ID:               id,
		ClerkOrgID:       "org_" + id,
		Name:             "Tenant " + id,
		Slug:             "tenant-" + id,
		StripeCustomerID: customerID,
		Tier:             tier,
		Status:           "active",
		RequestLimit:     1000,
	}
	if err := store.Create(context.Background(), tn); err != nil {
		t.Fatalf("create test tenant: %v", err)
	}
	return tn
}

func TestStripe_CheckoutCompleted(t *testing.T) {
	store := tenant.NewMemoryStore()
	tn := createTestTenant(t, store, "tn-1", "", "free")
	handler := NewWebhookHandler(store, testStripeSecret, nil)

	sess := map[string]interface{}{
		"id":           "cs_test_123",
		"customer":     "cus_abc",
		"subscription": "sub_xyz",
		"mode":         "subscription",
		"metadata": map[string]string{
			"tenant_id": tn.ID,
			"tier":      "pro",
		},
	}
	event := stripeEvent{
		ID:   "evt_1",
		Type: "checkout.session.completed",
		Data: stripeEventData{Object: mustMarshal(t, sess)},
	}
	body := mustMarshal(t, event)

	req := newStripeRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	got, err := store.Get(context.Background(), tn.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.StripeCustomerID != "cus_abc" {
		t.Errorf("StripeCustomerID = %q, want %q", got.StripeCustomerID, "cus_abc")
	}
	if got.StripeSubscriptionID != "sub_xyz" {
		t.Errorf("StripeSubscriptionID = %q, want %q", got.StripeSubscriptionID, "sub_xyz")
	}
	if got.Tier != "pro" {
		t.Errorf("Tier = %q, want %q", got.Tier, "pro")
	}
	if got.RequestLimit != 100_000 {
		t.Errorf("RequestLimit = %d, want %d", got.RequestLimit, 100_000)
	}
}

func TestStripe_InvoicePaid(t *testing.T) {
	store := tenant.NewMemoryStore()
	tn := createTestTenant(t, store, "tn-2", "cus_invoice", "pro")

	// Simulate some usage.
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		if err := store.IncrementRequestCount(ctx, tn.ID); err != nil {
			t.Fatalf("IncrementRequestCount: %v", err)
		}
	}
	before, _ := store.Get(ctx, tn.ID)
	if before.RequestCount != 50 {
		t.Fatalf("pre-check: RequestCount = %d, want 50", before.RequestCount)
	}

	handler := NewWebhookHandler(store, testStripeSecret, nil)

	inv := map[string]interface{}{
		"id":           "in_test_123",
		"customer":     "cus_invoice",
		"subscription": "sub_xyz",
		"status":       "paid",
	}
	event := stripeEvent{
		ID:   "evt_2",
		Type: "invoice.paid",
		Data: stripeEventData{Object: mustMarshal(t, inv)},
	}
	body := mustMarshal(t, event)

	req := newStripeRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RequestCount != 0 {
		t.Errorf("RequestCount = %d, want 0 (should be reset)", got.RequestCount)
	}
	if got.Status != "active" {
		t.Errorf("Status = %q, want %q", got.Status, "active")
	}
}

func TestStripe_SubscriptionUpdated_Active(t *testing.T) {
	store := tenant.NewMemoryStore()
	tn := createTestTenant(t, store, "tn-3", "cus_sub_active", "pro")
	handler := NewWebhookHandler(store, testStripeSecret, nil)

	sub := map[string]interface{}{
		"id":       "sub_123",
		"customer": "cus_sub_active",
		"status":   "active",
		"metadata": map[string]string{"tier": "team"},
		"items":    map[string]interface{}{"data": []interface{}{}},
	}
	event := stripeEvent{
		ID:   "evt_3",
		Type: "customer.subscription.updated",
		Data: stripeEventData{Object: mustMarshal(t, sub)},
	}
	body := mustMarshal(t, event)

	req := newStripeRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	got, _ := store.Get(context.Background(), tn.ID)
	if got.Status != "active" {
		t.Errorf("Status = %q, want %q", got.Status, "active")
	}
	if got.Tier != "team" {
		t.Errorf("Tier = %q, want %q", got.Tier, "team")
	}
	if got.RequestLimit != 1_000_000 {
		t.Errorf("RequestLimit = %d, want %d", got.RequestLimit, 1_000_000)
	}
}

func TestStripe_SubscriptionUpdated_PastDue(t *testing.T) {
	store := tenant.NewMemoryStore()
	tn := createTestTenant(t, store, "tn-4", "cus_sub_past", "pro")
	handler := NewWebhookHandler(store, testStripeSecret, nil)

	sub := map[string]interface{}{
		"id":       "sub_456",
		"customer": "cus_sub_past",
		"status":   "past_due",
		"metadata": map[string]string{},
		"items":    map[string]interface{}{"data": []interface{}{}},
	}
	event := stripeEvent{
		ID:   "evt_4",
		Type: "customer.subscription.updated",
		Data: stripeEventData{Object: mustMarshal(t, sub)},
	}
	body := mustMarshal(t, event)

	req := newStripeRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	got, _ := store.Get(context.Background(), tn.ID)
	if got.Status != "suspended" {
		t.Errorf("Status = %q, want %q", got.Status, "suspended")
	}
}

func TestStripe_SubscriptionDeleted(t *testing.T) {
	store := tenant.NewMemoryStore()
	tn := createTestTenant(t, store, "tn-5", "cus_sub_del", "pro")
	handler := NewWebhookHandler(store, testStripeSecret, nil)

	sub := map[string]interface{}{
		"id":       "sub_789",
		"customer": "cus_sub_del",
		"status":   "canceled",
		"metadata": map[string]string{},
		"items":    map[string]interface{}{"data": []interface{}{}},
	}
	event := stripeEvent{
		ID:   "evt_5",
		Type: "customer.subscription.deleted",
		Data: stripeEventData{Object: mustMarshal(t, sub)},
	}
	body := mustMarshal(t, event)

	req := newStripeRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	got, _ := store.Get(context.Background(), tn.ID)
	if got.Tier != "free" {
		t.Errorf("Tier = %q, want %q", got.Tier, "free")
	}
	if got.Status != "canceled" {
		t.Errorf("Status = %q, want %q", got.Status, "canceled")
	}
	if got.RequestLimit != 1_000 {
		t.Errorf("RequestLimit = %d, want %d (free tier)", got.RequestLimit, 1_000)
	}
}

func TestStripe_InvalidSignature(t *testing.T) {
	store := tenant.NewMemoryStore()
	handler := NewWebhookHandler(store, testStripeSecret, nil)

	body := []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{}}}`)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", "t=12345,v1=invalidsignature")
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestStripe_TierLimits(t *testing.T) {
	expected := map[string]int64{
		"free": 1_000,
		"pro":  100_000,
		"team": 1_000_000,
	}

	for tier, limit := range expected {
		got, ok := tierLimits[tier]
		if !ok {
			t.Errorf("tierLimits missing key %q", tier)
			continue
		}
		if got != limit {
			t.Errorf("tierLimits[%q] = %d, want %d", tier, got, limit)
		}
	}
}

func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return data
}
