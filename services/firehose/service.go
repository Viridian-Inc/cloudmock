package firehose

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// FirehoseService is the cloudmock implementation of the AWS Data Firehose API.
type FirehoseService struct {
	store *Store
}

// New returns a new FirehoseService for the given AWS account ID and region.
func New(accountID, region string) *FirehoseService {
	return &FirehoseService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *FirehoseService) Name() string { return "firehose" }

// Actions returns the list of Firehose API actions supported by this service.
func (s *FirehoseService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateDeliveryStream", Method: http.MethodPost, IAMAction: "firehose:CreateDeliveryStream"},
		{Name: "DeleteDeliveryStream", Method: http.MethodPost, IAMAction: "firehose:DeleteDeliveryStream"},
		{Name: "DescribeDeliveryStream", Method: http.MethodPost, IAMAction: "firehose:DescribeDeliveryStream"},
		{Name: "ListDeliveryStreams", Method: http.MethodPost, IAMAction: "firehose:ListDeliveryStreams"},
		{Name: "PutRecord", Method: http.MethodPost, IAMAction: "firehose:PutRecord"},
		{Name: "PutRecordBatch", Method: http.MethodPost, IAMAction: "firehose:PutRecordBatch"},
		{Name: "UpdateDestination", Method: http.MethodPost, IAMAction: "firehose:UpdateDestination"},
		{Name: "TagDeliveryStream", Method: http.MethodPost, IAMAction: "firehose:TagDeliveryStream"},
		{Name: "UntagDeliveryStream", Method: http.MethodPost, IAMAction: "firehose:UntagDeliveryStream"},
		{Name: "ListTagsForDeliveryStream", Method: http.MethodPost, IAMAction: "firehose:ListTagsForDeliveryStream"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *FirehoseService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Firehose request to the appropriate handler.
// Firehose uses the JSON protocol; the action is parsed from X-Amz-Target by the
// gateway and placed in ctx.Action (e.g. "CreateDeliveryStream").
func (s *FirehoseService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateDeliveryStream":
		return handleCreateDeliveryStream(ctx, s.store)
	case "DeleteDeliveryStream":
		return handleDeleteDeliveryStream(ctx, s.store)
	case "DescribeDeliveryStream":
		return handleDescribeDeliveryStream(ctx, s.store)
	case "ListDeliveryStreams":
		return handleListDeliveryStreams(ctx, s.store)
	case "PutRecord":
		return handlePutRecord(ctx, s.store)
	case "PutRecordBatch":
		return handlePutRecordBatch(ctx, s.store)
	case "UpdateDestination":
		return handleUpdateDestination(ctx, s.store)
	case "TagDeliveryStream":
		return handleTagDeliveryStream(ctx, s.store)
	case "UntagDeliveryStream":
		return handleUntagDeliveryStream(ctx, s.store)
	case "ListTagsForDeliveryStream":
		return handleListTagsForDeliveryStream(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
