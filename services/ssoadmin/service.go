package ssoadmin

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// SSOAdminService is the cloudmock implementation of the AWS SSO Admin API.
type SSOAdminService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new SSOAdminService for the given AWS account ID and region.
func New(accountID, region string) *SSOAdminService {
	return &SSOAdminService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *SSOAdminService) Name() string { return "sso-admin" }

// Actions returns the list of SSO Admin API actions supported by this service.
func (s *SSOAdminService) Actions() []service.Action {
	return []service.Action{
		{Name: "ListInstances", Method: http.MethodPost, IAMAction: "sso:ListInstances"},
		{Name: "DescribeInstance", Method: http.MethodPost, IAMAction: "sso:DescribeInstance"},
		{Name: "CreatePermissionSet", Method: http.MethodPost, IAMAction: "sso:CreatePermissionSet"},
		{Name: "DescribePermissionSet", Method: http.MethodPost, IAMAction: "sso:DescribePermissionSet"},
		{Name: "ListPermissionSets", Method: http.MethodPost, IAMAction: "sso:ListPermissionSets"},
		{Name: "UpdatePermissionSet", Method: http.MethodPost, IAMAction: "sso:UpdatePermissionSet"},
		{Name: "DeletePermissionSet", Method: http.MethodPost, IAMAction: "sso:DeletePermissionSet"},
		{Name: "CreateAccountAssignment", Method: http.MethodPost, IAMAction: "sso:CreateAccountAssignment"},
		{Name: "ListAccountAssignments", Method: http.MethodPost, IAMAction: "sso:ListAccountAssignments"},
		{Name: "DeleteAccountAssignment", Method: http.MethodPost, IAMAction: "sso:DeleteAccountAssignment"},
		{Name: "AttachManagedPolicyToPermissionSet", Method: http.MethodPost, IAMAction: "sso:AttachManagedPolicyToPermissionSet"},
		{Name: "DetachManagedPolicyFromPermissionSet", Method: http.MethodPost, IAMAction: "sso:DetachManagedPolicyFromPermissionSet"},
		{Name: "ListManagedPoliciesInPermissionSet", Method: http.MethodPost, IAMAction: "sso:ListManagedPoliciesInPermissionSet"},
		{Name: "PutInlinePolicyToPermissionSet", Method: http.MethodPost, IAMAction: "sso:PutInlinePolicyToPermissionSet"},
		{Name: "GetInlinePolicyForPermissionSet", Method: http.MethodPost, IAMAction: "sso:GetInlinePolicyForPermissionSet"},
		{Name: "DeleteInlinePolicyFromPermissionSet", Method: http.MethodPost, IAMAction: "sso:DeleteInlinePolicyFromPermissionSet"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "sso:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "sso:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "sso:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *SSOAdminService) HealthCheck() error { return nil }

// HandleRequest routes an incoming SSO Admin request to the appropriate handler.
func (s *SSOAdminService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "ListInstances":
		return handleListInstances(ctx, s.store)
	case "DescribeInstance":
		return handleDescribeInstance(ctx, s.store)
	case "CreatePermissionSet":
		return handleCreatePermissionSet(ctx, s.store)
	case "DescribePermissionSet":
		return handleDescribePermissionSet(ctx, s.store)
	case "ListPermissionSets":
		return handleListPermissionSets(ctx, s.store)
	case "UpdatePermissionSet":
		return handleUpdatePermissionSet(ctx, s.store)
	case "DeletePermissionSet":
		return handleDeletePermissionSet(ctx, s.store)
	case "CreateAccountAssignment":
		return handleCreateAccountAssignment(ctx, s.store)
	case "ListAccountAssignments":
		return handleListAccountAssignments(ctx, s.store)
	case "DeleteAccountAssignment":
		return handleDeleteAccountAssignment(ctx, s.store)
	case "AttachManagedPolicyToPermissionSet":
		return handleAttachManagedPolicy(ctx, s.store)
	case "DetachManagedPolicyFromPermissionSet":
		return handleDetachManagedPolicy(ctx, s.store)
	case "ListManagedPoliciesInPermissionSet":
		return handleListManagedPolicies(ctx, s.store)
	case "PutInlinePolicyToPermissionSet":
		return handlePutInlinePolicy(ctx, s.store)
	case "GetInlinePolicyForPermissionSet":
		return handleGetInlinePolicy(ctx, s.store)
	case "DeleteInlinePolicyFromPermissionSet":
		return handleDeleteInlinePolicy(ctx, s.store)
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
