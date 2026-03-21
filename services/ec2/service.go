package ec2

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// EC2Service is the cloudmock implementation of the AWS EC2 API.
type EC2Service struct {
	store *Store
}

// New returns a new EC2Service for the given AWS account ID and region.
func New(accountID, region string) *EC2Service {
	return &EC2Service{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *EC2Service) Name() string { return "ec2" }

// Actions returns the list of EC2 API actions supported by this service.
func (s *EC2Service) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateVpc", Method: http.MethodPost, IAMAction: "ec2:CreateVpc"},
		{Name: "DescribeVpcs", Method: http.MethodPost, IAMAction: "ec2:DescribeVpcs"},
		{Name: "DeleteVpc", Method: http.MethodPost, IAMAction: "ec2:DeleteVpc"},
		{Name: "ModifyVpcAttribute", Method: http.MethodPost, IAMAction: "ec2:ModifyVpcAttribute"},
		{Name: "CreateSubnet", Method: http.MethodPost, IAMAction: "ec2:CreateSubnet"},
		{Name: "DescribeSubnets", Method: http.MethodPost, IAMAction: "ec2:DescribeSubnets"},
		{Name: "DeleteSubnet", Method: http.MethodPost, IAMAction: "ec2:DeleteSubnet"},
		{Name: "CreateSecurityGroup", Method: http.MethodPost, IAMAction: "ec2:CreateSecurityGroup"},
		{Name: "DescribeSecurityGroups", Method: http.MethodPost, IAMAction: "ec2:DescribeSecurityGroups"},
		{Name: "DeleteSecurityGroup", Method: http.MethodPost, IAMAction: "ec2:DeleteSecurityGroup"},
		{Name: "AuthorizeSecurityGroupIngress", Method: http.MethodPost, IAMAction: "ec2:AuthorizeSecurityGroupIngress"},
		{Name: "AuthorizeSecurityGroupEgress", Method: http.MethodPost, IAMAction: "ec2:AuthorizeSecurityGroupEgress"},
		{Name: "RevokeSecurityGroupIngress", Method: http.MethodPost, IAMAction: "ec2:RevokeSecurityGroupIngress"},
		{Name: "RevokeSecurityGroupEgress", Method: http.MethodPost, IAMAction: "ec2:RevokeSecurityGroupEgress"},
		// Internet Gateway
		{Name: "CreateInternetGateway", Method: http.MethodPost, IAMAction: "ec2:CreateInternetGateway"},
		{Name: "AttachInternetGateway", Method: http.MethodPost, IAMAction: "ec2:AttachInternetGateway"},
		{Name: "DetachInternetGateway", Method: http.MethodPost, IAMAction: "ec2:DetachInternetGateway"},
		{Name: "DeleteInternetGateway", Method: http.MethodPost, IAMAction: "ec2:DeleteInternetGateway"},
		{Name: "DescribeInternetGateways", Method: http.MethodPost, IAMAction: "ec2:DescribeInternetGateways"},
		// NAT Gateway
		{Name: "CreateNatGateway", Method: http.MethodPost, IAMAction: "ec2:CreateNatGateway"},
		{Name: "DescribeNatGateways", Method: http.MethodPost, IAMAction: "ec2:DescribeNatGateways"},
		{Name: "DeleteNatGateway", Method: http.MethodPost, IAMAction: "ec2:DeleteNatGateway"},
		// Route Table
		{Name: "CreateRouteTable", Method: http.MethodPost, IAMAction: "ec2:CreateRouteTable"},
		{Name: "DescribeRouteTables", Method: http.MethodPost, IAMAction: "ec2:DescribeRouteTables"},
		{Name: "DeleteRouteTable", Method: http.MethodPost, IAMAction: "ec2:DeleteRouteTable"},
		{Name: "CreateRoute", Method: http.MethodPost, IAMAction: "ec2:CreateRoute"},
		{Name: "DeleteRoute", Method: http.MethodPost, IAMAction: "ec2:DeleteRoute"},
		{Name: "ReplaceRoute", Method: http.MethodPost, IAMAction: "ec2:ReplaceRoute"},
		{Name: "AssociateRouteTable", Method: http.MethodPost, IAMAction: "ec2:AssociateRouteTable"},
		{Name: "DisassociateRouteTable", Method: http.MethodPost, IAMAction: "ec2:DisassociateRouteTable"},
		// Elastic IP
		{Name: "AllocateAddress", Method: http.MethodPost, IAMAction: "ec2:AllocateAddress"},
		{Name: "ReleaseAddress", Method: http.MethodPost, IAMAction: "ec2:ReleaseAddress"},
		{Name: "AssociateAddress", Method: http.MethodPost, IAMAction: "ec2:AssociateAddress"},
		{Name: "DisassociateAddress", Method: http.MethodPost, IAMAction: "ec2:DisassociateAddress"},
		{Name: "DescribeAddresses", Method: http.MethodPost, IAMAction: "ec2:DescribeAddresses"},
		// Network Interface
		{Name: "CreateNetworkInterface", Method: http.MethodPost, IAMAction: "ec2:CreateNetworkInterface"},
		{Name: "DescribeNetworkInterfaces", Method: http.MethodPost, IAMAction: "ec2:DescribeNetworkInterfaces"},
		{Name: "DeleteNetworkInterface", Method: http.MethodPost, IAMAction: "ec2:DeleteNetworkInterface"},
		// Network ACL
		{Name: "CreateNetworkAcl", Method: http.MethodPost, IAMAction: "ec2:CreateNetworkAcl"},
		{Name: "DescribeNetworkAcls", Method: http.MethodPost, IAMAction: "ec2:DescribeNetworkAcls"},
		{Name: "DeleteNetworkAcl", Method: http.MethodPost, IAMAction: "ec2:DeleteNetworkAcl"},
		{Name: "CreateNetworkAclEntry", Method: http.MethodPost, IAMAction: "ec2:CreateNetworkAclEntry"},
		{Name: "DeleteNetworkAclEntry", Method: http.MethodPost, IAMAction: "ec2:DeleteNetworkAclEntry"},
		// VPC Endpoint
		{Name: "CreateVpcEndpoint", Method: http.MethodPost, IAMAction: "ec2:CreateVpcEndpoint"},
		{Name: "DescribeVpcEndpoints", Method: http.MethodPost, IAMAction: "ec2:DescribeVpcEndpoints"},
		{Name: "DeleteVpcEndpoints", Method: http.MethodPost, IAMAction: "ec2:DeleteVpcEndpoints"},
		// VPC Peering
		{Name: "CreateVpcPeeringConnection", Method: http.MethodPost, IAMAction: "ec2:CreateVpcPeeringConnection"},
		{Name: "AcceptVpcPeeringConnection", Method: http.MethodPost, IAMAction: "ec2:AcceptVpcPeeringConnection"},
		{Name: "DeleteVpcPeeringConnection", Method: http.MethodPost, IAMAction: "ec2:DeleteVpcPeeringConnection"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *EC2Service) HealthCheck() error { return nil }

// HandleRequest routes an incoming EC2 request to the appropriate handler.
// EC2 uses form-encoded POST bodies; the Action may appear in the query string
// (already parsed into ctx.Params) or in the form-encoded body.
func (s *EC2Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "CreateVpc":
		return handleCreateVpc(ctx, s.store)
	case "DescribeVpcs":
		return handleDescribeVpcs(ctx, s.store)
	case "DeleteVpc":
		return handleDeleteVpc(ctx, s.store)
	case "ModifyVpcAttribute":
		return handleModifyVpcAttribute(ctx, s.store)
	case "CreateSubnet":
		return handleCreateSubnet(ctx, s.store)
	case "DescribeSubnets":
		return handleDescribeSubnets(ctx, s.store)
	case "DeleteSubnet":
		return handleDeleteSubnet(ctx, s.store)
	case "CreateSecurityGroup":
		return handleCreateSecurityGroup(ctx, s.store)
	case "DescribeSecurityGroups":
		return handleDescribeSecurityGroups(ctx, s.store)
	case "DeleteSecurityGroup":
		return handleDeleteSecurityGroup(ctx, s.store)
	case "AuthorizeSecurityGroupIngress":
		return handleAuthorizeSecurityGroupIngress(ctx, s.store)
	case "AuthorizeSecurityGroupEgress":
		return handleAuthorizeSecurityGroupEgress(ctx, s.store)
	case "RevokeSecurityGroupIngress":
		return handleRevokeSecurityGroupIngress(ctx, s.store)
	case "RevokeSecurityGroupEgress":
		return handleRevokeSecurityGroupEgress(ctx, s.store)
	// Internet Gateway
	case "CreateInternetGateway":
		return handleCreateInternetGateway(ctx, s.store)
	case "AttachInternetGateway":
		return handleAttachInternetGateway(ctx, s.store)
	case "DetachInternetGateway":
		return handleDetachInternetGateway(ctx, s.store)
	case "DeleteInternetGateway":
		return handleDeleteInternetGateway(ctx, s.store)
	case "DescribeInternetGateways":
		return handleDescribeInternetGateways(ctx, s.store)
	// NAT Gateway
	case "CreateNatGateway":
		return handleCreateNatGateway(ctx, s.store)
	case "DescribeNatGateways":
		return handleDescribeNatGateways(ctx, s.store)
	case "DeleteNatGateway":
		return handleDeleteNatGateway(ctx, s.store)
	// Route Table
	case "CreateRouteTable":
		return handleCreateRouteTable(ctx, s.store)
	case "DescribeRouteTables":
		return handleDescribeRouteTables(ctx, s.store)
	case "DeleteRouteTable":
		return handleDeleteRouteTable(ctx, s.store)
	case "CreateRoute":
		return handleCreateRoute(ctx, s.store)
	case "DeleteRoute":
		return handleDeleteRoute(ctx, s.store)
	case "ReplaceRoute":
		return handleReplaceRoute(ctx, s.store)
	case "AssociateRouteTable":
		return handleAssociateRouteTable(ctx, s.store)
	case "DisassociateRouteTable":
		return handleDisassociateRouteTable(ctx, s.store)
	// Elastic IP
	case "AllocateAddress":
		return handleAllocateAddress(ctx, s.store)
	case "ReleaseAddress":
		return handleReleaseAddress(ctx, s.store)
	case "AssociateAddress":
		return handleAssociateAddress(ctx, s.store)
	case "DisassociateAddress":
		return handleDisassociateAddress(ctx, s.store)
	case "DescribeAddresses":
		return handleDescribeAddresses(ctx, s.store)
	// Network Interface
	case "CreateNetworkInterface":
		return handleCreateNetworkInterface(ctx, s.store)
	case "DescribeNetworkInterfaces":
		return handleDescribeNetworkInterfaces(ctx, s.store)
	case "DeleteNetworkInterface":
		return handleDeleteNetworkInterface(ctx, s.store)
	// Network ACL
	case "CreateNetworkAcl":
		return handleCreateNetworkAcl(ctx, s.store)
	case "DescribeNetworkAcls":
		return handleDescribeNetworkAcls(ctx, s.store)
	case "DeleteNetworkAcl":
		return handleDeleteNetworkAcl(ctx, s.store)
	case "CreateNetworkAclEntry":
		return handleCreateNetworkAclEntry(ctx, s.store)
	case "DeleteNetworkAclEntry":
		return handleDeleteNetworkAclEntry(ctx, s.store)
	// VPC Endpoint
	case "CreateVpcEndpoint":
		return handleCreateVpcEndpoint(ctx, s.store)
	case "DescribeVpcEndpoints":
		return handleDescribeVpcEndpoints(ctx, s.store)
	case "DeleteVpcEndpoints":
		return handleDeleteVpcEndpoints(ctx, s.store)
	// VPC Peering
	case "CreateVpcPeeringConnection":
		return handleCreateVpcPeeringConnection(ctx, s.store)
	case "AcceptVpcPeeringConnection":
		return handleAcceptVpcPeeringConnection(ctx, s.store)
	case "DeleteVpcPeeringConnection":
		return handleDeleteVpcPeeringConnection(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
