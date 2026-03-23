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

	"github.com/neureaux/cloudmock/pkg/regression"
	pgstore "github.com/neureaux/cloudmock/pkg/regression/postgres"
)

// setupPostgres starts a PostgreSQL container, applies both schema files, and
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

// sampleRegression builds a Regression with all fields populated for testing.
func sampleRegression(service string) *regression.Regression {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return &regression.Regression{
		Algorithm:     regression.AlgoLatencyRegression,
		Severity:      regression.SeverityWarning,
		Confidence:    82,
		Service:       service,
		Action:        "Query",
		TenantID:      "tenant-abc",
		Title:         "p99 latency increased by 60%",
		BeforeValue:   100.0,
		AfterValue:    160.0,
		ChangePercent: 60.0,
		SampleSize:    500,
		DetectedAt:    now,
		WindowBefore: regression.TimeWindow{
			Start: now.Add(-2 * time.Hour),
			End:   now.Add(-1 * time.Hour),
		},
		WindowAfter: regression.TimeWindow{
			Start: now.Add(-1 * time.Hour),
			End:   now,
		},
		Status: "active",
	}
}

// TestSaveAndGet verifies that a saved regression can be retrieved with all
// fields intact.
func TestSaveAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	r := sampleRegression("svc-auth")

	require.NoError(t, store.Save(ctx, r))
	assert.NotEmpty(t, r.ID, "ID should be populated after Save")

	got, err := store.Get(ctx, r.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, r.ID, got.ID)
	assert.Equal(t, r.Algorithm, got.Algorithm)
	assert.Equal(t, r.Severity, got.Severity)
	assert.Equal(t, r.Confidence, got.Confidence)
	assert.Equal(t, r.Service, got.Service)
	assert.Equal(t, r.Action, got.Action)
	assert.Equal(t, r.TenantID, got.TenantID)
	assert.Equal(t, r.Title, got.Title)
	assert.InDelta(t, r.BeforeValue, got.BeforeValue, 0.001)
	assert.InDelta(t, r.AfterValue, got.AfterValue, 0.001)
	assert.InDelta(t, r.ChangePercent, got.ChangePercent, 0.001)
	assert.Equal(t, r.SampleSize, got.SampleSize)
	assert.WithinDuration(t, r.DetectedAt, got.DetectedAt, time.Millisecond)
	assert.WithinDuration(t, r.WindowBefore.Start, got.WindowBefore.Start, time.Millisecond)
	assert.WithinDuration(t, r.WindowBefore.End, got.WindowBefore.End, time.Millisecond)
	assert.WithinDuration(t, r.WindowAfter.Start, got.WindowAfter.Start, time.Millisecond)
	assert.WithinDuration(t, r.WindowAfter.End, got.WindowAfter.End, time.Millisecond)
	assert.Equal(t, "active", got.Status)
	assert.Nil(t, got.ResolvedAt)
}

// TestSaveAndGet_WithDeployID verifies that a regression linked to a deploy
// round-trips correctly.
func TestSaveAndGet_WithDeployID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	deployID := insertDeploy(t, ctx, pool, "svc-api")

	r := sampleRegression("svc-api")
	r.DeployID = deployID

	require.NoError(t, store.Save(ctx, r))

	got, err := store.Get(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, deployID, got.DeployID)
}

