package monitor

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a monitor or alert does not exist.
var ErrNotFound = errors.New("monitor: not found")

// MonitorStore persists and retrieves monitors.
type MonitorStore interface {
	Save(ctx context.Context, m *Monitor) error
	Get(ctx context.Context, id string) (*Monitor, error)
	List(ctx context.Context, filter MonitorFilter) ([]Monitor, error)
	Update(ctx context.Context, m *Monitor) error
	Delete(ctx context.Context, id string) error
	ListEnabled(ctx context.Context) ([]Monitor, error)
}

// AlertStore persists and retrieves alert events.
type AlertStore interface {
	SaveAlert(ctx context.Context, a *AlertEvent) error
	GetAlert(ctx context.Context, id string) (*AlertEvent, error)
	ListAlerts(ctx context.Context, filter AlertFilter) ([]AlertEvent, error)
}
