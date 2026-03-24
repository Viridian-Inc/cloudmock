package dataplane_test

import (
	"context"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/dataplane"
	"github.com/neureaux/cloudmock/pkg/dataplane/memory"
	"github.com/neureaux/cloudmock/pkg/gateway"
)

// runParityTests runs the same test suite against any DataPlane implementation.
func runParityTests(t *testing.T, dp *dataplane.DataPlane) {
	ctx := context.Background()

	t.Run("TraceWriteAndRead", func(t *testing.T) {
		// Write 3 spans: root + 2 children
		root := &dataplane.Span{
			TraceID: "parity-trace-1", SpanID: "root-1",
			Service: "bff", Action: "GetUser", Method: "GET", Path: "/users/1",
			StartTime: time.Now().Add(-100 * time.Millisecond),
			EndTime:   time.Now().Add(-50 * time.Millisecond),
			StatusCode: 200, TenantID: "tenant-1",
		}
		child1 := &dataplane.Span{
			TraceID: "parity-trace-1", SpanID: "child-1", ParentSpanID: "root-1",
			Service: "dynamodb", Action: "Query",
			StartTime:  time.Now().Add(-90 * time.Millisecond),
			EndTime:    time.Now().Add(-60 * time.Millisecond),
			StatusCode: 200,
		}
		child2 := &dataplane.Span{
			TraceID: "parity-trace-1", SpanID: "child-2", ParentSpanID: "root-1",
			Service: "lambda", Action: "Invoke",
			StartTime:  time.Now().Add(-80 * time.Millisecond),
			EndTime:    time.Now().Add(-55 * time.Millisecond),
			StatusCode: 200,
		}
		err := dp.TraceW.WriteSpans(ctx, []*dataplane.Span{root, child1, child2})
		if err != nil {
			t.Fatalf("WriteSpans: %v", err)
		}

		// Read back
		tc, err := dp.Traces.Get(ctx, "parity-trace-1")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if tc.Service != "bff" {
			t.Errorf("expected root service bff, got %s", tc.Service)
		}

		// Not found
		_, err = dp.Traces.Get(ctx, "nonexistent")
		if err != dataplane.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("TraceSearch", func(t *testing.T) {
		results, err := dp.Traces.Search(ctx, dataplane.TraceFilter{
			Service: "bff", Limit: 10,
		})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected at least 1 search result")
		}
	})

	t.Run("SLORulesRoundtrip", func(t *testing.T) {
		rules := []config.SLORule{
			{Service: "bff", Action: "*", P50Ms: 50, P95Ms: 200, P99Ms: 500, ErrorRate: 0.01},
		}
		err := dp.SLO.SetRules(ctx, rules)
		if err != nil {
			t.Fatalf("SetRules: %v", err)
		}
		got, err := dp.SLO.Rules(ctx)
		if err != nil {
			t.Fatalf("Rules: %v", err)
		}
		if len(got) != 1 || got[0].Service != "bff" {
			t.Errorf("expected 1 rule for bff, got %d rules", len(got))
		}
	})

	t.Run("ViewsCRUD", func(t *testing.T) {
		view := dataplane.SavedView{
			ID: "test-view", Name: "Test View",
			Filters:   map[string]any{"service": "bff"},
			CreatedBy: "test",
		}
		err := dp.Config.SaveView(ctx, view)
		if err != nil {
			t.Fatalf("SaveView: %v", err)
		}
		views, err := dp.Config.ListViews(ctx)
		if err != nil {
			t.Fatalf("ListViews: %v", err)
		}
		found := false
		for _, v := range views {
			if v.ID == "test-view" {
				found = true
			}
		}
		if !found {
			t.Error("saved view not found")
		}
		err = dp.Config.DeleteView(ctx, "test-view")
		if err != nil {
			t.Fatalf("DeleteView: %v", err)
		}
	})

	t.Run("TopologyEdgeUpsert", func(t *testing.T) {
		edge := dataplane.ObservedEdge{
			Source: "bff", Target: "dynamodb", EdgeType: "traffic", RequestCount: 1,
		}
		err := dp.Topology.RecordEdge(ctx, edge)
		if err != nil {
			t.Fatalf("RecordEdge: %v", err)
		}
		// Record same edge again
		err = dp.Topology.RecordEdge(ctx, edge)
		if err != nil {
			t.Fatalf("RecordEdge second: %v", err)
		}

		downstream, err := dp.Topology.Downstream(ctx, "bff")
		if err != nil {
			t.Fatalf("Downstream: %v", err)
		}
		if len(downstream) == 0 {
			t.Error("expected at least 1 downstream service")
		}
	})
}

func buildLocalDataPlane() *dataplane.DataPlane {
	traceStore := gateway.NewTraceStore(500)
	requestLog := gateway.NewRequestLog(1000)
	requestStats := gateway.NewRequestStats()
	sloEngine := gateway.NewSLOEngine(nil)
	cfg := config.Default()

	memTraces := memory.NewTraceStore(traceStore)
	memRequests := memory.NewRequestStore(requestLog)
	memMetrics := memory.NewMetricStore(requestStats, requestLog)

	return &dataplane.DataPlane{
		Traces:   memTraces,
		TraceW:   memTraces,
		Requests: memRequests,
		RequestW: memRequests,
		Metrics:  memMetrics,
		MetricW:  memMetrics,
		SLO:      memory.NewSLOStore(sloEngine),
		Config:   memory.NewConfigStore(cfg),
		Topology: memory.NewTopologyStore(),
		Mode:     "local",
	}
}

func TestLocalDataPlane(t *testing.T) {
	dp := buildLocalDataPlane()
	runParityTests(t, dp)
}

func TestProductionDataPlane(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	// TODO: Build production DataPlane with DuckDB + PostgreSQL.
	// DuckDB can use :memory: but PostgreSQL still needs a container.
	// For now, skip — the individual store tests in
	// duckdb/ and postgres/ packages cover this.
	t.Skip("production parity test requires Docker for PostgreSQL — individual store tests provide coverage")
}
