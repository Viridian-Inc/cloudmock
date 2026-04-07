package opensearch

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// OpenSearchService is the cloudmock implementation of the AWS OpenSearch Service API.
type OpenSearchService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new OpenSearchService.
func New(accountID, region string) *OpenSearchService {
	return &OpenSearchService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *OpenSearchService) Name() string { return "opensearch" }

// Actions returns the list of API actions supported.
func (s *OpenSearchService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateDomain", Method: http.MethodPost, IAMAction: "es:CreateDomain"},
		{Name: "DescribeDomain", Method: http.MethodPost, IAMAction: "es:DescribeDomain"},
		{Name: "ListDomainNames", Method: http.MethodPost, IAMAction: "es:ListDomainNames"},
		{Name: "DeleteDomain", Method: http.MethodPost, IAMAction: "es:DeleteDomain"},
		{Name: "UpdateDomainConfig", Method: http.MethodPost, IAMAction: "es:UpdateDomainConfig"},
		{Name: "DescribeDomainConfig", Method: http.MethodPost, IAMAction: "es:DescribeDomainConfig"},
		{Name: "AddTags", Method: http.MethodPost, IAMAction: "es:AddTags"},
		{Name: "RemoveTags", Method: http.MethodPost, IAMAction: "es:RemoveTags"},
		{Name: "ListTags", Method: http.MethodPost, IAMAction: "es:ListTags"},
		{Name: "DescribeDomains", Method: http.MethodPost, IAMAction: "es:DescribeDomains"},
		{Name: "GetCompatibleVersions", Method: http.MethodPost, IAMAction: "es:GetCompatibleVersions"},
		{Name: "CreateVpcEndpoint", Method: http.MethodPost, IAMAction: "es:CreateVpcEndpoint"},
		{Name: "DescribeVpcEndpoints", Method: http.MethodPost, IAMAction: "es:DescribeVpcEndpoints"},
		{Name: "ListVpcEndpoints", Method: http.MethodPost, IAMAction: "es:ListVpcEndpoints"},
		{Name: "DeleteVpcEndpoint", Method: http.MethodPost, IAMAction: "es:DeleteVpcEndpoint"},
		{Name: "UpgradeDomain", Method: http.MethodPost, IAMAction: "es:UpgradeDomain"},
		{Name: "GetUpgradeStatus", Method: http.MethodPost, IAMAction: "es:GetUpgradeStatus"},
		{Name: "IndexDocument", Method: http.MethodPost, IAMAction: "es:ESHttpPut"},
		{Name: "Search", Method: http.MethodPost, IAMAction: "es:ESHttpGet"},
		{Name: "ClusterHealth", Method: http.MethodPost, IAMAction: "es:ESHttpGet"},
	}
}

// HealthCheck always returns nil.
func (s *OpenSearchService) HealthCheck() error { return nil }

// HandleRequest routes an incoming request to the appropriate handler.
func (s *OpenSearchService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateDomain":
		return handleCreateDomain(ctx, s.store)
	case "DescribeDomain":
		return handleDescribeDomain(ctx, s.store)
	case "ListDomainNames":
		return handleListDomainNames(ctx, s.store)
	case "DeleteDomain":
		return handleDeleteDomain(ctx, s.store)
	case "UpdateDomainConfig":
		return handleUpdateDomainConfig(ctx, s.store)
	case "DescribeDomainConfig":
		return handleDescribeDomainConfig(ctx, s.store)
	case "AddTags":
		return handleAddTags(ctx, s.store)
	case "RemoveTags":
		return handleRemoveTags(ctx, s.store)
	case "ListTags":
		return handleListTags(ctx, s.store)
	case "DescribeDomains":
		return handleDescribeDomains(ctx, s.store)
	case "GetCompatibleVersions":
		return handleGetCompatibleVersions(ctx, s.store)
	case "CreateVpcEndpoint":
		return handleCreateVpcEndpoint(ctx, s.store)
	case "DescribeVpcEndpoints":
		return handleDescribeVpcEndpoints(ctx, s.store)
	case "ListVpcEndpoints":
		return handleListVpcEndpoints(ctx, s.store)
	case "DeleteVpcEndpoint":
		return handleDeleteVpcEndpoint(ctx, s.store)
	case "UpgradeDomain":
		return handleUpgradeDomain(ctx, s.store)
	case "GetUpgradeStatus":
		return handleGetUpgradeStatus(ctx, s.store)
	case "IndexDocument":
		return handleIndexDocument(ctx, s.store)
	case "Search":
		return handleSearch(ctx, s.store)
	case "ClusterHealth":
		return handleClusterHealth(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
