// Package stripe handles Stripe webhook events and usage metering
// for the hosted SaaS tier.
package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/saas/tenant"
)

// signatureToleranceSeconds is the max age of a Stripe webhook event.
const signatureToleranceSeconds = 300 // 5 minutes

// tierLimits maps subscription tiers to their request limits.
var tierLimits = map[string]int64{
	"free": 1_000,
	"pro":  100_000,
	"team": 1_000_000,
}

// WebhookHandler processes Stripe webhook events.
type WebhookHandler struct {
	tenants tenant.Store
	secret  string // webhook signing secret
	logger  *slog.Logger
}

// stripeEvent is the top-level Stripe webhook event envelope.
type stripeEvent struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data stripeEventData `json:"data"`
}

// stripeEventData wraps the event object.
type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

// checkoutSession is the relevant fields from a Stripe Checkout Session.
type checkoutSession struct {
	ID             string            `json:"id"`
	CustomerID     string            `json:"customer"`
	SubscriptionID string            `json:"subscription"`
	Metadata       map[string]string `json:"metadata"`
	Mode           string            `json:"mode"`
}

// invoice is the relevant fields from a Stripe Invoice.
type invoice struct {
	ID             string `json:"id"`
	CustomerID     string `json:"customer"`
	SubscriptionID string `json:"subscription"`
	Status         string `json:"status"`
}

// subscription is the relevant fields from a Stripe Subscription.
type subscription struct {
	ID         string             `json:"id"`
	CustomerID string             `json:"customer"`
	Status     string             `json:"status"`
	Items      subscriptionItems  `json:"items"`
	Metadata   map[string]string  `json:"metadata"`
}

// subscriptionItems wraps the subscription items list.
type subscriptionItems struct {
	Data []subscriptionItem `json:"data"`
}

// subscriptionItem is a single line item in a subscription.
type subscriptionItem struct {
	ID    string        `json:"id"`
	Price subscriptionPrice `json:"price"`
}

// subscriptionPrice holds the price details for a subscription item.
type subscriptionPrice struct {
	ID      string `json:"id"`
	Product string `json:"product"`
}

// NewWebhookHandler creates a new Stripe webhook handler.
func NewWebhookHandler(tenants tenant.Store, webhookSecret string, logger *slog.Logger) *WebhookHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookHandler{
		tenants: tenants,
		secret:  webhookSecret,
		logger:  logger,
	}
}

