package identitystore

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// IdentityStoreService is the cloudmock implementation of the AWS Identity Store API.
type IdentityStoreService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new IdentityStoreService for the given AWS account ID and region.
func New(accountID, region string) *IdentityStoreService {
	return &IdentityStoreService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *IdentityStoreService) Name() string { return "identitystore" }

// Actions returns the list of Identity Store API actions supported by this service.
func (s *IdentityStoreService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateUser", Method: http.MethodPost, IAMAction: "identitystore:CreateUser"},
		{Name: "DescribeUser", Method: http.MethodPost, IAMAction: "identitystore:DescribeUser"},
		{Name: "ListUsers", Method: http.MethodPost, IAMAction: "identitystore:ListUsers"},
		{Name: "UpdateUser", Method: http.MethodPost, IAMAction: "identitystore:UpdateUser"},
		{Name: "DeleteUser", Method: http.MethodPost, IAMAction: "identitystore:DeleteUser"},
		{Name: "CreateGroup", Method: http.MethodPost, IAMAction: "identitystore:CreateGroup"},
		{Name: "DescribeGroup", Method: http.MethodPost, IAMAction: "identitystore:DescribeGroup"},
		{Name: "ListGroups", Method: http.MethodPost, IAMAction: "identitystore:ListGroups"},
		{Name: "UpdateGroup", Method: http.MethodPost, IAMAction: "identitystore:UpdateGroup"},
		{Name: "DeleteGroup", Method: http.MethodPost, IAMAction: "identitystore:DeleteGroup"},
		{Name: "CreateGroupMembership", Method: http.MethodPost, IAMAction: "identitystore:CreateGroupMembership"},
		{Name: "GetGroupMembership", Method: http.MethodPost, IAMAction: "identitystore:GetGroupMembership"},
		{Name: "GetGroupMembershipId", Method: http.MethodPost, IAMAction: "identitystore:GetGroupMembershipId"},
		{Name: "ListGroupMemberships", Method: http.MethodPost, IAMAction: "identitystore:ListGroupMemberships"},
		{Name: "DeleteGroupMembership", Method: http.MethodPost, IAMAction: "identitystore:DeleteGroupMembership"},
		{Name: "IsMemberInGroups", Method: http.MethodPost, IAMAction: "identitystore:IsMemberInGroups"},
	}
}

// HealthCheck always returns nil.
func (s *IdentityStoreService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Identity Store request to the appropriate handler.
func (s *IdentityStoreService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateUser":
		return handleCreateUser(params, s.store)
	case "DescribeUser":
		return handleDescribeUser(params, s.store)
	case "ListUsers":
		return handleListUsers(params, s.store)
	case "UpdateUser":
		return handleUpdateUser(params, s.store)
	case "DeleteUser":
		return handleDeleteUser(params, s.store)
	case "CreateGroup":
		return handleCreateGroup(params, s.store)
	case "DescribeGroup":
		return handleDescribeGroup(params, s.store)
	case "ListGroups":
		return handleListGroups(params, s.store)
	case "UpdateGroup":
		return handleUpdateGroup(params, s.store)
	case "DeleteGroup":
		return handleDeleteGroup(params, s.store)
	case "CreateGroupMembership":
		return handleCreateGroupMembership(params, s.store)
	case "GetGroupMembership":
		return handleGetGroupMembership(params, s.store)
	case "GetGroupMembershipId":
		return handleGetGroupMembershipID(params, s.store)
	case "ListGroupMemberships":
		return handleListGroupMemberships(params, s.store)
	case "DeleteGroupMembership":
		return handleDeleteGroupMembership(params, s.store)
	case "IsMemberInGroups":
		return handleIsMemberInGroups(params, s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
