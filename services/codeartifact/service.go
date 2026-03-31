package codeartifact

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator resolves other services for cross-service integration.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// CodeArtifactService is the cloudmock implementation of the AWS CodeArtifact API.
// CodeArtifact uses the rest-json protocol.
type CodeArtifactService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new CodeArtifactService for the given AWS account ID and region.
func New(accountID, region string) *CodeArtifactService {
	return &CodeArtifactService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new CodeArtifactService with a ServiceLocator.
func NewWithLocator(accountID, region string, locator ServiceLocator) *CodeArtifactService {
	return &CodeArtifactService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service calls.
func (s *CodeArtifactService) SetLocator(locator ServiceLocator) { s.locator = locator }

// Name returns the AWS service name used for routing.
func (s *CodeArtifactService) Name() string { return "codeartifact" }

// Actions returns the list of CodeArtifact API actions supported.
func (s *CodeArtifactService) Actions() []service.Action {
	return []service.Action{
		// Domains
		{Name: "CreateDomain", Method: http.MethodPost, IAMAction: "codeartifact:CreateDomain"},
		{Name: "DescribeDomain", Method: http.MethodPost, IAMAction: "codeartifact:DescribeDomain"},
		{Name: "ListDomains", Method: http.MethodPost, IAMAction: "codeartifact:ListDomains"},
		{Name: "DeleteDomain", Method: http.MethodPost, IAMAction: "codeartifact:DeleteDomain"},
		// Repositories
		{Name: "CreateRepository", Method: http.MethodPost, IAMAction: "codeartifact:CreateRepository"},
		{Name: "DescribeRepository", Method: http.MethodPost, IAMAction: "codeartifact:DescribeRepository"},
		{Name: "ListRepositories", Method: http.MethodPost, IAMAction: "codeartifact:ListRepositories"},
		{Name: "DeleteRepository", Method: http.MethodPost, IAMAction: "codeartifact:DeleteRepository"},
		// Packages
		{Name: "DescribePackage", Method: http.MethodPost, IAMAction: "codeartifact:DescribePackage"},
		{Name: "ListPackages", Method: http.MethodPost, IAMAction: "codeartifact:ListPackages"},
		{Name: "ListPackageVersions", Method: http.MethodPost, IAMAction: "codeartifact:ListPackageVersions"},
		{Name: "DescribePackageVersion", Method: http.MethodPost, IAMAction: "codeartifact:DescribePackageVersion"},
		// Endpoints & Auth
		{Name: "GetRepositoryEndpoint", Method: http.MethodPost, IAMAction: "codeartifact:GetRepositoryEndpoint"},
		{Name: "GetAuthorizationToken", Method: http.MethodPost, IAMAction: "codeartifact:GetAuthorizationToken"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "codeartifact:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "codeartifact:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "codeartifact:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *CodeArtifactService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CodeArtifact request to the appropriate handler.
func (s *CodeArtifactService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	// Domains
	case "CreateDomain":
		return handleCreateDomain(ctx, s.store)
	case "DescribeDomain":
		return handleDescribeDomain(ctx, s.store)
	case "ListDomains":
		return handleListDomains(ctx, s.store)
	case "DeleteDomain":
		return handleDeleteDomain(ctx, s.store)
	// Repositories
	case "CreateRepository":
		return handleCreateRepository(ctx, s.store)
	case "DescribeRepository":
		return handleDescribeRepository(ctx, s.store)
	case "ListRepositories":
		return handleListRepositories(ctx, s.store)
	case "DeleteRepository":
		return handleDeleteRepository(ctx, s.store)
	// Packages
	case "DescribePackage":
		return handleDescribePackage(ctx, s.store)
	case "ListPackages":
		return handleListPackages(ctx, s.store)
	case "ListPackageVersions":
		return handleListPackageVersions(ctx, s.store)
	case "DescribePackageVersion":
		return handleDescribePackageVersion(ctx, s.store)
	// Endpoints & Auth
	case "GetRepositoryEndpoint":
		return handleGetRepositoryEndpoint(ctx, s.store)
	case "GetAuthorizationToken":
		return handleGetAuthorizationToken(ctx, s.store)
	// Tags
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
