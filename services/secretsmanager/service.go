package secretsmanager

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/schema"
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

// ResourceSchemas returns the schema for Secrets Manager resource types.
func (s *SecretsManagerService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "secretsmanager",
			ResourceType:  "aws_secretsmanager_secret",
			TerraformType: "cloudmock_secretsmanager_secret",
			AWSType:       "AWS::SecretsManager::Secret",
			CreateAction:  "CreateSecret",
			ReadAction:    "DescribeSecret",
			UpdateAction:  "UpdateSecret",
			DeleteAction:  "DeleteSecret",
			ListAction:    "ListSecrets",
			ImportID:      "arn",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "description", Type: "string"},
				{Name: "kms_key_id", Type: "string"},
				{Name: "recovery_window_in_days", Type: "int", Default: 30},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "id", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "secretsmanager",
			ResourceType:  "aws_secretsmanager_secret_version",
			TerraformType: "cloudmock_secretsmanager_secret_version",
			AWSType:       "AWS::SecretsManager::SecretTargetAttachment",
			CreateAction:  "PutSecretValue",
			ReadAction:    "GetSecretValue",
			DeleteAction:  "DeleteSecret",
			ImportID:      "secret_id",
			Attributes: []schema.AttributeSchema{
				{Name: "secret_id", Type: "string", Required: true, ForceNew: true},
				{Name: "secret_string", Type: "string"},
				{Name: "secret_binary", Type: "string"},
				{Name: "version_id", Type: "string", Computed: true},
				{Name: "version_stages", Type: "list", Computed: true},
			},
		},
	}
}

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
