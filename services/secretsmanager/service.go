package secretsmanager

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// SecretsManagerService is the cloudmock implementation of the AWS Secrets Manager API.
type SecretsManagerService struct {
	store *Store
}

// New returns a new SecretsManagerService for the given AWS account ID and region.
func New(accountID, region string) *SecretsManagerService {
	return &SecretsManagerService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *SecretsManagerService) Name() string { return "secretsmanager" }

// Actions returns the list of Secrets Manager API actions supported by this service.
func (s *SecretsManagerService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateSecret", Method: http.MethodPost, IAMAction: "secretsmanager:CreateSecret"},
		{Name: "GetSecretValue", Method: http.MethodPost, IAMAction: "secretsmanager:GetSecretValue"},
		{Name: "PutSecretValue", Method: http.MethodPost, IAMAction: "secretsmanager:PutSecretValue"},
		{Name: "UpdateSecret", Method: http.MethodPost, IAMAction: "secretsmanager:UpdateSecret"},
		{Name: "DeleteSecret", Method: http.MethodPost, IAMAction: "secretsmanager:DeleteSecret"},
		{Name: "RestoreSecret", Method: http.MethodPost, IAMAction: "secretsmanager:RestoreSecret"},
		{Name: "DescribeSecret", Method: http.MethodPost, IAMAction: "secretsmanager:DescribeSecret"},
		{Name: "ListSecrets", Method: http.MethodPost, IAMAction: "secretsmanager:ListSecrets"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "secretsmanager:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "secretsmanager:UntagResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *SecretsManagerService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Secrets Manager request to the appropriate handler.
// Secrets Manager uses the JSON protocol; the action is parsed from X-Amz-Target by the gateway
// and placed in ctx.Action (e.g. "CreateSecret").
func (s *SecretsManagerService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateSecret":
		return handleCreateSecret(ctx, s.store)
	case "GetSecretValue":
		return handleGetSecretValue(ctx, s.store)
	case "PutSecretValue":
		return handlePutSecretValue(ctx, s.store)
	case "UpdateSecret":
		return handleUpdateSecret(ctx, s.store)
	case "DeleteSecret":
		return handleDeleteSecret(ctx, s.store)
	case "RestoreSecret":
		return handleRestoreSecret(ctx, s.store)
	case "DescribeSecret":
		return handleDescribeSecret(ctx, s.store)
	case "ListSecrets":
		return handleListSecrets(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
