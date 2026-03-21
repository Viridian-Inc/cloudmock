package sqs

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// SQSService is the cloudmock implementation of the AWS Simple Queue Service API.
type SQSService struct {
	store *QueueStore
}

// New returns a new SQSService for the given AWS account ID and region.
func New(accountID, region string) *SQSService {
	return &SQSService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *SQSService) Name() string { return "sqs" }

// Actions returns the list of SQS API actions supported by this service.
func (s *SQSService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateQueue", Method: http.MethodPost, IAMAction: "sqs:CreateQueue"},
		{Name: "DeleteQueue", Method: http.MethodPost, IAMAction: "sqs:DeleteQueue"},
		{Name: "ListQueues", Method: http.MethodPost, IAMAction: "sqs:ListQueues"},
		{Name: "GetQueueUrl", Method: http.MethodPost, IAMAction: "sqs:GetQueueUrl"},
		{Name: "GetQueueAttributes", Method: http.MethodPost, IAMAction: "sqs:GetQueueAttributes"},
		{Name: "SetQueueAttributes", Method: http.MethodPost, IAMAction: "sqs:SetQueueAttributes"},
		{Name: "SendMessage", Method: http.MethodPost, IAMAction: "sqs:SendMessage"},
		{Name: "ReceiveMessage", Method: http.MethodPost, IAMAction: "sqs:ReceiveMessage"},
		{Name: "DeleteMessage", Method: http.MethodPost, IAMAction: "sqs:DeleteMessage"},
		{Name: "PurgeQueue", Method: http.MethodPost, IAMAction: "sqs:PurgeQueue"},
		{Name: "ChangeMessageVisibility", Method: http.MethodPost, IAMAction: "sqs:ChangeMessageVisibility"},
		{Name: "SendMessageBatch", Method: http.MethodPost, IAMAction: "sqs:SendMessageBatch"},
		{Name: "DeleteMessageBatch", Method: http.MethodPost, IAMAction: "sqs:DeleteMessageBatch"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *SQSService) HealthCheck() error { return nil }

// EnqueueDirect adds a message to the named queue without going through
// the HTTP/form-parsing path. This is used for cross-service delivery
// (e.g., SNS → SQS, EventBridge → SQS, S3 notifications → SQS).
// Returns true if the message was enqueued.
func (s *SQSService) EnqueueDirect(queueName, messageBody string) bool {
	q, ok := s.store.GetByName(queueName)
	if !ok {
		return false
	}
	q.SendMessage(messageBody, 0, nil, "", "")
	return true
}

// HandleRequest routes an incoming SQS request to the appropriate handler.
// SQS uses form-encoded bodies; the Action may appear in the query string
// (already parsed into ctx.Params) or in the form-encoded body.
func (s *SQSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "CreateQueue":
		return handleCreateQueue(ctx, s.store)
	case "DeleteQueue":
		return handleDeleteQueue(ctx, s.store)
	case "ListQueues":
		return handleListQueues(ctx, s.store)
	case "GetQueueUrl":
		return handleGetQueueUrl(ctx, s.store)
	case "GetQueueAttributes":
		return handleGetQueueAttributes(ctx, s.store)
	case "SetQueueAttributes":
		return handleSetQueueAttributes(ctx, s.store)
	case "SendMessage":
		return handleSendMessage(ctx, s.store)
	case "ReceiveMessage":
		return handleReceiveMessage(ctx, s.store)
	case "DeleteMessage":
		return handleDeleteMessage(ctx, s.store)
	case "PurgeQueue":
		return handlePurgeQueue(ctx, s.store)
	case "ChangeMessageVisibility":
		return handleChangeMessageVisibility(ctx, s.store)
	case "SendMessageBatch":
		return handleSendMessageBatch(ctx, s.store)
	case "DeleteMessageBatch":
		return handleDeleteMessageBatch(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
