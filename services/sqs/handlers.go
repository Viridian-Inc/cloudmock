package sqs

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

// ---- CreateQueue ----

type xmlCreateQueueResponse struct {
	XMLName xml.Name              `xml:"CreateQueueResponse"`
	Result  xmlCreateQueueResult  `xml:"CreateQueueResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlCreateQueueResult struct {
	QueueUrl string `xml:"QueueUrl"`
}

func handleCreateQueue(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("QueueName")
	if name == "" {
		return xmlErr(service.ErrValidation("QueueName is required."))
	}

	attrs := parseAttributes(form)

	q, _ := store.CreateQueue(name, attrs)

	resp := &xmlCreateQueueResponse{
		Result: xmlCreateQueueResult{QueueUrl: q.QueueURL()},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- DeleteQueue ----

type xmlDeleteQueueResponse struct {
	XMLName xml.Name            `xml:"DeleteQueueResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteQueue(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	if !store.DeleteQueue(queueURL) {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteQueueResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}

// ---- ListQueues ----

type xmlListQueuesResponse struct {
	XMLName xml.Name              `xml:"ListQueuesResponse"`
	Result  xmlListQueuesResult   `xml:"ListQueuesResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlListQueuesResult struct {
	QueueUrls []string `xml:"QueueUrl"`
}

func handleListQueues(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	prefix := form.Get("QueueNamePrefix")

	urls := store.ListQueues(prefix)

	resp := &xmlListQueuesResponse{
		Result: xmlListQueuesResult{QueueUrls: urls},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- GetQueueUrl ----

type xmlGetQueueUrlResponse struct {
	XMLName xml.Name              `xml:"GetQueueUrlResponse"`
	Result  xmlGetQueueUrlResult  `xml:"GetQueueUrlResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlGetQueueUrlResult struct {
	QueueUrl string `xml:"QueueUrl"`
}

func handleGetQueueUrl(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("QueueName")
	if name == "" {
		return xmlErr(service.ErrValidation("QueueName is required."))
	}

	q, ok := store.GetByName(name)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	resp := &xmlGetQueueUrlResponse{
		Result: xmlGetQueueUrlResult{QueueUrl: q.QueueURL()},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- GetQueueAttributes ----

type xmlGetQueueAttributesResponse struct {
	XMLName xml.Name                     `xml:"GetQueueAttributesResponse"`
	Result  xmlGetQueueAttributesResult  `xml:"GetQueueAttributesResult"`
	Meta    xmlResponseMetadata          `xml:"ResponseMetadata"`
}

type xmlGetQueueAttributesResult struct {
	Attributes []xmlAttribute `xml:"Attribute"`
}

type xmlAttribute struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

func handleGetQueueAttributes(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q, ok := store.GetByURL(queueURL)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	// Collect requested attribute names.
	wantedNames := parseAttributeNames(form)
	wantAll := len(wantedNames) == 0
	for _, n := range wantedNames {
		if n == "All" {
			wantAll = true
			break
		}
	}

	wanted := make(map[string]bool, len(wantedNames))
	for _, n := range wantedNames {
		wanted[n] = true
	}

	// Build result.
	attrs := make([]xmlAttribute, 0)

	// Dynamic / computed attributes.
	computedAttrs := map[string]string{
		"ApproximateNumberOfMessages":           strconv.Itoa(q.ApproximateNumberOfMessages()),
		"ApproximateNumberOfMessagesNotVisible": strconv.Itoa(q.ApproximateNumberOfMessagesNotVisible()),
		"ApproximateNumberOfMessagesDelayed":    "0",
		"QueueArn": fmt.Sprintf("arn:aws:sqs:%s:%s:%s",
			regionFromURL(queueURL), accountFromURL(queueURL), q.QueueName()),
		"CreatedTimestamp":      "0",
		"LastModifiedTimestamp": "0",
	}

	// Merge static attributes.
	allAttrs := make(map[string]string)
	for k, v := range q.GetAttributes() {
		allAttrs[k] = v
	}
	for k, v := range computedAttrs {
		allAttrs[k] = v
	}

	for name, val := range allAttrs {
		if wantAll || wanted[name] {
			attrs = append(attrs, xmlAttribute{Name: name, Value: val})
		}
	}

	resp := &xmlGetQueueAttributesResponse{
		Result: xmlGetQueueAttributesResult{Attributes: attrs},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- SetQueueAttributes ----

type xmlSetQueueAttributesResponse struct {
	XMLName xml.Name            `xml:"SetQueueAttributesResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleSetQueueAttributes(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q, ok := store.GetByURL(queueURL)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	newAttrs := parseAttributes(form)
	q.SetAttributes(newAttrs)

	return xmlOK(&xmlSetQueueAttributesResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}

// ---- SendMessage ----

type xmlSendMessageResponse struct {
	XMLName xml.Name              `xml:"SendMessageResponse"`
	Result  xmlSendMessageResult  `xml:"SendMessageResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlSendMessageResult struct {
	MD5OfMessageBody string `xml:"MD5OfMessageBody"`
	MessageId        string `xml:"MessageId"`
}

func handleSendMessage(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q, ok := store.GetByURL(queueURL)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	body := form.Get("MessageBody")
	if body == "" {
		return xmlErr(service.ErrValidation("MessageBody is required."))
	}

	delay := 0
	if d := form.Get("DelaySeconds"); d != "" {
		if v, err := strconv.Atoi(d); err == nil {
			delay = v
		}
	}

	msgAttrs := parseMessageAttributes(form)
	groupID := form.Get("MessageGroupId")
	dedupID := form.Get("MessageDeduplicationId")

	msgID := q.SendMessage(body, delay, msgAttrs, groupID, dedupID)
	if msgID == "" {
		// Deduplicated — return a synthetic response.
		msgID = newUUID()
	}

	resp := &xmlSendMessageResponse{
		Result: xmlSendMessageResult{
			MD5OfMessageBody: md5Hex(body),
			MessageId:        msgID,
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- ReceiveMessage ----

type xmlReceiveMessageResponse struct {
	XMLName xml.Name                `xml:"ReceiveMessageResponse"`
	Result  xmlReceiveMessageResult `xml:"ReceiveMessageResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlReceiveMessageResult struct {
	Messages []xmlMessage `xml:"Message"`
}

type xmlMessage struct {
	MessageId     string              `xml:"MessageId"`
	ReceiptHandle string              `xml:"ReceiptHandle"`
	MD5OfBody     string              `xml:"MD5OfBody"`
	Body          string              `xml:"Body"`
	Attributes    []xmlAttribute      `xml:"Attribute,omitempty"`
}

func handleReceiveMessage(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q, ok := store.GetByURL(queueURL)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	maxCount := 1
	if s := form.Get("MaxNumberOfMessages"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			if v > 10 {
				v = 10
			}
			maxCount = v
		}
	}

	visTimeout := 30
	if s := form.Get("VisibilityTimeout"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			visTimeout = v
		}
	}

	waitTime := 0
	if s := form.Get("WaitTimeSeconds"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			waitTime = v
		}
	}

	msgs := q.ReceiveMessages(maxCount, visTimeout, waitTime)

	xmlMsgs := make([]xmlMessage, 0, len(msgs))
	for _, m := range msgs {
		xmlMsgs = append(xmlMsgs, xmlMessage{
			MessageId:     m.MessageId,
			ReceiptHandle: m.ReceiptHandle,
			MD5OfBody:     m.MD5OfBody,
			Body:          m.Body,
			Attributes: []xmlAttribute{
				{Name: "SentTimestamp", Value: strconv.FormatInt(m.SentTimestamp.UnixMilli(), 10)},
				{Name: "ApproximateReceiveCount", Value: strconv.Itoa(m.ReceiveCount)},
				{Name: "ApproximateFirstReceiveTimestamp", Value: strconv.FormatInt(m.FirstReceiveTimestamp.UnixMilli(), 10)},
			},
		})
	}

	resp := &xmlReceiveMessageResponse{
		Result: xmlReceiveMessageResult{Messages: xmlMsgs},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- DeleteMessage ----

type xmlDeleteMessageResponse struct {
	XMLName xml.Name            `xml:"DeleteMessageResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteMessage(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q, ok := store.GetByURL(queueURL)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	receiptHandle := form.Get("ReceiptHandle")
	if receiptHandle == "" {
		return xmlErr(service.ErrValidation("ReceiptHandle is required."))
	}

	if !q.DeleteMessage(receiptHandle) {
		return xmlErr(service.NewAWSError("ReceiptHandleIsInvalid",
			"The input receipt handle is invalid.", http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteMessageResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}

// ---- PurgeQueue ----

type xmlPurgeQueueResponse struct {
	XMLName xml.Name            `xml:"PurgeQueueResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handlePurgeQueue(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q, ok := store.GetByURL(queueURL)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q.Purge()

	return xmlOK(&xmlPurgeQueueResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}

// ---- ChangeMessageVisibility ----

type xmlChangeMessageVisibilityResponse struct {
	XMLName xml.Name            `xml:"ChangeMessageVisibilityResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleChangeMessageVisibility(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q, ok := store.GetByURL(queueURL)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	receiptHandle := form.Get("ReceiptHandle")
	if receiptHandle == "" {
		return xmlErr(service.ErrValidation("ReceiptHandle is required."))
	}

	timeout := 30
	if s := form.Get("VisibilityTimeout"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			timeout = v
		}
	}

	if !q.ChangeMessageVisibility(receiptHandle, timeout) {
		return xmlErr(service.NewAWSError("ReceiptHandleIsInvalid",
			"The input receipt handle is invalid.", http.StatusBadRequest))
	}

	return xmlOK(&xmlChangeMessageVisibilityResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}

// ---- SendMessageBatch ----

type xmlSendMessageBatchResponse struct {
	XMLName xml.Name                       `xml:"SendMessageBatchResponse"`
	Result  xmlSendMessageBatchResult      `xml:"SendMessageBatchResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlSendMessageBatchResult struct {
	Successful []xmlSendMessageBatchResultEntry `xml:"SendMessageBatchResultEntry"`
	Failed     []xmlBatchResultErrorEntry       `xml:"BatchResultErrorEntry,omitempty"`
}

type xmlSendMessageBatchResultEntry struct {
	Id               string `xml:"Id"`
	MessageId        string `xml:"MessageId"`
	MD5OfMessageBody string `xml:"MD5OfMessageBody"`
}

type xmlBatchResultErrorEntry struct {
	Id          string `xml:"Id"`
	Code        string `xml:"Code"`
	Message     string `xml:"Message"`
	SenderFault bool   `xml:"SenderFault"`
}

func handleSendMessageBatch(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q, ok := store.GetByURL(queueURL)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	// Entries are passed as SendMessageBatchRequestEntry.N.{Id,MessageBody,...}
	entries := parseBatchSendEntries(form)

	if len(entries) == 0 {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.EmptyBatchRequest",
			"The batch request doesn't contain any entries.", http.StatusBadRequest))
	}
	if len(entries) > 10 {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.TooManyEntriesInBatchRequest",
			"Maximum number of entries per request are 10.", http.StatusBadRequest))
	}

	successful := make([]xmlSendMessageBatchResultEntry, 0)
	failed := make([]xmlBatchResultErrorEntry, 0)

	for _, entry := range entries {
		if entry.id == "" || entry.body == "" {
			failed = append(failed, xmlBatchResultErrorEntry{
				Id:          entry.id,
				Code:        "MissingParameter",
				Message:     "Id and MessageBody are required.",
				SenderFault: true,
			})
			continue
		}
		msgID := q.SendMessage(entry.body, entry.delay, nil, entry.groupID, entry.dedupID)
		if msgID == "" {
			msgID = newUUID()
		}
		successful = append(successful, xmlSendMessageBatchResultEntry{
			Id:               entry.id,
			MessageId:        msgID,
			MD5OfMessageBody: md5Hex(entry.body),
		})
	}

	resp := &xmlSendMessageBatchResponse{
		Result: xmlSendMessageBatchResult{Successful: successful, Failed: failed},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- DeleteMessageBatch ----

type xmlDeleteMessageBatchResponse struct {
	XMLName xml.Name                       `xml:"DeleteMessageBatchResponse"`
	Result  xmlDeleteMessageBatchResult    `xml:"DeleteMessageBatchResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlDeleteMessageBatchResult struct {
	Successful []xmlDeleteMessageBatchResultEntry `xml:"DeleteMessageBatchResultEntry"`
	Failed     []xmlBatchResultErrorEntry         `xml:"BatchResultErrorEntry,omitempty"`
}

type xmlDeleteMessageBatchResultEntry struct {
	Id string `xml:"Id"`
}

func handleDeleteMessageBatch(ctx *service.RequestContext, store *QueueStore) (*service.Response, error) {
	form := parseForm(ctx)
	queueURL := resolveQueueURL(ctx, form, store)
	if queueURL == "" {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	q, ok := store.GetByURL(queueURL)
	if !ok {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.NonExistentQueue",
			"The specified queue does not exist.", http.StatusBadRequest))
	}

	entries := parseBatchDeleteEntries(form)

	if len(entries) == 0 {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.EmptyBatchRequest",
			"The batch request doesn't contain any entries.", http.StatusBadRequest))
	}
	if len(entries) > 10 {
		return xmlErr(service.NewAWSError("AWS.SimpleQueueService.TooManyEntriesInBatchRequest",
			"Maximum number of entries per request are 10.", http.StatusBadRequest))
	}

	successful := make([]xmlDeleteMessageBatchResultEntry, 0)
	failed := make([]xmlBatchResultErrorEntry, 0)

	for _, entry := range entries {
		if !q.DeleteMessage(entry.receiptHandle) {
			failed = append(failed, xmlBatchResultErrorEntry{
				Id:          entry.id,
				Code:        "ReceiptHandleIsInvalid",
				Message:     "The input receipt handle is invalid.",
				SenderFault: true,
			})
		} else {
			successful = append(successful, xmlDeleteMessageBatchResultEntry{Id: entry.id})
		}
	}

	resp := &xmlDeleteMessageBatchResponse{
		Result: xmlDeleteMessageBatchResult{Successful: successful, Failed: failed},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- helper functions ----

// parseForm merges the query-string params and the form-encoded body into a
// single url.Values.
func parseForm(ctx *service.RequestContext) url.Values {
	form := make(url.Values)

	// Start with query-string params (already in ctx.Params).
	for k, v := range ctx.Params {
		form.Set(k, v)
	}

	// Overlay body form values (body takes precedence for duplicates).
	if len(ctx.Body) > 0 {
		if bodyVals, err := url.ParseQuery(string(ctx.Body)); err == nil {
			for k, vs := range bodyVals {
				for _, v := range vs {
					form.Add(k, v)
				}
			}
		}
	}
	return form
}

// resolveQueueURL resolves the QueueUrl from form values and, if not found,
// from the URL path (SQS often routes /{accountId}/{queueName}).
func resolveQueueURL(ctx *service.RequestContext, form url.Values, store *QueueStore) string {
	if u := form.Get("QueueUrl"); u != "" {
		return u
	}

	// Try the request URL path: /{accountId}/{queueName}
	if ctx.RawRequest != nil {
		path := ctx.RawRequest.URL.Path
		path = strings.TrimPrefix(path, "/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			queueName := parts[1]
			if q, ok := store.GetByName(queueName); ok {
				return q.QueueURL()
			}
		}
	}
	return ""
}

// parseAttributes parses Attribute.N.Name / Attribute.N.Value pairs from form values.
func parseAttributes(form url.Values) map[string]string {
	attrs := make(map[string]string)
	// The SDK sends Attribute.1.Name=X&Attribute.1.Value=Y ...
	for i := 1; ; i++ {
		name := form.Get(fmt.Sprintf("Attribute.%d.Name", i))
		if name == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Attribute.%d.Value", i))
		attrs[name] = val
	}
	return attrs
}

// parseAttributeNames parses AttributeName.N values from form values.
func parseAttributeNames(form url.Values) []string {
	names := make([]string, 0)
	for i := 1; ; i++ {
		name := form.Get(fmt.Sprintf("AttributeName.%d", i))
		if name == "" {
			break
		}
		names = append(names, name)
	}
	return names
}

// parseMessageAttributes parses MessageAttribute.N.Name/Value/DataType triples.
func parseMessageAttributes(form url.Values) map[string]MessageAttribute {
	attrs := make(map[string]MessageAttribute)
	for i := 1; ; i++ {
		name := form.Get(fmt.Sprintf("MessageAttribute.%d.Name", i))
		if name == "" {
			break
		}
		dataType := form.Get(fmt.Sprintf("MessageAttribute.%d.Value.DataType", i))
		strVal := form.Get(fmt.Sprintf("MessageAttribute.%d.Value.StringValue", i))
		attrs[name] = MessageAttribute{DataType: dataType, StringValue: strVal}
	}
	return attrs
}

type batchSendEntry struct {
	id      string
	body    string
	delay   int
	groupID string
	dedupID string
}

// parseBatchSendEntries parses SendMessageBatchRequestEntry.N.* form fields.
func parseBatchSendEntries(form url.Values) []batchSendEntry {
	entries := make([]batchSendEntry, 0)
	for i := 1; ; i++ {
		id := form.Get(fmt.Sprintf("SendMessageBatchRequestEntry.%d.Id", i))
		if id == "" {
			break
		}
		body := form.Get(fmt.Sprintf("SendMessageBatchRequestEntry.%d.MessageBody", i))
		delayStr := form.Get(fmt.Sprintf("SendMessageBatchRequestEntry.%d.DelaySeconds", i))
		delay := 0
		if delayStr != "" {
			if v, err := strconv.Atoi(delayStr); err == nil {
				delay = v
			}
		}
		entries = append(entries, batchSendEntry{
			id:      id,
			body:    body,
			delay:   delay,
			groupID: form.Get(fmt.Sprintf("SendMessageBatchRequestEntry.%d.MessageGroupId", i)),
			dedupID: form.Get(fmt.Sprintf("SendMessageBatchRequestEntry.%d.MessageDeduplicationId", i)),
		})
	}
	return entries
}

type batchDeleteEntry struct {
	id            string
	receiptHandle string
}

// parseBatchDeleteEntries parses DeleteMessageBatchRequestEntry.N.* form fields.
func parseBatchDeleteEntries(form url.Values) []batchDeleteEntry {
	entries := make([]batchDeleteEntry, 0)
	for i := 1; ; i++ {
		id := form.Get(fmt.Sprintf("DeleteMessageBatchRequestEntry.%d.Id", i))
		if id == "" {
			break
		}
		rh := form.Get(fmt.Sprintf("DeleteMessageBatchRequestEntry.%d.ReceiptHandle", i))
		entries = append(entries, batchDeleteEntry{id: id, receiptHandle: rh})
	}
	return entries
}

// regionFromURL extracts the region from a queue URL like
// http://sqs.{region}.localhost:4566/{accountId}/{name}
func regionFromURL(queueURL string) string {
	// Strip scheme.
	s := strings.TrimPrefix(queueURL, "http://")
	s = strings.TrimPrefix(s, "https://")
	host := strings.SplitN(s, "/", 2)[0]
	// host is sqs.{region}.localhost:4566 — strip port then split.
	if colon := strings.LastIndex(host, ":"); colon >= 0 {
		host = host[:colon]
	}
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "us-east-1"
}

// accountFromURL extracts the accountId from a queue URL.
func accountFromURL(queueURL string) string {
	u, err := url.Parse(queueURL)
	if err != nil {
		return "000000000000"
	}
	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return "000000000000"
}

// xmlOK wraps a response body in a 200 XML response.
func xmlOK(body any) (*service.Response, error) {
	data, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        data,
		RawContentType: "text/xml",
	}, nil
}

// xmlErr wraps an AWSError in an XML error response.
func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}
