package es

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ESService is the cloudmock implementation of the AWS Elasticsearch Service API.
type ESService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ESService.
func New(accountID, region string) *ESService {
	return &ESService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *ESService) Name() string { return "es" }

// Actions returns the list of API actions supported.
func (s *ESService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateElasticsearchDomain", Method: http.MethodPost, IAMAction: "es:CreateElasticsearchDomain"},
		{Name: "DescribeElasticsearchDomain", Method: http.MethodPost, IAMAction: "es:DescribeElasticsearchDomain"},
		{Name: "ListDomainNames", Method: http.MethodPost, IAMAction: "es:ListDomainNames"},
		{Name: "DeleteElasticsearchDomain", Method: http.MethodPost, IAMAction: "es:DeleteElasticsearchDomain"},
		{Name: "UpdateElasticsearchDomainConfig", Method: http.MethodPost, IAMAction: "es:UpdateElasticsearchDomainConfig"},
		{Name: "DescribeElasticsearchDomainConfig", Method: http.MethodPost, IAMAction: "es:DescribeElasticsearchDomainConfig"},
		{Name: "AddTags", Method: http.MethodPost, IAMAction: "es:AddTags"},
		{Name: "RemoveTags", Method: http.MethodPost, IAMAction: "es:RemoveTags"},
		{Name: "ListTags", Method: http.MethodPost, IAMAction: "es:ListTags"},
	}
}

// HealthCheck always returns nil.
func (s *ESService) HealthCheck() error { return nil }

// HandleRequest routes an incoming request to the appropriate handler.
func (s *ESService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "CreateElasticsearchDomain":
		return handleCreateElasticsearchDomain(ctx, s.store)
	case "DescribeElasticsearchDomain":
		return handleDescribeElasticsearchDomain(ctx, s.store)
	case "ListDomainNames":
		return handleListDomainNames(ctx, s.store)
	case "DeleteElasticsearchDomain":
		return handleDeleteElasticsearchDomain(ctx, s.store)
	case "UpdateElasticsearchDomainConfig":
		return handleUpdateElasticsearchDomainConfig(ctx, s.store)
	case "DescribeElasticsearchDomainConfig":
		return handleDescribeElasticsearchDomainConfig(ctx, s.store)
	case "AddTags":
		return handleAddTags(ctx, s.store)
	case "RemoveTags":
		return handleRemoveTags(ctx, s.store)
	case "ListTags":
		return handleListTags(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
