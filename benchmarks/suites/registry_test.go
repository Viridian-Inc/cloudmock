package suites

import (
	"context"
	"testing"

	"github.com/neureaux/cloudmock/benchmarks/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSuite struct{}

func (f *fakeSuite) Name() string { return "fake" }
func (f *fakeSuite) Tier() int    { return 1 }
func (f *fakeSuite) Operations() []harness.Operation {
	return []harness.Operation{
		{
			Name: "FakeOp",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				return "ok", nil
			},
		},
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeSuite{})
	s, ok := r.Get("fake")
	require.True(t, ok)
	assert.Equal(t, "fake", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeSuite{})
	all := r.List()
	assert.Len(t, all, 1)
	assert.Equal(t, "fake", all[0].Name())
}

func TestRegistry_FilterByTier(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeSuite{})
	tier1 := r.FilterByTier(1)
	assert.Len(t, tier1, 1)
	tier2 := r.FilterByTier(2)
	assert.Len(t, tier2, 0)
}

func TestRegistry_GetMissing(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	assert.False(t, ok)
}
