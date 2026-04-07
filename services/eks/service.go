package eks

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// EKSService is the cloudmock implementation of the AWS Elastic Kubernetes Service API.
type EKSService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new EKSService for the given AWS account ID and region.
func New(accountID, region string) *EKSService {
	return &EKSService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new EKSService with a service locator for cross-service communication.
func NewWithLocator(accountID, region string, locator ServiceLocator) *EKSService {
	return &EKSService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service communication (EC2 for node groups).
func (s *EKSService) SetLocator(locator ServiceLocator) {
	s.locator = locator
	s.store.SetLocator(locator)
}

// Name returns the AWS service name used for routing.
func (s *EKSService) Name() string { return "eks" }

// Actions returns the list of EKS API actions supported by this service.
// EKS uses REST-JSON path-based routing, so these are descriptive only.
func (s *EKSService) Actions() []service.Action {
	return []service.Action{
		// Clusters
		{Name: "CreateCluster", Method: http.MethodPost, IAMAction: "eks:CreateCluster"},
		{Name: "DescribeCluster", Method: http.MethodGet, IAMAction: "eks:DescribeCluster"},
		{Name: "ListClusters", Method: http.MethodGet, IAMAction: "eks:ListClusters"},
		{Name: "DeleteCluster", Method: http.MethodDelete, IAMAction: "eks:DeleteCluster"},
		{Name: "UpdateClusterConfig", Method: http.MethodPost, IAMAction: "eks:UpdateClusterConfig"},
		// Node Groups
		{Name: "CreateNodegroup", Method: http.MethodPost, IAMAction: "eks:CreateNodegroup"},
		{Name: "DescribeNodegroup", Method: http.MethodGet, IAMAction: "eks:DescribeNodegroup"},
		{Name: "ListNodegroups", Method: http.MethodGet, IAMAction: "eks:ListNodegroups"},
		{Name: "DeleteNodegroup", Method: http.MethodDelete, IAMAction: "eks:DeleteNodegroup"},
		{Name: "UpdateNodegroupConfig", Method: http.MethodPost, IAMAction: "eks:UpdateNodegroupConfig"},
		// Fargate Profiles
		{Name: "CreateFargateProfile", Method: http.MethodPost, IAMAction: "eks:CreateFargateProfile"},
		{Name: "DescribeFargateProfile", Method: http.MethodGet, IAMAction: "eks:DescribeFargateProfile"},
		{Name: "ListFargateProfiles", Method: http.MethodGet, IAMAction: "eks:ListFargateProfiles"},
		{Name: "DeleteFargateProfile", Method: http.MethodDelete, IAMAction: "eks:DeleteFargateProfile"},
		// Addons
		{Name: "CreateAddon", Method: http.MethodPost, IAMAction: "eks:CreateAddon"},
		{Name: "DescribeAddon", Method: http.MethodGet, IAMAction: "eks:DescribeAddon"},
		{Name: "ListAddons", Method: http.MethodGet, IAMAction: "eks:ListAddons"},
		{Name: "DeleteAddon", Method: http.MethodDelete, IAMAction: "eks:DeleteAddon"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "eks:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "eks:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "eks:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *EKSService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for EKS resource types.
func (s *EKSService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "eks",
			ResourceType:  "aws_eks_cluster",
			TerraformType: "cloudmock_eks_cluster",
			AWSType:       "AWS::EKS::Cluster",
			CreateAction:  "CreateCluster",
			ReadAction:    "DescribeCluster",
			DeleteAction:  "DeleteCluster",
			ListAction:    "ListClusters",
			ImportID:      "name",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "role_arn", Type: "string", Required: true, ForceNew: true},
				{Name: "version", Type: "string"},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "endpoint", Type: "string", Computed: true},
				{Name: "status", Type: "string", Computed: true},
				{Name: "vpc_config", Type: "map", Required: true},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "eks",
			ResourceType:  "aws_eks_node_group",
			TerraformType: "cloudmock_eks_node_group",
			AWSType:       "AWS::EKS::Nodegroup",
			CreateAction:  "CreateNodegroup",
			ReadAction:    "DescribeNodegroup",
			DeleteAction:  "DeleteNodegroup",
			ListAction:    "ListNodegroups",
			ImportID:      "cluster_name:node_group_name",
			Attributes: []schema.AttributeSchema{
				{Name: "cluster_name", Type: "string", Required: true, ForceNew: true},
				{Name: "node_group_name", Type: "string", Required: true, ForceNew: true},
				{Name: "node_role_arn", Type: "string", Required: true, ForceNew: true},
				{Name: "scaling_config", Type: "map", Required: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "status", Type: "string", Computed: true},
				{Name: "instance_types", Type: "list"},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// HandleRequest routes an incoming EKS request to the appropriate handler.
// EKS uses REST-JSON path-based routing.
func (s *EKSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return HandleRESTRequest(ctx, s.store, s.locator)
}
