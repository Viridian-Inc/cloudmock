// Package clerk handles Clerk webhook events and JWT verification
// for the hosted SaaS tier.
package clerk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/auth"
	"github.com/neureaux/cloudmock/pkg/saas/tenant"
)

// Svix signature tolerance window.
const signatureToleranceSeconds = 300 // 5 minutes

// ClerkEvent is the top-level webhook envelope sent by Clerk.
type ClerkEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// OrgData is the payload for organization.* events.
type OrgData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// UserData is the payload for user.* events.
type UserData struct {
	ID             string `json:"id"`
	EmailAddresses []struct {
		EmailAddress string `json:"email_address"`
	} `json:"email_addresses"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// WebhookHandler processes Clerk webhook events.
type WebhookHandler struct {
	tenants       tenant.Store
	users         auth.UserStore
	webhookSecret string
	logger        *slog.Logger
}

// NewWebhookHandler creates a new Clerk webhook handler.
func NewWebhookHandler(tenants tenant.Store, users auth.UserStore, webhookSecret string, logger *slog.Logger) *WebhookHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookHandler{
		tenants:       tenants,
		users:         users,
		webhookSecret: webhookSecret,
		logger:        logger,
	}
}

// HandleWebhook is the HTTP handler for POST /api/webhooks/clerk.
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		h.logger.Error("clerk webhook: failed to read body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Verify svix signature.
	if err := h.verifySvixSignature(r.Header, body); err != nil {
		h.logger.Warn("clerk webhook: signature verification failed", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var event ClerkEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("clerk webhook: failed to parse event", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	switch event.Type {
	case "organization.created":
		if err := h.handleOrgCreated(ctx, event.Data); err != nil {
			h.logger.Error("clerk webhook: org created failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case "organization.deleted":
		if err := h.handleOrgDeleted(ctx, event.Data); err != nil {
			h.logger.Error("clerk webhook: org deleted failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case "user.created":
		if err := h.handleUserCreated(ctx, event.Data); err != nil {
			h.logger.Error("clerk webhook: user created failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	default:
		h.logger.Debug("clerk webhook: unhandled event type", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"received":true}`))
}

func (h *WebhookHandler) handleOrgCreated(ctx context.Context, data json.RawMessage) error {
	var org OrgData
	if err := json.Unmarshal(data, &org); err != nil {
		return fmt.Errorf("unmarshal org data: %w", err)
	}

	t := &tenant.Tenant{
		ClerkOrgID: org.ID,
		Name:       org.Name,
		Slug:       org.Slug,
		Tier:       "free",
		Status:     "active",
	}

	if err := h.tenants.Create(ctx, t); err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}

	h.logger.Info("clerk webhook: tenant created",
		"tenant_id", t.ID,
		"clerk_org_id", org.ID,
		"slug", org.Slug,
	)
	return nil
}

func (h *WebhookHandler) handleOrgDeleted(ctx context.Context, data json.RawMessage) error {
	var org OrgData
	if err := json.Unmarshal(data, &org); err != nil {
		return fmt.Errorf("unmarshal org data: %w", err)
	}

	existing, err := h.tenants.GetByClerkOrgID(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("find tenant by clerk org id %s: %w", org.ID, err)
	}

	if err := h.tenants.Delete(ctx, existing.ID); err != nil {
		return fmt.Errorf("delete tenant %s: %w", existing.ID, err)
	}

	h.logger.Info("clerk webhook: tenant deleted",
		"tenant_id", existing.ID,
		"clerk_org_id", org.ID,
	)
	return nil
}

func (h *WebhookHandler) handleUserCreated(ctx context.Context, data json.RawMessage) error {
	var ud UserData
	if err := json.Unmarshal(data, &ud); err != nil {
		return fmt.Errorf("unmarshal user data: %w", err)
	}

	email := ""
	if len(ud.EmailAddresses) > 0 {
		email = ud.EmailAddresses[0].EmailAddress
	}

	name := strings.TrimSpace(ud.FirstName + " " + ud.LastName)
	if name == "" {
		name = email
	}

	user := &auth.User{
		ID:    ud.ID,
		Email: email,
		Name:  name,
		Role:  auth.RoleViewer,
	}

	if err := h.users.Create(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	h.logger.Info("clerk webhook: user created",
		"user_id", ud.ID,
		"email", email,
	)
	return nil
}

// verifySvixSignature verifies the Svix webhook signature.
// Clerk uses Svix for webhook delivery. The signature scheme uses
// HMAC-SHA256 with a timestamp to prevent replay attacks.
//
// Headers used:
//   - svix-id:        unique message identifier
//   - svix-timestamp: unix timestamp of when the message was sent
//   - svix-signature: comma-separated list of base64-encoded signatures (v1,...)
func (h *WebhookHandler) verifySvixSignature(header http.Header, body []byte) error {
	msgID := header.Get("svix-id")
	timestamp := header.Get("svix-timestamp")
	signatures := header.Get("svix-signature")

	if msgID == "" || timestamp == "" || signatures == "" {
		return fmt.Errorf("missing svix headers")
	}

	// Parse and validate timestamp to prevent replay attacks.
	ts, err := parseTimestamp(timestamp)
	if err != nil {
		return fmt.Errorf("invalid svix-timestamp: %w", err)
	}
	diff := time.Since(ts)
	if diff < 0 {
		diff = -diff
	}
	if diff > signatureToleranceSeconds*time.Second {
		return fmt.Errorf("timestamp outside tolerance window")
	}

	// Build the signed content: "{msg_id}.{timestamp}.{body}"
	signedContent := msgID + "." + timestamp + "." + string(body)

	// Decode the webhook secret. Svix secrets are base64-encoded with a "whsec_" prefix.
	secret := h.webhookSecret
	if after, ok := strings.CutPrefix(secret, "whsec_"); ok {
		secret = after
	}
	secretBytes, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return fmt.Errorf("decode webhook secret: %w", err)
	}

	// Compute expected signature.
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedContent))
	expected := mac.Sum(nil)

	// Check each provided signature (comma-separated, "v1,..." prefix).
	for _, sig := range strings.Split(signatures, " ") {
		parts := strings.SplitN(sig, ",", 2)
		if len(parts) != 2 {
			continue
		}
		// Only verify v1 signatures.
		if parts[0] != "v1" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			continue
		}
		if hmac.Equal(decoded, expected) {
			return nil
		}
	}

	return fmt.Errorf("no matching signature found")
}

// parseTimestamp parses a unix timestamp string.
func parseTimestamp(ts string) (time.Time, error) {
	var sec int64
	for _, c := range ts {
		if c < '0' || c > '9' {
			return time.Time{}, fmt.Errorf("non-digit in timestamp: %q", ts)
		}
		sec = sec*10 + int64(c-'0')
	}
	return time.Unix(sec, 0), nil
}

