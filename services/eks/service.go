package eks

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
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

// HandleRequest routes an incoming EKS request to the appropriate handler.
// EKS uses REST-JSON path-based routing.
func (s *EKSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return HandleRESTRequest(ctx, s.store, s.locator)
}
