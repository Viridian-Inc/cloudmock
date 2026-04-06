package globalaccelerator

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS globalaccelerator service.
type Service struct {
	store *Store
}

// New returns a new globalaccelerator Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "globalaccelerator" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "AddCustomRoutingEndpoints", Method: http.MethodPost, IAMAction: "globalaccelerator:AddCustomRoutingEndpoints"},
		{Name: "AddEndpoints", Method: http.MethodPost, IAMAction: "globalaccelerator:AddEndpoints"},
		{Name: "AdvertiseByoipCidr", Method: http.MethodPost, IAMAction: "globalaccelerator:AdvertiseByoipCidr"},
		{Name: "AllowCustomRoutingTraffic", Method: http.MethodPost, IAMAction: "globalaccelerator:AllowCustomRoutingTraffic"},
		{Name: "CreateAccelerator", Method: http.MethodPost, IAMAction: "globalaccelerator:CreateAccelerator"},
		{Name: "CreateCrossAccountAttachment", Method: http.MethodPost, IAMAction: "globalaccelerator:CreateCrossAccountAttachment"},
		{Name: "CreateCustomRoutingAccelerator", Method: http.MethodPost, IAMAction: "globalaccelerator:CreateCustomRoutingAccelerator"},
		{Name: "CreateCustomRoutingEndpointGroup", Method: http.MethodPost, IAMAction: "globalaccelerator:CreateCustomRoutingEndpointGroup"},
		{Name: "CreateCustomRoutingListener", Method: http.MethodPost, IAMAction: "globalaccelerator:CreateCustomRoutingListener"},
		{Name: "CreateEndpointGroup", Method: http.MethodPost, IAMAction: "globalaccelerator:CreateEndpointGroup"},
		{Name: "CreateListener", Method: http.MethodPost, IAMAction: "globalaccelerator:CreateListener"},
		{Name: "DeleteAccelerator", Method: http.MethodPost, IAMAction: "globalaccelerator:DeleteAccelerator"},
		{Name: "DeleteCrossAccountAttachment", Method: http.MethodPost, IAMAction: "globalaccelerator:DeleteCrossAccountAttachment"},
		{Name: "DeleteCustomRoutingAccelerator", Method: http.MethodPost, IAMAction: "globalaccelerator:DeleteCustomRoutingAccelerator"},
		{Name: "DeleteCustomRoutingEndpointGroup", Method: http.MethodPost, IAMAction: "globalaccelerator:DeleteCustomRoutingEndpointGroup"},
		{Name: "DeleteCustomRoutingListener", Method: http.MethodPost, IAMAction: "globalaccelerator:DeleteCustomRoutingListener"},
		{Name: "DeleteEndpointGroup", Method: http.MethodPost, IAMAction: "globalaccelerator:DeleteEndpointGroup"},
		{Name: "DeleteListener", Method: http.MethodPost, IAMAction: "globalaccelerator:DeleteListener"},
		{Name: "DenyCustomRoutingTraffic", Method: http.MethodPost, IAMAction: "globalaccelerator:DenyCustomRoutingTraffic"},
		{Name: "DeprovisionByoipCidr", Method: http.MethodPost, IAMAction: "globalaccelerator:DeprovisionByoipCidr"},
		{Name: "DescribeAccelerator", Method: http.MethodPost, IAMAction: "globalaccelerator:DescribeAccelerator"},
		{Name: "DescribeAcceleratorAttributes", Method: http.MethodPost, IAMAction: "globalaccelerator:DescribeAcceleratorAttributes"},
		{Name: "DescribeCrossAccountAttachment", Method: http.MethodPost, IAMAction: "globalaccelerator:DescribeCrossAccountAttachment"},
		{Name: "DescribeCustomRoutingAccelerator", Method: http.MethodPost, IAMAction: "globalaccelerator:DescribeCustomRoutingAccelerator"},
		{Name: "DescribeCustomRoutingAcceleratorAttributes", Method: http.MethodPost, IAMAction: "globalaccelerator:DescribeCustomRoutingAcceleratorAttributes"},
		{Name: "DescribeCustomRoutingEndpointGroup", Method: http.MethodPost, IAMAction: "globalaccelerator:DescribeCustomRoutingEndpointGroup"},
		{Name: "DescribeCustomRoutingListener", Method: http.MethodPost, IAMAction: "globalaccelerator:DescribeCustomRoutingListener"},
		{Name: "DescribeEndpointGroup", Method: http.MethodPost, IAMAction: "globalaccelerator:DescribeEndpointGroup"},
		{Name: "DescribeListener", Method: http.MethodPost, IAMAction: "globalaccelerator:DescribeListener"},
		{Name: "ListAccelerators", Method: http.MethodPost, IAMAction: "globalaccelerator:ListAccelerators"},
		{Name: "ListByoipCidrs", Method: http.MethodPost, IAMAction: "globalaccelerator:ListByoipCidrs"},
		{Name: "ListCrossAccountAttachments", Method: http.MethodPost, IAMAction: "globalaccelerator:ListCrossAccountAttachments"},
		{Name: "ListCrossAccountResourceAccounts", Method: http.MethodPost, IAMAction: "globalaccelerator:ListCrossAccountResourceAccounts"},
		{Name: "ListCrossAccountResources", Method: http.MethodPost, IAMAction: "globalaccelerator:ListCrossAccountResources"},
		{Name: "ListCustomRoutingAccelerators", Method: http.MethodPost, IAMAction: "globalaccelerator:ListCustomRoutingAccelerators"},
		{Name: "ListCustomRoutingEndpointGroups", Method: http.MethodPost, IAMAction: "globalaccelerator:ListCustomRoutingEndpointGroups"},
		{Name: "ListCustomRoutingListeners", Method: http.MethodPost, IAMAction: "globalaccelerator:ListCustomRoutingListeners"},
		{Name: "ListCustomRoutingPortMappings", Method: http.MethodPost, IAMAction: "globalaccelerator:ListCustomRoutingPortMappings"},
		{Name: "ListCustomRoutingPortMappingsByDestination", Method: http.MethodPost, IAMAction: "globalaccelerator:ListCustomRoutingPortMappingsByDestination"},
		{Name: "ListEndpointGroups", Method: http.MethodPost, IAMAction: "globalaccelerator:ListEndpointGroups"},
		{Name: "ListListeners", Method: http.MethodPost, IAMAction: "globalaccelerator:ListListeners"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "globalaccelerator:ListTagsForResource"},
		{Name: "ProvisionByoipCidr", Method: http.MethodPost, IAMAction: "globalaccelerator:ProvisionByoipCidr"},
		{Name: "RemoveCustomRoutingEndpoints", Method: http.MethodPost, IAMAction: "globalaccelerator:RemoveCustomRoutingEndpoints"},
		{Name: "RemoveEndpoints", Method: http.MethodPost, IAMAction: "globalaccelerator:RemoveEndpoints"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "globalaccelerator:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "globalaccelerator:UntagResource"},
		{Name: "UpdateAccelerator", Method: http.MethodPost, IAMAction: "globalaccelerator:UpdateAccelerator"},
		{Name: "UpdateAcceleratorAttributes", Method: http.MethodPost, IAMAction: "globalaccelerator:UpdateAcceleratorAttributes"},
		{Name: "UpdateCrossAccountAttachment", Method: http.MethodPost, IAMAction: "globalaccelerator:UpdateCrossAccountAttachment"},
		{Name: "UpdateCustomRoutingAccelerator", Method: http.MethodPost, IAMAction: "globalaccelerator:UpdateCustomRoutingAccelerator"},
		{Name: "UpdateCustomRoutingAcceleratorAttributes", Method: http.MethodPost, IAMAction: "globalaccelerator:UpdateCustomRoutingAcceleratorAttributes"},
		{Name: "UpdateCustomRoutingListener", Method: http.MethodPost, IAMAction: "globalaccelerator:UpdateCustomRoutingListener"},
		{Name: "UpdateEndpointGroup", Method: http.MethodPost, IAMAction: "globalaccelerator:UpdateEndpointGroup"},
		{Name: "UpdateListener", Method: http.MethodPost, IAMAction: "globalaccelerator:UpdateListener"},
		{Name: "WithdrawByoipCidr", Method: http.MethodPost, IAMAction: "globalaccelerator:WithdrawByoipCidr"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "AddCustomRoutingEndpoints":
		return handleAddCustomRoutingEndpoints(ctx, s.store)
	case "AddEndpoints":
		return handleAddEndpoints(ctx, s.store)
	case "AdvertiseByoipCidr":
		return handleAdvertiseByoipCidr(ctx, s.store)
	case "AllowCustomRoutingTraffic":
		return handleAllowCustomRoutingTraffic(ctx, s.store)
	case "CreateAccelerator":
		return handleCreateAccelerator(ctx, s.store)
	case "CreateCrossAccountAttachment":
		return handleCreateCrossAccountAttachment(ctx, s.store)
	case "CreateCustomRoutingAccelerator":
		return handleCreateCustomRoutingAccelerator(ctx, s.store)
	case "CreateCustomRoutingEndpointGroup":
		return handleCreateCustomRoutingEndpointGroup(ctx, s.store)
	case "CreateCustomRoutingListener":
		return handleCreateCustomRoutingListener(ctx, s.store)
	case "CreateEndpointGroup":
		return handleCreateEndpointGroup(ctx, s.store)
	case "CreateListener":
		return handleCreateListener(ctx, s.store)
	case "DeleteAccelerator":
		return handleDeleteAccelerator(ctx, s.store)
	case "DeleteCrossAccountAttachment":
		return handleDeleteCrossAccountAttachment(ctx, s.store)
	case "DeleteCustomRoutingAccelerator":
		return handleDeleteCustomRoutingAccelerator(ctx, s.store)
	case "DeleteCustomRoutingEndpointGroup":
		return handleDeleteCustomRoutingEndpointGroup(ctx, s.store)
	case "DeleteCustomRoutingListener":
		return handleDeleteCustomRoutingListener(ctx, s.store)
	case "DeleteEndpointGroup":
		return handleDeleteEndpointGroup(ctx, s.store)
	case "DeleteListener":
		return handleDeleteListener(ctx, s.store)
	case "DenyCustomRoutingTraffic":
		return handleDenyCustomRoutingTraffic(ctx, s.store)
	case "DeprovisionByoipCidr":
		return handleDeprovisionByoipCidr(ctx, s.store)
	case "DescribeAccelerator":
		return handleDescribeAccelerator(ctx, s.store)
	case "DescribeAcceleratorAttributes":
		return handleDescribeAcceleratorAttributes(ctx, s.store)
	case "DescribeCrossAccountAttachment":
		return handleDescribeCrossAccountAttachment(ctx, s.store)
	case "DescribeCustomRoutingAccelerator":
		return handleDescribeCustomRoutingAccelerator(ctx, s.store)
	case "DescribeCustomRoutingAcceleratorAttributes":
		return handleDescribeCustomRoutingAcceleratorAttributes(ctx, s.store)
	case "DescribeCustomRoutingEndpointGroup":
		return handleDescribeCustomRoutingEndpointGroup(ctx, s.store)
	case "DescribeCustomRoutingListener":
		return handleDescribeCustomRoutingListener(ctx, s.store)
	case "DescribeEndpointGroup":
		return handleDescribeEndpointGroup(ctx, s.store)
	case "DescribeListener":
		return handleDescribeListener(ctx, s.store)
	case "ListAccelerators":
		return handleListAccelerators(ctx, s.store)
	case "ListByoipCidrs":
		return handleListByoipCidrs(ctx, s.store)
	case "ListCrossAccountAttachments":
		return handleListCrossAccountAttachments(ctx, s.store)
	case "ListCrossAccountResourceAccounts":
		return handleListCrossAccountResourceAccounts(ctx, s.store)
	case "ListCrossAccountResources":
		return handleListCrossAccountResources(ctx, s.store)
	case "ListCustomRoutingAccelerators":
		return handleListCustomRoutingAccelerators(ctx, s.store)
	case "ListCustomRoutingEndpointGroups":
		return handleListCustomRoutingEndpointGroups(ctx, s.store)
	case "ListCustomRoutingListeners":
		return handleListCustomRoutingListeners(ctx, s.store)
	case "ListCustomRoutingPortMappings":
		return handleListCustomRoutingPortMappings(ctx, s.store)
	case "ListCustomRoutingPortMappingsByDestination":
		return handleListCustomRoutingPortMappingsByDestination(ctx, s.store)
	case "ListEndpointGroups":
		return handleListEndpointGroups(ctx, s.store)
	case "ListListeners":
		return handleListListeners(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "ProvisionByoipCidr":
		return handleProvisionByoipCidr(ctx, s.store)
	case "RemoveCustomRoutingEndpoints":
		return handleRemoveCustomRoutingEndpoints(ctx, s.store)
	case "RemoveEndpoints":
		return handleRemoveEndpoints(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateAccelerator":
		return handleUpdateAccelerator(ctx, s.store)
	case "UpdateAcceleratorAttributes":
		return handleUpdateAcceleratorAttributes(ctx, s.store)
	case "UpdateCrossAccountAttachment":
		return handleUpdateCrossAccountAttachment(ctx, s.store)
	case "UpdateCustomRoutingAccelerator":
		return handleUpdateCustomRoutingAccelerator(ctx, s.store)
	case "UpdateCustomRoutingAcceleratorAttributes":
		return handleUpdateCustomRoutingAcceleratorAttributes(ctx, s.store)
	case "UpdateCustomRoutingListener":
		return handleUpdateCustomRoutingListener(ctx, s.store)
	case "UpdateEndpointGroup":
		return handleUpdateEndpointGroup(ctx, s.store)
	case "UpdateListener":
		return handleUpdateListener(ctx, s.store)
	case "WithdrawByoipCidr":
		return handleWithdrawByoipCidr(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
