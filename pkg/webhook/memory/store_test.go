package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Viridian-Inc/cloudmock/pkg/webhook"
	whmemory "github.com/Viridian-Inc/cloudmock/pkg/webhook/memory"
)

func sampleConfig(url string) *webhook.Config {
	return &webhook.Config{
		URL:    url,
		Type:   "generic",
		Events: []string{"incident.created", "incident.resolved"},
		Active: true,
	}
}

func TestSave_GeneratesID(t *testing.T) {
	s := whmemory.NewStore()
	cfg := sampleConfig("https://example.com/hook")

	require.NoError(t, s.Save(context.Background(), cfg))
	assert.NotEmpty(t, cfg.ID)
}

func TestGet_Found(t *testing.T) {
	s := whmemory.NewStore()
	cfg := sampleConfig("https://example.com/hook")
	require.NoError(t, s.Save(context.Background(), cfg))

	got, err := s.Get(context.Background(), cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, cfg.ID, got.ID)
	assert.Equal(t, cfg.URL, got.URL)
}

func TestGet_NotFound(t *testing.T) {
	s := whmemory.NewStore()
	_, err := s.Get(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, webhook.ErrNotFound)
}

func TestList(t *testing.T) {
	s := whmemory.NewStore()
	require.NoError(t, s.Save(context.Background(), sampleConfig("https://a.example.com")))
	require.NoError(t, s.Save(context.Background(), sampleConfig("https://b.example.com")))

	list, err := s.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestDelete(t *testing.T) {
	s := whmemory.NewStore()
	cfg := sampleConfig("https://example.com/hook")
	require.NoError(t, s.Save(context.Background(), cfg))

	require.NoError(t, s.Delete(context.Background(), cfg.ID))

	_, err := s.Get(context.Background(), cfg.ID)
	assert.ErrorIs(t, err, webhook.ErrNotFound)
}

func TestDelete_NotFound(t *testing.T) {
	s := whmemory.NewStore()
	err := s.Delete(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, webhook.ErrNotFound)
}

func TestListByEvent(t *testing.T) {
	s := whmemory.NewStore()
	ctx := context.Background()

	// Subscribed to created only.
	c1 := &webhook.Config{
		URL:    "https://a.example.com",
		Type:   "slack",
		Events: []string{"incident.created"},
		Active: true,
	}
	// Subscribed to both.
	c2 := &webhook.Config{
		URL:    "https://b.example.com",
		Type:   "generic",
		Events: []string{"incident.created", "incident.resolved"},
		Active: true,
	}
	// Inactive.
	c3 := &webhook.Config{
		URL:    "https://c.example.com",
		Type:   "generic",
		Events: []string{"incident.created"},
		Active: false,
	}

	require.NoError(t, s.Save(ctx, c1))
	require.NoError(t, s.Save(ctx, c2))
	require.NoError(t, s.Save(ctx, c3))

	created, err := s.ListByEvent(ctx, "incident.created")
	require.NoError(t, err)
	// c3 is inactive so should not be returned.
	assert.Len(t, created, 2)

	resolved, err := s.ListByEvent(ctx, "incident.resolved")
	require.NoError(t, err)
	assert.Len(t, resolved, 1)
	assert.Equal(t, "https://b.example.com", resolved[0].URL)
}

func TestSave_UpdateExisting(t *testing.T) {
	s := whmemory.NewStore()
	cfg := sampleConfig("https://example.com/hook")
	require.NoError(t, s.Save(context.Background(), cfg))

	id := cfg.ID
	cfg.URL = "https://updated.example.com/hook"
	require.NoError(t, s.Save(context.Background(), cfg))

	list, err := s.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 1, "update should not create a duplicate")
	assert.Equal(t, id, list[0].ID)
	assert.Equal(t, "https://updated.example.com/hook", list[0].URL)
}
