package incident_test

import (
	"context"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/incident"
	"github.com/neureaux/cloudmock/pkg/incident/memory"
	"github.com/neureaux/cloudmock/pkg/regression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRegStore implements regression.RegressionStore with only ActiveForDeploy
// providing real behaviour; every other method is a no-op stub.
type mockRegStore struct {
	active []regression.Regression
}

func (m *mockRegStore) ActiveForDeploy(_ context.Context, deployID string) ([]regression.Regression, error) {
	var result []regression.Regression
	for _, r := range m.active {
		if r.DeployID == deployID && r.Status == "active" {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRegStore) Save(_ context.Context, _ *regression.Regression) error { return nil }
func (m *mockRegStore) List(_ context.Context, _ regression.RegressionFilter) ([]regression.Regression, error) {
	return nil, nil
}
func (m *mockRegStore) Get(_ context.Context, _ string) (*regression.Regression, error) {
	return nil, regression.ErrNotFound
}
func (m *mockRegStore) UpdateStatus(_ context.Context, _ string, _ string) error { return nil }

var _ regression.RegressionStore = (*mockRegStore)(nil)

func TestOnRegression_CreatesIncident(t *testing.T) {
	store := memory.NewStore()
	regStore := &mockRegStore{}
	svc := incident.NewService(store, regStore, 5*time.Minute)

	ctx := context.Background()
	now := time.Now()

	r := regression.Regression{
		ID:         "r1",
		Service:    "payments",
		DeployID:   "deploy-1",
		TenantID:   "tenant-a",
		Title:      "Latency regression on /checkout",
		Severity:   regression.SeverityWarning,
		Status:     "active",
		DetectedAt: now,
	}

	err := svc.OnRegression(ctx, r)
	require.NoError(t, err)

	incidents, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
	require.NoError(t, err)
	require.Len(t, incidents, 1)

	inc := incidents[0]
	assert.Equal(t, "active", inc.Status)
	assert.Equal(t, "warning", inc.Severity)
	assert.Equal(t, "Latency regression on /checkout", inc.Title)
	assert.Equal(t, []string{"payments"}, inc.AffectedServices)
	assert.Equal(t, []string{"tenant-a"}, inc.AffectedTenants)
	assert.Equal(t, "deploy-1", inc.RelatedDeployID)
	assert.Equal(t, 1, inc.AlertCount)
	assert.Equal(t, "Latency regression on /checkout", inc.RootCause)
}

func TestOnRegression_GroupsByDeploy(t *testing.T) {
	store := memory.NewStore()
	regStore := &mockRegStore{}
	svc := incident.NewService(store, regStore, 5*time.Minute)

	ctx := context.Background()
	now := time.Now()

	r1 := regression.Regression{
		ID:         "r1",
		Service:    "payments",
		DeployID:   "deploy-1",
		Title:      "Latency regression",
		Severity:   regression.SeverityWarning,
		Status:     "active",
		DetectedAt: now,
	}
	r2 := regression.Regression{
		ID:         "r2",
		Service:    "payments",
		DeployID:   "deploy-1",
		Title:      "Error rate spike",
		Severity:   regression.SeverityWarning,
		Status:     "active",
		DetectedAt: now.Add(1 * time.Minute),
	}

	require.NoError(t, svc.OnRegression(ctx, r1))
	require.NoError(t, svc.OnRegression(ctx, r2))

	incidents, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
	require.NoError(t, err)
	require.Len(t, incidents, 1)
	assert.Equal(t, 2, incidents[0].AlertCount)
}

func TestOnRegression_SeparatesUnrelated(t *testing.T) {
	store := memory.NewStore()
	regStore := &mockRegStore{}
	svc := incident.NewService(store, regStore, 5*time.Minute)

	ctx := context.Background()
	now := time.Now()

	r1 := regression.Regression{
		ID:         "r1",
		Service:    "payments",
		DeployID:   "",
		Title:      "Payments latency",
		Severity:   regression.SeverityWarning,
		Status:     "active",
		DetectedAt: now,
	}
	r2 := regression.Regression{
		ID:         "r2",
		Service:    "shipping",
		DeployID:   "",
		Title:      "Shipping error rate",
		Severity:   regression.SeverityWarning,
		Status:     "active",
		DetectedAt: now,
	}

	require.NoError(t, svc.OnRegression(ctx, r1))
	require.NoError(t, svc.OnRegression(ctx, r2))

	incidents, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
	require.NoError(t, err)
	assert.Len(t, incidents, 2)
}

func TestOnSLOBreach_CreatesIncident(t *testing.T) {
	store := memory.NewStore()
	regStore := &mockRegStore{}
	svc := incident.NewService(store, regStore, 5*time.Minute)

	ctx := context.Background()

	err := svc.OnSLOBreach(ctx, "api-gateway", "ListItems", 2.5, 0.95)
	require.NoError(t, err)

	incidents, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
	require.NoError(t, err)
	require.Len(t, incidents, 1)

	inc := incidents[0]
	assert.Contains(t, inc.Title, "SLO")
	assert.Contains(t, inc.Title, "api-gateway")
	assert.Contains(t, inc.Title, "95%")
	assert.Equal(t, "critical", inc.Severity)
	assert.Equal(t, []string{"api-gateway"}, inc.AffectedServices)
}

func TestAutoResolve_OnRegressionResolved(t *testing.T) {
	store := memory.NewStore()
	regStore := &mockRegStore{}
	svc := incident.NewService(store, regStore, 5*time.Minute)

	ctx := context.Background()
	now := time.Now()

	r := regression.Regression{
		ID:         "r1",
		Service:    "payments",
		DeployID:   "deploy-1",
		Title:      "Latency regression",
		Severity:   regression.SeverityWarning,
		Status:     "active",
		DetectedAt: now,
	}

	require.NoError(t, svc.OnRegression(ctx, r))

	// No active regressions remain for this deploy.
	regStore.active = nil

	r.Status = "resolved"
	require.NoError(t, svc.OnRegression(ctx, r))

	incidents, err := store.List(ctx, incident.IncidentFilter{Status: "resolved"})
	require.NoError(t, err)
	require.Len(t, incidents, 1)
	assert.NotNil(t, incidents[0].ResolvedAt)
}

func TestAutoResolve_PartialResolution(t *testing.T) {
	store := memory.NewStore()
	regStore := &mockRegStore{}
	svc := incident.NewService(store, regStore, 5*time.Minute)

	ctx := context.Background()
	now := time.Now()

	r1 := regression.Regression{
		ID:       "r1",
		Service:  "payments",
		DeployID: "deploy-1",
		Title:    "Latency regression",
		Severity: regression.SeverityWarning,
		Status:   "active",
		DetectedAt: now,
	}
	r2 := regression.Regression{
		ID:       "r2",
		Service:  "payments",
		DeployID: "deploy-1",
		Title:    "Error rate spike",
		Severity: regression.SeverityWarning,
		Status:   "active",
		DetectedAt: now.Add(1 * time.Minute),
	}

	require.NoError(t, svc.OnRegression(ctx, r1))
	require.NoError(t, svc.OnRegression(ctx, r2))

	// Resolve r1 but r2 is still active.
	regStore.active = []regression.Regression{r2}
	r1.Status = "resolved"
	require.NoError(t, svc.OnRegression(ctx, r1))

	incidents, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
	require.NoError(t, err)
	require.Len(t, incidents, 1, "incident should stay active while regressions remain")

	// Now resolve r2 as well.
	regStore.active = nil
	r2.Status = "resolved"
	require.NoError(t, svc.OnRegression(ctx, r2))

	resolved, err := store.List(ctx, incident.IncidentFilter{Status: "resolved"})
	require.NoError(t, err)
	require.Len(t, resolved, 1)
	assert.NotNil(t, resolved[0].ResolvedAt)
}

func TestGroupWindow_Expiry(t *testing.T) {
	store := memory.NewStore()
	regStore := &mockRegStore{}
	// Use a zero group window so every alert creates a new incident.
	svc := incident.NewService(store, regStore, 0)

	ctx := context.Background()
	now := time.Now()

	r1 := regression.Regression{
		ID:         "r1",
		Service:    "payments",
		DeployID:   "deploy-1",
		Title:      "Latency regression",
		Severity:   regression.SeverityWarning,
		Status:     "active",
		DetectedAt: now,
	}
	r2 := regression.Regression{
		ID:         "r2",
		Service:    "payments",
		DeployID:   "deploy-1",
		Title:      "Another regression",
		Severity:   regression.SeverityWarning,
		Status:     "active",
		DetectedAt: now.Add(1 * time.Second),
	}

	require.NoError(t, svc.OnRegression(ctx, r1))
	require.NoError(t, svc.OnRegression(ctx, r2))

	incidents, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
	require.NoError(t, err)
	assert.Len(t, incidents, 2, "expired group window should create separate incidents")
}

func TestSeverityEscalation(t *testing.T) {
	store := memory.NewStore()
	regStore := &mockRegStore{}
	svc := incident.NewService(store, regStore, 5*time.Minute)

	ctx := context.Background()
	now := time.Now()

	r1 := regression.Regression{
		ID:         "r1",
		Service:    "payments",
		DeployID:   "deploy-1",
		Title:      "Latency regression",
		Severity:   regression.SeverityWarning,
		Status:     "active",
		DetectedAt: now,
	}
	r2 := regression.Regression{
		ID:         "r2",
		Service:    "payments",
		DeployID:   "deploy-1",
		Title:      "Critical error spike",
		Severity:   regression.SeverityCritical,
		Status:     "active",
		DetectedAt: now.Add(1 * time.Minute),
	}

	require.NoError(t, svc.OnRegression(ctx, r1))
	require.NoError(t, svc.OnRegression(ctx, r2))

	incidents, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
	require.NoError(t, err)
	require.Len(t, incidents, 1)
	assert.Equal(t, "critical", incidents[0].Severity)
}
