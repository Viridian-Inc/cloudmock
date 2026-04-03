package ecs

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/schema"
	"github.com/neureaux/cloudmock/pkg/service"
)

// ECSService is the cloudmock implementation of the AWS Elastic Container Service API.
type ECSService struct {
	store *Store
}

// New returns a new ECSService for the given AWS account ID and region.
func New(accountID, region string) *ECSService {
	return &ECSService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
// ECS uses "ecs" in the credential scope of Authorization headers.
func (s *ECSService) Name() string { return "ecs" }

// Actions returns the list of ECS API actions supported by this service.
func (s *ECSService) Actions() []service.Action {
	return []service.Action{
		// Cluster
		{Name: "CreateCluster", Method: http.MethodPost, IAMAction: "ecs:CreateCluster"},
		{Name: "DeleteCluster", Method: http.MethodPost, IAMAction: "ecs:DeleteCluster"},
		{Name: "DescribeClusters", Method: http.MethodPost, IAMAction: "ecs:DescribeClusters"},
		{Name: "ListClusters", Method: http.MethodPost, IAMAction: "ecs:ListClusters"},
		// Task Definitions
		{Name: "RegisterTaskDefinition", Method: http.MethodPost, IAMAction: "ecs:RegisterTaskDefinition"},
		{Name: "DeregisterTaskDefinition", Method: http.MethodPost, IAMAction: "ecs:DeregisterTaskDefinition"},
		{Name: "DescribeTaskDefinition", Method: http.MethodPost, IAMAction: "ecs:DescribeTaskDefinition"},
		{Name: "ListTaskDefinitions", Method: http.MethodPost, IAMAction: "ecs:ListTaskDefinitions"},
		// Services
		{Name: "CreateService", Method: http.MethodPost, IAMAction: "ecs:CreateService"},
		{Name: "DeleteService", Method: http.MethodPost, IAMAction: "ecs:DeleteService"},
		{Name: "DescribeServices", Method: http.MethodPost, IAMAction: "ecs:DescribeServices"},
		{Name: "ListServices", Method: http.MethodPost, IAMAction: "ecs:ListServices"},
		{Name: "UpdateService", Method: http.MethodPost, IAMAction: "ecs:UpdateService"},
		// Tasks
		{Name: "RunTask", Method: http.MethodPost, IAMAction: "ecs:RunTask"},
		{Name: "StopTask", Method: http.MethodPost, IAMAction: "ecs:StopTask"},
		{Name: "DescribeTasks", Method: http.MethodPost, IAMAction: "ecs:DescribeTasks"},
		{Name: "ListTasks", Method: http.MethodPost, IAMAction: "ecs:ListTasks"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "ecs:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "ecs:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "ecs:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *ECSService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for ECS resource types.
func (s *ECSService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "ecs",
			ResourceType:  "aws_ecs_cluster",
			TerraformType: "cloudmock_ecs_cluster",
			AWSType:       "AWS::ECS::Cluster",
			CreateAction:  "CreateCluster",
			ReadAction:    "DescribeClusters",
			DeleteAction:  "DeleteCluster",
			ListAction:    "ListClusters",
			ImportID:      "name",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "ecs",
			ResourceType:  "aws_ecs_task_definition",
			TerraformType: "cloudmock_ecs_task_definition",
			AWSType:       "AWS::ECS::TaskDefinition",
			CreateAction:  "RegisterTaskDefinition",
			ReadAction:    "DescribeTaskDefinition",
			DeleteAction:  "DeregisterTaskDefinition",
			ListAction:    "ListTaskDefinitions",
			ImportID:      "arn",
			Attributes: []schema.AttributeSchema{
				{Name: "family", Type: "string", Required: true, ForceNew: true},
				{Name: "container_definitions", Type: "string", Required: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "revision", Type: "int", Computed: true},
				{Name: "cpu", Type: "string"},
				{Name: "memory", Type: "string"},
				{Name: "network_mode", Type: "string"},
				{Name: "requires_compatibilities", Type: "list"},
				{Name: "task_role_arn", Type: "string"},
				{Name: "execution_role_arn", Type: "string"},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "ecs",
			ResourceType:  "aws_ecs_service",
			TerraformType: "cloudmock_ecs_service",
			AWSType:       "AWS::ECS::Service",
			CreateAction:  "CreateService",
			ReadAction:    "DescribeServices",
			UpdateAction:  "UpdateService",
			DeleteAction:  "DeleteService",
			ListAction:    "ListServices",
			ImportID:      "cluster/name",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "cluster", Type: "string", Required: true, ForceNew: true},
				{Name: "task_definition", Type: "string", Required: true},
				{Name: "desired_count", Type: "int", Default: 1},
				{Name: "launch_type", Type: "string"},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// HandleRequest routes an incoming ECS request to the appropriate handler.
// ECS uses the JSON protocol; the action is parsed from X-Amz-Target by the gateway
// and placed in ctx.Action (e.g. "CreateCluster").
func (s *ECSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	// Clusters
	case "CreateCluster":
		return handleCreateCluster(ctx, s.store)
	case "DeleteCluster":
		return handleDeleteCluster(ctx, s.store)
	case "DescribeClusters":
		return handleDescribeClusters(ctx, s.store)
	case "ListClusters":
		return handleListClusters(ctx, s.store)
	// Task Definitions
	case "RegisterTaskDefinition":
		return handleRegisterTaskDefinition(ctx, s.store)
	case "DeregisterTaskDefinition":
		return handleDeregisterTaskDefinition(ctx, s.store)
	case "DescribeTaskDefinition":
		return handleDescribeTaskDefinition(ctx, s.store)
	case "ListTaskDefinitions":
		return handleListTaskDefinitions(ctx, s.store)
	// Services
	case "CreateService":
		return handleCreateService(ctx, s.store)
	case "DeleteService":
		return handleDeleteService(ctx, s.store)
	case "DescribeServices":
		return handleDescribeServices(ctx, s.store)
	case "ListServices":
		return handleListServices(ctx, s.store)
	case "UpdateService":
		return handleUpdateService(ctx, s.store)
	// Tasks
	case "RunTask":
		return handleRunTask(ctx, s.store)
	case "StopTask":
		return handleStopTask(ctx, s.store)
	case "DescribeTasks":
		return handleDescribeTasks(ctx, s.store)
	case "ListTasks":
		return handleListTasks(ctx, s.store)
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
