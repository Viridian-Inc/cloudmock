package stepfunctions

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// StepFunctionsService is the cloudmock implementation of the AWS Step Functions API.
type StepFunctionsService struct {
	store *Store
}

// New returns a new StepFunctionsService for the given AWS account ID and region.
func New(accountID, region string) *StepFunctionsService {
	return &StepFunctionsService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
// Step Functions uses "states" as the credential scope service name.
func (s *StepFunctionsService) Name() string { return "states" }

// Actions returns the list of Step Functions API actions supported by this service.
func (s *StepFunctionsService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateStateMachine", Method: http.MethodPost, IAMAction: "states:CreateStateMachine"},
		{Name: "DeleteStateMachine", Method: http.MethodPost, IAMAction: "states:DeleteStateMachine"},
		{Name: "DescribeStateMachine", Method: http.MethodPost, IAMAction: "states:DescribeStateMachine"},
		{Name: "ListStateMachines", Method: http.MethodPost, IAMAction: "states:ListStateMachines"},
		{Name: "UpdateStateMachine", Method: http.MethodPost, IAMAction: "states:UpdateStateMachine"},
		{Name: "StartExecution", Method: http.MethodPost, IAMAction: "states:StartExecution"},
		{Name: "DescribeExecution", Method: http.MethodPost, IAMAction: "states:DescribeExecution"},
		{Name: "StopExecution", Method: http.MethodPost, IAMAction: "states:StopExecution"},
		{Name: "ListExecutions", Method: http.MethodPost, IAMAction: "states:ListExecutions"},
		{Name: "GetExecutionHistory", Method: http.MethodPost, IAMAction: "states:GetExecutionHistory"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "states:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "states:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "states:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *StepFunctionsService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for Step Functions resource types.
func (s *StepFunctionsService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "stepfunctions",
			ResourceType:  "aws_sfn_state_machine",
			TerraformType: "cloudmock_sfn_state_machine",
			AWSType:       "AWS::StepFunctions::StateMachine",
			CreateAction:  "CreateStateMachine",
			ReadAction:    "DescribeStateMachine",
			UpdateAction:  "UpdateStateMachine",
			DeleteAction:  "DeleteStateMachine",
			ListAction:    "ListStateMachines",
			ImportID:      "arn",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "definition", Type: "string", Required: true},
				{Name: "role_arn", Type: "string", Required: true},
				{Name: "type", Type: "string", Default: "STANDARD", ForceNew: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "creation_date", Type: "string", Computed: true},
				{Name: "status", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// HandleRequest routes an incoming Step Functions request to the appropriate handler.
// Step Functions uses the JSON protocol; the action is parsed from X-Amz-Target by the
// gateway (e.g. "AWSStepFunctions.CreateStateMachine" → "CreateStateMachine").
func (s *StepFunctionsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateStateMachine":
		return handleCreateStateMachine(ctx, s.store)
	case "DeleteStateMachine":
		return handleDeleteStateMachine(ctx, s.store)
	case "DescribeStateMachine":
		return handleDescribeStateMachine(ctx, s.store)
	case "ListStateMachines":
		return handleListStateMachines(ctx, s.store)
	case "UpdateStateMachine":
		return handleUpdateStateMachine(ctx, s.store)
	case "StartExecution":
		return handleStartExecution(ctx, s.store)
	case "DescribeExecution":
		return handleDescribeExecution(ctx, s.store)
	case "StopExecution":
		return handleStopExecution(ctx, s.store)
	case "ListExecutions":
		return handleListExecutions(ctx, s.store)
	case "GetExecutionHistory":
		return handleGetExecutionHistory(ctx, s.store)
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
