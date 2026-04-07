package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	pgstore "github.com/Viridian-Inc/cloudmock/pkg/dataplane/postgres"
)

func TestSLOStore_SetAndGetRules(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewSLOStore(pool)

	// Set initial rules.
	rules := []config.SLORule{
		{Service: "dynamodb", Action: "Query", P50Ms: 10, P95Ms: 50, P99Ms: 100, ErrorRate: 0.01},
		{Service: "s3", Action: "*", P50Ms: 20, P95Ms: 80, P99Ms: 200, ErrorRate: 0.02},
	}
	require.NoError(t, store.SetRules(ctx, rules))

	// Verify rules are returned.
	got, err := store.Rules(ctx)
	require.NoError(t, err)
	require.Len(t, got, 2)

	// Verify the rules match (order may vary, so check by service).
	found := map[string]config.SLORule{}
	for _, r := range got {
		found[r.Service] = r
	}
	assert.Equal(t, "Query", found["dynamodb"].Action)
	assert.InDelta(t, 10.0, found["dynamodb"].P50Ms, 0.001)
	assert.Equal(t, "*", found["s3"].Action)
}

func TestSLOStore_SetRules_DeactivatesOld(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewSLOStore(pool)

	// Set first batch.
	rules1 := []config.SLORule{
		{Service: "dynamodb", Action: "*", P50Ms: 10, P95Ms: 50, P99Ms: 100, ErrorRate: 0.01},
	}
	require.NoError(t, store.SetRules(ctx, rules1))

	// Set second batch — old rules should be deactivated.
	rules2 := []config.SLORule{
		{Service: "lambda", Action: "Invoke", P50Ms: 5, P95Ms: 20, P99Ms: 50, ErrorRate: 0.005},
		{Service: "s3", Action: "*", P50Ms: 20, P95Ms: 80, P99Ms: 200, ErrorRate: 0.02},
	}
	require.NoError(t, store.SetRules(ctx, rules2))

	got, err := store.Rules(ctx)
	require.NoError(t, err)
	require.Len(t, got, 2, "only the new rules should be active")

	services := map[string]bool{}
	for _, r := range got {
		services[r.Service] = true
	}
	assert.True(t, services["lambda"])
	assert.True(t, services["s3"])
	assert.False(t, services["dynamodb"], "dynamodb rule should be deactivated")
}

func TestSLOStore_History(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewSLOStore(pool)

	// First set.
	require.NoError(t, store.SetRules(ctx, []config.SLORule{
		{Service: "dynamodb", Action: "*", P50Ms: 10, P95Ms: 50, P99Ms: 100, ErrorRate: 0.01},
	}))

	// Second set.
	require.NoError(t, store.SetRules(ctx, []config.SLORule{
		{Service: "s3", Action: "*", P50Ms: 20, P95Ms: 80, P99Ms: 200, ErrorRate: 0.02},
		{Service: "lambda", Action: "Invoke", P50Ms: 5, P95Ms: 20, P99Ms: 50, ErrorRate: 0.005},
	}))

	history, err := store.History(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, history, 3, "1 from first set + 2 from second set")

	// Most recent first.
	for _, h := range history {
		assert.Equal(t, "created", h.ChangeType)
		assert.NotNil(t, h.NewValues)
	}
}

func TestSLOStore_Status(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewSLOStore(pool)

	require.NoError(t, store.SetRules(ctx, []config.SLORule{
		{Service: "*", Action: "*", P50Ms: 50, P95Ms: 200, P99Ms: 500, ErrorRate: 0.01},
	}))

	status, err := store.Status(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.True(t, status.Healthy)
	assert.Len(t, status.Windows, 1)
	assert.Equal(t, "*", status.Windows[0].Service)
}
