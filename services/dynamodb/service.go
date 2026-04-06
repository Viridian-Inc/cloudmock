package dynamodb

import (
	"net/http"
	"os"

	"github.com/neureaux/cloudmock/pkg/rustddb"
	"github.com/neureaux/cloudmock/pkg/schema"
	"github.com/neureaux/cloudmock/pkg/service"
)

// DynamoDBService is the cloudmock implementation of the AWS DynamoDB API.
type DynamoDBService struct {
	store     *TableStore
	rustStore *rustddb.Store // nil unless CLOUDMOCK_RUST_DDB=true
	ttlDone   chan struct{}
}

// New returns a new DynamoDBService for the given AWS account ID and region.
func New(accountID, region string) *DynamoDBService {
	store := NewTableStore(accountID, region)
	done := make(chan struct{})
	store.startTTLReaper(done)

	svc := &DynamoDBService{
		store:   store,
		ttlDone: done,
	}

	// Enable Rust-accelerated store for hot-path operations.
	if os.Getenv("CLOUDMOCK_RUST_DDB") == "true" || os.Getenv("CLOUDMOCK_TEST_MODE") == "true" {
		svc.rustStore = rustddb.New(accountID, region)
	}

	return svc
}

// Name returns the AWS service name used for routing.
func (s *DynamoDBService) Name() string { return "dynamodb" }

// Actions returns the list of DynamoDB API actions supported by this service.
func (s *DynamoDBService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateTable", Method: http.MethodPost, IAMAction: "dynamodb:CreateTable"},
		{Name: "DeleteTable", Method: http.MethodPost, IAMAction: "dynamodb:DeleteTable"},
		{Name: "DescribeTable", Method: http.MethodPost, IAMAction: "dynamodb:DescribeTable"},
		{Name: "UpdateTable", Method: http.MethodPost, IAMAction: "dynamodb:UpdateTable"},
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
		{Name: "DescribeStream", Method: http.MethodPost, IAMAction: "dynamodb:DescribeStream"},
		{Name: "GetShardIterator", Method: http.MethodPost, IAMAction: "dynamodb:GetShardIterator"},
		{Name: "GetRecords", Method: http.MethodPost, IAMAction: "dynamodb:GetRecords"},
		{Name: "UpdateTimeToLive", Method: http.MethodPost, IAMAction: "dynamodb:UpdateTimeToLive"},
		{Name: "DescribeTimeToLive", Method: http.MethodPost, IAMAction: "dynamodb:DescribeTimeToLive"},
		{Name: "PutResourcePolicy", Method: http.MethodPost, IAMAction: "dynamodb:PutResourcePolicy"},
		{Name: "GetResourcePolicy", Method: http.MethodPost, IAMAction: "dynamodb:GetResourcePolicy"},
		{Name: "DeleteResourcePolicy", Method: http.MethodPost, IAMAction: "dynamodb:DeleteResourcePolicy"},
		{Name: "DescribeContinuousBackups", Method: http.MethodPost, IAMAction: "dynamodb:DescribeContinuousBackups"},
		{Name: "UpdateContinuousBackups", Method: http.MethodPost, IAMAction: "dynamodb:UpdateContinuousBackups"},
		{Name: "ListTagsOfResource", Method: http.MethodPost, IAMAction: "dynamodb:ListTagsOfResource"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "dynamodb:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "dynamodb:UntagResource"},
		// PartiQL
		{Name: "ExecuteStatement", Method: http.MethodPost, IAMAction: "dynamodb:PartiQLSelect"},
		// Backups
		{Name: "CreateBackup", Method: http.MethodPost, IAMAction: "dynamodb:CreateBackup"},
		{Name: "ListBackups", Method: http.MethodPost, IAMAction: "dynamodb:ListBackups"},
		{Name: "DescribeBackup", Method: http.MethodPost, IAMAction: "dynamodb:DescribeBackup"},
		// Global Tables
		{Name: "CreateGlobalTable", Method: http.MethodPost, IAMAction: "dynamodb:CreateGlobalTable"},
		{Name: "DescribeGlobalTable", Method: http.MethodPost, IAMAction: "dynamodb:DescribeGlobalTable"},
		{Name: "ListGlobalTables", Method: http.MethodPost, IAMAction: "dynamodb:ListGlobalTables"},
		// Exports
		{Name: "ExportTableToPointInTime", Method: http.MethodPost, IAMAction: "dynamodb:ExportTableToPointInTime"},
		{Name: "DescribeExport", Method: http.MethodPost, IAMAction: "dynamodb:DescribeExport"},
		{Name: "ListExports", Method: http.MethodPost, IAMAction: "dynamodb:ListExports"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *DynamoDBService) HealthCheck() error { return nil }

// GetTableNames returns all table names for topology queries.
func (s *DynamoDBService) GetTableNames() []string {
	return s.store.ListTables()
}

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
	// Rust fast path: try Rust store first for hot-path operations.
	if s.rustStore != nil {
		r := s.rustStore.Handle(ctx.Action, ctx.Body)
		if r.Status != 0 {
			return &service.Response{
				StatusCode:     r.Status,
				RawBody:        r.Body,
				RawContentType: "application/x-amz-json-1.0",
			}, nil
		}
		// status=0 means Rust doesn't handle this action — fall through to Go.
	}

	switch ctx.Action {
	case "CreateTable":
		return handleCreateTable(ctx, s.store)
	case "DeleteTable":
		return handleDeleteTable(ctx, s.store)
	case "DescribeTable":
		return handleDescribeTable(ctx, s.store)
	case "UpdateTable":
		return handleUpdateTable(ctx, s.store)
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
	case "DescribeStream":
		return handleDescribeStream(ctx, s.store)
	case "GetShardIterator":
		return handleGetShardIterator(ctx, s.store)
	case "GetRecords":
		return handleGetRecords(ctx, s.store)
	case "UpdateTimeToLive":
		return handleUpdateTimeToLive(ctx, s.store)
	case "DescribeTimeToLive":
		return handleDescribeTimeToLive(ctx, s.store)
	case "PutResourcePolicy":
		return handlePutResourcePolicy(ctx, s.store)
	case "GetResourcePolicy":
		return handleGetResourcePolicy(ctx, s.store)
	case "DeleteResourcePolicy":
		return handleDeleteResourcePolicy(ctx, s.store)
	case "DescribeContinuousBackups":
		return handleDescribeContinuousBackups(ctx, s.store)
	case "UpdateContinuousBackups":
		return handleUpdateContinuousBackups(ctx, s.store)
	case "ListTagsOfResource":
		return handleListTagsOfResource(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	// PartiQL
	case "ExecuteStatement":
		return handleExecuteStatement(ctx, s.store)
	// Backups
	case "CreateBackup":
		return handleCreateBackup(ctx, s.store)
	case "ListBackups":
		return handleListBackups(ctx, s.store)
	case "DescribeBackup":
		return handleDescribeBackup(ctx, s.store)
	// Global Tables
	case "CreateGlobalTable":
		return handleCreateGlobalTable(ctx, s.store)
	case "DescribeGlobalTable":
		return handleDescribeGlobalTable(ctx, s.store)
	case "ListGlobalTables":
		return handleListGlobalTables(ctx, s.store)
	// Exports
	case "ExportTableToPointInTime":
		return handleExportTableToPointInTime(ctx, s.store)
	case "DescribeExport":
		return handleDescribeExport(ctx, s.store)
	case "ListExports":
		return handleListExports(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
