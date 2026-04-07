package store_test

import (
	"context"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/platform/model"
	"github.com/Viridian-Inc/cloudmock/pkg/platform/store"
)

// createTestTenant is a convenience helper that creates a tenant in the DB and
// returns its ID.
func createTestTenant(t *testing.T, ts *store.TenantStore, clerkOrgID, slug string) string {
	t.Helper()
	tenant := &model.Tenant{
		ClerkOrgID: clerkOrgID,
		Name:       slug,
		Slug:       slug,
		Status:     "active",
	}
	if err := ts.Create(context.Background(), tenant); err != nil {
		t.Fatalf("createTestTenant: %v", err)
	}
	return tenant.ID
}

func TestAppStore_CreateAndGet(t *testing.T) {
	pool := setupTestDB(t)
	ts := store.NewTenantStore(pool)
	as := store.NewAppStore(pool)
	ctx := context.Background()

	tenantID := createTestTenant(t, ts, "org_app001", "app-tenant-1")

	app := &model.App{
		TenantID:  tenantID,
		Name:      "My App",
		Slug:      "my-app",
		Endpoint:  "https://my-app.example.com",
		InfraType: "shared",
		Status:    "provisioning",
	}

	if err := as.Create(ctx, app); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if app.ID == "" {
		t.Fatal("expected ID to be set after Create")
	}
	if app.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set after Create")
	}

	got, err := as.Get(ctx, app.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != app.Name {
		t.Errorf("Name: got %q, want %q", got.Name, app.Name)
	}
	if got.Endpoint != app.Endpoint {
		t.Errorf("Endpoint: got %q, want %q", got.Endpoint, app.Endpoint)
	}
	if got.TenantID != tenantID {
		t.Errorf("TenantID: got %q, want %q", got.TenantID, tenantID)
	}
}

func TestAppStore_ListByTenant(t *testing.T) {
	pool := setupTestDB(t)
	ts := store.NewTenantStore(pool)
	as := store.NewAppStore(pool)
	ctx := context.Background()

	tenantID := createTestTenant(t, ts, "org_app002", "app-tenant-2")
	otherTenantID := createTestTenant(t, ts, "org_app003", "app-tenant-3")

	// Create two apps for our tenant.
	for i, name := range []string{"alpha", "beta"} {
		app := &model.App{
			TenantID:  tenantID,
			Name:      name,
			Slug:      name,
			Endpoint:  "https://" + name + ".example.com",
			InfraType: "shared",
			Status:    "provisioning",
		}
		if err := as.Create(ctx, app); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	// Create one app for the other tenant (should not appear in results).
	other := &model.App{
		TenantID:  otherTenantID,
		Name:      "other",
		Slug:      "other",
		Endpoint:  "https://other.example.com",
		InfraType: "shared",
		Status:    "provisioning",
	}
	if err := as.Create(ctx, other); err != nil {
		t.Fatalf("Create other: %v", err)
	}

	apps, err := as.ListByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("ListByTenant: %v", err)
	}
	if len(apps) != 2 {
		t.Errorf("expected 2 apps, got %d", len(apps))
	}
	for _, a := range apps {
		if a.TenantID != tenantID {
			t.Errorf("app %s has wrong tenant_id %s", a.ID, a.TenantID)
		}
	}
}

func TestAppStore_GetByEndpoint(t *testing.T) {
	pool := setupTestDB(t)
	ts := store.NewTenantStore(pool)
	as := store.NewAppStore(pool)
	ctx := context.Background()

	tenantID := createTestTenant(t, ts, "org_app004", "app-tenant-4")

	app := &model.App{
		TenantID:  tenantID,
		Name:      "endpoint-app",
		Slug:      "endpoint-app",
		Endpoint:  "https://endpoint-app.example.com",
		InfraType: "shared",
		Status:    "running",
	}
	if err := as.Create(ctx, app); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := as.GetByEndpoint(ctx, "https://endpoint-app.example.com")
	if err != nil {
		t.Fatalf("GetByEndpoint: %v", err)
	}
	if got.ID != app.ID {
		t.Errorf("ID: got %q, want %q", got.ID, app.ID)
	}

	_, err = as.GetByEndpoint(ctx, "https://nonexistent.example.com")
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAppStore_UpdateInfraFields(t *testing.T) {
	pool := setupTestDB(t)
	ts := store.NewTenantStore(pool)
	as := store.NewAppStore(pool)
	ctx := context.Background()

	tenantID := createTestTenant(t, ts, "org_app005", "app-tenant-5")

	app := &model.App{
		TenantID:  tenantID,
		Name:      "infra-app",
		Slug:      "infra-app",
		Endpoint:  "https://infra-app.example.com",
		InfraType: "shared",
		Status:    "provisioning",
	}
	if err := as.Create(ctx, app); err != nil {
		t.Fatalf("Create: %v", err)
	}

	app.InfraType = "dedicated"
	app.FlyAppName = "fly-app-123"
	app.FlyMachineID = "mach-456"
	app.Status = "running"

	if err := as.Update(ctx, app); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := as.Get(ctx, app.ID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.InfraType != "dedicated" {
		t.Errorf("InfraType: got %q, want %q", got.InfraType, "dedicated")
	}
	if got.FlyAppName != "fly-app-123" {
		t.Errorf("FlyAppName: got %q, want %q", got.FlyAppName, "fly-app-123")
	}
	if got.FlyMachineID != "mach-456" {
		t.Errorf("FlyMachineID: got %q, want %q", got.FlyMachineID, "mach-456")
	}
	if got.Status != "running" {
		t.Errorf("Status: got %q, want %q", got.Status, "running")
	}
}

func TestAppStore_CascadeDeleteFromTenant(t *testing.T) {
	pool := setupTestDB(t)
	ts := store.NewTenantStore(pool)
	as := store.NewAppStore(pool)
	ctx := context.Background()

	tenantID := createTestTenant(t, ts, "org_app006", "app-tenant-6")

	app := &model.App{
		TenantID:  tenantID,
		Name:      "cascade-app",
		Slug:      "cascade-app",
		Endpoint:  "https://cascade-app.example.com",
		InfraType: "shared",
		Status:    "running",
	}
	if err := as.Create(ctx, app); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Deleting the tenant should cascade-delete the app.
	if err := ts.Delete(ctx, tenantID); err != nil {
		t.Fatalf("Delete tenant: %v", err)
	}

	_, err := as.Get(ctx, app.ID)
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound after cascade delete, got %v", err)
	}
}

func TestAppStore_DuplicateSlugPerTenantRejected(t *testing.T) {
	pool := setupTestDB(t)
	ts := store.NewTenantStore(pool)
	as := store.NewAppStore(pool)
	ctx := context.Background()

	tenantID := createTestTenant(t, ts, "org_app007", "app-tenant-7")

	app1 := &model.App{
		TenantID:  tenantID,
		Name:      "dup-app",
		Slug:      "dup-slug",
		Endpoint:  "https://dup-app1.example.com",
		InfraType: "shared",
		Status:    "provisioning",
	}
	if err := as.Create(ctx, app1); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	app2 := &model.App{
		TenantID:  tenantID,
		Name:      "dup-app2",
		Slug:      "dup-slug", // same slug, same tenant
		Endpoint:  "https://dup-app2.example.com",
		InfraType: "shared",
		Status:    "provisioning",
	}
	if err := as.Create(ctx, app2); err == nil {
		t.Fatal("expected error on duplicate (tenant_id, slug), got nil")
	}
}
