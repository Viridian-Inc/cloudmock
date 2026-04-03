package tenant

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestCreate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tn := &Tenant{
		ClerkOrgID: "org_abc",
		Name:       "Acme Corp",
		Slug:       "acme",
		Tier:       "free",
		Status:     "active",
	}

	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if tn.ID == "" {
		t.Fatal("expected ID to be set after Create")
	}
	if tn.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if tn.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}

	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", got.Name, "Acme Corp")
	}
	if got.Slug != "acme" {
		t.Errorf("Slug = %q, want %q", got.Slug, "acme")
	}
	if got.ClerkOrgID != "org_abc" {
		t.Errorf("ClerkOrgID = %q, want %q", got.ClerkOrgID, "org_abc")
	}
	if got.Tier != "free" {
		t.Errorf("Tier = %q, want %q", got.Tier, "free")
	}
	if got.Status != "active" {
		t.Errorf("Status = %q, want %q", got.Status, "active")
	}
}

func TestCreateDuplicate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	t1 := &Tenant{ClerkOrgID: "org_1", Name: "Acme", Slug: "acme", Tier: "free", Status: "active"}
	if err := store.Create(ctx, t1); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	// Duplicate slug should fail.
	t2 := &Tenant{ClerkOrgID: "org_2", Name: "Acme 2", Slug: "acme", Tier: "free", Status: "active"}
	err := store.Create(ctx, t2)
	if err == nil {
		t.Fatal("expected error for duplicate slug, got nil")
	}
	var dupErr *DuplicateError
	if !errors.As(err, &dupErr) {
		t.Fatalf("expected DuplicateError, got %T: %v", err, err)
	}
	if dupErr.Field != "slug" {
		t.Errorf("DuplicateError.Field = %q, want %q", dupErr.Field, "slug")
	}

	// Duplicate ClerkOrgID should also fail.
	t3 := &Tenant{ClerkOrgID: "org_1", Name: "Other", Slug: "other", Tier: "free", Status: "active"}
	err = store.Create(ctx, t3)
	if err == nil {
		t.Fatal("expected error for duplicate ClerkOrgID, got nil")
	}
	if !errors.As(err, &dupErr) {
		t.Fatalf("expected DuplicateError, got %T: %v", err, err)
	}
	if dupErr.Field != "clerk_org_id" {
		t.Errorf("DuplicateError.Field = %q, want %q", dupErr.Field, "clerk_org_id")
	}
}

func TestGetByID(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tn := &Tenant{ClerkOrgID: "org_1", Name: "One", Slug: "one", Tier: "free", Status: "active"}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != tn.ID {
		t.Errorf("ID = %q, want %q", got.ID, tn.ID)
	}
	if got.Name != "One" {
		t.Errorf("Name = %q, want %q", got.Name, "One")
	}

	// Non-existent ID returns ErrNotFound.
	_, err = store.Get(ctx, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestGetBySlug(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tn := &Tenant{ClerkOrgID: "org_1", Name: "Slug Test", Slug: "slug-test", Tier: "pro", Status: "active"}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetBySlug(ctx, "slug-test")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.ID != tn.ID {
		t.Errorf("ID = %q, want %q", got.ID, tn.ID)
	}
	if got.Name != "Slug Test" {
		t.Errorf("Name = %q, want %q", got.Name, "Slug Test")
	}

	_, err = store.GetBySlug(ctx, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetBySlug nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestGetByClerkOrgID(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tn := &Tenant{ClerkOrgID: "org_xyz", Name: "Org Test", Slug: "org-test", Tier: "team", Status: "active"}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByClerkOrgID(ctx, "org_xyz")
	if err != nil {
		t.Fatalf("GetByClerkOrgID: %v", err)
	}
	if got.ID != tn.ID {
		t.Errorf("ID = %q, want %q", got.ID, tn.ID)
	}
	if got.Tier != "team" {
		t.Errorf("Tier = %q, want %q", got.Tier, "team")
	}

	_, err = store.GetByClerkOrgID(ctx, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByClerkOrgID nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tn := &Tenant{ClerkOrgID: "org_1", Name: "Update Me", Slug: "update-me", Tier: "free", Status: "active"}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	originalUpdatedAt := tn.UpdatedAt
	tn.Tier = "pro"
	tn.RequestLimit = 100000

	if err := store.Update(ctx, tn); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.Tier != "pro" {
		t.Errorf("Tier = %q, want %q", got.Tier, "pro")
	}
	if got.RequestLimit != 100000 {
		t.Errorf("RequestLimit = %d, want %d", got.RequestLimit, 100000)
	}
	if !got.UpdatedAt.After(originalUpdatedAt) || got.UpdatedAt.Equal(originalUpdatedAt) {
		// UpdatedAt should be refreshed.
	}

	// Update nonexistent returns ErrNotFound.
	notFound := &Tenant{ID: "nonexistent"}
	if err := store.Update(ctx, notFound); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tn := &Tenant{ClerkOrgID: "org_1", Name: "Delete Me", Slug: "delete-me", Tier: "free", Status: "active"}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Delete(ctx, tn.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Get(ctx, tn.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete: got %v, want ErrNotFound", err)
	}

	// Deleting again should fail.
	if err := store.Delete(ctx, tn.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete twice: got %v, want ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tenants := []*Tenant{
		{ClerkOrgID: "org_1", Name: "One", Slug: "one", Tier: "free", Status: "active"},
		{ClerkOrgID: "org_2", Name: "Two", Slug: "two", Tier: "pro", Status: "active"},
		{ClerkOrgID: "org_3", Name: "Three", Slug: "three", Tier: "team", Status: "active"},
	}

	for _, tn := range tenants {
		if err := store.Create(ctx, tn); err != nil {
			t.Fatalf("Create %s: %v", tn.Name, err)
		}
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("List returned %d tenants, want 3", len(list))
	}

	slugs := map[string]bool{}
	for _, tn := range list {
		slugs[tn.Slug] = true
	}
	for _, expected := range []string{"one", "two", "three"} {
		if !slugs[expected] {
			t.Errorf("List missing slug %q", expected)
		}
	}
}

func TestIncrementRequestCount(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tn := &Tenant{ClerkOrgID: "org_1", Name: "Counter", Slug: "counter", Tier: "free", Status: "active"}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	for i := 0; i < 5; i++ {
		if err := store.IncrementRequestCount(ctx, tn.ID); err != nil {
			t.Fatalf("IncrementRequestCount iteration %d: %v", i, err)
		}
	}

	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RequestCount != 5 {
		t.Errorf("RequestCount = %d, want 5", got.RequestCount)
	}

	// Increment nonexistent returns ErrNotFound.
	if err := store.IncrementRequestCount(ctx, "nonexistent"); !errors.Is(err, ErrNotFound) {
		t.Errorf("IncrementRequestCount nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestConcurrentIncrements(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tn := &Tenant{ClerkOrgID: "org_1", Name: "Concurrent", Slug: "concurrent", Tier: "free", Status: "active"}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if err := store.IncrementRequestCount(ctx, tn.ID); err != nil {
				t.Errorf("IncrementRequestCount: %v", err)
			}
		}()
	}
	wg.Wait()

	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RequestCount != goroutines {
		t.Errorf("RequestCount = %d, want %d", got.RequestCount, goroutines)
	}
}
