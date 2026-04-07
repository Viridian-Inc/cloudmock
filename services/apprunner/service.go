package apprunner

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// AppRunnerService is the cloudmock implementation of the AWS App Runner API.
type AppRunnerService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new AppRunnerService for the given AWS account ID and region.
func New(accountID, region string) *AppRunnerService {
	return &AppRunnerService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *AppRunnerService) Name() string { return "apprunner" }

// Actions returns the list of App Runner API actions supported by this service.
func (s *AppRunnerService) Actions() []service.Action {
	return []service.Action{
		// Services
		{Name: "CreateService", Method: http.MethodPost, IAMAction: "apprunner:CreateService"},
		{Name: "DescribeService", Method: http.MethodPost, IAMAction: "apprunner:DescribeService"},
		{Name: "ListServices", Method: http.MethodPost, IAMAction: "apprunner:ListServices"},
		{Name: "UpdateService", Method: http.MethodPost, IAMAction: "apprunner:UpdateService"},
		{Name: "DeleteService", Method: http.MethodPost, IAMAction: "apprunner:DeleteService"},
		{Name: "PauseService", Method: http.MethodPost, IAMAction: "apprunner:PauseService"},
		{Name: "ResumeService", Method: http.MethodPost, IAMAction: "apprunner:ResumeService"},
		// Connections
		{Name: "CreateConnection", Method: http.MethodPost, IAMAction: "apprunner:CreateConnection"},
		{Name: "DescribeConnection", Method: http.MethodPost, IAMAction: "apprunner:DescribeConnection"},
		// Auto Scaling Configurations
		{Name: "CreateAutoScalingConfiguration", Method: http.MethodPost, IAMAction: "apprunner:CreateAutoScalingConfiguration"},
		{Name: "DescribeAutoScalingConfiguration", Method: http.MethodPost, IAMAction: "apprunner:DescribeAutoScalingConfiguration"},
		{Name: "ListAutoScalingConfigurations", Method: http.MethodPost, IAMAction: "apprunner:ListAutoScalingConfigurations"},
		{Name: "DeleteAutoScalingConfiguration", Method: http.MethodPost, IAMAction: "apprunner:DeleteAutoScalingConfiguration"},
		// VPC Connectors
		{Name: "CreateVpcConnector", Method: http.MethodPost, IAMAction: "apprunner:CreateVpcConnector"},
		{Name: "DescribeVpcConnector", Method: http.MethodPost, IAMAction: "apprunner:DescribeVpcConnector"},
		{Name: "ListVpcConnectors", Method: http.MethodPost, IAMAction: "apprunner:ListVpcConnectors"},
		{Name: "DeleteVpcConnector", Method: http.MethodPost, IAMAction: "apprunner:DeleteVpcConnector"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "apprunner:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "apprunner:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "apprunner:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *AppRunnerService) HealthCheck() error { return nil }

// HandleRequest routes an incoming App Runner request to the appropriate handler.
func (s *AppRunnerService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	// App Runner uses JSON protocol with X-Amz-Target header routing.
	action := ctx.Action
	if action == "" {
		// Fallback: try parsing from body.
		var body map[string]any
		if len(ctx.Body) > 0 {
			json.Unmarshal(ctx.Body, &body)
		}
	}

	switch action {
	// Services
	case "CreateService":
		return handleCreateService(ctx, s.store)
	case "DescribeService":
		return handleDescribeService(ctx, s.store)
	case "ListServices":
		return handleListServices(ctx, s.store)
	case "UpdateService":
		return handleUpdateService(ctx, s.store)
	case "DeleteService":
		return handleDeleteService(ctx, s.store)
	case "PauseService":
		return handlePauseService(ctx, s.store)
	case "ResumeService":
		return handleResumeService(ctx, s.store)
	// Connections
	case "CreateConnection":
		return handleCreateConnection(ctx, s.store)
	case "DescribeConnection":
		return handleDescribeConnection(ctx, s.store)
	// Auto Scaling Configurations
	case "CreateAutoScalingConfiguration":
		return handleCreateAutoScalingConfiguration(ctx, s.store)
	case "DescribeAutoScalingConfiguration":
		return handleDescribeAutoScalingConfiguration(ctx, s.store)
	case "ListAutoScalingConfigurations":
		return handleListAutoScalingConfigurations(ctx, s.store)
	case "DeleteAutoScalingConfiguration":
		return handleDeleteAutoScalingConfiguration(ctx, s.store)
	// VPC Connectors
	case "CreateVpcConnector":
		return handleCreateVpcConnector(ctx, s.store)
	case "DescribeVpcConnector":
		return handleDescribeVpcConnector(ctx, s.store)
	case "ListVpcConnectors":
		return handleListVpcConnectors(ctx, s.store)
	case "DeleteVpcConnector":
		return handleDeleteVpcConnector(ctx, s.store)
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
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
