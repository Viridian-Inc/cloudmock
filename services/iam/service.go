package iam

import (
	"net/http"
	"net/url"

	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
	"github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// IAMService is the cloudmock implementation of the AWS IAM API.
type IAMService struct {
	accountID string
	store     *Store
}

// New returns a new IAMService for the given account, backed by the existing
// IAM engine and pkg store (for credential/auth integration).
func New(accountID string, engine *iampkg.Engine, pkgStore *iampkg.Store) *IAMService {
	return &IAMService{
		accountID: accountID,
		store:     NewStore(accountID, engine, pkgStore),
	}
}

// Name returns the AWS service name used for routing.
func (s *IAMService) Name() string { return "iam" }

// HealthCheck always returns nil.
func (s *IAMService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for IAM resource types.
func (s *IAMService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "iam",
			ResourceType:  "aws_iam_role",
			TerraformType: "cloudmock_iam_role",
			AWSType:       "AWS::IAM::Role",
			CreateAction:  "CreateRole",
			ReadAction:    "GetRole",
			DeleteAction:  "DeleteRole",
			ListAction:    "ListRoles",
			ImportID:      "name",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "assume_role_policy", Type: "string", Required: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "unique_id", Type: "string", Computed: true},
				{Name: "path", Type: "string", Default: "/"},
				{Name: "description", Type: "string"},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "iam",
			ResourceType:  "aws_iam_policy",
			TerraformType: "cloudmock_iam_policy",
			AWSType:       "AWS::IAM::ManagedPolicy",
			CreateAction:  "CreatePolicy",
			ReadAction:    "GetPolicy",
			DeleteAction:  "DeletePolicy",
			ListAction:    "ListPolicies",
			ImportID:      "arn",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "policy", Type: "string", Required: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "path", Type: "string", Default: "/"},
				{Name: "description", Type: "string"},
			},
		},
		{
			ServiceName:   "iam",
			ResourceType:  "aws_iam_user",
			TerraformType: "cloudmock_iam_user",
			AWSType:       "AWS::IAM::User",
			CreateAction:  "CreateUser",
			ReadAction:    "GetUser",
			DeleteAction:  "DeleteUser",
			ListAction:    "ListUsers",
			ImportID:      "name",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "unique_id", Type: "string", Computed: true},
				{Name: "path", Type: "string", Default: "/"},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// Actions returns all IAM API actions this service supports.
