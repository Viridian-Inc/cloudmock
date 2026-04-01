package glue

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// GlueService is the cloudmock implementation of the AWS Glue API.
type GlueService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new GlueService for the given AWS account ID and region.
func New(accountID, region string) *GlueService {
	return &GlueService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *GlueService) Name() string { return "glue" }

// SetLocator sets the service locator for cross-service lookups (e.g., S3 for crawlers).
func (s *GlueService) SetLocator(locator ServiceLocator) {
	s.store.SetLocator(locator)
}

// Actions returns the list of Glue API actions supported by this service.
func (s *GlueService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateDatabase", Method: http.MethodPost, IAMAction: "glue:CreateDatabase"},
		{Name: "GetDatabase", Method: http.MethodPost, IAMAction: "glue:GetDatabase"},
		{Name: "GetDatabases", Method: http.MethodPost, IAMAction: "glue:GetDatabases"},
		{Name: "DeleteDatabase", Method: http.MethodPost, IAMAction: "glue:DeleteDatabase"},
		{Name: "CreateTable", Method: http.MethodPost, IAMAction: "glue:CreateTable"},
		{Name: "GetTable", Method: http.MethodPost, IAMAction: "glue:GetTable"},
		{Name: "GetTables", Method: http.MethodPost, IAMAction: "glue:GetTables"},
		{Name: "DeleteTable", Method: http.MethodPost, IAMAction: "glue:DeleteTable"},
		{Name: "UpdateTable", Method: http.MethodPost, IAMAction: "glue:UpdateTable"},
		{Name: "CreateCrawler", Method: http.MethodPost, IAMAction: "glue:CreateCrawler"},
		{Name: "GetCrawler", Method: http.MethodPost, IAMAction: "glue:GetCrawler"},
		{Name: "GetCrawlers", Method: http.MethodPost, IAMAction: "glue:GetCrawlers"},
		{Name: "DeleteCrawler", Method: http.MethodPost, IAMAction: "glue:DeleteCrawler"},
		{Name: "StartCrawler", Method: http.MethodPost, IAMAction: "glue:StartCrawler"},
		{Name: "StopCrawler", Method: http.MethodPost, IAMAction: "glue:StopCrawler"},
		{Name: "CreateJob", Method: http.MethodPost, IAMAction: "glue:CreateJob"},
		{Name: "GetJob", Method: http.MethodPost, IAMAction: "glue:GetJob"},
		{Name: "GetJobs", Method: http.MethodPost, IAMAction: "glue:GetJobs"},
		{Name: "DeleteJob", Method: http.MethodPost, IAMAction: "glue:DeleteJob"},
		{Name: "StartJobRun", Method: http.MethodPost, IAMAction: "glue:StartJobRun"},
		{Name: "GetJobRun", Method: http.MethodPost, IAMAction: "glue:GetJobRun"},
		{Name: "GetJobRuns", Method: http.MethodPost, IAMAction: "glue:GetJobRuns"},
		{Name: "BatchStopJobRun", Method: http.MethodPost, IAMAction: "glue:BatchStopJobRun"},
		{Name: "CreateConnection", Method: http.MethodPost, IAMAction: "glue:CreateConnection"},
		{Name: "GetConnection", Method: http.MethodPost, IAMAction: "glue:GetConnection"},
		{Name: "GetConnections", Method: http.MethodPost, IAMAction: "glue:GetConnections"},
		{Name: "DeleteConnection", Method: http.MethodPost, IAMAction: "glue:DeleteConnection"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "glue:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "glue:UntagResource"},
		{Name: "GetTags", Method: http.MethodPost, IAMAction: "glue:GetTags"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *GlueService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Glue request to the appropriate handler.
func (s *GlueService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateDatabase":
		return handleCreateDatabase(ctx, s.store)
	case "GetDatabase":
		return handleGetDatabase(ctx, s.store)
	case "GetDatabases":
		return handleGetDatabases(ctx, s.store)
	case "DeleteDatabase":
		return handleDeleteDatabase(ctx, s.store)
	case "CreateTable":
		return handleCreateTable(ctx, s.store)
	case "GetTable":
		return handleGetTable(ctx, s.store)
	case "GetTables":
		return handleGetTables(ctx, s.store)
	case "DeleteTable":
		return handleDeleteTable(ctx, s.store)
	case "UpdateTable":
		return handleUpdateTable(ctx, s.store)
	case "CreateCrawler":
		return handleCreateCrawler(ctx, s.store)
	case "GetCrawler":
		return handleGetCrawler(ctx, s.store)
	case "GetCrawlers":
		return handleGetCrawlers(ctx, s.store)
	case "DeleteCrawler":
		return handleDeleteCrawler(ctx, s.store)
	case "StartCrawler":
		return handleStartCrawler(ctx, s.store)
	case "StopCrawler":
		return handleStopCrawler(ctx, s.store)
	case "CreateJob":
		return handleCreateJob(ctx, s.store)
	case "GetJob":
		return handleGetJob(ctx, s.store)
	case "GetJobs":
		return handleGetJobs(ctx, s.store)
	case "DeleteJob":
		return handleDeleteJob(ctx, s.store)
	case "StartJobRun":
		return handleStartJobRun(ctx, s.store)
	case "GetJobRun":
		return handleGetJobRun(ctx, s.store)
	case "GetJobRuns":
		return handleGetJobRuns(ctx, s.store)
	case "BatchStopJobRun":
		return handleBatchStopJobRun(ctx, s.store)
	case "CreateConnection":
		return handleCreateConnection(ctx, s.store)
	case "GetConnection":
		return handleGetConnection(ctx, s.store)
	case "GetConnections":
		return handleGetConnections(ctx, s.store)
	case "DeleteConnection":
		return handleDeleteConnection(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "GetTags":
		return handleGetTags(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
