package duckdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neureaux/cloudmock/pkg/dataplane"
	duckstore "github.com/neureaux/cloudmock/pkg/dataplane/duckdb"
)

func setupTestDB(t *testing.T) *duckstore.Client {
	t.Helper()
	client, err := duckstore.NewClient(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.InitSchema(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestTraceWriteAndGet(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	store := duckstore.NewTraceStore(client)

	now := time.Now().UTC().Truncate(time.Millisecond)

	spans := []*dataplane.Span{
		{
			TraceID:    "aaaabbbbccccddddeeeeffffgggghhhh",
			SpanID:     "1111222233334444",
			Service:    "api-gateway",
			Action:     "HandleRequest",
			Method:     "GET",
			Path:       "/api/users",
			StartTime:  now,
			EndTime:    now.Add(100 * time.Millisecond),
			StatusCode: 200,
			TenantID:   "tenant-1",
			OrgID:      "org-1",
			UserID:     "user-1",
			Metadata:   map[string]string{"env": "test"},
		},
		{
			TraceID:      "aaaabbbbccccddddeeeeffffgggghhhh",
			SpanID:       "2222333344445555",
			ParentSpanID: "1111222233334444",
			Service:      "user-service",
			Action:       "GetUsers",
			Method:       "GET",
			Path:         "/internal/users",
			StartTime:    now.Add(5 * time.Millisecond),
			EndTime:      now.Add(50 * time.Millisecond),
			StatusCode:   200,
			TenantID:     "tenant-1",
			OrgID:        "org-1",
			Metadata:     map[string]string{"db": "postgres"},
		},
		{
			TraceID:      "aaaabbbbccccddddeeeeffffgggghhhh",
			SpanID:       "3333444455556666",
			ParentSpanID: "1111222233334444",
			Service:      "cache-service",
			Action:       "CacheLookup",
			Method:       "GET",
			Path:         "/internal/cache",
			StartTime:    now.Add(2 * time.Millisecond),
			EndTime:      now.Add(10 * time.Millisecond),
			StatusCode:   200,
			TenantID:     "tenant-1",
		},
	}

	// Write spans.
	err := store.WriteSpans(ctx, spans)
	require.NoError(t, err)

	// Read back the trace.
	tc, err := store.Get(ctx, "aaaabbbbccccddddeeeeffffgggghhhh")
	require.NoError(t, err)
	require.NotNil(t, tc)

	assert.Equal(t, "aaaabbbbccccddddeeeeffffgggghhhh", tc.TraceID)
	assert.Equal(t, "1111222233334444", tc.SpanID)
	assert.Equal(t, "api-gateway", tc.Service)
	assert.Equal(t, "", tc.ParentSpanID)
	assert.Equal(t, 200, tc.StatusCode)
	require.Len(t, tc.Children, 2)

	// Verify children are the expected spans.
	childServices := map[string]bool{}
	for _, child := range tc.Children {
		childServices[child.Service] = true
		assert.Equal(t, "aaaabbbbccccddddeeeeffffgggghhhh", child.TraceID)
		assert.Equal(t, "1111222233334444", child.ParentSpanID)
	}
	assert.True(t, childServices["user-service"])
	assert.True(t, childServices["cache-service"])

	// Not found case.
	_, err = store.Get(ctx, "00000000000000000000000000000000")
	assert.ErrorIs(t, err, dataplane.ErrNotFound)
}

func TestTraceSearch(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	store := duckstore.NewTraceStore(client)

	now := time.Now().UTC().Truncate(time.Millisecond)

	// Write two traces: one ok, one with error.
	spans := []*dataplane.Span{
		{
			TraceID:    "aaaaaaaabbbbbbbbccccccccdddddddd",
			SpanID:     "aaaa111122223333",
			Service:    "api-gateway",
			Action:     "HandleOK",
			Method:     "GET",
			Path:       "/ok",
			StartTime:  now,
			EndTime:    now.Add(50 * time.Millisecond),
			StatusCode: 200,
			TenantID:   "tenant-search",
		},
		{
			TraceID:    "eeeeeeeefffffff0111111112222222a",
			SpanID:     "bbbb111122223333",
			Service:    "api-gateway",
			Action:     "HandleErr",
			Method:     "POST",
			Path:       "/fail",
			StartTime:  now.Add(time.Second),
			EndTime:    now.Add(time.Second + 200*time.Millisecond),
			StatusCode: 500,
			Error:      "internal error",
			TenantID:   "tenant-search",
		},
	}
	require.NoError(t, store.WriteSpans(ctx, spans))

	// Search all for this tenant.
	results, err := store.Search(ctx, dataplane.TraceFilter{
		TenantID: "tenant-search",
		Limit:    10,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)

	// Search by service.
	results, err = store.Search(ctx, dataplane.TraceFilter{
		Service: "api-gateway",
		Limit:   10,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Search with error filter.
	hasErr := true
	results, err = store.Search(ctx, dataplane.TraceFilter{
		HasError: &hasErr,
		TenantID: "tenant-search",
		Limit:    10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, results)
	for _, r := range results {
		assert.True(t, r.HasError)
	}
}

func TestTraceTimeline(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	store := duckstore.NewTraceStore(client)

	now := time.Now().UTC().Truncate(time.Millisecond)

	spans := []*dataplane.Span{
		{
			TraceID:    "tttttttttttttttttttttttttttttttt",
			SpanID:     "tl11111122223333",
			Service:    "root-svc",
			Action:     "Root",
			StartTime:  now,
			EndTime:    now.Add(100 * time.Millisecond),
			StatusCode: 200,
		},
		{
			TraceID:      "tttttttttttttttttttttttttttttttt",
			SpanID:       "tl22222233334444",
			ParentSpanID: "tl11111122223333",
			Service:      "child-svc",
			Action:       "Child",
			StartTime:    now.Add(10 * time.Millisecond),
			EndTime:      now.Add(60 * time.Millisecond),
			StatusCode:   200,
		},
	}
	require.NoError(t, store.WriteSpans(ctx, spans))

	tl, err := store.Timeline(ctx, "tttttttttttttttttttttttttttttttt")
	require.NoError(t, err)
	require.Len(t, tl, 2)

	// Root should have depth 0, offset 0.
	assert.Equal(t, 0, tl[0].Depth)
	assert.InDelta(t, 0, tl[0].StartOffsetMs, 0.1)

	// Child should have depth 1, non-zero offset.
	assert.Equal(t, 1, tl[1].Depth)
	assert.Greater(t, tl[1].StartOffsetMs, 0.0)

	// Not found.
	_, err = store.Timeline(ctx, "00000000000000000000000000000000")
	assert.ErrorIs(t, err, dataplane.ErrNotFound)
}
