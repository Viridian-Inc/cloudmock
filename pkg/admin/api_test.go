package admin_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
	name      string
	actions   []service.Action
	healthy   bool
	resetCalls int
}

func (f *fakeService) Name() string                { return f.name }
func (f *fakeService) Actions() []service.Action    { return f.actions }
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

func TestRequests(t *testing.T) {
	cfg := config.Default()
	reg := routing.NewRegistry()
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()

	rl.Add(gateway.RequestEntry{Service: "s3", Action: "ListBuckets"})
	rl.Add(gateway.RequestEntry{Service: "dynamodb", Action: "PutItem"})
	rl.Add(gateway.RequestEntry{Service: "s3", Action: "GetObject"})

	api := admin.New(cfg, reg, rl, rs)

	// All requests
	req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entries []gateway.RequestEntry
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entries))
	assert.Len(t, entries, 3)

	// Filtered by service
	req = httptest.NewRequest(http.MethodGet, "/api/requests?service=s3&limit=10", nil)
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
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "%s %s", tt.method, tt.path)
	}
}

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
