package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock/pkg/platform/model"
)

// AuditFilter specifies filter criteria for audit log queries.
type AuditFilter struct {
	TenantID     string
	ActorID      string
	Action       string
	ResourceType string
	Limit        int
	Offset       int
}

// AuditStore handles persistence for AuditEntry records.
type AuditStore struct {
	pool *pgxpool.Pool
}

// NewAuditStore creates an AuditStore backed by the given pool.
func NewAuditStore(pool *pgxpool.Pool) *AuditStore {
	return &AuditStore{pool: pool}
}

// buildAuditWhere builds a WHERE clause and positional args from an AuditFilter.
// Returns ("", nil) when no filters are set.
func buildAuditWhere(f AuditFilter) (string, []any) {
	var clauses []string
	var args []any
	n := 1

	if f.TenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", n))
		args = append(args, f.TenantID)
		n++
	}
	if f.ActorID != "" {
		clauses = append(clauses, fmt.Sprintf("actor_id = $%d", n))
		args = append(args, f.ActorID)
		n++
	}
	if f.Action != "" {
		clauses = append(clauses, fmt.Sprintf("action = $%d", n))
		args = append(args, f.Action)
		n++
	}
	if f.ResourceType != "" {
		clauses = append(clauses, fmt.Sprintf("resource_type = $%d", n))
		args = append(args, f.ResourceType)
		n++
	}
	_ = n // suppress unused warning if no more params needed

	if len(clauses) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

// Append inserts a new audit log entry (append-only by design).
// Metadata is marshalled to JSONB.
func (s *AuditStore) Append(ctx context.Context, e *model.AuditEntry) error {
	var metaJSON []byte
	var err error
	if e.Metadata != nil {
		metaJSON, err = json.Marshal(e.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	const q = `
		INSERT INTO audit_log
			(tenant_id, actor_id, actor_type, action, resource_type, resource_id,
			 ip_address, user_agent, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	row := s.pool.QueryRow(ctx, q,
		e.TenantID,
		e.ActorID,
		e.ActorType,
		e.Action,
		e.ResourceType,
		e.ResourceID,
		e.IPAddress,
		e.UserAgent,
		metaJSON,
	)
	if err := row.Scan(&e.ID, &e.CreatedAt); err != nil {
		return fmt.Errorf("insert audit entry: %w", err)
	}
	return nil
}

// Query returns audit log entries matching the filter, ordered newest-first.
func (s *AuditStore) Query(ctx context.Context, f AuditFilter) ([]model.AuditEntry, error) {
	where, args := buildAuditWhere(f)

	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := f.Offset

	q := fmt.Sprintf(`
		SELECT id, tenant_id, actor_id, actor_type, action, resource_type, resource_id,
		       ip_address, user_agent, COALESCE(metadata, 'null'::jsonb), created_at
		FROM audit_log
		%s
		ORDER BY created_at DESC
		LIMIT %d OFFSET %d`, where, limit, offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit log: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditEntry
	for rows.Next() {
		var e model.AuditEntry
		var metaJSON []byte
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.ActorID, &e.ActorType, &e.Action,
			&e.ResourceType, &e.ResourceID,
			&e.IPAddress, &e.UserAgent, &metaJSON,
			&e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		if len(metaJSON) > 0 && string(metaJSON) != "null" {
			if err := json.Unmarshal(metaJSON, &e.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return entries, nil
}

// Count returns the number of audit log entries matching the filter.
func (s *AuditStore) Count(ctx context.Context, f AuditFilter) (int64, error) {
	where, args := buildAuditWhere(f)

	q := fmt.Sprintf(`SELECT COUNT(*) FROM audit_log %s`, where)

	var count int64
	if err := s.pool.QueryRow(ctx, q, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count audit log: %w", err)
	}
	return count, nil
}
