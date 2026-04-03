package redshift

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/schema"
	"github.com/neureaux/cloudmock/pkg/service"
)

// RedshiftService is the cloudmock implementation of the AWS Redshift API.
type RedshiftService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new RedshiftService for the given AWS account ID and region.
func New(accountID, region string) *RedshiftService {
	return &RedshiftService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *RedshiftService) Name() string { return "redshift" }

// Actions returns the list of Redshift API actions supported by this service.
func (s *RedshiftService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateCluster", Method: http.MethodPost, IAMAction: "redshift:CreateCluster"},
		{Name: "DescribeClusters", Method: http.MethodPost, IAMAction: "redshift:DescribeClusters"},
		{Name: "DeleteCluster", Method: http.MethodPost, IAMAction: "redshift:DeleteCluster"},
		{Name: "ModifyCluster", Method: http.MethodPost, IAMAction: "redshift:ModifyCluster"},
		{Name: "RebootCluster", Method: http.MethodPost, IAMAction: "redshift:RebootCluster"},
		{Name: "CreateClusterSnapshot", Method: http.MethodPost, IAMAction: "redshift:CreateClusterSnapshot"},
		{Name: "DescribeClusterSnapshots", Method: http.MethodPost, IAMAction: "redshift:DescribeClusterSnapshots"},
		{Name: "DeleteClusterSnapshot", Method: http.MethodPost, IAMAction: "redshift:DeleteClusterSnapshot"},
		{Name: "RestoreFromClusterSnapshot", Method: http.MethodPost, IAMAction: "redshift:RestoreFromClusterSnapshot"},
		{Name: "CreateClusterSubnetGroup", Method: http.MethodPost, IAMAction: "redshift:CreateClusterSubnetGroup"},
		{Name: "DescribeClusterSubnetGroups", Method: http.MethodPost, IAMAction: "redshift:DescribeClusterSubnetGroups"},
		{Name: "DeleteClusterSubnetGroup", Method: http.MethodPost, IAMAction: "redshift:DeleteClusterSubnetGroup"},
		{Name: "CreateClusterParameterGroup", Method: http.MethodPost, IAMAction: "redshift:CreateClusterParameterGroup"},
		{Name: "DescribeClusterParameterGroups", Method: http.MethodPost, IAMAction: "redshift:DescribeClusterParameterGroups"},
		{Name: "DeleteClusterParameterGroup", Method: http.MethodPost, IAMAction: "redshift:DeleteClusterParameterGroup"},
		{Name: "PauseCluster", Method: http.MethodPost, IAMAction: "redshift:PauseCluster"},
		{Name: "ResumeCluster", Method: http.MethodPost, IAMAction: "redshift:ResumeCluster"},
		{Name: "CreateTags", Method: http.MethodPost, IAMAction: "redshift:CreateTags"},
		{Name: "DeleteTags", Method: http.MethodPost, IAMAction: "redshift:DeleteTags"},
		{Name: "DescribeTags", Method: http.MethodPost, IAMAction: "redshift:DescribeTags"},
		{Name: "AddTagsToResource", Method: http.MethodPost, IAMAction: "redshift:CreateTags"},
		{Name: "RemoveTagsFromResource", Method: http.MethodPost, IAMAction: "redshift:DeleteTags"},
		{Name: "ExecuteStatement", Method: http.MethodPost, IAMAction: "redshift-data:ExecuteStatement"},
		{Name: "DescribeStatement", Method: http.MethodPost, IAMAction: "redshift-data:DescribeStatement"},
		{Name: "GetStatementResult", Method: http.MethodPost, IAMAction: "redshift-data:GetStatementResult"},
	}
}

// HealthCheck always returns nil.
func (s *RedshiftService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for Redshift resource types.
func (s *RedshiftService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "redshift",
			ResourceType:  "aws_redshift_cluster",
			TerraformType: "cloudmock_redshift_cluster",
			AWSType:       "AWS::Redshift::Cluster",
			CreateAction:  "CreateCluster",
			ReadAction:    "DescribeClusters",
			UpdateAction:  "ModifyCluster",
			DeleteAction:  "DeleteCluster",
			ImportID:      "cluster_identifier",
			Attributes: []schema.AttributeSchema{
				{Name: "cluster_identifier", Type: "string", Required: true, ForceNew: true},
				{Name: "node_type", Type: "string", Required: true},
				{Name: "master_username", Type: "string", Required: true, ForceNew: true},
				{Name: "master_password", Type: "string", Required: true},
				{Name: "cluster_type", Type: "string", Default: "single-node"},
				{Name: "number_of_nodes", Type: "int", Default: 1},
				{Name: "database_name", Type: "string", Default: "dev"},
				{Name: "port", Type: "int", Default: 5439},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "endpoint", Type: "string", Computed: true},
				{Name: "dns_name", Type: "string", Computed: true},
				{Name: "publicly_accessible", Type: "bool", Default: true},
				{Name: "skip_final_snapshot", Type: "bool", Default: false},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// HandleRequest routes an incoming Redshift request to the appropriate handler.
func (s *RedshiftService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "CreateCluster":
		return handleCreateCluster(ctx, s.store)
	case "DescribeClusters":
		return handleDescribeClusters(ctx, s.store)
	case "DeleteCluster":
		return handleDeleteCluster(ctx, s.store)
	case "ModifyCluster":
		return handleModifyCluster(ctx, s.store)
	case "RebootCluster":
		return handleRebootCluster(ctx, s.store)
	case "CreateClusterSnapshot":
		return handleCreateClusterSnapshot(ctx, s.store)
	case "DescribeClusterSnapshots":
		return handleDescribeClusterSnapshots(ctx, s.store)
	case "DeleteClusterSnapshot":
		return handleDeleteClusterSnapshot(ctx, s.store)
	case "RestoreFromClusterSnapshot":
		return handleRestoreFromClusterSnapshot(ctx, s.store)
	case "CreateClusterSubnetGroup":
		return handleCreateClusterSubnetGroup(ctx, s.store)
	case "DescribeClusterSubnetGroups":
		return handleDescribeClusterSubnetGroups(ctx, s.store)
	case "DeleteClusterSubnetGroup":
		return handleDeleteClusterSubnetGroup(ctx, s.store)
	case "CreateClusterParameterGroup":
		return handleCreateClusterParameterGroup(ctx, s.store)
	case "DescribeClusterParameterGroups":
		return handleDescribeClusterParameterGroups(ctx, s.store)
	case "DeleteClusterParameterGroup":
		return handleDeleteClusterParameterGroup(ctx, s.store)
	case "PauseCluster":
		return handlePauseCluster(ctx, s.store)
	case "ResumeCluster":
		return handleResumeCluster(ctx, s.store)
	case "CreateTags":
		return handleCreateTags(ctx, s.store)
	case "DeleteTags":
		return handleDeleteTags(ctx, s.store)
	case "DescribeTags":
		return handleDescribeTags(ctx, s.store)
	case "AddTagsToResource":
		return handleAddTagsToResource(ctx, s.store)
	case "RemoveTagsFromResource":
		return handleRemoveTagsFromResource(ctx, s.store)
	case "ExecuteStatement":
		return handleExecuteStatement(ctx, s.store)
	case "DescribeStatement":
		return handleDescribeStatement(ctx, s.store)
	case "GetStatementResult":
		return handleGetStatementResult(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
