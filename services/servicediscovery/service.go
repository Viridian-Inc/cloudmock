package servicediscovery

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceDiscoveryService is the cloudmock implementation of the AWS Cloud Map API.
type ServiceDiscoveryService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ServiceDiscoveryService for the given AWS account ID and region.
func New(accountID, region string) *ServiceDiscoveryService {
	return &ServiceDiscoveryService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *ServiceDiscoveryService) Name() string { return "servicediscovery" }

// Actions returns the list of Cloud Map API actions supported by this service.
func (s *ServiceDiscoveryService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateHttpNamespace", Method: http.MethodPost, IAMAction: "servicediscovery:CreateHttpNamespace"},
		{Name: "CreatePrivateDnsNamespace", Method: http.MethodPost, IAMAction: "servicediscovery:CreatePrivateDnsNamespace"},
		{Name: "CreatePublicDnsNamespace", Method: http.MethodPost, IAMAction: "servicediscovery:CreatePublicDnsNamespace"},
		{Name: "GetNamespace", Method: http.MethodPost, IAMAction: "servicediscovery:GetNamespace"},
		{Name: "ListNamespaces", Method: http.MethodPost, IAMAction: "servicediscovery:ListNamespaces"},
		{Name: "DeleteNamespace", Method: http.MethodPost, IAMAction: "servicediscovery:DeleteNamespace"},
		{Name: "CreateService", Method: http.MethodPost, IAMAction: "servicediscovery:CreateService"},
		{Name: "GetService", Method: http.MethodPost, IAMAction: "servicediscovery:GetService"},
		{Name: "ListServices", Method: http.MethodPost, IAMAction: "servicediscovery:ListServices"},
		{Name: "UpdateService", Method: http.MethodPost, IAMAction: "servicediscovery:UpdateService"},
		{Name: "DeleteService", Method: http.MethodPost, IAMAction: "servicediscovery:DeleteService"},
		{Name: "RegisterInstance", Method: http.MethodPost, IAMAction: "servicediscovery:RegisterInstance"},
		{Name: "DeregisterInstance", Method: http.MethodPost, IAMAction: "servicediscovery:DeregisterInstance"},
		{Name: "GetInstance", Method: http.MethodPost, IAMAction: "servicediscovery:GetInstance"},
		{Name: "ListInstances", Method: http.MethodPost, IAMAction: "servicediscovery:ListInstances"},
		{Name: "DiscoverInstances", Method: http.MethodPost, IAMAction: "servicediscovery:DiscoverInstances"},
		{Name: "UpdateInstanceCustomHealthStatus", Method: http.MethodPost, IAMAction: "servicediscovery:UpdateInstanceCustomHealthStatus"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "servicediscovery:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "servicediscovery:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "servicediscovery:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *ServiceDiscoveryService) HealthCheck() error { return nil }

// HandleRequest routes an incoming request to the appropriate handler.
func (s *ServiceDiscoveryService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateHttpNamespace":
		return handleCreateHttpNamespace(ctx, s.store)
	case "CreatePrivateDnsNamespace":
		return handleCreatePrivateDnsNamespace(ctx, s.store)
	case "CreatePublicDnsNamespace":
		return handleCreatePublicDnsNamespace(ctx, s.store)
	case "GetNamespace":
		return handleGetNamespace(ctx, s.store)
	case "ListNamespaces":
		return handleListNamespaces(ctx, s.store)
	case "DeleteNamespace":
		return handleDeleteNamespace(ctx, s.store)
	case "CreateService":
		return handleCreateService(ctx, s.store)
	case "GetService":
		return handleGetService(ctx, s.store)
	case "ListServices":
		return handleListServices(ctx, s.store)
	case "UpdateService":
		return handleUpdateService(ctx, s.store)
	case "DeleteService":
		return handleDeleteService(ctx, s.store)
	case "RegisterInstance":
		return handleRegisterInstance(ctx, s.store)
	case "DeregisterInstance":
		return handleDeregisterInstance(ctx, s.store)
	case "GetInstance":
		return handleGetInstance(ctx, s.store)
	case "ListInstances":
		return handleListInstances(ctx, s.store)
	case "DiscoverInstances":
		return handleDiscoverInstances(ctx, s.store)
	case "UpdateInstanceCustomHealthStatus":
		return handleUpdateInstanceCustomHealthStatus(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
