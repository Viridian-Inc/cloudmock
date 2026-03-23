package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/neureaux/cloudmock/pkg/webhook"
)

// Store implements webhook.Store using a mutex-protected slice.
type Store struct {
	mu      sync.RWMutex
	configs []webhook.Config
}

// NewStore creates a new in-memory webhook store.
func NewStore() *Store {
	return &Store{}
}

// Save adds or replaces a webhook config. If cfg.ID is empty a UUID is generated.
func (s *Store) Save(_ context.Context, cfg *webhook.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}

	// Replace if exists.
	for i := range s.configs {
		if s.configs[i].ID == cfg.ID {
			s.configs[i] = *cfg
			return nil
		}
	}

	s.configs = append(s.configs, *cfg)
	return nil
}

// Get returns the config with the given ID, or webhook.ErrNotFound.
func (s *Store) Get(_ context.Context, id string) (*webhook.Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.configs {
		if s.configs[i].ID == id {
			c := s.configs[i]
			return &c, nil
		}
	}
	return nil, webhook.ErrNotFound
}

// List returns all stored webhook configs.
func (s *Store) List(_ context.Context) ([]webhook.Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]webhook.Config, len(s.configs))
	copy(result, s.configs)
	return result, nil
}

// Delete removes the config with the given ID. Returns webhook.ErrNotFound if
// no config with that ID exists.
func (s *Store) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.configs {
		if s.configs[i].ID == id {
			s.configs = append(s.configs[:i], s.configs[i+1:]...)
			return nil
		}
	}
	return webhook.ErrNotFound
}

// ListByEvent returns all active configs that subscribe to the given event.
func (s *Store) ListByEvent(_ context.Context, event string) ([]webhook.Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []webhook.Config
	for _, cfg := range s.configs {
		if !cfg.Active {
			continue
		}
		for _, e := range cfg.Events {
			if e == event {
				result = append(result, cfg)
				break
			}
		}
	}
	return result, nil
}

// Compile-time interface check.
var _ webhook.Store = (*Store)(nil)
