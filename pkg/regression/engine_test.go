package regression

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// --- Mock MetricSource ---

type mockSource struct {
	mu       sync.Mutex
	services []string
	tenants  map[string][]string
	metrics  map[string]*WindowMetrics // key: "service:action:start-unix"
}

func (m *mockSource) ListServices(_ context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.services, nil
}

func (m *mockSource) ListTenants(_ context.Context, service string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tenants == nil {
		return nil, nil
	}
	return m.tenants[service], nil
}

func (m *mockSource) WindowMetrics(_ context.Context, service, action string, window TimeWindow) (*WindowMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s:%d", service, action, window.Start.Unix())
	if wm, ok := m.metrics[key]; ok {
		return wm, nil
	}
	// Fallback: match by service and rough time (before vs after based on sign of offset from now)
	// Try keys without action
	key = fmt.Sprintf("%s::%d", service, window.Start.Unix())
	if wm, ok := m.metrics[key]; ok {
		return wm, nil
	}
	return &WindowMetrics{Service: service, Action: action, RequestCount: 0}, nil
}

func (m *mockSource) TenantWindowMetrics(_ context.Context, service, tenantID string, window TimeWindow) (*WindowMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("tenant:%s:%s:%d", service, tenantID, window.Start.Unix())
	if wm, ok := m.metrics[key]; ok {
		return wm, nil
	}
	return &WindowMetrics{Service: service, RequestCount: 0}, nil
}

func (m *mockSource) FleetWindowMetrics(_ context.Context, service string, window TimeWindow) (*WindowMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("fleet:%s:%d", service, window.Start.Unix())
	if wm, ok := m.metrics[key]; ok {
		return wm, nil
	}
	return &WindowMetrics{Service: service, RequestCount: 0}, nil
}

// setMetric is a helper for tests that stores metrics keyed by service + start time.
func (m *mockSource) setMetric(key string, wm *WindowMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.metrics == nil {
		m.metrics = make(map[string]*WindowMetrics)
	}
	m.metrics[key] = wm
}

// --- Mock RegressionStore ---

type mockStore struct {
	mu          sync.Mutex
	regressions []Regression
	saved       int
}

func (m *mockStore) Save(_ context.Context, r *Regression) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.ID == "" {
		r.ID = fmt.Sprintf("reg-%d", m.saved+1)
	}
	m.regressions = append(m.regressions, *r)
	m.saved++
	return nil
}

