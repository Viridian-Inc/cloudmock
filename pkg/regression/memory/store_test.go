package memory_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/regression"
	"github.com/Viridian-Inc/cloudmock/pkg/regression/memory"
)

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

func TestSaveAndGet(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

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
	assert.Equal(t, "active", got.Status)
	assert.Nil(t, got.ResolvedAt)
}

func TestSave_SetsDetectedAt(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	r := sampleRegression("svc-auth")
	r.DetectedAt = time.Time{} // zero value

	before := time.Now()
	require.NoError(t, store.Save(ctx, r))

	got, err := store.Get(ctx, r.ID)
	require.NoError(t, err)
	assert.False(t, got.DetectedAt.IsZero(), "DetectedAt should be set")
	assert.True(t, !got.DetectedAt.Before(before.Add(-time.Second)),
		"DetectedAt should be recent")
}

func TestGet_NotFound(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	_, err := store.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, regression.ErrNotFound)
}

func TestList_Filters(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	r1 := sampleRegression("svc-order")
	r1.DeployID = "deploy-a"
	r1.Severity = regression.SeverityCritical
	r1.Status = "active"

	r2 := sampleRegression("svc-payment")
	r2.Severity = regression.SeverityWarning
	r2.Status = "dismissed"

	r3 := sampleRegression("svc-order")
	r3.DeployID = "deploy-a"
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
		got, err := store.List(ctx, regression.RegressionFilter{DeployID: "deploy-a"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, g := range got {
			assert.Equal(t, "deploy-a", g.DeployID)
		}
	})

	t.Run("filter by algorithm", func(t *testing.T) {
		got, err := store.List(ctx, regression.RegressionFilter{Algorithm: regression.AlgoErrorRate})
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, regression.AlgoErrorRate, got[0].Algorithm)
	})

	t.Run("limit", func(t *testing.T) {
		got, err := store.List(ctx, regression.RegressionFilter{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("newest first", func(t *testing.T) {
		got, err := store.List(ctx, regression.RegressionFilter{})
		require.NoError(t, err)
		require.Len(t, got, 3)
		// r3 was saved last, should appear first
		assert.Equal(t, r3.ID, got[0].ID)
		assert.Equal(t, r2.ID, got[1].ID)
		assert.Equal(t, r1.ID, got[2].ID)
	})
}

func TestUpdateStatus(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	r := sampleRegression("svc-cache")
	require.NoError(t, store.Save(ctx, r))

	// Transition to dismissed.
	require.NoError(t, store.UpdateStatus(ctx, r.ID, "dismissed"))

	got, err := store.Get(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "dismissed", got.Status)
	assert.NotNil(t, got.ResolvedAt, "resolved_at should be set when dismissed")

	// Transition back to active.
	require.NoError(t, store.UpdateStatus(ctx, r.ID, "active"))

	got2, err := store.Get(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", got2.Status)
}

func TestUpdateStatus_ResolvedSetsTimestamp(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

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

func TestUpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	err := store.UpdateStatus(ctx, "nonexistent", "resolved")
	assert.ErrorIs(t, err, regression.ErrNotFound)
}

func TestActiveForDeploy(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	// Two active regressions for deploy A.
	r1 := sampleRegression("svc-gateway")
	r1.DeployID = "deploy-a"
	r1.Status = "active"

	r2 := sampleRegression("svc-gateway")
	r2.DeployID = "deploy-a"
	r2.Status = "active"

	// One dismissed regression for deploy A -- should not appear.
	r3 := sampleRegression("svc-gateway")
	r3.DeployID = "deploy-a"
	r3.Status = "dismissed"

	// One active regression for deploy B -- should not appear.
	r4 := sampleRegression("svc-edge")
	r4.DeployID = "deploy-b"
	r4.Status = "active"

	require.NoError(t, store.Save(ctx, r1))
	require.NoError(t, store.Save(ctx, r2))
	require.NoError(t, store.Save(ctx, r3))
	require.NoError(t, store.Save(ctx, r4))

	got, err := store.ActiveForDeploy(ctx, "deploy-a")
	require.NoError(t, err)
	assert.Len(t, got, 2)
	for _, g := range got {
		assert.Equal(t, "deploy-a", g.DeployID)
		assert.Equal(t, "active", g.Status)
	}
}

func TestWindowMetrics_Basic(t *testing.T) {
	log := gateway.NewRequestLog(100)
	trace := gateway.NewTraceStore(100)

	now := time.Now().UTC()
	window := regression.TimeWindow{
		Start: now.Add(-1 * time.Hour),
		End:   now.Add(1 * time.Hour),
	}

	// Add some request entries with varying latencies and status codes.
	for i := 0; i < 20; i++ {
		status := 200
		if i%5 == 0 { // 4 out of 20 are errors (i=0,5,10,15)
			status = 500
		}
		log.Add(gateway.RequestEntry{
			ID:         fmt.Sprintf("req-%d", i),
			TraceID:    fmt.Sprintf("trace-%d", i),
			Timestamp:  now.Add(time.Duration(i) * time.Second),
			Service:    "svc-test",
			Action:     "GetItem",
			Method:     "POST",
			Path:       "/",
			StatusCode: status,
			LatencyMs:  float64(10 + i*5), // 10, 15, 20, ..., 105
			RequestHeaders: map[string]string{
				"X-Tenant-Id": fmt.Sprintf("tenant-%d", i%3),
			},
			ResponseBody: "hello world", // 11 bytes
		})
	}

	src := memory.NewMetricSource(log, trace)

	t.Run("WindowMetrics", func(t *testing.T) {
		wm, err := src.WindowMetrics(context.Background(), "svc-test", "GetItem", window)
		require.NoError(t, err)
		assert.Equal(t, int64(20), wm.RequestCount)
		assert.Greater(t, wm.P50Ms, 0.0)
		assert.Greater(t, wm.P95Ms, wm.P50Ms)
		assert.GreaterOrEqual(t, wm.P99Ms, wm.P95Ms)
		assert.InDelta(t, 0.20, wm.ErrorRate, 0.01) // 4/20
		assert.InDelta(t, 11.0, wm.AvgRespSize, 0.01)
	})

	t.Run("FleetWindowMetrics", func(t *testing.T) {
		wm, err := src.FleetWindowMetrics(context.Background(), "svc-test", window)
		require.NoError(t, err)
		assert.Equal(t, int64(20), wm.RequestCount)
	})

	t.Run("TenantWindowMetrics", func(t *testing.T) {
		wm, err := src.TenantWindowMetrics(context.Background(), "svc-test", "tenant-0", window)
		require.NoError(t, err)
		// tenant-0 appears for i=0,3,6,9,12,15,18 => 7 entries
		assert.Equal(t, int64(7), wm.RequestCount)
	})

	t.Run("ListServices", func(t *testing.T) {
		services, err := src.ListServices(context.Background())
		require.NoError(t, err)
		assert.Contains(t, services, "svc-test")
	})

	t.Run("ListTenants", func(t *testing.T) {
		tenants, err := src.ListTenants(context.Background(), "svc-test")
		require.NoError(t, err)
		assert.Len(t, tenants, 3)
		assert.Contains(t, tenants, "tenant-0")
		assert.Contains(t, tenants, "tenant-1")
		assert.Contains(t, tenants, "tenant-2")
	})

	t.Run("empty window returns zero metrics", func(t *testing.T) {
		emptyWindow := regression.TimeWindow{
			Start: now.Add(-10 * time.Hour),
			End:   now.Add(-9 * time.Hour),
		}
		wm, err := src.WindowMetrics(context.Background(), "svc-test", "GetItem", emptyWindow)
		require.NoError(t, err)
		assert.Equal(t, int64(0), wm.RequestCount)
		assert.Equal(t, 0.0, wm.P50Ms)
	})
}
