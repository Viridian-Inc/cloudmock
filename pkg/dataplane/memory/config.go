package memory

import (
	"context"
	"sync"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// ConfigStore satisfies the dataplane.ConfigStore interface using in-memory
// storage for config, deploys, saved views, and services.
type ConfigStore struct {
	mu       sync.RWMutex
	cfg      *config.Config
	deploys  []dataplane.DeployEvent
	views    []dataplane.SavedView
	services []dataplane.ServiceEntry
}

// NewConfigStore creates a ConfigStore with optional initial config.
func NewConfigStore(cfg *config.Config) *ConfigStore {
	return &ConfigStore{cfg: cfg}
}

// GetConfig returns the stored config.
func (s *ConfigStore) GetConfig(_ context.Context) (*config.Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg == nil {
		return nil, dataplane.ErrNotFound
	}
	return s.cfg, nil
}

// SetConfig replaces the stored config.
func (s *ConfigStore) SetConfig(_ context.Context, cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	return nil
}

// ListDeploys returns deploy events matching the filter.
func (s *ConfigStore) ListDeploys(_ context.Context, filter dataplane.DeployFilter) ([]dataplane.DeployEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []dataplane.DeployEvent
	for i := len(s.deploys) - 1; i >= 0; i-- {
		d := s.deploys[i]
		if filter.Service != "" && d.Service != filter.Service {
			continue
		}
		out = append(out, d)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

// AddDeploy appends a deploy event.
func (s *ConfigStore) AddDeploy(_ context.Context, deploy dataplane.DeployEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deploys = append(s.deploys, deploy)
	return nil
}

// ListViews returns all saved views.
func (s *ConfigStore) ListViews(_ context.Context) ([]dataplane.SavedView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]dataplane.SavedView, len(s.views))
	copy(out, s.views)
	return out, nil
}

// SaveView adds or replaces a saved view (matched by ID).
func (s *ConfigStore) SaveView(_ context.Context, view dataplane.SavedView) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, v := range s.views {
		if v.ID == view.ID {
			s.views[i] = view
			return nil
		}
	}
	s.views = append(s.views, view)
	return nil
}

// DeleteView removes the saved view with the given ID.
func (s *ConfigStore) DeleteView(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, v := range s.views {
		if v.ID == id {
			s.views = append(s.views[:i], s.views[i+1:]...)
			return nil
		}
	}
	return dataplane.ErrNotFound
}

// ListServices returns all registered services.
func (s *ConfigStore) ListServices(_ context.Context) ([]dataplane.ServiceEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]dataplane.ServiceEntry, len(s.services))
	copy(out, s.services)
	return out, nil
}

// UpsertService adds or updates a service entry (matched by Name).
func (s *ConfigStore) UpsertService(_ context.Context, svc dataplane.ServiceEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.services {
		if existing.Name == svc.Name {
			s.services[i] = svc
			return nil
		}
	}
	s.services = append(s.services, svc)
	return nil
}

// Compile-time interface check.
var _ dataplane.ConfigStore = (*ConfigStore)(nil)
