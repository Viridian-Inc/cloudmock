package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Viridian-Inc/cloudmock/pkg/incident"
	"github.com/Viridian-Inc/cloudmock/pkg/incident/memory"
)

func sampleIncident(service string) *incident.Incident {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return &incident.Incident{
		Status:           "active",
		Severity:         "critical",
		Title:            "Latency spike in " + service,
		AffectedServices: []string{service},
		AffectedTenants:  []string{"tenant-a"},
		AlertCount:       3,
		FirstSeen:        now.Add(-10 * time.Minute),
		LastSeen:         now,
	}
}

func TestSaveAndGet(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	inc := sampleIncident("svc-auth")

	require.NoError(t, store.Save(ctx, inc))
	assert.NotEmpty(t, inc.ID, "ID should be populated after Save")

	got, err := store.Get(ctx, inc.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, inc.ID, got.ID)
	assert.Equal(t, inc.Status, got.Status)
	assert.Equal(t, inc.Severity, got.Severity)
	assert.Equal(t, inc.Title, got.Title)
	assert.Equal(t, inc.AffectedServices, got.AffectedServices)
	assert.Equal(t, inc.AffectedTenants, got.AffectedTenants)
	assert.Equal(t, inc.AlertCount, got.AlertCount)
	assert.WithinDuration(t, inc.FirstSeen, got.FirstSeen, time.Millisecond)
	assert.WithinDuration(t, inc.LastSeen, got.LastSeen, time.Millisecond)
	assert.Nil(t, got.ResolvedAt)
}

func TestGet_NotFound(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	_, err := store.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, incident.ErrNotFound)
}

func TestList_Filters(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	inc1 := sampleIncident("svc-order")
	inc1.Severity = "critical"
	inc1.Status = "active"

	inc2 := sampleIncident("svc-payment")
	inc2.Severity = "warning"
	inc2.Status = "resolved"

	inc3 := sampleIncident("svc-order")
	inc3.Severity = "warning"
	inc3.Status = "active"

	require.NoError(t, store.Save(ctx, inc1))
	require.NoError(t, store.Save(ctx, inc2))
	require.NoError(t, store.Save(ctx, inc3))

	t.Run("filter by status", func(t *testing.T) {
		got, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, g := range got {
			assert.Equal(t, "active", g.Status)
		}
	})

	t.Run("filter by severity", func(t *testing.T) {
		got, err := store.List(ctx, incident.IncidentFilter{Severity: "critical"})
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "critical", got[0].Severity)
	})

	t.Run("filter by service", func(t *testing.T) {
		got, err := store.List(ctx, incident.IncidentFilter{Service: "svc-order"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("limit", func(t *testing.T) {
		got, err := store.List(ctx, incident.IncidentFilter{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("no filter returns all", func(t *testing.T) {
		got, err := store.List(ctx, incident.IncidentFilter{})
		require.NoError(t, err)
		assert.Len(t, got, 3)
	})
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	inc := sampleIncident("svc-cache")
	require.NoError(t, store.Save(ctx, inc))

	// Modify and update.
	inc.AlertCount = 10
	inc.Status = "acknowledged"
	inc.Owner = "oncall-eng"
	require.NoError(t, store.Update(ctx, inc))

	got, err := store.Get(ctx, inc.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, got.AlertCount)
	assert.Equal(t, "acknowledged", got.Status)
	assert.Equal(t, "oncall-eng", got.Owner)
}

func TestUpdate_NotFound(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	inc := &incident.Incident{ID: "nonexistent"}
	err := store.Update(ctx, inc)
	assert.ErrorIs(t, err, incident.ErrNotFound)
}

func TestFindActiveByKey_Match(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	now := time.Now().UTC().Truncate(time.Millisecond)

	inc := &incident.Incident{
		Status:           "active",
		Severity:         "critical",
		Title:            "Latency spike",
		AffectedServices: []string{"svc-api", "svc-db"},
		AlertCount:       2,
		RelatedDeployID:  "deploy-123",
		FirstSeen:        now.Add(-5 * time.Minute),
		LastSeen:         now,
	}
	require.NoError(t, store.Save(ctx, inc))

	// Match by service and deploy ID.
	got, err := store.FindActiveByKey(ctx, "svc-api", "deploy-123", now.Add(-10*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, inc.ID, got.ID)

	// Match by service only (empty deploy ID).
	got2, err := store.FindActiveByKey(ctx, "svc-db", "", now.Add(-10*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, inc.ID, got2.ID)
}

func TestFindActiveByKey_ReturnsNewest(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	now := time.Now().UTC().Truncate(time.Millisecond)

	old := &incident.Incident{
		Status:           "active",
		Severity:         "warning",
		Title:            "Old incident",
		AffectedServices: []string{"svc-api"},
		AlertCount:       1,
		FirstSeen:        now.Add(-1 * time.Hour),
		LastSeen:         now.Add(-30 * time.Minute),
	}
	newer := &incident.Incident{
		Status:           "acknowledged",
		Severity:         "critical",
		Title:            "Newer incident",
		AffectedServices: []string{"svc-api"},
		AlertCount:       5,
		FirstSeen:        now.Add(-10 * time.Minute),
		LastSeen:         now,
	}
	require.NoError(t, store.Save(ctx, old))
	require.NoError(t, store.Save(ctx, newer))

	got, err := store.FindActiveByKey(ctx, "svc-api", "", now.Add(-2*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, newer.ID, got.ID, "should return the most recent match")
}

func TestFindActiveByKey_NoMatch(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	now := time.Now().UTC().Truncate(time.Millisecond)

	// Resolved incident should not match.
	inc := &incident.Incident{
		Status:           "resolved",
		Severity:         "critical",
		Title:            "Resolved incident",
		AffectedServices: []string{"svc-api"},
		AlertCount:       1,
		FirstSeen:        now.Add(-5 * time.Minute),
		LastSeen:         now,
	}
	require.NoError(t, store.Save(ctx, inc))

	_, err := store.FindActiveByKey(ctx, "svc-api", "", now.Add(-10*time.Minute))
	assert.ErrorIs(t, err, incident.ErrNotFound)

	// Wrong service should not match.
	inc2 := &incident.Incident{
		Status:           "active",
		Severity:         "critical",
		Title:            "Active incident",
		AffectedServices: []string{"svc-other"},
		AlertCount:       1,
		FirstSeen:        now.Add(-5 * time.Minute),
		LastSeen:         now,
	}
	require.NoError(t, store.Save(ctx, inc2))

	_, err = store.FindActiveByKey(ctx, "svc-api", "", now.Add(-10*time.Minute))
	assert.ErrorIs(t, err, incident.ErrNotFound)
}
