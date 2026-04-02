package appsync

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// AppSyncService is the cloudmock implementation of the AWS AppSync API.
type AppSyncService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new AppSyncService for the given AWS account ID and region.
func New(accountID, region string) *AppSyncService {
	return &AppSyncService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *AppSyncService) Name() string { return "appsync" }

// Actions returns the list of AppSync API actions supported by this service.
func (s *AppSyncService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateGraphqlApi", Method: http.MethodPost, IAMAction: "appsync:CreateGraphqlApi"},
		{Name: "GetGraphqlApi", Method: http.MethodGet, IAMAction: "appsync:GetGraphqlApi"},
		{Name: "ListGraphqlApis", Method: http.MethodGet, IAMAction: "appsync:ListGraphqlApis"},
		{Name: "UpdateGraphqlApi", Method: http.MethodPost, IAMAction: "appsync:UpdateGraphqlApi"},
		{Name: "DeleteGraphqlApi", Method: http.MethodDelete, IAMAction: "appsync:DeleteGraphqlApi"},
		{Name: "CreateDataSource", Method: http.MethodPost, IAMAction: "appsync:CreateDataSource"},
		{Name: "GetDataSource", Method: http.MethodGet, IAMAction: "appsync:GetDataSource"},
		{Name: "ListDataSources", Method: http.MethodGet, IAMAction: "appsync:ListDataSources"},
		{Name: "UpdateDataSource", Method: http.MethodPost, IAMAction: "appsync:UpdateDataSource"},
		{Name: "DeleteDataSource", Method: http.MethodDelete, IAMAction: "appsync:DeleteDataSource"},
		{Name: "CreateResolver", Method: http.MethodPost, IAMAction: "appsync:CreateResolver"},
		{Name: "GetResolver", Method: http.MethodGet, IAMAction: "appsync:GetResolver"},
		{Name: "ListResolvers", Method: http.MethodGet, IAMAction: "appsync:ListResolvers"},
		{Name: "UpdateResolver", Method: http.MethodPost, IAMAction: "appsync:UpdateResolver"},
		{Name: "DeleteResolver", Method: http.MethodDelete, IAMAction: "appsync:DeleteResolver"},
		{Name: "CreateFunction", Method: http.MethodPost, IAMAction: "appsync:CreateFunction"},
		{Name: "GetFunction", Method: http.MethodGet, IAMAction: "appsync:GetFunction"},
		{Name: "ListFunctions", Method: http.MethodGet, IAMAction: "appsync:ListFunctions"},
		{Name: "UpdateFunction", Method: http.MethodPost, IAMAction: "appsync:UpdateFunction"},
		{Name: "DeleteFunction", Method: http.MethodDelete, IAMAction: "appsync:DeleteFunction"},
		{Name: "CreateApiKey", Method: http.MethodPost, IAMAction: "appsync:CreateApiKey"},
		{Name: "ListApiKeys", Method: http.MethodGet, IAMAction: "appsync:ListApiKeys"},
		{Name: "UpdateApiKey", Method: http.MethodPost, IAMAction: "appsync:UpdateApiKey"},
		{Name: "DeleteApiKey", Method: http.MethodDelete, IAMAction: "appsync:DeleteApiKey"},
		{Name: "CreateType", Method: http.MethodPost, IAMAction: "appsync:CreateType"},
		{Name: "GetType", Method: http.MethodGet, IAMAction: "appsync:GetType"},
		{Name: "ListTypes", Method: http.MethodGet, IAMAction: "appsync:ListTypes"},
		{Name: "UpdateType", Method: http.MethodPost, IAMAction: "appsync:UpdateType"},
		{Name: "DeleteType", Method: http.MethodDelete, IAMAction: "appsync:DeleteType"},
		{Name: "StartSchemaCreation", Method: http.MethodPost, IAMAction: "appsync:StartSchemaCreation"},
		{Name: "GetSchemaCreationStatus", Method: http.MethodGet, IAMAction: "appsync:GetSchemaCreationStatus"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "appsync:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "appsync:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "appsync:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *AppSyncService) HealthCheck() error { return nil }

// HandleRequest routes an incoming AppSync request to the appropriate handler.
func (s *AppSyncService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateGraphqlApi":
		return handleCreateGraphqlApi(ctx, s.store)
	case "GetGraphqlApi":
		return handleGetGraphqlApi(ctx, s.store)
	case "ListGraphqlApis":
		return handleListGraphqlApis(ctx, s.store)
	case "UpdateGraphqlApi":
		return handleUpdateGraphqlApi(ctx, s.store)
	case "DeleteGraphqlApi":
		return handleDeleteGraphqlApi(ctx, s.store)
	case "CreateDataSource":
		return handleCreateDataSource(ctx, s.store)
	case "GetDataSource":
		return handleGetDataSource(ctx, s.store)
	case "ListDataSources":
		return handleListDataSources(ctx, s.store)
	case "UpdateDataSource":
		return handleUpdateDataSource(ctx, s.store)
	case "DeleteDataSource":
		return handleDeleteDataSource(ctx, s.store)
	case "CreateResolver":
		return handleCreateResolver(ctx, s.store)
	case "GetResolver":
		return handleGetResolver(ctx, s.store)
	case "ListResolvers":
		return handleListResolvers(ctx, s.store)
	case "UpdateResolver":
		return handleUpdateResolver(ctx, s.store)
	case "DeleteResolver":
		return handleDeleteResolver(ctx, s.store)
	case "CreateFunction":
		return handleCreateFunction(ctx, s.store)
	case "GetFunction":
		return handleGetFunction(ctx, s.store)
	case "ListFunctions":
		return handleListFunctions(ctx, s.store)
	case "UpdateFunction":
		return handleUpdateFunction(ctx, s.store)
	case "DeleteFunction":
		return handleDeleteFunction(ctx, s.store)
	case "CreateApiKey":
		return handleCreateApiKey(ctx, s.store)
	case "ListApiKeys":
		return handleListApiKeys(ctx, s.store)
	case "UpdateApiKey":
		return handleUpdateApiKey(ctx, s.store)
	case "DeleteApiKey":
		return handleDeleteApiKey(ctx, s.store)
	case "CreateType":
		return handleCreateType(ctx, s.store)
	case "GetType":
		return handleGetType(ctx, s.store)
	case "ListTypes":
		return handleListTypes(ctx, s.store)
	case "UpdateType":
		return handleUpdateType(ctx, s.store)
	case "DeleteType":
		return handleDeleteType(ctx, s.store)
	case "StartSchemaCreation":
		return handleStartSchemaCreation(ctx, s.store)
	case "GetSchemaCreationStatus":
		return handleGetSchemaCreationStatus(ctx, s.store)
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
