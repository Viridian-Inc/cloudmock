package store_test

import (
	"context"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/platform/model"
	"github.com/Viridian-Inc/cloudmock/pkg/platform/store"
)

func TestAuditStore_AppendAndQuery(t *testing.T) {
	pool := setupTestDB(t)
	as := store.NewAuditStore(pool)
	ctx := context.Background()

	// audit_log has no FK to tenants so any UUID string works.
	tenantID := "00000000-0000-0000-0000-000000000001"

	entry := &model.AuditEntry{
		TenantID:     tenantID,
		ActorID:      "user_abc",
		ActorType:    "user",
		Action:       "app.create",
		ResourceType: "app",
		ResourceID:   "app-123",
		UserAgent:    "Mozilla/5.0",
		Metadata:     map[string]any{"key": "value"},
	}

	if err := as.Append(ctx, entry); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("expected ID to be set after Append")
	}
	if entry.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set after Append")
	}

	entries, err := as.Query(ctx, store.AuditFilter{TenantID: tenantID})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.ID != entry.ID {
		t.Errorf("ID: got %q, want %q", got.ID, entry.ID)
	}
	if got.Action != "app.create" {
		t.Errorf("Action: got %q, want %q", got.Action, "app.create")
	}
	if got.Metadata == nil {
		t.Error("expected metadata to be set")
	} else if got.Metadata["key"] != "value" {
		t.Errorf("metadata key: got %v, want %q", got.Metadata["key"], "value")
	}
}

func TestAuditStore_QueryWithActionFilter(t *testing.T) {
	pool := setupTestDB(t)
	as := store.NewAuditStore(pool)
	ctx := context.Background()

	tenantID := "00000000-0000-0000-0000-000000000002"

	for _, action := range []string{"app.create", "app.delete", "key.create"} {
		e := &model.AuditEntry{
			TenantID:     tenantID,
			ActorID:      "user_abc",
			ActorType:    "user",
			Action:       action,
			ResourceType: "app",
			ResourceID:   "res-1",
		}
		if err := as.Append(ctx, e); err != nil {
			t.Fatalf("Append %s: %v", action, err)
		}
	}

	entries, err := as.Query(ctx, store.AuditFilter{TenantID: tenantID, Action: "app.create"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for action filter, got %d", len(entries))
	}
	if entries[0].Action != "app.create" {
		t.Errorf("Action: got %q, want %q", entries[0].Action, "app.create")
	}
}

func TestAuditStore_Count(t *testing.T) {
	pool := setupTestDB(t)
	as := store.NewAuditStore(pool)
	ctx := context.Background()

	tenantID := "00000000-0000-0000-0000-000000000003"

	for i := 0; i < 5; i++ {
		e := &model.AuditEntry{
			TenantID:     tenantID,
			ActorID:      "user_abc",
			ActorType:    "user",
			Action:       "key.create",
			ResourceType: "api_key",
			ResourceID:   "key-1",
		}
		if err := as.Append(ctx, e); err != nil {
			t.Fatalf("Append[%d]: %v", i, err)
		}
	}

	count, err := as.Count(ctx, store.AuditFilter{TenantID: tenantID})
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 5 {
		t.Errorf("Count: got %d, want 5", count)
	}

	count, err = as.Count(ctx, store.AuditFilter{TenantID: tenantID, Action: "key.create"})
	if err != nil {
		t.Fatalf("Count with filter: %v", err)
	}
	if count != 5 {
		t.Errorf("Count with action filter: got %d, want 5", count)
	}

	count, err = as.Count(ctx, store.AuditFilter{TenantID: tenantID, Action: "nonexistent"})
	if err != nil {
		t.Fatalf("Count nonexistent: %v", err)
	}
	if count != 0 {
		t.Errorf("Count nonexistent: got %d, want 0", count)
	}
}
