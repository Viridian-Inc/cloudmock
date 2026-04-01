// Package filestore implements monitor.MonitorStore and monitor.AlertStore
// backed by JSON files on disk via the generic filestore package.
package filestore

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/neureaux/cloudmock/pkg/filestore"
	"github.com/neureaux/cloudmock/pkg/monitor"
)

// Store implements both monitor.MonitorStore and monitor.AlertStore
// using JSON file persistence.
type Store struct {
	monitors *filestore.JSONFileStore[monitor.Monitor]
	alerts   *filestore.JSONFileStore[monitor.AlertEvent]
}

// New creates a file-backed monitor/alert store.
// monDir and alertDir are created if they don't exist.
func New(monDir, alertDir string) (*Store, error) {
	ms, err := filestore.New[monitor.Monitor](monDir)
	if err != nil {
		return nil, err
	}
	as, err := filestore.New[monitor.AlertEvent](alertDir)
	if err != nil {
		return nil, err
	}
	return &Store{monitors: ms, alerts: as}, nil
}

// ---------------------------------------------------------------------------
// MonitorStore implementation
// ---------------------------------------------------------------------------

func (s *Store) Save(_ context.Context, m *monitor.Monitor) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	return s.monitors.Save(m.ID, *m)
}

func (s *Store) Get(_ context.Context, id string) (*monitor.Monitor, error) {
	m, err := s.monitors.Get(id)
	if err != nil {
		return nil, monitor.ErrNotFound
	}
	return &m, nil
}

func (s *Store) List(_ context.Context, filter monitor.MonitorFilter) ([]monitor.Monitor, error) {
	all, err := s.monitors.List()
	if err != nil {
		return nil, err
	}

	var results []monitor.Monitor
	// Iterate in reverse for newest-first (file listing is unordered but
	// matches the memory store's behaviour approximately).
	for i := len(all) - 1; i >= 0; i-- {
		m := all[i]
		if filter.Service != "" && m.Service != filter.Service && m.Service != "*" {
			continue
		}
		if filter.Type != "" && m.Type != filter.Type {
			continue
		}
		if filter.Status != "" && m.Status != filter.Status {
			continue
		}
		if filter.Enabled != nil && m.Enabled != *filter.Enabled {
			continue
		}
		results = append(results, m)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

func (s *Store) Update(_ context.Context, m *monitor.Monitor) error {
	// Verify existence.
	if _, err := s.monitors.Get(m.ID); err != nil {
		return monitor.ErrNotFound
	}
	m.UpdatedAt = time.Now()
	return s.monitors.Save(m.ID, *m)
}

func (s *Store) Delete(_ context.Context, id string) error {
	if err := s.monitors.Delete(id); err != nil {
		return monitor.ErrNotFound
	}
	return nil
}

func (s *Store) ListEnabled(_ context.Context) ([]monitor.Monitor, error) {
	all, err := s.monitors.List()
	if err != nil {
		return nil, err
	}
	var results []monitor.Monitor
	for _, m := range all {
		if m.Enabled {
			results = append(results, m)
		}
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// AlertStore implementation
// ---------------------------------------------------------------------------

func (s *Store) SaveAlert(_ context.Context, a *monitor.AlertEvent) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	return s.alerts.Save(a.ID, *a)
}

func (s *Store) GetAlert(_ context.Context, id string) (*monitor.AlertEvent, error) {
	a, err := s.alerts.Get(id)
	if err != nil {
		return nil, monitor.ErrNotFound
	}
	return &a, nil
}

func (s *Store) ListAlerts(_ context.Context, filter monitor.AlertFilter) ([]monitor.AlertEvent, error) {
	all, err := s.alerts.List()
	if err != nil {
		return nil, err
	}

	var results []monitor.AlertEvent
	for i := len(all) - 1; i >= 0; i-- {
		a := all[i]
		if filter.MonitorID != "" && a.MonitorID != filter.MonitorID {
			continue
		}
		if filter.Status != "" && a.Status != filter.Status {
			continue
		}
		if filter.Service != "" && a.Service != filter.Service {
			continue
		}
		results = append(results, a)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

// Compile-time interface checks.
var (
	_ monitor.MonitorStore = (*Store)(nil)
	_ monitor.AlertStore   = (*Store)(nil)
)
