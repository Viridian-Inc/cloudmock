package stubs

import (
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/neureaux/cloudmock/pkg/stub"
)

// RegisterAll eagerly registers all Tier 2 stub services with the gateway registry.
func RegisterAll(registry *routing.Registry, accountID, region string) {
	reg := stub.NewStubRegistry()
	for _, model := range AllModels() {
		reg.Register(model)
		svc, _ := reg.CreateService(model.ServiceName, accountID, region)
		registry.Register(svc)
	}
}

// RegisterAllLazy registers lazy factories for all Tier 2 stub services.
// Each service is constructed only on the first Lookup call for that service name,
// reducing memory and CPU usage at startup.
func RegisterAllLazy(registry *routing.Registry, accountID, region string) {
	for _, model := range AllModels() {
		m := model // capture loop variable
		a, r := accountID, region
		registry.RegisterLazy(m.ServiceName, func() service.Service {
			// Each lazy factory gets its own stub registry so services are
			// independent and do not share state.
			reg := stub.NewStubRegistry()
			reg.Register(m)
			svc, _ := reg.CreateService(m.ServiceName, a, r)
			return svc
		})
	}
}
