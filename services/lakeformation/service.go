package lakeformation

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// LakeFormationService is the cloudmock implementation of the AWS Lake Formation API.
type LakeFormationService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new LakeFormationService.
func New(accountID, region string) *LakeFormationService {
	return &LakeFormationService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *LakeFormationService) Name() string { return "lakeformation" }

// Actions returns the list of API actions supported.
func (s *LakeFormationService) Actions() []service.Action {
	return []service.Action{
		{Name: "RegisterResource", Method: http.MethodPost, IAMAction: "lakeformation:RegisterResource"},
		{Name: "DeregisterResource", Method: http.MethodPost, IAMAction: "lakeformation:DeregisterResource"},
		{Name: "ListResources", Method: http.MethodPost, IAMAction: "lakeformation:ListResources"},
		{Name: "GrantPermissions", Method: http.MethodPost, IAMAction: "lakeformation:GrantPermissions"},
		{Name: "RevokePermissions", Method: http.MethodPost, IAMAction: "lakeformation:RevokePermissions"},
		{Name: "GetEffectivePermissionsForPath", Method: http.MethodPost, IAMAction: "lakeformation:GetEffectivePermissionsForPath"},
		{Name: "ListPermissions", Method: http.MethodPost, IAMAction: "lakeformation:ListPermissions"},
		{Name: "BatchGrantPermissions", Method: http.MethodPost, IAMAction: "lakeformation:BatchGrantPermissions"},
		{Name: "BatchRevokePermissions", Method: http.MethodPost, IAMAction: "lakeformation:BatchRevokePermissions"},
		{Name: "DescribeResource", Method: http.MethodPost, IAMAction: "lakeformation:DescribeResource"},
		{Name: "GetDataLakeSettings", Method: http.MethodPost, IAMAction: "lakeformation:GetDataLakeSettings"},
		{Name: "PutDataLakeSettings", Method: http.MethodPost, IAMAction: "lakeformation:PutDataLakeSettings"},
		{Name: "AddLFTagsToResource", Method: http.MethodPost, IAMAction: "lakeformation:AddLFTagsToResource"},
		{Name: "RemoveLFTagsFromResource", Method: http.MethodPost, IAMAction: "lakeformation:RemoveLFTagsFromResource"},
		{Name: "GetResourceLFTags", Method: http.MethodPost, IAMAction: "lakeformation:GetResourceLFTags"},
		{Name: "CreateLFTag", Method: http.MethodPost, IAMAction: "lakeformation:CreateLFTag"},
		{Name: "GetLFTag", Method: http.MethodPost, IAMAction: "lakeformation:GetLFTag"},
		{Name: "ListLFTags", Method: http.MethodPost, IAMAction: "lakeformation:ListLFTags"},
		{Name: "DeleteLFTag", Method: http.MethodPost, IAMAction: "lakeformation:DeleteLFTag"},
		{Name: "UpdateLFTag", Method: http.MethodPost, IAMAction: "lakeformation:UpdateLFTag"},
	}
}

// HealthCheck always returns nil.
func (s *LakeFormationService) HealthCheck() error { return nil }

// HandleRequest routes an incoming request to the appropriate handler.
func (s *LakeFormationService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "RegisterResource":
		return handleRegisterResource(ctx, s.store)
	case "DeregisterResource":
		return handleDeregisterResource(ctx, s.store)
	case "ListResources":
		return handleListResources(ctx, s.store)
	case "GrantPermissions":
		return handleGrantPermissions(ctx, s.store)
	case "RevokePermissions":
		return handleRevokePermissions(ctx, s.store)
	case "GetEffectivePermissionsForPath":
		return handleGetEffectivePermissionsForPath(ctx, s.store)
	case "ListPermissions":
		return handleListPermissions(ctx, s.store)
	case "BatchGrantPermissions":
		return handleBatchGrantPermissions(ctx, s.store)
	case "BatchRevokePermissions":
		return handleBatchRevokePermissions(ctx, s.store)
	case "DescribeResource":
		return handleDescribeResource(ctx, s.store)
	case "GetDataLakeSettings":
		return handleGetDataLakeSettings(ctx, s.store)
	case "PutDataLakeSettings":
		return handlePutDataLakeSettings(ctx, s.store)
	case "AddLFTagsToResource":
		return handleAddLFTagsToResource(ctx, s.store)
	case "RemoveLFTagsFromResource":
		return handleRemoveLFTagsFromResource(ctx, s.store)
	case "GetResourceLFTags":
		return handleGetResourceLFTags(ctx, s.store)
	case "CreateLFTag":
		return handleCreateLFTag(ctx, s.store)
	case "GetLFTag":
		return handleGetLFTag(ctx, s.store)
	case "ListLFTags":
		return handleListLFTags(ctx, s.store)
	case "DeleteLFTag":
		return handleDeleteLFTag(ctx, s.store)
	case "UpdateLFTag":
		return handleUpdateLFTag(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
