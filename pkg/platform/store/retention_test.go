package store_test

import (
	"context"
	"testing"

	"github.com/neureaux/cloudmock/pkg/platform/model"
	"github.com/neureaux/cloudmock/pkg/platform/store"
)

func setupRetentionFixture(t *testing.T) (tenantID string, rs *store.RetentionStore) {
	t.Helper()
	pool := setupTestDB(t)
	ts := store.NewTenantStore(pool)
	rs = store.NewRetentionStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{
		ClerkOrgID: "org_retention001",
		Name:       "Retention Tenant",
		Slug:       "retention-tenant",
		Status:     "active",
	}
	if err := ts.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant.ID, rs
}

func TestRetentionStore_UpsertAndGetByTenant(t *testing.T) {
	tenantID, rs := setupRetentionFixture(t)
	ctx := context.Background()

	if err := rs.Upsert(ctx, tenantID, "audit_log", 90); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := rs.Upsert(ctx, tenantID, "request_log", 30); err != nil {
		t.Fatalf("Upsert request_log: %v", err)
	}

	policies, err := rs.GetByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetByTenant: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}

	byType := make(map[string]model.DataRetention)
	for _, p := range policies {
		byType[p.ResourceType] = p
	}

	if p, ok := byType["audit_log"]; !ok {
		t.Error("missing audit_log policy")
	} else if p.RetentionDays != 90 {
		t.Errorf("audit_log RetentionDays: got %d, want 90", p.RetentionDays)
	}

	if p, ok := byType["request_log"]; !ok {
		t.Error("missing request_log policy")
	} else if p.RetentionDays != 30 {
		t.Errorf("request_log RetentionDays: got %d, want 30", p.RetentionDays)
	}
}

func TestRetentionStore_UpsertUpdatesExisting(t *testing.T) {
	tenantID, rs := setupRetentionFixture(t)
	ctx := context.Background()

	if err := rs.Upsert(ctx, tenantID, "audit_log", 365); err != nil {
		t.Fatalf("Upsert initial: %v", err)
	}

	// Update the same (tenantID, resource_type) with a new value.
	if err := rs.Upsert(ctx, tenantID, "audit_log", 180); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}

	policies, err := rs.GetByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetByTenant: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy after upsert, got %d", len(policies))
	}
	if policies[0].RetentionDays != 180 {
		t.Errorf("RetentionDays after upsert: got %d, want 180", policies[0].RetentionDays)
	}
}

func TestRetentionStore_GetByTenantEmpty(t *testing.T) {
	tenantID, rs := setupRetentionFixture(t)
	ctx := context.Background()

	policies, err := rs.GetByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetByTenant: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("expected 0 policies for new tenant, got %d", len(policies))
	}
}
