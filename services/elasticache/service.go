package elasticache

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// ElastiCacheService is the cloudmock implementation of the AWS ElastiCache API.
type ElastiCacheService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new ElastiCacheService for the given AWS account ID and region.
func New(accountID, region string) *ElastiCacheService {
	return &ElastiCacheService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new ElastiCacheService with a service locator for cross-service communication.
func NewWithLocator(accountID, region string, locator ServiceLocator) *ElastiCacheService {
	return &ElastiCacheService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service communication.
func (s *ElastiCacheService) SetLocator(locator ServiceLocator) {
	s.locator = locator
}

// Name returns the AWS service name used for routing.
func (s *ElastiCacheService) Name() string { return "elasticache" }

// Actions returns the list of ElastiCache API actions supported by this service.
func (s *ElastiCacheService) Actions() []service.Action {
	return []service.Action{
		// Cache Clusters
		{Name: "CreateCacheCluster", Method: http.MethodPost, IAMAction: "elasticache:CreateCacheCluster"},
		{Name: "DescribeCacheClusters", Method: http.MethodPost, IAMAction: "elasticache:DescribeCacheClusters"},
		{Name: "DeleteCacheCluster", Method: http.MethodPost, IAMAction: "elasticache:DeleteCacheCluster"},
		{Name: "ModifyCacheCluster", Method: http.MethodPost, IAMAction: "elasticache:ModifyCacheCluster"},
		// Replication Groups
		{Name: "CreateReplicationGroup", Method: http.MethodPost, IAMAction: "elasticache:CreateReplicationGroup"},
		{Name: "DescribeReplicationGroups", Method: http.MethodPost, IAMAction: "elasticache:DescribeReplicationGroups"},
		{Name: "DeleteReplicationGroup", Method: http.MethodPost, IAMAction: "elasticache:DeleteReplicationGroup"},
		{Name: "ModifyReplicationGroup", Method: http.MethodPost, IAMAction: "elasticache:ModifyReplicationGroup"},
		// Subnet Groups
		{Name: "CreateCacheSubnetGroup", Method: http.MethodPost, IAMAction: "elasticache:CreateCacheSubnetGroup"},
		{Name: "DescribeCacheSubnetGroups", Method: http.MethodPost, IAMAction: "elasticache:DescribeCacheSubnetGroups"},
		{Name: "DeleteCacheSubnetGroup", Method: http.MethodPost, IAMAction: "elasticache:DeleteCacheSubnetGroup"},
		// Parameter Groups
		{Name: "CreateCacheParameterGroup", Method: http.MethodPost, IAMAction: "elasticache:CreateCacheParameterGroup"},
		{Name: "DescribeCacheParameterGroups", Method: http.MethodPost, IAMAction: "elasticache:DescribeCacheParameterGroups"},
		{Name: "DeleteCacheParameterGroup", Method: http.MethodPost, IAMAction: "elasticache:DeleteCacheParameterGroup"},
		// Failover
		{Name: "TestFailover", Method: http.MethodPost, IAMAction: "elasticache:TestFailover"},
		// Snapshots
		{Name: "CreateSnapshot", Method: http.MethodPost, IAMAction: "elasticache:CreateSnapshot"},
		{Name: "DescribeSnapshots", Method: http.MethodPost, IAMAction: "elasticache:DescribeSnapshots"},
		{Name: "DeleteSnapshot", Method: http.MethodPost, IAMAction: "elasticache:DeleteSnapshot"},
		// Tags
		{Name: "AddTagsToResource", Method: http.MethodPost, IAMAction: "elasticache:AddTagsToResource"},
		{Name: "RemoveTagsFromResource", Method: http.MethodPost, IAMAction: "elasticache:RemoveTagsFromResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "elasticache:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *ElastiCacheService) HealthCheck() error { return nil }

// HandleRequest routes an incoming ElastiCache request to the appropriate handler.
// ElastiCache uses the query protocol (form-encoded, XML responses).
func (s *ElastiCacheService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	// Cache Clusters
	case "CreateCacheCluster":
		return handleCreateCacheCluster(ctx, s.store)
	case "DescribeCacheClusters":
		return handleDescribeCacheClusters(ctx, s.store)
	case "DeleteCacheCluster":
		return handleDeleteCacheCluster(ctx, s.store)
	case "ModifyCacheCluster":
		return handleModifyCacheCluster(ctx, s.store)
	// Replication Groups
	case "CreateReplicationGroup":
		return handleCreateReplicationGroup(ctx, s.store)
	case "DescribeReplicationGroups":
		return handleDescribeReplicationGroups(ctx, s.store)
	case "DeleteReplicationGroup":
		return handleDeleteReplicationGroup(ctx, s.store)
	case "ModifyReplicationGroup":
		return handleModifyReplicationGroup(ctx, s.store)
	// Subnet Groups
	case "CreateCacheSubnetGroup":
		return handleCreateCacheSubnetGroup(ctx, s.store)
	case "DescribeCacheSubnetGroups":
		return handleDescribeCacheSubnetGroups(ctx, s.store)
	case "DeleteCacheSubnetGroup":
		return handleDeleteCacheSubnetGroup(ctx, s.store)
	// Parameter Groups
	case "CreateCacheParameterGroup":
		return handleCreateCacheParameterGroup(ctx, s.store)
	case "DescribeCacheParameterGroups":
		return handleDescribeCacheParameterGroups(ctx, s.store)
	case "DeleteCacheParameterGroup":
		return handleDeleteCacheParameterGroup(ctx, s.store)
	// Failover
	case "TestFailover":
		return handleTestFailover(ctx, s.store)
	// Snapshots
	case "CreateSnapshot":
		return handleCreateSnapshot(ctx, s.store)
	case "DescribeSnapshots":
		return handleDescribeSnapshots(ctx, s.store)
	case "DeleteSnapshot":
		return handleDeleteSnapshot(ctx, s.store)
	// Tags
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
