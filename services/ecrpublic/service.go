package ecrpublic

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS ecr-public service.
type Service struct {
	store *Store
}

// New returns a new ecrpublic Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "ecr-public" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "BatchCheckLayerAvailability", Method: http.MethodPost, IAMAction: "ecr-public:BatchCheckLayerAvailability"},
		{Name: "BatchDeleteImage", Method: http.MethodPost, IAMAction: "ecr-public:BatchDeleteImage"},
		{Name: "CompleteLayerUpload", Method: http.MethodPost, IAMAction: "ecr-public:CompleteLayerUpload"},
		{Name: "CreateRepository", Method: http.MethodPost, IAMAction: "ecr-public:CreateRepository"},
		{Name: "DeleteRepository", Method: http.MethodPost, IAMAction: "ecr-public:DeleteRepository"},
		{Name: "DeleteRepositoryPolicy", Method: http.MethodPost, IAMAction: "ecr-public:DeleteRepositoryPolicy"},
		{Name: "DescribeImageTags", Method: http.MethodPost, IAMAction: "ecr-public:DescribeImageTags"},
		{Name: "DescribeImages", Method: http.MethodPost, IAMAction: "ecr-public:DescribeImages"},
		{Name: "DescribeRegistries", Method: http.MethodPost, IAMAction: "ecr-public:DescribeRegistries"},
		{Name: "DescribeRepositories", Method: http.MethodPost, IAMAction: "ecr-public:DescribeRepositories"},
		{Name: "GetAuthorizationToken", Method: http.MethodPost, IAMAction: "ecr-public:GetAuthorizationToken"},
		{Name: "GetRegistryCatalogData", Method: http.MethodPost, IAMAction: "ecr-public:GetRegistryCatalogData"},
		{Name: "GetRepositoryCatalogData", Method: http.MethodPost, IAMAction: "ecr-public:GetRepositoryCatalogData"},
		{Name: "GetRepositoryPolicy", Method: http.MethodPost, IAMAction: "ecr-public:GetRepositoryPolicy"},
		{Name: "InitiateLayerUpload", Method: http.MethodPost, IAMAction: "ecr-public:InitiateLayerUpload"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "ecr-public:ListTagsForResource"},
		{Name: "PutImage", Method: http.MethodPost, IAMAction: "ecr-public:PutImage"},
		{Name: "PutRegistryCatalogData", Method: http.MethodPost, IAMAction: "ecr-public:PutRegistryCatalogData"},
		{Name: "PutRepositoryCatalogData", Method: http.MethodPost, IAMAction: "ecr-public:PutRepositoryCatalogData"},
		{Name: "SetRepositoryPolicy", Method: http.MethodPost, IAMAction: "ecr-public:SetRepositoryPolicy"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "ecr-public:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "ecr-public:UntagResource"},
		{Name: "UploadLayerPart", Method: http.MethodPost, IAMAction: "ecr-public:UploadLayerPart"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "BatchCheckLayerAvailability":
		return handleBatchCheckLayerAvailability(ctx, s.store)
	case "BatchDeleteImage":
		return handleBatchDeleteImage(ctx, s.store)
	case "CompleteLayerUpload":
		return handleCompleteLayerUpload(ctx, s.store)
	case "CreateRepository":
		return handleCreateRepository(ctx, s.store)
	case "DeleteRepository":
		return handleDeleteRepository(ctx, s.store)
	case "DeleteRepositoryPolicy":
		return handleDeleteRepositoryPolicy(ctx, s.store)
	case "DescribeImageTags":
		return handleDescribeImageTags(ctx, s.store)
	case "DescribeImages":
		return handleDescribeImages(ctx, s.store)
	case "DescribeRegistries":
		return handleDescribeRegistries(ctx, s.store)
	case "DescribeRepositories":
		return handleDescribeRepositories(ctx, s.store)
	case "GetAuthorizationToken":
		return handleGetAuthorizationToken(ctx, s.store)
	case "GetRegistryCatalogData":
		return handleGetRegistryCatalogData(ctx, s.store)
	case "GetRepositoryCatalogData":
		return handleGetRepositoryCatalogData(ctx, s.store)
	case "GetRepositoryPolicy":
		return handleGetRepositoryPolicy(ctx, s.store)
	case "InitiateLayerUpload":
		return handleInitiateLayerUpload(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "PutImage":
		return handlePutImage(ctx, s.store)
	case "PutRegistryCatalogData":
		return handlePutRegistryCatalogData(ctx, s.store)
	case "PutRepositoryCatalogData":
		return handlePutRepositoryCatalogData(ctx, s.store)
	case "SetRepositoryPolicy":
		return handleSetRepositoryPolicy(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UploadLayerPart":
		return handleUploadLayerPart(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
