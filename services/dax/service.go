package dax

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// DAXService is the cloudmock implementation of the Amazon DynamoDB Accelerator (DAX) API.
type DAXService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new DAXService for the given AWS account ID and region.
func New(accountID, region string) *DAXService {
	return &DAXService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *DAXService) Name() string { return "dax" }

// Actions returns the list of DAX API actions supported by this service.
func (s *DAXService) Actions() []service.Action {
	return []service.Action{
		// Cluster actions
		{Name: "CreateCluster", Method: http.MethodPost, IAMAction: "dax:CreateCluster"},
		{Name: "DescribeClusters", Method: http.MethodPost, IAMAction: "dax:DescribeClusters"},
		{Name: "UpdateCluster", Method: http.MethodPost, IAMAction: "dax:UpdateCluster"},
		{Name: "DeleteCluster", Method: http.MethodPost, IAMAction: "dax:DeleteCluster"},
		{Name: "IncreaseReplicationFactor", Method: http.MethodPost, IAMAction: "dax:IncreaseReplicationFactor"},
		{Name: "DecreaseReplicationFactor", Method: http.MethodPost, IAMAction: "dax:DecreaseReplicationFactor"},
		// Subnet Group actions
		{Name: "CreateSubnetGroup", Method: http.MethodPost, IAMAction: "dax:CreateSubnetGroup"},
		{Name: "DescribeSubnetGroups", Method: http.MethodPost, IAMAction: "dax:DescribeSubnetGroups"},
		{Name: "DeleteSubnetGroup", Method: http.MethodPost, IAMAction: "dax:DeleteSubnetGroup"},
		// Parameter Group actions
		{Name: "CreateParameterGroup", Method: http.MethodPost, IAMAction: "dax:CreateParameterGroup"},
		{Name: "DescribeParameterGroups", Method: http.MethodPost, IAMAction: "dax:DescribeParameterGroups"},
		{Name: "UpdateParameterGroup", Method: http.MethodPost, IAMAction: "dax:UpdateParameterGroup"},
		{Name: "DeleteParameterGroup", Method: http.MethodPost, IAMAction: "dax:DeleteParameterGroup"},
		{Name: "DescribeParameters", Method: http.MethodPost, IAMAction: "dax:DescribeParameters"},
		{Name: "DescribeDefaultParameters", Method: http.MethodPost, IAMAction: "dax:DescribeDefaultParameters"},
		// Tagging actions
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "dax:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "dax:UntagResource"},
		{Name: "ListTags", Method: http.MethodPost, IAMAction: "dax:ListTags"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *DAXService) HealthCheck() error { return nil }

// HandleRequest routes an incoming DAX request to the appropriate handler.
func (s *DAXService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	// Cluster
	case "CreateCluster":
		return handleCreateCluster(ctx, s.store)
	case "DescribeClusters":
		return handleDescribeClusters(ctx, s.store)
	case "UpdateCluster":
		return handleUpdateCluster(ctx, s.store)
	case "DeleteCluster":
		return handleDeleteCluster(ctx, s.store)
	case "IncreaseReplicationFactor":
		return handleIncreaseReplicationFactor(ctx, s.store)
	case "DecreaseReplicationFactor":
		return handleDecreaseReplicationFactor(ctx, s.store)
	// Subnet Group
	case "CreateSubnetGroup":
		return handleCreateSubnetGroup(ctx, s.store)
	case "DescribeSubnetGroups":
		return handleDescribeSubnetGroups(ctx, s.store)
	case "DeleteSubnetGroup":
		return handleDeleteSubnetGroup(ctx, s.store)
	// Parameter Group
	case "CreateParameterGroup":
		return handleCreateParameterGroup(ctx, s.store)
	case "DescribeParameterGroups":
		return handleDescribeParameterGroups(ctx, s.store)
	case "UpdateParameterGroup":
		return handleUpdateParameterGroup(ctx, s.store)
	case "DeleteParameterGroup":
		return handleDeleteParameterGroup(ctx, s.store)
	case "DescribeParameters":
		return handleDescribeParameters(ctx, s.store)
	case "DescribeDefaultParameters":
		return handleDescribeDefaultParameters(ctx, s.store)
	// Tagging
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "ListTags":
		return handleListTags(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
