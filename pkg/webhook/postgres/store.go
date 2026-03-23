package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neureaux/cloudmock/pkg/webhook"
)

// Store implements webhook.Store against PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a Store backed by the given pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Save inserts or replaces a webhook config. If cfg.ID is empty, a new UUID is
// generated and written back to cfg.ID.
func (s *Store) Save(ctx context.Context, cfg *webhook.Config) error {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}

	headers, err := json.Marshal(cfg.Headers)
	if err != nil {
		return fmt.Errorf("webhook save: marshal headers: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO webhooks (id, url, type, events, headers, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			url     = EXCLUDED.url,
			type    = EXCLUDED.type,
			events  = EXCLUDED.events,
			headers = EXCLUDED.headers,
			active  = EXCLUDED.active
	`,
		cfg.ID, cfg.URL, cfg.Type, cfg.Events, headers, cfg.Active,
	)
	if err != nil {
		return fmt.Errorf("webhook save: %w", err)
	}
	return nil
}

// Get returns the webhook config with the given ID. Returns webhook.ErrNotFound
// if no row exists.
func (s *Store) Get(ctx context.Context, id string) (*webhook.Config, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, url, type, events, headers, active
		FROM webhooks
		WHERE id = $1`, id)

	cfg, err := scanConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, webhook.ErrNotFound
		}
		return nil, fmt.Errorf("webhook get: %w", err)
	}
	return &cfg, nil
}

// List returns all stored webhook configs.
func (s *Store) List(ctx context.Context) ([]webhook.Config, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, url, type, events, headers, active
		FROM webhooks
		ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("webhook list: %w", err)
	}
	defer rows.Close()

	var result []webhook.Config
	for rows.Next() {
		cfg, err := scanConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("webhook list scan: %w", err)
		}
		result = append(result, cfg)
	}
	return result, rows.Err()
}

// Delete removes the webhook config with the given ID. Returns webhook.ErrNotFound
// if no row exists.
func (s *Store) Delete(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM webhooks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("webhook delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return webhook.ErrNotFound
	}
	return nil
}

// ListByEvent returns all active webhook configs subscribed to the given event.
func (s *Store) ListByEvent(ctx context.Context, event string) ([]webhook.Config, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, url, type, events, headers, active
		FROM webhooks
		WHERE active = true AND $1 = ANY(events)
		ORDER BY created_at`, event)
	if err != nil {
		return nil, fmt.Errorf("webhook list by event: %w", err)
	}
	defer rows.Close()

	var result []webhook.Config
	for rows.Next() {
		cfg, err := scanConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("webhook list by event scan: %w", err)
		}
		result = append(result, cfg)
	}
	return result, rows.Err()
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanConfig reads one row into a webhook.Config value.
func scanConfig(s scanner) (webhook.Config, error) {
	var (
		cfg        webhook.Config
		events     []string
		headersRaw []byte
	)

	err := s.Scan(&cfg.ID, &cfg.URL, &cfg.Type, &events, &headersRaw, &cfg.Active)
	if err != nil {
		return cfg, err
	}

	cfg.Events = events
	if cfg.Events == nil {
		cfg.Events = []string{}
	}

	if len(headersRaw) > 0 {
		if err := json.Unmarshal(headersRaw, &cfg.Headers); err != nil {
			return cfg, fmt.Errorf("unmarshal headers: %w", err)
		}
	}
	if cfg.Headers == nil {
		cfg.Headers = map[string]string{}
	}

	return cfg, nil
}

// Compile-time interface check.
var _ webhook.Store = (*Store)(nil)
