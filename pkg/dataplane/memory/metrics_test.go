package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane/memory"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

func TestMetricStore_Record_And_ServiceStats(t *testing.T) {
	stats := gateway.NewRequestStats()
	log := gateway.NewRequestLog(100)
	s := memory.NewMetricStore(stats, log)

	ctx := context.Background()
	window := 5 * time.Minute

	// Record some metrics.
	for _, lat := range []float64{10, 20, 30, 40, 50} {
		if err := s.Record(ctx, "dynamodb", "Query", lat, 200); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	// Record an error.
	if err := s.Record(ctx, "dynamodb", "Query", 100, 500); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, err := s.ServiceStats(ctx, "dynamodb", window)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Service != "dynamodb" {
		t.Errorf("got Service=%q, want %q", m.Service, "dynamodb")
	}
	if m.RequestCount != 6 {
		t.Errorf("got RequestCount=%d, want 6", m.RequestCount)
	}
	if m.ErrorCount != 1 {
		t.Errorf("got ErrorCount=%d, want 1", m.ErrorCount)
	}
	if m.P50Ms == 0 {
		t.Error("P50Ms should be non-zero")
	}
	if m.P99Ms == 0 {
		t.Error("P99Ms should be non-zero")
	}
}

func TestMetricStore_Percentiles(t *testing.T) {
	stats := gateway.NewRequestStats()
	log := gateway.NewRequestLog(100)
	s := memory.NewMetricStore(stats, log)

	ctx := context.Background()

	for i := 1; i <= 100; i++ {
		if err := s.Record(ctx, "s3", "GetObject", float64(i), 200); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	p, err := s.Percentiles(ctx, "s3", "GetObject", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.P50Ms == 0 {
		t.Error("P50Ms should be non-zero")
	}
	if p.P95Ms == 0 {
		t.Error("P95Ms should be non-zero")
	}
	if p.P99Ms == 0 {
		t.Error("P99Ms should be non-zero")
	}
	if p.P50Ms >= p.P95Ms {
		t.Errorf("P50Ms (%.1f) should be < P95Ms (%.1f)", p.P50Ms, p.P95Ms)
	}
	if p.P95Ms >= p.P99Ms {
		t.Errorf("P95Ms (%.1f) should be < P99Ms (%.1f)", p.P95Ms, p.P99Ms)
	}
}

func TestMetricStore_ServiceStats_Empty(t *testing.T) {
	stats := gateway.NewRequestStats()
	log := gateway.NewRequestLog(100)
	s := memory.NewMetricStore(stats, log)

	m, err := s.ServiceStats(context.Background(), "nonexistent", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.RequestCount != 0 {
		t.Errorf("got RequestCount=%d, want 0", m.RequestCount)
	}
	if m.P50Ms != 0 {
		t.Errorf("got P50Ms=%.1f, want 0", m.P50Ms)
	}
}
