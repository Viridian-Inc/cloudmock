package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock/pkg/incident"
)

// Store implements incident.IncidentStore against PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a Store backed by the given pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Save inserts an Incident into the database. If inc.ID is empty, a new UUID
// is generated and written back to inc.ID.
func (s *Store) Save(ctx context.Context, inc *incident.Incident) error {
	if inc.ID == "" {
		inc.ID = uuid.New().String()
	}

	var deployID *string
	if inc.RelatedDeployID != "" {
		deployID = &inc.RelatedDeployID
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO incidents (
			id, status, severity, title,
			affected_services, affected_tenants,
			alert_count, root_cause, related_deploy_id,
			first_seen, last_seen, resolved_at, owner
		) VALUES (
			$1, $2, $3, $4,
			$5, $6,
			$7, $8, $9,
			$10, $11, $12, $13
		)`,
		inc.ID, inc.Status, inc.Severity, inc.Title,
		inc.AffectedServices, inc.AffectedTenants,
		inc.AlertCount, nullStr(inc.RootCause), deployID,
		inc.FirstSeen, inc.LastSeen, inc.ResolvedAt, nullStr(inc.Owner),
	)
	if err != nil {
		return fmt.Errorf("incident save: %w", err)
	}
	return nil
}

// Get returns the incident with the given ID. Returns incident.ErrNotFound if
// no row exists.
func (s *Store) Get(ctx context.Context, id string) (*incident.Incident, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, status, severity, title,
		       affected_services, affected_tenants,
		       alert_count, root_cause, related_deploy_id,
		       first_seen, last_seen, resolved_at, owner
		FROM incidents
		WHERE id = $1`, id)

	inc, err := scanIncident(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, incident.ErrNotFound
		}
		return nil, fmt.Errorf("incident get: %w", err)
	}
	return &inc, nil
}

// List returns incidents matching the given filter, ordered by first_seen DESC.
func (s *Store) List(ctx context.Context, filter incident.IncidentFilter) ([]incident.Incident, error) {
	query := `
		SELECT id, status, severity, title,
		       affected_services, affected_tenants,
		       alert_count, root_cause, related_deploy_id,
		       first_seen, last_seen, resolved_at, owner
		FROM incidents
		WHERE true`

	args := []any{}
	n := 1

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, filter.Status)
		n++
	}
	if filter.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", n)
		args = append(args, filter.Severity)
		n++
	}
	if filter.Service != "" {
		query += fmt.Sprintf(" AND $%d = ANY(affected_services)", n)
		args = append(args, filter.Service)
		n++
	}

	query += " ORDER BY first_seen DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", n)
		args = append(args, filter.Limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("incident list query: %w", err)
	}
	defer rows.Close()

	var results []incident.Incident
	for rows.Next() {
		inc, err := scanIncident(rows)
		if err != nil {
			return nil, fmt.Errorf("incident list scan: %w", err)
		}
		results = append(results, inc)
	}
	return results, rows.Err()
}

// Update replaces all fields of the incident with the given ID.
func (s *Store) Update(ctx context.Context, inc *incident.Incident) error {
	var deployID *string
	if inc.RelatedDeployID != "" {
		deployID = &inc.RelatedDeployID
	}

	ct, err := s.pool.Exec(ctx, `
		UPDATE incidents SET
			status = $2, severity = $3, title = $4,
			affected_services = $5, affected_tenants = $6,
			alert_count = $7, root_cause = $8, related_deploy_id = $9,
			first_seen = $10, last_seen = $11, resolved_at = $12, owner = $13
		WHERE id = $1`,
		inc.ID, inc.Status, inc.Severity, inc.Title,
		inc.AffectedServices, inc.AffectedTenants,
		inc.AlertCount, nullStr(inc.RootCause), deployID,
		inc.FirstSeen, inc.LastSeen, inc.ResolvedAt, nullStr(inc.Owner),
	)
	if err != nil {
		return fmt.Errorf("incident update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return incident.ErrNotFound
	}
	return nil
}

// FindActiveByKey returns the most recent active or acknowledged incident
// matching the given service, optional deploy ID, and recency threshold.
func (s *Store) FindActiveByKey(ctx context.Context, service, deployID string, since time.Time) (*incident.Incident, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, status, severity, title,
		       affected_services, affected_tenants,
		       alert_count, root_cause, related_deploy_id,
		       first_seen, last_seen, resolved_at, owner
		FROM incidents
		WHERE $1 = ANY(affected_services)
		AND ($2 = '' OR related_deploy_id::text = $2)
		AND last_seen > $3
		AND status IN ('active', 'acknowledged')
		ORDER BY last_seen DESC
		LIMIT 1`, service, deployID, since)

	inc, err := scanIncident(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, incident.ErrNotFound
		}
		return nil, fmt.Errorf("incident find active by key: %w", err)
	}
	return &inc, nil
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanIncident reads one row into an Incident value.
func scanIncident(s scanner) (incident.Incident, error) {
	var (
		inc        incident.Incident
		rootCause  *string
		deployID   *string
		resolvedAt *time.Time
		owner      *string
	)

	err := s.Scan(
		&inc.ID, &inc.Status, &inc.Severity, &inc.Title,
		&inc.AffectedServices, &inc.AffectedTenants,
		&inc.AlertCount, &rootCause, &deployID,
		&inc.FirstSeen, &inc.LastSeen, &resolvedAt, &owner,
	)
	if err != nil {
		return inc, err
	}

	if rootCause != nil {
		inc.RootCause = *rootCause
	}
	if deployID != nil {
		inc.RelatedDeployID = *deployID
	}
	if resolvedAt != nil {
		inc.ResolvedAt = resolvedAt
	}
	if owner != nil {
		inc.Owner = *owner
	}

	// Ensure nil slices from the DB become empty slices.
	if inc.AffectedServices == nil {
		inc.AffectedServices = []string{}
	}
	if inc.AffectedTenants == nil {
		inc.AffectedTenants = []string{}
	}

	return inc, nil
}

// nullStr returns nil for empty strings, otherwise a pointer to s.
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// AddComment inserts a comment for the given incident.
func (s *Store) AddComment(incidentID string, comment incident.Comment) error {
	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	_, err := s.pool.Exec(context.Background(),
		`INSERT INTO incident_comments (id, incident_id, author, body, mentions, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		comment.ID, incidentID, comment.Author, comment.Body, comment.Mentions, comment.CreatedAt,
	)
	return err
}

// GetComments returns all comments for the given incident, ordered by creation time.
func (s *Store) GetComments(incidentID string) ([]incident.Comment, error) {
	rows, err := s.pool.Query(context.Background(),
		`SELECT id, incident_id, author, body, mentions, created_at
		 FROM incident_comments WHERE incident_id = $1 ORDER BY created_at ASC`,
		incidentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []incident.Comment
	for rows.Next() {
		var c incident.Comment
		if err := rows.Scan(&c.ID, &c.IncidentID, &c.Author, &c.Body, &c.Mentions, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// Compile-time interface check.
var _ incident.IncidentStore = (*Store)(nil)
