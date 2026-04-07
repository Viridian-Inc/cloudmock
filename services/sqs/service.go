package sqs

import (
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
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

// ResourceSchemas returns the schema for SQS queue resources.
func (s *SQSService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "sqs",
			ResourceType:  "aws_sqs_queue",
			TerraformType: "cloudmock_sqs_queue",
			AWSType:       "AWS::SQS::Queue",
			CreateAction:  "CreateQueue",
			ReadAction:    "GetQueueAttributes",
			DeleteAction:  "DeleteQueue",
			ListAction:    "ListQueues",
			ImportID:      "url",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", ForceNew: true},
				{Name: "url", Type: "string", Computed: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "fifo_queue", Type: "bool", Default: false, ForceNew: true},
				{Name: "visibility_timeout_seconds", Type: "int", Default: 30},
				{Name: "message_retention_seconds", Type: "int", Default: 345600},
				{Name: "max_message_size", Type: "int", Default: 262144},
				{Name: "delay_seconds", Type: "int", Default: 0},
				{Name: "receive_wait_time_seconds", Type: "int", Default: 0},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// GetQueueNames returns all queue names for topology queries.
func (s *SQSService) GetQueueNames() []string {
	return s.store.ListQueues("")
}

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

// PollMessages receives messages from a named queue without going through
// the HTTP path. Used for event source mapping (SQS → Lambda).
// Returns parallel slices of message IDs, bodies, and receipt handles.
func (s *SQSService) PollMessages(queueName string, maxCount, visibilityTimeout int) (messageIDs, bodies, receiptHandles []string, ok bool) {
	q, qOK := s.store.GetByName(queueName)
	if !qOK {
		return nil, nil, nil, false
	}
	msgs := q.ReceiveMessages(maxCount, visibilityTimeout, 0)
	ids := make([]string, 0, len(msgs))
	bds := make([]string, 0, len(msgs))
	rhs := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.MessageId)
		bds = append(bds, m.Body)
		rhs = append(rhs, m.ReceiptHandle)
	}
	return ids, bds, rhs, true
}

// AckMessage deletes a message from a named queue by receipt handle.
// Used for event source mapping (SQS → Lambda) after successful processing.
func (s *SQSService) AckMessage(queueName, receiptHandle string) bool {
	q, ok := s.store.GetByName(queueName)
	if !ok {
		return false
	}
	return q.DeleteMessage(receiptHandle)
}

// HandleRequest routes an incoming SQS request to the appropriate handler.
// It supports both query/form-encoded (XML) and JSON protocol formats.
// JSON protocol is detected via Content-Type: application/x-amz-json-* or
// the presence of an X-Amz-Target header.
func (s *SQSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	// Detect JSON protocol.
	contentType := ctx.RawRequest.Header.Get("Content-Type")
	isJSON := strings.Contains(contentType, "application/x-amz-json") ||
		ctx.RawRequest.Header.Get("X-Amz-Target") != ""

	if isJSON {
		return s.handleJSON(ctx)
	}

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
