package ec2

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/schema"
	"github.com/neureaux/cloudmock/pkg/service"
)

// EC2Service is the cloudmock implementation of the AWS EC2 API.
type EC2Service struct {
	store *Store
}

// New returns a new EC2Service for the given AWS account ID and region.
// It automatically creates the default VPC (172.31.0.0/16) with 3 subnets,
// a default route table, security group, NACL, and an attached internet gateway.
func New(accountID, region string) *EC2Service {
	svc := &EC2Service{
		store: NewStore(accountID, region),
	}
	svc.createDefaultVPC(region)
	return svc
}

// createDefaultVPC seeds the store with the AWS default VPC resources.
func (s *EC2Service) createDefaultVPC(region string) {
	// Create the default VPC (172.31.0.0/16).
	vpc, err := s.store.CreateVPC("172.31.0.0/16", true, true)
	if err != nil {
		return
	}
	vpc.IsDefault = true

	// Create the internet gateway and attach it to the default VPC.
	igw := s.store.CreateInternetGateway()
	s.store.AttachInternetGateway(igw.IgwId, vpc.VpcId)

	// Add a default route (0.0.0.0/0 → IGW) to the main route table.
	rts := s.store.ListRouteTables(nil, vpc.VpcId)
	if len(rts) > 0 {
		s.store.CreateRoute(rts[0].RouteTableId, "0.0.0.0/0", igw.IgwId, "", "")
	}

	// Create 3 default subnets across availability zones.
	azSubnets := []struct {
		cidr string
		az   string
	}{
		{"172.31.0.0/20", region + "a"},
		{"172.31.16.0/20", region + "b"},
		{"172.31.32.0/20", region + "c"},
	}
	for _, s2 := range azSubnets {
		sub, _ := s.store.CreateSubnet(vpc.VpcId, s2.cidr, s2.az)
		if sub != nil {
			sub.MapPublicIpOnLaunch = true
		}
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
		// Instances
		{Name: "RunInstances", Method: http.MethodPost, IAMAction: "ec2:RunInstances"},
		{Name: "DescribeInstances", Method: http.MethodPost, IAMAction: "ec2:DescribeInstances"},
		{Name: "TerminateInstances", Method: http.MethodPost, IAMAction: "ec2:TerminateInstances"},
		{Name: "StopInstances", Method: http.MethodPost, IAMAction: "ec2:StopInstances"},
		{Name: "StartInstances", Method: http.MethodPost, IAMAction: "ec2:StartInstances"},
		{Name: "DescribeInstanceStatus", Method: http.MethodPost, IAMAction: "ec2:DescribeInstanceStatus"},
		// Tagging
		{Name: "CreateTags", Method: http.MethodPost, IAMAction: "ec2:CreateTags"},
		{Name: "DeleteTags", Method: http.MethodPost, IAMAction: "ec2:DeleteTags"},
		{Name: "DescribeTags", Method: http.MethodPost, IAMAction: "ec2:DescribeTags"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *EC2Service) HealthCheck() error { return nil }

// ResourceSchemas returns schemas for EC2 VPC, Subnet, SecurityGroup, and Instance resources.
func (s *EC2Service) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "ec2",
			ResourceType:  "aws_vpc",
			TerraformType: "cloudmock_ec2_vpc",
			AWSType:       "AWS::EC2::VPC",
			CreateAction:  "CreateVpc",
			ReadAction:    "DescribeVpcs",
			DeleteAction:  "DeleteVpc",
			UpdateAction:  "ModifyVpcAttribute",
			ImportID:      "vpc_id",
			Attributes: []schema.AttributeSchema{
				{Name: "vpc_id", Type: "string", Computed: true},
				{Name: "cidr_block", Type: "string", Required: true, ForceNew: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "enable_dns_support", Type: "bool", Default: true},
				{Name: "enable_dns_hostnames", Type: "bool", Default: false},
				{Name: "is_default", Type: "bool", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "ec2",
			ResourceType:  "aws_subnet",
			TerraformType: "cloudmock_ec2_subnet",
			AWSType:       "AWS::EC2::Subnet",
			CreateAction:  "CreateSubnet",
			ReadAction:    "DescribeSubnets",
			DeleteAction:  "DeleteSubnet",
			ImportID:      "subnet_id",
			Attributes: []schema.AttributeSchema{
				{Name: "subnet_id", Type: "string", Computed: true},
				{Name: "vpc_id", Type: "string", Required: true, ForceNew: true, RefTo: "cloudmock_ec2_vpc.vpc_id"},
				{Name: "cidr_block", Type: "string", Required: true, ForceNew: true},
				{Name: "availability_zone", Type: "string", ForceNew: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "map_public_ip_on_launch", Type: "bool", Default: false},
				{Name: "tags", Type: "map"},
			},
			References: []schema.ResourceRef{
				{FromAttr: "vpc_id", ToResource: "cloudmock_ec2_vpc", ToAttr: "vpc_id"},
			},
		},
		{
			ServiceName:   "ec2",
			ResourceType:  "aws_security_group",
			TerraformType: "cloudmock_ec2_security_group",
			AWSType:       "AWS::EC2::SecurityGroup",
			CreateAction:  "CreateSecurityGroup",
			ReadAction:    "DescribeSecurityGroups",
			DeleteAction:  "DeleteSecurityGroup",
			ImportID:      "security_group_id",
			Attributes: []schema.AttributeSchema{
				{Name: "security_group_id", Type: "string", Computed: true},
				{Name: "group_name", Type: "string", Required: true, ForceNew: true},
				{Name: "description", Type: "string", Required: true, ForceNew: true},
				{Name: "vpc_id", Type: "string", ForceNew: true, RefTo: "cloudmock_ec2_vpc.vpc_id"},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "ingress", Type: "set"},
				{Name: "egress", Type: "set"},
				{Name: "tags", Type: "map"},
			},
			References: []schema.ResourceRef{
				{FromAttr: "vpc_id", ToResource: "cloudmock_ec2_vpc", ToAttr: "vpc_id"},
			},
		},
		{
			ServiceName:   "ec2",
			ResourceType:  "aws_instance",
			TerraformType: "cloudmock_ec2_instance",
			AWSType:       "AWS::EC2::Instance",
			CreateAction:  "RunInstances",
			ReadAction:    "DescribeInstances",
			DeleteAction:  "TerminateInstances",
			ImportID:      "instance_id",
			Attributes: []schema.AttributeSchema{
				{Name: "instance_id", Type: "string", Computed: true},
				{Name: "ami", Type: "string", Required: true, ForceNew: true},
				{Name: "instance_type", Type: "string", Required: true},
				{Name: "subnet_id", Type: "string", ForceNew: true, RefTo: "cloudmock_ec2_subnet.subnet_id"},
				{Name: "vpc_security_group_ids", Type: "set"},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "private_ip", Type: "string", Computed: true},
				{Name: "public_ip", Type: "string", Computed: true},
				{Name: "instance_state", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
			References: []schema.ResourceRef{
				{FromAttr: "subnet_id", ToResource: "cloudmock_ec2_subnet", ToAttr: "subnet_id"},
			},
		},
	}
}

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
	// Instances
	case "RunInstances":
		return handleRunInstances(ctx, s.store)
	case "DescribeInstances":
		return handleDescribeInstances(ctx, s.store)
	case "TerminateInstances":
		return handleTerminateInstances(ctx, s.store)
	case "StopInstances":
		return handleStopInstances(ctx, s.store)
	case "StartInstances":
		return handleStartInstances(ctx, s.store)
	case "DescribeInstanceStatus":
		return handleDescribeInstanceStatus(ctx, s.store)
	// Tagging
	case "CreateTags":
		return handleCreateTags(ctx, s.store)
	case "DeleteTags":
		return handleDeleteTags(ctx, s.store)
	case "DescribeTags":
		return handleDescribeTags(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
