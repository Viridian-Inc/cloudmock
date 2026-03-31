package cloudcontrol

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// CloudControlService is the cloudmock implementation of the AWS Cloud Control API.
type CloudControlService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new CloudControlService for the given AWS account ID and region.
func New(accountID, region string) *CloudControlService {
	return &CloudControlService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *CloudControlService) Name() string { return "cloudcontrol" }

// Actions returns the list of Cloud Control API actions supported by this service.
func (s *CloudControlService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateResource", Method: http.MethodPost, IAMAction: "cloudformation:CreateResource"},
		{Name: "GetResource", Method: http.MethodPost, IAMAction: "cloudformation:GetResource"},
		{Name: "ListResources", Method: http.MethodPost, IAMAction: "cloudformation:ListResources"},
		{Name: "UpdateResource", Method: http.MethodPost, IAMAction: "cloudformation:UpdateResource"},
		{Name: "DeleteResource", Method: http.MethodPost, IAMAction: "cloudformation:DeleteResource"},
		{Name: "GetResourceRequestStatus", Method: http.MethodPost, IAMAction: "cloudformation:GetResourceRequestStatus"},
		{Name: "ListResourceRequests", Method: http.MethodPost, IAMAction: "cloudformation:ListResourceRequests"},
	}
}

// HealthCheck always returns nil.
func (s *CloudControlService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Cloud Control request to the appropriate handler.
func (s *CloudControlService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateResource":
		return handleCreateResource(params, s.store)
	case "GetResource":
		return handleGetResource(params, s.store)
	case "ListResources":
		return handleListResources(params, s.store)
	case "UpdateResource":
		return handleUpdateResource(params, s.store)
	case "DeleteResource":
		return handleDeleteResource(params, s.store)
	case "GetResourceRequestStatus":
		return handleGetResourceRequestStatus(params, s.store)
	case "ListResourceRequests":
		return handleListResourceRequests(s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
