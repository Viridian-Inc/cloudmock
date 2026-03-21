package rds

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// RDSService is the cloudmock implementation of the AWS Relational Database Service API.
type RDSService struct {
	store *Store
}

// New returns a new RDSService for the given AWS account ID and region.
func New(accountID, region string) *RDSService {
	return &RDSService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *RDSService) Name() string { return "rds" }

// Actions returns the list of RDS API actions supported by this service.
func (s *RDSService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateDBInstance", Method: http.MethodPost, IAMAction: "rds:CreateDBInstance"},
		{Name: "DeleteDBInstance", Method: http.MethodPost, IAMAction: "rds:DeleteDBInstance"},
		{Name: "DescribeDBInstances", Method: http.MethodPost, IAMAction: "rds:DescribeDBInstances"},
		{Name: "ModifyDBInstance", Method: http.MethodPost, IAMAction: "rds:ModifyDBInstance"},
		{Name: "CreateDBCluster", Method: http.MethodPost, IAMAction: "rds:CreateDBCluster"},
		{Name: "DeleteDBCluster", Method: http.MethodPost, IAMAction: "rds:DeleteDBCluster"},
		{Name: "DescribeDBClusters", Method: http.MethodPost, IAMAction: "rds:DescribeDBClusters"},
		{Name: "CreateDBSnapshot", Method: http.MethodPost, IAMAction: "rds:CreateDBSnapshot"},
		{Name: "DeleteDBSnapshot", Method: http.MethodPost, IAMAction: "rds:DeleteDBSnapshot"},
		{Name: "DescribeDBSnapshots", Method: http.MethodPost, IAMAction: "rds:DescribeDBSnapshots"},
		{Name: "CreateDBSubnetGroup", Method: http.MethodPost, IAMAction: "rds:CreateDBSubnetGroup"},
		{Name: "DescribeDBSubnetGroups", Method: http.MethodPost, IAMAction: "rds:DescribeDBSubnetGroups"},
		{Name: "DeleteDBSubnetGroup", Method: http.MethodPost, IAMAction: "rds:DeleteDBSubnetGroup"},
		{Name: "AddTagsToResource", Method: http.MethodPost, IAMAction: "rds:AddTagsToResource"},
		{Name: "RemoveTagsFromResource", Method: http.MethodPost, IAMAction: "rds:RemoveTagsFromResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "rds:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *RDSService) HealthCheck() error { return nil }

// HandleRequest routes an incoming RDS request to the appropriate handler.
// RDS uses form-encoded POST bodies; the Action may appear in the query string
// (already parsed into ctx.Params) or in the form-encoded body.
func (s *RDSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "CreateDBInstance":
		return handleCreateDBInstance(ctx, s.store)
	case "DeleteDBInstance":
		return handleDeleteDBInstance(ctx, s.store)
	case "DescribeDBInstances":
		return handleDescribeDBInstances(ctx, s.store)
	case "ModifyDBInstance":
		return handleModifyDBInstance(ctx, s.store)
	case "CreateDBCluster":
		return handleCreateDBCluster(ctx, s.store)
	case "DeleteDBCluster":
		return handleDeleteDBCluster(ctx, s.store)
	case "DescribeDBClusters":
		return handleDescribeDBClusters(ctx, s.store)
	case "CreateDBSnapshot":
		return handleCreateDBSnapshot(ctx, s.store)
	case "DeleteDBSnapshot":
		return handleDeleteDBSnapshot(ctx, s.store)
	case "DescribeDBSnapshots":
		return handleDescribeDBSnapshots(ctx, s.store)
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
