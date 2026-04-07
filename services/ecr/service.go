package ecr

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ECRService is the cloudmock implementation of the AWS Elastic Container Registry API.
type ECRService struct {
	store *Store
}

// New returns a new ECRService for the given AWS account ID and region.
func New(accountID, region string) *ECRService {
	return &ECRService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
// The Authorization header credential scope for ECR is "ecr".
func (s *ECRService) Name() string { return "ecr" }

// Actions returns the list of ECR API actions supported by this service.
func (s *ECRService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateRepository", Method: http.MethodPost, IAMAction: "ecr:CreateRepository"},
		{Name: "DeleteRepository", Method: http.MethodPost, IAMAction: "ecr:DeleteRepository"},
		{Name: "DescribeRepositories", Method: http.MethodPost, IAMAction: "ecr:DescribeRepositories"},
		{Name: "ListImages", Method: http.MethodPost, IAMAction: "ecr:ListImages"},
		{Name: "BatchGetImage", Method: http.MethodPost, IAMAction: "ecr:BatchGetImage"},
		{Name: "PutImage", Method: http.MethodPost, IAMAction: "ecr:PutImage"},
		{Name: "BatchDeleteImage", Method: http.MethodPost, IAMAction: "ecr:BatchDeleteImage"},
		{Name: "GetAuthorizationToken", Method: http.MethodPost, IAMAction: "ecr:GetAuthorizationToken"},
		{Name: "DescribeImageScanFindings", Method: http.MethodPost, IAMAction: "ecr:DescribeImageScanFindings"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "ecr:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "ecr:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "ecr:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *ECRService) HealthCheck() error { return nil }

// HandleRequest routes an incoming ECR request to the appropriate handler.
// ECR uses the JSON protocol; the action is parsed from X-Amz-Target by the gateway
// and placed in ctx.Action (e.g. "CreateRepository").
func (s *ECRService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateRepository":
		return handleCreateRepository(ctx, s.store)
	case "DeleteRepository":
		return handleDeleteRepository(ctx, s.store)
	case "DescribeRepositories":
		return handleDescribeRepositories(ctx, s.store)
	case "ListImages":
		return handleListImages(ctx, s.store)
	case "BatchGetImage":
		return handleBatchGetImage(ctx, s.store)
	case "PutImage":
		return handlePutImage(ctx, s.store)
	case "BatchDeleteImage":
		return handleBatchDeleteImage(ctx, s.store)
	case "GetAuthorizationToken":
		return handleGetAuthorizationToken(ctx, s.store)
	case "DescribeImageScanFindings":
		return handleDescribeImageScanFindings(ctx, s.store)
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