func (m *mockStore) List(_ context.Context, filter RegressionFilter) ([]Regression, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []Regression
	for _, r := range m.regressions {
		if filter.Service != "" && r.Service != filter.Service {
			continue
		}
		if filter.DeployID != "" && r.DeployID != filter.DeployID {
			continue
		}
		if filter.Algorithm != "" && r.Algorithm != filter.Algorithm {
			continue
		}
		if filter.Status != "" && r.Status != filter.Status {
			continue
		}
		results = append(results, r)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

func (m *mockStore) Get(_ context.Context, id string) (*Regression, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.regressions {
		if m.regressions[i].ID == id {
			return &m.regressions[i], nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockStore) UpdateStatus(_ context.Context, id string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.regressions {
		if m.regressions[i].ID == id {
			m.regressions[i].Status = status
			if status == "resolved" {
				now := time.Now()
				m.regressions[i].ResolvedAt = &now
			}
			return nil
		}
	}
	return ErrNotFound
}

func (m *mockStore) ActiveForDeploy(_ context.Context, deployID string) ([]Regression, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []Regression
	for _, r := range m.regressions {
		if r.DeployID == deployID && r.Status == "active" {
			results = append(results, r)
		}
	}
	return results, nil
}

func (m *mockStore) getSaved() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saved
}

func (m *mockStore) allRegressions() []Regression {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]Regression, len(m.regressions))
	copy(cp, m.regressions)
	return cp
}

// --- Tests ---

func TestEngine_Scan(t *testing.T) {
	now := time.Now()
	window := 5 * time.Minute

	beforeStart := now.Add(-2 * window)
	afterStart := now.Add(-window)

	src := &mockSource{
		services: []string{"api-gateway", "user-service"},
	}

	// api-gateway: significant P99 regression
	src.setMetric(fmt.Sprintf("api-gateway::%d", beforeStart.Unix()), &WindowMetrics{
		Service:      "api-gateway",
		P50Ms:        10,
		P95Ms:        50,
		P99Ms:        100,
		RequestCount: 1000,
	})
	src.setMetric(fmt.Sprintf("api-gateway::%d", afterStart.Unix()), &WindowMetrics{
		Service:      "api-gateway",
		P50Ms:        15,
		P95Ms:        80,
		P99Ms:        250, // 150% increase
		RequestCount: 1000,
	})

	// user-service: no regression
	src.setMetric(fmt.Sprintf("user-service::%d", beforeStart.Unix()), &WindowMetrics{
		Service:      "user-service",
		P50Ms:        5,
		P95Ms:        20,
		P99Ms:        40,
		RequestCount: 500,
	})
	src.setMetric(fmt.Sprintf("user-service::%d", afterStart.Unix()), &WindowMetrics{
		Service:      "user-service",
		P50Ms:        5,
		P95Ms:        21,
		P99Ms:        42,
		RequestCount: 500,
	})

	store := &mockStore{}
	cfg := DefaultAlgorithmConfig()

	eng := New(src, store, nil, cfg, time.Hour, window)

	err := eng.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	regs := store.allRegressions()
	if len(regs) == 0 {
		t.Fatal("expected at least one regression to be saved")
	}

	found := false
	for _, r := range regs {
		if r.Service == "api-gateway" && r.Algorithm == AlgoLatencyRegression {
			found = true
			if r.Status != "active" {
				t.Errorf("expected status 'active', got %q", r.Status)
			}
			if r.BeforeValue != 100 {
				t.Errorf("expected BeforeValue=100, got %f", r.BeforeValue)
			}
		}
	}
	if !found {
		t.Error("expected a latency regression for api-gateway")
	}
}

func TestEngine_OnDeploy(t *testing.T) {
	now := time.Now()
	window := 5 * time.Minute
	deployAt := now.Add(-1 * time.Second)

	beforeStart := deployAt.Add(-window)

	src := &mockSource{
		services: []string{"payment-service"},
	}

	// Before deploy: normal metrics
	src.setMetric(fmt.Sprintf("payment-service::%d", beforeStart.Unix()), &WindowMetrics{
		Service:      "payment-service",
		P50Ms:        10,
		P95Ms:        50,
		P99Ms:        100,
		RequestCount: 1000,
	})

	// After deploy: degraded
	src.setMetric(fmt.Sprintf("payment-service::%d", deployAt.Unix()), &WindowMetrics{
		Service:      "payment-service",
		P50Ms:        20,
		P95Ms:        120,
		P99Ms:        350, // 250% increase
		RequestCount: 800,
	})

	store := &mockStore{}
	cfg := DefaultAlgorithmConfig()

	eng := New(src, store, nil, cfg, time.Hour, window)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eng.Start(ctx)
	defer eng.Stop()

	eng.OnDeploy(dataplane.DeployEvent{
		ID:         "deploy-123",
		Service:    "payment-service",
		DeployedAt: deployAt,
	})

	// Wait for the first eval (using short eval times configured in engine)
	deadline := time.After(2 * time.Second)
	for {
		if store.getSaved() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for deploy evaluation to save a regression")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	regs := store.allRegressions()
	found := false
	for _, r := range regs {
		if r.DeployID == "deploy-123" {
			found = true
			if r.Service != "payment-service" {
				t.Errorf("expected service 'payment-service', got %q", r.Service)
			}
		}
	}
	if !found {
		t.Error("expected a regression with deploy_id 'deploy-123'")
	}
}

func TestEngine_AutoResolution(t *testing.T) {
	now := time.Now()
	window := 5 * time.Minute

	afterStart := now.Add(-window)

	src := &mockSource{
		services: []string{"api-gateway"},
	}

	// Current metrics: recovered (within 10% of before value)
	src.setMetric(fmt.Sprintf("api-gateway::%d", afterStart.Unix()), &WindowMetrics{
		Service:      "api-gateway",
		P50Ms:        10,
		P95Ms:        50,
		P99Ms:        105, // within 10% of BeforeValue=100
		RequestCount: 1000,
	})

	store := &mockStore{
		regressions: []Regression{
			{
				ID:          "reg-existing",
				Algorithm:   AlgoLatencyRegression,
				Service:     "api-gateway",
				Status:      "active",
				BeforeValue: 100,
				AfterValue:  250,
				DetectedAt:  now.Add(-10 * time.Minute),
			},
		},
	}
	cfg := DefaultAlgorithmConfig()

	eng := New(src, store, nil, cfg, time.Hour, window)

	err := eng.checkResolutions(context.Background())
	if err != nil {
		t.Fatalf("checkResolutions() error: %v", err)
	}

	regs := store.allRegressions()
	for _, r := range regs {
		if r.ID == "reg-existing" {
			if r.Status != "resolved" {
				t.Errorf("expected status 'resolved', got %q", r.Status)
			}
			if r.ResolvedAt == nil {
				t.Error("expected ResolvedAt to be set")
			}
			return
		}
	}
	t.Error("existing regression not found in store")
}

func TestEngine_TenantOutlier(t *testing.T) {
	now := time.Now()
	window := 5 * time.Minute

	afterStart := now.Add(-window)

	src := &mockSource{
		services: []string{"api-gateway"},
		tenants:  map[string][]string{"api-gateway": {"tenant-a", "tenant-b"}},
	}

	// Fleet metrics
	src.setMetric(fmt.Sprintf("fleet:api-gateway:%d", afterStart.Unix()), &WindowMetrics{
		Service:      "api-gateway",
		P99Ms:        50,
		RequestCount: 5000,
	})

	// tenant-a: outlier (P99 is 4x fleet)
	src.setMetric(fmt.Sprintf("tenant:api-gateway:tenant-a:%d", afterStart.Unix()), &WindowMetrics{
		Service:      "tenant-a",
		P99Ms:        400, // 8x fleet -> (400/50 - 1)*100 = 700% change
		RequestCount: 500,
	})

	// tenant-b: normal
	src.setMetric(fmt.Sprintf("tenant:api-gateway:tenant-b:%d", afterStart.Unix()), &WindowMetrics{
		Service:      "tenant-b",
		P99Ms:        55,
		RequestCount: 500,
	})

	store := &mockStore{}
	cfg := DefaultAlgorithmConfig()

	eng := New(src, store, nil, cfg, time.Hour, window)

	err := eng.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	regs := store.allRegressions()
	found := false
	for _, r := range regs {
		if r.Algorithm == AlgoTenantOutlier && r.TenantID == "tenant-a" {
			found = true
			if r.Service != "api-gateway" {
				t.Errorf("expected service 'api-gateway', got %q", r.Service)
			}
		}
	}
	if !found {
		t.Error("expected a tenant outlier regression for tenant-a")
	}
}

func TestEngine_StartStop(t *testing.T) {
	src := &mockSource{services: []string{}}
	store := &mockStore{}
	cfg := DefaultAlgorithmConfig()

	goroutinesBefore := runtime.NumGoroutine()

	eng := New(src, store, nil, cfg, 50*time.Millisecond, 5*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eng.Start(ctx)

	// Let it tick a couple times
	time.Sleep(150 * time.Millisecond)

	eng.Stop()

	// Allow goroutines to wind down
	time.Sleep(50 * time.Millisecond)

	goroutinesAfter := runtime.NumGoroutine()

	// Allow some slack (other goroutines in the runtime)
	if goroutinesAfter > goroutinesBefore+2 {
		t.Errorf("possible goroutine leak: before=%d, after=%d", goroutinesBefore, goroutinesAfter)
	}
}
