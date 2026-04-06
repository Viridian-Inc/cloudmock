package store_test

import (
	"context"
	"testing"

	"github.com/neureaux/cloudmock/pkg/platform/model"
	"github.com/neureaux/cloudmock/pkg/platform/store"
)

func TestTenantStore_CreateAndGet(t *testing.T) {
	pool := setupTestDB(t)
	s := store.NewTenantStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{
		ClerkOrgID: "org_abc123",
		Name:       "Acme Corp",
		Slug:       "acme-corp",
		Status:     "active",
	}

	if err := s.Create(ctx, tenant); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tenant.ID == "" {
		t.Fatal("expected ID to be set after Create")
	}
	if tenant.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set after Create")
	}
	if tenant.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set after Create")
	}

	got, err := s.Get(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != tenant.Name {
		t.Errorf("Name mismatch: got %q, want %q", got.Name, tenant.Name)
	}
	if got.Slug != tenant.Slug {
		t.Errorf("Slug mismatch: got %q, want %q", got.Slug, tenant.Slug)
	}
}

func TestTenantStore_GetByClerkOrgID(t *testing.T) {
	pool := setupTestDB(t)
	s := store.NewTenantStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{
		ClerkOrgID: "org_clerk456",
		Name:       "Beta Inc",
		Slug:       "beta-inc",
		Status:     "active",
	}
	if err := s.Create(ctx, tenant); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.GetByClerkOrgID(ctx, "org_clerk456")
	if err != nil {
		t.Fatalf("GetByClerkOrgID: %v", err)
	}
	if got.ID != tenant.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, tenant.ID)
	}

	_, err = s.GetByClerkOrgID(ctx, "org_nonexistent")
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTenantStore_GetBySlug(t *testing.T) {
	pool := setupTestDB(t)
	s := store.NewTenantStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{
		ClerkOrgID: "org_slug789",
		Name:       "Gamma LLC",
		Slug:       "gamma-llc",
		Status:     "active",
	}
	if err := s.Create(ctx, tenant); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.GetBySlug(ctx, "gamma-llc")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.ID != tenant.ID {
		t.Errorf("ID mismatch: got %q, want %q", got.ID, tenant.ID)
	}

	_, err = s.GetBySlug(ctx, "no-such-slug")
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTenantStore_Update(t *testing.T) {
	pool := setupTestDB(t)
	s := store.NewTenantStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{
		ClerkOrgID: "org_upd001",
		Name:       "Old Name",
		Slug:       "old-slug",
		Status:     "active",
	}
	if err := s.Create(ctx, tenant); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tenant.Name = "New Name"
	tenant.Slug = "new-slug"
	tenant.Status = "suspended"
	tenant.HasPaymentMethod = true
	tenant.StripeCustomerID = "cus_stripe123"

	if err := s.Update(ctx, tenant); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Get(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("Name: got %q, want %q", got.Name, "New Name")
	}
	if got.Slug != "new-slug" {
		t.Errorf("Slug: got %q, want %q", got.Slug, "new-slug")
	}
	if got.Status != "suspended" {
		t.Errorf("Status: got %q, want %q", got.Status, "suspended")
	}
	if !got.HasPaymentMethod {
		t.Error("HasPaymentMethod: expected true")
	}
	if got.StripeCustomerID != "cus_stripe123" {
		t.Errorf("StripeCustomerID: got %q, want %q", got.StripeCustomerID, "cus_stripe123")
	}
}

func TestTenantStore_List(t *testing.T) {
	pool := setupTestDB(t)
	s := store.NewTenantStore(pool)
	ctx := context.Background()

	for i, item := range []struct {
		clerkOrgID, name, slug string
	}{
		{"org_list1", "First", "first-tenant"},
		{"org_list2", "Second", "second-tenant"},
		{"org_list3", "Third", "third-tenant"},
	} {
		tenant := &model.Tenant{
			ClerkOrgID: item.clerkOrgID,
			Name:       item.name,
			Slug:       item.slug,
			Status:     "active",
		}
		if err := s.Create(ctx, tenant); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	tenants, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tenants) < 3 {
		t.Errorf("expected at least 3 tenants, got %d", len(tenants))
	}
	// Verify newest-first ordering.
	for i := 1; i < len(tenants); i++ {
		if tenants[i].CreatedAt.After(tenants[i-1].CreatedAt) {
			t.Errorf("tenants not in descending created_at order at index %d", i)
		}
	}
}

func TestTenantStore_Delete(t *testing.T) {
	pool := setupTestDB(t)
	s := store.NewTenantStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{
		ClerkOrgID: "org_del001",
		Name:       "To Delete",
		Slug:       "to-delete",
		Status:     "active",
	}
	if err := s.Create(ctx, tenant); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.Delete(ctx, tenant.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Get(ctx, tenant.ID)
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound after Delete, got %v", err)
	}

	// Deleting again should return ErrNotFound.
	if err := s.Delete(ctx, tenant.ID); err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound on second Delete, got %v", err)
	}
}

func TestTenantStore_DuplicateSlugRejected(t *testing.T) {
	pool := setupTestDB(t)
	s := store.NewTenantStore(pool)
	ctx := context.Background()

	tenant1 := &model.Tenant{
		ClerkOrgID: "org_dup1",
		Name:       "Dup One",
		Slug:       "dup-slug",
		Status:     "active",
	}
	if err := s.Create(ctx, tenant1); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	tenant2 := &model.Tenant{
		ClerkOrgID: "org_dup2",
		Name:       "Dup Two",
		Slug:       "dup-slug", // same slug
		Status:     "active",
	}
	if err := s.Create(ctx, tenant2); err == nil {
		t.Fatal("expected error on duplicate slug, got nil")
	}
}
