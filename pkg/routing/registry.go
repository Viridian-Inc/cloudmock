package routing

import (
	"fmt"
	"sync"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Registry holds the set of registered AWS service mocks.
type Registry struct {
	mu       sync.RWMutex
	services map[string]service.Service
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]service.Service),
	}
}

// Register adds svc to the registry, keyed by svc.Name().
// If a service with the same name is already registered it is replaced.
func (reg *Registry) Register(svc service.Service) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.services[svc.Name()] = svc
}

// Lookup returns the service registered under name, or an error if none exists.
func (reg *Registry) Lookup(name string) (service.Service, error) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	svc, ok := reg.services[name]
	if !ok {
		return nil, fmt.Errorf("routing: no service registered for %q", name)
	}
	return svc, nil
}

// List returns all registered services in unspecified order.
func (reg *Registry) List() []service.Service {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	out := make([]service.Service, 0, len(reg.services))
	for _, svc := range reg.services {
		out = append(out, svc)
	}
	return out
}
