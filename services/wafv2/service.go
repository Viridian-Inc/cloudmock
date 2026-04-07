package wafv2

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// WAFv2Service is the cloudmock implementation of the AWS WAFv2 API.
type WAFv2Service struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new WAFv2Service for the given AWS account ID and region.
func New(accountID, region string) *WAFv2Service {
	return &WAFv2Service{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *WAFv2Service) Name() string { return "wafv2" }

// Actions returns the list of WAFv2 API actions supported by this service.
func (s *WAFv2Service) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateWebACL", Method: http.MethodPost, IAMAction: "wafv2:CreateWebACL"},
		{Name: "GetWebACL", Method: http.MethodPost, IAMAction: "wafv2:GetWebACL"},
		{Name: "ListWebACLs", Method: http.MethodPost, IAMAction: "wafv2:ListWebACLs"},
		{Name: "UpdateWebACL", Method: http.MethodPost, IAMAction: "wafv2:UpdateWebACL"},
		{Name: "DeleteWebACL", Method: http.MethodPost, IAMAction: "wafv2:DeleteWebACL"},
		{Name: "CreateRuleGroup", Method: http.MethodPost, IAMAction: "wafv2:CreateRuleGroup"},
		{Name: "GetRuleGroup", Method: http.MethodPost, IAMAction: "wafv2:GetRuleGroup"},
		{Name: "ListRuleGroups", Method: http.MethodPost, IAMAction: "wafv2:ListRuleGroups"},
		{Name: "UpdateRuleGroup", Method: http.MethodPost, IAMAction: "wafv2:UpdateRuleGroup"},
		{Name: "DeleteRuleGroup", Method: http.MethodPost, IAMAction: "wafv2:DeleteRuleGroup"},
		{Name: "CreateIPSet", Method: http.MethodPost, IAMAction: "wafv2:CreateIPSet"},
		{Name: "GetIPSet", Method: http.MethodPost, IAMAction: "wafv2:GetIPSet"},
		{Name: "ListIPSets", Method: http.MethodPost, IAMAction: "wafv2:ListIPSets"},
		{Name: "UpdateIPSet", Method: http.MethodPost, IAMAction: "wafv2:UpdateIPSet"},
		{Name: "DeleteIPSet", Method: http.MethodPost, IAMAction: "wafv2:DeleteIPSet"},
		{Name: "CreateRegexPatternSet", Method: http.MethodPost, IAMAction: "wafv2:CreateRegexPatternSet"},
		{Name: "GetRegexPatternSet", Method: http.MethodPost, IAMAction: "wafv2:GetRegexPatternSet"},
		{Name: "ListRegexPatternSets", Method: http.MethodPost, IAMAction: "wafv2:ListRegexPatternSets"},
		{Name: "UpdateRegexPatternSet", Method: http.MethodPost, IAMAction: "wafv2:UpdateRegexPatternSet"},
		{Name: "DeleteRegexPatternSet", Method: http.MethodPost, IAMAction: "wafv2:DeleteRegexPatternSet"},
		{Name: "AssociateWebACL", Method: http.MethodPost, IAMAction: "wafv2:AssociateWebACL"},
		{Name: "DisassociateWebACL", Method: http.MethodPost, IAMAction: "wafv2:DisassociateWebACL"},
		{Name: "GetWebACLForResource", Method: http.MethodPost, IAMAction: "wafv2:GetWebACLForResource"},
		{Name: "PutLoggingConfiguration", Method: http.MethodPost, IAMAction: "wafv2:PutLoggingConfiguration"},
		{Name: "GetLoggingConfiguration", Method: http.MethodPost, IAMAction: "wafv2:GetLoggingConfiguration"},
		{Name: "DeleteLoggingConfiguration", Method: http.MethodPost, IAMAction: "wafv2:DeleteLoggingConfiguration"},
		{Name: "GetSampledRequests", Method: http.MethodPost, IAMAction: "wafv2:GetSampledRequests"},
		{Name: "CheckRequest", Method: http.MethodPost, IAMAction: "wafv2:CheckRequest"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "wafv2:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "wafv2:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "wafv2:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *WAFv2Service) HealthCheck() error { return nil }

// HandleRequest routes an incoming WAFv2 request to the appropriate handler.
func (s *WAFv2Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
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
	case "CreateRuleGroup":
		return handleCreateRuleGroup(ctx, s.store)
	case "GetRuleGroup":
		return handleGetRuleGroup(ctx, s.store)
	case "ListRuleGroups":
		return handleListRuleGroups(ctx, s.store)
	case "UpdateRuleGroup":
		return handleUpdateRuleGroup(ctx, s.store)
	case "DeleteRuleGroup":
		return handleDeleteRuleGroup(ctx, s.store)
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
	case "CreateRegexPatternSet":
		return handleCreateRegexPatternSet(ctx, s.store)
	case "GetRegexPatternSet":
		return handleGetRegexPatternSet(ctx, s.store)
	case "ListRegexPatternSets":
		return handleListRegexPatternSets(ctx, s.store)
	case "UpdateRegexPatternSet":
		return handleUpdateRegexPatternSet(ctx, s.store)
	case "DeleteRegexPatternSet":
		return handleDeleteRegexPatternSet(ctx, s.store)
	case "AssociateWebACL":
		return handleAssociateWebACL(ctx, s.store)
	case "DisassociateWebACL":
		return handleDisassociateWebACL(ctx, s.store)
	case "GetWebACLForResource":
		return handleGetWebACLForResource(ctx, s.store)
	case "PutLoggingConfiguration":
		return handlePutLoggingConfiguration(ctx, s.store)
	case "GetLoggingConfiguration":
		return handleGetLoggingConfiguration(ctx, s.store)
	case "DeleteLoggingConfiguration":
		return handleDeleteLoggingConfiguration(ctx, s.store)
	case "GetSampledRequests":
		return handleGetSampledRequests(ctx, s.store)
	case "CheckRequest":
		return handleCheckRequest(ctx, s.store)
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
