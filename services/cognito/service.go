package cognito

import (
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// CognitoService is the cloudmock implementation of the AWS Cognito User Pools API.
type CognitoService struct {
	store     *Store
	keys      *KeyStore
	authCodes *authCodeStore
}

// New returns a new CognitoService for the given AWS account ID and region.
func New(accountID, region string) *CognitoService {
	keys, err := NewKeyStore()
	if err != nil {
		panic("cognito: failed to generate RSA key pair: " + err.Error())
	}
	return &CognitoService{
		store:     NewStore(accountID, region),
		keys:      keys,
		authCodes: newAuthCodeStore(),
	}
}

// Name returns the AWS service credential scope name used for routing.
// The Authorization header credential scope for Cognito User Pools is "cognito-idp".
func (s *CognitoService) Name() string { return "cognito-idp" }

// Actions returns the list of Cognito User Pools API actions supported by this service.
func (s *CognitoService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateUserPool", Method: http.MethodPost, IAMAction: "cognito-idp:CreateUserPool"},
		{Name: "DeleteUserPool", Method: http.MethodPost, IAMAction: "cognito-idp:DeleteUserPool"},
		{Name: "DescribeUserPool", Method: http.MethodPost, IAMAction: "cognito-idp:DescribeUserPool"},
		{Name: "ListUserPools", Method: http.MethodPost, IAMAction: "cognito-idp:ListUserPools"},
		{Name: "CreateUserPoolClient", Method: http.MethodPost, IAMAction: "cognito-idp:CreateUserPoolClient"},
		{Name: "DescribeUserPoolClient", Method: http.MethodPost, IAMAction: "cognito-idp:DescribeUserPoolClient"},
		{Name: "ListUserPoolClients", Method: http.MethodPost, IAMAction: "cognito-idp:ListUserPoolClients"},
		{Name: "AdminCreateUser", Method: http.MethodPost, IAMAction: "cognito-idp:AdminCreateUser"},
		{Name: "AdminGetUser", Method: http.MethodPost, IAMAction: "cognito-idp:AdminGetUser"},
		{Name: "AdminDeleteUser", Method: http.MethodPost, IAMAction: "cognito-idp:AdminDeleteUser"},
		{Name: "AdminSetUserPassword", Method: http.MethodPost, IAMAction: "cognito-idp:AdminSetUserPassword"},
		{Name: "SignUp", Method: http.MethodPost, IAMAction: "cognito-idp:SignUp"},
		{Name: "InitiateAuth", Method: http.MethodPost, IAMAction: "cognito-idp:InitiateAuth"},
		{Name: "AdminConfirmSignUp", Method: http.MethodPost, IAMAction: "cognito-idp:AdminConfirmSignUp"},
		// Groups
		{Name: "CreateGroup", Method: http.MethodPost, IAMAction: "cognito-idp:CreateGroup"},
		{Name: "DeleteGroup", Method: http.MethodPost, IAMAction: "cognito-idp:DeleteGroup"},
		{Name: "GetGroup", Method: http.MethodPost, IAMAction: "cognito-idp:GetGroup"},
		{Name: "ListGroups", Method: http.MethodPost, IAMAction: "cognito-idp:ListGroups"},
		{Name: "AdminAddUserToGroup", Method: http.MethodPost, IAMAction: "cognito-idp:AdminAddUserToGroup"},
		{Name: "AdminRemoveUserFromGroup", Method: http.MethodPost, IAMAction: "cognito-idp:AdminRemoveUserFromGroup"},
		// Password reset
		{Name: "ForgotPassword", Method: http.MethodPost, IAMAction: "cognito-idp:ForgotPassword"},
		{Name: "ConfirmForgotPassword", Method: http.MethodPost, IAMAction: "cognito-idp:ConfirmForgotPassword"},
		{Name: "ChangePassword", Method: http.MethodPost, IAMAction: "cognito-idp:ChangePassword"},
		// Identity providers (SAML / OIDC federation)
		{Name: "CreateIdentityProvider", Method: http.MethodPost, IAMAction: "cognito-idp:CreateIdentityProvider"},
		{Name: "DescribeIdentityProvider", Method: http.MethodPost, IAMAction: "cognito-idp:DescribeIdentityProvider"},
		{Name: "UpdateIdentityProvider", Method: http.MethodPost, IAMAction: "cognito-idp:UpdateIdentityProvider"},
		{Name: "DeleteIdentityProvider", Method: http.MethodPost, IAMAction: "cognito-idp:DeleteIdentityProvider"},
		{Name: "ListIdentityProviders", Method: http.MethodPost, IAMAction: "cognito-idp:ListIdentityProviders"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *CognitoService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for Cognito resource types.
func (s *CognitoService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "cognito",
			ResourceType:  "aws_cognito_user_pool",
			TerraformType: "cloudmock_cognito_user_pool",
			AWSType:       "AWS::Cognito::UserPool",
			CreateAction:  "CreateUserPool",
			ReadAction:    "DescribeUserPool",
			DeleteAction:  "DeleteUserPool",
			ListAction:    "ListUserPools",
			ImportID:      "id",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "id", Type: "string", Computed: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "endpoint", Type: "string", Computed: true},
				{Name: "alias_attributes", Type: "list"},
				{Name: "auto_verified_attributes", Type: "list"},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "cognito",
			ResourceType:  "aws_cognito_user_pool_client",
			TerraformType: "cloudmock_cognito_user_pool_client",
			AWSType:       "AWS::Cognito::UserPoolClient",
			CreateAction:  "CreateUserPoolClient",
			ReadAction:    "DescribeUserPoolClient",
			ListAction:    "ListUserPoolClients",
			ImportID:      "user_pool_id/id",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "user_pool_id", Type: "string", Required: true, ForceNew: true},
				{Name: "id", Type: "string", Computed: true},
				{Name: "generate_secret", Type: "bool", Default: false},
				{Name: "allowed_oauth_flows", Type: "list"},
				{Name: "allowed_oauth_scopes", Type: "list"},
				{Name: "callback_urls", Type: "list"},
			},
		},
	}
}

