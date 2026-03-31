package neptune

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// NeptuneService is the cloudmock implementation of the AWS Neptune API.
type NeptuneService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new NeptuneService.
func New(accountID, region string) *NeptuneService {
	return &NeptuneService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *NeptuneService) Name() string { return "neptune" }

// Actions returns the list of Neptune API actions supported.
func (s *NeptuneService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateDBCluster", Method: http.MethodPost, IAMAction: "neptune:CreateDBCluster"},
		{Name: "DescribeDBClusters", Method: http.MethodPost, IAMAction: "neptune:DescribeDBClusters"},
		{Name: "DeleteDBCluster", Method: http.MethodPost, IAMAction: "neptune:DeleteDBCluster"},
		{Name: "ModifyDBCluster", Method: http.MethodPost, IAMAction: "neptune:ModifyDBCluster"},
		{Name: "CreateDBInstance", Method: http.MethodPost, IAMAction: "neptune:CreateDBInstance"},
		{Name: "DescribeDBInstances", Method: http.MethodPost, IAMAction: "neptune:DescribeDBInstances"},
		{Name: "DeleteDBInstance", Method: http.MethodPost, IAMAction: "neptune:DeleteDBInstance"},
		{Name: "ModifyDBInstance", Method: http.MethodPost, IAMAction: "neptune:ModifyDBInstance"},
		{Name: "CreateDBClusterSnapshot", Method: http.MethodPost, IAMAction: "neptune:CreateDBClusterSnapshot"},
		{Name: "DescribeDBClusterSnapshots", Method: http.MethodPost, IAMAction: "neptune:DescribeDBClusterSnapshots"},
		{Name: "DeleteDBClusterSnapshot", Method: http.MethodPost, IAMAction: "neptune:DeleteDBClusterSnapshot"},
		{Name: "CreateDBSubnetGroup", Method: http.MethodPost, IAMAction: "neptune:CreateDBSubnetGroup"},
		{Name: "DescribeDBSubnetGroups", Method: http.MethodPost, IAMAction: "neptune:DescribeDBSubnetGroups"},
		{Name: "DeleteDBSubnetGroup", Method: http.MethodPost, IAMAction: "neptune:DeleteDBSubnetGroup"},
		{Name: "CreateDBClusterParameterGroup", Method: http.MethodPost, IAMAction: "neptune:CreateDBClusterParameterGroup"},
		{Name: "DescribeDBClusterParameterGroups", Method: http.MethodPost, IAMAction: "neptune:DescribeDBClusterParameterGroups"},
		{Name: "DeleteDBClusterParameterGroup", Method: http.MethodPost, IAMAction: "neptune:DeleteDBClusterParameterGroup"},
		{Name: "AddTagsToResource", Method: http.MethodPost, IAMAction: "neptune:AddTagsToResource"},
		{Name: "RemoveTagsFromResource", Method: http.MethodPost, IAMAction: "neptune:RemoveTagsFromResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "neptune:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *NeptuneService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Neptune request to the appropriate handler.
func (s *NeptuneService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "CreateDBCluster":
		return handleCreateDBCluster(ctx, s.store)
	case "DescribeDBClusters":
		return handleDescribeDBClusters(ctx, s.store)
	case "DeleteDBCluster":
		return handleDeleteDBCluster(ctx, s.store)
	case "ModifyDBCluster":
		return handleModifyDBCluster(ctx, s.store)
	case "CreateDBInstance":
		return handleCreateDBInstance(ctx, s.store)
	case "DescribeDBInstances":
		return handleDescribeDBInstances(ctx, s.store)
	case "DeleteDBInstance":
		return handleDeleteDBInstance(ctx, s.store)
	case "ModifyDBInstance":
		return handleModifyDBInstance(ctx, s.store)
	case "CreateDBClusterSnapshot":
		return handleCreateDBClusterSnapshot(ctx, s.store)
	case "DescribeDBClusterSnapshots":
		return handleDescribeDBClusterSnapshots(ctx, s.store)
	case "DeleteDBClusterSnapshot":
		return handleDeleteDBClusterSnapshot(ctx, s.store)
	case "CreateDBSubnetGroup":
		return handleCreateDBSubnetGroup(ctx, s.store)
	case "DescribeDBSubnetGroups":
		return handleDescribeDBSubnetGroups(ctx, s.store)
	case "DeleteDBSubnetGroup":
		return handleDeleteDBSubnetGroup(ctx, s.store)
	case "CreateDBClusterParameterGroup":
		return handleCreateDBClusterParameterGroup(ctx, s.store)
	case "DescribeDBClusterParameterGroups":
		return handleDescribeDBClusterParameterGroups(ctx, s.store)
	case "DeleteDBClusterParameterGroup":
		return handleDeleteDBClusterParameterGroup(ctx, s.store)
	case "AddTagsToResource":
		return handleAddTagsToResource(ctx, s.store)
	case "RemoveTagsFromResource":
		return handleRemoveTagsFromResource(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
