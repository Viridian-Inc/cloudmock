package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock/pkg/platform/model"
)

// UsageStore handles persistence for UsageRecord records.
type UsageStore struct {
	pool *pgxpool.Pool
}

// NewUsageStore creates a UsageStore backed by the given pool.
func NewUsageStore(pool *pgxpool.Pool) *UsageStore {
	return &UsageStore{pool: pool}
}

// beginningOfMonth returns midnight on the first day of t's month.
func beginningOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// IncrementRequestCount upserts a usage record for the current hourly period.
// It first tries to UPDATE an existing row; if no row exists it INSERTs one.
func (s *UsageStore) IncrementRequestCount(ctx context.Context, tenantID, appID string) error {
	now := time.Now().UTC()
	// Truncate to hour.
	periodStart := now.Truncate(time.Hour)
	periodEnd := periodStart.Add(time.Hour)

	const updateQ = `
		UPDATE usage_records
		SET request_count = request_count + 1
		WHERE tenant_id = $1
		  AND app_id = $2
		  AND period_start = $3`

	tag, err := s.pool.Exec(ctx, updateQ, tenantID, appID, periodStart)
	if err != nil {
		return fmt.Errorf("update usage record: %w", err)
	}

	if tag.RowsAffected() > 0 {
		return nil
	}

	const insertQ = `
		INSERT INTO usage_records (tenant_id, app_id, period_start, period_end, request_count)
		VALUES ($1, $2, $3, $4, 1)
		ON CONFLICT DO NOTHING`

	insertTag, err := s.pool.Exec(ctx, insertQ, tenantID, appID, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("insert usage record: %w", err)
	}

	// If the INSERT was a no-op (concurrent insert won the race), retry the UPDATE.
	if insertTag.RowsAffected() == 0 {
		_, err = s.pool.Exec(ctx, updateQ, tenantID, appID, periodStart)
		if err != nil {
			return fmt.Errorf("retry update usage record: %w", err)
		}
	}

	return nil
}

// GetCurrentPeriodCount returns the sum of request_count for the current calendar month.
func (s *UsageStore) GetCurrentPeriodCount(ctx context.Context, tenantID string) (int64, error) {
	monthStart := beginningOfMonth(time.Now().UTC())

	const q = `
		SELECT COALESCE(SUM(request_count), 0)
		FROM usage_records
		WHERE tenant_id = $1
		  AND period_start >= $2`

	var count int64
	if err := s.pool.QueryRow(ctx, q, tenantID, monthStart).Scan(&count); err != nil {
		return 0, fmt.Errorf("get current period count: %w", err)
	}
	return count, nil
}

// GetByTenant returns usage records for a tenant within [start, end).
func (s *UsageStore) GetByTenant(ctx context.Context, tenantID string, start, end time.Time) ([]model.UsageRecord, error) {
	const q = `
		SELECT id, tenant_id, app_id, period_start, period_end,
		       request_count, reported_to_stripe, created_at
		FROM usage_records
		WHERE tenant_id = $1
		  AND period_start >= $2
		  AND period_start < $3
		ORDER BY period_start ASC`

	rows, err := s.pool.Query(ctx, q, tenantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("get usage by tenant: %w", err)
	}
	defer rows.Close()

	return scanUsageRows(rows)
}

// GetUnreported returns all usage records that have not been reported to Stripe
// and whose period_end is in the past.
func (s *UsageStore) GetUnreported(ctx context.Context) ([]model.UsageRecord, error) {
	const q = `
		SELECT id, tenant_id, app_id, period_start, period_end,
		       request_count, reported_to_stripe, created_at
		FROM usage_records
		WHERE NOT reported_to_stripe
		  AND period_end < now()
		ORDER BY period_end ASC`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("get unreported usage records: %w", err)
	}
	defer rows.Close()

	return scanUsageRows(rows)
}

// MarkReported sets reported_to_stripe = true for the given record.
func (s *UsageStore) MarkReported(ctx context.Context, id string) error {
	const q = `UPDATE usage_records SET reported_to_stripe = true WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("mark reported: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PurgeOlderThan deletes reported usage records older than before for the given tenant.
// Returns the number of rows deleted.
func (s *UsageStore) PurgeOlderThan(ctx context.Context, tenantID string, before time.Time) (int64, error) {
	const q = `
		DELETE FROM usage_records
		WHERE tenant_id = $1
		  AND reported_to_stripe = true
		  AND period_end < $2`

	tag, err := s.pool.Exec(ctx, q, tenantID, before)
	if err != nil {
		return 0, fmt.Errorf("purge usage records: %w", err)
	}
	return tag.RowsAffected(), nil
}

// scanUsageRows scans pgx rows into a slice of UsageRecord.
func scanUsageRows(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]model.UsageRecord, error) {
	var records []model.UsageRecord
	for rows.Next() {
		var r model.UsageRecord
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.AppID, &r.PeriodStart, &r.PeriodEnd,
			&r.RequestCount, &r.ReportedToStripe, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan usage record: %w", err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return records, nil
}
