package kms

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/schema"
	"github.com/neureaux/cloudmock/pkg/service"
)

// KMSService is the cloudmock implementation of the AWS Key Management Service API.
type KMSService struct {
	store *Store
}

// New returns a new KMSService for the given AWS account ID and region.
func New(accountID, region string) *KMSService {
	return &KMSService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *KMSService) Name() string { return "kms" }

// Actions returns the list of KMS API actions supported by this service.
func (s *KMSService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateKey", Method: http.MethodPost, IAMAction: "kms:CreateKey"},
		{Name: "DescribeKey", Method: http.MethodPost, IAMAction: "kms:DescribeKey"},
		{Name: "ListKeys", Method: http.MethodPost, IAMAction: "kms:ListKeys"},
		{Name: "Encrypt", Method: http.MethodPost, IAMAction: "kms:Encrypt"},
		{Name: "Decrypt", Method: http.MethodPost, IAMAction: "kms:Decrypt"},
		{Name: "CreateAlias", Method: http.MethodPost, IAMAction: "kms:CreateAlias"},
		{Name: "ListAliases", Method: http.MethodPost, IAMAction: "kms:ListAliases"},
		{Name: "EnableKey", Method: http.MethodPost, IAMAction: "kms:EnableKey"},
		{Name: "DisableKey", Method: http.MethodPost, IAMAction: "kms:DisableKey"},
		{Name: "ScheduleKeyDeletion", Method: http.MethodPost, IAMAction: "kms:ScheduleKeyDeletion"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *KMSService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for KMS resource types.
func (s *KMSService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "kms",
			ResourceType:  "aws_kms_key",
			TerraformType: "cloudmock_kms_key",
			AWSType:       "AWS::KMS::Key",
			CreateAction:  "CreateKey",
			ReadAction:    "DescribeKey",
			DeleteAction:  "ScheduleKeyDeletion",
			ListAction:    "ListKeys",
			ImportID:      "key_id",
			Attributes: []schema.AttributeSchema{
				{Name: "description", Type: "string"},
				{Name: "key_usage", Type: "string", Default: "ENCRYPT_DECRYPT"},
				{Name: "policy", Type: "string"},
				{Name: "enable_key_rotation", Type: "bool", Default: false},
				{Name: "is_enabled", Type: "bool", Default: true},
				{Name: "deletion_window_in_days", Type: "int", Default: 30},
				{Name: "key_id", Type: "string", Computed: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "kms",
			ResourceType:  "aws_kms_alias",
			TerraformType: "cloudmock_kms_alias",
			AWSType:       "AWS::KMS::Alias",
			CreateAction:  "CreateAlias",
			ReadAction:    "ListAliases",
			DeleteAction:  "DeleteAlias",
			ListAction:    "ListAliases",
			ImportID:      "name",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "target_key_id", Type: "string", Required: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "target_key_arn", Type: "string", Computed: true},
			},
		},
	}
}

// HandleRequest routes an incoming KMS request to the appropriate handler.
// KMS uses the JSON protocol; the action is parsed from X-Amz-Target by the gateway
// and placed in ctx.Action (e.g. "CreateKey").
func (s *KMSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateKey":
		return handleCreateKey(ctx, s.store)
	case "DescribeKey":
		return handleDescribeKey(ctx, s.store)
	case "ListKeys":
		return handleListKeys(ctx, s.store)
	case "Encrypt":
		return handleEncrypt(ctx, s.store)
	case "Decrypt":
		return handleDecrypt(ctx, s.store)
	case "CreateAlias":
		return handleCreateAlias(ctx, s.store)
	case "ListAliases":
		return handleListAliases(ctx, s.store)
	case "EnableKey":
		return handleEnableKey(ctx, s.store)
	case "DisableKey":
		return handleDisableKey(ctx, s.store)
	case "ScheduleKeyDeletion":
		return handleScheduleKeyDeletion(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
