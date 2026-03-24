package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// ConfigStore implements dataplane.ConfigStore against PostgreSQL.
type ConfigStore struct {
	pool *pgxpool.Pool
}

// NewConfigStore creates a ConfigStore backed by the given pool.
func NewConfigStore(pool *pgxpool.Pool) *ConfigStore {
	return &ConfigStore{pool: pool}
}

// GetConfig assembles a Config by reading all key/value pairs from the config table.
func (s *ConfigStore) GetConfig(ctx context.Context) (*config.Config, error) {
	rows, err := s.pool.Query(ctx, `SELECT key, value FROM config`)
	if err != nil {
		return nil, fmt.Errorf("config get query: %w", err)
	}
	defer rows.Close()

	kv := make(map[string]json.RawMessage)
	for rows.Next() {
		var key string
		var value json.RawMessage
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("config get scan: %w", err)
		}
		kv[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(kv) == 0 {
		return nil, dataplane.ErrNotFound
	}

	// Marshal all k/v into a single JSON object, then unmarshal into Config.
	blob, err := json.Marshal(kv)
	if err != nil {
		return nil, fmt.Errorf("config marshal kv: %w", err)
	}

	// The kv map has structure like {"region": "\"us-east-1\"", "slo": {...}}.
	// We need to reconstruct: iterate keys and build a flat config JSON.
	flat := make(map[string]json.RawMessage)
	for k, v := range kv {
		flat[k] = v
	}
	blob, err = json.Marshal(flat)
	if err != nil {
		return nil, fmt.Errorf("config marshal flat: %w", err)
	}

	cfg := config.Default()
	if err := json.Unmarshal(blob, cfg); err != nil {
		return nil, fmt.Errorf("config unmarshal: %w", err)
	}
	return cfg, nil
}

// SetConfig stores the config as individual key/value pairs using upsert.
func (s *ConfigStore) SetConfig(ctx context.Context, cfg *config.Config) error {
	blob, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("config marshal: %w", err)
	}

	var kv map[string]json.RawMessage
	if err := json.Unmarshal(blob, &kv); err != nil {
		return fmt.Errorf("config unmarshal to kv: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("config set begin: %w", err)
	}
	defer tx.Rollback(ctx)

	for k, v := range kv {
		if _, err := tx.Exec(ctx,
			`INSERT INTO config (key, value, updated_at)
			 VALUES ($1, $2, now())
			 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = now()`,
			k, v,
		); err != nil {
			return fmt.Errorf("config set key %s: %w", k, err)
		}
	}

	return tx.Commit(ctx)
}

// ListDeploys returns deploy events with optional service filter and limit.
func (s *ConfigStore) ListDeploys(ctx context.Context, filter dataplane.DeployFilter) ([]dataplane.DeployEvent, error) {
	query := `SELECT id, service, version, commit_sha, author, description, deployed_at, metadata
		 FROM deploys`
	var args []any
	argN := 1

	if filter.Service != "" {
		query += fmt.Sprintf(` WHERE service = $%d`, argN)
		args = append(args, filter.Service)
		argN++
	}
	query += ` ORDER BY deployed_at DESC`
	if filter.Limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, argN)
		args = append(args, filter.Limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("deploys list query: %w", err)
	}
	defer rows.Close()

	var deploys []dataplane.DeployEvent
	for rows.Next() {
		var d dataplane.DeployEvent
		var commitSHA, author, description *string
		var metadata json.RawMessage
		if err := rows.Scan(&d.ID, &d.Service, &d.Version, &commitSHA, &author, &description, &d.DeployedAt, &metadata); err != nil {
			return nil, fmt.Errorf("deploys list scan: %w", err)
		}
		if commitSHA != nil {
			d.CommitSHA = *commitSHA
		}
		if author != nil {
			d.Author = *author
		}
		if description != nil {
			d.Description = *description
		}
		if metadata != nil {
			_ = json.Unmarshal(metadata, &d.Metadata)
		}
		deploys = append(deploys, d)
	}
	return deploys, rows.Err()
}

