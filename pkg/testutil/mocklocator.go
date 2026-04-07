// Package testutil provides shared test utilities for CloudMock service tests.
package testutil

import (
	"fmt"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// MockLocator is a ServiceLocator for unit tests that returns pre-configured services.
type MockLocator struct {
	services map[string]service.Service
}

// NewMockLocator creates a MockLocator from a map of service name → service.
func NewMockLocator(services map[string]service.Service) *MockLocator {
	if services == nil {
		services = make(map[string]service.Service)
	}
	return &MockLocator{services: services}
}

// Lookup returns the service registered under the given name, or an error if not found.
func (m *MockLocator) Lookup(name string) (service.Service, error) {
	svc, ok := m.services[name]
	if !ok {
		return nil, fmt.Errorf("service %q not registered in mock locator", name)
	}
	return svc, nil
}

// Register adds a service to the mock locator.
func (m *MockLocator) Register(svc service.Service) {
	m.services[svc.Name()] = svc
}
