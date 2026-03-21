package dynamodb

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/schema"
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
		{Name: "TransactWriteItems", Method: http.MethodPost, IAMAction: "dynamodb:TransactWriteItems"},
		{Name: "TransactGetItems", Method: http.MethodPost, IAMAction: "dynamodb:TransactGetItems"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *DynamoDBService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for DynamoDB table resources.
func (s *DynamoDBService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "dynamodb",
			ResourceType:  "aws_dynamodb_table",
			TerraformType: "cloudmock_dynamodb_table",
			AWSType:       "AWS::DynamoDB::Table",
			CreateAction:  "CreateTable",
			ReadAction:    "DescribeTable",
			DeleteAction:  "DeleteTable",
			ListAction:    "ListTables",
			ImportID:      "table_name",
			Attributes: []schema.AttributeSchema{
				{Name: "table_name", Type: "string", Required: true, ForceNew: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "billing_mode", Type: "string", Default: "PROVISIONED"},
				{Name: "read_capacity", Type: "int"},
				{Name: "write_capacity", Type: "int"},
				{Name: "hash_key", Type: "string", Required: true, ForceNew: true},
				{Name: "range_key", Type: "string", ForceNew: true},
				{Name: "attribute", Type: "set", Required: true},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

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
	case "TransactWriteItems":
		return handleTransactWriteItems(ctx, s.store)
	case "TransactGetItems":
		return handleTransactGetItems(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
