package elasticmapreduce

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// EMRService is the cloudmock implementation of the AWS EMR API.
type EMRService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new EMRService for the given AWS account ID and region.
func New(accountID, region string) *EMRService {
	return &EMRService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *EMRService) Name() string { return "elasticmapreduce" }

// SetLocator sets the service locator for cross-service lookups (e.g., EC2).
func (s *EMRService) SetLocator(locator ServiceLocator) {
	s.store.SetLocator(locator)
}

// Actions returns the list of EMR API actions supported by this service.
func (s *EMRService) Actions() []service.Action {
	return []service.Action{
		{Name: "RunJobFlow", Method: http.MethodPost, IAMAction: "elasticmapreduce:RunJobFlow"},
		{Name: "DescribeCluster", Method: http.MethodPost, IAMAction: "elasticmapreduce:DescribeCluster"},
		{Name: "ListClusters", Method: http.MethodPost, IAMAction: "elasticmapreduce:ListClusters"},
		{Name: "TerminateJobFlows", Method: http.MethodPost, IAMAction: "elasticmapreduce:TerminateJobFlows"},
		{Name: "AddJobFlowSteps", Method: http.MethodPost, IAMAction: "elasticmapreduce:AddJobFlowSteps"},
		{Name: "ListSteps", Method: http.MethodPost, IAMAction: "elasticmapreduce:ListSteps"},
		{Name: "DescribeStep", Method: http.MethodPost, IAMAction: "elasticmapreduce:DescribeStep"},
		{Name: "AddInstanceGroups", Method: http.MethodPost, IAMAction: "elasticmapreduce:AddInstanceGroups"},
		{Name: "ListInstanceGroups", Method: http.MethodPost, IAMAction: "elasticmapreduce:ListInstanceGroups"},
		{Name: "ModifyInstanceGroups", Method: http.MethodPost, IAMAction: "elasticmapreduce:ModifyInstanceGroups"},
		{Name: "SetTerminationProtection", Method: http.MethodPost, IAMAction: "elasticmapreduce:SetTerminationProtection"},
		{Name: "SetVisibleToAllUsers", Method: http.MethodPost, IAMAction: "elasticmapreduce:SetVisibleToAllUsers"},
		{Name: "AddTags", Method: http.MethodPost, IAMAction: "elasticmapreduce:AddTags"},
		{Name: "RemoveTags", Method: http.MethodPost, IAMAction: "elasticmapreduce:RemoveTags"},
		{Name: "CreateSecurityConfiguration", Method: http.MethodPost, IAMAction: "elasticmapreduce:CreateSecurityConfiguration"},
		{Name: "DescribeSecurityConfiguration", Method: http.MethodPost, IAMAction: "elasticmapreduce:DescribeSecurityConfiguration"},
		{Name: "ListSecurityConfigurations", Method: http.MethodPost, IAMAction: "elasticmapreduce:ListSecurityConfigurations"},
		{Name: "DeleteSecurityConfiguration", Method: http.MethodPost, IAMAction: "elasticmapreduce:DeleteSecurityConfiguration"},
	}
}

// HealthCheck always returns nil.
func (s *EMRService) HealthCheck() error { return nil }

// HandleRequest routes an incoming EMR request to the appropriate handler.
func (s *EMRService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "RunJobFlow":
		return handleRunJobFlow(ctx, s.store)
	case "DescribeCluster":
		return handleDescribeCluster(ctx, s.store)
	case "ListClusters":
		return handleListClusters(ctx, s.store)
	case "TerminateJobFlows":
		return handleTerminateJobFlows(ctx, s.store)
	case "AddJobFlowSteps":
		return handleAddJobFlowSteps(ctx, s.store)
	case "ListSteps":
		return handleListSteps(ctx, s.store)
	case "DescribeStep":
		return handleDescribeStep(ctx, s.store)
	case "AddInstanceGroups":
		return handleAddInstanceGroups(ctx, s.store)
	case "ListInstanceGroups":
		return handleListInstanceGroups(ctx, s.store)
	case "ModifyInstanceGroups":
		return handleModifyInstanceGroups(ctx, s.store)
	case "SetTerminationProtection":
		return handleSetTerminationProtection(ctx, s.store)
	case "SetVisibleToAllUsers":
		return handleSetVisibleToAllUsers(ctx, s.store)
	case "AddTags":
		return handleAddTags(ctx, s.store)
	case "RemoveTags":
		return handleRemoveTags(ctx, s.store)
	case "CreateSecurityConfiguration":
		return handleCreateSecurityConfiguration(ctx, s.store)
	case "DescribeSecurityConfiguration":
		return handleDescribeSecurityConfiguration(ctx, s.store)
	case "ListSecurityConfigurations":
		return handleListSecurityConfigurations(ctx, s.store)
	case "DeleteSecurityConfiguration":
		return handleDeleteSecurityConfiguration(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