func (s *IAMService) Actions() []service.Action {
	return []service.Action{
		// Users
		{Name: "CreateUser", Method: http.MethodPost, IAMAction: "iam:CreateUser"},
		{Name: "GetUser", Method: http.MethodPost, IAMAction: "iam:GetUser"},
		{Name: "ListUsers", Method: http.MethodPost, IAMAction: "iam:ListUsers"},
		{Name: "DeleteUser", Method: http.MethodPost, IAMAction: "iam:DeleteUser"},
		{Name: "UpdateUser", Method: http.MethodPost, IAMAction: "iam:UpdateUser"},
		// Tags
		{Name: "TagUser", Method: http.MethodPost, IAMAction: "iam:TagUser"},
		{Name: "UntagUser", Method: http.MethodPost, IAMAction: "iam:UntagUser"},
		{Name: "ListUserTags", Method: http.MethodPost, IAMAction: "iam:ListUserTags"},
		// Roles
		{Name: "CreateRole", Method: http.MethodPost, IAMAction: "iam:CreateRole"},
		{Name: "GetRole", Method: http.MethodPost, IAMAction: "iam:GetRole"},
		{Name: "ListRoles", Method: http.MethodPost, IAMAction: "iam:ListRoles"},
		{Name: "DeleteRole", Method: http.MethodPost, IAMAction: "iam:DeleteRole"},
		// Policies
		{Name: "CreatePolicy", Method: http.MethodPost, IAMAction: "iam:CreatePolicy"},
		{Name: "GetPolicy", Method: http.MethodPost, IAMAction: "iam:GetPolicy"},
		{Name: "ListPolicies", Method: http.MethodPost, IAMAction: "iam:ListPolicies"},
		{Name: "DeletePolicy", Method: http.MethodPost, IAMAction: "iam:DeletePolicy"},
		{Name: "AttachUserPolicy", Method: http.MethodPost, IAMAction: "iam:AttachUserPolicy"},
		{Name: "DetachUserPolicy", Method: http.MethodPost, IAMAction: "iam:DetachUserPolicy"},
		{Name: "AttachRolePolicy", Method: http.MethodPost, IAMAction: "iam:AttachRolePolicy"},
		{Name: "DetachRolePolicy", Method: http.MethodPost, IAMAction: "iam:DetachRolePolicy"},
		{Name: "ListAttachedUserPolicies", Method: http.MethodPost, IAMAction: "iam:ListAttachedUserPolicies"},
		{Name: "ListAttachedRolePolicies", Method: http.MethodPost, IAMAction: "iam:ListAttachedRolePolicies"},
		// Groups
		{Name: "CreateGroup", Method: http.MethodPost, IAMAction: "iam:CreateGroup"},
		{Name: "GetGroup", Method: http.MethodPost, IAMAction: "iam:GetGroup"},
		{Name: "ListGroups", Method: http.MethodPost, IAMAction: "iam:ListGroups"},
		{Name: "DeleteGroup", Method: http.MethodPost, IAMAction: "iam:DeleteGroup"},
		{Name: "AddUserToGroup", Method: http.MethodPost, IAMAction: "iam:AddUserToGroup"},
		{Name: "RemoveUserFromGroup", Method: http.MethodPost, IAMAction: "iam:RemoveUserFromGroup"},
		// Access Keys
		{Name: "CreateAccessKey", Method: http.MethodPost, IAMAction: "iam:CreateAccessKey"},
		{Name: "ListAccessKeys", Method: http.MethodPost, IAMAction: "iam:ListAccessKeys"},
		{Name: "DeleteAccessKey", Method: http.MethodPost, IAMAction: "iam:DeleteAccessKey"},
		// Instance Profiles
		{Name: "CreateInstanceProfile", Method: http.MethodPost, IAMAction: "iam:CreateInstanceProfile"},
		{Name: "GetInstanceProfile", Method: http.MethodPost, IAMAction: "iam:GetInstanceProfile"},
		{Name: "ListInstanceProfiles", Method: http.MethodPost, IAMAction: "iam:ListInstanceProfiles"},
		{Name: "DeleteInstanceProfile", Method: http.MethodPost, IAMAction: "iam:DeleteInstanceProfile"},
		{Name: "AddRoleToInstanceProfile", Method: http.MethodPost, IAMAction: "iam:AddRoleToInstanceProfile"},
		{Name: "RemoveRoleFromInstanceProfile", Method: http.MethodPost, IAMAction: "iam:RemoveRoleFromInstanceProfile"},
		// OIDC Providers
		{Name: "CreateOpenIDConnectProvider", Method: http.MethodPost, IAMAction: "iam:CreateOpenIDConnectProvider"},
		{Name: "GetOpenIDConnectProvider", Method: http.MethodPost, IAMAction: "iam:GetOpenIDConnectProvider"},
		{Name: "ListOpenIDConnectProviders", Method: http.MethodPost, IAMAction: "iam:ListOpenIDConnectProviders"},
		{Name: "DeleteOpenIDConnectProvider", Method: http.MethodPost, IAMAction: "iam:DeleteOpenIDConnectProvider"},
		// SAML Providers
		{Name: "CreateSAMLProvider", Method: http.MethodPost, IAMAction: "iam:CreateSAMLProvider"},
		{Name: "ListSAMLProviders", Method: http.MethodPost, IAMAction: "iam:ListSAMLProviders"},
		{Name: "DeleteSAMLProvider", Method: http.MethodPost, IAMAction: "iam:DeleteSAMLProvider"},
	}
}

