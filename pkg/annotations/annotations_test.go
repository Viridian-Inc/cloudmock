package annotations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndList(t *testing.T) {
	s := NewStore()

	now := time.Now()
	ann := Annotation{
		Title:     "Deploy v2.1",
		Body:      "Deployed new version",
		Author:    "alice",
		Timestamp: now,
		Tags:      []string{"deploy"},
		Service:   "payments",
	}

	err := s.Create(ann)
	require.NoError(t, err)

	results, err := s.List(now.Add(-time.Hour), now.Add(time.Hour), "")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Deploy v2.1", results[0].Title)
	assert.NotEmpty(t, results[0].ID)
}

func TestListFiltersByService(t *testing.T) {
	s := NewStore()

	now := time.Now()
	require.NoError(t, s.Create(Annotation{
		Title:     "Deploy payments",
		Timestamp: now,
		Service:   "payments",
	}))
	require.NoError(t, s.Create(Annotation{
		Title:     "Deploy shipping",
		Timestamp: now,
		Service:   "shipping",
	}))

	results, err := s.List(now.Add(-time.Hour), now.Add(time.Hour), "payments")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Deploy payments", results[0].Title)
}

func TestListFiltersByTimeRange(t *testing.T) {
	s := NewStore()

	t1 := time.Now().Add(-2 * time.Hour)
	t2 := time.Now()

	require.NoError(t, s.Create(Annotation{Title: "Old", Timestamp: t1}))
	require.NoError(t, s.Create(Annotation{Title: "New", Timestamp: t2}))

	results, err := s.List(t2.Add(-time.Minute), t2.Add(time.Minute), "")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "New", results[0].Title)
}

func TestDelete(t *testing.T) {
	s := NewStore()

	now := time.Now()
	require.NoError(t, s.Create(Annotation{
		ID:        "ann-1",
		Title:     "To delete",
		Timestamp: now,
	}))

	err := s.Delete("ann-1")
	require.NoError(t, err)

	results, err := s.List(now.Add(-time.Hour), now.Add(time.Hour), "")
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestDeleteNotFound(t *testing.T) {
	s := NewStore()
	err := s.Delete("nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}
