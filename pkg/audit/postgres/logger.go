package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neureaux/cloudmock/pkg/audit"
)

// Logger implements audit.Logger against PostgreSQL.
type Logger struct {
	pool *pgxpool.Pool
}

// NewLogger creates an audit Logger backed by the given pool.
func NewLogger(pool *pgxpool.Pool) *Logger {
	return &Logger{pool: pool}
}

// Log inserts an audit entry into the audit_log table.
func (l *Logger) Log(ctx context.Context, entry audit.Entry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	details := []byte("{}")
	if entry.Details != nil {
		var err error
		details, err = json.Marshal(entry.Details)
		if err != nil {
			return fmt.Errorf("audit log marshal details: %w", err)
		}
	}

	_, err := l.pool.Exec(ctx, `
		INSERT INTO audit_log (actor, action, resource, details, timestamp)
		VALUES ($1, $2, $3, $4, $5)`,
		entry.Actor, entry.Action, entry.Resource, details, entry.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("audit log insert: %w", err)
	}
	return nil
}

// Query returns audit entries matching the filter, newest first.
func (l *Logger) Query(ctx context.Context, filter audit.Filter) ([]audit.Entry, error) {
	query := `SELECT id, actor, action, resource, details, timestamp FROM audit_log WHERE true`

	args := []any{}
	n := 1

	if filter.Actor != "" {
		query += fmt.Sprintf(" AND actor = $%d", n)
		args = append(args, filter.Actor)
		n++
	}
	if filter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", n)
		args = append(args, filter.Action)
		n++
	}
	if filter.Resource != "" {
		query += fmt.Sprintf(" AND resource = $%d", n)
		args = append(args, filter.Resource)
		n++
	}

	query += " ORDER BY timestamp DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d", n)
	args = append(args, limit)

	rows, err := l.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("audit log query: %w", err)
	}
	defer rows.Close()

	var results []audit.Entry
	for rows.Next() {
		var e audit.Entry
		var detailsRaw []byte
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Resource, &detailsRaw, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("audit log scan: %w", err)
		}
		if len(detailsRaw) > 0 {
			_ = json.Unmarshal(detailsRaw, &e.Details)
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

// Compile-time interface check.
var _ audit.Logger = (*Logger)(nil)
