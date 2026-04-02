package sqs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// handleJSON dispatches a JSON-protocol SQS request based on X-Amz-Target.
func (s *SQSService) handleJSON(ctx *service.RequestContext) (*service.Response, error) {
	target := ctx.RawRequest.Header.Get("X-Amz-Target")
	action := strings.TrimPrefix(target, "AmazonSQS.")

	switch action {
	case "CreateQueue":
		return jsonCreateQueue(ctx, s.store)
	case "DeleteQueue":
		return jsonDeleteQueue(ctx, s.store)
	case "ListQueues":
		return jsonListQueues(ctx, s.store)
	case "GetQueueUrl":
		return jsonGetQueueUrl(ctx, s.store)
	case "GetQueueAttributes":
		return jsonGetQueueAttributes(ctx, s.store)
	case "SetQueueAttributes":
		return jsonSetQueueAttributes(ctx, s.store)
	case "SendMessage":
		return jsonSendMessage(ctx, s.store)
	case "ReceiveMessage":
		return jsonReceiveMessage(ctx, s.store)
	case "DeleteMessage":
		return jsonDeleteMessage(ctx, s.store)
	case "PurgeQueue":
		return jsonPurgeQueue(ctx, s.store)
	case "ChangeMessageVisibility":
		return jsonChangeMessageVisibility(ctx, s.store)
	case "SendMessageBatch":
		return jsonSendMessageBatch(ctx, s.store)
	case "DeleteMessageBatch":
		return jsonDeleteMessageBatch(ctx, s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}

// parseJSONBody reads the request body and unmarshals it into the given target.
func parseJSONBody(ctx *service.RequestContext, target any) error {
	body := ctx.Body
	if len(body) == 0 && ctx.RawRequest != nil && ctx.RawRequest.Body != nil {
		var err error
		body, err = io.ReadAll(ctx.RawRequest.Body)
		if err != nil {
			return err
		}
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, target)
}

// jsonOK wraps a response body in a 200 JSON response.
func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

// jsonErr wraps an AWSError in a JSON error response.
func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

// resolveQueueFromURL resolves the queue from a QueueUrl field in JSON input.
func resolveQueueFromURL(queueURL string, store *QueueStore) (Queue, error) {
	q, ok := store.GetByURL(queueURL)
	if ok {
		return q, nil
	}
	// Try extracting queue name from URL path.
	queueURL = strings.TrimRight(queueURL, "/")
	parts := strings.Split(queueURL, "/")
	if len(parts) >= 1 {
		name := parts[len(parts)-1]
		if q, ok := store.GetByName(name); ok {
			return q, nil
		}
	}
	return nil, service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
		"The specified queue does not exist.", http.StatusBadRequest)
}

// ---- CreateQueue ----

type jsonCreateQueueInput struct {
	QueueName  string            `json:"QueueName"`
	Attributes map[string]string `json:"Attributes"`
	Tags       map[string]string `json:"tags"`
}

type jsonCreateQueueOutput struct {
	QueueUrl string `json:"QueueUrl"`
}

func jsonCreateQueue(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonCreateQueueInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.QueueName == "" {
		return jsonErr(service.ErrValidation("QueueName is required."))
	}

	q, _ := store.CreateQueue(input.QueueName, input.Attributes)
	return jsonOK(&jsonCreateQueueOutput{QueueUrl: q.QueueURL()})
}

// ---- DeleteQueue ----

type jsonDeleteQueueInput struct {
	QueueUrl string `json:"QueueUrl"`
}

func jsonDeleteQueue(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonDeleteQueueInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	if input.QueueUrl == "" {
		return jsonErr(service.ErrValidation("QueueUrl is required."))
	}

	if !store.DeleteQueue(input.QueueUrl) {
		return jsonErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	return jsonOK(map[string]any{})
}

// ---- ListQueues ----

type jsonListQueuesInput struct {
	QueueNamePrefix string `json:"QueueNamePrefix"`
}

type jsonListQueuesOutput struct {
	QueueUrls []string `json:"QueueUrls"`
}

func jsonListQueues(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonListQueuesInput
	_ = parseJSONBody(ctx, &input)

	urls := store.ListQueues(input.QueueNamePrefix)
	if urls == nil {
		urls = []string{}
	}
	return jsonOK(&jsonListQueuesOutput{QueueUrls: urls})
}

// ---- GetQueueUrl ----

type jsonGetQueueUrlInput struct {
	QueueName string `json:"QueueName"`
}

type jsonGetQueueUrlOutput struct {
	QueueUrl string `json:"QueueUrl"`
}

func jsonGetQueueUrl(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonGetQueueUrlInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.QueueName == "" {
		return jsonErr(service.ErrValidation("QueueName is required."))
	}

	q, ok := store.GetByName(input.QueueName)
	if !ok {
		return jsonErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}
	return jsonOK(&jsonGetQueueUrlOutput{QueueUrl: q.QueueURL()})
}

// ---- GetQueueAttributes ----

type jsonGetQueueAttributesInput struct {
	QueueUrl       string   `json:"QueueUrl"`
	AttributeNames []string `json:"AttributeNames"`
}

type jsonGetQueueAttributesOutput struct {
	Attributes map[string]string `json:"Attributes"`
}

func jsonGetQueueAttributes(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonGetQueueAttributesInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	q, err := resolveQueueFromURL(input.QueueUrl, store)
	if err != nil {
		return jsonErr(err.(*service.AWSError))
	}

	wantAll := len(input.AttributeNames) == 0
	for _, n := range input.AttributeNames {
		if n == "All" {
			wantAll = true
			break
		}
	}

	wanted := make(map[string]bool, len(input.AttributeNames))
	for _, n := range input.AttributeNames {
		wanted[n] = true
	}

	// Computed attributes.
	computedAttrs := map[string]string{
		"ApproximateNumberOfMessages":           strconv.Itoa(q.ApproximateNumberOfMessages()),
		"ApproximateNumberOfMessagesNotVisible": strconv.Itoa(q.ApproximateNumberOfMessagesNotVisible()),
		"ApproximateNumberOfMessagesDelayed":    "0",
		"QueueArn": fmt.Sprintf("arn:aws:sqs:%s:%s:%s",
			regionFromURL(q.QueueURL()), accountFromURL(q.QueueURL()), q.QueueName()),
		"CreatedTimestamp":      "0",
		"LastModifiedTimestamp": "0",
	}

	allAttrs := make(map[string]string)
	for k, v := range q.GetAttributes() {
		allAttrs[k] = v
	}
	for k, v := range computedAttrs {
		allAttrs[k] = v
	}

	result := make(map[string]string)
	for name, val := range allAttrs {
		if wantAll || wanted[name] {
			result[name] = val
		}
	}

	return jsonOK(&jsonGetQueueAttributesOutput{Attributes: result})
}

// ---- SetQueueAttributes ----

type jsonSetQueueAttributesInput struct {
	QueueUrl   string            `json:"QueueUrl"`
	Attributes map[string]string `json:"Attributes"`
}

func jsonSetQueueAttributes(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonSetQueueAttributesInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	q, err := resolveQueueFromURL(input.QueueUrl, store)
	if err != nil {
		return jsonErr(err.(*service.AWSError))
	}

	q.SetAttributes(input.Attributes)

	return jsonOK(map[string]any{})
}

// ---- SendMessage ----

type jsonSendMessageInput struct {
	QueueUrl               string                        `json:"QueueUrl"`
	MessageBody            string                        `json:"MessageBody"`
	DelaySeconds           int                           `json:"DelaySeconds"`
	MessageAttributes      map[string]jsonMessageAttrVal `json:"MessageAttributes"`
	MessageGroupId         string                        `json:"MessageGroupId"`
	MessageDeduplicationId string                        `json:"MessageDeduplicationId"`
}

type jsonMessageAttrVal struct {
	DataType    string `json:"DataType"`
	StringValue string `json:"StringValue"`
}

type jsonSendMessageOutput struct {
	MessageId      string `json:"MessageId"`
	MD5OfMessageBody string `json:"MD5OfMessageBody"`
}

func jsonSendMessage(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonSendMessageInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	q, err := resolveQueueFromURL(input.QueueUrl, store)
	if err != nil {
		return jsonErr(err.(*service.AWSError))
	}

	if input.MessageBody == "" {
		return jsonErr(service.ErrValidation("MessageBody is required."))
	}

	var msgAttrs map[string]MessageAttribute
	if len(input.MessageAttributes) > 0 {
		msgAttrs = make(map[string]MessageAttribute, len(input.MessageAttributes))
		for k, v := range input.MessageAttributes {
			msgAttrs[k] = MessageAttribute{DataType: v.DataType, StringValue: v.StringValue}
		}
	}

	msgID := q.SendMessage(input.MessageBody, input.DelaySeconds, msgAttrs, input.MessageGroupId, input.MessageDeduplicationId)
	if msgID == "" {
		msgID = newUUID()
	}

	return jsonOK(&jsonSendMessageOutput{
		MessageId:      msgID,
		MD5OfMessageBody: md5Hex(input.MessageBody),
	})
}

// ---- ReceiveMessage ----

type jsonReceiveMessageInput struct {
	QueueUrl            string   `json:"QueueUrl"`
	MaxNumberOfMessages int      `json:"MaxNumberOfMessages"`
	VisibilityTimeout   int      `json:"VisibilityTimeout"`
	WaitTimeSeconds     int      `json:"WaitTimeSeconds"`
	AttributeNames      []string `json:"AttributeNames"`
}

type jsonReceiveMessageOutput struct {
	Messages []jsonMessageOut `json:"Messages"`
}

type jsonMessageOut struct {
	MessageId     string            `json:"MessageId"`
	ReceiptHandle string            `json:"ReceiptHandle"`
	Body          string            `json:"Body"`
	MD5OfBody     string            `json:"MD5OfBody"`
	Attributes    map[string]string `json:"Attributes,omitempty"`
}

func jsonReceiveMessage(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonReceiveMessageInput
	_ = parseJSONBody(ctx, &input)

	q, err := resolveQueueFromURL(input.QueueUrl, store)
	if err != nil {
		return jsonErr(err.(*service.AWSError))
	}

	maxCount := input.MaxNumberOfMessages
	if maxCount <= 0 {
		maxCount = 1
	}
	if maxCount > 10 {
		maxCount = 10
	}

	visTimeout := input.VisibilityTimeout
	if visTimeout <= 0 {
		visTimeout = 30
	}

	msgs := q.ReceiveMessages(maxCount, visTimeout, input.WaitTimeSeconds)

	outMsgs := make([]jsonMessageOut, 0, len(msgs))
	for _, m := range msgs {
		attrs := map[string]string{
			"SentTimestamp":                     strconv.FormatInt(m.SentTimestamp.UnixMilli(), 10),
			"ApproximateReceiveCount":           strconv.Itoa(m.ReceiveCount),
			"ApproximateFirstReceiveTimestamp":  strconv.FormatInt(m.FirstReceiveTimestamp.UnixMilli(), 10),
		}
		outMsgs = append(outMsgs, jsonMessageOut{
			MessageId:     m.MessageId,
			ReceiptHandle: m.ReceiptHandle,
			Body:          m.Body,
			MD5OfBody:     m.MD5OfBody,
			Attributes:    attrs,
		})
	}

	return jsonOK(&jsonReceiveMessageOutput{Messages: outMsgs})
}

// ---- DeleteMessage ----

type jsonDeleteMessageInput struct {
	QueueUrl      string `json:"QueueUrl"`
	ReceiptHandle string `json:"ReceiptHandle"`
}

func jsonDeleteMessage(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonDeleteMessageInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	q, err := resolveQueueFromURL(input.QueueUrl, store)
	if err != nil {
		return jsonErr(err.(*service.AWSError))
	}

	if input.ReceiptHandle == "" {
		return jsonErr(service.ErrValidation("ReceiptHandle is required."))
	}

	if !q.DeleteMessage(input.ReceiptHandle) {
		return jsonErr(service.NewAWSError("ReceiptHandleIsInvalid",
			"The input receipt handle is invalid.", http.StatusBadRequest))
	}

	return jsonOK(map[string]any{})
}

// ---- PurgeQueue ----

type jsonPurgeQueueInput struct {
	QueueUrl string `json:"QueueUrl"`
}

func jsonPurgeQueue(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonPurgeQueueInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	q, err := resolveQueueFromURL(input.QueueUrl, store)
	if err != nil {
		return jsonErr(err.(*service.AWSError))
	}

	q.Purge()
	return jsonOK(map[string]any{})
}

// ---- ChangeMessageVisibility ----

type jsonChangeMessageVisibilityInput struct {
	QueueUrl          string `json:"QueueUrl"`
	ReceiptHandle     string `json:"ReceiptHandle"`
	VisibilityTimeout int    `json:"VisibilityTimeout"`
}

func jsonChangeMessageVisibility(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonChangeMessageVisibilityInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	q, err := resolveQueueFromURL(input.QueueUrl, store)
	if err != nil {
		return jsonErr(err.(*service.AWSError))
	}

	if input.ReceiptHandle == "" {
		return jsonErr(service.ErrValidation("ReceiptHandle is required."))
	}

	if !q.ChangeMessageVisibility(input.ReceiptHandle, input.VisibilityTimeout) {
		return jsonErr(service.NewAWSError("ReceiptHandleIsInvalid",
			"The input receipt handle is invalid.", http.StatusBadRequest))
	}

	return jsonOK(map[string]any{})
}

// ---- SendMessageBatch ----

type jsonSendMessageBatchInput struct {
	QueueUrl string                         `json:"QueueUrl"`
	Entries  []jsonSendMessageBatchEntry    `json:"Entries"`
}

type jsonSendMessageBatchEntry struct {
	Id                     string `json:"Id"`
	MessageBody            string `json:"MessageBody"`
	DelaySeconds           int    `json:"DelaySeconds"`
	MessageGroupId         string `json:"MessageGroupId"`
	MessageDeduplicationId string `json:"MessageDeduplicationId"`
}

type jsonSendMessageBatchOutput struct {
	Successful []jsonSendMessageBatchResultEntry `json:"Successful"`
	Failed     []jsonBatchResultErrorEntry       `json:"Failed"`
}

type jsonSendMessageBatchResultEntry struct {
	Id               string `json:"Id"`
	MessageId        string `json:"MessageId"`
	MD5OfMessageBody string `json:"MD5OfMessageBody"`
}

type jsonBatchResultErrorEntry struct {
	Id          string `json:"Id"`
	Code        string `json:"Code"`
	Message     string `json:"Message"`
	SenderFault bool   `json:"SenderFault"`
}

func jsonSendMessageBatch(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonSendMessageBatchInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	q, err := resolveQueueFromURL(input.QueueUrl, store)
	if err != nil {
		return jsonErr(err.(*service.AWSError))
	}

	if len(input.Entries) == 0 {
		return jsonErr(service.NewAWSError("AWS.SimpleQueueService.EmptyBatchRequest",
			"The batch request doesn't contain any entries.", http.StatusBadRequest))
	}
	if len(input.Entries) > 10 {
		return jsonErr(service.NewAWSError("AWS.SimpleQueueService.TooManyEntriesInBatchRequest",
			"Maximum number of entries per request are 10.", http.StatusBadRequest))
	}

	successful := make([]jsonSendMessageBatchResultEntry, 0)
	failed := make([]jsonBatchResultErrorEntry, 0)

	for _, entry := range input.Entries {
		if entry.Id == "" || entry.MessageBody == "" {
			failed = append(failed, jsonBatchResultErrorEntry{
				Id:          entry.Id,
				Code:        "MissingParameter",
				Message:     "Id and MessageBody are required.",
				SenderFault: true,
			})
			continue
		}
		msgID := q.SendMessage(entry.MessageBody, entry.DelaySeconds, nil, entry.MessageGroupId, entry.MessageDeduplicationId)
		if msgID == "" {
			msgID = newUUID()
		}
		successful = append(successful, jsonSendMessageBatchResultEntry{
			Id:               entry.Id,
			MessageId:        msgID,
			MD5OfMessageBody: md5Hex(entry.MessageBody),
		})
	}

	return jsonOK(&jsonSendMessageBatchOutput{Successful: successful, Failed: failed})
}

// ---- DeleteMessageBatch ----

type jsonDeleteMessageBatchInput struct {
	QueueUrl string                          `json:"QueueUrl"`
	Entries  []jsonDeleteMessageBatchEntry   `json:"Entries"`
}

type jsonDeleteMessageBatchEntry struct {
	Id            string `json:"Id"`
	ReceiptHandle string `json:"ReceiptHandle"`
}

type jsonDeleteMessageBatchOutput struct {
	Successful []jsonDeleteMessageBatchResultEntry `json:"Successful"`
	Failed     []jsonBatchResultErrorEntry         `json:"Failed"`
}

type jsonDeleteMessageBatchResultEntry struct {
	Id string `json:"Id"`
}

func jsonDeleteMessageBatch(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	var input jsonDeleteMessageBatchInput
	if err := parseJSONBody(ctx, &input); err != nil {
		return jsonErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	q, err := resolveQueueFromURL(input.QueueUrl, store)
	if err != nil {
		return jsonErr(err.(*service.AWSError))
	}

	if len(input.Entries) == 0 {
		return jsonErr(service.NewAWSError("AWS.SimpleQueueService.EmptyBatchRequest",
			"The batch request doesn't contain any entries.", http.StatusBadRequest))
	}
	if len(input.Entries) > 10 {
		return jsonErr(service.NewAWSError("AWS.SimpleQueueService.TooManyEntriesInBatchRequest",
			"Maximum number of entries per request are 10.", http.StatusBadRequest))
	}

	successful := make([]jsonDeleteMessageBatchResultEntry, 0)
	failed := make([]jsonBatchResultErrorEntry, 0)

	for _, entry := range input.Entries {
		if !q.DeleteMessage(entry.ReceiptHandle) {
			failed = append(failed, jsonBatchResultErrorEntry{
				Id:          entry.Id,
				Code:        "ReceiptHandleIsInvalid",
				Message:     "The input receipt handle is invalid.",
				SenderFault: true,
			})
		} else {
			successful = append(successful, jsonDeleteMessageBatchResultEntry{Id: entry.Id})
		}
	}

	return jsonOK(&jsonDeleteMessageBatchOutput{Successful: successful, Failed: failed})
}
