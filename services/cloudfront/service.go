package cloudfront

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/schema"
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
		// Cache Policies
		{Name: "CreateCachePolicy", Method: http.MethodPost, IAMAction: "cloudfront:CreateCachePolicy"},
		{Name: "GetCachePolicy", Method: http.MethodGet, IAMAction: "cloudfront:GetCachePolicy"},
		{Name: "ListCachePolicies", Method: http.MethodGet, IAMAction: "cloudfront:ListCachePolicies"},
		{Name: "UpdateCachePolicy", Method: http.MethodPut, IAMAction: "cloudfront:UpdateCachePolicy"},
		{Name: "DeleteCachePolicy", Method: http.MethodDelete, IAMAction: "cloudfront:DeleteCachePolicy"},
		// Origin Request Policies
		{Name: "CreateOriginRequestPolicy", Method: http.MethodPost, IAMAction: "cloudfront:CreateOriginRequestPolicy"},
		{Name: "GetOriginRequestPolicy", Method: http.MethodGet, IAMAction: "cloudfront:GetOriginRequestPolicy"},
		{Name: "ListOriginRequestPolicies", Method: http.MethodGet, IAMAction: "cloudfront:ListOriginRequestPolicies"},
		{Name: "DeleteOriginRequestPolicy", Method: http.MethodDelete, IAMAction: "cloudfront:DeleteOriginRequestPolicy"},
		// Functions
		{Name: "CreateFunction", Method: http.MethodPost, IAMAction: "cloudfront:CreateFunction"},
		{Name: "GetFunction", Method: http.MethodGet, IAMAction: "cloudfront:GetFunction"},
		{Name: "ListFunctions", Method: http.MethodGet, IAMAction: "cloudfront:ListFunctions"},
		{Name: "UpdateFunction", Method: http.MethodPut, IAMAction: "cloudfront:UpdateFunction"},
		{Name: "DeleteFunction", Method: http.MethodDelete, IAMAction: "cloudfront:DeleteFunction"},
		{Name: "PublishFunction", Method: http.MethodPost, IAMAction: "cloudfront:PublishFunction"},
		{Name: "TestFunction", Method: http.MethodPost, IAMAction: "cloudfront:TestFunction"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "cloudfront:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "cloudfront:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "cloudfront:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *CloudFrontService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for CloudFront resource types.
func (s *CloudFrontService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "cloudfront",
			ResourceType:  "aws_cloudfront_distribution",
			TerraformType: "cloudmock_cloudfront_distribution",
			AWSType:       "AWS::CloudFront::Distribution",
			CreateAction:  "CreateDistribution",
			ReadAction:    "GetDistribution",
			UpdateAction:  "UpdateDistribution",
			DeleteAction:  "DeleteDistribution",
			ListAction:    "ListDistributions",
			ImportID:      "id",
			Attributes: []schema.AttributeSchema{
				{Name: "enabled", Type: "bool", Required: true},
				{Name: "origins", Type: "list", Required: true},
				{Name: "default_cache_behavior", Type: "map", Required: true},
				{Name: "id", Type: "string", Computed: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "domain_name", Type: "string", Computed: true},
				{Name: "hosted_zone_id", Type: "string", Computed: true},
				{Name: "status", Type: "string", Computed: true},
				{Name: "price_class", Type: "string", Default: "PriceClass_All"},
				{Name: "aliases", Type: "list"},
				{Name: "comment", Type: "string"},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// HandleRequest routes an incoming CloudFront request to the appropriate handler.
// CloudFront uses REST-XML path-based routing.
func (s *CloudFrontService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return HandleRESTRequest(ctx, s.store, s.locator)
}
