package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/dataplane"
	"github.com/neureaux/cloudmock/pkg/dataplane/memory"
)

func TestConfigStore_GetSetConfig(t *testing.T) {
	s := memory.NewConfigStore(nil)
	ctx := context.Background()

	// Initially no config.
	_, err := s.GetConfig(ctx)
	if err != dataplane.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	cfg := &config.Config{Region: "us-east-1", AccountID: "123456789012"}
	if err := s.SetConfig(ctx, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.GetConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Region != "us-east-1" {
		t.Errorf("got Region=%q, want %q", got.Region, "us-east-1")
	}
}

func TestConfigStore_Deploys(t *testing.T) {
	s := memory.NewConfigStore(nil)
	ctx := context.Background()

	if err := s.AddDeploy(ctx, dataplane.DeployEvent{
		ID: "d1", Service: "api", Version: "v1.0.0", DeployedAt: time.Now(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.AddDeploy(ctx, dataplane.DeployEvent{
		ID: "d2", Service: "worker", Version: "v2.0.0", DeployedAt: time.Now(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.AddDeploy(ctx, dataplane.DeployEvent{
		ID: "d3", Service: "api", Version: "v1.1.0", DeployedAt: time.Now(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// List all.
	all, err := s.ListDeploys(ctx, dataplane.DeployFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d deploys, want 3", len(all))
	}

	// Filter by service.
	apiDeploys, err := s.ListDeploys(ctx, dataplane.DeployFilter{Service: "api"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apiDeploys) != 2 {
		t.Fatalf("got %d api deploys, want 2", len(apiDeploys))
	}

	// With limit.
	limited, err := s.ListDeploys(ctx, dataplane.DeployFilter{Limit: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("got %d deploys with limit, want 1", len(limited))
	}
}

func TestConfigStore_Views(t *testing.T) {
	s := memory.NewConfigStore(nil)
	ctx := context.Background()

	if err := s.SaveView(ctx, dataplane.SavedView{
		ID: "v1", Name: "Errors", CreatedBy: "alice", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	views, err := s.ListViews(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("got %d views, want 1", len(views))
	}
	if views[0].Name != "Errors" {
		t.Errorf("got Name=%q, want %q", views[0].Name, "Errors")
	}

	// Update existing view.
	if err := s.SaveView(ctx, dataplane.SavedView{
		ID: "v1", Name: "All Errors", CreatedBy: "alice", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	views, _ = s.ListViews(ctx)
	if len(views) != 1 {
		t.Fatalf("got %d views after upsert, want 1", len(views))
	}
	if views[0].Name != "All Errors" {
		t.Errorf("got Name=%q, want %q", views[0].Name, "All Errors")
	}

	// Delete.
	if err := s.DeleteView(ctx, "v1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	views, _ = s.ListViews(ctx)
	if len(views) != 0 {
		t.Fatalf("got %d views after delete, want 0", len(views))
	}

	// Delete non-existent.
	if err := s.DeleteView(ctx, "nonexistent"); err != dataplane.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestConfigStore_Services(t *testing.T) {
	s := memory.NewConfigStore(nil)
	ctx := context.Background()

	if err := s.UpsertService(ctx, dataplane.ServiceEntry{
		Name: "dynamodb", ServiceType: "database", Owner: "platform",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	services, err := s.ListServices(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("got %d services, want 1", len(services))
	}
	if services[0].Owner != "platform" {
		t.Errorf("got Owner=%q, want %q", services[0].Owner, "platform")
	}

	// Upsert existing.
	if err := s.UpsertService(ctx, dataplane.ServiceEntry{
		Name: "dynamodb", ServiceType: "database", Owner: "infra",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	services, _ = s.ListServices(ctx)
	if len(services) != 1 {
		t.Fatalf("got %d services after upsert, want 1", len(services))
	}
	if services[0].Owner != "infra" {
		t.Errorf("got Owner=%q, want %q", services[0].Owner, "infra")
	}
}
