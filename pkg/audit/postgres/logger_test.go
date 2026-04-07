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

	"github.com/Viridian-Inc/cloudmock/pkg/audit"
	auditpg "github.com/Viridian-Inc/cloudmock/pkg/audit/postgres"
)

// setupPostgres starts a PostgreSQL container, applies schema files 01–04, and
// returns a connected pool. The container is cleaned up when the test ends.
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

	// Apply schemas in order.
	schemas := []string{
		"../../../docker/init/postgres/01-schema.sql",
		"../../../docker/init/postgres/02-regression-schema.sql",
		"../../../docker/init/postgres/03-incident-schema.sql",
		"../../../docker/init/postgres/04-audit-schema.sql",
	}
	for _, path := range schemas {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		if _, err := pool.Exec(ctx, string(data)); err != nil {
			t.Fatalf("failed to apply %s: %v", path, err)
		}
	}

	return pool
}

func TestLogAndQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	l := auditpg.NewLogger(pool)

	// Log some entries.
	require.NoError(t, l.Log(ctx, audit.Entry{
		Actor:    "alice",
		Action:   "deploy.created",
		Resource: "deploy:d1",
		Details:  map[string]any{"version": "v1.2.3"},
	}))
	require.NoError(t, l.Log(ctx, audit.Entry{
		Actor:    "bob",
		Action:   "view.saved",
		Resource: "view:v1",
	}))
	require.NoError(t, l.Log(ctx, audit.Entry{
		Actor:    "alice",
		Action:   "slo.rules.updated",
		Resource: "slo:config",
	}))

	// Query all — newest first.
	all, err := l.Query(ctx, audit.Filter{})
	require.NoError(t, err)
	assert.Len(t, all, 3)
	assert.Equal(t, "slo.rules.updated", all[0].Action)

	// Each entry should have a UUID id.
	for _, e := range all {
		assert.NotEmpty(t, e.ID)
		assert.False(t, e.Timestamp.IsZero())
	}

	// Query by actor.
	aliceEntries, err := l.Query(ctx, audit.Filter{Actor: "alice"})
	require.NoError(t, err)
	assert.Len(t, aliceEntries, 2)

	// Query by action.
	deployEntries, err := l.Query(ctx, audit.Filter{Action: "deploy.created"})
	require.NoError(t, err)
	assert.Len(t, deployEntries, 1)
	assert.Equal(t, "deploy:d1", deployEntries[0].Resource)

	// Query by resource.
	sloEntries, err := l.Query(ctx, audit.Filter{Resource: "slo:config"})
	require.NoError(t, err)
	assert.Len(t, sloEntries, 1)

	// Query with limit.
	limited, err := l.Query(ctx, audit.Filter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, limited, 2)

	// Verify details round-trip.
	deployEntries2, err := l.Query(ctx, audit.Filter{Action: "deploy.created"})
	require.NoError(t, err)
	require.Len(t, deployEntries2, 1)
	assert.Equal(t, "v1.2.3", deployEntries2[0].Details["version"])
}
