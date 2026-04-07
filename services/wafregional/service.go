package wafregional

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// WAFRegionalService is the cloudmock implementation of the AWS WAF Regional (legacy) API.
type WAFRegionalService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new WAFRegionalService for the given AWS account ID and region.
func New(accountID, region string) *WAFRegionalService {
	return &WAFRegionalService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *WAFRegionalService) Name() string { return "waf-regional" }

// Actions returns the list of WAF Regional API actions supported by this service.
func (s *WAFRegionalService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateWebACL", Method: http.MethodPost, IAMAction: "waf-regional:CreateWebACL"},
		{Name: "GetWebACL", Method: http.MethodPost, IAMAction: "waf-regional:GetWebACL"},
		{Name: "ListWebACLs", Method: http.MethodPost, IAMAction: "waf-regional:ListWebACLs"},
		{Name: "UpdateWebACL", Method: http.MethodPost, IAMAction: "waf-regional:UpdateWebACL"},
		{Name: "DeleteWebACL", Method: http.MethodPost, IAMAction: "waf-regional:DeleteWebACL"},
		{Name: "CreateRule", Method: http.MethodPost, IAMAction: "waf-regional:CreateRule"},
		{Name: "GetRule", Method: http.MethodPost, IAMAction: "waf-regional:GetRule"},
		{Name: "ListRules", Method: http.MethodPost, IAMAction: "waf-regional:ListRules"},
		{Name: "UpdateRule", Method: http.MethodPost, IAMAction: "waf-regional:UpdateRule"},
		{Name: "DeleteRule", Method: http.MethodPost, IAMAction: "waf-regional:DeleteRule"},
		{Name: "CreateIPSet", Method: http.MethodPost, IAMAction: "waf-regional:CreateIPSet"},
		{Name: "GetIPSet", Method: http.MethodPost, IAMAction: "waf-regional:GetIPSet"},
		{Name: "ListIPSets", Method: http.MethodPost, IAMAction: "waf-regional:ListIPSets"},
		{Name: "UpdateIPSet", Method: http.MethodPost, IAMAction: "waf-regional:UpdateIPSet"},
		{Name: "DeleteIPSet", Method: http.MethodPost, IAMAction: "waf-regional:DeleteIPSet"},
		{Name: "CreateByteMatchSet", Method: http.MethodPost, IAMAction: "waf-regional:CreateByteMatchSet"},
		{Name: "GetByteMatchSet", Method: http.MethodPost, IAMAction: "waf-regional:GetByteMatchSet"},
		{Name: "ListByteMatchSets", Method: http.MethodPost, IAMAction: "waf-regional:ListByteMatchSets"},
		{Name: "DeleteByteMatchSet", Method: http.MethodPost, IAMAction: "waf-regional:DeleteByteMatchSet"},
		{Name: "UpdateByteMatchSet", Method: http.MethodPost, IAMAction: "waf-regional:UpdateByteMatchSet"},
		{Name: "AssociateWebACL", Method: http.MethodPost, IAMAction: "waf-regional:AssociateWebACL"},
		{Name: "DisassociateWebACL", Method: http.MethodPost, IAMAction: "waf-regional:DisassociateWebACL"},
		{Name: "GetWebACLForResource", Method: http.MethodPost, IAMAction: "waf-regional:GetWebACLForResource"},
		{Name: "GetChangeToken", Method: http.MethodPost, IAMAction: "waf-regional:GetChangeToken"},
		{Name: "GetChangeTokenStatus", Method: http.MethodPost, IAMAction: "waf-regional:GetChangeTokenStatus"},
	}
}

// HealthCheck always returns nil.
func (s *WAFRegionalService) HealthCheck() error { return nil }

// HandleRequest routes an incoming WAF Regional request to the appropriate handler.
func (s *WAFRegionalService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateWebACL":
		return handleCreateWebACL(ctx, s.store)
	case "GetWebACL":
		return handleGetWebACL(ctx, s.store)
	case "ListWebACLs":
		return handleListWebACLs(ctx, s.store)
	case "UpdateWebACL":
		return handleUpdateWebACL(ctx, s.store)
	case "DeleteWebACL":
		return handleDeleteWebACL(ctx, s.store)
	case "CreateRule":
		return handleCreateRule(ctx, s.store)
	case "GetRule":
		return handleGetRule(ctx, s.store)
	case "ListRules":
		return handleListRules(ctx, s.store)
	case "UpdateRule":
		return handleUpdateRule(ctx, s.store)
	case "DeleteRule":
		return handleDeleteRule(ctx, s.store)
	case "CreateIPSet":
		return handleCreateIPSet(ctx, s.store)
	case "GetIPSet":
		return handleGetIPSet(ctx, s.store)
	case "ListIPSets":
		return handleListIPSets(ctx, s.store)
	case "UpdateIPSet":
		return handleUpdateIPSet(ctx, s.store)
	case "DeleteIPSet":
		return handleDeleteIPSet(ctx, s.store)
	case "CreateByteMatchSet":
		return handleCreateByteMatchSet(ctx, s.store)
	case "GetByteMatchSet":
		return handleGetByteMatchSet(ctx, s.store)
	case "ListByteMatchSets":
		return handleListByteMatchSets(ctx, s.store)
	case "DeleteByteMatchSet":
		return handleDeleteByteMatchSet(ctx, s.store)
	case "UpdateByteMatchSet":
		return handleUpdateByteMatchSet(ctx, s.store)
	case "AssociateWebACL":
		return handleAssociateWebACL(ctx, s.store)
	case "DisassociateWebACL":
		return handleDisassociateWebACL(ctx, s.store)
	case "GetWebACLForResource":
		return handleGetWebACLForResource(ctx, s.store)
	case "GetChangeToken":
		return handleGetChangeToken(ctx, s.store)
	case "GetChangeTokenStatus":
		return handleGetChangeTokenStatus(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
