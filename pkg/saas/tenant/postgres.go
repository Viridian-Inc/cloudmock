package tenant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store against PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a Store backed by the given connection pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Create(ctx context.Context, t *Tenant) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	_, err := s.pool.Exec(ctx, `
		INSERT INTO tenants (
			id, clerk_org_id, name, slug,
			stripe_customer_id, stripe_subscription_id,
			tier, status,
			fly_machine_id, fly_app_name,
			request_count, request_limit, data_retention_days,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6,
			$7, $8,
			$9, $10,
			$11, $12, $13,
			$14, $15
		)`,
		t.ID, t.ClerkOrgID, t.Name, t.Slug,
		nullStr(t.StripeCustomerID), nullStr(t.StripeSubscriptionID),
		t.Tier, t.Status,
		nullStr(t.FlyMachineID), nullStr(t.FlyAppName),
		t.RequestCount, t.RequestLimit, t.DataRetentionDays,
		t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("tenant create: %w", err)
	}
	return nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (*Tenant, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, clerk_org_id, name, slug,
		       stripe_customer_id, stripe_subscription_id,
		       tier, status,
		       fly_machine_id, fly_app_name,
		       request_count, request_limit, data_retention_days,
		       created_at, updated_at
		FROM tenants
		WHERE id = $1`, id)

	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("tenant get: %w", err)
	}
	return &t, nil
}

func (s *PostgresStore) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, clerk_org_id, name, slug,
		       stripe_customer_id, stripe_subscription_id,
		       tier, status,
		       fly_machine_id, fly_app_name,
		       request_count, request_limit, data_retention_days,
		       created_at, updated_at
		FROM tenants
		WHERE slug = $1`, slug)

	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("tenant get by slug: %w", err)
	}
	return &t, nil
}

func (s *PostgresStore) GetByClerkOrgID(ctx context.Context, clerkOrgID string) (*Tenant, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, clerk_org_id, name, slug,
		       stripe_customer_id, stripe_subscription_id,
		       tier, status,
		       fly_machine_id, fly_app_name,
		       request_count, request_limit, data_retention_days,
		       created_at, updated_at
		FROM tenants
		WHERE clerk_org_id = $1`, clerkOrgID)

	t, err := scanTenant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("tenant get by clerk org: %w", err)
	}
	return &t, nil
}

func (s *PostgresStore) List(ctx context.Context) ([]Tenant, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, clerk_org_id, name, slug,
		       stripe_customer_id, stripe_subscription_id,
		       tier, status,
		       fly_machine_id, fly_app_name,
		       request_count, request_limit, data_retention_days,
		       created_at, updated_at
		FROM tenants
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("tenant list: %w", err)
	}
	defer rows.Close()

	var result []Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			return nil, fmt.Errorf("tenant list scan: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (s *PostgresStore) Update(ctx context.Context, t *Tenant) error {
	t.UpdatedAt = time.Now()

	ct, err := s.pool.Exec(ctx, `
		UPDATE tenants SET
			clerk_org_id = $2, name = $3, slug = $4,
			stripe_customer_id = $5, stripe_subscription_id = $6,
			tier = $7, status = $8,
			fly_machine_id = $9, fly_app_name = $10,
			request_count = $11, request_limit = $12, data_retention_days = $13,
			updated_at = $14
		WHERE id = $1`,
		t.ID, t.ClerkOrgID, t.Name, t.Slug,
		nullStr(t.StripeCustomerID), nullStr(t.StripeSubscriptionID),
		t.Tier, t.Status,
		nullStr(t.FlyMachineID), nullStr(t.FlyAppName),
		t.RequestCount, t.RequestLimit, t.DataRetentionDays,
		t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("tenant update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("tenant delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) IncrementRequestCount(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `
		UPDATE tenants
		SET request_count = request_count + 1, updated_at = now()
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("tenant increment request count: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) RecordUsage(ctx context.Context, record *UsageRecord) error {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	record.CreatedAt = time.Now()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO usage_records (
			id, tenant_id, period_start, period_end,
			request_count, total_cost, reported_to_stripe,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		record.ID, record.TenantID, record.PeriodStart, record.PeriodEnd,
		record.RequestCount, record.TotalCost, record.ReportedToStripe,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("usage record create: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetUnreportedUsage(ctx context.Context) ([]UsageRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, period_start, period_end,
		       request_count, total_cost, reported_to_stripe,
		       created_at
		FROM usage_records
		WHERE reported_to_stripe = false
		ORDER BY period_start ASC`)
	if err != nil {
		return nil, fmt.Errorf("unreported usage query: %w", err)
	}
	defer rows.Close()

	var result []UsageRecord
	for rows.Next() {
		u, err := scanUsageRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("unreported usage scan: %w", err)
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (s *PostgresStore) MarkUsageReported(ctx context.Context, recordID string) error {
	ct, err := s.pool.Exec(ctx, `
		UPDATE usage_records
		SET reported_to_stripe = true
		WHERE id = $1`, recordID)
	if err != nil {
		return fmt.Errorf("mark usage reported: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanTenant reads one row into a Tenant value.
func scanTenant(s scanner) (Tenant, error) {
	var (
		t                    Tenant
		stripeCustomerID     *string
		stripeSubscriptionID *string
		flyMachineID         *string
		flyAppName           *string
	)

	err := s.Scan(
		&t.ID, &t.ClerkOrgID, &t.Name, &t.Slug,
		&stripeCustomerID, &stripeSubscriptionID,
		&t.Tier, &t.Status,
		&flyMachineID, &flyAppName,
		&t.RequestCount, &t.RequestLimit, &t.DataRetentionDays,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return t, err
	}

	if stripeCustomerID != nil {
		t.StripeCustomerID = *stripeCustomerID
	}
	if stripeSubscriptionID != nil {
		t.StripeSubscriptionID = *stripeSubscriptionID
	}
	if flyMachineID != nil {
		t.FlyMachineID = *flyMachineID
	}
	if flyAppName != nil {
		t.FlyAppName = *flyAppName
	}

	return t, nil
}

// scanUsageRecord reads one row into a UsageRecord value.
func scanUsageRecord(s scanner) (UsageRecord, error) {
	var u UsageRecord
	err := s.Scan(
		&u.ID, &u.TenantID, &u.PeriodStart, &u.PeriodEnd,
		&u.RequestCount, &u.TotalCost, &u.ReportedToStripe,
		&u.CreatedAt,
	)
	return u, err
}

// nullStr returns nil for empty strings, otherwise a pointer to s.
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Compile-time interface check.
var _ Store = (*PostgresStore)(nil)
