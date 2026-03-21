package stub

import "fmt"

// StubRegistry holds service models and can create StubService instances from them.
type StubRegistry struct {
	models map[string]*ServiceModel
}

// NewStubRegistry creates an empty StubRegistry.
func NewStubRegistry() *StubRegistry {
	return &StubRegistry{
		models: make(map[string]*ServiceModel),
	}
}

// Register adds a ServiceModel to the registry, keyed by its ServiceName.
func (r *StubRegistry) Register(model *ServiceModel) {
	r.models[model.ServiceName] = model
}

// CreateService creates a new StubService from a registered model.
func (r *StubRegistry) CreateService(serviceName, accountID, region string) (*StubService, error) {
	model, ok := r.models[serviceName]
	if !ok {
		return nil, fmt.Errorf("no model registered for service %q", serviceName)
	}
	return NewStubService(model, accountID, region), nil
}

// ListServices returns the names of all registered service models.
func (r *StubRegistry) ListServices() []string {
	names := make([]string, 0, len(r.models))
	for name := range r.models {
		names = append(names, name)
	}
	return names
}
