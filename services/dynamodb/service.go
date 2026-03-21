package dynamodb

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// DynamoDBService is the cloudmock implementation of the AWS DynamoDB API.
type DynamoDBService struct {
	store *TableStore
}

// New returns a new DynamoDBService for the given AWS account ID and region.
func New(accountID, region string) *DynamoDBService {
	return &DynamoDBService{
		store: NewTableStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *DynamoDBService) Name() string { return "dynamodb" }

// Actions returns the list of DynamoDB API actions supported by this service.
func (s *DynamoDBService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateTable", Method: http.MethodPost, IAMAction: "dynamodb:CreateTable"},
		{Name: "DeleteTable", Method: http.MethodPost, IAMAction: "dynamodb:DeleteTable"},
		{Name: "DescribeTable", Method: http.MethodPost, IAMAction: "dynamodb:DescribeTable"},
		{Name: "ListTables", Method: http.MethodPost, IAMAction: "dynamodb:ListTables"},
		{Name: "PutItem", Method: http.MethodPost, IAMAction: "dynamodb:PutItem"},
		{Name: "GetItem", Method: http.MethodPost, IAMAction: "dynamodb:GetItem"},
		{Name: "DeleteItem", Method: http.MethodPost, IAMAction: "dynamodb:DeleteItem"},
		{Name: "UpdateItem", Method: http.MethodPost, IAMAction: "dynamodb:UpdateItem"},
		{Name: "Query", Method: http.MethodPost, IAMAction: "dynamodb:Query"},
		{Name: "Scan", Method: http.MethodPost, IAMAction: "dynamodb:Scan"},
		{Name: "BatchGetItem", Method: http.MethodPost, IAMAction: "dynamodb:BatchGetItem"},
		{Name: "BatchWriteItem", Method: http.MethodPost, IAMAction: "dynamodb:BatchWriteItem"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *DynamoDBService) HealthCheck() error { return nil }

// HandleRequest routes an incoming DynamoDB request to the appropriate handler.
func (s *DynamoDBService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateTable":
		return handleCreateTable(ctx, s.store)
	case "DeleteTable":
		return handleDeleteTable(ctx, s.store)
	case "DescribeTable":
		return handleDescribeTable(ctx, s.store)
	case "ListTables":
		return handleListTables(ctx, s.store)
	case "PutItem":
		return handlePutItem(ctx, s.store)
	case "GetItem":
		return handleGetItem(ctx, s.store)
	case "DeleteItem":
		return handleDeleteItem(ctx, s.store)
	case "UpdateItem":
		return handleUpdateItem(ctx, s.store)
	case "Query":
		return handleQuery(ctx, s.store)
	case "Scan":
		return handleScan(ctx, s.store)
	case "BatchGetItem":
		return handleBatchGetItem(ctx, s.store)
	case "BatchWriteItem":
		return handleBatchWriteItem(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
