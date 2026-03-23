package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/dataplane"
	pgstore "github.com/neureaux/cloudmock/pkg/dataplane/postgres"
)

func TestConfigStore_GetSetConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewConfigStore(pool)

	// Initially no config.
	_, err := store.GetConfig(ctx)
	require.ErrorIs(t, err, dataplane.ErrNotFound)

	// Set config.
	cfg := config.Default()
	cfg.Region = "eu-west-1"
	cfg.AccountID = "111111111111"
	require.NoError(t, store.SetConfig(ctx, cfg))

	// Get config.
	got, err := store.GetConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", got.Region)
	assert.Equal(t, "111111111111", got.AccountID)

	// Update config.
	cfg.Region = "ap-southeast-1"
	require.NoError(t, store.SetConfig(ctx, cfg))

	got, err = store.GetConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ap-southeast-1", got.Region)
}

func TestConfigStore_Deploys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewConfigStore(pool)

	// UpsertService first to satisfy FK.
	require.NoError(t, store.UpsertService(ctx, dataplane.ServiceEntry{
		Name: "api", ServiceType: "http",
	}))
	require.NoError(t, store.UpsertService(ctx, dataplane.ServiceEntry{
		Name: "worker", ServiceType: "async",
	}))

	now := time.Now().Truncate(time.Microsecond)

	require.NoError(t, store.AddDeploy(ctx, dataplane.DeployEvent{
		ID: "d1", Service: "api", Version: "v1.0.0", DeployedAt: now,
		Author: "alice", CommitSHA: "abc123",
	}))
	require.NoError(t, store.AddDeploy(ctx, dataplane.DeployEvent{
		ID: "d2", Service: "worker", Version: "v2.0.0", DeployedAt: now.Add(time.Second),
	}))
	require.NoError(t, store.AddDeploy(ctx, dataplane.DeployEvent{
		ID: "d3", Service: "api", Version: "v1.1.0", DeployedAt: now.Add(2 * time.Second),
	}))

	// List all.
	all, err := store.ListDeploys(ctx, dataplane.DeployFilter{})
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// Filter by service.
	apiDeploys, err := store.ListDeploys(ctx, dataplane.DeployFilter{Service: "api"})
	require.NoError(t, err)
	assert.Len(t, apiDeploys, 2)

	// With limit.
	limited, err := store.ListDeploys(ctx, dataplane.DeployFilter{Limit: 1})
	require.NoError(t, err)
	assert.Len(t, limited, 1)
}

func TestConfigStore_DeployAutoCreatesService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewConfigStore(pool)

	// AddDeploy should auto-create the service for FK.
	require.NoError(t, store.AddDeploy(ctx, dataplane.DeployEvent{
		ID: "d1", Service: "new-svc", Version: "v1.0.0", DeployedAt: time.Now(),
	}))

	deploys, err := store.ListDeploys(ctx, dataplane.DeployFilter{Service: "new-svc"})
	require.NoError(t, err)
	assert.Len(t, deploys, 1)
}

func TestConfigStore_Views(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewConfigStore(pool)

	// Save a view.
	require.NoError(t, store.SaveView(ctx, dataplane.SavedView{
		ID: "v1", Name: "Errors", CreatedBy: "alice",
		Filters:   map[string]interface{}{"status": "error"},
		CreatedAt: time.Now(),
	}))

	views, err := store.ListViews(ctx)
	require.NoError(t, err)
	require.Len(t, views, 1)
	assert.Equal(t, "Errors", views[0].Name)
	assert.Equal(t, "alice", views[0].CreatedBy)

	// Update existing view.
	require.NoError(t, store.SaveView(ctx, dataplane.SavedView{
		ID: "v1", Name: "All Errors", CreatedBy: "alice",
		Filters:   map[string]interface{}{"status": "error"},
		CreatedAt: time.Now(),
	}))
	views, err = store.ListViews(ctx)
	require.NoError(t, err)
	require.Len(t, views, 1)
	assert.Equal(t, "All Errors", views[0].Name)

	// Delete.
	require.NoError(t, store.DeleteView(ctx, "v1"))
	views, err = store.ListViews(ctx)
	require.NoError(t, err)
	assert.Len(t, views, 0)

	// Delete non-existent.
	err = store.DeleteView(ctx, "nonexistent")
	require.ErrorIs(t, err, dataplane.ErrNotFound)
}

func TestConfigStore_Services(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewConfigStore(pool)

	require.NoError(t, store.UpsertService(ctx, dataplane.ServiceEntry{
		Name: "dynamodb", ServiceType: "database", Owner: "platform",
	}))

	services, err := store.ListServices(ctx)
	require.NoError(t, err)
	require.Len(t, services, 1)
	assert.Equal(t, "platform", services[0].Owner)

	// Upsert existing.
	require.NoError(t, store.UpsertService(ctx, dataplane.ServiceEntry{
		Name: "dynamodb", ServiceType: "database", Owner: "infra",
	}))
	services, err = store.ListServices(ctx)
	require.NoError(t, err)
	require.Len(t, services, 1)
	assert.Equal(t, "infra", services[0].Owner)
}
