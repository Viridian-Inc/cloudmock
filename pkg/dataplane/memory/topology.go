package memory

import (
	"context"
	"sync"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// TopologyStore satisfies the dataplane.TopologyStore interface using
// in-memory slices for nodes and edges.
type TopologyStore struct {
	mu    sync.RWMutex
	nodes []dataplane.TopologyNode
	edges []dataplane.ObservedEdge
}

// NewTopologyStore creates an empty TopologyStore.
func NewTopologyStore() *TopologyStore {
	return &TopologyStore{}
}

// GetTopology returns the full topology graph.
func (s *TopologyStore) GetTopology(_ context.Context) (*dataplane.TopologyGraph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]dataplane.TopologyNode, len(s.nodes))
	copy(nodes, s.nodes)
	edges := make([]dataplane.ObservedEdge, len(s.edges))
	copy(edges, s.edges)

	return &dataplane.TopologyGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// RecordEdge upserts an edge. If an edge with the same Source, Target, and
// EdgeType already exists, its RequestCount is incremented. Otherwise, a new
// edge is appended and nodes are created as needed.
func (s *TopologyStore) RecordEdge(_ context.Context, edge dataplane.ObservedEdge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Upsert edge.
	for i, e := range s.edges {
		if e.Source == edge.Source && e.Target == edge.Target && e.EdgeType == edge.EdgeType {
			s.edges[i].RequestCount += edge.RequestCount
			return nil
		}
	}
	s.edges = append(s.edges, edge)

	// Ensure nodes exist.
	s.ensureNode(edge.Source)
	s.ensureNode(edge.Target)

	return nil
}

// Upstream returns all services that have an edge targeting the given service.
func (s *TopologyStore) Upstream(_ context.Context, service string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	var result []string
	for _, e := range s.edges {
		if e.Target == service && !seen[e.Source] {
			seen[e.Source] = true
			result = append(result, e.Source)
		}
	}
	return result, nil
}

// Downstream returns all services that the given service calls.
func (s *TopologyStore) Downstream(_ context.Context, service string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	var result []string
	for _, e := range s.edges {
		if e.Source == service && !seen[e.Target] {
			seen[e.Target] = true
			result = append(result, e.Target)
		}
	}
	return result, nil
}

// ensureNode adds a node if it doesn't already exist. Must be called with mu held.
func (s *TopologyStore) ensureNode(name string) {
	for _, n := range s.nodes {
		if n.Name == name {
			return
		}
	}
	s.nodes = append(s.nodes, dataplane.TopologyNode{Name: name})
}

// Compile-time interface check.
var _ dataplane.TopologyStore = (*TopologyStore)(nil)
