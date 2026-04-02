package athena

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// AthenaService is the cloudmock implementation of the AWS Athena API.
type AthenaService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new AthenaService for the given AWS account ID and region.
func New(accountID, region string) *AthenaService {
	return &AthenaService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// SetLocator sets the service locator for cross-service lookups (e.g., Glue catalog).
func (s *AthenaService) SetLocator(locator ServiceLocator) {
	s.store.SetLocator(locator)
}

// RegisterSchema registers a table schema for SQL validation.
func (s *AthenaService) RegisterSchema(database, table string, columns []string) {
	s.store.schemaRegistry.Register(database, table, columns)
}

// Name returns the AWS service name used for routing.
func (s *AthenaService) Name() string { return "athena" }

// Actions returns the list of Athena API actions supported by this service.
func (s *AthenaService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateWorkGroup", Method: http.MethodPost, IAMAction: "athena:CreateWorkGroup"},
		{Name: "GetWorkGroup", Method: http.MethodPost, IAMAction: "athena:GetWorkGroup"},
		{Name: "ListWorkGroups", Method: http.MethodPost, IAMAction: "athena:ListWorkGroups"},
		{Name: "DeleteWorkGroup", Method: http.MethodPost, IAMAction: "athena:DeleteWorkGroup"},
		{Name: "UpdateWorkGroup", Method: http.MethodPost, IAMAction: "athena:UpdateWorkGroup"},
		{Name: "CreateNamedQuery", Method: http.MethodPost, IAMAction: "athena:CreateNamedQuery"},
		{Name: "GetNamedQuery", Method: http.MethodPost, IAMAction: "athena:GetNamedQuery"},
		{Name: "ListNamedQueries", Method: http.MethodPost, IAMAction: "athena:ListNamedQueries"},
		{Name: "DeleteNamedQuery", Method: http.MethodPost, IAMAction: "athena:DeleteNamedQuery"},
		{Name: "StartQueryExecution", Method: http.MethodPost, IAMAction: "athena:StartQueryExecution"},
		{Name: "GetQueryExecution", Method: http.MethodPost, IAMAction: "athena:GetQueryExecution"},
		{Name: "ListQueryExecutions", Method: http.MethodPost, IAMAction: "athena:ListQueryExecutions"},
		{Name: "StopQueryExecution", Method: http.MethodPost, IAMAction: "athena:StopQueryExecution"},
		{Name: "GetQueryResults", Method: http.MethodPost, IAMAction: "athena:GetQueryResults"},
		{Name: "BatchGetNamedQuery", Method: http.MethodPost, IAMAction: "athena:BatchGetNamedQuery"},
		{Name: "BatchGetQueryExecution", Method: http.MethodPost, IAMAction: "athena:BatchGetQueryExecution"},
		{Name: "CreateDataCatalog", Method: http.MethodPost, IAMAction: "athena:CreateDataCatalog"},
		{Name: "GetDataCatalog", Method: http.MethodPost, IAMAction: "athena:GetDataCatalog"},
		{Name: "ListDataCatalogs", Method: http.MethodPost, IAMAction: "athena:ListDataCatalogs"},
		{Name: "UpdateDataCatalog", Method: http.MethodPost, IAMAction: "athena:UpdateDataCatalog"},
		{Name: "DeleteDataCatalog", Method: http.MethodPost, IAMAction: "athena:DeleteDataCatalog"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "athena:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "athena:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "athena:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *AthenaService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Athena request to the appropriate handler.
func (s *AthenaService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateWorkGroup":
		return handleCreateWorkGroup(ctx, s.store)
	case "GetWorkGroup":
		return handleGetWorkGroup(ctx, s.store)
	case "ListWorkGroups":
		return handleListWorkGroups(ctx, s.store)
	case "DeleteWorkGroup":
		return handleDeleteWorkGroup(ctx, s.store)
	case "UpdateWorkGroup":
		return handleUpdateWorkGroup(ctx, s.store)
	case "CreateNamedQuery":
		return handleCreateNamedQuery(ctx, s.store)
	case "GetNamedQuery":
		return handleGetNamedQuery(ctx, s.store)
	case "ListNamedQueries":
		return handleListNamedQueries(ctx, s.store)
	case "DeleteNamedQuery":
		return handleDeleteNamedQuery(ctx, s.store)
	case "StartQueryExecution":
		return handleStartQueryExecution(ctx, s.store)
	case "GetQueryExecution":
		return handleGetQueryExecution(ctx, s.store)
	case "ListQueryExecutions":
		return handleListQueryExecutions(ctx, s.store)
	case "StopQueryExecution":
		return handleStopQueryExecution(ctx, s.store)
	case "GetQueryResults":
		return handleGetQueryResults(ctx, s.store)
	case "BatchGetNamedQuery":
		return handleBatchGetNamedQuery(ctx, s.store)
	case "BatchGetQueryExecution":
		return handleBatchGetQueryExecution(ctx, s.store)
	case "CreateDataCatalog":
		return handleCreateDataCatalog(ctx, s.store)
	case "GetDataCatalog":
		return handleGetDataCatalog(ctx, s.store)
	case "ListDataCatalogs":
		return handleListDataCatalogs(ctx, s.store)
	case "UpdateDataCatalog":
		return handleUpdateDataCatalog(ctx, s.store)
	case "DeleteDataCatalog":
		return handleDeleteDataCatalog(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