// AddDeploy inserts a deploy event. The referenced service is upserted first
// to satisfy the FK constraint.
func (s *ConfigStore) AddDeploy(ctx context.Context, deploy dataplane.DeployEvent) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("deploy add begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Ensure service exists for FK.
	if _, err := tx.Exec(ctx,
		`INSERT INTO services (name, service_type) VALUES ($1, 'unknown')
		 ON CONFLICT (name) DO NOTHING`,
		deploy.Service,
	); err != nil {
		return fmt.Errorf("deploy ensure service: %w", err)
	}

	id := deploy.ID
	if id == "" {
		id = uuid.New().String()
	}
	deployedAt := deploy.DeployedAt
	if deployedAt.IsZero() {
		deployedAt = time.Now()
	}

	metadata, _ := json.Marshal(deploy.Metadata)

	if _, err := tx.Exec(ctx,
		`INSERT INTO deploys (id, service, version, commit_sha, author, description, deployed_at, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, deploy.Service, deploy.Version, nilIfEmpty(deploy.CommitSHA),
		nilIfEmpty(deploy.Author), nilIfEmpty(deploy.Description), deployedAt, metadata,
	); err != nil {
		return fmt.Errorf("deploy insert: %w", err)
	}

	return tx.Commit(ctx)
}

// ListViews returns all saved views ordered by creation time.
func (s *ConfigStore) ListViews(ctx context.Context) ([]dataplane.SavedView, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, filters, created_by, created_at FROM saved_views ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("views list query: %w", err)
	}
	defer rows.Close()

	var views []dataplane.SavedView
	for rows.Next() {
		var v dataplane.SavedView
		var filters json.RawMessage
		var createdBy *string
		if err := rows.Scan(&v.ID, &v.Name, &filters, &createdBy, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("views list scan: %w", err)
		}
		if createdBy != nil {
			v.CreatedBy = *createdBy
		}
		if filters != nil {
			_ = json.Unmarshal(filters, &v.Filters)
		}
		views = append(views, v)
	}
	return views, rows.Err()
}

// SaveView upserts a saved view.
func (s *ConfigStore) SaveView(ctx context.Context, view dataplane.SavedView) error {
	filters, _ := json.Marshal(view.Filters)
	createdAt := view.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	if _, err := s.pool.Exec(ctx,
		`INSERT INTO saved_views (id, name, filters, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE SET name = $2, filters = $3, created_by = $4`,
		view.ID, view.Name, filters, nilIfEmpty(view.CreatedBy), createdAt,
	); err != nil {
		return fmt.Errorf("view save: %w", err)
	}
	return nil
}

// DeleteView removes a saved view by ID.
func (s *ConfigStore) DeleteView(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM saved_views WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("view delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return dataplane.ErrNotFound
	}
	return nil
}

// ListServices returns all services ordered by name.
func (s *ConfigStore) ListServices(ctx context.Context) ([]dataplane.ServiceEntry, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT name, service_type, group_name, description, owner, repo_url
		 FROM services ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("services list query: %w", err)
	}
	defer rows.Close()

	var services []dataplane.ServiceEntry
	for rows.Next() {
		var svc dataplane.ServiceEntry
		var groupName, description, owner, repoURL *string
		if err := rows.Scan(&svc.Name, &svc.ServiceType, &groupName, &description, &owner, &repoURL); err != nil {
			return nil, fmt.Errorf("services list scan: %w", err)
		}
		if groupName != nil {
			svc.GroupName = *groupName
		}
		if description != nil {
			svc.Description = *description
		}
		if owner != nil {
			svc.Owner = *owner
		}
		if repoURL != nil {
			svc.RepoURL = *repoURL
		}
		services = append(services, svc)
	}
	return services, rows.Err()
}

// UpsertService inserts or updates a service entry.
func (s *ConfigStore) UpsertService(ctx context.Context, svc dataplane.ServiceEntry) error {
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO services (name, service_type, group_name, description, owner, repo_url, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, now())
		 ON CONFLICT (name) DO UPDATE SET
		   service_type = $2, group_name = $3, description = $4, owner = $5, repo_url = $6, updated_at = now()`,
		svc.Name, svc.ServiceType, nilIfEmpty(svc.GroupName),
		nilIfEmpty(svc.Description), nilIfEmpty(svc.Owner), nilIfEmpty(svc.RepoURL),
	); err != nil {
		return fmt.Errorf("service upsert: %w", err)
	}
	return nil
}

// nilIfEmpty returns nil if the string is empty, otherwise a pointer to it.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Compile-time interface check.
var _ dataplane.ConfigStore = (*ConfigStore)(nil)
