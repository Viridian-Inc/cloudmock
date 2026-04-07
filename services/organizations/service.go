package organizations

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// OrganizationsService is the cloudmock implementation of the AWS Organizations API.
type OrganizationsService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new OrganizationsService for the given AWS account ID and region.
func New(accountID, region string) *OrganizationsService {
	return &OrganizationsService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *OrganizationsService) Name() string { return "organizations" }

// Actions returns the list of Organizations API actions supported by this service.
func (s *OrganizationsService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrganization", Method: http.MethodPost, IAMAction: "organizations:CreateOrganization"},
		{Name: "DescribeOrganization", Method: http.MethodPost, IAMAction: "organizations:DescribeOrganization"},
		{Name: "DeleteOrganization", Method: http.MethodPost, IAMAction: "organizations:DeleteOrganization"},
		{Name: "ListRoots", Method: http.MethodPost, IAMAction: "organizations:ListRoots"},
		{Name: "CreateOrganizationalUnit", Method: http.MethodPost, IAMAction: "organizations:CreateOrganizationalUnit"},
		{Name: "DescribeOrganizationalUnit", Method: http.MethodPost, IAMAction: "organizations:DescribeOrganizationalUnit"},
		{Name: "ListOrganizationalUnitsForParent", Method: http.MethodPost, IAMAction: "organizations:ListOrganizationalUnitsForParent"},
		{Name: "DeleteOrganizationalUnit", Method: http.MethodPost, IAMAction: "organizations:DeleteOrganizationalUnit"},
		{Name: "CreateAccount", Method: http.MethodPost, IAMAction: "organizations:CreateAccount"},
		{Name: "DescribeAccount", Method: http.MethodPost, IAMAction: "organizations:DescribeAccount"},
		{Name: "ListAccounts", Method: http.MethodPost, IAMAction: "organizations:ListAccounts"},
		{Name: "ListAccountsForParent", Method: http.MethodPost, IAMAction: "organizations:ListAccountsForParent"},
		{Name: "MoveAccount", Method: http.MethodPost, IAMAction: "organizations:MoveAccount"},
		{Name: "CreatePolicy", Method: http.MethodPost, IAMAction: "organizations:CreatePolicy"},
		{Name: "DescribePolicy", Method: http.MethodPost, IAMAction: "organizations:DescribePolicy"},
		{Name: "ListPolicies", Method: http.MethodPost, IAMAction: "organizations:ListPolicies"},
		{Name: "UpdatePolicy", Method: http.MethodPost, IAMAction: "organizations:UpdatePolicy"},
		{Name: "DeletePolicy", Method: http.MethodPost, IAMAction: "organizations:DeletePolicy"},
		{Name: "ListChildren", Method: http.MethodPost, IAMAction: "organizations:ListChildren"},
		{Name: "ListParents", Method: http.MethodPost, IAMAction: "organizations:ListParents"},
		{Name: "AttachPolicy", Method: http.MethodPost, IAMAction: "organizations:AttachPolicy"},
		{Name: "DetachPolicy", Method: http.MethodPost, IAMAction: "organizations:DetachPolicy"},
		{Name: "ListTargetsForPolicy", Method: http.MethodPost, IAMAction: "organizations:ListTargetsForPolicy"},
		{Name: "ListPoliciesForTarget", Method: http.MethodPost, IAMAction: "organizations:ListPoliciesForTarget"},
		{Name: "EnablePolicyType", Method: http.MethodPost, IAMAction: "organizations:EnablePolicyType"},
		{Name: "DisablePolicyType", Method: http.MethodPost, IAMAction: "organizations:DisablePolicyType"},
		{Name: "DescribeCreateAccountStatus", Method: http.MethodPost, IAMAction: "organizations:DescribeCreateAccountStatus"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "organizations:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "organizations:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "organizations:ListTagsForResource"},
	}
}

// SetProvisioner attaches an AccountProvisioner for multi-account integration.
func (s *OrganizationsService) SetProvisioner(p AccountProvisioner) {
	s.store.SetProvisioner(p)
}

// HealthCheck always returns nil.
func (s *OrganizationsService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Organizations request to the appropriate handler.
func (s *OrganizationsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateOrganization":
		return handleCreateOrganization(ctx, s.store)
	case "DescribeOrganization":
		return handleDescribeOrganization(ctx, s.store)
	case "DeleteOrganization":
		return handleDeleteOrganization(ctx, s.store)
	case "ListRoots":
		return handleListRoots(ctx, s.store)
	case "CreateOrganizationalUnit":
		return handleCreateOrganizationalUnit(ctx, s.store)
	case "DescribeOrganizationalUnit":
		return handleDescribeOrganizationalUnit(ctx, s.store)
	case "ListOrganizationalUnitsForParent":
		return handleListOrganizationalUnitsForParent(ctx, s.store)
	case "DeleteOrganizationalUnit":
		return handleDeleteOrganizationalUnit(ctx, s.store)
	case "CreateAccount":
		return handleCreateAccount(ctx, s.store)
	case "DescribeAccount":
		return handleDescribeAccount(ctx, s.store)
	case "ListAccounts":
		return handleListAccounts(ctx, s.store)
	case "ListAccountsForParent":
		return handleListAccountsForParent(ctx, s.store)
	case "MoveAccount":
		return handleMoveAccount(ctx, s.store)
	case "CreatePolicy":
		return handleCreatePolicy(ctx, s.store)
	case "DescribePolicy":
		return handleDescribePolicy(ctx, s.store)
	case "ListPolicies":
		return handleListPolicies(ctx, s.store)
	case "UpdatePolicy":
		return handleUpdatePolicy(ctx, s.store)
	case "DeletePolicy":
		return handleDeletePolicy(ctx, s.store)
	case "ListChildren":
		return handleListChildren(ctx, s.store)
	case "ListParents":
		return handleListParents(ctx, s.store)
	case "AttachPolicy":
		return handleAttachPolicy(ctx, s.store)
	case "DetachPolicy":
		return handleDetachPolicy(ctx, s.store)
	case "ListTargetsForPolicy":
		return handleListTargetsForPolicy(ctx, s.store)
	case "ListPoliciesForTarget":
		return handleListPoliciesForTarget(ctx, s.store)
	case "EnablePolicyType":
		return handleEnablePolicyType(ctx, s.store)
	case "DisablePolicyType":
		return handleDisablePolicyType(ctx, s.store)
	case "DescribeCreateAccountStatus":
		return handleDescribeCreateAccountStatus(ctx, s.store)
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
