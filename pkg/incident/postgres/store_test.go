package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Viridian-Inc/cloudmock/pkg/incident"
	pgstore "github.com/Viridian-Inc/cloudmock/pkg/incident/postgres"
)

// setupPostgres starts a PostgreSQL container, applies all three schema files,
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

	// Apply base schema (includes deploys table needed for FK).
	schema01, err := os.ReadFile("../../../docker/init/postgres/01-schema.sql")
	if err != nil {
		t.Fatalf("failed to read 01-schema.sql: %v", err)
	}
	if _, err := pool.Exec(ctx, string(schema01)); err != nil {
		t.Fatalf("failed to apply 01-schema.sql: %v", err)
	}

	// Apply regression schema.
	schema02, err := os.ReadFile("../../../docker/init/postgres/02-regression-schema.sql")
	if err != nil {
		t.Fatalf("failed to read 02-regression-schema.sql: %v", err)
	}
	if _, err := pool.Exec(ctx, string(schema02)); err != nil {
		t.Fatalf("failed to apply 02-regression-schema.sql: %v", err)
	}

	// Apply incident schema.
	schema03, err := os.ReadFile("../../../docker/init/postgres/03-incident-schema.sql")
	if err != nil {
		t.Fatalf("failed to read 03-incident-schema.sql: %v", err)
	}
	if _, err := pool.Exec(ctx, string(schema03)); err != nil {
		t.Fatalf("failed to apply 03-incident-schema.sql: %v", err)
	}

	return pool
}

// insertService creates a service row to satisfy the FK from deploys.
func insertService(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) {
	t.Helper()
	_, err := pool.Exec(ctx,
		`INSERT INTO services (name, service_type) VALUES ($1, 'unknown') ON CONFLICT DO NOTHING`,
		name)
	require.NoError(t, err)
}

