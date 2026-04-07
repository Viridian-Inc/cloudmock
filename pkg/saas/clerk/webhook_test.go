package clerk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authmemory "github.com/Viridian-Inc/cloudmock/pkg/auth/memory"
	"github.com/Viridian-Inc/cloudmock/pkg/saas/tenant"
)

// testWebhookSecret is a base64-encoded HMAC key used in tests.
// The raw key is 32 bytes of 0x01.
var testWebhookSecret string

func init() {
	key := bytes.Repeat([]byte{0x01}, 32)
	testWebhookSecret = "whsec_" + base64.StdEncoding.EncodeToString(key)
}

// signSvix generates valid Svix signature headers for the given body.
func signSvix(body []byte, secret string) (msgID, timestamp, signature string) {
	msgID = "msg_test123"
	timestamp = fmt.Sprintf("%d", time.Now().Unix())

	// Strip whsec_ prefix and decode.
	raw := secret
	if len(raw) > 6 && raw[:6] == "whsec_" {
		raw = raw[6:]
	}
	secretBytes, _ := base64.StdEncoding.DecodeString(raw)

	signedContent := msgID + "." + timestamp + "." + string(body)
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedContent))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	signature = "v1," + sig
	return
}

// newSignedRequest creates a POST request with valid Svix signature headers.
func newSignedRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	msgID, ts, sig := signSvix(body, testWebhookSecret)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/clerk", bytes.NewReader(body))
	req.Header.Set("svix-id", msgID)
	req.Header.Set("svix-timestamp", ts)
	req.Header.Set("svix-signature", sig)
	return req
}

func TestWebhook_OrgCreated(t *testing.T) {
	tenantStore := tenant.NewMemoryStore()
	userStore := authmemory.NewStore()
	handler := NewWebhookHandler(tenantStore, userStore, testWebhookSecret, nil)

	event := ClerkEvent{
		Type: "organization.created",
		Data: mustMarshal(t, OrgData{
			ID:   "org_clerk_abc",
			Name: "Acme Corp",
			Slug: "acme",
		}),
	}
	body := mustMarshal(t, event)

	req := newSignedRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify tenant was created.
	got, err := tenantStore.GetByClerkOrgID(context.Background(), "org_clerk_abc")
	if err != nil {
		t.Fatalf("GetByClerkOrgID: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", got.Name, "Acme Corp")
	}
	if got.Slug != "acme" {
		t.Errorf("Slug = %q, want %q", got.Slug, "acme")
	}
	if got.Tier != "free" {
		t.Errorf("Tier = %q, want %q", got.Tier, "free")
	}
	if got.Status != "active" {
		t.Errorf("Status = %q, want %q", got.Status, "active")
	}
}

func TestWebhook_OrgDeleted(t *testing.T) {
	ctx := context.Background()
	tenantStore := tenant.NewMemoryStore()
	userStore := authmemory.NewStore()
	handler := NewWebhookHandler(tenantStore, userStore, testWebhookSecret, nil)

	// Pre-create a tenant.
	tn := &tenant.Tenant{
		ClerkOrgID: "org_to_delete",
		Name:       "Delete Me",
		Slug:       "delete-me",
		Tier:       "free",
		Status:     "active",
	}
	if err := tenantStore.Create(ctx, tn); err != nil {
		t.Fatalf("pre-create tenant: %v", err)
	}

	event := ClerkEvent{
		Type: "organization.deleted",
		Data: mustMarshal(t, OrgData{ID: "org_to_delete"}),
	}
	body := mustMarshal(t, event)

	req := newSignedRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify tenant was deleted.
	_, err := tenantStore.GetByClerkOrgID(ctx, "org_to_delete")
	if err == nil {
		t.Fatal("expected tenant to be deleted, but GetByClerkOrgID returned nil error")
	}
}

func TestWebhook_UserCreated(t *testing.T) {
	tenantStore := tenant.NewMemoryStore()
	userStore := authmemory.NewStore()
	handler := NewWebhookHandler(tenantStore, userStore, testWebhookSecret, nil)

	userData := map[string]interface{}{
		"id": "user_clerk_123",
		"email_addresses": []map[string]string{
			{"email_address": "alice@example.com"},
		},
		"first_name": "Alice",
		"last_name":  "Smith",
	}
	event := ClerkEvent{
		Type: "user.created",
		Data: mustMarshal(t, userData),
	}
	body := mustMarshal(t, event)

	req := newSignedRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify user was created in the user store.
	user, err := userStore.GetByID(context.Background(), "user_clerk_123")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "alice@example.com")
	}
	if user.Name != "Alice Smith" {
		t.Errorf("Name = %q, want %q", user.Name, "Alice Smith")
	}
}

func TestWebhook_UnknownEvent(t *testing.T) {
	tenantStore := tenant.NewMemoryStore()
	userStore := authmemory.NewStore()
	handler := NewWebhookHandler(tenantStore, userStore, testWebhookSecret, nil)

	event := ClerkEvent{
		Type: "some.unknown.event",
		Data: mustMarshal(t, map[string]string{"id": "whatever"}),
	}
	body := mustMarshal(t, event)

	req := newSignedRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestWebhook_MalformedJSON(t *testing.T) {
	tenantStore := tenant.NewMemoryStore()
	userStore := authmemory.NewStore()
	handler := NewWebhookHandler(tenantStore, userStore, testWebhookSecret, nil)

	body := []byte(`{this is not json!!!}`)

	req := newSignedRequest(t, body)
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestWebhook_MissingSignature(t *testing.T) {
	tenantStore := tenant.NewMemoryStore()
	userStore := authmemory.NewStore()
	handler := NewWebhookHandler(tenantStore, userStore, testWebhookSecret, nil)

	event := ClerkEvent{
		Type: "organization.created",
		Data: mustMarshal(t, OrgData{ID: "org_1", Name: "Test", Slug: "test"}),
	}
	body := mustMarshal(t, event)

	// No svix headers at all.
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/clerk", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handler.HandleWebhook(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
}

// mustMarshal is a test helper that marshals v to JSON or fails.
func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return data
}
