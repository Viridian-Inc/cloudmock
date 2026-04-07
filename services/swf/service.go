package swf

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// SWFService is the cloudmock implementation of the AWS Simple Workflow Service API.
type SWFService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new SWFService for the given AWS account ID and region.
func New(accountID, region string) *SWFService {
	return &SWFService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *SWFService) Name() string { return "swf" }

// Actions returns the list of SWF API actions supported by this service.
func (s *SWFService) Actions() []service.Action {
	return []service.Action{
		{Name: "RegisterDomain", Method: http.MethodPost, IAMAction: "swf:RegisterDomain"},
		{Name: "DescribeDomain", Method: http.MethodPost, IAMAction: "swf:DescribeDomain"},
		{Name: "ListDomains", Method: http.MethodPost, IAMAction: "swf:ListDomains"},
		{Name: "DeprecateDomain", Method: http.MethodPost, IAMAction: "swf:DeprecateDomain"},
		{Name: "RegisterWorkflowType", Method: http.MethodPost, IAMAction: "swf:RegisterWorkflowType"},
		{Name: "DescribeWorkflowType", Method: http.MethodPost, IAMAction: "swf:DescribeWorkflowType"},
		{Name: "ListWorkflowTypes", Method: http.MethodPost, IAMAction: "swf:ListWorkflowTypes"},
		{Name: "DeprecateWorkflowType", Method: http.MethodPost, IAMAction: "swf:DeprecateWorkflowType"},
		{Name: "RegisterActivityType", Method: http.MethodPost, IAMAction: "swf:RegisterActivityType"},
		{Name: "DescribeActivityType", Method: http.MethodPost, IAMAction: "swf:DescribeActivityType"},
		{Name: "ListActivityTypes", Method: http.MethodPost, IAMAction: "swf:ListActivityTypes"},
		{Name: "DeprecateActivityType", Method: http.MethodPost, IAMAction: "swf:DeprecateActivityType"},
		{Name: "StartWorkflowExecution", Method: http.MethodPost, IAMAction: "swf:StartWorkflowExecution"},
		{Name: "DescribeWorkflowExecution", Method: http.MethodPost, IAMAction: "swf:DescribeWorkflowExecution"},
		{Name: "ListOpenWorkflowExecutions", Method: http.MethodPost, IAMAction: "swf:ListOpenWorkflowExecutions"},
		{Name: "ListClosedWorkflowExecutions", Method: http.MethodPost, IAMAction: "swf:ListClosedWorkflowExecutions"},
		{Name: "TerminateWorkflowExecution", Method: http.MethodPost, IAMAction: "swf:TerminateWorkflowExecution"},
		{Name: "SignalWorkflowExecution", Method: http.MethodPost, IAMAction: "swf:SignalWorkflowExecution"},
		{Name: "RequestCancelWorkflowExecution", Method: http.MethodPost, IAMAction: "swf:RequestCancelWorkflowExecution"},
		{Name: "PollForDecisionTask", Method: http.MethodPost, IAMAction: "swf:PollForDecisionTask"},
		{Name: "RespondDecisionTaskCompleted", Method: http.MethodPost, IAMAction: "swf:RespondDecisionTaskCompleted"},
		{Name: "PollForActivityTask", Method: http.MethodPost, IAMAction: "swf:PollForActivityTask"},
		{Name: "RespondActivityTaskCompleted", Method: http.MethodPost, IAMAction: "swf:RespondActivityTaskCompleted"},
		{Name: "RespondActivityTaskFailed", Method: http.MethodPost, IAMAction: "swf:RespondActivityTaskFailed"},
		{Name: "GetWorkflowExecutionHistory", Method: http.MethodPost, IAMAction: "swf:GetWorkflowExecutionHistory"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "swf:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "swf:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "swf:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *SWFService) HealthCheck() error { return nil }

// HandleRequest routes an incoming SWF request to the appropriate handler.
func (s *SWFService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "RegisterDomain":
		return handleRegisterDomain(ctx, s.store)
	case "DescribeDomain":
		return handleDescribeDomain(ctx, s.store)
	case "ListDomains":
		return handleListDomains(ctx, s.store)
	case "DeprecateDomain":
		return handleDeprecateDomain(ctx, s.store)
	case "RegisterWorkflowType":
		return handleRegisterWorkflowType(ctx, s.store)
	case "DescribeWorkflowType":
		return handleDescribeWorkflowType(ctx, s.store)
	case "ListWorkflowTypes":
		return handleListWorkflowTypes(ctx, s.store)
	case "DeprecateWorkflowType":
		return handleDeprecateWorkflowType(ctx, s.store)
	case "RegisterActivityType":
		return handleRegisterActivityType(ctx, s.store)
	case "DescribeActivityType":
		return handleDescribeActivityType(ctx, s.store)
	case "ListActivityTypes":
		return handleListActivityTypes(ctx, s.store)
	case "DeprecateActivityType":
		return handleDeprecateActivityType(ctx, s.store)
	case "StartWorkflowExecution":
		return handleStartWorkflowExecution(ctx, s.store)
	case "DescribeWorkflowExecution":
		return handleDescribeWorkflowExecution(ctx, s.store)
	case "ListOpenWorkflowExecutions":
		return handleListOpenWorkflowExecutions(ctx, s.store)
	case "ListClosedWorkflowExecutions":
		return handleListClosedWorkflowExecutions(ctx, s.store)
	case "TerminateWorkflowExecution":
		return handleTerminateWorkflowExecution(ctx, s.store)
	case "SignalWorkflowExecution":
		return handleSignalWorkflowExecution(ctx, s.store)
	case "RequestCancelWorkflowExecution":
		return handleRequestCancelWorkflowExecution(ctx, s.store)
	case "PollForDecisionTask":
		return handlePollForDecisionTask(ctx, s.store)
	case "RespondDecisionTaskCompleted":
		return handleRespondDecisionTaskCompleted(ctx, s.store)
	case "PollForActivityTask":
		return handlePollForActivityTask(ctx, s.store)
	case "RespondActivityTaskCompleted":
		return handleRespondActivityTaskCompleted(ctx, s.store)
	case "RespondActivityTaskFailed":
		return handleRespondActivityTaskFailed(ctx, s.store)
	case "GetWorkflowExecutionHistory":
		return handleGetWorkflowExecutionHistory(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("UnknownOperationException",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
