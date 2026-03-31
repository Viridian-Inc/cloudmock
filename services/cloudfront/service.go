package cloudfront

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// CloudFrontService is the cloudmock implementation of the AWS CloudFront API.
type CloudFrontService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new CloudFrontService for the given AWS account ID and region.
func New(accountID, region string) *CloudFrontService {
	return &CloudFrontService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new CloudFrontService with a service locator for cross-service communication.
func NewWithLocator(accountID, region string, locator ServiceLocator) *CloudFrontService {
	return &CloudFrontService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service communication (S3/ELB origin validation).
func (s *CloudFrontService) SetLocator(locator ServiceLocator) {
	s.locator = locator
}

// Name returns the AWS service name used for routing.
func (s *CloudFrontService) Name() string { return "cloudfront" }

// Actions returns the list of CloudFront API actions supported by this service.
// CloudFront uses REST-XML path-based routing, so these are descriptive only.
func (s *CloudFrontService) Actions() []service.Action {
	return []service.Action{
		// Distributions
		{Name: "CreateDistribution", Method: http.MethodPost, IAMAction: "cloudfront:CreateDistribution"},
		{Name: "GetDistribution", Method: http.MethodGet, IAMAction: "cloudfront:GetDistribution"},
		{Name: "ListDistributions", Method: http.MethodGet, IAMAction: "cloudfront:ListDistributions"},
		{Name: "UpdateDistribution", Method: http.MethodPut, IAMAction: "cloudfront:UpdateDistribution"},
		{Name: "DeleteDistribution", Method: http.MethodDelete, IAMAction: "cloudfront:DeleteDistribution"},
		// Invalidations
		{Name: "CreateInvalidation", Method: http.MethodPost, IAMAction: "cloudfront:CreateInvalidation"},
		{Name: "GetInvalidation", Method: http.MethodGet, IAMAction: "cloudfront:GetInvalidation"},
		{Name: "ListInvalidations", Method: http.MethodGet, IAMAction: "cloudfront:ListInvalidations"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "cloudfront:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "cloudfront:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "cloudfront:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *CloudFrontService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CloudFront request to the appropriate handler.
// CloudFront uses REST-XML path-based routing.
func (s *CloudFrontService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return HandleRESTRequest(ctx, s.store, s.locator)
}
