// Package dynamostore implements monitor.MonitorStore and monitor.AlertStore
// backed by DynamoDB via the generic dynamostore package.
package dynamostore

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Viridian-Inc/cloudmock/pkg/dynamostore"
	"github.com/Viridian-Inc/cloudmock/pkg/monitor"
)

const (
	featureMonitor = "MONITOR"
	featureAlert   = "ALERT"
)

// Store implements both monitor.MonitorStore and monitor.AlertStore.
type Store struct {
	db *dynamostore.Store
}

// New creates a DynamoDB-backed monitor/alert store.
func New(db *dynamostore.Store) *Store {
	return &Store{db: db}
}

// ---------------------------------------------------------------------------
// MonitorStore implementation
// ---------------------------------------------------------------------------

func (s *Store) Save(ctx context.Context, m *monitor.Monitor) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	return s.db.Put(ctx, featureMonitor, m.ID, m)
}

func (s *Store) Get(ctx context.Context, id string) (*monitor.Monitor, error) {
	var m monitor.Monitor
	if err := s.db.Get(ctx, featureMonitor, id, &m); err != nil {
		if err == dynamostore.ErrNotFound {
			return nil, monitor.ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (s *Store) List(ctx context.Context, filter monitor.MonitorFilter) ([]monitor.Monitor, error) {
	var all []monitor.Monitor
	if err := s.db.List(ctx, featureMonitor, &all); err != nil {
		return nil, err
	}

	var results []monitor.Monitor
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

func (s *Store) Update(ctx context.Context, m *monitor.Monitor) error {
	m.UpdatedAt = time.Now()
	if err := s.db.UpdateData(ctx, featureMonitor, m.ID, m); err != nil {
		if err == dynamostore.ErrNotFound {
			return monitor.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	return s.db.Delete(ctx, featureMonitor, id)
}

func (s *Store) ListEnabled(ctx context.Context) ([]monitor.Monitor, error) {
	var all []monitor.Monitor
	if err := s.db.List(ctx, featureMonitor, &all); err != nil {
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

func (s *Store) SaveAlert(ctx context.Context, a *monitor.AlertEvent) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	return s.db.Put(ctx, featureAlert, a.ID, a)
}

func (s *Store) GetAlert(ctx context.Context, id string) (*monitor.AlertEvent, error) {
	var a monitor.AlertEvent
	if err := s.db.Get(ctx, featureAlert, id, &a); err != nil {
		if err == dynamostore.ErrNotFound {
			return nil, monitor.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (s *Store) ListAlerts(ctx context.Context, filter monitor.AlertFilter) ([]monitor.AlertEvent, error) {
	var all []monitor.AlertEvent
	if err := s.db.List(ctx, featureAlert, &all); err != nil {
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
