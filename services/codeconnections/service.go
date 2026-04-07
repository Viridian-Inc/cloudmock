package codeconnections

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceLocator resolves other services for cross-service integration.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// CodeConnectionsService is the cloudmock implementation of the AWS CodeConnections API.
type CodeConnectionsService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new CodeConnectionsService for the given AWS account ID and region.
func New(accountID, region string) *CodeConnectionsService {
	return &CodeConnectionsService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new CodeConnectionsService with a ServiceLocator.
func NewWithLocator(accountID, region string, locator ServiceLocator) *CodeConnectionsService {
	return &CodeConnectionsService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service calls.
func (s *CodeConnectionsService) SetLocator(locator ServiceLocator) { s.locator = locator }

// Name returns the AWS service name used for routing.
func (s *CodeConnectionsService) Name() string { return "codeconnections" }

// Actions returns the list of CodeConnections API actions supported.
func (s *CodeConnectionsService) Actions() []service.Action {
	return []service.Action{
		// Connections
		{Name: "CreateConnection", Method: http.MethodPost, IAMAction: "codeconnections:CreateConnection"},
		{Name: "GetConnection", Method: http.MethodPost, IAMAction: "codeconnections:GetConnection"},
		{Name: "ListConnections", Method: http.MethodPost, IAMAction: "codeconnections:ListConnections"},
		{Name: "DeleteConnection", Method: http.MethodPost, IAMAction: "codeconnections:DeleteConnection"},
		{Name: "UpdateConnectionStatus", Method: http.MethodPost, IAMAction: "codeconnections:UpdateConnectionStatus"},
		// Hosts
		{Name: "CreateHost", Method: http.MethodPost, IAMAction: "codeconnections:CreateHost"},
		{Name: "GetHost", Method: http.MethodPost, IAMAction: "codeconnections:GetHost"},
		{Name: "ListHosts", Method: http.MethodPost, IAMAction: "codeconnections:ListHosts"},
		{Name: "DeleteHost", Method: http.MethodPost, IAMAction: "codeconnections:DeleteHost"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "codeconnections:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "codeconnections:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "codeconnections:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *CodeConnectionsService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CodeConnections request to the appropriate handler.
func (s *CodeConnectionsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	// Connections
	case "CreateConnection":
		return handleCreateConnection(ctx, s.store)
	case "GetConnection":
		return handleGetConnection(ctx, s.store)
	case "ListConnections":
		return handleListConnections(ctx, s.store)
	case "DeleteConnection":
		return handleDeleteConnection(ctx, s.store)
	case "UpdateConnectionStatus":
		return handleUpdateConnectionStatus(ctx, s.store)
	// Hosts
	case "CreateHost":
		return handleCreateHost(ctx, s.store)
	case "GetHost":
		return handleGetHost(ctx, s.store)
	case "ListHosts":
		return handleListHosts(ctx, s.store)
	case "DeleteHost":
		return handleDeleteHost(ctx, s.store)
	// Tags
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
