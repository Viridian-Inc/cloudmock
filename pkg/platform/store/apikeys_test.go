package store_test

import (
	"context"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/platform/model"
	"github.com/Viridian-Inc/cloudmock/pkg/platform/store"
)

func setupAPIKeyFixture(t *testing.T) (tenantID, appID string, ts *store.TenantStore, as *store.AppStore, ks *store.APIKeyStore) {
	t.Helper()
	pool := setupTestDB(t)
	ts = store.NewTenantStore(pool)
	as = store.NewAppStore(pool)
	ks = store.NewAPIKeyStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{
		ClerkOrgID: "org_apikey001",
		Name:       "APIKey Tenant",
		Slug:       "apikey-tenant",
		Status:     "active",
	}
	if err := ts.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	app := &model.App{
		TenantID:  tenant.ID,
		Name:      "APIKey App",
		Slug:      "apikey-app",
		Endpoint:  "https://apikey-app.example.com",
		InfraType: "shared",
		Status:    "running",
	}
	if err := as.Create(ctx, app); err != nil {
		t.Fatalf("create app: %v", err)
	}

	return tenant.ID, app.ID, ts, as, ks
}

func TestAPIKeyStore_CreateAndGetByPlaintext(t *testing.T) {
	tenantID, appID, _, _, ks := setupAPIKeyFixture(t)
	ctx := context.Background()

	plaintext, key, err := ks.Create(ctx, tenantID, appID, "test key", "developer")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if plaintext == "" {
		t.Fatal("expected non-empty plaintext")
	}
	if key.ID == "" {
		t.Fatal("expected key ID to be set")
	}
	if key.KeyHash == "" {
		t.Fatal("expected key hash to be set")
	}
	if key.TenantID != tenantID {
		t.Errorf("TenantID: got %q, want %q", key.TenantID, tenantID)
	}
	if key.AppID != appID {
		t.Errorf("AppID: got %q, want %q", key.AppID, appID)
	}

	got, err := ks.GetByPlaintext(ctx, plaintext)
	if err != nil {
		t.Fatalf("GetByPlaintext: %v", err)
	}
	if got.ID != key.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, key.ID)
	}
	if got.Name != "test key" {
		t.Errorf("Name: got %q, want %q", got.Name, "test key")
	}
	if got.Role != "developer" {
		t.Errorf("Role: got %q, want %q", got.Role, "developer")
	}
}

func TestAPIKeyStore_ListByApp_KeyHashCleared(t *testing.T) {
	tenantID, appID, _, _, ks := setupAPIKeyFixture(t)
	ctx := context.Background()

	for _, name := range []string{"key-one", "key-two"} {
		if _, _, err := ks.Create(ctx, tenantID, appID, name, "developer"); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	keys, err := ks.ListByApp(ctx, appID)
	if err != nil {
		t.Fatalf("ListByApp: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	for _, k := range keys {
		if k.KeyHash != "" {
			t.Errorf("expected KeyHash to be empty in ListByApp, got %q", k.KeyHash)
		}
	}
}

func TestAPIKeyStore_RevokePreventslookup(t *testing.T) {
	tenantID, appID, _, _, ks := setupAPIKeyFixture(t)
	ctx := context.Background()

	plaintext, key, err := ks.Create(ctx, tenantID, appID, "revokable", "developer")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := ks.Revoke(ctx, key.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	_, err = ks.GetByPlaintext(ctx, plaintext)
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound after revoke, got %v", err)
	}
}

func TestAPIKeyStore_RevokeRemovesFromList(t *testing.T) {
	tenantID, appID, _, _, ks := setupAPIKeyFixture(t)
	ctx := context.Background()

	_, key, err := ks.Create(ctx, tenantID, appID, "will-be-revoked", "developer")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, _, err := ks.Create(ctx, tenantID, appID, "stays-active", "developer"); err != nil {
		t.Fatalf("Create active: %v", err)
	}

	if err := ks.Revoke(ctx, key.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	keys, err := ks.ListByApp(ctx, appID)
	if err != nil {
		t.Fatalf("ListByApp: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 active key, got %d", len(keys))
	}
	if keys[0].Name != "stays-active" {
		t.Errorf("unexpected key name: %q", keys[0].Name)
	}
}

func TestAPIKeyStore_TouchLastUsed(t *testing.T) {
	tenantID, appID, _, _, ks := setupAPIKeyFixture(t)
	ctx := context.Background()

	plaintext, key, err := ks.Create(ctx, tenantID, appID, "touch-test", "developer")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if key.LastUsedAt != nil {
		t.Fatal("expected LastUsedAt to be nil before touch")
	}

	if err := ks.TouchLastUsed(ctx, key.ID); err != nil {
		t.Fatalf("TouchLastUsed: %v", err)
	}

	got, err := ks.GetByPlaintext(ctx, plaintext)
	if err != nil {
		t.Fatalf("GetByPlaintext after touch: %v", err)
	}
	if got.LastUsedAt == nil {
		t.Error("expected LastUsedAt to be set after touch")
	}
}
