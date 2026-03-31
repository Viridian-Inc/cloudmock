package tenant

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a tenant or record does not exist.
var ErrNotFound = errors.New("tenant: not found")

// Store defines the interface for tenant persistence.
type Store interface {
	Create(ctx context.Context, t *Tenant) error
	Get(ctx context.Context, id string) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	GetByClerkOrgID(ctx context.Context, clerkOrgID string) (*Tenant, error)
	List(ctx context.Context) ([]Tenant, error)
	Update(ctx context.Context, t *Tenant) error
	Delete(ctx context.Context, id string) error
	IncrementRequestCount(ctx context.Context, id string) error
	RecordUsage(ctx context.Context, record *UsageRecord) error
	GetUnreportedUsage(ctx context.Context) ([]UsageRecord, error)
	MarkUsageReported(ctx context.Context, recordID string) error
}
