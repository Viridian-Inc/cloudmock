package stubs

import (
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/stub"
)

// RegisterAll registers all 74 Tier 2 stub services with the gateway registry.
func RegisterAll(registry *routing.Registry, accountID, region string) {
	reg := stub.NewStubRegistry()
	for _, model := range AllModels() {
		reg.Register(model)
		svc, _ := reg.CreateService(model.ServiceName, accountID, region)
		registry.Register(svc)
	}
}