// HandleWebhook is the HTTP handler for POST /api/webhooks/stripe.
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		h.logger.Error("stripe webhook: failed to read body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Verify Stripe signature.
	sigHeader := r.Header.Get("Stripe-Signature")
	if err := verifyStripeSignature(body, sigHeader, h.secret); err != nil {
		h.logger.Warn("stripe webhook: signature verification failed", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var event stripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("stripe webhook: failed to parse event", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	switch event.Type {
	case "checkout.session.completed":
		if err := h.handleCheckoutCompleted(ctx, event.Data.Object); err != nil {
			h.logger.Error("stripe webhook: checkout completed failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case "invoice.paid":
		if err := h.handleInvoicePaid(ctx, event.Data.Object); err != nil {
			h.logger.Error("stripe webhook: invoice paid failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case "customer.subscription.updated":
		if err := h.handleSubscriptionUpdated(ctx, event.Data.Object); err != nil {
			h.logger.Error("stripe webhook: subscription updated failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case "customer.subscription.deleted":
		if err := h.handleSubscriptionDeleted(ctx, event.Data.Object); err != nil {
			h.logger.Error("stripe webhook: subscription deleted failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	default:
		h.logger.Debug("stripe webhook: unhandled event type", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"received":true}`))
}

func (h *WebhookHandler) handleCheckoutCompleted(ctx context.Context, data json.RawMessage) error {
	var sess checkoutSession
	if err := json.Unmarshal(data, &sess); err != nil {
		return fmt.Errorf("unmarshal checkout session: %w", err)
	}

	// Only handle subscription checkouts.
	if sess.Mode != "subscription" {
		h.logger.Debug("stripe webhook: ignoring non-subscription checkout", "mode", sess.Mode)
		return nil
	}

	tenantID, ok := sess.Metadata["tenant_id"]
	if !ok {
		return fmt.Errorf("checkout session %s missing tenant_id in metadata", sess.ID)
	}

	tier, ok := sess.Metadata["tier"]
	if !ok {
		tier = "pro" // default to pro if not specified
	}

	t, err := h.tenants.Get(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("get tenant %s: %w", tenantID, err)
	}

	t.StripeCustomerID = sess.CustomerID
	t.StripeSubscriptionID = sess.SubscriptionID
	t.Tier = tier
	t.Status = "active"
	if limit, exists := tierLimits[tier]; exists {
		t.RequestLimit = limit
	}

	if err := h.tenants.Update(ctx, t); err != nil {
		return fmt.Errorf("update tenant %s: %w", tenantID, err)
	}

	h.logger.Info("stripe webhook: checkout completed",
		"tenant_id", tenantID,
		"tier", tier,
		"customer_id", sess.CustomerID,
		"subscription_id", sess.SubscriptionID,
	)
	return nil
}

func (h *WebhookHandler) handleInvoicePaid(ctx context.Context, data json.RawMessage) error {
	var inv invoice
	if err := json.Unmarshal(data, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}

	// Find tenant by stripe customer ID.
	t, err := h.findTenantByCustomerID(ctx, inv.CustomerID)
	if err != nil {
		return fmt.Errorf("find tenant for customer %s: %w", inv.CustomerID, err)
	}

	// Reset request count for the new billing period.
	t.RequestCount = 0
	t.Status = "active"

	if err := h.tenants.Update(ctx, t); err != nil {
		return fmt.Errorf("update tenant %s: %w", t.ID, err)
	}

	h.logger.Info("stripe webhook: invoice paid, request count reset",
		"tenant_id", t.ID,
		"invoice_id", inv.ID,
	)
	return nil
}

func (h *WebhookHandler) handleSubscriptionUpdated(ctx context.Context, data json.RawMessage) error {
	var sub subscription
	if err := json.Unmarshal(data, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	t, err := h.findTenantByCustomerID(ctx, sub.CustomerID)
	if err != nil {
		return fmt.Errorf("find tenant for customer %s: %w", sub.CustomerID, err)
	}

	// Update tier from metadata if present.
	if tier, ok := sub.Metadata["tier"]; ok {
		t.Tier = tier
		if limit, exists := tierLimits[tier]; exists {
			t.RequestLimit = limit
		}
	}

	// Map Stripe subscription status to tenant status.
	switch sub.Status {
	case "active", "trialing":
		t.Status = "active"
	case "past_due", "unpaid":
		t.Status = "suspended"
	case "canceled", "incomplete_expired":
		t.Status = "canceled"
	}

	t.StripeSubscriptionID = sub.ID

	if err := h.tenants.Update(ctx, t); err != nil {
		return fmt.Errorf("update tenant %s: %w", t.ID, err)
	}

	h.logger.Info("stripe webhook: subscription updated",
		"tenant_id", t.ID,
		"status", sub.Status,
		"tier", t.Tier,
	)
	return nil
}

func (h *WebhookHandler) handleSubscriptionDeleted(ctx context.Context, data json.RawMessage) error {
	var sub subscription
	if err := json.Unmarshal(data, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}

	t, err := h.findTenantByCustomerID(ctx, sub.CustomerID)
	if err != nil {
		return fmt.Errorf("find tenant for customer %s: %w", sub.CustomerID, err)
	}

	t.Status = "canceled"
	t.Tier = "free"
	if limit, exists := tierLimits["free"]; exists {
		t.RequestLimit = limit
	}

	if err := h.tenants.Update(ctx, t); err != nil {
		return fmt.Errorf("update tenant %s: %w", t.ID, err)
	}

	h.logger.Info("stripe webhook: subscription deleted",
		"tenant_id", t.ID,
		"customer_id", sub.CustomerID,
	)
	return nil
}

// findTenantByCustomerID looks up a tenant by their Stripe customer ID.
// It lists all tenants and finds the match (a dedicated index could be
// added to the Store interface in the future).
func (h *WebhookHandler) findTenantByCustomerID(ctx context.Context, customerID string) (*tenant.Tenant, error) {
	tenants, err := h.tenants.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	for i := range tenants {
		if tenants[i].StripeCustomerID == customerID {
			return &tenants[i], nil
		}
	}
	return nil, fmt.Errorf("no tenant with stripe_customer_id %s", customerID)
}

// verifyStripeSignature verifies a Stripe webhook signature using HMAC-SHA256.
//
// The Stripe-Signature header format is:
//
//	t=timestamp,v1=signature[,v1=signature...]
//
// The signed payload is: "{timestamp}.{body}"
func verifyStripeSignature(body []byte, sigHeader, secret string) error {
	if sigHeader == "" {
		return fmt.Errorf("missing Stripe-Signature header")
	}

	var timestamp string
	var signatures []string

	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}

	if timestamp == "" {
		return fmt.Errorf("missing timestamp in Stripe-Signature")
	}
	if len(signatures) == 0 {
		return fmt.Errorf("missing v1 signature in Stripe-Signature")
	}

	// Validate timestamp to prevent replay attacks.
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	diff := time.Since(time.Unix(ts, 0))
	if diff < 0 {
		diff = -diff
	}
	if diff > signatureToleranceSeconds*time.Second {
		return fmt.Errorf("timestamp outside tolerance window")
	}

	// Compute expected signature: HMAC-SHA256(secret, "{timestamp}.{body}")
	signedPayload := timestamp + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	// Check each v1 signature.
	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return nil
		}
	}

	return fmt.Errorf("no matching v1 signature")
}
