package ssm

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// SSMService is the cloudmock implementation of the AWS SSM Parameter Store API.
type SSMService struct {
	store *Store
}

// New returns a new SSMService for the given AWS account ID and region.
func New(accountID, region string) *SSMService {
	return &SSMService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *SSMService) Name() string { return "ssm" }

// Actions returns the list of SSM API actions supported by this service.
func (s *SSMService) Actions() []service.Action {
	return []service.Action{
		{Name: "PutParameter", Method: http.MethodPost, IAMAction: "ssm:PutParameter"},
		{Name: "GetParameter", Method: http.MethodPost, IAMAction: "ssm:GetParameter"},
		{Name: "GetParameters", Method: http.MethodPost, IAMAction: "ssm:GetParameters"},
		{Name: "GetParametersByPath", Method: http.MethodPost, IAMAction: "ssm:GetParametersByPath"},
		{Name: "DeleteParameter", Method: http.MethodPost, IAMAction: "ssm:DeleteParameter"},
		{Name: "DeleteParameters", Method: http.MethodPost, IAMAction: "ssm:DeleteParameters"},
		{Name: "DescribeParameters", Method: http.MethodPost, IAMAction: "ssm:DescribeParameters"},
		{Name: "CreateDocument", Method: http.MethodPost, IAMAction: "ssm:CreateDocument"},
		{Name: "DescribeDocument", Method: http.MethodPost, IAMAction: "ssm:DescribeDocument"},
		{Name: "GetDocument", Method: http.MethodPost, IAMAction: "ssm:GetDocument"},
		{Name: "ListDocuments", Method: http.MethodPost, IAMAction: "ssm:ListDocuments"},
		{Name: "DeleteDocument", Method: http.MethodPost, IAMAction: "ssm:DeleteDocument"},
		{Name: "StartAutomationExecution", Method: http.MethodPost, IAMAction: "ssm:StartAutomationExecution"},
		{Name: "DescribeAutomationExecutions", Method: http.MethodPost, IAMAction: "ssm:DescribeAutomationExecutions"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *SSMService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for SSM resource types.
func (s *SSMService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "ssm",
			ResourceType:  "aws_ssm_parameter",
			TerraformType: "cloudmock_ssm_parameter",
			AWSType:       "AWS::SSM::Parameter",
			CreateAction:  "PutParameter",
			ReadAction:    "GetParameter",
			UpdateAction:  "PutParameter",
			DeleteAction:  "DeleteParameter",
			ListAction:    "DescribeParameters",
			ImportID:      "name",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "type", Type: "string", Required: true},
				{Name: "value", Type: "string", Required: true},
				{Name: "description", Type: "string"},
				{Name: "key_id", Type: "string"},
				{Name: "tier", Type: "string", Default: "Standard"},
				{Name: "overwrite", Type: "bool", Default: false},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "version", Type: "int", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// HandleRequest routes an incoming SSM request to the appropriate handler.
// SSM uses the JSON protocol; the action is parsed from X-Amz-Target by the gateway
// and placed in ctx.Action (e.g. "PutParameter").
func (s *SSMService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "PutParameter":
		return handlePutParameter(ctx, s.store)
	case "GetParameter":
		return handleGetParameter(ctx, s.store)
	case "GetParameters":
		return handleGetParameters(ctx, s.store)
	case "GetParametersByPath":
		return handleGetParametersByPath(ctx, s.store)
	case "DeleteParameter":
		return handleDeleteParameter(ctx, s.store)
	case "DeleteParameters":
		return handleDeleteParameters(ctx, s.store)
	case "DescribeParameters":
		return handleDescribeParameters(ctx, s.store)
	case "CreateDocument":
		return handleCreateDocument(ctx, s.store)
	case "DescribeDocument":
		return handleDescribeDocument(ctx, s.store)
	case "GetDocument":
		return handleGetDocument(ctx, s.store)
	case "ListDocuments":
		return handleListDocuments(ctx, s.store)
	case "DeleteDocument":
		return handleDeleteDocument(ctx, s.store)
	case "StartAutomationExecution":
		return handleStartAutomationExecution(ctx, s.store)
	case "DescribeAutomationExecutions":
		return handleDescribeAutomationExecutions(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
