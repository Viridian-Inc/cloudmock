package codebuild

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator resolves other services for cross-service integration.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// CodeBuildService is the cloudmock implementation of the AWS CodeBuild API.
type CodeBuildService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new CodeBuildService for the given AWS account ID and region.
func New(accountID, region string) *CodeBuildService {
	return &CodeBuildService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new CodeBuildService with a ServiceLocator for cross-service integration.
func NewWithLocator(accountID, region string, locator ServiceLocator) *CodeBuildService {
	return &CodeBuildService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service calls.
func (s *CodeBuildService) SetLocator(locator ServiceLocator) { s.locator = locator }

// Name returns the AWS service name used for routing.
func (s *CodeBuildService) Name() string { return "codebuild" }

// Actions returns the list of CodeBuild API actions supported.
func (s *CodeBuildService) Actions() []service.Action {
	return []service.Action{
		// Projects
		{Name: "CreateProject", Method: http.MethodPost, IAMAction: "codebuild:CreateProject"},
		{Name: "BatchGetProjects", Method: http.MethodPost, IAMAction: "codebuild:BatchGetProjects"},
		{Name: "ListProjects", Method: http.MethodPost, IAMAction: "codebuild:ListProjects"},
		{Name: "UpdateProject", Method: http.MethodPost, IAMAction: "codebuild:UpdateProject"},
		{Name: "DeleteProject", Method: http.MethodPost, IAMAction: "codebuild:DeleteProject"},
		// Builds
		{Name: "StartBuild", Method: http.MethodPost, IAMAction: "codebuild:StartBuild"},
		{Name: "BatchGetBuilds", Method: http.MethodPost, IAMAction: "codebuild:BatchGetBuilds"},
		{Name: "ListBuilds", Method: http.MethodPost, IAMAction: "codebuild:ListBuilds"},
		{Name: "ListBuildsForProject", Method: http.MethodPost, IAMAction: "codebuild:ListBuildsForProject"},
		{Name: "StopBuild", Method: http.MethodPost, IAMAction: "codebuild:StopBuild"},
		// Report Groups
		{Name: "CreateReportGroup", Method: http.MethodPost, IAMAction: "codebuild:CreateReportGroup"},
		{Name: "BatchGetReportGroups", Method: http.MethodPost, IAMAction: "codebuild:BatchGetReportGroups"},
		{Name: "ListReportGroups", Method: http.MethodPost, IAMAction: "codebuild:ListReportGroups"},
		{Name: "DeleteReportGroup", Method: http.MethodPost, IAMAction: "codebuild:DeleteReportGroup"},
	}
}

// HealthCheck always returns nil.
func (s *CodeBuildService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CodeBuild request to the appropriate handler.
func (s *CodeBuildService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	// Projects
	case "CreateProject":
		return handleCreateProject(ctx, s.store)
	case "BatchGetProjects":
		return handleBatchGetProjects(ctx, s.store)
	case "ListProjects":
		return handleListProjects(ctx, s.store)
	case "UpdateProject":
		return handleUpdateProject(ctx, s.store)
	case "DeleteProject":
		return handleDeleteProject(ctx, s.store)
	// Builds
	case "StartBuild":
		return handleStartBuild(ctx, s.store)
	case "BatchGetBuilds":
		return handleBatchGetBuilds(ctx, s.store)
	case "ListBuilds":
		return handleListBuilds(ctx, s.store)
	case "ListBuildsForProject":
		return handleListBuildsForProject(ctx, s.store)
	case "StopBuild":
		return handleStopBuild(ctx, s.store)
	// Report Groups
	case "CreateReportGroup":
		return handleCreateReportGroup(ctx, s.store)
	case "BatchGetReportGroups":
		return handleBatchGetReportGroups(ctx, s.store)
	case "ListReportGroups":
		return handleListReportGroups(ctx, s.store)
	case "DeleteReportGroup":
		return handleDeleteReportGroup(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
