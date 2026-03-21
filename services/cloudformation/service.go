package cloudformation

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// CloudFormationService is the cloudmock implementation of the AWS CloudFormation API.
type CloudFormationService struct {
	store *StackStore
}

// New returns a new CloudFormationService for the given AWS account ID and region.
func New(accountID, region string) *CloudFormationService {
	return &CloudFormationService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *CloudFormationService) Name() string { return "cloudformation" }

// Actions returns the list of CloudFormation API actions supported by this service.
func (s *CloudFormationService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateStack", Method: http.MethodPost, IAMAction: "cloudformation:CreateStack"},
		{Name: "DeleteStack", Method: http.MethodPost, IAMAction: "cloudformation:DeleteStack"},
		{Name: "DescribeStacks", Method: http.MethodPost, IAMAction: "cloudformation:DescribeStacks"},
		{Name: "ListStacks", Method: http.MethodPost, IAMAction: "cloudformation:ListStacks"},
		{Name: "DescribeStackResources", Method: http.MethodPost, IAMAction: "cloudformation:DescribeStackResources"},
		{Name: "DescribeStackEvents", Method: http.MethodPost, IAMAction: "cloudformation:DescribeStackEvents"},
		{Name: "GetTemplate", Method: http.MethodPost, IAMAction: "cloudformation:GetTemplate"},
		{Name: "ValidateTemplate", Method: http.MethodPost, IAMAction: "cloudformation:ValidateTemplate"},
		{Name: "ListExports", Method: http.MethodPost, IAMAction: "cloudformation:ListExports"},
		{Name: "CreateChangeSet", Method: http.MethodPost, IAMAction: "cloudformation:CreateChangeSet"},
		{Name: "DescribeChangeSet", Method: http.MethodPost, IAMAction: "cloudformation:DescribeChangeSet"},
		{Name: "ExecuteChangeSet", Method: http.MethodPost, IAMAction: "cloudformation:ExecuteChangeSet"},
		{Name: "DeleteChangeSet", Method: http.MethodPost, IAMAction: "cloudformation:DeleteChangeSet"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *CloudFormationService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CloudFormation request to the appropriate handler.
// CloudFormation uses query-string / form-encoded bodies; the Action is in the form body.
func (s *CloudFormationService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "CreateStack":
		return handleCreateStack(ctx, s.store)
	case "DeleteStack":
		return handleDeleteStack(ctx, s.store)
	case "DescribeStacks":
		return handleDescribeStacks(ctx, s.store)
	case "ListStacks":
		return handleListStacks(ctx, s.store)
	case "DescribeStackResources":
		return handleDescribeStackResources(ctx, s.store)
	case "DescribeStackEvents":
		return handleDescribeStackEvents(ctx, s.store)
	case "GetTemplate":
		return handleGetTemplate(ctx, s.store)
	case "ValidateTemplate":
		return handleValidateTemplate(ctx, s.store)
	case "ListExports":
		return handleListExports(ctx, s.store)
	case "CreateChangeSet":
		return handleCreateChangeSet(ctx, s.store)
	case "DescribeChangeSet":
		return handleDescribeChangeSet(ctx, s.store)
	case "ExecuteChangeSet":
		return handleExecuteChangeSet(ctx, s.store)
	case "DeleteChangeSet":
		return handleDeleteChangeSet(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
