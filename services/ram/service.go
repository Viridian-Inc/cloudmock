package ram

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// RAMService is the cloudmock implementation of the AWS Resource Access Manager API.
type RAMService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new RAMService for the given AWS account ID and region.
func New(accountID, region string) *RAMService {
	return &RAMService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *RAMService) Name() string { return "ram" }

// Actions returns the list of RAM API actions supported by this service.
func (s *RAMService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateResourceShare", Method: http.MethodPost, IAMAction: "ram:CreateResourceShare"},
		{Name: "GetResourceShares", Method: http.MethodPost, IAMAction: "ram:GetResourceShares"},
		{Name: "UpdateResourceShare", Method: http.MethodPost, IAMAction: "ram:UpdateResourceShare"},
		{Name: "DeleteResourceShare", Method: http.MethodPost, IAMAction: "ram:DeleteResourceShare"},
		{Name: "AssociateResourceShare", Method: http.MethodPost, IAMAction: "ram:AssociateResourceShare"},
		{Name: "DisassociateResourceShare", Method: http.MethodPost, IAMAction: "ram:DisassociateResourceShare"},
		{Name: "GetResourceShareAssociations", Method: http.MethodPost, IAMAction: "ram:GetResourceShareAssociations"},
		{Name: "GetResourceShareInvitations", Method: http.MethodPost, IAMAction: "ram:GetResourceShareInvitations"},
		{Name: "AcceptResourceShareInvitation", Method: http.MethodPost, IAMAction: "ram:AcceptResourceShareInvitation"},
		{Name: "RejectResourceShareInvitation", Method: http.MethodPost, IAMAction: "ram:RejectResourceShareInvitation"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "ram:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "ram:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "ram:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *RAMService) HealthCheck() error { return nil }

// HandleRequest routes an incoming RAM request to the appropriate handler.
func (s *RAMService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateResourceShare":
		return handleCreateResourceShare(ctx, s.store)
	case "GetResourceShares":
		return handleGetResourceShares(ctx, s.store)
	case "UpdateResourceShare":
		return handleUpdateResourceShare(ctx, s.store)
	case "DeleteResourceShare":
		return handleDeleteResourceShare(ctx, s.store)
	case "AssociateResourceShare":
		return handleAssociateResourceShare(ctx, s.store)
	case "DisassociateResourceShare":
		return handleDisassociateResourceShare(ctx, s.store)
	case "GetResourceShareAssociations":
		return handleGetResourceShareAssociations(ctx, s.store)
	case "GetResourceShareInvitations":
		return handleGetResourceShareInvitations(ctx, s.store)
	case "AcceptResourceShareInvitation":
		return handleAcceptResourceShareInvitation(ctx, s.store)
	case "RejectResourceShareInvitation":
		return handleRejectResourceShareInvitation(ctx, s.store)
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
