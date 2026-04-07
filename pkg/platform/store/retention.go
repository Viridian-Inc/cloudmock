package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock/pkg/platform/model"
)

// RetentionStore handles persistence for DataRetention records.
type RetentionStore struct {
	pool *pgxpool.Pool
}

// NewRetentionStore creates a RetentionStore backed by the given pool.
func NewRetentionStore(pool *pgxpool.Pool) *RetentionStore {
	return &RetentionStore{pool: pool}
}

// Upsert inserts or updates a data retention policy for (tenantID, resourceType).
func (s *RetentionStore) Upsert(ctx context.Context, tenantID, resourceType string, days int) error {
	const q = `
		INSERT INTO data_retention (tenant_id, resource_type, retention_days)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, resource_type)
		DO UPDATE SET retention_days = EXCLUDED.retention_days,
		              updated_at     = now()`

	if _, err := s.pool.Exec(ctx, q, tenantID, resourceType, days); err != nil {
		return fmt.Errorf("upsert data retention: %w", err)
	}
	return nil
}

// GetByTenant returns all data retention policies for the given tenant.
func (s *RetentionStore) GetByTenant(ctx context.Context, tenantID string) ([]model.DataRetention, error) {
	const q = `
		SELECT id, tenant_id, resource_type, retention_days, updated_at
		FROM data_retention
		WHERE tenant_id = $1
		ORDER BY resource_type ASC`

	rows, err := s.pool.Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get data retention by tenant: %w", err)
	}
	defer rows.Close()

	var policies []model.DataRetention
	for rows.Next() {
		var p model.DataRetention
		if err := rows.Scan(&p.ID, &p.TenantID, &p.ResourceType, &p.RetentionDays, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan data retention row: %w", err)
		}
		policies = append(policies, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return policies, nil
}
