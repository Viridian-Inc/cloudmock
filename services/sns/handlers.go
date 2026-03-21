package sns

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

// ---- CreateTopic ----

type xmlCreateTopicResponse struct {
	XMLName xml.Name              `xml:"CreateTopicResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlCreateTopicResult  `xml:"CreateTopicResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlCreateTopicResult struct {
	TopicArn string `xml:"TopicArn"`
}

func handleCreateTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("Name")
	if name == "" {
		return xmlErr(service.ErrValidation("Name is required."))
	}

	attrs := parseAttributes(form)
	tags := parseTags(form)

	t := store.CreateTopic(name, attrs, tags)

	resp := &xmlCreateTopicResponse{
		Xmlns:  snsXmlns,
		Result: xmlCreateTopicResult{TopicArn: t.ARN},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- DeleteTopic ----

type xmlDeleteTopicResponse struct {
	XMLName xml.Name            `xml:"DeleteTopicResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	topicArn := form.Get("TopicArn")
	if topicArn == "" {
		return xmlErr(service.ErrValidation("TopicArn is required."))
	}

	if !store.DeleteTopic(topicArn) {
		return xmlErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteTopicResponse{
		Xmlns: snsXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ListTopics ----

type xmlListTopicsResponse struct {
	XMLName xml.Name             `xml:"ListTopicsResponse"`
	Xmlns   string               `xml:"xmlns,attr"`
	Result  xmlListTopicsResult  `xml:"ListTopicsResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlListTopicsResult struct {
	Topics []xmlTopicEntry `xml:"Topics>member"`
}

type xmlTopicEntry struct {
	TopicArn string `xml:"TopicArn"`
}

func handleListTopics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	arns := store.ListTopics()

	entries := make([]xmlTopicEntry, 0, len(arns))
	for _, arn := range arns {
		entries = append(entries, xmlTopicEntry{TopicArn: arn})
	}

	resp := &xmlListTopicsResponse{
		Xmlns:  snsXmlns,
		Result: xmlListTopicsResult{Topics: entries},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- GetTopicAttributes ----

type xmlGetTopicAttributesResponse struct {
	XMLName xml.Name                      `xml:"GetTopicAttributesResponse"`
	Xmlns   string                        `xml:"xmlns,attr"`
	Result  xmlGetTopicAttributesResult   `xml:"GetTopicAttributesResult"`
	Meta    xmlResponseMetadata           `xml:"ResponseMetadata"`
}

type xmlGetTopicAttributesResult struct {
	Attributes []xmlAttributeEntry `xml:"Attributes>entry"`
}

type xmlAttributeEntry struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

func handleGetTopicAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	topicArn := form.Get("TopicArn")
	if topicArn == "" {
		return xmlErr(service.ErrValidation("TopicArn is required."))
	}

	t, ok := store.GetTopic(topicArn)
	if !ok {
		return xmlErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	entries := make([]xmlAttributeEntry, 0, len(t.Attributes)+2)
	// Always include TopicArn and DisplayName as standard attributes.
	entries = append(entries, xmlAttributeEntry{Key: "TopicArn", Value: t.ARN})
	if _, hasDisplay := t.Attributes["DisplayName"]; !hasDisplay {
		entries = append(entries, xmlAttributeEntry{Key: "DisplayName", Value: t.Name})
	}
	for k, v := range t.Attributes {
		entries = append(entries, xmlAttributeEntry{Key: k, Value: v})
	}

	resp := &xmlGetTopicAttributesResponse{
		Xmlns:  snsXmlns,
		Result: xmlGetTopicAttributesResult{Attributes: entries},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- SetTopicAttributes ----

type xmlSetTopicAttributesResponse struct {
	XMLName xml.Name            `xml:"SetTopicAttributesResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleSetTopicAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	topicArn := form.Get("TopicArn")
	attrName := form.Get("AttributeName")
	attrValue := form.Get("AttributeValue")

	if topicArn == "" {
		return xmlErr(service.ErrValidation("TopicArn is required."))
	}
	if attrName == "" {
		return xmlErr(service.ErrValidation("AttributeName is required."))
	}

	if !store.SetTopicAttribute(topicArn, attrName, attrValue) {
		return xmlErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	return xmlOK(&xmlSetTopicAttributesResponse{
		Xmlns: snsXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- Subscribe ----

type xmlSubscribeResponse struct {
	XMLName xml.Name            `xml:"SubscribeResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlSubscribeResult  `xml:"SubscribeResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlSubscribeResult struct {
	SubscriptionArn string `xml:"SubscriptionArn"`
}

func handleSubscribe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	topicArn := form.Get("TopicArn")
	protocol := form.Get("Protocol")
	endpoint := form.Get("Endpoint")

	if topicArn == "" {
		return xmlErr(service.ErrValidation("TopicArn is required."))
	}
	if protocol == "" {
		return xmlErr(service.ErrValidation("Protocol is required."))
	}

	owner := ctx.AccountID
	if owner == "" {
		owner = "000000000000"
	}

	sub, ok := store.Subscribe(topicArn, protocol, endpoint, owner)
	if !ok {
		return xmlErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	resp := &xmlSubscribeResponse{
		Xmlns:  snsXmlns,
		Result: xmlSubscribeResult{SubscriptionArn: sub.ARN},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- Unsubscribe ----

type xmlUnsubscribeResponse struct {
	XMLName xml.Name            `xml:"UnsubscribeResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleUnsubscribe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	subARN := form.Get("SubscriptionArn")
	if subARN == "" {
		return xmlErr(service.ErrValidation("SubscriptionArn is required."))
	}

	if !store.Unsubscribe(subARN) {
		return xmlErr(service.NewAWSError("NotFound",
			"Subscription does not exist.", http.StatusNotFound))
	}

	return xmlOK(&xmlUnsubscribeResponse{
		Xmlns: snsXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ListSubscriptions ----

type xmlListSubscriptionsResponse struct {
	XMLName xml.Name                    `xml:"ListSubscriptionsResponse"`
	Xmlns   string                      `xml:"xmlns,attr"`
	Result  xmlListSubscriptionsResult  `xml:"ListSubscriptionsResult"`
	Meta    xmlResponseMetadata         `xml:"ResponseMetadata"`
}

type xmlListSubscriptionsResult struct {
	Subscriptions []xmlSubscriptionEntry `xml:"Subscriptions>member"`
}

type xmlSubscriptionEntry struct {
	SubscriptionArn string `xml:"SubscriptionArn"`
	Owner           string `xml:"Owner"`
	Protocol        string `xml:"Protocol"`
	Endpoint        string `xml:"Endpoint"`
	TopicArn        string `xml:"TopicArn"`
}

func handleListSubscriptions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	subs := store.ListSubscriptions()

	entries := make([]xmlSubscriptionEntry, 0, len(subs))
	for _, sub := range subs {
		entries = append(entries, xmlSubscriptionEntry{
			SubscriptionArn: sub.ARN,
			Owner:           sub.Owner,
			Protocol:        sub.Protocol,
			Endpoint:        sub.Endpoint,
			TopicArn:        sub.TopicArn,
		})
	}

	resp := &xmlListSubscriptionsResponse{
		Xmlns:  snsXmlns,
		Result: xmlListSubscriptionsResult{Subscriptions: entries},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- ListSubscriptionsByTopic ----

type xmlListSubscriptionsByTopicResponse struct {
	XMLName xml.Name                           `xml:"ListSubscriptionsByTopicResponse"`
	Xmlns   string                             `xml:"xmlns,attr"`
	Result  xmlListSubscriptionsByTopicResult  `xml:"ListSubscriptionsByTopicResult"`
	Meta    xmlResponseMetadata                `xml:"ResponseMetadata"`
}

type xmlListSubscriptionsByTopicResult struct {
	Subscriptions []xmlSubscriptionEntry `xml:"Subscriptions>member"`
}

func handleListSubscriptionsByTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	topicArn := form.Get("TopicArn")
	if topicArn == "" {
		return xmlErr(service.ErrValidation("TopicArn is required."))
	}

	subs, ok := store.ListSubscriptionsByTopic(topicArn)
	if !ok {
		return xmlErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	entries := make([]xmlSubscriptionEntry, 0, len(subs))
	for _, sub := range subs {
		entries = append(entries, xmlSubscriptionEntry{
			SubscriptionArn: sub.ARN,
			Owner:           sub.Owner,
			Protocol:        sub.Protocol,
			Endpoint:        sub.Endpoint,
			TopicArn:        sub.TopicArn,
		})
	}

	resp := &xmlListSubscriptionsByTopicResponse{
		Xmlns:  snsXmlns,
		Result: xmlListSubscriptionsByTopicResult{Subscriptions: entries},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- Publish ----

type xmlPublishResponse struct {
	XMLName xml.Name            `xml:"PublishResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlPublishResult    `xml:"PublishResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlPublishResult struct {
	MessageId string `xml:"MessageId"`
}

func handlePublish(ctx *service.RequestContext, store *Store, locator ServiceLocator) (*service.Response, error) {
	form := parseForm(ctx)

	// TopicArn or TargetArn may be used.
	topicArn := form.Get("TopicArn")
	if topicArn == "" {
		topicArn = form.Get("TargetArn")
	}
	if topicArn == "" {
		return xmlErr(service.ErrValidation("TopicArn or TargetArn is required."))
	}

	message := form.Get("Message")
	if message == "" {
		return xmlErr(service.ErrValidation("Message is required."))
	}

	subject := form.Get("Subject")
	msgAttrs := parseMessageAttributes(form)

	msgID, ok := store.Publish(topicArn, message, subject, msgAttrs)
	if !ok {
		return xmlErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	// Deliver to subscriptions (SNS → SQS fan-out, SNS → Lambda).
	if locator != nil {
		deliverToSQSSubscriptions(store, locator, topicArn, msgID, message, subject)
		deliverToLambdaSubscriptions(store, locator, topicArn, msgID, message, subject)
	}

	resp := &xmlPublishResponse{
		Xmlns:  snsXmlns,
		Result: xmlPublishResult{MessageId: msgID},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// deliverToSQSSubscriptions iterates SNS subscriptions with protocol "sqs"
// and enqueues an SNS notification wrapper message into the target SQS queue.
func deliverToSQSSubscriptions(store *Store, locator ServiceLocator, topicArn, msgID, message, subject string) {
	subs, ok := store.ListSubscriptionsByTopic(topicArn)
	if !ok {
		return
	}

	for _, sub := range subs {
		if sub.Protocol != "sqs" {
			continue
		}

		// The endpoint is the SQS queue ARN. Extract the queue name from it.
		queueName := extractQueueNameFromARN(sub.Endpoint)
		if queueName == "" {
			continue
		}

		// Build the SNS notification JSON envelope that mirrors real AWS behavior.
		notification := buildSNSNotification(topicArn, msgID, message, subject, sub.ARN)

		// Find the SQS service and enqueue directly.
		enqueueSNSToSQS(locator, queueName, notification)
	}
}

// extractQueueNameFromARN extracts the queue name from an SQS ARN like
// "arn:aws:sqs:us-east-1:123456789012:my-queue" → "my-queue"
func extractQueueNameFromARN(arn string) string {
	parts := splitARN(arn)
	if len(parts) < 6 {
		return ""
	}
	return parts[5]
}

// splitARN splits an ARN into its components.
func splitARN(arn string) []string {
	// arn:aws:service:region:account:resource
	result := make([]string, 0, 6)
	s := arn
	for i := 0; i < 5; i++ {
		idx := indexOf(s, ':')
		if idx < 0 {
			result = append(result, s)
			return result
		}
		result = append(result, s[:idx])
		s = s[idx+1:]
	}
	result = append(result, s)
	return result
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// buildSNSNotification creates the JSON envelope that SNS puts around messages
// when delivering to SQS, matching the AWS format.
func buildSNSNotification(topicArn, messageID, message, subject, subscriptionArn string) string {
	// Simplified SNS notification envelope matching AWS format.
	return fmt.Sprintf(`{"Type":"Notification","MessageId":"%s","TopicArn":"%s","Subject":"%s","Message":%s,"SubscribeURL":"","UnsubscribeURL":"","Timestamp":"%s"}`,
		messageID, topicArn, subject, jsonEscape(message), newUUID())
}

// SQSEnqueuer is an interface for directly enqueuing messages into SQS queues.
// This avoids a hard dependency on the sqs package.
type SQSEnqueuer interface {
	EnqueueDirect(queueName, messageBody string) bool
}

// enqueueSNSToSQS finds the SQS service via the locator and enqueues the message.
func enqueueSNSToSQS(locator ServiceLocator, queueName, messageBody string) {
	svc, err := locator.Lookup("sqs")
	if err != nil {
		return
	}
	if enqueuer, ok := svc.(SQSEnqueuer); ok {
		enqueuer.EnqueueDirect(queueName, messageBody)
	}
}

// LambdaInvoker is an interface for invoking Lambda functions directly.
type LambdaInvoker interface {
	InvokeDirect(functionName string, event []byte) ([]byte, error)
}

// deliverToLambdaSubscriptions iterates SNS subscriptions with protocol "lambda"
// and invokes the target Lambda function with an SNS event payload.
func deliverToLambdaSubscriptions(store *Store, locator ServiceLocator, topicArn, msgID, message, subject string) {
	subs, ok := store.ListSubscriptionsByTopic(topicArn)
	if !ok {
		return
	}

	for _, sub := range subs {
		if sub.Protocol != "lambda" {
			continue
		}

		// The endpoint is the Lambda function ARN. Extract the function name.
		funcName := extractFunctionNameFromARN(sub.Endpoint)
		if funcName == "" {
			continue
		}

		// Build the SNS event payload matching AWS format.
		payload := buildSNSLambdaEvent(topicArn, msgID, message, subject, sub.ARN)

		// Find the Lambda service and invoke.
		invokeSNSToLambda(locator, funcName, []byte(payload))
	}
}

// extractFunctionNameFromARN extracts the function name from a Lambda ARN like
// "arn:aws:lambda:us-east-1:123456789012:function:my-func" → "my-func"
func extractFunctionNameFromARN(arn string) string {
	parts := splitARN(arn)
	if len(parts) < 6 {
		return ""
	}
	// Lambda ARN resource part is "function:name" — the splitARN only splits on 5 colons,
	// so parts[5] = "function:my-func". Extract after "function:".
	resource := parts[5]
	const prefix = "function:"
	if len(resource) > len(prefix) && resource[:len(prefix)] == prefix {
		return resource[len(prefix):]
	}
	return resource
}

// buildSNSLambdaEvent creates the SNS event payload for Lambda invocation,
// matching the AWS format.
func buildSNSLambdaEvent(topicArn, messageID, message, subject, subscriptionArn string) string {
	return fmt.Sprintf(`{"Records":[{"EventSource":"aws:sns","EventVersion":"1.0","EventSubscriptionArn":"%s","Sns":{"Type":"Notification","MessageId":"%s","TopicArn":"%s","Subject":"%s","Message":%s,"Timestamp":"%s"}}]}`,
		subscriptionArn, messageID, topicArn, subject, jsonEscape(message), newUUID())
}

// invokeSNSToLambda finds the Lambda service via the locator and invokes the function.
func invokeSNSToLambda(locator ServiceLocator, functionName string, payload []byte) {
	svc, err := locator.Lookup("lambda")
	if err != nil {
		return
	}
	if invoker, ok := svc.(LambdaInvoker); ok {
		invoker.InvokeDirect(functionName, payload)
	}
}

// jsonEscape wraps a string in JSON quotes, escaping as needed.
func jsonEscape(s string) string {
	// Simple JSON string encoding.
	result := `"`
	for _, c := range s {
		switch c {
		case '"':
			result += `\"`
		case '\\':
			result += `\\`
		case '\n':
			result += `\n`
		case '\r':
			result += `\r`
		case '\t':
			result += `\t`
		default:
			result += string(c)
		}
	}
	result += `"`
	return result
}

// ---- TagResource ----

type xmlTagResourceResponse struct {
	XMLName xml.Name            `xml:"TagResourceResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	resourceArn := form.Get("ResourceArn")
	if resourceArn == "" {
		return xmlErr(service.ErrValidation("ResourceArn is required."))
	}

	tags := parseTags(form)

	if !store.TagResource(resourceArn, tags) {
		return xmlErr(service.NewAWSError("NotFound",
			"Resource does not exist.", http.StatusNotFound))
	}

	return xmlOK(&xmlTagResourceResponse{
		Xmlns: snsXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- UntagResource ----

type xmlUntagResourceResponse struct {
	XMLName xml.Name            `xml:"UntagResourceResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	resourceArn := form.Get("ResourceArn")
	if resourceArn == "" {
		return xmlErr(service.ErrValidation("ResourceArn is required."))
	}

	// TagKeys.member.1, TagKeys.member.2, ...
	keys := make([]string, 0)
	for i := 1; ; i++ {
		k := form.Get(fmt.Sprintf("TagKeys.member.%d", i))
		if k == "" {
			break
		}
		keys = append(keys, k)
	}

	if !store.UntagResource(resourceArn, keys) {
		return xmlErr(service.NewAWSError("NotFound",
			"Resource does not exist.", http.StatusNotFound))
	}

	return xmlOK(&xmlUntagResourceResponse{
		Xmlns: snsXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- helper functions ----

const snsXmlns = "http://sns.amazonaws.com/doc/2010-03-31/"

// parseForm merges query-string params and form-encoded body into url.Values.
func parseForm(ctx *service.RequestContext) url.Values {
	form := make(url.Values)
	for k, v := range ctx.Params {
		form.Set(k, v)
	}
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

// parseAttributes parses Attributes.entry.N.key / Attributes.entry.N.value pairs.
// Also handles the flat Attribute.N.Name / Attribute.N.Value format used by the SDK.
func parseAttributes(form url.Values) map[string]string {
	attrs := make(map[string]string)
	for i := 1; ; i++ {
		name := form.Get(fmt.Sprintf("Attributes.entry.%d.key", i))
		if name == "" {
			name = form.Get(fmt.Sprintf("Attribute.%d.Name", i))
		}
		if name == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Attributes.entry.%d.value", i))
		if val == "" {
			val = form.Get(fmt.Sprintf("Attribute.%d.Value", i))
		}
		attrs[name] = val
	}
	return attrs
}

// parseTags parses Tags.member.N.Key / Tags.member.N.Value pairs.
func parseTags(form url.Values) map[string]string {
	tags := make(map[string]string)
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Tags.member.%d.Key", i))
		if key == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Tags.member.%d.Value", i))
		tags[key] = val
	}
	return tags
}

// parseMessageAttributes parses MessageAttributes.entry.N.key / .value.DataType / .value.StringValue.
func parseMessageAttributes(form url.Values) map[string]string {
	attrs := make(map[string]string)
	for i := 1; ; i++ {
		name := form.Get(fmt.Sprintf("MessageAttributes.entry.%d.Name", i))
		if name == "" {
			name = form.Get(fmt.Sprintf("MessageAttribute.%d.Name", i))
		}
		if name == "" {
			break
		}
		val := form.Get(fmt.Sprintf("MessageAttributes.entry.%d.Value.StringValue", i))
		if val == "" {
			val = form.Get(fmt.Sprintf("MessageAttribute.%d.Value.StringValue", i))
		}
		attrs[name] = val
	}
	return attrs
}

// xmlOK wraps a response body in a 200 XML response.
func xmlOK(body interface{}) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

// xmlErr wraps an AWSError in an XML error response.
func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

// newUUID returns a random UUID-shaped identifier.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// randomHex returns n random bytes as a hex string.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
