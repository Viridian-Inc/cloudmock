package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/admin"
	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
	ddbsvc "github.com/neureaux/cloudmock/services/dynamodb"
	s3svc "github.com/neureaux/cloudmock/services/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeService implements service.Service and admin.Resettable.
type fakeService struct {
	name       string
	actions    []service.Action
	healthy    bool
	resetCalls int
}

func (f *fakeService) Name() string             { return f.name }
func (f *fakeService) Actions() []service.Action { return f.actions }
func (f *fakeService) HealthCheck() error {
	if !f.healthy {
		return fmt.Errorf("unhealthy")
	}
	return nil
}
func (f *fakeService) HandleRequest(_ *service.RequestContext) (*service.Response, error) {
	return nil, nil
}
func (f *fakeService) Reset() {
	f.resetCalls++
}

func newTestAPI(t *testing.T, svcs ...service.Service) (*admin.API, *routing.Registry) {
	t.Helper()
	cfg := config.Default()
	reg := routing.NewRegistry()
	for _, svc := range svcs {
		reg.Register(svc)
	}
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	return api, reg
}

// newTestAPIWithLogAndStats creates an API with explicit access to the log and stats.
func newTestAPIWithLogAndStats(t *testing.T) (*admin.API, *gateway.RequestLog, *gateway.RequestStats) {
	t.Helper()
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	return api, rl, rs
}

// newDataPlaneTestAPI creates an admin API using NewWithDataPlane with a nil
// DataPlane. This registers ALL routes (including incidents, webhooks,
// profiling, sourcemaps) while using in-memory / nil-fallback paths.
func newDataPlaneTestAPI(t *testing.T) *admin.API {
	t.Helper()
	cfg := config.Default()
	reg := routing.NewRegistry()
	return admin.NewWithDataPlane(cfg, reg, nil)
}

// ---------------------------------------------------------------------------
// Service endpoints
// ---------------------------------------------------------------------------

func TestListServices(t *testing.T) {
	svc := &fakeService{
		name:    "s3",
		actions: []service.Action{{Name: "ListBuckets"}, {Name: "CreateBucket"}},
		healthy: true,
	}
	api, _ := newTestAPI(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var infos []admin.ServiceInfo
	require.NoError(t, json.NewDecoder(w.Body).Decode(&infos))
	require.Len(t, infos, 1)
	assert.Equal(t, "s3", infos[0].Name)
	assert.Equal(t, 2, infos[0].ActionCount)
	assert.True(t, infos[0].Healthy)
}

func TestListServices_Empty(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var infos []admin.ServiceInfo
	require.NoError(t, json.NewDecoder(w.Body).Decode(&infos))
	assert.Empty(t, infos)
}

func TestListServices_Multiple(t *testing.T) {
	svc1 := &fakeService{name: "s3", healthy: true, actions: []service.Action{{Name: "A"}}}
	svc2 := &fakeService{name: "dynamodb", healthy: true, actions: []service.Action{{Name: "B"}, {Name: "C"}}}
	api, _ := newTestAPI(t, svc1, svc2)

	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var infos []admin.ServiceInfo
	require.NoError(t, json.NewDecoder(w.Body).Decode(&infos))
	assert.Len(t, infos, 2)
}

func TestGetService(t *testing.T) {
	svc := &fakeService{name: "dynamodb", healthy: true}
	api, _ := newTestAPI(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/services/dynamodb", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var info admin.ServiceInfo
	require.NoError(t, json.NewDecoder(w.Body).Decode(&info))
	assert.Equal(t, "dynamodb", info.Name)
}

func TestGetService_NotFound(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/services/nonexistent", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestResetService(t *testing.T) {
	svc := &fakeService{name: "sqs", healthy: true}
	api, _ := newTestAPI(t, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/services/sqs/reset", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, svc.resetCalls)
}

func TestResetAll(t *testing.T) {
	svc1 := &fakeService{name: "s3", healthy: true}
	svc2 := &fakeService{name: "sqs", healthy: true}
	api, _ := newTestAPI(t, svc1, svc2)

	req := httptest.NewRequest(http.MethodPost, "/api/reset", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, svc1.resetCalls)
	assert.Equal(t, 1, svc2.resetCalls)
}

// ---------------------------------------------------------------------------
// Health endpoint
// ---------------------------------------------------------------------------

func TestHealth(t *testing.T) {
	svc := &fakeService{name: "s3", healthy: true}
	api, _ := newTestAPI(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admin.HealthResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "healthy", resp.Status)
	assert.True(t, resp.Services["s3"])
}

func TestHealth_Degraded(t *testing.T) {
	svc := &fakeService{name: "s3", healthy: false}
	api, _ := newTestAPI(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admin.HealthResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "degraded", resp.Status)
}

func TestHealth_NoServices(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admin.HealthResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "healthy", resp.Status)
	assert.NotNil(t, resp.Services)
}

func TestHealth_HasServicesKey(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	var raw map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&raw))
	_, ok := raw["services"]
	assert.True(t, ok, "response should contain 'services' key")
}

func TestHealth_MultipleServices(t *testing.T) {
	svc1 := &fakeService{name: "s3", healthy: true}
	svc2 := &fakeService{name: "dynamodb", healthy: false}
	api, _ := newTestAPI(t, svc1, svc2)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admin.HealthResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "degraded", resp.Status)
	assert.True(t, resp.Services["s3"])
	assert.False(t, resp.Services["dynamodb"])
}

// ---------------------------------------------------------------------------
// Version endpoint
// ---------------------------------------------------------------------------

func TestVersionEndpoint(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	_, hasVersion := resp["version"]
	assert.True(t, hasVersion, "response should contain 'version' key")
	_, hasBuildTime := resp["build_time"]
	assert.True(t, hasBuildTime, "response should contain 'build_time' key")
}

// ---------------------------------------------------------------------------
// Config endpoint
// ---------------------------------------------------------------------------

func TestConfig(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var cfg config.Config
	require.NoError(t, json.NewDecoder(w.Body).Decode(&cfg))
	assert.Equal(t, "us-east-1", cfg.Region)
}

