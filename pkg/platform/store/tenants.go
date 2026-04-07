package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock/pkg/platform/model"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// TenantStore handles persistence for Tenant records.
type TenantStore struct {
	pool *pgxpool.Pool
}

// NewTenantStore creates a TenantStore backed by the given pool.
func NewTenantStore(pool *pgxpool.Pool) *TenantStore {
	return &TenantStore{pool: pool}
}

// Create inserts a new tenant and populates ID, CreatedAt, and UpdatedAt via RETURNING.
func (s *TenantStore) Create(ctx context.Context, t *model.Tenant) error {
	const q = `
		INSERT INTO tenants (clerk_org_id, name, slug, status, has_payment_method, stripe_customer_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	row := s.pool.QueryRow(ctx, q,
		t.ClerkOrgID,
		t.Name,
		t.Slug,
		t.Status,
		t.HasPaymentMethod,
		nilIfEmpty(t.StripeCustomerID),
	)
	if err := row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}
	return nil
}

// Get fetches a tenant by primary key.
func (s *TenantStore) Get(ctx context.Context, id string) (*model.Tenant, error) {
	const q = `
		SELECT id, clerk_org_id, name, slug, status, has_payment_method,
		       COALESCE(stripe_customer_id, ''), created_at, updated_at
		FROM tenants
		WHERE id = $1`

	return s.scanOne(s.pool.QueryRow(ctx, q, id))
}

// GetByClerkOrgID fetches a tenant by Clerk organisation ID.
func (s *TenantStore) GetByClerkOrgID(ctx context.Context, clerkOrgID string) (*model.Tenant, error) {
	const q = `
		SELECT id, clerk_org_id, name, slug, status, has_payment_method,
		       COALESCE(stripe_customer_id, ''), created_at, updated_at
		FROM tenants
		WHERE clerk_org_id = $1`

	return s.scanOne(s.pool.QueryRow(ctx, q, clerkOrgID))
}

// GetBySlug fetches a tenant by slug.
func (s *TenantStore) GetBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	const q = `
		SELECT id, clerk_org_id, name, slug, status, has_payment_method,
		       COALESCE(stripe_customer_id, ''), created_at, updated_at
		FROM tenants
		WHERE slug = $1`

	return s.scanOne(s.pool.QueryRow(ctx, q, slug))
}

// List returns all tenants ordered newest-first.
func (s *TenantStore) List(ctx context.Context) ([]model.Tenant, error) {
	const q = `
		SELECT id, clerk_org_id, name, slug, status, has_payment_method,
		       COALESCE(stripe_customer_id, ''), created_at, updated_at
		FROM tenants
		ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(
			&t.ID, &t.ClerkOrgID, &t.Name, &t.Slug, &t.Status,
			&t.HasPaymentMethod, &t.StripeCustomerID,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tenant row: %w", err)
		}
		tenants = append(tenants, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return tenants, nil
}

// Update persists mutable fields of the given tenant.
func (s *TenantStore) Update(ctx context.Context, t *model.Tenant) error {
	const q = `
		UPDATE tenants
		SET name               = $2,
		    slug               = $3,
		    status             = $4,
		    has_payment_method = $5,
		    stripe_customer_id = $6,
		    updated_at         = now()
		WHERE id = $1
		RETURNING updated_at`

	row := s.pool.QueryRow(ctx, q,
		t.ID,
		t.Name,
		t.Slug,
		t.Status,
		t.HasPaymentMethod,
		nilIfEmpty(t.StripeCustomerID),
	)
	if err := row.Scan(&t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("update tenant: %w", err)
	}
	return nil
}

// Delete removes a tenant by primary key.
func (s *TenantStore) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM tenants WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanOne scans a single tenant row, mapping pgx.ErrNoRows to ErrNotFound.
func (s *TenantStore) scanOne(row pgx.Row) (*model.Tenant, error) {
	var t model.Tenant
	err := row.Scan(
		&t.ID, &t.ClerkOrgID, &t.Name, &t.Slug, &t.Status,
		&t.HasPaymentMethod, &t.StripeCustomerID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan tenant: %w", err)
	}
	return &t, nil
}

// nilIfEmpty returns nil when s is empty, otherwise a pointer to s.
// Use this for nullable TEXT columns that should store NULL rather than "".
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
