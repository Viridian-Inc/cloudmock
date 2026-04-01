package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/neureaux/cloudmock/pkg/monitor"
)

// Store implements both monitor.MonitorStore and monitor.AlertStore
// using mutex-protected slices.
type Store struct {
	mu       sync.RWMutex
	monitors []monitor.Monitor
	alerts   []monitor.AlertEvent
}

// NewStore creates a new in-memory monitor/alert store.
func NewStore() *Store {
	return &Store{}
}

// ---------------------------------------------------------------------------
// MonitorStore implementation
// ---------------------------------------------------------------------------

// Save appends or replaces the monitor. If m.ID is empty a UUID is generated.
func (s *Store) Save(_ context.Context, m *monitor.Monitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now

	s.monitors = append(s.monitors, *m)
	return nil
}

// Get returns the monitor with the given ID, or ErrNotFound.
func (s *Store) Get(_ context.Context, id string) (*monitor.Monitor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.monitors {
		if s.monitors[i].ID == id {
			m := s.monitors[i]
			return &m, nil
		}
	}
	return nil, monitor.ErrNotFound
}

// List returns monitors matching the filter, newest first.
func (s *Store) List(_ context.Context, filter monitor.MonitorFilter) ([]monitor.Monitor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []monitor.Monitor
	for i := len(s.monitors) - 1; i >= 0; i-- {
		m := s.monitors[i]
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

// Update finds the monitor by ID and replaces it entirely.
func (s *Store) Update(_ context.Context, m *monitor.Monitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	m.UpdatedAt = time.Now()
	for i := range s.monitors {
		if s.monitors[i].ID == m.ID {
			s.monitors[i] = *m
			return nil
		}
	}
	return monitor.ErrNotFound
}

// Delete removes the monitor with the given ID.
func (s *Store) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.monitors {
		if s.monitors[i].ID == id {
			s.monitors = append(s.monitors[:i], s.monitors[i+1:]...)
			return nil
		}
	}
	return monitor.ErrNotFound
}

// ListEnabled returns all monitors with Enabled == true.
func (s *Store) ListEnabled(_ context.Context) ([]monitor.Monitor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []monitor.Monitor
	for _, m := range s.monitors {
		if m.Enabled {
			results = append(results, m)
		}
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// AlertStore implementation
// ---------------------------------------------------------------------------

// SaveAlert appends the alert event. If a.ID is empty a UUID is generated.
func (s *Store) SaveAlert(_ context.Context, a *monitor.AlertEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}

	s.alerts = append(s.alerts, *a)
	return nil
}

// GetAlert returns the alert event with the given ID, or ErrNotFound.
func (s *Store) GetAlert(_ context.Context, id string) (*monitor.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.alerts {
		if s.alerts[i].ID == id {
			a := s.alerts[i]
			return &a, nil
		}
	}
	return nil, monitor.ErrNotFound
}

// ListAlerts returns alert events matching the filter, newest first.
func (s *Store) ListAlerts(_ context.Context, filter monitor.AlertFilter) ([]monitor.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []monitor.AlertEvent
	for i := len(s.alerts) - 1; i >= 0; i-- {
		a := s.alerts[i]
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
