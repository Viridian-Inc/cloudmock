// Package dynamostore implements tenant.Store backed by DynamoDB
// via the generic dynamostore package.
package dynamostore

import (
	"context"
	"time"

	"github.com/google/uuid"

	ds "github.com/Viridian-Inc/cloudmock/pkg/dynamostore"
	"github.com/Viridian-Inc/cloudmock/pkg/saas/tenant"
)

const (
	featureTenant = "TENANT"
	featureUsage  = "USAGE"
)

// Store implements tenant.Store.
type Store struct {
	db *ds.Store
}

// New creates a DynamoDB-backed tenant store.
func New(db *ds.Store) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, t *tenant.Tenant) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	return s.db.Put(ctx, featureTenant, t.ID, t)
}

func (s *Store) Get(ctx context.Context, id string) (*tenant.Tenant, error) {
	var t tenant.Tenant
	if err := s.db.Get(ctx, featureTenant, id, &t); err != nil {
		if err == ds.ErrNotFound {
			return nil, tenant.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (s *Store) GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	all, err := s.listAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range all {
		if t.Slug == slug {
			return &t, nil
		}
	}
	return nil, tenant.ErrNotFound
}

func (s *Store) GetByClerkOrgID(ctx context.Context, clerkOrgID string) (*tenant.Tenant, error) {
	all, err := s.listAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range all {
		if t.ClerkOrgID == clerkOrgID {
			return &t, nil
		}
	}
	return nil, tenant.ErrNotFound
}

func (s *Store) List(ctx context.Context) ([]tenant.Tenant, error) {
	return s.listAll(ctx)
}

func (s *Store) Update(ctx context.Context, t *tenant.Tenant) error {
	t.UpdatedAt = time.Now()
	if err := s.db.UpdateData(ctx, featureTenant, t.ID, t); err != nil {
		if err == ds.ErrNotFound {
			return tenant.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	return s.db.Delete(ctx, featureTenant, id)
}

func (s *Store) IncrementRequestCount(ctx context.Context, id string) error {
	t, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	t.RequestCount++
	t.UpdatedAt = time.Now()
	return s.db.UpdateData(ctx, featureTenant, id, t)
}

func (s *Store) RecordUsage(ctx context.Context, record *tenant.UsageRecord) error {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	return s.db.Put(ctx, featureUsage, record.ID, record)
}

func (s *Store) GetUnreportedUsage(ctx context.Context) ([]tenant.UsageRecord, error) {
	var all []tenant.UsageRecord
	if err := s.db.List(ctx, featureUsage, &all); err != nil {
		return nil, err
	}
	var result []tenant.UsageRecord
	for _, r := range all {
		if !r.ReportedToStripe {
			result = append(result, r)
		}
	}
	return result, nil
}

func (s *Store) MarkUsageReported(ctx context.Context, recordID string) error {
	var r tenant.UsageRecord
	if err := s.db.Get(ctx, featureUsage, recordID, &r); err != nil {
		return err
	}
	r.ReportedToStripe = true
	return s.db.UpdateData(ctx, featureUsage, recordID, r)
}

func (s *Store) listAll(ctx context.Context) ([]tenant.Tenant, error) {
	var all []tenant.Tenant
	if err := s.db.List(ctx, featureTenant, &all); err != nil {
		return nil, err
	}
	return all, nil
}

// Compile-time interface check.
var _ tenant.Store = (*Store)(nil)
