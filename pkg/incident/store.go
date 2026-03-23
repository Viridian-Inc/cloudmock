package incident

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when an incident does not exist.
var ErrNotFound = errors.New("incident not found")

// IncidentStore persists and retrieves incidents.
type IncidentStore interface {
	Save(ctx context.Context, inc *Incident) error
	Get(ctx context.Context, id string) (*Incident, error)
	List(ctx context.Context, filter IncidentFilter) ([]Incident, error)
	Update(ctx context.Context, inc *Incident) error
	FindActiveByKey(ctx context.Context, service, deployID string, since time.Time) (*Incident, error)
}
