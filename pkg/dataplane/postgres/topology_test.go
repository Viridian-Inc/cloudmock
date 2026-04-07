package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
	pgstore "github.com/Viridian-Inc/cloudmock/pkg/dataplane/postgres"
)

func TestTopologyStore_RecordEdge_And_GetTopology(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewTopologyStore(pool)

	require.NoError(t, store.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "api", Target: "dynamodb", EdgeType: "http", RequestCount: 1,
	}))
	require.NoError(t, store.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "api", Target: "s3", EdgeType: "http", RequestCount: 1,
	}))

	graph, err := store.GetTopology(ctx)
	require.NoError(t, err)
	assert.Len(t, graph.Edges, 2)
	assert.Len(t, graph.Nodes, 3, "api, dynamodb, s3")
}

func TestTopologyStore_RecordEdge_Increments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewTopologyStore(pool)

	for i := 0; i < 5; i++ {
		require.NoError(t, store.RecordEdge(ctx, dataplane.ObservedEdge{
			Source: "api", Target: "dynamodb", EdgeType: "http", RequestCount: 1,
		}))
	}

	graph, err := store.GetTopology(ctx)
	require.NoError(t, err)
	require.Len(t, graph.Edges, 1, "should be upserted into one edge")
	// Initial insert sets count=1, then 4 increments = 5 total.
	assert.Equal(t, int64(5), graph.Edges[0].RequestCount)
}

func TestTopologyStore_Upstream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewTopologyStore(pool)

	require.NoError(t, store.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "api", Target: "dynamodb", EdgeType: "http",
	}))
	require.NoError(t, store.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "worker", Target: "dynamodb", EdgeType: "http",
	}))
	require.NoError(t, store.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "api", Target: "s3", EdgeType: "http",
	}))

	upstream, err := store.Upstream(ctx, "dynamodb")
	require.NoError(t, err)
	assert.Len(t, upstream, 2)

	found := map[string]bool{}
	for _, u := range upstream {
		found[u] = true
	}
	assert.True(t, found["api"])
	assert.True(t, found["worker"])
}

func TestTopologyStore_Downstream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewTopologyStore(pool)

	require.NoError(t, store.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "api", Target: "dynamodb", EdgeType: "http",
	}))
	require.NoError(t, store.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "api", Target: "s3", EdgeType: "http",
	}))
	require.NoError(t, store.RecordEdge(ctx, dataplane.ObservedEdge{
		Source: "api", Target: "lambda", EdgeType: "http",
	}))

	downstream, err := store.Downstream(ctx, "api")
	require.NoError(t, err)
	assert.Len(t, downstream, 3)
}

func TestTopologyStore_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	store := pgstore.NewTopologyStore(pool)

	graph, err := store.GetTopology(ctx)
	require.NoError(t, err)
	assert.Empty(t, graph.Nodes)
	assert.Empty(t, graph.Edges)

	upstream, err := store.Upstream(ctx, "anything")
	require.NoError(t, err)
	assert.Empty(t, upstream)
}