// TestList_Filters verifies that the List method filters correctly.
func TestList_Filters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	deployID := insertDeploy(t, ctx, pool, "svc-order")

	r1 := sampleRegression("svc-order")
	r1.DeployID = deployID
	r1.Severity = regression.SeverityCritical
	r1.Status = "active"

	r2 := sampleRegression("svc-payment")
	r2.Severity = regression.SeverityWarning
	r2.Status = "dismissed"

	r3 := sampleRegression("svc-order")
	r3.DeployID = deployID
	r3.Severity = regression.SeverityInfo
	r3.Algorithm = regression.AlgoErrorRate
	r3.Status = "active"

	require.NoError(t, store.Save(ctx, r1))
	require.NoError(t, store.Save(ctx, r2))
	require.NoError(t, store.Save(ctx, r3))

	t.Run("filter by service", func(t *testing.T) {
		got, err := store.List(ctx, regression.RegressionFilter{Service: "svc-order"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, g := range got {
			assert.Equal(t, "svc-order", g.Service)
		}
	})

	t.Run("filter by severity", func(t *testing.T) {
		got, err := store.List(ctx, regression.RegressionFilter{Severity: regression.SeverityCritical})
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, regression.SeverityCritical, got[0].Severity)
	})

	t.Run("filter by status", func(t *testing.T) {
		got, err := store.List(ctx, regression.RegressionFilter{Status: "dismissed"})
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "dismissed", got[0].Status)
	})

	t.Run("filter by deploy_id", func(t *testing.T) {
		got, err := store.List(ctx, regression.RegressionFilter{DeployID: deployID})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, g := range got {
			assert.Equal(t, deployID, g.DeployID)
		}
	})

	t.Run("limit", func(t *testing.T) {
		got, err := store.List(ctx, regression.RegressionFilter{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}

// TestUpdateStatus verifies that status transitions work and resolved_at is set
// appropriately.
func TestUpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	r := sampleRegression("svc-cache")
	require.NoError(t, store.Save(ctx, r))

	// Transition to dismissed.
	require.NoError(t, store.UpdateStatus(ctx, r.ID, "dismissed"))

	got, err := store.Get(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "dismissed", got.Status)
	assert.NotNil(t, got.ResolvedAt, "resolved_at should be set when dismissed")

	// Transition back to active (resolved_at not changed by store, just status).
	require.NoError(t, store.UpdateStatus(ctx, r.ID, "active"))

	got2, err := store.Get(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", got2.Status)
}

// TestUpdateStatus_ResolvedSetsTimestamp verifies resolved_at is set when
// status is "resolved".
func TestUpdateStatus_ResolvedSetsTimestamp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	r := sampleRegression("svc-db")
	require.NoError(t, store.Save(ctx, r))

	before := time.Now()
	require.NoError(t, store.UpdateStatus(ctx, r.ID, "resolved"))

	got, err := store.Get(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "resolved", got.Status)
	require.NotNil(t, got.ResolvedAt)
	assert.True(t, got.ResolvedAt.After(before) || got.ResolvedAt.Equal(before),
		"resolved_at should be >= the time before the update")
}

// TestActiveForDeploy verifies that only active regressions for a specific
// deploy are returned.
func TestActiveForDeploy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewStore(pool)

	deployA := insertDeploy(t, ctx, pool, "svc-gateway")
	deployB := insertDeploy(t, ctx, pool, "svc-edge")

	// Two active regressions for deploy A.
	r1 := sampleRegression("svc-gateway")
	r1.DeployID = deployA
	r1.Status = "active"

	r2 := sampleRegression("svc-gateway")
	r2.DeployID = deployA
	r2.Status = "active"

	// One dismissed regression for deploy A — should not appear.
	r3 := sampleRegression("svc-gateway")
	r3.DeployID = deployA
	r3.Status = "dismissed"

	// One active regression for deploy B — should not appear.
	r4 := sampleRegression("svc-edge")
	r4.DeployID = deployB
	r4.Status = "active"

	require.NoError(t, store.Save(ctx, r1))
	require.NoError(t, store.Save(ctx, r2))
	require.NoError(t, store.Save(ctx, r3))
	require.NoError(t, store.Save(ctx, r4))

	// r3 is saved as "dismissed" but UpdateStatus sets resolved_at server-side;
	// here we just saved it with status="dismissed" directly which is valid.

	got, err := store.ActiveForDeploy(ctx, deployA)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	for _, g := range got {
		assert.Equal(t, deployA, g.DeployID)
		assert.Equal(t, "active", g.Status)
	}
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
	assert.ErrorIs(t, err, regression.ErrNotFound)
}
