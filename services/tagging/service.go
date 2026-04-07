package tagging

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// TaggingService is the cloudmock implementation of the AWS Resource Groups Tagging API.
type TaggingService struct {
	store     *Store
	locator   ServiceLocator
	accountID string
	region    string
}

// New returns a new TaggingService for the given AWS account ID and region.
func New(accountID, region string) *TaggingService {
	return &TaggingService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// SetLocator sets the service locator for cross-service tag aggregation.
func (s *TaggingService) SetLocator(locator ServiceLocator) {
	s.locator = locator
}

// Name returns the AWS service name used for routing.
func (s *TaggingService) Name() string { return "tagging" }

// Actions returns the list of Tagging API actions supported by this service.
func (s *TaggingService) Actions() []service.Action {
	return []service.Action{
		{Name: "GetResources", Method: http.MethodPost, IAMAction: "tag:GetResources"},
		{Name: "GetTagKeys", Method: http.MethodPost, IAMAction: "tag:GetTagKeys"},
		{Name: "GetTagValues", Method: http.MethodPost, IAMAction: "tag:GetTagValues"},
		{Name: "TagResources", Method: http.MethodPost, IAMAction: "tag:TagResources"},
		{Name: "UntagResources", Method: http.MethodPost, IAMAction: "tag:UntagResources"},
		{Name: "GetComplianceSummary", Method: http.MethodPost, IAMAction: "tag:GetComplianceSummary"},
	}
}

// HealthCheck always returns nil.
func (s *TaggingService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Tagging request to the appropriate handler.
func (s *TaggingService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "GetResources":
		return handleGetResources(params, s.store)
	case "GetTagKeys":
		return handleGetTagKeys(s.store)
	case "GetTagValues":
		return handleGetTagValues(params, s.store)
	case "TagResources":
		return handleTagResources(params, s.store)
	case "UntagResources":
		return handleUntagResources(params, s.store)
	case "GetComplianceSummary":
		return handleGetComplianceSummary(s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
