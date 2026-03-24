package prometheus_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neureaux/cloudmock/pkg/config"
	pkgprom "github.com/neureaux/cloudmock/pkg/dataplane/prometheus"
)

// promResponse is a minimal Prometheus HTTP API envelope.
type promResponse struct {
	Status string      `json:"status"`
	Data   any `json:"data"`
}

// vectorData matches the Prometheus instant-query response shape for a vector.
type vectorData struct {
	ResultType string          `json:"resultType"`
	Result     []vectorSample  `json:"result"`
}

type vectorSample struct {
	Metric map[string]string `json:"metric"`
	Value  [2]any    `json:"value"` // [unixTimestamp, "value"]
}

// buildMockServer creates an httptest.Server that replies to every
// /api/v1/query with the values supplied in order. After all values have been
// served the last value is repeated.
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
						Value:  [2]any{ts, formatFloat(v)},
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

func formatFloat(v float64) string {
	return strconv(v)
}

// strconv converts a float64 to its string representation using json encoding
// so that we avoid importing strconv and keep the test self-contained.
func strconv(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func newClientFromServer(t *testing.T, srv *httptest.Server) *pkgprom.Client {
	t.Helper()
	c, err := pkgprom.NewClient(config.PrometheusConfig{URL: srv.URL})
	require.NoError(t, err)
	return c
}

// ---------------------------------------------------------------------------
// ServiceStats
// ---------------------------------------------------------------------------

func TestServiceStats_ReturnsCorrectValues(t *testing.T) {
	// The MetricReader issues 5 queries in order:
	// 1. request rate   -> 10 req/s
	// 2. error rate     -> 1 req/s
	// 3. p50 latency    -> 0.05 s  (50 ms)
	// 4. p95 latency    -> 0.2  s  (200 ms)
	// 5. p99 latency    -> 0.5  s  (500 ms)
	srv := buildMockServer(t, []float64{10, 1, 0.05, 0.2, 0.5})
	defer srv.Close()

	c := newClientFromServer(t, srv)
	reader := pkgprom.NewMetricReader(c)

	window := 5 * time.Minute
	stats, err := reader.ServiceStats(context.Background(), "my-svc", window)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, "my-svc", stats.Service)

	// RequestCount = round(10 req/s * 300 s) = 3000
	assert.Equal(t, int64(3000), stats.RequestCount)
	// ErrorCount = round(1 req/s * 300 s) = 300
	assert.Equal(t, int64(300), stats.ErrorCount)
	// ErrorRate = 1/10 = 0.1
	assert.InDelta(t, 0.1, stats.ErrorRate, 1e-9)

	// Latencies converted from seconds to milliseconds
	assert.InDelta(t, 50.0, stats.P50Ms, 1e-9)
	assert.InDelta(t, 200.0, stats.P95Ms, 1e-9)
	assert.InDelta(t, 500.0, stats.P99Ms, 1e-9)
}

func TestServiceStats_EmptyResponse(t *testing.T) {
	srv := buildEmptyServer(t)
	defer srv.Close()

	c := newClientFromServer(t, srv)
	reader := pkgprom.NewMetricReader(c)

	stats, err := reader.ServiceStats(context.Background(), "no-data-svc", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, int64(0), stats.RequestCount)
	assert.Equal(t, int64(0), stats.ErrorCount)
	assert.Equal(t, 0.0, stats.ErrorRate)
	assert.Equal(t, 0.0, stats.P50Ms)
	assert.Equal(t, 0.0, stats.P95Ms)
	assert.Equal(t, 0.0, stats.P99Ms)
}

func TestServiceStats_ErrorResponse(t *testing.T) {
	srv := buildErrorServer(t)
	defer srv.Close()

	c := newClientFromServer(t, srv)
	reader := pkgprom.NewMetricReader(c)

	_, err := reader.ServiceStats(context.Background(), "err-svc", time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service stats request rate")
}

// ---------------------------------------------------------------------------
// Percentiles
// ---------------------------------------------------------------------------

func TestPercentiles_ReturnsCorrectValues(t *testing.T) {
	// Queries in order: p50 -> p95 -> p99  (in seconds)
	srv := buildMockServer(t, []float64{0.1, 0.3, 0.8})
	defer srv.Close()

	c := newClientFromServer(t, srv)
	reader := pkgprom.NewMetricReader(c)

	pct, err := reader.Percentiles(context.Background(), "my-svc", "GetUser", 10*time.Minute)
	require.NoError(t, err)
	require.NotNil(t, pct)

	assert.InDelta(t, 100.0, pct.P50Ms, 1e-9)
	assert.InDelta(t, 300.0, pct.P95Ms, 1e-9)
	assert.InDelta(t, 800.0, pct.P99Ms, 1e-9)
}

func TestPercentiles_EmptyResponse(t *testing.T) {
	srv := buildEmptyServer(t)
	defer srv.Close()

	c := newClientFromServer(t, srv)
	reader := pkgprom.NewMetricReader(c)

	pct, err := reader.Percentiles(context.Background(), "no-data-svc", "action", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, pct)

	assert.Equal(t, 0.0, pct.P50Ms)
	assert.Equal(t, 0.0, pct.P95Ms)
	assert.Equal(t, 0.0, pct.P99Ms)
}

// ---------------------------------------------------------------------------
// MetricWriter (no-op)
// ---------------------------------------------------------------------------

func TestMetricWriter_RecordIsNoOp(t *testing.T) {
	w := pkgprom.NewMetricWriter()
	err := w.Record(context.Background(), "svc", "action", 42.0, 200)
	assert.NoError(t, err)
}
