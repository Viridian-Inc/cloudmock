package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
)

// TopologyStore implements dataplane.TopologyStore against PostgreSQL.
type TopologyStore struct {
	pool *pgxpool.Pool
}

// NewTopologyStore creates a TopologyStore backed by the given pool.
func NewTopologyStore(pool *pgxpool.Pool) *TopologyStore {
	return &TopologyStore{pool: pool}
}

// GetTopology returns the full topology graph from services and edges.
func (s *TopologyStore) GetTopology(ctx context.Context) (*dataplane.TopologyGraph, error) {
	// Query nodes.
	nodeRows, err := s.pool.Query(ctx,
		`SELECT name, service_type, group_name FROM services ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("topology nodes query: %w", err)
	}
	defer nodeRows.Close()

	var nodes []dataplane.TopologyNode
	for nodeRows.Next() {
		var n dataplane.TopologyNode
		var group *string
		if err := nodeRows.Scan(&n.Name, &n.ServiceType, &group); err != nil {
			return nil, fmt.Errorf("topology nodes scan: %w", err)
		}
		if group != nil {
			n.Group = *group
		}
		nodes = append(nodes, n)
	}
	if err := nodeRows.Err(); err != nil {
		return nil, err
	}

	// Query edges.
	edgeRows, err := s.pool.Query(ctx,
		`SELECT source_service, target_service, edge_type, request_count
		 FROM topology_edges ORDER BY first_seen`)
	if err != nil {
		return nil, fmt.Errorf("topology edges query: %w", err)
	}
	defer edgeRows.Close()

	var edges []dataplane.ObservedEdge
	for edgeRows.Next() {
		var e dataplane.ObservedEdge
		if err := edgeRows.Scan(&e.Source, &e.Target, &e.EdgeType, &e.RequestCount); err != nil {
			return nil, fmt.Errorf("topology edges scan: %w", err)
		}
		edges = append(edges, e)
	}
	if err := edgeRows.Err(); err != nil {
		return nil, err
	}

	return &dataplane.TopologyGraph{Nodes: nodes, Edges: edges}, nil
}

// RecordEdge upserts an edge between two services. Source and target services
// are created automatically to satisfy FK constraints.
func (s *TopologyStore) RecordEdge(ctx context.Context, edge dataplane.ObservedEdge) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("topology record begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Ensure both services exist.
	for _, svc := range []string{edge.Source, edge.Target} {
		if _, err := tx.Exec(ctx,
			`INSERT INTO services (name, service_type) VALUES ($1, 'unknown')
			 ON CONFLICT (name) DO NOTHING`, svc,
		); err != nil {
			return fmt.Errorf("topology ensure service %s: %w", svc, err)
		}
	}

	now := time.Now()
	if _, err := tx.Exec(ctx,
		`INSERT INTO topology_edges (source_service, target_service, edge_type, first_seen, last_seen, request_count)
		 VALUES ($1, $2, $3, $4, $4, 1)
		 ON CONFLICT (source_service, target_service, edge_type)
		 DO UPDATE SET last_seen = $4, request_count = topology_edges.request_count + 1`,
		edge.Source, edge.Target, edge.EdgeType, now,
	); err != nil {
		return fmt.Errorf("topology upsert edge: %w", err)
	}

	return tx.Commit(ctx)
}

// Upstream returns all services that have an edge targeting the given service.
func (s *TopologyStore) Upstream(ctx context.Context, service string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT source_service FROM topology_edges WHERE target_service = $1`,
		service)
	if err != nil {
		return nil, fmt.Errorf("topology upstream query: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("topology upstream scan: %w", err)
		}
		result = append(result, name)
	}
	return result, rows.Err()
}

// Downstream returns all services that the given service calls.
func (s *TopologyStore) Downstream(ctx context.Context, service string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT target_service FROM topology_edges WHERE source_service = $1`,
		service)
	if err != nil {
		return nil, fmt.Errorf("topology downstream query: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("topology downstream scan: %w", err)
		}
		result = append(result, name)
	}
	return result, rows.Err()
}

// Compile-time interface check.
var _ dataplane.TopologyStore = (*TopologyStore)(nil)
