package docdb

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// DocDBService is the cloudmock implementation of the AWS DocumentDB API.
type DocDBService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new DocDBService.
func New(accountID, region string) *DocDBService {
	return &DocDBService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *DocDBService) Name() string { return "docdb" }

// Actions returns the list of DocDB API actions supported.
func (s *DocDBService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateDBCluster", Method: http.MethodPost, IAMAction: "rds:CreateDBCluster"},
		{Name: "DescribeDBClusters", Method: http.MethodPost, IAMAction: "rds:DescribeDBClusters"},
		{Name: "DeleteDBCluster", Method: http.MethodPost, IAMAction: "rds:DeleteDBCluster"},
		{Name: "ModifyDBCluster", Method: http.MethodPost, IAMAction: "rds:ModifyDBCluster"},
		{Name: "CreateDBInstance", Method: http.MethodPost, IAMAction: "rds:CreateDBInstance"},
		{Name: "DescribeDBInstances", Method: http.MethodPost, IAMAction: "rds:DescribeDBInstances"},
		{Name: "DeleteDBInstance", Method: http.MethodPost, IAMAction: "rds:DeleteDBInstance"},
		{Name: "ModifyDBInstance", Method: http.MethodPost, IAMAction: "rds:ModifyDBInstance"},
		{Name: "CreateDBClusterSnapshot", Method: http.MethodPost, IAMAction: "rds:CreateDBClusterSnapshot"},
		{Name: "DescribeDBClusterSnapshots", Method: http.MethodPost, IAMAction: "rds:DescribeDBClusterSnapshots"},
		{Name: "DeleteDBClusterSnapshot", Method: http.MethodPost, IAMAction: "rds:DeleteDBClusterSnapshot"},
		{Name: "CreateDBSubnetGroup", Method: http.MethodPost, IAMAction: "rds:CreateDBSubnetGroup"},
		{Name: "DescribeDBSubnetGroups", Method: http.MethodPost, IAMAction: "rds:DescribeDBSubnetGroups"},
		{Name: "DeleteDBSubnetGroup", Method: http.MethodPost, IAMAction: "rds:DeleteDBSubnetGroup"},
		{Name: "AddTagsToResource", Method: http.MethodPost, IAMAction: "rds:AddTagsToResource"},
		{Name: "RemoveTagsFromResource", Method: http.MethodPost, IAMAction: "rds:RemoveTagsFromResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "rds:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *DocDBService) HealthCheck() error { return nil }

// HandleRequest routes an incoming DocDB request to the appropriate handler.
func (s *DocDBService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
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
