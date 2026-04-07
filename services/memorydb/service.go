package memorydb

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// MemoryDBService is the cloudmock implementation of the AWS MemoryDB API.
type MemoryDBService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new MemoryDBService.
func New(accountID, region string) *MemoryDBService {
	return &MemoryDBService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *MemoryDBService) Name() string { return "memorydb" }

// Actions returns the list of MemoryDB API actions supported.
func (s *MemoryDBService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateCluster", Method: http.MethodPost, IAMAction: "memorydb:CreateCluster"},
		{Name: "DescribeClusters", Method: http.MethodPost, IAMAction: "memorydb:DescribeClusters"},
		{Name: "DeleteCluster", Method: http.MethodPost, IAMAction: "memorydb:DeleteCluster"},
		{Name: "UpdateCluster", Method: http.MethodPost, IAMAction: "memorydb:UpdateCluster"},
		{Name: "CreateACL", Method: http.MethodPost, IAMAction: "memorydb:CreateACL"},
		{Name: "DescribeACLs", Method: http.MethodPost, IAMAction: "memorydb:DescribeACLs"},
		{Name: "DeleteACL", Method: http.MethodPost, IAMAction: "memorydb:DeleteACL"},
		{Name: "UpdateACL", Method: http.MethodPost, IAMAction: "memorydb:UpdateACL"},
		{Name: "CreateUser", Method: http.MethodPost, IAMAction: "memorydb:CreateUser"},
		{Name: "DescribeUsers", Method: http.MethodPost, IAMAction: "memorydb:DescribeUsers"},
		{Name: "DeleteUser", Method: http.MethodPost, IAMAction: "memorydb:DeleteUser"},
		{Name: "UpdateUser", Method: http.MethodPost, IAMAction: "memorydb:UpdateUser"},
		{Name: "CreateSubnetGroup", Method: http.MethodPost, IAMAction: "memorydb:CreateSubnetGroup"},
		{Name: "DescribeSubnetGroups", Method: http.MethodPost, IAMAction: "memorydb:DescribeSubnetGroups"},
		{Name: "DeleteSubnetGroup", Method: http.MethodPost, IAMAction: "memorydb:DeleteSubnetGroup"},
		{Name: "CreateParameterGroup", Method: http.MethodPost, IAMAction: "memorydb:CreateParameterGroup"},
		{Name: "DescribeParameterGroups", Method: http.MethodPost, IAMAction: "memorydb:DescribeParameterGroups"},
		{Name: "DeleteParameterGroup", Method: http.MethodPost, IAMAction: "memorydb:DeleteParameterGroup"},
		{Name: "CreateSnapshot", Method: http.MethodPost, IAMAction: "memorydb:CreateSnapshot"},
		{Name: "DescribeSnapshots", Method: http.MethodPost, IAMAction: "memorydb:DescribeSnapshots"},
		{Name: "DeleteSnapshot", Method: http.MethodPost, IAMAction: "memorydb:DeleteSnapshot"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "memorydb:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "memorydb:UntagResource"},
		{Name: "ListTags", Method: http.MethodPost, IAMAction: "memorydb:ListTags"},
	}
}

// HealthCheck always returns nil.
func (s *MemoryDBService) HealthCheck() error { return nil }

// HandleRequest routes an incoming request to the appropriate handler.
func (s *MemoryDBService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateCluster":
		return handleCreateCluster(ctx, s.store)
	case "DescribeClusters":
		return handleDescribeClusters(ctx, s.store)
	case "DeleteCluster":
		return handleDeleteCluster(ctx, s.store)
	case "UpdateCluster":
		return handleUpdateCluster(ctx, s.store)
	case "CreateACL":
		return handleCreateACL(ctx, s.store)
	case "DescribeACLs":
		return handleDescribeACLs(ctx, s.store)
	case "DeleteACL":
		return handleDeleteACL(ctx, s.store)
	case "UpdateACL":
		return handleUpdateACL(ctx, s.store)
	case "CreateUser":
		return handleCreateUser(ctx, s.store)
	case "DescribeUsers":
		return handleDescribeUsers(ctx, s.store)
	case "DeleteUser":
		return handleDeleteUser(ctx, s.store)
	case "UpdateUser":
		return handleUpdateUser(ctx, s.store)
	case "CreateSubnetGroup":
		return handleCreateSubnetGroup(ctx, s.store)
	case "DescribeSubnetGroups":
		return handleDescribeSubnetGroups(ctx, s.store)
	case "DeleteSubnetGroup":
		return handleDeleteSubnetGroup(ctx, s.store)
	case "CreateParameterGroup":
		return handleCreateParameterGroup(ctx, s.store)
	case "DescribeParameterGroups":
		return handleDescribeParameterGroups(ctx, s.store)
	case "DeleteParameterGroup":
		return handleDeleteParameterGroup(ctx, s.store)
	case "CreateSnapshot":
		return handleCreateSnapshot(ctx, s.store)
	case "DescribeSnapshots":
		return handleDescribeSnapshots(ctx, s.store)
	case "DeleteSnapshot":
		return handleDeleteSnapshot(ctx, s.store)
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
