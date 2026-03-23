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

func TestRequestWriteAndQuery(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	store := duckstore.NewRequestStore(client)

	now := time.Now().UTC().Truncate(time.Millisecond)

	entry := dataplane.RequestEntry{
		ID:         "req-span-1111111",
		TraceID:    "rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr",
		SpanID:     "req-span-1111111",
		Timestamp:  now,
		Service:    "order-service",
		Action:     "CreateOrder",
		Method:     "POST",
		Path:       "/api/orders",
		StatusCode: 201,
		Latency:    75 * time.Millisecond,
		LatencyMs:  75.0,
		CallerID:   "client-abc",
		Level:      "info",
		TenantID:   "tenant-req",
		OrgID:      "org-req",
		UserID:     "user-req",
		MemAllocKB: 128.5,
		Goroutines: 42,
		RequestHeaders: map[string]string{
			"Content-Type": "application/json",
		},
		RequestBody:  `{"item":"widget"}`,
		ResponseBody: `{"id":"order-123"}`,
	}

	require.NoError(t, store.Write(ctx, entry))

	// Write another entry with an error.
	errEntry := dataplane.RequestEntry{
		ID:         "req-span-2222222",
		TraceID:    "ssssssssssssssssssssssssssssssss",
		SpanID:     "req-span-2222222",
		Timestamp:  now.Add(time.Second),
		Service:    "order-service",
		Action:     "CreateOrder",
		Method:     "POST",
		Path:       "/api/orders",
		StatusCode: 500,
		Latency:    200 * time.Millisecond,
		LatencyMs:  200.0,
		Error:      "database timeout",
		TenantID:   "tenant-req",
	}
	require.NoError(t, store.Write(ctx, errEntry))

	// Query all for this service.
	results, err := store.Query(ctx, dataplane.RequestFilter{
		Service:  "order-service",
		TenantID: "tenant-req",
		Limit:    10,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)

	// Query by method.
	results, err = store.Query(ctx, dataplane.RequestFilter{
		Method:   "POST",
		TenantID: "tenant-req",
		Limit:    10,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)

	// Query errors only.
	results, err = store.Query(ctx, dataplane.RequestFilter{
		ErrorOnly: true,
		TenantID:  "tenant-req",
		Limit:     10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, results)
	for _, r := range results {
		assert.NotEmpty(t, r.Error)
	}

	// Query by latency range.
	results, err = store.Query(ctx, dataplane.RequestFilter{
		MinLatencyMs: 100,
		TenantID:     "tenant-req",
		Limit:        10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, results)
	for _, r := range results {
		assert.GreaterOrEqual(t, r.LatencyMs, 100.0)
	}

	// Query by time range.
	results, err = store.Query(ctx, dataplane.RequestFilter{
		From:     now.Add(-time.Minute),
		To:       now.Add(time.Minute),
		TenantID: "tenant-req",
		Limit:    10,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestRequestGetByID(t *testing.T) {
	ctx := context.Background()
	client := setupTestDB(t)
	store := duckstore.NewRequestStore(client)

	now := time.Now().UTC().Truncate(time.Millisecond)

	entry := dataplane.RequestEntry{
		ID:         "getbyid-span1111",
		TraceID:    "gggggggggggggggggggggggggggggggg",
		SpanID:     "getbyid-span1111",
		Timestamp:  now,
		Service:    "lookup-svc",
		Action:     "Lookup",
		Method:     "GET",
		Path:       "/api/lookup",
		StatusCode: 200,
		Latency:    25 * time.Millisecond,
		LatencyMs:  25.0,
		TenantID:   "tenant-getbyid",
	}
	require.NoError(t, store.Write(ctx, entry))

	got, err := store.GetByID(ctx, "getbyid-span1111")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "getbyid-span1111", got.SpanID)
	assert.Equal(t, "lookup-svc", got.Service)
	assert.Equal(t, "GET", got.Method)
	assert.Equal(t, 200, got.StatusCode)
	assert.InDelta(t, 25.0, got.LatencyMs, 0.1)

	// Not found case.
	_, err = store.GetByID(ctx, "nonexistent-00000")
	assert.ErrorIs(t, err, dataplane.ErrNotFound)
}
