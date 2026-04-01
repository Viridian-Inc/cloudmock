package sagemaker

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// SageMakerService is the cloudmock implementation of the AWS SageMaker API.
type SageMakerService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new SageMakerService for the given AWS account ID and region.
func New(accountID, region string) *SageMakerService {
	return &SageMakerService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *SageMakerService) Name() string { return "sagemaker" }

// Actions returns the list of SageMaker API actions supported by this service.
func (s *SageMakerService) Actions() []service.Action {
	return []service.Action{
		// Notebook instances
		{Name: "CreateNotebookInstance", Method: http.MethodPost, IAMAction: "sagemaker:CreateNotebookInstance"},
		{Name: "DescribeNotebookInstance", Method: http.MethodPost, IAMAction: "sagemaker:DescribeNotebookInstance"},
		{Name: "ListNotebookInstances", Method: http.MethodPost, IAMAction: "sagemaker:ListNotebookInstances"},
		{Name: "DeleteNotebookInstance", Method: http.MethodPost, IAMAction: "sagemaker:DeleteNotebookInstance"},
		{Name: "StartNotebookInstance", Method: http.MethodPost, IAMAction: "sagemaker:StartNotebookInstance"},
		{Name: "StopNotebookInstance", Method: http.MethodPost, IAMAction: "sagemaker:StopNotebookInstance"},
		{Name: "UpdateNotebookInstance", Method: http.MethodPost, IAMAction: "sagemaker:UpdateNotebookInstance"},
		// Training jobs
		{Name: "CreateTrainingJob", Method: http.MethodPost, IAMAction: "sagemaker:CreateTrainingJob"},
		{Name: "DescribeTrainingJob", Method: http.MethodPost, IAMAction: "sagemaker:DescribeTrainingJob"},
		{Name: "ListTrainingJobs", Method: http.MethodPost, IAMAction: "sagemaker:ListTrainingJobs"},
		{Name: "StopTrainingJob", Method: http.MethodPost, IAMAction: "sagemaker:StopTrainingJob"},
		// Models
		{Name: "CreateModel", Method: http.MethodPost, IAMAction: "sagemaker:CreateModel"},
		{Name: "DescribeModel", Method: http.MethodPost, IAMAction: "sagemaker:DescribeModel"},
		{Name: "ListModels", Method: http.MethodPost, IAMAction: "sagemaker:ListModels"},
		{Name: "DeleteModel", Method: http.MethodPost, IAMAction: "sagemaker:DeleteModel"},
		// Endpoint configs
		{Name: "CreateEndpointConfig", Method: http.MethodPost, IAMAction: "sagemaker:CreateEndpointConfig"},
		{Name: "DescribeEndpointConfig", Method: http.MethodPost, IAMAction: "sagemaker:DescribeEndpointConfig"},
		{Name: "ListEndpointConfigs", Method: http.MethodPost, IAMAction: "sagemaker:ListEndpointConfigs"},
		{Name: "DeleteEndpointConfig", Method: http.MethodPost, IAMAction: "sagemaker:DeleteEndpointConfig"},
		// Endpoints
		{Name: "CreateEndpoint", Method: http.MethodPost, IAMAction: "sagemaker:CreateEndpoint"},
		{Name: "DescribeEndpoint", Method: http.MethodPost, IAMAction: "sagemaker:DescribeEndpoint"},
		{Name: "ListEndpoints", Method: http.MethodPost, IAMAction: "sagemaker:ListEndpoints"},
		{Name: "DeleteEndpoint", Method: http.MethodPost, IAMAction: "sagemaker:DeleteEndpoint"},
		{Name: "UpdateEndpoint", Method: http.MethodPost, IAMAction: "sagemaker:UpdateEndpoint"},
		{Name: "InvokeEndpoint", Method: http.MethodPost, IAMAction: "sagemaker:InvokeEndpoint"},
		// Processing jobs
		{Name: "CreateProcessingJob", Method: http.MethodPost, IAMAction: "sagemaker:CreateProcessingJob"},
		{Name: "DescribeProcessingJob", Method: http.MethodPost, IAMAction: "sagemaker:DescribeProcessingJob"},
		{Name: "ListProcessingJobs", Method: http.MethodPost, IAMAction: "sagemaker:ListProcessingJobs"},
		{Name: "StopProcessingJob", Method: http.MethodPost, IAMAction: "sagemaker:StopProcessingJob"},
		// Transform jobs
		{Name: "CreateTransformJob", Method: http.MethodPost, IAMAction: "sagemaker:CreateTransformJob"},
		{Name: "DescribeTransformJob", Method: http.MethodPost, IAMAction: "sagemaker:DescribeTransformJob"},
		{Name: "ListTransformJobs", Method: http.MethodPost, IAMAction: "sagemaker:ListTransformJobs"},
		{Name: "StopTransformJob", Method: http.MethodPost, IAMAction: "sagemaker:StopTransformJob"},
		// Tags
		{Name: "AddTags", Method: http.MethodPost, IAMAction: "sagemaker:AddTags"},
		{Name: "DeleteTags", Method: http.MethodPost, IAMAction: "sagemaker:DeleteTags"},
		{Name: "ListTags", Method: http.MethodPost, IAMAction: "sagemaker:ListTags"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *SageMakerService) HealthCheck() error { return nil }

// HandleRequest routes an incoming SageMaker request to the appropriate handler.
// SageMaker uses JSON protocol with TargetPrefix "SageMaker".
func (s *SageMakerService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	// Notebook instances
	case "CreateNotebookInstance":
		return handleCreateNotebookInstance(ctx, s.store)
	case "DescribeNotebookInstance":
		return handleDescribeNotebookInstance(ctx, s.store)
	case "ListNotebookInstances":
		return handleListNotebookInstances(ctx, s.store)
	case "DeleteNotebookInstance":
		return handleDeleteNotebookInstance(ctx, s.store)
	case "StartNotebookInstance":
		return handleStartNotebookInstance(ctx, s.store)
	case "StopNotebookInstance":
		return handleStopNotebookInstance(ctx, s.store)
	case "UpdateNotebookInstance":
		return handleUpdateNotebookInstance(ctx, s.store)
	// Training jobs
	case "CreateTrainingJob":
		return handleCreateTrainingJob(ctx, s.store)
	case "DescribeTrainingJob":
		return handleDescribeTrainingJob(ctx, s.store)
	case "ListTrainingJobs":
		return handleListTrainingJobs(ctx, s.store)
	case "StopTrainingJob":
		return handleStopTrainingJob(ctx, s.store)
	// Models
	case "CreateModel":
		return handleCreateModel(ctx, s.store)
	case "DescribeModel":
		return handleDescribeModel(ctx, s.store)
	case "ListModels":
		return handleListModels(ctx, s.store)
	case "DeleteModel":
		return handleDeleteModel(ctx, s.store)
	// Endpoint configs
	case "CreateEndpointConfig":
		return handleCreateEndpointConfig(ctx, s.store)
	case "DescribeEndpointConfig":
		return handleDescribeEndpointConfig(ctx, s.store)
	case "ListEndpointConfigs":
		return handleListEndpointConfigs(ctx, s.store)
	case "DeleteEndpointConfig":
		return handleDeleteEndpointConfig(ctx, s.store)
	// Endpoints
	case "CreateEndpoint":
		return handleCreateEndpoint(ctx, s.store)
	case "DescribeEndpoint":
		return handleDescribeEndpoint(ctx, s.store)
	case "ListEndpoints":
		return handleListEndpoints(ctx, s.store)
	case "DeleteEndpoint":
		return handleDeleteEndpoint(ctx, s.store)
	case "UpdateEndpoint":
		return handleUpdateEndpoint(ctx, s.store)
	case "InvokeEndpoint":
		return handleInvokeEndpoint(ctx, s.store)
	// Processing jobs
	case "CreateProcessingJob":
		return handleCreateProcessingJob(ctx, s.store)
	case "DescribeProcessingJob":
		return handleDescribeProcessingJob(ctx, s.store)
	case "ListProcessingJobs":
		return handleListProcessingJobs(ctx, s.store)
	case "StopProcessingJob":
		return handleStopProcessingJob(ctx, s.store)
	// Transform jobs
	case "CreateTransformJob":
		return handleCreateTransformJob(ctx, s.store)
	case "DescribeTransformJob":
		return handleDescribeTransformJob(ctx, s.store)
	case "ListTransformJobs":
		return handleListTransformJobs(ctx, s.store)
	case "StopTransformJob":
		return handleStopTransformJob(ctx, s.store)
	// Tags
	case "AddTags":
		return handleAddTags(ctx, s.store)
	case "DeleteTags":
		return handleDeleteTags(ctx, s.store)
	case "ListTags":
		return handleListTags(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
