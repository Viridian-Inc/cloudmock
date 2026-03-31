package timestreamwrite

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// TimestreamWriteService is the cloudmock implementation of the AWS Timestream Write API.
type TimestreamWriteService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new TimestreamWriteService.
func New(accountID, region string) *TimestreamWriteService {
	return &TimestreamWriteService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *TimestreamWriteService) Name() string { return "timestream-write" }

// Actions returns the list of API actions supported.
func (s *TimestreamWriteService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateDatabase", Method: http.MethodPost, IAMAction: "timestream:CreateDatabase"},
		{Name: "DescribeDatabase", Method: http.MethodPost, IAMAction: "timestream:DescribeDatabase"},
		{Name: "ListDatabases", Method: http.MethodPost, IAMAction: "timestream:ListDatabases"},
		{Name: "UpdateDatabase", Method: http.MethodPost, IAMAction: "timestream:UpdateDatabase"},
		{Name: "DeleteDatabase", Method: http.MethodPost, IAMAction: "timestream:DeleteDatabase"},
		{Name: "CreateTable", Method: http.MethodPost, IAMAction: "timestream:CreateTable"},
		{Name: "DescribeTable", Method: http.MethodPost, IAMAction: "timestream:DescribeTable"},
		{Name: "ListTables", Method: http.MethodPost, IAMAction: "timestream:ListTables"},
		{Name: "UpdateTable", Method: http.MethodPost, IAMAction: "timestream:UpdateTable"},
		{Name: "DeleteTable", Method: http.MethodPost, IAMAction: "timestream:DeleteTable"},
		{Name: "WriteRecords", Method: http.MethodPost, IAMAction: "timestream:WriteRecords"},
		{Name: "DescribeEndpoints", Method: http.MethodPost, IAMAction: "timestream:DescribeEndpoints"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "timestream:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "timestream:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "timestream:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *TimestreamWriteService) HealthCheck() error { return nil }

// HandleRequest routes an incoming request to the appropriate handler.
func (s *TimestreamWriteService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateDatabase":
		return handleCreateDatabase(ctx, s.store)
	case "DescribeDatabase":
		return handleDescribeDatabase(ctx, s.store)
	case "ListDatabases":
		return handleListDatabases(ctx, s.store)
	case "UpdateDatabase":
		return handleUpdateDatabase(ctx, s.store)
	case "DeleteDatabase":
		return handleDeleteDatabase(ctx, s.store)
	case "CreateTable":
		return handleCreateTable(ctx, s.store)
	case "DescribeTable":
		return handleDescribeTable(ctx, s.store)
	case "ListTables":
		return handleListTables(ctx, s.store)
	case "UpdateTable":
		return handleUpdateTable(ctx, s.store)
	case "DeleteTable":
		return handleDeleteTable(ctx, s.store)
	case "WriteRecords":
		return handleWriteRecords(ctx, s.store)
	case "DescribeEndpoints":
		return handleDescribeEndpoints(ctx, s.store)
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
