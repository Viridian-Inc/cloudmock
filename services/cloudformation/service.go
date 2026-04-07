package cloudformation

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// CloudFormationService is the cloudmock implementation of the AWS CloudFormation API.
type CloudFormationService struct {
	store     *StackStore
	accountID string
	region    string
}

// New returns a new CloudFormationService for the given AWS account ID and region.
func New(accountID, region string) *CloudFormationService {
	return &CloudFormationService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// SetLocator wires the service locator, enabling real resource provisioning
// when CreateStack is called.
func (s *CloudFormationService) SetLocator(locator ServiceLocator) {
	p := NewProvisioner(locator, s.accountID, s.region)
	s.store.SetProvisioner(p)
}

// GetStackResources returns all resources across all stacks for topology queries.
func (s *CloudFormationService) GetStackResources() map[string][]StackResource {
	stacks := s.store.AllStacks()
	result := make(map[string][]StackResource, len(stacks))
	for _, st := range stacks {
		if st.StackStatus == "DELETE_COMPLETE" {
			continue
		}
		result[st.StackName] = st.Resources
	}
	return result
}

// GetStackResourcesSummary returns parallel slices for topology building.
func (s *CloudFormationService) GetStackResourcesSummary() (stackNames []string, resourceTypes [][]string, logicalIDs [][]string) {
	stacks := s.store.AllStacks()
	for _, st := range stacks {
		if st.StackStatus == "DELETE_COMPLETE" {
			continue
		}
		types := make([]string, 0, len(st.Resources))
		ids := make([]string, 0, len(st.Resources))
		for _, r := range st.Resources {
			types = append(types, r.ResourceType)
			ids = append(ids, r.LogicalResourceId)
		}
		stackNames = append(stackNames, st.StackName)
		resourceTypes = append(resourceTypes, types)
		logicalIDs = append(logicalIDs, ids)
	}
	return stackNames, resourceTypes, logicalIDs
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
