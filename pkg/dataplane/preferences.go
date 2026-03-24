package dataplane

import (
	"context"
	"encoding/json"
)

// PreferenceStore provides backend-persisted key-value storage for user
// preferences, replacing frontend localStorage/sessionStorage.
type PreferenceStore interface {
	Get(ctx context.Context, namespace, key string) (json.RawMessage, error)
	Set(ctx context.Context, namespace, key string, value json.RawMessage) error
	Delete(ctx context.Context, namespace, key string) error
	ListByNamespace(ctx context.Context, namespace string) (map[string]json.RawMessage, error)
}