func TestConfig_HasJSON(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

// ---------------------------------------------------------------------------
// Stats endpoint
// ---------------------------------------------------------------------------

func TestStats(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	rs.Increment("s3")
	rs.Increment("s3")
	rs.Increment("dynamodb")

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats map[string]int64
	require.NoError(t, json.NewDecoder(w.Body).Decode(&stats))
	assert.Equal(t, int64(2), stats["s3"])
	assert.Equal(t, int64(1), stats["dynamodb"])
}

func TestStats_Empty(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats map[string]int64
	require.NoError(t, json.NewDecoder(w.Body).Decode(&stats))
	assert.Empty(t, stats)
}

// ---------------------------------------------------------------------------
// Requests endpoint
// ---------------------------------------------------------------------------

func TestRequests(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	rl.Add(gateway.RequestEntry{Service: "s3", Action: "ListBuckets"})
	rl.Add(gateway.RequestEntry{Service: "dynamodb", Action: "PutItem"})
	rl.Add(gateway.RequestEntry{Service: "s3", Action: "GetObject"})

	api := admin.New(cfg, reg, rl, rs)

	// All requests (level=all to include infra-level entries)
	req := httptest.NewRequest(http.MethodGet, "/api/requests?level=all", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entries []gateway.RequestEntry
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
	assert.Len(t, entries, 3)

	// Filtered by service
	req = httptest.NewRequest(http.MethodGet, "/api/requests?service=s3&limit=10&level=all", nil)
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	entries = nil
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
	assert.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, "s3", e.Service)
	}
}

func TestRequests_Empty(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/requests?level=all", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entries []gateway.RequestEntry
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
	assert.Empty(t, entries)
}

func TestRequests_LimitParam(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	for i := 0; i < 10; i++ {
		rl.Add(gateway.RequestEntry{Service: "s3", Action: "ListBuckets"})
	}

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/requests?limit=3&level=all", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entries []gateway.RequestEntry
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
	assert.LessOrEqual(t, len(entries), 3)
}

func TestRequests_ServiceFilter(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	rl.Add(gateway.RequestEntry{Service: "s3", Action: "ListBuckets"})
	rl.Add(gateway.RequestEntry{Service: "dynamodb", Action: "PutItem"})
	rl.Add(gateway.RequestEntry{Service: "s3", Action: "GetObject"})

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/requests?service=dynamodb&level=all", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entries []gateway.RequestEntry
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
	assert.Len(t, entries, 1)
	assert.Equal(t, "dynamodb", entries[0].Service)
}

// ---------------------------------------------------------------------------
// Traces endpoint
// ---------------------------------------------------------------------------

func TestTracesEndpoint(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/traces", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// When no trace store is configured, should return an empty array.
	var raw json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&raw))
	assert.True(t, len(raw) > 0)
}

