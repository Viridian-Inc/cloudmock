package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neureaux/cloudmock/pkg/platform/model"
)

// AppStore handles persistence for App records.
type AppStore struct {
	pool *pgxpool.Pool
}

// NewAppStore creates an AppStore backed by the given pool.
func NewAppStore(pool *pgxpool.Pool) *AppStore {
	return &AppStore{pool: pool}
}

// Create inserts a new app and populates ID and CreatedAt via RETURNING.
func (s *AppStore) Create(ctx context.Context, a *model.App) error {
	const q = `
		INSERT INTO apps (tenant_id, name, slug, endpoint, infra_type, fly_app_name, fly_machine_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`

	row := s.pool.QueryRow(ctx, q,
		a.TenantID,
		a.Name,
		a.Slug,
		a.Endpoint,
		a.InfraType,
		nilIfEmpty(a.FlyAppName),
		nilIfEmpty(a.FlyMachineID),
		a.Status,
	)
	if err := row.Scan(&a.ID, &a.CreatedAt); err != nil {
		return fmt.Errorf("insert app: %w", err)
	}
	return nil
}

// Get fetches an app by primary key.
func (s *AppStore) Get(ctx context.Context, id string) (*model.App, error) {
	const q = `
		SELECT id, tenant_id, name, slug, endpoint, infra_type,
		       COALESCE(fly_app_name, ''), COALESCE(fly_machine_id, ''),
		       status, created_at
		FROM apps
		WHERE id = $1`

	return s.scanOne(s.pool.QueryRow(ctx, q, id))
}

// GetByEndpoint fetches an app by its endpoint value.
func (s *AppStore) GetByEndpoint(ctx context.Context, endpoint string) (*model.App, error) {
	const q = `
		SELECT id, tenant_id, name, slug, endpoint, infra_type,
		       COALESCE(fly_app_name, ''), COALESCE(fly_machine_id, ''),
		       status, created_at
		FROM apps
		WHERE endpoint = $1`

	return s.scanOne(s.pool.QueryRow(ctx, q, endpoint))
}

// ListByTenant returns all apps for the given tenant, oldest-first.
func (s *AppStore) ListByTenant(ctx context.Context, tenantID string) ([]model.App, error) {
	const q = `
		SELECT id, tenant_id, name, slug, endpoint, infra_type,
		       COALESCE(fly_app_name, ''), COALESCE(fly_machine_id, ''),
		       status, created_at
		FROM apps
		WHERE tenant_id = $1
		ORDER BY created_at ASC`

	rows, err := s.pool.Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}
	defer rows.Close()

	var apps []model.App
	for rows.Next() {
		var a model.App
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.Name, &a.Slug, &a.Endpoint, &a.InfraType,
			&a.FlyAppName, &a.FlyMachineID,
			&a.Status, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan app row: %w", err)
		}
		apps = append(apps, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return apps, nil
}

// Update persists mutable fields of the given app.
func (s *AppStore) Update(ctx context.Context, a *model.App) error {
	const q = `
		UPDATE apps
		SET name           = $2,
		    slug           = $3,
		    endpoint       = $4,
		    infra_type     = $5,
		    fly_app_name   = $6,
		    fly_machine_id = $7,
		    status         = $8
		WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q,
		a.ID,
		a.Name,
		a.Slug,
		a.Endpoint,
		a.InfraType,
		nilIfEmpty(a.FlyAppName),
		nilIfEmpty(a.FlyMachineID),
		a.Status,
	)
	if err != nil {
		return fmt.Errorf("update app: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes an app by primary key.
func (s *AppStore) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM apps WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete app: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanOne scans a single app row, mapping pgx.ErrNoRows to ErrNotFound.
func (s *AppStore) scanOne(row pgx.Row) (*model.App, error) {
	var a model.App
	err := row.Scan(
		&a.ID, &a.TenantID, &a.Name, &a.Slug, &a.Endpoint, &a.InfraType,
		&a.FlyAppName, &a.FlyMachineID,
		&a.Status, &a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan app: %w", err)
	}
	return &a, nil
}
