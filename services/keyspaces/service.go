package keyspaces

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS cassandra service.
type Service struct {
	store *Store
}

// New returns a new keyspaces Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "cassandra" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateKeyspace", Method: http.MethodPost, IAMAction: "cassandra:CreateKeyspace"},
		{Name: "CreateTable", Method: http.MethodPost, IAMAction: "cassandra:CreateTable"},
		{Name: "CreateType", Method: http.MethodPost, IAMAction: "cassandra:CreateType"},
		{Name: "DeleteKeyspace", Method: http.MethodPost, IAMAction: "cassandra:DeleteKeyspace"},
		{Name: "DeleteTable", Method: http.MethodPost, IAMAction: "cassandra:DeleteTable"},
		{Name: "DeleteType", Method: http.MethodPost, IAMAction: "cassandra:DeleteType"},
		{Name: "GetKeyspace", Method: http.MethodPost, IAMAction: "cassandra:GetKeyspace"},
		{Name: "GetTable", Method: http.MethodPost, IAMAction: "cassandra:GetTable"},
		{Name: "GetTableAutoScalingSettings", Method: http.MethodPost, IAMAction: "cassandra:GetTableAutoScalingSettings"},
		{Name: "GetType", Method: http.MethodPost, IAMAction: "cassandra:GetType"},
		{Name: "ListKeyspaces", Method: http.MethodPost, IAMAction: "cassandra:ListKeyspaces"},
		{Name: "ListTables", Method: http.MethodPost, IAMAction: "cassandra:ListTables"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "cassandra:ListTagsForResource"},
		{Name: "ListTypes", Method: http.MethodPost, IAMAction: "cassandra:ListTypes"},
		{Name: "RestoreTable", Method: http.MethodPost, IAMAction: "cassandra:RestoreTable"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "cassandra:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "cassandra:UntagResource"},
		{Name: "UpdateKeyspace", Method: http.MethodPost, IAMAction: "cassandra:UpdateKeyspace"},
		{Name: "UpdateTable", Method: http.MethodPost, IAMAction: "cassandra:UpdateTable"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateKeyspace":
		return handleCreateKeyspace(ctx, s.store)
	case "CreateTable":
		return handleCreateTable(ctx, s.store)
	case "CreateType":
		return handleCreateType(ctx, s.store)
	case "DeleteKeyspace":
		return handleDeleteKeyspace(ctx, s.store)
	case "DeleteTable":
		return handleDeleteTable(ctx, s.store)
	case "DeleteType":
		return handleDeleteType(ctx, s.store)
	case "GetKeyspace":
		return handleGetKeyspace(ctx, s.store)
	case "GetTable":
		return handleGetTable(ctx, s.store)
	case "GetTableAutoScalingSettings":
		return handleGetTableAutoScalingSettings(ctx, s.store)
	case "GetType":
		return handleGetType(ctx, s.store)
	case "ListKeyspaces":
		return handleListKeyspaces(ctx, s.store)
	case "ListTables":
		return handleListTables(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "ListTypes":
		return handleListTypes(ctx, s.store)
	case "RestoreTable":
		return handleRestoreTable(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateKeyspace":
		return handleUpdateKeyspace(ctx, s.store)
	case "UpdateTable":
		return handleUpdateTable(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
