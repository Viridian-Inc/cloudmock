package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neureaux/cloudmock/pkg/audit"
	"github.com/neureaux/cloudmock/pkg/audit/memory"
)

func TestLogAndQuery(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLogger()

	entries := []audit.Entry{
		{Actor: "alice", Action: "deploy.created", Resource: "deploy:d1"},
		{Actor: "bob", Action: "view.saved", Resource: "view:v1"},
		{Actor: "alice", Action: "slo.rules.updated", Resource: "slo:config"},
	}
	for _, e := range entries {
		require.NoError(t, l.Log(ctx, e))
	}

	// Query all — should be newest first.
	all, err := l.Query(ctx, audit.Filter{})
	require.NoError(t, err)
	assert.Len(t, all, 3)
	assert.Equal(t, "slo.rules.updated", all[0].Action)
	assert.Equal(t, "view.saved", all[1].Action)
	assert.Equal(t, "deploy.created", all[2].Action)
}

func TestQueryByActor(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLogger()

	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "alice", Action: "a"}))
	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "bob", Action: "b"}))
	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "alice", Action: "c"}))

	result, err := l.Query(ctx, audit.Filter{Actor: "alice"})
	require.NoError(t, err)
	assert.Len(t, result, 2)
	for _, e := range result {
		assert.Equal(t, "alice", e.Actor)
	}
}

func TestQueryByAction(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLogger()

	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "sys", Action: "deploy.created"}))
	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "sys", Action: "view.saved"}))
	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "sys", Action: "deploy.created"}))

	result, err := l.Query(ctx, audit.Filter{Action: "deploy.created"})
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestQueryByResource(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLogger()

	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "sys", Action: "a", Resource: "deploy:d1"}))
	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "sys", Action: "b", Resource: "view:v1"}))

	result, err := l.Query(ctx, audit.Filter{Resource: "deploy:d1"})
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "deploy:d1", result[0].Resource)
}

func TestQueryLimit(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLogger()

	for i := 0; i < 10; i++ {
		require.NoError(t, l.Log(ctx, audit.Entry{Actor: "sys", Action: "x"}))
	}

	result, err := l.Query(ctx, audit.Filter{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestAutoID(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLogger()

	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "sys", Action: "x"}))

	result, err := l.Query(ctx, audit.Filter{})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.NotEmpty(t, result[0].ID)
}

func TestAutoTimestamp(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLogger()

	before := time.Now()
	require.NoError(t, l.Log(ctx, audit.Entry{Actor: "sys", Action: "x"}))

	result, err := l.Query(ctx, audit.Filter{})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.False(t, result[0].Timestamp.IsZero())
	assert.True(t, result[0].Timestamp.After(before) || result[0].Timestamp.Equal(before))
}

func TestExplicitIDPreserved(t *testing.T) {
	ctx := context.Background()
	l := memory.NewLogger()

	require.NoError(t, l.Log(ctx, audit.Entry{ID: "my-id", Actor: "sys", Action: "x"}))

	result, err := l.Query(ctx, audit.Filter{})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "my-id", result[0].ID)
}
