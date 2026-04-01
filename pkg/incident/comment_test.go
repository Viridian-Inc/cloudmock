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

func TestAddAndGetComments(t *testing.T) {
	store := memory.NewStore()
	regStore := &mockRegStore{}
	svc := incident.NewService(store, regStore, 5*time.Minute)

	ctx := context.Background()
	now := time.Now()

	// Create an incident first.
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

	incidents, err := store.List(ctx, incident.IncidentFilter{Status: "active"})
	require.NoError(t, err)
	require.Len(t, incidents, 1)
	incID := incidents[0].ID

	// Add comments.
	c1 := incident.Comment{
		Author:    "alice",
		Body:      "Looking into this now",
		Mentions:  []string{"bob"},
		CreatedAt: now,
	}
	err = store.AddComment(incID, c1)
	require.NoError(t, err)

	c2 := incident.Comment{
		Author:    "bob",
		Body:      "I see the same issue on my end",
		CreatedAt: now.Add(time.Minute),
	}
	err = store.AddComment(incID, c2)
	require.NoError(t, err)

	// Get comments.
	comments, err := store.GetComments(incID)
	require.NoError(t, err)
	require.Len(t, comments, 2)

	assert.Equal(t, "alice", comments[0].Author)
	assert.Equal(t, "Looking into this now", comments[0].Body)
	assert.Equal(t, []string{"bob"}, comments[0].Mentions)
	assert.NotEmpty(t, comments[0].ID)
	assert.Equal(t, incID, comments[0].IncidentID)

	assert.Equal(t, "bob", comments[1].Author)
}

func TestAddCommentToNonexistentIncident(t *testing.T) {
	store := memory.NewStore()

	err := store.AddComment("nonexistent", incident.Comment{
		Author: "alice",
		Body:   "test",
	})
	assert.ErrorIs(t, err, incident.ErrNotFound)
}

func TestGetCommentsEmpty(t *testing.T) {
	store := memory.NewStore()

	comments, err := store.GetComments("any-id")
	require.NoError(t, err)
	assert.Len(t, comments, 0)
}