// HandleRequest routes an incoming IAM request to the appropriate handler.
// IAM uses form-encoded POST bodies with an Action parameter.
func (s *IAMService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action

	// If Action was not extracted from query string, try the form body.
	if action == "" {
		if formVals, err := url.ParseQuery(string(ctx.Body)); err == nil {
			action = formVals.Get("Action")
		}
	}

	form, _ := url.ParseQuery(string(ctx.Body))
	// Also merge query params
	for k, v := range ctx.Params {
		if form.Get(k) == "" {
			form.Set(k, v)
		}
	}

	switch action {
	// Users
	case "CreateUser":
		return handleCreateUser(s.store, form)
	case "GetUser":
		return handleGetUser(s.store, form)
	case "ListUsers":
		return handleListUsers(s.store)
	case "DeleteUser":
		return handleDeleteUser(s.store, form)
	case "UpdateUser":
		return handleUpdateUser(s.store, form)
	// Tags
	case "TagUser":
		return handleTagUser(s.store, form)
	case "UntagUser":
		return handleUntagUser(s.store, form)
	case "ListUserTags":
		return handleListUserTags(s.store, form)
	// Roles
	case "CreateRole":
		return handleCreateRole(s.store, form)
	case "GetRole":
		return handleGetRole(s.store, form)
	case "ListRoles":
		return handleListRoles(s.store)
	case "DeleteRole":
		return handleDeleteRole(s.store, form)
	// Policies
	case "CreatePolicy":
		return handleCreatePolicy(s.store, form)
	case "GetPolicy":
		return handleGetPolicy(s.store, form)
	case "ListPolicies":
		return handleListPolicies(s.store)
	case "DeletePolicy":
		return handleDeletePolicy(s.store, form)
	case "AttachUserPolicy":
		return handleAttachUserPolicy(s.store, form)
	case "DetachUserPolicy":
		return handleDetachUserPolicy(s.store, form)
	case "AttachRolePolicy":
		return handleAttachRolePolicy(s.store, form)
	case "DetachRolePolicy":
		return handleDetachRolePolicy(s.store, form)
	case "ListAttachedUserPolicies":
		return handleListAttachedUserPolicies(s.store, form)
	case "ListAttachedRolePolicies":
		return handleListAttachedRolePolicies(s.store, form)
	// Groups
	case "CreateGroup":
		return handleCreateGroup(s.store, form)
	case "GetGroup":
		return handleGetGroup(s.store, form)
	case "ListGroups":
		return handleListGroups(s.store)
	case "DeleteGroup":
		return handleDeleteGroup(s.store, form)
	case "AddUserToGroup":
		return handleAddUserToGroup(s.store, form)
	case "RemoveUserFromGroup":
		return handleRemoveUserFromGroup(s.store, form)
	// Access Keys
	case "CreateAccessKey":
		return handleCreateAccessKey(s.store, form)
	case "ListAccessKeys":
		return handleListAccessKeys(s.store, form)
	case "DeleteAccessKey":
		return handleDeleteAccessKey(s.store, form)
	// Instance Profiles
	case "CreateInstanceProfile":
		return handleCreateInstanceProfile(s.store, form)
	case "GetInstanceProfile":
		return handleGetInstanceProfile(s.store, form)
	case "ListInstanceProfiles":
		return handleListInstanceProfiles(s.store)
	case "DeleteInstanceProfile":
		return handleDeleteInstanceProfile(s.store, form)
	case "AddRoleToInstanceProfile":
		return handleAddRoleToInstanceProfile(s.store, form)
	case "RemoveRoleFromInstanceProfile":
		return handleRemoveRoleFromInstanceProfile(s.store, form)
	// OIDC Providers
	case "CreateOpenIDConnectProvider":
		return handleCreateOpenIDConnectProvider(s.store, form)
	case "GetOpenIDConnectProvider":
		return handleGetOpenIDConnectProvider(s.store, form)
	case "ListOpenIDConnectProviders":
		return handleListOpenIDConnectProviders(s.store, form)
	case "DeleteOpenIDConnectProvider":
		return handleDeleteOpenIDConnectProvider(s.store, form)
	// SAML Providers
	case "CreateSAMLProvider":
		return handleCreateSAMLProvider(s.store, form)
	case "ListSAMLProviders":
		return handleListSAMLProviders(s.store, form)
	case "DeleteSAMLProvider":
		return handleDeleteSAMLProvider(s.store, form)
	default:
		awsErr := service.NewAWSError(
			"InvalidAction",
			"The action "+action+" is not valid for this web service.",
			http.StatusBadRequest,
		)
		return &service.Response{Format: service.FormatXML}, awsErr
	}
}
