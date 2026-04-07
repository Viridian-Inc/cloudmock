package memory_test

import (
	"context"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/dataplane/memory"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

func TestSLOStore_Rules(t *testing.T) {
	rules := []config.SLORule{
		{Service: "dynamodb", Action: "*", P50Ms: 10, P95Ms: 50, P99Ms: 100, ErrorRate: 0.01},
	}
	engine := gateway.NewSLOEngine(rules)
	s := memory.NewSLOStore(engine)

	got, err := s.Rules(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rules, want 1", len(got))
	}
	if got[0].Service != "dynamodb" {
		t.Errorf("got Service=%q, want %q", got[0].Service, "dynamodb")
	}
}

func TestSLOStore_SetRules(t *testing.T) {
	engine := gateway.NewSLOEngine(nil)
	s := memory.NewSLOStore(engine)

	newRules := []config.SLORule{
		{Service: "s3", Action: "*", P50Ms: 20, P95Ms: 80, P99Ms: 200, ErrorRate: 0.02},
		{Service: "lambda", Action: "Invoke", P50Ms: 5, P95Ms: 20, P99Ms: 50, ErrorRate: 0.005},
	}

	if err := s.SetRules(context.Background(), newRules); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := s.Rules(context.Background())
	if len(got) != 2 {
		t.Fatalf("got %d rules, want 2", len(got))
	}
	if got[0].Service != "s3" {
		t.Errorf("got Service=%q, want %q", got[0].Service, "s3")
	}
}

func TestSLOStore_Status(t *testing.T) {
	rules := []config.SLORule{
		{Service: "*", Action: "*", P50Ms: 10, P95Ms: 50, P99Ms: 100, ErrorRate: 0.01},
	}
	engine := gateway.NewSLOEngine(rules)

	// Record some requests to create a window.
	for i := 0; i < 20; i++ {
		engine.Record("dynamodb", "Query", float64(i+1), 200)
	}

	s := memory.NewSLOStore(engine)
	status, err := s.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("status should not be nil")
	}
	if len(status.Windows) == 0 {
		t.Fatal("expected at least one window")
	}
	if status.Windows[0].Total != 20 {
		t.Errorf("got Total=%d, want 20", status.Windows[0].Total)
	}
	if !status.Healthy {
		t.Error("expected healthy status with no errors")
	}
}

func TestSLOStore_History(t *testing.T) {
	engine := gateway.NewSLOEngine(nil)
	s := memory.NewSLOStore(engine)

	history, err := s.History(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("got %d history entries, want 0", len(history))
	}
}
