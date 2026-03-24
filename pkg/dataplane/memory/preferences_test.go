package memory_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/neureaux/cloudmock/pkg/dataplane"
	"github.com/neureaux/cloudmock/pkg/dataplane/memory"
)

func TestPreferenceStore_SetGetRoundtrip(t *testing.T) {
	s := memory.NewPreferenceStore()
	ctx := context.Background()

	value := json.RawMessage(`{"theme":"dark"}`)
	if err := s.Set(ctx, "ui", "appearance", value); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.Get(ctx, "ui", "appearance")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("got %s, want %s", got, value)
	}
}

func TestPreferenceStore_GetNonexistent(t *testing.T) {
	s := memory.NewPreferenceStore()
	ctx := context.Background()

	_, err := s.Get(ctx, "ui", "nonexistent")
	if err != dataplane.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Also test missing namespace entirely.
	_, err = s.Get(ctx, "missing-ns", "key")
	if err != dataplane.ErrNotFound {
		t.Errorf("expected ErrNotFound for missing namespace, got %v", err)
	}
}

func TestPreferenceStore_ListByNamespace(t *testing.T) {
	s := memory.NewPreferenceStore()
	ctx := context.Background()

	_ = s.Set(ctx, "ui", "theme", json.RawMessage(`"dark"`))
	_ = s.Set(ctx, "ui", "lang", json.RawMessage(`"en"`))
	_ = s.Set(ctx, "other", "key", json.RawMessage(`"val"`))

	result, err := s.ListByNamespace(ctx, "ui")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d keys, want 2", len(result))
	}
	if string(result["theme"]) != `"dark"` {
		t.Errorf("got theme=%s, want %q", result["theme"], `"dark"`)
	}

	// Empty namespace returns empty map.
	empty, err := s.ListByNamespace(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected empty map, got %d entries", len(empty))
	}
}

func TestPreferenceStore_Delete(t *testing.T) {
	s := memory.NewPreferenceStore()
	ctx := context.Background()

	_ = s.Set(ctx, "ui", "theme", json.RawMessage(`"dark"`))

	if err := s.Delete(ctx, "ui", "theme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := s.Get(ctx, "ui", "theme")
	if err != dataplane.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}

	// Delete from non-existent namespace should not error.
	if err := s.Delete(ctx, "nonexistent", "key"); err != nil {
		t.Fatalf("unexpected error deleting from missing namespace: %v", err)
	}
}

func TestPreferenceStore_SetUpsert(t *testing.T) {
	s := memory.NewPreferenceStore()
	ctx := context.Background()

	_ = s.Set(ctx, "ui", "theme", json.RawMessage(`"dark"`))
	_ = s.Set(ctx, "ui", "theme", json.RawMessage(`"light"`))

	got, err := s.Get(ctx, "ui", "theme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `"light"` {
		t.Errorf("got %s, want %q", got, `"light"`)
	}
}
