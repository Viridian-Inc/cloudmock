package regression_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Viridian-Inc/cloudmock/pkg/regression"
)

// ---------------------------------------------------------------------------
// Prometheus mock helpers (same pattern as pkg/dataplane/prometheus/metrics_test.go)
// ---------------------------------------------------------------------------

type promResponse struct {
	Status string      `json:"status"`
	Data   any `json:"data"`
}

type vectorData struct {
	ResultType string         `json:"resultType"`
	Result     []vectorSample `json:"result"`
}

type vectorSample struct {
	Metric map[string]string `json:"metric"`
	Value  [2]any    `json:"value"` // [unixTimestamp, "stringValue"]
}

func floatStr(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// buildMockServer creates an httptest.Server that serves the supplied values
// in sequence for each /api/v1/query call. The last value is repeated once
// all values have been consumed.
func buildMockServer(t *testing.T, values []float64) *httptest.Server {
	t.Helper()
	idx := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := values[idx]
		if idx < len(values)-1 {
			idx++
		}
		ts := float64(time.Now().Unix())
		resp := promResponse{
			Status: "success",
			Data: vectorData{
				ResultType: "vector",
				Result: []vectorSample{
					{
						Metric: map[string]string{},
						Value:  [2]any{ts, floatStr(v)},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// buildEmptyServer returns a server that always responds with an empty vector.
func buildEmptyServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := promResponse{
			Status: "success",
			Data: vectorData{
				ResultType: "vector",
				Result:     []vectorSample{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// buildErrorServer returns a server that always responds with a Prometheus
// error envelope.
func buildErrorServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := struct {
			Status    string `json:"status"`
			ErrorType string `json:"errorType"`
			Error     string `json:"error"`
		}{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     "mock error",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func newPromAPI(t *testing.T, srv *httptest.Server) promv1.API {
	t.Helper()
	c, err := promapi.NewClient(promapi.Config{Address: srv.URL})
	require.NoError(t, err)
	return promv1.NewAPI(c)
}

// ---------------------------------------------------------------------------
// WindowMetrics
// ---------------------------------------------------------------------------

func TestMetricSource_WindowMetrics_ReturnsCorrectValues(t *testing.T) {
	// WindowMetrics issues 5 Prometheus queries in order:
	// 1. p50   -> 0.05 s  (50 ms)
	// 2. p95   -> 0.2  s  (200 ms)
	// 3. p99   -> 0.5  s  (500 ms)
	// 4. error rate -> 0.02
	// 5. request count -> 1000
	srv := buildMockServer(t, []float64{0.05, 0.2, 0.5, 0.02, 1000})
	defer srv.Close()

	api := newPromAPI(t, srv)
	src := regression.NewMetricSource(api, nil)

	window := regression.TimeWindow{
		Start: time.Now().Add(-5 * time.Minute),
		End:   time.Now(),
	}
	wm, err := src.WindowMetrics(context.Background(), "my-svc", "GetUser", window)
	require.NoError(t, err)
	require.NotNil(t, wm)

	assert.Equal(t, "my-svc", wm.Service)
	assert.Equal(t, "GetUser", wm.Action)

	// Latencies converted from seconds to milliseconds.
	assert.InDelta(t, 50.0, wm.P50Ms, 1e-9)
	assert.InDelta(t, 200.0, wm.P95Ms, 1e-9)
	assert.InDelta(t, 500.0, wm.P99Ms, 1e-9)

	assert.InDelta(t, 0.02, wm.ErrorRate, 1e-9)
	assert.Equal(t, int64(1000), wm.RequestCount)
}

func TestMetricSource_WindowMetrics_EmptyResponse(t *testing.T) {
	srv := buildEmptyServer(t)
	defer srv.Close()

	api := newPromAPI(t, srv)
	src := regression.NewMetricSource(api, nil)

	window := regression.TimeWindow{
		Start: time.Now().Add(-5 * time.Minute),
		End:   time.Now(),
	}
	wm, err := src.WindowMetrics(context.Background(), "no-data-svc", "action", window)
	require.NoError(t, err)
	require.NotNil(t, wm)

	assert.Equal(t, 0.0, wm.P50Ms)
	assert.Equal(t, 0.0, wm.P95Ms)
	assert.Equal(t, 0.0, wm.P99Ms)
	assert.Equal(t, 0.0, wm.ErrorRate)
	assert.Equal(t, int64(0), wm.RequestCount)
}

func TestMetricSource_WindowMetrics_PrometheusError(t *testing.T) {
	srv := buildErrorServer(t)
	defer srv.Close()

	api := newPromAPI(t, srv)
	src := regression.NewMetricSource(api, nil)

	window := regression.TimeWindow{
		Start: time.Now().Add(-5 * time.Minute),
		End:   time.Now(),
	}
	_, err := src.WindowMetrics(context.Background(), "err-svc", "action", window)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "window metrics p50")
}

// ---------------------------------------------------------------------------
// TenantWindowMetrics
// ---------------------------------------------------------------------------

func TestMetricSource_TenantWindowMetrics_ReturnsP99(t *testing.T) {
	// Single query: tenant p99 -> 0.3 s (300 ms)
	srv := buildMockServer(t, []float64{0.3})
	defer srv.Close()

	api := newPromAPI(t, srv)
	src := regression.NewMetricSource(api, nil)

	window := regression.TimeWindow{
		Start: time.Now().Add(-5 * time.Minute),
		End:   time.Now(),
	}
	wm, err := src.TenantWindowMetrics(context.Background(), "my-svc", "tenant-123", window)
	require.NoError(t, err)
	require.NotNil(t, wm)

	assert.InDelta(t, 300.0, wm.P99Ms, 1e-9)
}

func TestMetricSource_TenantWindowMetrics_EmptyResponse(t *testing.T) {
	srv := buildEmptyServer(t)
	defer srv.Close()

	api := newPromAPI(t, srv)
	src := regression.NewMetricSource(api, nil)

	window := regression.TimeWindow{
		Start: time.Now().Add(-5 * time.Minute),
		End:   time.Now(),
	}
	wm, err := src.TenantWindowMetrics(context.Background(), "no-data-svc", "t1", window)
	require.NoError(t, err)
	require.NotNil(t, wm)
	assert.Equal(t, 0.0, wm.P99Ms)
}

// ---------------------------------------------------------------------------
// FleetWindowMetrics
// ---------------------------------------------------------------------------

func TestMetricSource_FleetWindowMetrics_ReturnsP99(t *testing.T) {
	// Single query: fleet p99 -> 0.8 s (800 ms)
	srv := buildMockServer(t, []float64{0.8})
	defer srv.Close()

	api := newPromAPI(t, srv)
	src := regression.NewMetricSource(api, nil)

	window := regression.TimeWindow{
		Start: time.Now().Add(-5 * time.Minute),
		End:   time.Now(),
	}
	wm, err := src.FleetWindowMetrics(context.Background(), "my-svc", window)
	require.NoError(t, err)
	require.NotNil(t, wm)

	assert.InDelta(t, 800.0, wm.P99Ms, 1e-9)
}

func TestMetricSource_FleetWindowMetrics_EmptyResponse(t *testing.T) {
	srv := buildEmptyServer(t)
	defer srv.Close()

	api := newPromAPI(t, srv)
	src := regression.NewMetricSource(api, nil)

	window := regression.TimeWindow{
		Start: time.Now().Add(-5 * time.Minute),
		End:   time.Now(),
	}
	wm, err := src.FleetWindowMetrics(context.Background(), "no-data-svc", window)
	require.NoError(t, err)
	require.NotNil(t, wm)
	assert.Equal(t, 0.0, wm.P99Ms)
}