// HandleRequest routes an incoming Cognito User Pools request to the appropriate handler.
// It first checks for OAuth/OIDC REST-style paths, then falls back to the JSON protocol
// where the action is parsed from X-Amz-Target by the gateway and placed in ctx.Action.
func (s *CognitoService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	path := ctx.RawRequest.URL.Path

	// OAuth/OIDC REST endpoints — route by path.
	if strings.Contains(path, "/.well-known/") || strings.HasPrefix(path, "/oauth2/") || path == "/login" {
		return s.handleOAuth(ctx)
	}

	// Existing JSON protocol — route by action.
	switch ctx.Action {
	case "CreateUserPool":
		return handleCreateUserPool(ctx, s.store)
	case "DeleteUserPool":
		return handleDeleteUserPool(ctx, s.store)
	case "DescribeUserPool":
		return handleDescribeUserPool(ctx, s.store)
	case "ListUserPools":
		return handleListUserPools(ctx, s.store)
	case "CreateUserPoolClient":
		return handleCreateUserPoolClient(ctx, s.store)
	case "DescribeUserPoolClient":
		return handleDescribeUserPoolClient(ctx, s.store)
	case "ListUserPoolClients":
		return handleListUserPoolClients(ctx, s.store)
	case "AdminCreateUser":
		return handleAdminCreateUser(ctx, s.store)
	case "AdminGetUser":
		return handleAdminGetUser(ctx, s.store)
	case "AdminDeleteUser":
		return handleAdminDeleteUser(ctx, s.store)
	case "AdminSetUserPassword":
		return handleAdminSetUserPassword(ctx, s.store)
	case "SignUp":
		return handleSignUp(ctx, s.store)
	case "InitiateAuth":
		return handleInitiateAuth(ctx, s.store, s.keys)
	case "AdminConfirmSignUp":
		return handleAdminConfirmSignUp(ctx, s.store)
	// Groups
	case "CreateGroup":
		return handleCreateGroup(ctx, s.store)
	case "DeleteGroup":
		return handleDeleteGroup(ctx, s.store)
	case "GetGroup":
		return handleGetGroup(ctx, s.store)
	case "ListGroups":
		return handleListGroups(ctx, s.store)
	case "AdminAddUserToGroup":
		return handleAdminAddUserToGroup(ctx, s.store)
	case "AdminRemoveUserFromGroup":
		return handleAdminRemoveUserFromGroup(ctx, s.store)
	// Password reset
	case "ForgotPassword":
		return handleForgotPassword(ctx, s.store)
	case "ConfirmForgotPassword":
		return handleConfirmForgotPassword(ctx, s.store)
	case "ChangePassword":
		return handleChangePassword(ctx, s.store)
	// Identity providers (SAML / OIDC federation)
	case "CreateIdentityProvider":
		return handleCreateIdentityProvider(ctx, s.store)
	case "DescribeIdentityProvider":
		return handleDescribeIdentityProvider(ctx, s.store)
	case "UpdateIdentityProvider":
		return handleUpdateIdentityProvider(ctx, s.store)
	case "DeleteIdentityProvider":
		return handleDeleteIdentityProvider(ctx, s.store)
	case "ListIdentityProviders":
		return handleListIdentityProviders(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
