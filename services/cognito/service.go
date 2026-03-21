package cognito

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// CognitoService is the cloudmock implementation of the AWS Cognito User Pools API.
type CognitoService struct {
	store *Store
}

// New returns a new CognitoService for the given AWS account ID and region.
func New(accountID, region string) *CognitoService {
	return &CognitoService{
		store: NewStore(accountID, region),
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
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *CognitoService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Cognito User Pools request to the appropriate handler.
// Cognito uses the JSON protocol; the action is parsed from X-Amz-Target by the gateway
// and placed in ctx.Action (e.g. "CreateUserPool").
func (s *CognitoService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
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
		return handleInitiateAuth(ctx, s.store)
	case "AdminConfirmSignUp":
		return handleAdminConfirmSignUp(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
