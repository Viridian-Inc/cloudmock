package kinesis

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// KinesisService is the cloudmock implementation of the AWS Kinesis Data Streams API.
type KinesisService struct {
	store *Store
}

// New returns a new KinesisService for the given AWS account ID and region.
func New(accountID, region string) *KinesisService {
	return &KinesisService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *KinesisService) Name() string { return "kinesis" }

// Actions returns the list of Kinesis API actions supported by this service.
func (s *KinesisService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateStream", Method: http.MethodPost, IAMAction: "kinesis:CreateStream"},
		{Name: "DeleteStream", Method: http.MethodPost, IAMAction: "kinesis:DeleteStream"},
		{Name: "DescribeStream", Method: http.MethodPost, IAMAction: "kinesis:DescribeStream"},
		{Name: "ListStreams", Method: http.MethodPost, IAMAction: "kinesis:ListStreams"},
		{Name: "PutRecord", Method: http.MethodPost, IAMAction: "kinesis:PutRecord"},
		{Name: "PutRecords", Method: http.MethodPost, IAMAction: "kinesis:PutRecords"},
		{Name: "GetShardIterator", Method: http.MethodPost, IAMAction: "kinesis:GetShardIterator"},
		{Name: "GetRecords", Method: http.MethodPost, IAMAction: "kinesis:GetRecords"},
		{Name: "IncreaseStreamRetentionPeriod", Method: http.MethodPost, IAMAction: "kinesis:IncreaseStreamRetentionPeriod"},
		{Name: "DecreaseStreamRetentionPeriod", Method: http.MethodPost, IAMAction: "kinesis:DecreaseStreamRetentionPeriod"},
		{Name: "AddTagsToStream", Method: http.MethodPost, IAMAction: "kinesis:AddTagsToStream"},
		{Name: "RemoveTagsFromStream", Method: http.MethodPost, IAMAction: "kinesis:RemoveTagsFromStream"},
		{Name: "ListTagsForStream", Method: http.MethodPost, IAMAction: "kinesis:ListTagsForStream"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *KinesisService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Kinesis request to the appropriate handler.
// Kinesis uses the JSON protocol; the action is parsed from X-Amz-Target by the gateway
// and placed in ctx.Action (e.g. "CreateStream").
func (s *KinesisService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateStream":
		return handleCreateStream(ctx, s.store)
	case "DeleteStream":
		return handleDeleteStream(ctx, s.store)
	case "DescribeStream":
		return handleDescribeStream(ctx, s.store)
	case "ListStreams":
		return handleListStreams(ctx, s.store)
	case "PutRecord":
		return handlePutRecord(ctx, s.store)
	case "PutRecords":
		return handlePutRecords(ctx, s.store)
	case "GetShardIterator":
		return handleGetShardIterator(ctx, s.store)
	case "GetRecords":
		return handleGetRecords(ctx, s.store)
	case "IncreaseStreamRetentionPeriod":
		return handleIncreaseStreamRetentionPeriod(ctx, s.store)
	case "DecreaseStreamRetentionPeriod":
		return handleDecreaseStreamRetentionPeriod(ctx, s.store)
	case "AddTagsToStream":
		return handleAddTagsToStream(ctx, s.store)
	case "RemoveTagsFromStream":
		return handleRemoveTagsFromStream(ctx, s.store)
	case "ListTagsForStream":
		return handleListTagsForStream(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
