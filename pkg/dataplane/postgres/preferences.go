package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
)

// PreferenceStore implements dataplane.PreferenceStore against PostgreSQL.
type PreferenceStore struct {
	pool *pgxpool.Pool
}

// NewPreferenceStore creates a PreferenceStore backed by the given pool.
func NewPreferenceStore(pool *pgxpool.Pool) *PreferenceStore {
	return &PreferenceStore{pool: pool}
}

// Get returns the value for the given namespace and key.
func (s *PreferenceStore) Get(ctx context.Context, namespace, key string) (json.RawMessage, error) {
	var value json.RawMessage
	err := s.pool.QueryRow(ctx,
		`SELECT value FROM preferences WHERE namespace = $1 AND key = $2`,
		namespace, key,
	).Scan(&value)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, dataplane.ErrNotFound
		}
		return nil, fmt.Errorf("preference get: %w", err)
	}
	return value, nil
}

// Set upserts a value for the given namespace and key.
func (s *PreferenceStore) Set(ctx context.Context, namespace, key string, value json.RawMessage) error {
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO preferences (namespace, key, value, updated_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (namespace, key) DO UPDATE SET value = $3, updated_at = now()`,
		namespace, key, value,
	); err != nil {
		return fmt.Errorf("preference set: %w", err)
	}
	return nil
}

// Delete removes a key from the given namespace.
func (s *PreferenceStore) Delete(ctx context.Context, namespace, key string) error {
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM preferences WHERE namespace = $1 AND key = $2`,
		namespace, key,
	); err != nil {
		return fmt.Errorf("preference delete: %w", err)
	}
	return nil
}

// ListByNamespace returns all key-value pairs for the given namespace.
func (s *PreferenceStore) ListByNamespace(ctx context.Context, namespace string) (map[string]json.RawMessage, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT key, value FROM preferences WHERE namespace = $1`,
		namespace,
	)
	if err != nil {
		return nil, fmt.Errorf("preference list: %w", err)
	}
	defer rows.Close()

	result := make(map[string]json.RawMessage)
	for rows.Next() {
		var key string
		var value json.RawMessage
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("preference list scan: %w", err)
		}
		result[key] = value
	}
	return result, rows.Err()
}

// Compile-time interface check.
var _ dataplane.PreferenceStore = (*PreferenceStore)(nil)
