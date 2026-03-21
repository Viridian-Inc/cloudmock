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
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