// insertDeploy creates a deploy row and returns its UUID string.
func insertDeploy(t *testing.T, ctx context.Context, pool *pgxpool.Pool, service string) string {
	t.Helper()
	insertService(t, ctx, pool, service)
	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO deploys (service, version) VALUES ($1, 'v1.0.0') RETURNING id`,
		service).Scan(&id)
	require.NoError(t, err)
	return id
}

func sampleIncident(service string) *incident.Incident {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return &incident.Incident{
		Status:           "active",
		Severity:         "critical",
		Title:            "Latency spike in " + service,
		AffectedServices: []string{service},
		AffectedTenants:  []string{"tenant-a"},
		AlertCount:       3,
		FirstSeen:        now.Add(-10 * time.Minute),
		LastSeen:         now,
	}
}

// TestSaveAndGet verifies that a saved incident can be retrieved with all
// fields intact.
func TestSaveAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	inc := sampleIncident("svc-auth")

	require.NoError(t, store.Save(ctx, inc))
	assert.NotEmpty(t, inc.ID, "ID should be populated after Save")

	got, err := store.Get(ctx, inc.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, inc.ID, got.ID)
	assert.Equal(t, inc.Status, got.Status)
	assert.Equal(t, inc.Severity, got.Severity)
	assert.Equal(t, inc.Title, got.Title)
	assert.Equal(t, inc.AffectedServices, got.AffectedServices)
	assert.Equal(t, inc.AffectedTenants, got.AffectedTenants)
	assert.Equal(t, inc.AlertCount, got.AlertCount)
	assert.Empty(t, got.RootCause)
	assert.Empty(t, got.RelatedDeployID)
	assert.WithinDuration(t, inc.FirstSeen, got.FirstSeen, time.Millisecond)
	assert.WithinDuration(t, inc.LastSeen, got.LastSeen, time.Millisecond)
	assert.Nil(t, got.ResolvedAt)
	assert.Empty(t, got.Owner)
}

// TestSaveAndGet_WithDeployID verifies that an incident linked to a deploy
// round-trips correctly.
func TestSaveAndGet_WithDeployID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	deployID := insertDeploy(t, ctx, pool, "svc-api")

	inc := sampleIncident("svc-api")
	inc.RelatedDeployID = deployID

	require.NoError(t, store.Save(ctx, inc))

	got, err := store.Get(ctx, inc.ID)
	require.NoError(t, err)
	assert.Equal(t, deployID, got.RelatedDeployID)
}

// TestGet_NotFound verifies that retrieving a non-existent ID returns ErrNotFound.
func TestGet_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	_, err := store.Get(ctx, uuid.New().String())
	assert.ErrorIs(t, err, incident.ErrNotFound)
}

// TestList_Filters verifies that the List method filters correctly.
func TestList_Filters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	inc1 := sampleIncident("svc-order")
	inc1.Severity = "critical"
	inc1.Status = "active"

	inc2 := sampleIncident("svc-payment")
	inc2.Severity = "warning"
	inc2.Status = "resolved"

	inc3 := sampleIncident("svc-order")
	inc3.Severity = "warning"
	inc3.Status = "active"

	require.NoError(t, store.Save(ctx, inc1))
	require.NoError(t, store.Save(ctx, inc2))
	require.NoError(t, store.Save(ctx, inc3))

	t.Run("filter by status", func(t *testing.T) {
		got, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, g := range got {
			assert.Equal(t, "active", g.Status)
		}
	})

	t.Run("filter by severity", func(t *testing.T) {
		got, err := store.List(ctx, incident.IncidentFilter{Severity: "critical"})
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "critical", got[0].Severity)
	})

	t.Run("filter by service", func(t *testing.T) {
		got, err := store.List(ctx, incident.IncidentFilter{Service: "svc-order"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("limit", func(t *testing.T) {
		got, err := store.List(ctx, incident.IncidentFilter{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}

// TestUpdate verifies that Update replaces all fields.
func TestUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	inc := sampleIncident("svc-cache")
	require.NoError(t, store.Save(ctx, inc))

	// Modify and update.
	inc.AlertCount = 10
	inc.Status = "acknowledged"
	inc.Owner = "oncall-eng"
	inc.RootCause = "bad deploy"
	require.NoError(t, store.Update(ctx, inc))

	got, err := store.Get(ctx, inc.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, got.AlertCount)
	assert.Equal(t, "acknowledged", got.Status)
	assert.Equal(t, "oncall-eng", got.Owner)
	assert.Equal(t, "bad deploy", got.RootCause)
}

// TestUpdate_NotFound verifies that updating a non-existent incident returns ErrNotFound.
func TestUpdate_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	inc := &incident.Incident{
		ID:               uuid.New().String(),
		Status:           "active",
		Severity:         "warning",
		Title:            "Ghost incident",
		AffectedServices: []string{"svc-ghost"},
		AffectedTenants:  []string{},
		AlertCount:       1,
		FirstSeen:        time.Now().UTC(),
		LastSeen:         time.Now().UTC(),
	}
	err := store.Update(ctx, inc)
	assert.ErrorIs(t, err, incident.ErrNotFound)
}

// TestFindActiveByKey verifies the active-by-key lookup.
func TestFindActiveByKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	now := time.Now().UTC().Truncate(time.Millisecond)
	deployID := insertDeploy(t, ctx, pool, "svc-api")

	inc := &incident.Incident{
		Status:           "active",
		Severity:         "critical",
		Title:            "Latency spike",
		AffectedServices: []string{"svc-api", "svc-db"},
		AffectedTenants:  []string{},
		AlertCount:       2,
		RelatedDeployID:  deployID,
		FirstSeen:        now.Add(-5 * time.Minute),
		LastSeen:         now,
	}
	require.NoError(t, store.Save(ctx, inc))

	t.Run("match by service and deploy", func(t *testing.T) {
		got, err := store.FindActiveByKey(ctx, "svc-api", deployID, now.Add(-10*time.Minute))
		require.NoError(t, err)
		assert.Equal(t, inc.ID, got.ID)
	})

	t.Run("match by service only", func(t *testing.T) {
		got, err := store.FindActiveByKey(ctx, "svc-db", "", now.Add(-10*time.Minute))
		require.NoError(t, err)
		assert.Equal(t, inc.ID, got.ID)
	})

	t.Run("no match wrong service", func(t *testing.T) {
		_, err := store.FindActiveByKey(ctx, "svc-other", "", now.Add(-10*time.Minute))
		assert.ErrorIs(t, err, incident.ErrNotFound)
	})

	t.Run("no match resolved status", func(t *testing.T) {
		resolved := &incident.Incident{
			Status:           "resolved",
			Severity:         "warning",
			Title:            "Resolved",
			AffectedServices: []string{"svc-resolved"},
			AffectedTenants:  []string{},
			AlertCount:       1,
			FirstSeen:        now.Add(-5 * time.Minute),
			LastSeen:         now,
		}
		require.NoError(t, store.Save(ctx, resolved))

		_, err := store.FindActiveByKey(ctx, "svc-resolved", "", now.Add(-10*time.Minute))
		assert.ErrorIs(t, err, incident.ErrNotFound)
	})

	t.Run("no match too old", func(t *testing.T) {
		_, err := store.FindActiveByKey(ctx, "svc-api", deployID, now.Add(1*time.Minute))
		assert.ErrorIs(t, err, incident.ErrNotFound)
	})
}
