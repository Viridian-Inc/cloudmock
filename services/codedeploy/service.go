package codedeploy

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator resolves other services for cross-service integration.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// CodeDeployService is the cloudmock implementation of the AWS CodeDeploy API.
type CodeDeployService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new CodeDeployService for the given AWS account ID and region.
func New(accountID, region string) *CodeDeployService {
	return &CodeDeployService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new CodeDeployService with a ServiceLocator.
func NewWithLocator(accountID, region string, locator ServiceLocator) *CodeDeployService {
	return &CodeDeployService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service calls.
func (s *CodeDeployService) SetLocator(locator ServiceLocator) { s.locator = locator }

// Name returns the AWS service name used for routing.
func (s *CodeDeployService) Name() string { return "codedeploy" }

// Actions returns the list of CodeDeploy API actions supported.
func (s *CodeDeployService) Actions() []service.Action {
	return []service.Action{
		// Applications
		{Name: "CreateApplication", Method: http.MethodPost, IAMAction: "codedeploy:CreateApplication"},
		{Name: "GetApplication", Method: http.MethodPost, IAMAction: "codedeploy:GetApplication"},
		{Name: "ListApplications", Method: http.MethodPost, IAMAction: "codedeploy:ListApplications"},
		{Name: "UpdateApplication", Method: http.MethodPost, IAMAction: "codedeploy:UpdateApplication"},
		{Name: "DeleteApplication", Method: http.MethodPost, IAMAction: "codedeploy:DeleteApplication"},
		// Deployment Configs
		{Name: "CreateDeploymentConfig", Method: http.MethodPost, IAMAction: "codedeploy:CreateDeploymentConfig"},
		{Name: "GetDeploymentConfig", Method: http.MethodPost, IAMAction: "codedeploy:GetDeploymentConfig"},
		{Name: "ListDeploymentConfigs", Method: http.MethodPost, IAMAction: "codedeploy:ListDeploymentConfigs"},
		{Name: "DeleteDeploymentConfig", Method: http.MethodPost, IAMAction: "codedeploy:DeleteDeploymentConfig"},
		// Deployment Groups
		{Name: "CreateDeploymentGroup", Method: http.MethodPost, IAMAction: "codedeploy:CreateDeploymentGroup"},
		{Name: "GetDeploymentGroup", Method: http.MethodPost, IAMAction: "codedeploy:GetDeploymentGroup"},
		{Name: "ListDeploymentGroups", Method: http.MethodPost, IAMAction: "codedeploy:ListDeploymentGroups"},
		{Name: "DeleteDeploymentGroup", Method: http.MethodPost, IAMAction: "codedeploy:DeleteDeploymentGroup"},
		{Name: "UpdateDeploymentGroup", Method: http.MethodPost, IAMAction: "codedeploy:UpdateDeploymentGroup"},
		// Deployments
		{Name: "CreateDeployment", Method: http.MethodPost, IAMAction: "codedeploy:CreateDeployment"},
		{Name: "GetDeployment", Method: http.MethodPost, IAMAction: "codedeploy:GetDeployment"},
		{Name: "ListDeployments", Method: http.MethodPost, IAMAction: "codedeploy:ListDeployments"},
		{Name: "StopDeployment", Method: http.MethodPost, IAMAction: "codedeploy:StopDeployment"},
		{Name: "BatchGetDeployments", Method: http.MethodPost, IAMAction: "codedeploy:BatchGetDeployments"},
		{Name: "BatchGetDeploymentTargets", Method: http.MethodPost, IAMAction: "codedeploy:BatchGetDeploymentTargets"},
		// On-Premises Instance Tags
		{Name: "AddTagsToOnPremisesInstances", Method: http.MethodPost, IAMAction: "codedeploy:AddTagsToOnPremisesInstances"},
		{Name: "RemoveTagsFromOnPremisesInstances", Method: http.MethodPost, IAMAction: "codedeploy:RemoveTagsFromOnPremisesInstances"},
	}
}

// HealthCheck always returns nil.
func (s *CodeDeployService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CodeDeploy request to the appropriate handler.
func (s *CodeDeployService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	// Applications
	case "CreateApplication":
		return handleCreateApplication(ctx, s.store)
	case "GetApplication":
		return handleGetApplication(ctx, s.store)
	case "ListApplications":
		return handleListApplications(ctx, s.store)
	case "UpdateApplication":
		return handleUpdateApplication(ctx, s.store)
	case "DeleteApplication":
		return handleDeleteApplication(ctx, s.store)
	// Deployment Configs
	case "CreateDeploymentConfig":
		return handleCreateDeploymentConfig(ctx, s.store)
	case "GetDeploymentConfig":
		return handleGetDeploymentConfig(ctx, s.store)
	case "ListDeploymentConfigs":
		return handleListDeploymentConfigs(ctx, s.store)
	case "DeleteDeploymentConfig":
		return handleDeleteDeploymentConfig(ctx, s.store)
	// Deployment Groups
	case "CreateDeploymentGroup":
		return handleCreateDeploymentGroup(ctx, s.store)
	case "GetDeploymentGroup":
		return handleGetDeploymentGroup(ctx, s.store)
	case "ListDeploymentGroups":
		return handleListDeploymentGroups(ctx, s.store)
	case "DeleteDeploymentGroup":
		return handleDeleteDeploymentGroup(ctx, s.store)
	case "UpdateDeploymentGroup":
		return handleUpdateDeploymentGroup(ctx, s.store)
	// Deployments
	case "CreateDeployment":
		return handleCreateDeployment(ctx, s.store)
	case "GetDeployment":
		return handleGetDeployment(ctx, s.store)
	case "ListDeployments":
		return handleListDeployments(ctx, s.store)
	case "StopDeployment":
		return handleStopDeployment(ctx, s.store)
	case "BatchGetDeployments":
		return handleBatchGetDeployments(ctx, s.store)
	case "BatchGetDeploymentTargets":
		return handleBatchGetDeploymentTargets(ctx, s.store)
	// Tags
	case "AddTagsToOnPremisesInstances":
		return handleAddTagsToOnPremisesInstances(ctx, s.store)
	case "RemoveTagsFromOnPremisesInstances":
		return handleRemoveTagsFromOnPremisesInstances(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
