package routing

import (
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockService implements service.Service for testing.
type mockService struct {
	name string
}

func (m *mockService) Name() string { return m.name }

func (m *mockService) Actions() []service.Action { return nil }

func (m *mockService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return nil, nil
}

func (m *mockService) HealthCheck() error { return nil }

func TestRegistry_RegisterAndLookup(t *testing.T) {
	reg := NewRegistry()

	svc := &mockService{name: "s3"}
	reg.Register(svc)

	got, err := reg.Lookup("s3")
	require.NoError(t, err)
	assert.Equal(t, svc, got)
}

func TestRegistry_LookupNonexistent(t *testing.T) {
	reg := NewRegistry()

	got, err := reg.Lookup("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()

	s3 := &mockService{name: "s3"}
	dynamo := &mockService{name: "dynamodb"}
	reg.Register(s3)
	reg.Register(dynamo)

	list := reg.List()
	assert.Len(t, list, 2)

	names := make(map[string]bool)
	for _, svc := range list {
		names[svc.Name()] = true
	}
	assert.True(t, names["s3"])
	assert.True(t, names["dynamodb"])
}

func TestRegistry_ListEmpty(t *testing.T) {
	reg := NewRegistry()
	list := reg.List()
	assert.Empty(t, list)
}

func TestRegistry_RegisterOverwrite(t *testing.T) {
	reg := NewRegistry()

	first := &mockService{name: "s3"}
	second := &mockService{name: "s3"}
	reg.Register(first)
	reg.Register(second)

	got, err := reg.Lookup("s3")
	require.NoError(t, err)
	assert.Equal(t, second, got)

	// Should still be just one entry.
	assert.Len(t, reg.List(), 1)
}
