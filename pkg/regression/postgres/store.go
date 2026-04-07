package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock/pkg/regression"
)

// Store implements regression.RegressionStore against PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a Store backed by the given pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Save inserts a Regression into the database. If r.ID is empty, a new UUID is
// generated and written back to r.ID.
func (s *Store) Save(ctx context.Context, r *regression.Regression) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}

	var deployID *string
	if r.DeployID != "" {
		deployID = &r.DeployID
	}
	var tenantID *string
	if r.TenantID != "" {
		tenantID = &r.TenantID
	}
	var action *string
	if r.Action != "" {
		action = &r.Action
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO regressions (
			id, algorithm, severity, confidence,
			service, action, deploy_id, tenant_id,
			title, before_value, after_value, change_percent,
			sample_size, detected_at,
			window_before, window_after,
			status, resolved_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14,
			tstzrange($15::timestamptz, $16::timestamptz, '[)'),
			tstzrange($17::timestamptz, $18::timestamptz, '[)'),
			$19, $20
		)`,
		r.ID, string(r.Algorithm), string(r.Severity), r.Confidence,
		r.Service, action, deployID, tenantID,
		r.Title, r.BeforeValue, r.AfterValue, r.ChangePercent,
		r.SampleSize, r.DetectedAt,
		r.WindowBefore.Start, r.WindowBefore.End,
		r.WindowAfter.Start, r.WindowAfter.End,
		r.Status, r.ResolvedAt,
	)
	if err != nil {
		return fmt.Errorf("regression save: %w", err)
	}
	return nil
}

// List returns regressions matching the given filter, ordered by detected_at DESC.
func (s *Store) List(ctx context.Context, filter regression.RegressionFilter) ([]regression.Regression, error) {
	query := `
		SELECT id, algorithm, severity, confidence,
		       service, action, deploy_id, tenant_id,
		       title, before_value, after_value, change_percent,
		       sample_size, detected_at,
		       window_before, window_after,
		       status, resolved_at
		FROM regressions
		WHERE true`

	args := []any{}
	n := 1

	if filter.Service != "" {
		query += fmt.Sprintf(" AND service = $%d", n)
		args = append(args, filter.Service)
		n++
	}
	if filter.DeployID != "" {
		query += fmt.Sprintf(" AND deploy_id = $%d", n)
		args = append(args, filter.DeployID)
		n++
	}
	if filter.Algorithm != "" {
		query += fmt.Sprintf(" AND algorithm = $%d", n)
		args = append(args, string(filter.Algorithm))
		n++
	}
	if filter.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", n)
		args = append(args, string(filter.Severity))
		n++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, filter.Status)
		n++
	}

	query += " ORDER BY detected_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", n)
		args = append(args, filter.Limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("regression list query: %w", err)
	}
	defer rows.Close()

	var results []regression.Regression
	for rows.Next() {
		r, err := scanRegression(rows)
		if err != nil {
			return nil, fmt.Errorf("regression list scan: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// Get returns the regression with the given ID. Returns regression.ErrNotFound
// if no row exists.
func (s *Store) Get(ctx context.Context, id string) (*regression.Regression, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, algorithm, severity, confidence,
		       service, action, deploy_id, tenant_id,
		       title, before_value, after_value, change_percent,
		       sample_size, detected_at,
		       window_before, window_after,
		       status, resolved_at
		FROM regressions
		WHERE id = $1`, id)

	r, err := scanRegression(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, regression.ErrNotFound
		}
		return nil, fmt.Errorf("regression get: %w", err)
	}
	return &r, nil
}

// UpdateStatus sets the status of the regression with the given ID. If the
// new status is "resolved" or "dismissed", resolved_at is set to now().
func (s *Store) UpdateStatus(ctx context.Context, id string, status string) error {
	var tag pgtype.Text
	_ = tag

	var q string
	if status == "resolved" || status == "dismissed" {
		q = `UPDATE regressions SET status = $1, resolved_at = now() WHERE id = $2`
	} else {
		q = `UPDATE regressions SET status = $1 WHERE id = $2`
	}

	ct, err := s.pool.Exec(ctx, q, status, id)
	if err != nil {
		return fmt.Errorf("regression update status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return regression.ErrNotFound
	}
	return nil
}

// ActiveForDeploy returns all active regressions for the given deploy ID.
func (s *Store) ActiveForDeploy(ctx context.Context, deployID string) ([]regression.Regression, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, algorithm, severity, confidence,
		       service, action, deploy_id, tenant_id,
		       title, before_value, after_value, change_percent,
		       sample_size, detected_at,
		       window_before, window_after,
		       status, resolved_at
		FROM regressions
		WHERE deploy_id = $1 AND status = 'active'
		ORDER BY detected_at DESC`, deployID)
	if err != nil {
		return nil, fmt.Errorf("regression active for deploy query: %w", err)
	}
	defer rows.Close()

	var results []regression.Regression
	for rows.Next() {
		r, err := scanRegression(rows)
		if err != nil {
			return nil, fmt.Errorf("regression active for deploy scan: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanRegression reads one row into a Regression value.
func scanRegression(s scanner) (regression.Regression, error) {
	var (
		r          regression.Regression
		algorithm  string
		severity   string
		action     *string
		deployID   *string
		tenantID   *string
		windowBef  pgtype.Range[pgtype.Timestamptz]
		windowAft  pgtype.Range[pgtype.Timestamptz]
		resolvedAt *time.Time
	)

	err := s.Scan(
		&r.ID, &algorithm, &severity, &r.Confidence,
		&r.Service, &action, &deployID, &tenantID,
		&r.Title, &r.BeforeValue, &r.AfterValue, &r.ChangePercent,
		&r.SampleSize, &r.DetectedAt,
		&windowBef, &windowAft,
		&r.Status, &resolvedAt,
	)
	if err != nil {
		return r, err
	}

	r.Algorithm = regression.AlgorithmType(algorithm)
	r.Severity = regression.Severity(severity)

	if action != nil {
		r.Action = *action
	}
	if deployID != nil {
		r.DeployID = *deployID
	}
	if tenantID != nil {
		r.TenantID = *tenantID
	}
	if resolvedAt != nil {
		r.ResolvedAt = resolvedAt
	}

	if windowBef.Lower.Valid {
		r.WindowBefore.Start = windowBef.Lower.Time
	}
	if windowBef.Upper.Valid {
		r.WindowBefore.End = windowBef.Upper.Time
	}
	if windowAft.Lower.Valid {
		r.WindowAfter.Start = windowAft.Lower.Time
	}
	if windowAft.Upper.Valid {
		r.WindowAfter.End = windowAft.Upper.Time
	}

	return r, nil
}

// Compile-time interface check.
var _ regression.RegressionStore = (*Store)(nil)
