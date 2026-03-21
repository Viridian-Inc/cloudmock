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

func TestRegistry_RegisterLazy_FactoryNotCalledUntilLookup(t *testing.T) {
	reg := NewRegistry()

	called := 0
	reg.RegisterLazy("kinesis", func() service.Service {
		called++
		return &mockService{name: "kinesis"}
	})

	// Factory must not have been invoked yet.
	assert.Equal(t, 0, called, "factory should not be called at registration time")

	// Trigger initialization via Lookup.
	svc, err := reg.Lookup("kinesis")
	require.NoError(t, err)
	assert.Equal(t, 1, called, "factory should be called on first Lookup")
	assert.Equal(t, "kinesis", svc.Name())
}

func TestRegistry_RegisterLazy_LookupInitializesService(t *testing.T) {
	reg := NewRegistry()

	underlying := &mockService{name: "sqs"}
	reg.RegisterLazy("sqs", func() service.Service { return underlying })

	got, err := reg.Lookup("sqs")
	require.NoError(t, err)
	assert.Equal(t, underlying, got)
}

func TestRegistry_RegisterLazy_FactoryCalledOnlyOnce(t *testing.T) {
	reg := NewRegistry()

	calls := 0
	reg.RegisterLazy("sns", func() service.Service {
		calls++
		return &mockService{name: "sns"}
	})

	first, err := reg.Lookup("sns")
	require.NoError(t, err)

	second, err := reg.Lookup("sns")
	require.NoError(t, err)

	assert.Equal(t, 1, calls, "factory should be called exactly once regardless of Lookup count")
	assert.Equal(t, first, second, "subsequent Lookups should return the same instance")
}

func TestRegistry_RegisterLazy_ListIncludesLazyServices(t *testing.T) {
	reg := NewRegistry()

	reg.Register(&mockService{name: "s3"})
	reg.RegisterLazy("kinesis", func() service.Service { return &mockService{name: "kinesis"} })
	reg.RegisterLazy("firehose", func() service.Service { return &mockService{name: "firehose"} })

	list := reg.List()
	assert.Len(t, list, 3, "List should include both eager and lazy services")

	names := make(map[string]bool)
	for _, svc := range list {
		names[svc.Name()] = true
	}
	assert.True(t, names["s3"])
	assert.True(t, names["kinesis"])
	assert.True(t, names["firehose"])
}

func TestRegistry_RegisterLazy_ListDoesNotInitializeLazy(t *testing.T) {
	reg := NewRegistry()

	called := 0
	reg.RegisterLazy("rds", func() service.Service {
		called++
		return &mockService{name: "rds"}
	})

	_ = reg.List()
	assert.Equal(t, 0, called, "List should not trigger lazy initialization")
}

func TestRegistry_Register_EagerOverridesLazy(t *testing.T) {
	reg := NewRegistry()

	reg.RegisterLazy("s3", func() service.Service { return &mockService{name: "s3"} })

	// Eagerly register a different instance under the same name.
	eager := &mockService{name: "s3"}
	reg.Register(eager)

	got, err := reg.Lookup("s3")
	require.NoError(t, err)
	assert.Equal(t, eager, got, "eager registration should take precedence over lazy factory")
}

func TestRegistry_RegisterLazy_EagerAlreadyRegisteredIsIgnored(t *testing.T) {
	reg := NewRegistry()

	// Register eagerly first.
	eager := &mockService{name: "dynamo"}
	reg.Register(eager)

	// A subsequent lazy registration for the same name should be ignored.
	factoryCalled := false
	reg.RegisterLazy("dynamo", func() service.Service {
		factoryCalled = true
		return &mockService{name: "dynamo"}
	})

	got, err := reg.Lookup("dynamo")
	require.NoError(t, err)
	assert.Equal(t, eager, got)
	assert.False(t, factoryCalled, "lazy factory should be ignored when service is already eager-registered")
}

func TestLazyService_Placeholder(t *testing.T) {
	l := &LazyService{name: "glue"}

	assert.Equal(t, "glue", l.Name())
	assert.Nil(t, l.Actions())
	assert.NoError(t, l.HealthCheck())

	_, err := l.HandleRequest(nil)
	assert.Error(t, err)
}
