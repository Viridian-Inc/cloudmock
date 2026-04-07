package appconfig

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// AppConfigService is the cloudmock implementation of the AWS AppConfig API.
type AppConfigService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new AppConfigService for the given AWS account ID and region.
func New(accountID, region string) *AppConfigService {
	return &AppConfigService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *AppConfigService) Name() string { return "appconfig" }

// Actions returns the list of AppConfig API actions supported by this service.
func (s *AppConfigService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateApplication", Method: http.MethodPost, IAMAction: "appconfig:CreateApplication"},
		{Name: "GetApplication", Method: http.MethodPost, IAMAction: "appconfig:GetApplication"},
		{Name: "ListApplications", Method: http.MethodPost, IAMAction: "appconfig:ListApplications"},
		{Name: "UpdateApplication", Method: http.MethodPost, IAMAction: "appconfig:UpdateApplication"},
		{Name: "DeleteApplication", Method: http.MethodPost, IAMAction: "appconfig:DeleteApplication"},
		{Name: "CreateEnvironment", Method: http.MethodPost, IAMAction: "appconfig:CreateEnvironment"},
		{Name: "GetEnvironment", Method: http.MethodPost, IAMAction: "appconfig:GetEnvironment"},
		{Name: "ListEnvironments", Method: http.MethodPost, IAMAction: "appconfig:ListEnvironments"},
		{Name: "UpdateEnvironment", Method: http.MethodPost, IAMAction: "appconfig:UpdateEnvironment"},
		{Name: "DeleteEnvironment", Method: http.MethodPost, IAMAction: "appconfig:DeleteEnvironment"},
		{Name: "CreateConfigurationProfile", Method: http.MethodPost, IAMAction: "appconfig:CreateConfigurationProfile"},
		{Name: "GetConfigurationProfile", Method: http.MethodPost, IAMAction: "appconfig:GetConfigurationProfile"},
		{Name: "ListConfigurationProfiles", Method: http.MethodPost, IAMAction: "appconfig:ListConfigurationProfiles"},
		{Name: "UpdateConfigurationProfile", Method: http.MethodPost, IAMAction: "appconfig:UpdateConfigurationProfile"},
		{Name: "DeleteConfigurationProfile", Method: http.MethodPost, IAMAction: "appconfig:DeleteConfigurationProfile"},
		{Name: "CreateDeploymentStrategy", Method: http.MethodPost, IAMAction: "appconfig:CreateDeploymentStrategy"},
		{Name: "GetDeploymentStrategy", Method: http.MethodPost, IAMAction: "appconfig:GetDeploymentStrategy"},
		{Name: "ListDeploymentStrategies", Method: http.MethodPost, IAMAction: "appconfig:ListDeploymentStrategies"},
		{Name: "UpdateDeploymentStrategy", Method: http.MethodPost, IAMAction: "appconfig:UpdateDeploymentStrategy"},
		{Name: "DeleteDeploymentStrategy", Method: http.MethodPost, IAMAction: "appconfig:DeleteDeploymentStrategy"},
		{Name: "StartDeployment", Method: http.MethodPost, IAMAction: "appconfig:StartDeployment"},
		{Name: "GetDeployment", Method: http.MethodPost, IAMAction: "appconfig:GetDeployment"},
		{Name: "ListDeployments", Method: http.MethodPost, IAMAction: "appconfig:ListDeployments"},
		{Name: "StopDeployment", Method: http.MethodPost, IAMAction: "appconfig:StopDeployment"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "appconfig:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "appconfig:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "appconfig:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *AppConfigService) HealthCheck() error { return nil }

// HandleRequest routes an incoming AppConfig request to the appropriate handler.
func (s *AppConfigService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
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
	case "CreateEnvironment":
		return handleCreateEnvironment(ctx, s.store)
	case "GetEnvironment":
		return handleGetEnvironment(ctx, s.store)
	case "ListEnvironments":
		return handleListEnvironments(ctx, s.store)
	case "UpdateEnvironment":
		return handleUpdateEnvironment(ctx, s.store)
	case "DeleteEnvironment":
		return handleDeleteEnvironment(ctx, s.store)
	case "CreateConfigurationProfile":
		return handleCreateConfigurationProfile(ctx, s.store)
	case "GetConfigurationProfile":
		return handleGetConfigurationProfile(ctx, s.store)
	case "ListConfigurationProfiles":
		return handleListConfigurationProfiles(ctx, s.store)
	case "UpdateConfigurationProfile":
		return handleUpdateConfigurationProfile(ctx, s.store)
	case "DeleteConfigurationProfile":
		return handleDeleteConfigurationProfile(ctx, s.store)
	case "CreateDeploymentStrategy":
		return handleCreateDeploymentStrategy(ctx, s.store)
	case "GetDeploymentStrategy":
		return handleGetDeploymentStrategy(ctx, s.store)
	case "ListDeploymentStrategies":
		return handleListDeploymentStrategies(ctx, s.store)
	case "UpdateDeploymentStrategy":
		return handleUpdateDeploymentStrategy(ctx, s.store)
	case "DeleteDeploymentStrategy":
		return handleDeleteDeploymentStrategy(ctx, s.store)
	case "StartDeployment":
		return handleStartDeployment(ctx, s.store)
	case "GetDeployment":
		return handleGetDeployment(ctx, s.store)
	case "ListDeployments":
		return handleListDeployments(ctx, s.store)
	case "StopDeployment":
		return handleStopDeployment(ctx, s.store)
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
