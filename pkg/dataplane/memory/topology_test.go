package memory_test

import (
	"context"
	"testing"

	"github.com/neureaux/cloudmock/pkg/dataplane"
	"github.com/neureaux/cloudmock/pkg/dataplane/memory"
)

func TestTopologyStore_RecordEdge_And_GetTopology(t *testing.T) {
	s := memory.NewTopologyStore()
	ctx := context.Background()

	if err := s.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "api", Target: "dynamodb", EdgeType: "http", RequestCount: 1,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "api", Target: "s3", EdgeType: "http", RequestCount: 1,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	graph, err := s.GetTopology(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(graph.Edges) != 2 {
		t.Errorf("got %d edges, want 2", len(graph.Edges))
	}
	if len(graph.Nodes) != 3 {
		t.Errorf("got %d nodes, want 3 (api, dynamodb, s3)", len(graph.Nodes))
	}
}

func TestTopologyStore_RecordEdge_Upsert(t *testing.T) {
	s := memory.NewTopologyStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := s.RecordEdge(ctx, dataplane.ObservedEdge{
			Source: "api", Target: "dynamodb", EdgeType: "http", RequestCount: 1,
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	graph, err := s.GetTopology(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("got %d edges, want 1 (upserted)", len(graph.Edges))
	}
	if graph.Edges[0].RequestCount != 5 {
		t.Errorf("got RequestCount=%d, want 5", graph.Edges[0].RequestCount)
	}
}

func TestTopologyStore_Upstream(t *testing.T) {
	s := memory.NewTopologyStore()
	ctx := context.Background()

	_ = s.RecordEdge(ctx, dataplane.ObservedEdge{Source: "api", Target: "dynamodb", EdgeType: "http", RequestCount: 1})
	_ = s.RecordEdge(ctx, dataplane.ObservedEdge{Source: "worker", Target: "dynamodb", EdgeType: "http", RequestCount: 1})
	_ = s.RecordEdge(ctx, dataplane.ObservedEdge{Source: "api", Target: "s3", EdgeType: "http", RequestCount: 1})

	upstream, err := s.Upstream(ctx, "dynamodb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(upstream) != 2 {
		t.Fatalf("got %d upstream, want 2", len(upstream))
	}

	// Verify it contains both api and worker.
	found := make(map[string]bool)
	for _, u := range upstream {
		found[u] = true
	}
	if !found["api"] || !found["worker"] {
		t.Errorf("expected api and worker in upstream, got %v", upstream)
	}
}

func TestTopologyStore_Downstream(t *testing.T) {
	s := memory.NewTopologyStore()
	ctx := context.Background()

	_ = s.RecordEdge(ctx, dataplane.ObservedEdge{Source: "api", Target: "dynamodb", EdgeType: "http", RequestCount: 1})
	_ = s.RecordEdge(ctx, dataplane.ObservedEdge{Source: "api", Target: "s3", EdgeType: "http", RequestCount: 1})
	_ = s.RecordEdge(ctx, dataplane.ObservedEdge{Source: "api", Target: "lambda", EdgeType: "http", RequestCount: 1})

	downstream, err := s.Downstream(ctx, "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(downstream) != 3 {
		t.Fatalf("got %d downstream, want 3", len(downstream))
	}
}

func TestTopologyStore_Empty(t *testing.T) {
	s := memory.NewTopologyStore()
	ctx := context.Background()

	graph, err := s.GetTopology(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("got %d nodes, want 0", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Errorf("got %d edges, want 0", len(graph.Edges))
	}

	upstream, _ := s.Upstream(ctx, "anything")
	if len(upstream) != 0 {
		t.Errorf("got %d upstream, want 0", len(upstream))
	}
}
