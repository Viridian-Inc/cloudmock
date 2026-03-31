package codecommit

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator resolves other services for cross-service integration.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// CodeCommitService is the cloudmock implementation of the AWS CodeCommit API.
type CodeCommitService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new CodeCommitService for the given AWS account ID and region.
func New(accountID, region string) *CodeCommitService {
	return &CodeCommitService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new CodeCommitService with a ServiceLocator.
func NewWithLocator(accountID, region string, locator ServiceLocator) *CodeCommitService {
	return &CodeCommitService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service calls.
func (s *CodeCommitService) SetLocator(locator ServiceLocator) { s.locator = locator }

// Name returns the AWS service name used for routing.
func (s *CodeCommitService) Name() string { return "codecommit" }

// Actions returns the list of CodeCommit API actions supported.
func (s *CodeCommitService) Actions() []service.Action {
	return []service.Action{
		// Repositories
		{Name: "CreateRepository", Method: http.MethodPost, IAMAction: "codecommit:CreateRepository"},
		{Name: "GetRepository", Method: http.MethodPost, IAMAction: "codecommit:GetRepository"},
		{Name: "ListRepositories", Method: http.MethodPost, IAMAction: "codecommit:ListRepositories"},
		{Name: "DeleteRepository", Method: http.MethodPost, IAMAction: "codecommit:DeleteRepository"},
		{Name: "UpdateRepositoryName", Method: http.MethodPost, IAMAction: "codecommit:UpdateRepositoryName"},
		{Name: "UpdateRepositoryDescription", Method: http.MethodPost, IAMAction: "codecommit:UpdateRepositoryDescription"},
		// Branches
		{Name: "CreateBranch", Method: http.MethodPost, IAMAction: "codecommit:CreateBranch"},
		{Name: "GetBranch", Method: http.MethodPost, IAMAction: "codecommit:GetBranch"},
		{Name: "ListBranches", Method: http.MethodPost, IAMAction: "codecommit:ListBranches"},
		{Name: "DeleteBranch", Method: http.MethodPost, IAMAction: "codecommit:DeleteBranch"},
		// Pull Requests
		{Name: "CreatePullRequest", Method: http.MethodPost, IAMAction: "codecommit:CreatePullRequest"},
		{Name: "GetPullRequest", Method: http.MethodPost, IAMAction: "codecommit:GetPullRequest"},
		{Name: "ListPullRequests", Method: http.MethodPost, IAMAction: "codecommit:ListPullRequests"},
		{Name: "UpdatePullRequestStatus", Method: http.MethodPost, IAMAction: "codecommit:UpdatePullRequestStatus"},
		{Name: "MergePullRequestBySquash", Method: http.MethodPost, IAMAction: "codecommit:MergePullRequestBySquash"},
		// Commits & Diffs
		{Name: "GetCommit", Method: http.MethodPost, IAMAction: "codecommit:GetCommit"},
		{Name: "GetDifferences", Method: http.MethodPost, IAMAction: "codecommit:GetDifferences"},
	}
}

// HealthCheck always returns nil.
func (s *CodeCommitService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CodeCommit request to the appropriate handler.
func (s *CodeCommitService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	// Repositories
	case "CreateRepository":
		return handleCreateRepository(ctx, s.store)
	case "GetRepository":
		return handleGetRepository(ctx, s.store)
	case "ListRepositories":
		return handleListRepositories(ctx, s.store)
	case "DeleteRepository":
		return handleDeleteRepository(ctx, s.store)
	case "UpdateRepositoryName":
		return handleUpdateRepositoryName(ctx, s.store)
	case "UpdateRepositoryDescription":
		return handleUpdateRepositoryDescription(ctx, s.store)
	// Branches
	case "CreateBranch":
		return handleCreateBranch(ctx, s.store)
	case "GetBranch":
		return handleGetBranch(ctx, s.store)
	case "ListBranches":
		return handleListBranches(ctx, s.store)
	case "DeleteBranch":
		return handleDeleteBranch(ctx, s.store)
	// Pull Requests
	case "CreatePullRequest":
		return handleCreatePullRequest(ctx, s.store)
	case "GetPullRequest":
		return handleGetPullRequest(ctx, s.store)
	case "ListPullRequests":
		return handleListPullRequests(ctx, s.store)
	case "UpdatePullRequestStatus":
		return handleUpdatePullRequestStatus(ctx, s.store)
	case "MergePullRequestBySquash":
		return handleMergePullRequestBySquash(ctx, s.store)
	// Commits & Diffs
	case "GetCommit":
		return handleGetCommit(ctx, s.store)
	case "GetDifferences":
		return handleGetDifferences(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
