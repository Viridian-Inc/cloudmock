package memory

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
)

// PreferenceStore satisfies the dataplane.PreferenceStore interface using
// in-memory storage.
type PreferenceStore struct {
	mu   sync.RWMutex
	data map[string]map[string]json.RawMessage // namespace → key → value
}

// NewPreferenceStore creates an empty in-memory PreferenceStore.
func NewPreferenceStore() *PreferenceStore {
	return &PreferenceStore{
		data: make(map[string]map[string]json.RawMessage),
	}
}

// Get returns the value for the given namespace and key.
// Returns ErrNotFound if the key does not exist.
func (s *PreferenceStore) Get(_ context.Context, namespace, key string) (json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ns, ok := s.data[namespace]
	if !ok {
		return nil, dataplane.ErrNotFound
	}
	v, ok := ns[key]
	if !ok {
		return nil, dataplane.ErrNotFound
	}
	return v, nil
}

// Set upserts a value for the given namespace and key.
func (s *PreferenceStore) Set(_ context.Context, namespace, key string, value json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ns, ok := s.data[namespace]
	if !ok {
		ns = make(map[string]json.RawMessage)
		s.data[namespace] = ns
	}
	ns[key] = value
	return nil
}

// Delete removes a key from the given namespace.
func (s *PreferenceStore) Delete(_ context.Context, namespace, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ns, ok := s.data[namespace]
	if !ok {
		return nil
	}
	delete(ns, key)
	return nil
}

// ListByNamespace returns all key-value pairs for the given namespace.
// Returns an empty map if the namespace does not exist.
func (s *PreferenceStore) ListByNamespace(_ context.Context, namespace string) (map[string]json.RawMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ns, ok := s.data[namespace]
	if !ok {
		return make(map[string]json.RawMessage), nil
	}

	out := make(map[string]json.RawMessage, len(ns))
	for k, v := range ns {
		out[k] = v
	}
	return out, nil
}

// Compile-time interface check.
var _ dataplane.PreferenceStore = (*PreferenceStore)(nil)
