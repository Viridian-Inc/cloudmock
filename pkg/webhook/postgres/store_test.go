package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/neureaux/cloudmock/pkg/webhook"
	pgstore "github.com/neureaux/cloudmock/pkg/webhook/postgres"
)

// setupPostgres starts a PostgreSQL container, applies all six schema files,
// and returns a connected pool. The container is cleaned up when the test ends.
func setupPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "cloudmock",
			"POSTGRES_USER":     "cloudmock",
			"POSTGRES_PASSWORD": "cloudmock",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	dsn := fmt.Sprintf("postgres://cloudmock:cloudmock@%s:%s/cloudmock?sslmode=disable", host, port.Port())

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	schemas := []string{
		"../../../docker/init/postgres/01-schema.sql",
		"../../../docker/init/postgres/02-regression-schema.sql",
		"../../../docker/init/postgres/03-incident-schema.sql",
		"../../../docker/init/postgres/04-audit-schema.sql",
		"../../../docker/init/postgres/05-auth-schema.sql",
		"../../../docker/init/postgres/06-webhook-schema.sql",
	}

	for _, path := range schemas {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("failed to apply %s: %v", path, err)
		}
	}

	return pool
}

func sampleConfig(url string) *webhook.Config {
	return &webhook.Config{
		URL:     url,
		Type:    "generic",
		Events:  []string{"incident.created", "incident.resolved"},
		Headers: map[string]string{"X-Token": "secret"},
		Active:  true,
	}
}

func TestSaveAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	cfg := sampleConfig("https://example.com/hook")
	require.NoError(t, store.Save(ctx, cfg))
	assert.NotEmpty(t, cfg.ID)

	got, err := store.Get(ctx, cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, cfg.ID, got.ID)
	assert.Equal(t, cfg.URL, got.URL)
	assert.Equal(t, cfg.Type, got.Type)
	assert.Equal(t, cfg.Events, got.Events)
	assert.Equal(t, cfg.Headers, got.Headers)
	assert.Equal(t, cfg.Active, got.Active)
}

func TestGet_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	_, err := store.Get(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, webhook.ErrNotFound)
}

func TestList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	require.NoError(t, store.Save(ctx, sampleConfig("https://a.example.com")))
	require.NoError(t, store.Save(ctx, sampleConfig("https://b.example.com")))

	list, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	cfg := sampleConfig("https://example.com/hook")
	require.NoError(t, store.Save(ctx, cfg))

	require.NoError(t, store.Delete(ctx, cfg.ID))

	_, err := store.Get(ctx, cfg.ID)
	assert.ErrorIs(t, err, webhook.ErrNotFound)
}

func TestDelete_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	err := store.Delete(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, webhook.ErrNotFound)
}

func TestListByEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	c1 := &webhook.Config{
		URL:    "https://slack.example.com",
		Type:   "slack",
		Events: []string{"incident.created"},
		Active: true,
	}
	c2 := &webhook.Config{
		URL:    "https://pd.example.com",
		Type:   "pagerduty",
		Events: []string{"incident.created", "incident.resolved"},
		Active: true,
	}
	c3 := &webhook.Config{
		URL:    "https://inactive.example.com",
		Type:   "generic",
		Events: []string{"incident.created"},
		Active: false,
	}

	require.NoError(t, store.Save(ctx, c1))
	require.NoError(t, store.Save(ctx, c2))
	require.NoError(t, store.Save(ctx, c3))

	t.Run("incident.created returns active only", func(t *testing.T) {
		got, err := store.ListByEvent(ctx, "incident.created")
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("incident.resolved returns one", func(t *testing.T) {
		got, err := store.ListByEvent(ctx, "incident.resolved")
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "pagerduty", got[0].Type)
	})

	t.Run("unknown event returns empty", func(t *testing.T) {
		got, err := store.ListByEvent(ctx, "unknown.event")
		require.NoError(t, err)
		assert.Len(t, got, 0)
	})
}

func TestSave_Upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	cfg := sampleConfig("https://example.com/hook")
	require.NoError(t, store.Save(ctx, cfg))
	id := cfg.ID

	cfg.URL = "https://updated.example.com/hook"
	cfg.Active = false
	require.NoError(t, store.Save(ctx, cfg))

	list, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1, "upsert should not create a duplicate")
	assert.Equal(t, id, list[0].ID)
	assert.Equal(t, "https://updated.example.com/hook", list[0].URL)
	assert.False(t, list[0].Active)
}
