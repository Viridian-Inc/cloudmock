package routing

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceFactory is a function that constructs and returns a Service.
// It is called at most once per service name when using RegisterLazy.
type ServiceFactory func() service.Service

// Registry holds the set of registered AWS service mocks.
type Registry struct {
	mu                 sync.RWMutex
	services           map[string]service.Service // initialized (eager or already-used lazy)
	factories          map[string]ServiceFactory  // lazy factories not yet initialized
	versionedServices  map[ServiceKey]service.Service
	versionedFactories map[ServiceKey]ServiceFactory
	defaultVersions    map[Provider]map[string]string
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		services:           make(map[string]service.Service),
		factories:          make(map[string]ServiceFactory),
		versionedServices:  make(map[ServiceKey]service.Service),
		versionedFactories: make(map[ServiceKey]ServiceFactory),
		defaultVersions:    make(map[Provider]map[string]string),
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

// RegisterVersioned adds svc under a provider/service/API-version key.
// It allows two implementations for the same logical service to coexist when
// cloud providers expose parallel API versions.
func (reg *Registry) RegisterVersioned(key ServiceKey, svc service.Service) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	key = normalizeServiceKey(key)
	reg.versionedServices[key] = svc
	delete(reg.versionedFactories, key)
}

// RegisterLazyVersioned records a versioned service factory. The factory is
// called at most once, on the first LookupVersioned for that exact key.
func (reg *Registry) RegisterLazyVersioned(key ServiceKey, factory ServiceFactory) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	key = normalizeServiceKey(key)
	if _, exists := reg.versionedServices[key]; exists {
		return
	}
	reg.versionedFactories[key] = factory
}

// SetDefaultVersion declares the API version used when a request omits an
// explicit version for a provider/service pair.
func (reg *Registry) SetDefaultVersion(provider Provider, serviceName, apiVersion string) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	provider = normalizeProvider(provider)
	serviceName = normalizeServiceName(provider, serviceName)
	if reg.defaultVersions[provider] == nil {
		reg.defaultVersions[provider] = make(map[string]string)
	}
	reg.defaultVersions[provider][serviceName] = strings.TrimSpace(apiVersion)
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

// LookupTarget resolves a provider-aware route target. AWS targets without an
// explicit API version fall back to the legacy Lookup path for backward
// compatibility with existing CloudMock service registration.
func (reg *Registry) LookupTarget(target RouteTarget) (service.Service, error) {
	provider := normalizeProvider(target.Provider)
	if provider == ProviderAWS && strings.TrimSpace(target.APIVersion) == "" {
		return reg.Lookup(target.Service)
	}

	svc, err := reg.LookupVersioned(ServiceKey{
		Provider:   provider,
		Service:    target.Service,
		APIVersion: target.APIVersion,
	})
	if err == nil {
		return svc, nil
	}
	if provider == ProviderAzure && strings.TrimSpace(target.APIVersion) != "" && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(target.Service)), "microsoft.resources/") {
		if svc, fallbackErr := reg.LookupVersioned(ServiceKey{
			Provider:   provider,
			Service:    "Microsoft.Resources/resources",
			APIVersion: target.APIVersion,
		}); fallbackErr == nil {
			return svc, nil
		}
	}
	if provider == ProviderAWS {
		return reg.Lookup(target.Service)
	}
	return nil, err
}

// LookupVersioned returns the service registered under key. If key omits
// APIVersion and a default version is configured for that provider/service,
// the default version is used.
func (reg *Registry) LookupVersioned(key ServiceKey) (service.Service, error) {
	key = normalizeServiceKey(key)

	reg.mu.RLock()
	key = reg.applyDefaultVersionLocked(key)
	svc, ok := reg.versionedServices[key]
	reg.mu.RUnlock()
	if ok {
		return svc, nil
	}

	reg.mu.Lock()
	defer reg.mu.Unlock()

	key = reg.applyDefaultVersionLocked(key)
	if svc, ok = reg.versionedServices[key]; ok {
		return svc, nil
	}

	factory, hasFactory := reg.versionedFactories[key]
	if !hasFactory {
		return nil, fmt.Errorf("routing: no service registered for provider=%q service=%q api-version=%q", key.Provider, key.Service, key.APIVersion)
	}

	svc = factory()
	reg.versionedServices[key] = svc
	delete(reg.versionedFactories, key)
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

// All returns all services, forcing initialization of any lazy services.
// This is used by snapshot export/import which needs the real service instances.
func (reg *Registry) All() []service.Service {
	reg.mu.Lock()
	// Initialize all pending lazy factories.
	for name, factory := range reg.factories {
		svc := factory()
		reg.services[name] = svc
		delete(reg.factories, name)
	}
	out := make([]service.Service, 0, len(reg.services))
	for _, svc := range reg.services {
		out = append(out, svc)
	}
	reg.mu.Unlock()
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

func (reg *Registry) applyDefaultVersionLocked(key ServiceKey) ServiceKey {
	if key.APIVersion != "" {
		return key
	}
	if byService := reg.defaultVersions[key.Provider]; byService != nil {
		key.APIVersion = byService[key.Service]
	}
	return key
}

func normalizeServiceKey(key ServiceKey) ServiceKey {
	provider := normalizeProvider(key.Provider)
	return ServiceKey{
		Provider:   provider,
		Service:    normalizeServiceName(provider, key.Service),
		APIVersion: strings.TrimSpace(key.APIVersion),
	}
}

func normalizeProvider(provider Provider) Provider {
	if provider == "" {
		return ProviderAWS
	}
	return Provider(strings.ToLower(strings.TrimSpace(string(provider))))
}

func normalizeServiceName(provider Provider, serviceName string) string {
	name := strings.TrimSpace(serviceName)
	switch provider {
	case ProviderAzure:
		return strings.ToLower(name)
	default:
		return strings.ToLower(name)
	}
}
