package route53resolver

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Route53ResolverService is the cloudmock implementation of the Amazon Route 53 Resolver API.
type Route53ResolverService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new Route53ResolverService for the given AWS account ID and region.
func New(accountID, region string) *Route53ResolverService {
	return &Route53ResolverService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *Route53ResolverService) Name() string { return "route53resolver" }

// Actions returns the list of Route 53 Resolver API actions supported by this service.
func (s *Route53ResolverService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateResolverEndpoint", Method: http.MethodPost, IAMAction: "route53resolver:CreateResolverEndpoint"},
		{Name: "GetResolverEndpoint", Method: http.MethodPost, IAMAction: "route53resolver:GetResolverEndpoint"},
		{Name: "ListResolverEndpoints", Method: http.MethodPost, IAMAction: "route53resolver:ListResolverEndpoints"},
		{Name: "DeleteResolverEndpoint", Method: http.MethodPost, IAMAction: "route53resolver:DeleteResolverEndpoint"},
		{Name: "CreateResolverRule", Method: http.MethodPost, IAMAction: "route53resolver:CreateResolverRule"},
		{Name: "GetResolverRule", Method: http.MethodPost, IAMAction: "route53resolver:GetResolverRule"},
		{Name: "ListResolverRules", Method: http.MethodPost, IAMAction: "route53resolver:ListResolverRules"},
		{Name: "DeleteResolverRule", Method: http.MethodPost, IAMAction: "route53resolver:DeleteResolverRule"},
		{Name: "AssociateResolverRule", Method: http.MethodPost, IAMAction: "route53resolver:AssociateResolverRule"},
		{Name: "GetResolverRuleAssociation", Method: http.MethodPost, IAMAction: "route53resolver:GetResolverRuleAssociation"},
		{Name: "ListResolverRuleAssociations", Method: http.MethodPost, IAMAction: "route53resolver:ListResolverRuleAssociations"},
		{Name: "DisassociateResolverRule", Method: http.MethodPost, IAMAction: "route53resolver:DisassociateResolverRule"},
		{Name: "CreateResolverQueryLogConfig", Method: http.MethodPost, IAMAction: "route53resolver:CreateResolverQueryLogConfig"},
		{Name: "GetResolverQueryLogConfig", Method: http.MethodPost, IAMAction: "route53resolver:GetResolverQueryLogConfig"},
		{Name: "ListResolverQueryLogConfigs", Method: http.MethodPost, IAMAction: "route53resolver:ListResolverQueryLogConfigs"},
		{Name: "DeleteResolverQueryLogConfig", Method: http.MethodPost, IAMAction: "route53resolver:DeleteResolverQueryLogConfig"},
	}
}

// HealthCheck always returns nil.
func (s *Route53ResolverService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Route 53 Resolver request to the appropriate handler.
func (s *Route53ResolverService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateResolverEndpoint":
		return handleCreateResolverEndpoint(params, s.store)
	case "GetResolverEndpoint":
		return handleGetResolverEndpoint(params, s.store)
	case "ListResolverEndpoints":
		return handleListResolverEndpoints(s.store)
	case "DeleteResolverEndpoint":
		return handleDeleteResolverEndpoint(params, s.store)
	case "CreateResolverRule":
		return handleCreateResolverRule(params, s.store)
	case "GetResolverRule":
		return handleGetResolverRule(params, s.store)
	case "ListResolverRules":
		return handleListResolverRules(s.store)
	case "DeleteResolverRule":
		return handleDeleteResolverRule(params, s.store)
	case "AssociateResolverRule":
		return handleAssociateResolverRule(params, s.store)
	case "GetResolverRuleAssociation":
		return handleGetResolverRuleAssociation(params, s.store)
	case "ListResolverRuleAssociations":
		return handleListResolverRuleAssociations(s.store)
	case "DisassociateResolverRule":
		return handleDisassociateResolverRule(params, s.store)
	case "CreateResolverQueryLogConfig":
		return handleCreateQueryLogConfig(params, s.store)
	case "GetResolverQueryLogConfig":
		return handleGetQueryLogConfig(params, s.store)
	case "ListResolverQueryLogConfigs":
		return handleListQueryLogConfigs(s.store)
	case "DeleteResolverQueryLogConfig":
		return handleDeleteQueryLogConfig(params, s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
