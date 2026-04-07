package pipes

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// PipesService is the cloudmock implementation of the AWS EventBridge Pipes API.
type PipesService struct {
	store     *Store
	accountID string
	region    string
	locator   ServiceLocator
}

// New returns a new PipesService for the given AWS account ID and region.
func New(accountID, region string) *PipesService {
	return &PipesService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// NewWithLocator returns a new PipesService with a ServiceLocator for cross-service integration.
func NewWithLocator(accountID, region string, locator ServiceLocator) *PipesService {
	s := New(accountID, region)
	s.locator = locator
	s.store.SetLocator(locator)
	return s
}

// SetLocator sets the service locator for cross-service calls.
func (s *PipesService) SetLocator(locator ServiceLocator) {
	s.locator = locator
	s.store.SetLocator(locator)
}

// Name returns the AWS service name used for routing.
func (s *PipesService) Name() string { return "pipes" }

// Actions returns the list of Pipes API actions supported by this service.
func (s *PipesService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreatePipe", Method: http.MethodPost, IAMAction: "pipes:CreatePipe"},
		{Name: "DescribePipe", Method: http.MethodGet, IAMAction: "pipes:DescribePipe"},
		{Name: "ListPipes", Method: http.MethodGet, IAMAction: "pipes:ListPipes"},
		{Name: "UpdatePipe", Method: http.MethodPut, IAMAction: "pipes:UpdatePipe"},
		{Name: "DeletePipe", Method: http.MethodDelete, IAMAction: "pipes:DeletePipe"},
		{Name: "StartPipe", Method: http.MethodPost, IAMAction: "pipes:StartPipe"},
		{Name: "StopPipe", Method: http.MethodPost, IAMAction: "pipes:StopPipe"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "pipes:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "pipes:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "pipes:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *PipesService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Pipes request to the appropriate handler.
func (s *PipesService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreatePipe":
		return handleCreatePipe(ctx, s.store)
	case "DescribePipe":
		return handleDescribePipe(ctx, s.store)
	case "ListPipes":
		return handleListPipes(ctx, s.store)
	case "UpdatePipe":
		return handleUpdatePipe(ctx, s.store)
	case "DeletePipe":
		return handleDeletePipe(ctx, s.store)
	case "StartPipe":
		return handleStartPipe(ctx, s.store)
	case "StopPipe":
		return handleStopPipe(ctx, s.store)
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
