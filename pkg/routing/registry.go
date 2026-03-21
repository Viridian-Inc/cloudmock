package routing

import (
	"fmt"
	"sync"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceFactory is a function that constructs and returns a Service.
// It is called at most once per service name when using RegisterLazy.
type ServiceFactory func() service.Service

// Registry holds the set of registered AWS service mocks.
type Registry struct {
	mu        sync.RWMutex
	services  map[string]service.Service  // initialized (eager or already-used lazy)
	factories map[string]ServiceFactory   // lazy factories not yet initialized
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		services:  make(map[string]service.Service),
		factories: make(map[string]ServiceFactory),
	}
}

// Register adds svc to the registry, keyed by svc.Name().
// If a service with the same name is already registered it is replaced.
// Registering an eager service also removes any pending lazy factory for that name.
func (reg *Registry) Register(svc service.Service) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	name := svc.Name()
	reg.services[name] = svc
	delete(reg.factories, name)
}

// RegisterLazy records a factory for name. The factory is called at most once,
// the first time Lookup is called for that name. Registering a lazy factory
// overwrites any existing factory for the same name, but does not affect an
// already-initialized eager service.
func (reg *Registry) RegisterLazy(name string, factory ServiceFactory) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	// If the service was already eagerly registered, the factory is ignored.
	if _, exists := reg.services[name]; exists {
		return
	}
	reg.factories[name] = factory
}

// Lookup returns the service registered under name, or an error if none exists.
// If the service has a pending lazy factory it is initialized on first call and
// the result is cached for all subsequent calls.
func (reg *Registry) Lookup(name string) (service.Service, error) {
	// Fast path: already initialized.
	reg.mu.RLock()
	svc, ok := reg.services[name]
	reg.mu.RUnlock()
	if ok {
		return svc, nil
	}

	// Slow path: check for a lazy factory.
	reg.mu.Lock()
	defer reg.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have
	// initialized it between our RUnlock and Lock).
	if svc, ok = reg.services[name]; ok {
		return svc, nil
	}

	factory, hasFactory := reg.factories[name]
	if !hasFactory {
		return nil, fmt.Errorf("routing: no service registered for %q", name)
	}

	// Initialize, cache, and remove the factory.
	svc = factory()
	reg.services[name] = svc
	delete(reg.factories, name)
	return svc, nil
}

// List returns lightweight representations of all known services — both
// already-initialized and pending-lazy — in unspecified order.
// Lazy services are represented by a LazyService placeholder so that they are
// visible without forcing initialization.
func (reg *Registry) List() []service.Service {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	out := make([]service.Service, 0, len(reg.services)+len(reg.factories))
	for _, svc := range reg.services {
		out = append(out, svc)
	}
	for name := range reg.factories {
		out = append(out, &LazyService{name: name})
	}
	return out
}

// LazyService is a lightweight placeholder that satisfies service.Service for
// services registered via RegisterLazy that have not yet been initialized.
// It must never be used to handle real requests; Lookup will always return the
// real service after the first call.
type LazyService struct {
	name string
}

func (l *LazyService) Name() string { return l.name }

func (l *LazyService) Actions() []service.Action { return nil }

func (l *LazyService) HandleRequest(_ *service.RequestContext) (*service.Response, error) {
	return nil, fmt.Errorf("routing: lazy service %q has not been initialized", l.name)
}

func (l *LazyService) HealthCheck() error { return nil }
