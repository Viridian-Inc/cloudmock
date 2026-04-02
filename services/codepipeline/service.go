package codepipeline

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator resolves other services for cross-service integration.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// CodePipelineService is the cloudmock implementation of the AWS CodePipeline API.
type CodePipelineService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new CodePipelineService for the given AWS account ID and region.
func New(accountID, region string) *CodePipelineService {
	return &CodePipelineService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new CodePipelineService with a ServiceLocator.
func NewWithLocator(accountID, region string, locator ServiceLocator) *CodePipelineService {
	return &CodePipelineService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service calls.
func (s *CodePipelineService) SetLocator(locator ServiceLocator) { s.locator = locator }

// Name returns the AWS service name used for routing.
func (s *CodePipelineService) Name() string { return "codepipeline" }

// Actions returns the list of CodePipeline API actions supported.
func (s *CodePipelineService) Actions() []service.Action {
	return []service.Action{
		// Pipelines
		{Name: "CreatePipeline", Method: http.MethodPost, IAMAction: "codepipeline:CreatePipeline"},
		{Name: "GetPipeline", Method: http.MethodPost, IAMAction: "codepipeline:GetPipeline"},
		{Name: "ListPipelines", Method: http.MethodPost, IAMAction: "codepipeline:ListPipelines"},
		{Name: "UpdatePipeline", Method: http.MethodPost, IAMAction: "codepipeline:UpdatePipeline"},
		{Name: "DeletePipeline", Method: http.MethodPost, IAMAction: "codepipeline:DeletePipeline"},
		// Executions
		{Name: "GetPipelineState", Method: http.MethodPost, IAMAction: "codepipeline:GetPipelineState"},
		{Name: "GetPipelineExecution", Method: http.MethodPost, IAMAction: "codepipeline:GetPipelineExecution"},
		{Name: "ListPipelineExecutions", Method: http.MethodPost, IAMAction: "codepipeline:ListPipelineExecutions"},
		{Name: "StartPipelineExecution", Method: http.MethodPost, IAMAction: "codepipeline:StartPipelineExecution"},
		{Name: "StopPipelineExecution", Method: http.MethodPost, IAMAction: "codepipeline:StopPipelineExecution"},
		// Approval & Retry
		{Name: "PutApprovalResult", Method: http.MethodPost, IAMAction: "codepipeline:PutApprovalResult"},
		{Name: "RetryStageExecution", Method: http.MethodPost, IAMAction: "codepipeline:RetryStageExecution"},
		// Webhooks
		{Name: "PutWebhook", Method: http.MethodPost, IAMAction: "codepipeline:PutWebhook"},
		{Name: "ListWebhooks", Method: http.MethodPost, IAMAction: "codepipeline:ListWebhooks"},
		{Name: "DeleteWebhook", Method: http.MethodPost, IAMAction: "codepipeline:DeleteWebhook"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "codepipeline:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "codepipeline:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "codepipeline:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *CodePipelineService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CodePipeline request to the appropriate handler.
func (s *CodePipelineService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreatePipeline":
		return handleCreatePipeline(ctx, s.store)
	case "GetPipeline":
		return handleGetPipeline(ctx, s.store)
	case "ListPipelines":
		return handleListPipelines(ctx, s.store)
	case "UpdatePipeline":
		return handleUpdatePipeline(ctx, s.store)
	case "DeletePipeline":
		return handleDeletePipeline(ctx, s.store)
	case "GetPipelineState":
		return handleGetPipelineState(ctx, s.store)
	case "GetPipelineExecution":
		return handleGetPipelineExecution(ctx, s.store)
	case "ListPipelineExecutions":
		return handleListPipelineExecutions(ctx, s.store)
	case "StartPipelineExecution":
		return handleStartPipelineExecution(ctx, s.store)
	case "StopPipelineExecution":
		return handleStopPipelineExecution(ctx, s.store)
	case "PutApprovalResult":
		return handlePutApprovalResult(ctx, s.store)
	case "RetryStageExecution":
		return handleRetryStageExecution(ctx, s.store)
	// Webhooks
	case "PutWebhook":
		return handlePutWebhook(ctx, s.store)
	case "ListWebhooks":
		return handleListWebhooks(ctx, s.store)
	case "DeleteWebhook":
		return handleDeleteWebhook(ctx, s.store)
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