func TestTracesEndpoint_WithStore(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ts := gateway.NewTraceStore(100)
	api.SetTraceStore(ts)

	req := httptest.NewRequest(http.MethodGet, "/api/traces", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Topology endpoint
// ---------------------------------------------------------------------------

func TestTopologyEndpoint(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/topology", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	_, hasNodes := resp["nodes"]
	assert.True(t, hasNodes, "topology response should have 'nodes' key")
	_, hasEdges := resp["edges"]
	assert.True(t, hasEdges, "topology response should have 'edges' key")
}

func TestTopologyEndpoint_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/topology", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Topology config endpoint
// ---------------------------------------------------------------------------

func TestTopologyConfigEndpoint_Get(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/topology/config", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTopologyConfigEndpoint_PutAndGet(t *testing.T) {
	api, _ := newTestAPI(t)

	// PUT a topology config
	body := `{"nodes":[{"id":"n1","label":"Node1"}],"edges":[{"source":"n1","target":"n2"}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/topology/config", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var putResp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&putResp))
	assert.Equal(t, "ok", putResp["status"])

	// GET it back
	req = httptest.NewRequest(http.MethodGet, "/api/topology/config", nil)
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var cfg map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&cfg))
	_, hasNodes := cfg["nodes"]
	assert.True(t, hasNodes)
}

func TestTopologyConfigEndpoint_InvalidJSON(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPut, "/api/topology/config", bytes.NewBufferString("{invalid"))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// SLO endpoint
// ---------------------------------------------------------------------------

func TestSLOEndpoint_NoEngine(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/slo", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	// Without an SLO engine, should report enabled: false.
	assert.Equal(t, false, resp["enabled"])
}

func TestSLOEndpoint_WithEngine(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)

	sloEngine := gateway.NewSLOEngine(nil)
	api.SetSLOEngine(sloEngine)

	req := httptest.NewRequest(http.MethodGet, "/api/slo", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Views CRUD endpoints
// ---------------------------------------------------------------------------

func TestViewsCRUD(t *testing.T) {
	api, _ := newTestAPI(t)

	// List — initially empty.
	t.Run("list_empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/views", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var views []admin.SavedView
		require.NoError(t, json.NewDecoder(w.Body).Decode(&views))
		assert.Empty(t, views)
	})

	// Create a view.
	var createdID string
	t.Run("create", func(t *testing.T) {
		body := `{"name":"my-view","filters":{"service":"s3"}}`
		req := httptest.NewRequest(http.MethodPost, "/api/views", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var view admin.SavedView
		require.NoError(t, json.NewDecoder(w.Body).Decode(&view))
		assert.Equal(t, "my-view", view.Name)
		assert.NotEmpty(t, view.ID)
		createdID = view.ID
	})

	// List — should include the view.
	t.Run("list_after_create", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/views", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var views []admin.SavedView
		require.NoError(t, json.NewDecoder(w.Body).Decode(&views))
		require.Len(t, views, 1)
		assert.Equal(t, createdID, views[0].ID)
	})

	// Delete the view.
	t.Run("delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/views?id="+createdID, nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	// List — should be empty again.
	t.Run("list_after_delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/views", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var views []admin.SavedView
		require.NoError(t, json.NewDecoder(w.Body).Decode(&views))
		assert.Empty(t, views)
	})
}

func TestViews_DeleteMissingID(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/views", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestViews_DeleteNotFound(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/views?id=nonexistent", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestViews_InvalidJSON(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/views", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestViews_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/views", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Chaos CRUD endpoints
// ---------------------------------------------------------------------------

func TestChaosCRUD(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ce := gateway.NewChaosEngine()
	api.SetChaosEngine(ce)

	// GET — initially empty rules.
	t.Run("list_empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/chaos", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		rules, ok := resp["rules"].([]any)
		require.True(t, ok)
		assert.Empty(t, rules)
	})

	// POST — create a rule.
	var ruleID string
	t.Run("create", func(t *testing.T) {
		body := `{"service":"s3","action":"*","enabled":true,"type":"error","errorCode":500,"percentage":100}`
		req := httptest.NewRequest(http.MethodPost, "/api/chaos", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var rule gateway.ChaosRule
		require.NoError(t, json.NewDecoder(w.Body).Decode(&rule))
		assert.NotEmpty(t, rule.ID)
		assert.Equal(t, "s3", rule.Service)
		ruleID = rule.ID
	})

	// GET — should include the created rule.
	t.Run("list_after_create", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/chaos", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		rules := resp["rules"].([]any)
		assert.Len(t, rules, 1)
		assert.Equal(t, true, resp["active"])
	})

	// DELETE /api/chaos/:id — delete a specific rule.
	t.Run("delete_specific", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/chaos/"+ruleID, nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Confirm deleted.
	t.Run("list_after_delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/chaos", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		rules := resp["rules"].([]any)
		assert.Empty(t, rules)
	})
}

func TestChaos_NoEngine(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/chaos", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChaos_DisableAll(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ce := gateway.NewChaosEngine()
	api.SetChaosEngine(ce)

	// Create a rule first
	body := `{"service":"s3","action":"*","enabled":true,"type":"error","errorCode":500,"percentage":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/chaos", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// DELETE /api/chaos disables all.
	req = httptest.NewRequest(http.MethodDelete, "/api/chaos", nil)
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "all_disabled", resp["status"])
}

func TestChaos_InvalidJSON(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ce := gateway.NewChaosEngine()
	api.SetChaosEngine(ce)

	req := httptest.NewRequest(http.MethodPost, "/api/chaos", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Deploys endpoint
// ---------------------------------------------------------------------------

func TestDeploysEndpoint(t *testing.T) {
	api, _ := newTestAPI(t)

	// List — initially empty.
	t.Run("list_empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/deploys", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var deploys []json.RawMessage
		require.NoError(t, json.NewDecoder(w.Body).Decode(&deploys))
		assert.Empty(t, deploys)
	})

	// POST — create a deploy event.
	t.Run("create", func(t *testing.T) {
		body := `{"service":"api-gateway","commit":"abc123","author":"ci-bot","message":"deploy v1.2.0","branch":"main"}`
		req := httptest.NewRequest(http.MethodPost, "/api/deploys", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var deploy map[string]any
		require.NoError(t, json.NewDecoder(w.Body).Decode(&deploy))
		assert.NotEmpty(t, deploy["id"])
		assert.Equal(t, "api-gateway", deploy["service"])
	})

	// List — should include the deploy.
	t.Run("list_after_create", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/deploys", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var deploys []map[string]any
		require.NoError(t, json.NewDecoder(w.Body).Decode(&deploys))
		require.Len(t, deploys, 1)
		assert.Equal(t, "api-gateway", deploys[0]["service"])
	})
}

func TestDeploys_InvalidJSON(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/deploys", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeploys_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/deploys", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Preferences CRUD endpoints
// ---------------------------------------------------------------------------

func TestPreferencesCRUD(t *testing.T) {
	api, _ := newTestAPI(t)

	// PUT — set a preference.
	t.Run("set", func(t *testing.T) {
		body := `{"namespace":"test","key":"theme","value":"dark"}`
		req := httptest.NewRequest(http.MethodPut, "/api/preferences", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	// GET — retrieve the single preference.
	t.Run("get_single", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/preferences?namespace=test&key=theme", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "dark")
	})

	// GET — list all in namespace.
	t.Run("list_namespace", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/preferences?namespace=test", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
		_, hasTheme := result["theme"]
		assert.True(t, hasTheme)
	})

	// DELETE — remove the preference.
	t.Run("delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/preferences?namespace=test&key=theme", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	// GET — should be not found after delete.
	t.Run("get_after_delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/preferences?namespace=test&key=theme", nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestPreferences_MissingNamespace(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/preferences", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPreferences_PutMissingFields(t *testing.T) {
	api, _ := newTestAPI(t)

	body := `{"key":"x","value":"y"}`
	req := httptest.NewRequest(http.MethodPut, "/api/preferences", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPreferences_DeleteMissingParams(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/preferences?namespace=test", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPreferences_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/preferences", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestPreferences_InvalidJSON(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPut, "/api/preferences", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Audit endpoint
// ---------------------------------------------------------------------------

func TestAuditEndpoint_NoLogger(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/audit", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entries []json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
	assert.Empty(t, entries)
}

func TestAuditEndpoint_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/audit", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Resources endpoints
// ---------------------------------------------------------------------------

// newResourcesTestAPI creates an admin API with real S3 and DynamoDB services registered.
func newResourcesTestAPI(t *testing.T) *admin.API {
	t.Helper()
	cfg := config.Default()
	reg := routing.NewRegistry()

	reg.Register(s3svc.New())
	reg.Register(ddbsvc.New(cfg.AccountID, cfg.Region))

	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	return admin.New(cfg, reg, rl, rs)
}

func TestResources_S3(t *testing.T) {
	api := newResourcesTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/resources/s3", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp admin.ResourcesResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "s3", resp.Service)
	assert.NotNil(t, resp.Resources)
}

func TestResources_DynamoDB(t *testing.T) {
	api := newResourcesTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/resources/dynamodb", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp admin.ResourcesResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "dynamodb", resp.Service)
	assert.NotNil(t, resp.Resources)
}

func TestResources_NotFound(t *testing.T) {
	api := newResourcesTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/resources/nonexistent", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Method not allowed tests
// ---------------------------------------------------------------------------

func TestMethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/services"},
		{http.MethodPost, "/api/health"},
		{http.MethodPost, "/api/config"},
		{http.MethodPost, "/api/stats"},
		{http.MethodGet, "/api/reset"},
		{http.MethodGet, "/api/services/s3/reset"},
		{http.MethodPost, "/api/topology"},
		{http.MethodPost, "/api/requests"},
		{http.MethodPost, "/api/traces"},
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			api.ServeHTTP(w, req)
			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "%s %s", tt.method, tt.path)
		})
	}
}

// ---------------------------------------------------------------------------
// Error response format tests
// ---------------------------------------------------------------------------

func TestErrorResponseFormat(t *testing.T) {
	api, _ := newTestAPI(t)

	// Trigger an error via method not allowed.
	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var errResp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	_, hasError := errResp["error"]
	assert.True(t, hasError, "error response should have 'error' key")
}

func TestErrorResponseFormat_Views(t *testing.T) {
	api, _ := newTestAPI(t)

	// Missing id on DELETE returns structured error.
	req := httptest.NewRequest(http.MethodDelete, "/api/views", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	_, hasError := errResp["error"]
	assert.True(t, hasError)
}

// ---------------------------------------------------------------------------
// Not found endpoint
// ---------------------------------------------------------------------------

func TestNotFoundEndpoint(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Content-Type checks
// ---------------------------------------------------------------------------

func TestResponseContentType(t *testing.T) {
	api, _ := newTestAPI(t)

	endpoints := []string{
		"/api/health",
		"/api/version",
		"/api/config",
		"/api/stats",
		"/api/services",
	}

	for _, path := range endpoints {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			api.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		})
	}
}

// ---------------------------------------------------------------------------
// AdminAuthMiddleware tests
// ---------------------------------------------------------------------------

func TestAdminAuthMiddleware(t *testing.T) {
	api, _ := newTestAPI(t)
	protected := admin.AdminAuthMiddleware(api, "secret-key")

	t.Run("health_bypasses_auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("stream_bypasses_auth", func(t *testing.T) {
		// Stream endpoint is excluded from auth; we verify the middleware
		// does not reject it with 401 by using a cancelled context so the
		// SSE handler returns immediately instead of blocking forever.
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		req := httptest.NewRequest(http.MethodGet, "/api/stream", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing_key_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("wrong_key_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
		req.Header.Set("X-Admin-Key", "wrong-key")
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("correct_header_key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
		req.Header.Set("X-Admin-Key", "secret-key")
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("correct_query_key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/services?key=secret-key", nil)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// Chaos rule update via PUT /api/chaos/:id
// ---------------------------------------------------------------------------

func TestChaosRule_Update(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ce := gateway.NewChaosEngine()
	api.SetChaosEngine(ce)

	// Create a rule.
	body := `{"service":"s3","action":"*","enabled":true,"type":"error","errorCode":500,"percentage":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/chaos", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created gateway.ChaosRule
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))

	// Update the rule via PUT.
	updateBody := `{"service":"s3","action":"GetObject","enabled":false,"type":"latency","latencyMs":200,"percentage":50}`
	req = httptest.NewRequest(http.MethodPut, "/api/chaos/"+created.ID, bytes.NewBufferString(updateBody))
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated gateway.ChaosRule
	require.NoError(t, json.NewDecoder(w.Body).Decode(&updated))
	assert.Equal(t, "GetObject", updated.Action)
	assert.Equal(t, false, updated.Enabled)
}

func TestChaosRule_DeleteNotFound(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ce := gateway.NewChaosEngine()
	api.SetChaosEngine(ce)

	req := httptest.NewRequest(http.MethodDelete, "/api/chaos/nonexistent", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Multiple deploys
// ---------------------------------------------------------------------------

func TestDeploys_MultiplePosts(t *testing.T) {
	api, _ := newTestAPI(t)

	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{"service":"svc-%d","commit":"sha%d","author":"bot","message":"deploy","branch":"main"}`, i, i)
		req := httptest.NewRequest(http.MethodPost, "/api/deploys", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/deploys", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var deploys []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&deploys))
	assert.Len(t, deploys, 3)
}

// ---------------------------------------------------------------------------
// Views — multiple creates
// ---------------------------------------------------------------------------

func TestViews_MultipleCreates(t *testing.T) {
	api, _ := newTestAPI(t)

	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{"name":"view-%d","filters":{"service":"svc-%d"}}`, i, i)
		req := httptest.NewRequest(http.MethodPost, "/api/views", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/views", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	var views []admin.SavedView
	require.NoError(t, json.NewDecoder(w.Body).Decode(&views))
	assert.Len(t, views, 3)
}

// ---------------------------------------------------------------------------
// Preferences — multiple namespaces
// ---------------------------------------------------------------------------

func TestPreferences_MultipleNamespaces(t *testing.T) {
	api, _ := newTestAPI(t)

	// Set prefs in two namespaces.
	for _, ns := range []string{"ui", "editor"} {
		body := fmt.Sprintf(`{"namespace":"%s","key":"color","value":"blue"}`, ns)
		req := httptest.NewRequest(http.MethodPut, "/api/preferences", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	}

	// List one namespace — should only have its own keys.
	req := httptest.NewRequest(http.MethodGet, "/api/preferences?namespace=ui", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Len(t, result, 1)
	_, hasColor := result["color"]
	assert.True(t, hasColor)
}

// ---------------------------------------------------------------------------
// Requests — default level filtering
// ---------------------------------------------------------------------------

func TestRequests_DefaultLevel(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	rl.Add(gateway.RequestEntry{Service: "s3", Action: "ListBuckets"})

	api := admin.New(cfg, reg, rl, rs)

	// Without level= param, defaults to "app" level filtering.
	req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Lambda logs endpoint (nil fallback)
// ---------------------------------------------------------------------------

func TestLambdaLogs_NoBuffer(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/lambda/logs", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entries []json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
	assert.Empty(t, entries)
}

func TestLambdaLogs_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/lambda/logs", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// IAM evaluate endpoint (nil fallback)
// ---------------------------------------------------------------------------

func TestIAMEvaluate_NoEngine(t *testing.T) {
	api, _ := newTestAPI(t)

	body := `{"principal":"arn:aws:iam::123:user/test","action":"s3:GetObject","resource":"arn:aws:s3:::bucket/*"}`
	req := httptest.NewRequest(http.MethodPost, "/api/iam/evaluate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "DENY", resp["decision"])
}

func TestIAMEvaluate_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/iam/evaluate", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// SES emails endpoint (nil fallback)
// ---------------------------------------------------------------------------

func TestSESEmails_NoStore(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ses/emails", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var emails []json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&emails))
	assert.Empty(t, emails)
}

func TestSESEmails_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/ses/emails", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Request by ID endpoint
// ---------------------------------------------------------------------------

func TestRequestByID_NotFound(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/requests/nonexistent-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRequestByID_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/requests/some-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Tenants endpoint
// ---------------------------------------------------------------------------

func TestTenants_Empty(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenants", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenants_WithData(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	rl.Add(gateway.RequestEntry{Service: "s3", CallerID: "tenant-1", StatusCode: 200, LatencyMs: 10})
	rl.Add(gateway.RequestEntry{Service: "s3", CallerID: "tenant-1", StatusCode: 500, LatencyMs: 20})
	rl.Add(gateway.RequestEntry{Service: "dynamodb", CallerID: "tenant-2", StatusCode: 200, LatencyMs: 5})

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/tenants", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var tenants []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tenants))
	assert.Len(t, tenants, 2)
}

func TestTenants_ByID(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	rl.Add(gateway.RequestEntry{Service: "s3", CallerID: "tenant-1", StatusCode: 200, LatencyMs: 10})

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/tenants?id=tenant-1", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "tenant-1", resp["tenant_id"])
}

func TestTenants_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/tenants", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Cost endpoint
// ---------------------------------------------------------------------------

func TestCost_Empty(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/cost", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp, "total_cost_usd")
	assert.Contains(t, resp, "services")
}

func TestCost_WithData(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	rl.Add(gateway.RequestEntry{Service: "s3", Action: "GetObject"})
	rl.Add(gateway.RequestEntry{Service: "dynamodb", Action: "PutItem"})

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/cost", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	totalCost := resp["total_cost_usd"].(float64)
	assert.Greater(t, totalCost, float64(0))
}

func TestCost_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/cost", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Compare endpoint
// ---------------------------------------------------------------------------

func TestCompare_InsufficientData(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/compare?service=s3", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "insufficient data", resp["status"])
}

func TestCompare_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/compare", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Metrics endpoint
// ---------------------------------------------------------------------------

func TestMetrics_Empty(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMetrics_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/metrics", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Metrics timeline endpoint
// ---------------------------------------------------------------------------

func TestMetricsTimeline_Empty(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/timeline", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Regressions endpoint (nil fallback)
// ---------------------------------------------------------------------------

func TestRegressions_NoEngine(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/regressions", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Incidents endpoint (nil fallback — only registered in NewWithDataPlane)
// ---------------------------------------------------------------------------

func TestIncidents_NoService(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/incidents", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Webhooks endpoint (nil fallback — only registered in NewWithDataPlane)
// ---------------------------------------------------------------------------

func TestWebhooks_NoDispatcher(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/webhooks", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Profile/Profiling endpoints (nil fallback — only registered in NewWithDataPlane)
// ---------------------------------------------------------------------------

func TestProfile_NoEngine(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/profile/s3", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestProfiles_NoEngine(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/profiles", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Sourcemaps endpoint (nil fallback — only registered in NewWithDataPlane)
// ---------------------------------------------------------------------------

func TestSourcemaps_NoSymbolizer(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/sourcemaps?file=test.js", bytes.NewBufferString("sourcemap-data"))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Auth endpoints (nil user store fallbacks)
// ---------------------------------------------------------------------------

func TestAuthLogin_NoUserStore(t *testing.T) {
	api, _ := newTestAPI(t)

	body := `{"email":"test@example.com","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAuthLogin_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestAuthRegister_NoUserStore(t *testing.T) {
	api, _ := newTestAPI(t)

	body := `{"email":"test@example.com","password":"pass","name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAuthRegister_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/register", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestAuthMe_NoUser(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMe_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Users endpoint (nil user store fallback)
// ---------------------------------------------------------------------------

func TestUsers_NoUserStore(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUsers_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestUserByID_NoUserStore(t *testing.T) {
	api, _ := newTestAPI(t)

	body := `{"role":"admin"}`
	req := httptest.NewRequest(http.MethodPut, "/api/users/some-id", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserByID_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/users/some-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Blast radius endpoint
// ---------------------------------------------------------------------------

func TestBlastRadius_Empty(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/blast-radius?node=s3:bucket", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Shadow test endpoint
// ---------------------------------------------------------------------------

func TestShadowTest_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/shadow", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	// Shadow endpoint only accepts POST.
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Tenant export endpoint
// ---------------------------------------------------------------------------

func TestTenantExport_Empty(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenants/export?format=csv", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Trace compare endpoint (nil fallback)
// ---------------------------------------------------------------------------

func TestTraceCompare_NoComparer(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/traces/compare?a=trace1&b=trace2", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	// Trace compare not registered on legacy New() — routes to traces handler
	// which interprets "compare" as a trace ID.
	// The exact behavior depends on route matching, but should not panic.
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// Explain request endpoint
// ---------------------------------------------------------------------------

func TestExplainRequest_NotFound(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/explain/nonexistent-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestExplainRequest_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/explain/some-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestExplainRequest_Found(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	// Add the main request to explain.
	rl.Add(gateway.RequestEntry{
		ID:        "explain-req-1",
		Service:   "dynamodb",
		Action:    "Query",
		Method:    "POST",
		Path:      "/",
		StatusCode: 200,
		LatencyMs: 45.0,
		Timestamp: now,
	})
	// Add similar requests for percentile calculation.
	for i := 0; i < 10; i++ {
		rl.Add(gateway.RequestEntry{
			ID:        fmt.Sprintf("similar-%d", i),
			Service:   "dynamodb",
			Action:    "Query",
			StatusCode: 200,
			LatencyMs: float64(20 + i*3),
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
		})
	}

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/explain/explain-req-1", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp["request"])
	assert.NotNil(t, resp["analysis"])
	assert.NotEmpty(t, resp["narrative"])
}

func TestExplainRequest_ErrorRequest(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	rl.Add(gateway.RequestEntry{
		ID:        "explain-err-1",
		Service:   "s3",
		Action:    "GetObject",
		Method:    "GET",
		Path:      "/bucket/key",
		StatusCode: 500,
		LatencyMs: 200.0,
		Timestamp: now,
	})
	// Add some healthy requests for context.
	for i := 0; i < 5; i++ {
		rl.Add(gateway.RequestEntry{
			ID:        fmt.Sprintf("healthy-%d", i),
			Service:   "s3",
			Action:    "GetObject",
			StatusCode: 200,
			LatencyMs: float64(10 + i),
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
		})
	}

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/explain/explain-err-1", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	analysis := resp["analysis"].(map[string]any)
	assert.Equal(t, true, analysis["is_error"])
}

func TestExplainRequest_EmptyID(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/explain/", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExplainRequest_POST_WithRequestID(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	rl.Add(gateway.RequestEntry{
		ID:         "post-explain-1",
		Service:    "dynamodb",
		Action:     "Query",
		Method:     "POST",
		Path:       "/",
		StatusCode: 200,
		LatencyMs:  45.0,
		Timestamp:  now,
	})
	for i := 0; i < 5; i++ {
		rl.Add(gateway.RequestEntry{
			ID:         fmt.Sprintf("post-similar-%d", i),
			Service:    "dynamodb",
			Action:     "Query",
			StatusCode: 200,
			LatencyMs:  float64(20 + i*3),
			Timestamp:  now.Add(-time.Duration(i) * time.Minute),
		})
	}

	api := admin.New(cfg, reg, rl, rs)

	body := `{"request_id":"post-explain-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/explain/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp["request"])
	assert.NotNil(t, resp["analysis"])
	assert.NotEmpty(t, resp["narrative"])
}

func TestExplainRequest_POST_MissingRequestID(t *testing.T) {
	api, _ := newTestAPI(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/explain/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExplainRequest_POST_NotFound(t *testing.T) {
	api, _ := newTestAPI(t)

	body := `{"request_id":"nonexistent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/explain/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestExplainRequest_POST_InvalidJSON(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/explain/", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExplainRequest_MethodNotAllowed_PUT(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPut, "/api/explain/some-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// SLO PUT (update rules)
// ---------------------------------------------------------------------------

func TestSLO_UpdateRules(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)

	sloEngine := gateway.NewSLOEngine(nil)
	api.SetSLOEngine(sloEngine)

	body := `[{"service":"s3","target_p99_ms":100,"target_error_rate":0.01}]`
	req := httptest.NewRequest(http.MethodPut, "/api/slo", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestSLO_UpdateRules_InvalidJSON(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)

	sloEngine := gateway.NewSLOEngine(nil)
	api.SetSLOEngine(sloEngine)

	req := httptest.NewRequest(http.MethodPut, "/api/slo", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Cost routes / tenants / trend (nil fallback — only registered via NewWithDataPlane)
// ---------------------------------------------------------------------------

func TestCostRoutes_NoEngine(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/cost/routes", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCostTenants_NoEngine(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/cost/tenants", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCostTrend_NoEngine(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/cost/trend", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Metrics with data
// ---------------------------------------------------------------------------

func TestMetrics_WithData(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	rl.Add(gateway.RequestEntry{Service: "s3", Action: "GetObject", LatencyMs: 10, StatusCode: 200, Timestamp: now})
	rl.Add(gateway.RequestEntry{Service: "s3", Action: "PutObject", LatencyMs: 20, StatusCode: 200, Timestamp: now})
	rl.Add(gateway.RequestEntry{Service: "dynamodb", Action: "PutItem", LatencyMs: 5, StatusCode: 500, Timestamp: now})

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics?minutes=60", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var metrics []admin.ServiceMetrics
	require.NoError(t, json.NewDecoder(w.Body).Decode(&metrics))
	assert.NotEmpty(t, metrics)
}

func TestMetricsTimeline_WithData(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	rl.Add(gateway.RequestEntry{Service: "s3", LatencyMs: 10, StatusCode: 200, Timestamp: now})

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/timeline?minutes=5&bucket=1m", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Compare with sufficient data
// ---------------------------------------------------------------------------

func TestCompare_WithData(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	// Need at least 4 entries for comparison.
	for i := 0; i < 10; i++ {
		rl.Add(gateway.RequestEntry{
			Service:    "s3",
			Action:     "GetObject",
			LatencyMs:  float64(10 + i),
			StatusCode: 200,
			Timestamp:  now.Add(-time.Duration(i) * time.Minute),
		})
	}

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/compare?service=s3", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	// Should have before/after comparison, not "insufficient data".
	_, hasBefore := resp["before"]
	assert.True(t, hasBefore)
}

// ---------------------------------------------------------------------------
// Shadow test endpoint
// ---------------------------------------------------------------------------

func TestShadowTest_InvalidJSON(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/shadow", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShadowTest_MissingTarget(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/shadow", bytes.NewBufferString(`{"service":"s3"}`))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShadowTest_EmptyTraffic(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/shadow", bytes.NewBufferString(`{"target":"http://localhost:9999","service":"s3","limit":5}`))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(0), resp["count"])
}

// ---------------------------------------------------------------------------
// Blast radius with missing node param
// ---------------------------------------------------------------------------

func TestBlastRadius_MissingNode(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/blast-radius", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Tenant export with data
// ---------------------------------------------------------------------------

func TestTenantExport_WithData(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	rl.Add(gateway.RequestEntry{Service: "s3", CallerID: "tenant-1", StatusCode: 200, LatencyMs: 10})

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/tenants/export?format=csv", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "tenant-1")
}

// ---------------------------------------------------------------------------
// Request by ID — found
// ---------------------------------------------------------------------------

func TestRequestByID_Found(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	rl.Add(gateway.RequestEntry{ID: "req-123", Service: "s3", Action: "GetObject", StatusCode: 200})

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/requests/req-123", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var entry map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entry))
	assert.Equal(t, "req-123", entry["id"])
}

// ---------------------------------------------------------------------------
// SES email by ID — nil store
// ---------------------------------------------------------------------------

func TestSESEmailByID_NoStore(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ses/emails/some-msg-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSESEmailByID_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/ses/emails/some-msg-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Topology with registered services and IaC config
// ---------------------------------------------------------------------------

func TestTopology_WithServices(t *testing.T) {
	api := newResourcesTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/topology", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	_, hasGroups := resp["groups"]
	assert.True(t, hasGroups, "topology should have 'groups' key")
}

func TestTopology_WithIaCConfig(t *testing.T) {
	api, _ := newTestAPI(t)

	// Push IaC topology first
	body := `{"nodes":[{"id":"lambda:myFunc","label":"myFunc","service":"lambda","type":"function","group":"Compute"}],"edges":[{"source":"lambda:myFunc","target":"dynamodb:myTable","type":"read"}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/topology/config", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Now get topology — should include the IaC nodes
	req = httptest.NewRequest(http.MethodGet, "/api/topology", nil)
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	nodes := resp["nodes"].([]any)
	assert.GreaterOrEqual(t, len(nodes), 1)
}

// ---------------------------------------------------------------------------
// Blast radius with topology data
// ---------------------------------------------------------------------------

func TestBlastRadius_WithTopology(t *testing.T) {
	api, _ := newTestAPI(t)

	// Push IaC topology first
	body := `{"nodes":[{"id":"n1","label":"A"},{"id":"n2","label":"B"},{"id":"n3","label":"C"}],"edges":[{"source":"n1","target":"n2"},{"source":"n2","target":"n3"}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/topology/config", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Query blast radius for n2
	req = httptest.NewRequest(http.MethodGet, "/api/blast-radius?node=n2", nil)
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "n2", resp["node"])
}

// ---------------------------------------------------------------------------
// Trace compare via NewWithDataPlane (nil comparer)
// ---------------------------------------------------------------------------

func TestTraceCompare_NoComparer_DataPlane(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/traces/compare?a=t1&b=t2", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTraceCompare_MethodNotAllowed(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/traces/compare?a=t1&b=t2", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestTraceCompare_MissingParams(t *testing.T) {
	api := newDataPlaneTestAPI(t)

	// Missing both a and b — nil comparer returns 503 before param check.
	req := httptest.NewRequest(http.MethodGet, "/api/traces/compare", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ---------------------------------------------------------------------------
// Trace by ID — nil store
// ---------------------------------------------------------------------------

func TestTraceByID_NoStore(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/traces/some-trace-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTraceByID_Timeline_NoStore(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/traces/some-trace-id/timeline", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTraceByID_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/traces/some-trace-id", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestTraceByID_WithStore_Found(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ts := gateway.NewTraceStore(100)
	api.SetTraceStore(ts)

	now := time.Now()
	ts.Add(&gateway.TraceContext{
		TraceID:    "trace-abc",
		SpanID:     "span-1",
		Service:    "s3",
		Action:     "GetObject",
		StartTime:  now,
		EndTime:    now.Add(10 * time.Millisecond),
		Duration:   10 * time.Millisecond,
		DurationMs: 10,
		StatusCode: 200,
	})

	// Get trace by ID.
	req := httptest.NewRequest(http.MethodGet, "/api/traces/trace-abc", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Get timeline.
	req = httptest.NewRequest(http.MethodGet, "/api/traces/trace-abc/timeline", nil)
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTraceByID_WithStore_NotFound(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ts := gateway.NewTraceStore(100)
	api.SetTraceStore(ts)

	req := httptest.NewRequest(http.MethodGet, "/api/traces/nonexistent", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTraceByID_EmptyID(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ts := gateway.NewTraceStore(100)
	api.SetTraceStore(ts)

	req := httptest.NewRequest(http.MethodGet, "/api/traces/", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Explain request with slow request (triggers IsSlow anomaly)
// ---------------------------------------------------------------------------

func TestExplainRequest_SlowRequest(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	// Add a very slow request.
	rl.Add(gateway.RequestEntry{
		ID:        "explain-slow-1",
		Service:   "dynamodb",
		Action:    "Scan",
		Method:    "POST",
		Path:      "/",
		StatusCode: 200,
		LatencyMs: 500.0,
		Timestamp: now,
	})
	// Add normal-speed requests for baseline.
	for i := 0; i < 10; i++ {
		rl.Add(gateway.RequestEntry{
			ID:        fmt.Sprintf("normal-%d", i),
			Service:   "dynamodb",
			Action:    "Scan",
			StatusCode: 200,
			LatencyMs: float64(10 + i),
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
		})
	}

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/explain/explain-slow-1", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	analysis := resp["analysis"].(map[string]any)
	assert.Equal(t, true, analysis["is_slow"])
	anomalies := analysis["anomalies"].([]any)
	assert.NotEmpty(t, anomalies)
}

// ---------------------------------------------------------------------------
// Explain request with high error rate (triggers high error rate anomaly)
// ---------------------------------------------------------------------------

func TestExplainRequest_HighErrorRate(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	// Main request is an error.
	rl.Add(gateway.RequestEntry{
		ID:        "explain-high-err",
		Service:   "lambda",
		Action:    "Invoke",
		StatusCode: 500,
		LatencyMs: 30.0,
		Timestamp: now,
	})
	// All similar requests are errors (>50% error rate).
	for i := 0; i < 10; i++ {
		rl.Add(gateway.RequestEntry{
			ID:        fmt.Sprintf("err-%d", i),
			Service:   "lambda",
			Action:    "Invoke",
			StatusCode: 500,
			LatencyMs: float64(20 + i),
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
		})
	}

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/explain/explain-high-err", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Topology config method not allowed
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Setter and getter functions coverage
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Traces with filter params
// ---------------------------------------------------------------------------

func TestTraces_WithFilterParams(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ts := gateway.NewTraceStore(100)
	api.SetTraceStore(ts)

	now := time.Now()
	ts.Add(&gateway.TraceContext{
		TraceID:    "t1",
		SpanID:     "s1",
		Service:    "s3",
		StartTime:  now,
		EndTime:    now.Add(10 * time.Millisecond),
		DurationMs: 10,
		StatusCode: 200,
	})
	ts.Add(&gateway.TraceContext{
		TraceID:    "t2",
		SpanID:     "s2",
		Service:    "dynamodb",
		StartTime:  now,
		EndTime:    now.Add(10 * time.Millisecond),
		DurationMs: 10,
		StatusCode: 500,
		Error:      "something failed",
	})

	// Filter by service.
	req := httptest.NewRequest(http.MethodGet, "/api/traces?service=s3", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Filter by error.
	req = httptest.NewRequest(http.MethodGet, "/api/traces?error=true", nil)
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// With limit.
	req = httptest.NewRequest(http.MethodGet, "/api/traces?limit=1", nil)
	w = httptest.NewRecorder()
	api.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Explain request with deploy near request time (deploy anomaly detection)
// ---------------------------------------------------------------------------

func TestExplainRequest_WithNearbyDeploy(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	rl.Add(gateway.RequestEntry{
		ID:        "explain-deploy-1",
		Service:   "s3",
		Action:    "GetObject",
		StatusCode: 200,
		LatencyMs: 30.0,
		Timestamp: now,
	})

	api := admin.New(cfg, reg, rl, rs)

	// Create a deploy near the request time.
	deployBody := fmt.Sprintf(`{"service":"api","commit":"abc123","author":"ci","message":"v1","branch":"main","timestamp":"%s"}`, now.Add(-2*time.Minute).Format(time.RFC3339))
	dreq := httptest.NewRequest(http.MethodPost, "/api/deploys", bytes.NewBufferString(deployBody))
	dw := httptest.NewRecorder()
	api.ServeHTTP(dw, dreq)
	require.Equal(t, http.StatusCreated, dw.Code)

	// Now explain the request — should detect nearby deploy.
	req := httptest.NewRequest(http.MethodGet, "/api/explain/explain-deploy-1", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Explain request with unusual error (low error rate anomaly)
// ---------------------------------------------------------------------------

func TestExplainRequest_UnusualError(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	now := time.Now()
	// Main request is an error.
	rl.Add(gateway.RequestEntry{
		ID:        "explain-unusual-err",
		Service:   "sqs",
		Action:    "SendMessage",
		StatusCode: 500,
		LatencyMs: 15.0,
		Timestamp: now,
	})
	// But all other requests succeed — low error rate.
	for i := 0; i < 20; i++ {
		rl.Add(gateway.RequestEntry{
			ID:        fmt.Sprintf("ok-%d", i),
			Service:   "sqs",
			Action:    "SendMessage",
			StatusCode: 200,
			LatencyMs: float64(10 + i),
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
		})
	}

	api := admin.New(cfg, reg, rl, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/explain/explain-unusual-err", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	analysis := resp["analysis"].(map[string]any)
	assert.Equal(t, true, analysis["is_error"])
	// Should detect unusual error since error rate is very low.
	anomalies := analysis["anomalies"].([]any)
	assert.NotEmpty(t, anomalies)
}

func TestSettersAndGetters(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)

	// Broadcaster getter.
	b := api.Broadcaster()
	assert.NotNil(t, b)

	// ChaosEngine getter (nil before setting).
	assert.Nil(t, api.ChaosEngine())

	// SetChaosEngine + ChaosEngine.
	ce := gateway.NewChaosEngine()
	api.SetChaosEngine(ce)
	assert.Equal(t, ce, api.ChaosEngine())

	// SetTraceStore (already tested via traces tests, but verify no panic).
	ts := gateway.NewTraceStore(100)
	api.SetTraceStore(ts)

	// SetSLOEngine (already tested, just ensure no panic with nil).
	api.SetSLOEngine(nil)
}

// ---------------------------------------------------------------------------
// Explain with trace store (exercises trace integration in explain)
// ---------------------------------------------------------------------------

func TestExplainRequest_WithTraceStore(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)
	ts := gateway.NewTraceStore(100)
	api.SetTraceStore(ts)

	now := time.Now()

	// Add trace with children spans.
	ts.Add(&gateway.TraceContext{
		TraceID:    "trace-explain-1",
		SpanID:     "root-span",
		Service:    "bff",
		Action:     "HandleRequest",
		StartTime:  now,
		EndTime:    now.Add(50 * time.Millisecond),
		DurationMs: 50,
		StatusCode: 200,
	})
	ts.Add(&gateway.TraceContext{
		TraceID:      "trace-explain-1",
		SpanID:       "child-span-1",
		ParentSpanID: "root-span",
		Service:      "dynamodb",
		Action:       "Query",
		StartTime:    now.Add(5 * time.Millisecond),
		EndTime:      now.Add(25 * time.Millisecond),
		DurationMs:   20,
		StatusCode:   200,
	})
	ts.Add(&gateway.TraceContext{
		TraceID:      "trace-explain-1",
		SpanID:       "child-span-2",
		ParentSpanID: "root-span",
		Service:      "s3",
		Action:       "GetObject",
		StartTime:    now.Add(26 * time.Millisecond),
		EndTime:      now.Add(45 * time.Millisecond),
		DurationMs:   19,
		StatusCode:   200,
	})

	// Add the request entry with matching trace ID.
	rl.Add(gateway.RequestEntry{
		ID:        "req-with-trace",
		Service:   "bff",
		Action:    "HandleRequest",
		Method:    "GET",
		Path:      "/api/data",
		StatusCode: 200,
		LatencyMs: 50.0,
		Timestamp: now,
		TraceID:   "trace-explain-1",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/explain/req-with-trace", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp["trace"])
	assert.NotEmpty(t, resp["narrative"])

	analysis := resp["analysis"].(map[string]any)
	spanCount := int(analysis["span_count"].(float64))
	assert.Greater(t, spanCount, 0)
}

func TestTopologyConfig_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/topology/config", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestSLO_MethodNotAllowed(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	api := admin.New(cfg, reg, rl, rs)

	sloEngine := gateway.NewSLOEngine(nil)
	api.SetSLOEngine(sloEngine)

	req := httptest.NewRequest(http.MethodDelete, "/api/slo", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
