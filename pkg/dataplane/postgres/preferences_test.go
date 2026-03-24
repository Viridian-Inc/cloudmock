package postgres_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/neureaux/cloudmock/pkg/dataplane"
	pgImpl "github.com/neureaux/cloudmock/pkg/dataplane/postgres"
)

func TestPreferenceStore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)

	// Apply preferences schema.
	applySchema(t, ctx, pool, "../../../docker/init/postgres/07-preferences-schema.sql")

	s := pgImpl.NewPreferenceStore(pool)

	// Get nonexistent → ErrNotFound.
	_, err := s.Get(ctx, "ui", "theme")
	if err != dataplane.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Set + Get roundtrip.
	value := json.RawMessage(`{"mode":"dark","fontSize":14}`)
	if err := s.Set(ctx, "ui", "theme", value); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.Get(ctx, "ui", "theme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("got %s, want %s", got, value)
	}

	// Upsert.
	value2 := json.RawMessage(`{"mode":"light"}`)
	if err := s.Set(ctx, "ui", "theme", value2); err != nil {
		t.Fatalf("Set upsert: %v", err)
	}
	got, _ = s.Get(ctx, "ui", "theme")
	if string(got) != string(value2) {
		t.Errorf("after upsert got %s, want %s", got, value2)
	}

	// ListByNamespace.
	_ = s.Set(ctx, "ui", "lang", json.RawMessage(`"en"`))
	_ = s.Set(ctx, "other", "key", json.RawMessage(`"val"`))

	result, err := s.ListByNamespace(ctx, "ui")
	if err != nil {
		t.Fatalf("ListByNamespace: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d keys, want 2", len(result))
	}

	// Empty namespace.
	empty, err := s.ListByNamespace(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("ListByNamespace empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected empty map, got %d entries", len(empty))
	}

	// Delete.
	if err := s.Delete(ctx, "ui", "theme"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = s.Get(ctx, "ui", "theme")
	if err != dataplane.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
