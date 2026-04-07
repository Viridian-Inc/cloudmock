package verifiedpermissions

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// VerifiedPermissionsService is the cloudmock implementation of the Amazon Verified Permissions API.
type VerifiedPermissionsService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new VerifiedPermissionsService for the given AWS account ID and region.
func New(accountID, region string) *VerifiedPermissionsService {
	return &VerifiedPermissionsService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *VerifiedPermissionsService) Name() string { return "verifiedpermissions" }

// Actions returns the list of Verified Permissions API actions supported by this service.
func (s *VerifiedPermissionsService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreatePolicyStore", Method: http.MethodPost, IAMAction: "verifiedpermissions:CreatePolicyStore"},
		{Name: "GetPolicyStore", Method: http.MethodPost, IAMAction: "verifiedpermissions:GetPolicyStore"},
		{Name: "ListPolicyStores", Method: http.MethodPost, IAMAction: "verifiedpermissions:ListPolicyStores"},
		{Name: "UpdatePolicyStore", Method: http.MethodPost, IAMAction: "verifiedpermissions:UpdatePolicyStore"},
		{Name: "DeletePolicyStore", Method: http.MethodPost, IAMAction: "verifiedpermissions:DeletePolicyStore"},
		{Name: "CreatePolicy", Method: http.MethodPost, IAMAction: "verifiedpermissions:CreatePolicy"},
		{Name: "GetPolicy", Method: http.MethodPost, IAMAction: "verifiedpermissions:GetPolicy"},
		{Name: "ListPolicies", Method: http.MethodPost, IAMAction: "verifiedpermissions:ListPolicies"},
		{Name: "UpdatePolicy", Method: http.MethodPost, IAMAction: "verifiedpermissions:UpdatePolicy"},
		{Name: "DeletePolicy", Method: http.MethodPost, IAMAction: "verifiedpermissions:DeletePolicy"},
		{Name: "PutSchema", Method: http.MethodPost, IAMAction: "verifiedpermissions:PutSchema"},
		{Name: "GetSchema", Method: http.MethodPost, IAMAction: "verifiedpermissions:GetSchema"},
		{Name: "IsAuthorized", Method: http.MethodPost, IAMAction: "verifiedpermissions:IsAuthorized"},
		{Name: "IsAuthorizedWithToken", Method: http.MethodPost, IAMAction: "verifiedpermissions:IsAuthorizedWithToken"},
		{Name: "CreatePolicyTemplate", Method: http.MethodPost, IAMAction: "verifiedpermissions:CreatePolicyTemplate"},
		{Name: "GetPolicyTemplate", Method: http.MethodPost, IAMAction: "verifiedpermissions:GetPolicyTemplate"},
		{Name: "ListPolicyTemplates", Method: http.MethodPost, IAMAction: "verifiedpermissions:ListPolicyTemplates"},
		{Name: "UpdatePolicyTemplate", Method: http.MethodPost, IAMAction: "verifiedpermissions:UpdatePolicyTemplate"},
		{Name: "DeletePolicyTemplate", Method: http.MethodPost, IAMAction: "verifiedpermissions:DeletePolicyTemplate"},
		{Name: "CreateIdentitySource", Method: http.MethodPost, IAMAction: "verifiedpermissions:CreateIdentitySource"},
		{Name: "GetIdentitySource", Method: http.MethodPost, IAMAction: "verifiedpermissions:GetIdentitySource"},
		{Name: "ListIdentitySources", Method: http.MethodPost, IAMAction: "verifiedpermissions:ListIdentitySources"},
		{Name: "DeleteIdentitySource", Method: http.MethodPost, IAMAction: "verifiedpermissions:DeleteIdentitySource"},
	}
}

// HealthCheck always returns nil.
func (s *VerifiedPermissionsService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Verified Permissions request to the appropriate handler.
func (s *VerifiedPermissionsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreatePolicyStore":
		return handleCreatePolicyStore(ctx, s.store)
	case "GetPolicyStore":
		return handleGetPolicyStore(ctx, s.store)
	case "ListPolicyStores":
		return handleListPolicyStores(ctx, s.store)
	case "UpdatePolicyStore":
		return handleUpdatePolicyStore(ctx, s.store)
	case "DeletePolicyStore":
		return handleDeletePolicyStore(ctx, s.store)
	case "CreatePolicy":
		return handleCreatePolicy(ctx, s.store)
	case "GetPolicy":
		return handleGetPolicy(ctx, s.store)
	case "ListPolicies":
		return handleListPolicies(ctx, s.store)
	case "UpdatePolicy":
		return handleUpdatePolicy(ctx, s.store)
	case "DeletePolicy":
		return handleDeletePolicy(ctx, s.store)
	case "PutSchema":
		return handlePutSchema(ctx, s.store)
	case "GetSchema":
		return handleGetSchema(ctx, s.store)
	case "IsAuthorized":
		return handleIsAuthorized(ctx, s.store)
	case "IsAuthorizedWithToken":
		return handleIsAuthorizedWithToken(ctx, s.store)
	case "CreatePolicyTemplate":
		return handleCreatePolicyTemplate(ctx, s.store)
	case "GetPolicyTemplate":
		return handleGetPolicyTemplate(ctx, s.store)
	case "ListPolicyTemplates":
		return handleListPolicyTemplates(ctx, s.store)
	case "UpdatePolicyTemplate":
		return handleUpdatePolicyTemplate(ctx, s.store)
	case "DeletePolicyTemplate":
		return handleDeletePolicyTemplate(ctx, s.store)
	case "CreateIdentitySource":
		return handleCreateIdentitySource(ctx, s.store)
	case "GetIdentitySource":
		return handleGetIdentitySource(ctx, s.store)
	case "ListIdentitySources":
		return handleListIdentitySources(ctx, s.store)
	case "DeleteIdentitySource":
		return handleDeleteIdentitySource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
