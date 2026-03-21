package sns

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// handleJSON dispatches a JSON-protocol SNS request based on X-Amz-Target.
func (s *SNSService) handleJSON(ctx *service.RequestContext) (*service.Response, error) {
	target := ctx.RawRequest.Header.Get("X-Amz-Target")
	action := strings.TrimPrefix(target, "SNS.")

	switch action {
	case "CreateTopic":
		return jsonCreateTopic(ctx, s.store)
	case "DeleteTopic":
		return jsonDeleteTopic(ctx, s.store)
	case "ListTopics":
		return jsonListTopics(ctx, s.store)
	case "GetTopicAttributes":
		return jsonGetTopicAttributes(ctx, s.store)
	case "SetTopicAttributes":
		return jsonSetTopicAttributes(ctx, s.store)
	case "Subscribe":
		return jsonSubscribe(ctx, s.store)
	case "Unsubscribe":
		return jsonUnsubscribe(ctx, s.store)
	case "ListSubscriptions":
		return jsonListSubscriptions(ctx, s.store)
	case "ListSubscriptionsByTopic":
		return jsonListSubscriptionsByTopic(ctx, s.store)
	case "Publish":
		return jsonPublish(ctx, s.store, s.locator)
	case "TagResource":
		return jsonTagResource(ctx, s.store)
	case "UntagResource":
		return jsonUntagResource(ctx, s.store)
	default:
		return snsJSONErr(service.NewAWSError("InvalidAction",
			"The action "+action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}

// snsParseJSONBody reads the request body and unmarshals it into the given target.
func snsParseJSONBody(ctx *service.RequestContext, target interface{}) error {
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

// snsJSONOK wraps a response body in a 200 JSON response.
func snsJSONOK(body interface{}) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

// snsJSONErr wraps an AWSError in a JSON error response.
func snsJSONErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

// ---- CreateTopic ----

type jsonCreateTopicInput struct {
	Name       string            `json:"Name"`
	Attributes map[string]string `json:"Attributes"`
	Tags       []jsonTag         `json:"Tags"`
}

type jsonTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type jsonCreateTopicOutput struct {
	TopicArn string `json:"TopicArn"`
}

func jsonCreateTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonCreateTopicInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.Name == "" {
		return snsJSONErr(service.ErrValidation("Name is required."))
	}

	tags := make(map[string]string, len(input.Tags))
	for _, t := range input.Tags {
		tags[t.Key] = t.Value
	}

	topic := store.CreateTopic(input.Name, input.Attributes, tags)
	return snsJSONOK(&jsonCreateTopicOutput{TopicArn: topic.ARN})
}

// ---- DeleteTopic ----

type jsonDeleteTopicInput struct {
	TopicArn string `json:"TopicArn"`
}

func jsonDeleteTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonDeleteTopicInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.TopicArn == "" {
		return snsJSONErr(service.ErrValidation("TopicArn is required."))
	}

	if !store.DeleteTopic(input.TopicArn) {
		return snsJSONErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	return snsJSONOK(map[string]interface{}{})
}

// ---- ListTopics ----

type jsonListTopicsInput struct {
	NextToken string `json:"NextToken"`
}

type jsonListTopicsOutput struct {
	Topics []jsonTopicEntry `json:"Topics"`
}

type jsonTopicEntry struct {
	TopicArn string `json:"TopicArn"`
}

func jsonListTopics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonListTopicsInput
	_ = snsParseJSONBody(ctx, &input)

	arns := store.ListTopics()
	entries := make([]jsonTopicEntry, 0, len(arns))
	for _, arn := range arns {
		entries = append(entries, jsonTopicEntry{TopicArn: arn})
	}
	return snsJSONOK(&jsonListTopicsOutput{Topics: entries})
}

// ---- GetTopicAttributes ----

type jsonGetTopicAttributesInput struct {
	TopicArn string `json:"TopicArn"`
}

type jsonGetTopicAttributesOutput struct {
	Attributes map[string]string `json:"Attributes"`
}

func jsonGetTopicAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonGetTopicAttributesInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.TopicArn == "" {
		return snsJSONErr(service.ErrValidation("TopicArn is required."))
	}

	t, ok := store.GetTopic(input.TopicArn)
	if !ok {
		return snsJSONErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	attrs := make(map[string]string)
	attrs["TopicArn"] = t.ARN
	if _, hasDisplay := t.Attributes["DisplayName"]; !hasDisplay {
		attrs["DisplayName"] = t.Name
	}
	for k, v := range t.Attributes {
		attrs[k] = v
	}

	return snsJSONOK(&jsonGetTopicAttributesOutput{Attributes: attrs})
}

// ---- SetTopicAttributes ----

type jsonSetTopicAttributesInput struct {
	TopicArn       string `json:"TopicArn"`
	AttributeName  string `json:"AttributeName"`
	AttributeValue string `json:"AttributeValue"`
}

func jsonSetTopicAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonSetTopicAttributesInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.TopicArn == "" {
		return snsJSONErr(service.ErrValidation("TopicArn is required."))
	}
	if input.AttributeName == "" {
		return snsJSONErr(service.ErrValidation("AttributeName is required."))
	}

	if !store.SetTopicAttribute(input.TopicArn, input.AttributeName, input.AttributeValue) {
		return snsJSONErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	return snsJSONOK(map[string]interface{}{})
}

// ---- Subscribe ----

type jsonSubscribeInput struct {
	TopicArn string `json:"TopicArn"`
	Protocol string `json:"Protocol"`
	Endpoint string `json:"Endpoint"`
}

type jsonSubscribeOutput struct {
	SubscriptionArn string `json:"SubscriptionArn"`
}

func jsonSubscribe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonSubscribeInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.TopicArn == "" {
		return snsJSONErr(service.ErrValidation("TopicArn is required."))
	}
	if input.Protocol == "" {
		return snsJSONErr(service.ErrValidation("Protocol is required."))
	}

	owner := ctx.AccountID
	if owner == "" {
		owner = "000000000000"
	}

	sub, ok := store.Subscribe(input.TopicArn, input.Protocol, input.Endpoint, owner)
	if !ok {
		return snsJSONErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	return snsJSONOK(&jsonSubscribeOutput{SubscriptionArn: sub.ARN})
}

// ---- Unsubscribe ----

type jsonUnsubscribeInput struct {
	SubscriptionArn string `json:"SubscriptionArn"`
}

func jsonUnsubscribe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonUnsubscribeInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.SubscriptionArn == "" {
		return snsJSONErr(service.ErrValidation("SubscriptionArn is required."))
	}

	if !store.Unsubscribe(input.SubscriptionArn) {
		return snsJSONErr(service.NewAWSError("NotFound",
			"Subscription does not exist.", http.StatusNotFound))
	}

	return snsJSONOK(map[string]interface{}{})
}

// ---- ListSubscriptions ----

type jsonListSubscriptionsOutput struct {
	Subscriptions []jsonSubscriptionEntry `json:"Subscriptions"`
}

type jsonSubscriptionEntry struct {
	SubscriptionArn string `json:"SubscriptionArn"`
	Owner           string `json:"Owner"`
	Protocol        string `json:"Protocol"`
	Endpoint        string `json:"Endpoint"`
	TopicArn        string `json:"TopicArn"`
}

func jsonListSubscriptions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	subs := store.ListSubscriptions()

	entries := make([]jsonSubscriptionEntry, 0, len(subs))
	for _, sub := range subs {
		entries = append(entries, jsonSubscriptionEntry{
			SubscriptionArn: sub.ARN,
			Owner:           sub.Owner,
			Protocol:        sub.Protocol,
			Endpoint:        sub.Endpoint,
			TopicArn:        sub.TopicArn,
		})
	}

	return snsJSONOK(&jsonListSubscriptionsOutput{Subscriptions: entries})
}

// ---- ListSubscriptionsByTopic ----

type jsonListSubscriptionsByTopicInput struct {
	TopicArn string `json:"TopicArn"`
}

type jsonListSubscriptionsByTopicOutput struct {
	Subscriptions []jsonSubscriptionEntry `json:"Subscriptions"`
}

func jsonListSubscriptionsByTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonListSubscriptionsByTopicInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.TopicArn == "" {
		return snsJSONErr(service.ErrValidation("TopicArn is required."))
	}

	subs, ok := store.ListSubscriptionsByTopic(input.TopicArn)
	if !ok {
		return snsJSONErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	entries := make([]jsonSubscriptionEntry, 0, len(subs))
	for _, sub := range subs {
		entries = append(entries, jsonSubscriptionEntry{
			SubscriptionArn: sub.ARN,
			Owner:           sub.Owner,
			Protocol:        sub.Protocol,
			Endpoint:        sub.Endpoint,
			TopicArn:        sub.TopicArn,
		})
	}

	return snsJSONOK(&jsonListSubscriptionsByTopicOutput{Subscriptions: entries})
}

// ---- Publish ----

type jsonPublishInput struct {
	TopicArn          string            `json:"TopicArn"`
	TargetArn         string            `json:"TargetArn"`
	Message           string            `json:"Message"`
	Subject           string            `json:"Subject"`
	MessageAttributes map[string]string `json:"MessageAttributes"`
}

type jsonPublishOutput struct {
	MessageId string `json:"MessageId"`
}

func jsonPublish(ctx *service.RequestContext, store *Store, locator ServiceLocator) (*service.Response, error) {
	var input jsonPublishInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}

	topicArn := input.TopicArn
	if topicArn == "" {
		topicArn = input.TargetArn
	}
	if topicArn == "" {
		return snsJSONErr(service.ErrValidation("TopicArn or TargetArn is required."))
	}
	if input.Message == "" {
		return snsJSONErr(service.ErrValidation("Message is required."))
	}

	msgID, ok := store.Publish(topicArn, input.Message, input.Subject, input.MessageAttributes)
	if !ok {
		return snsJSONErr(service.NewAWSError("NotFound",
			"Topic does not exist.", http.StatusNotFound))
	}

	if locator != nil {
		deliverToSQSSubscriptions(store, locator, topicArn, msgID, input.Message, input.Subject)
	}

	return snsJSONOK(&jsonPublishOutput{MessageId: msgID})
}

// ---- TagResource ----

type jsonTagResourceInput struct {
	ResourceArn string    `json:"ResourceArn"`
	Tags        []jsonTag `json:"Tags"`
}

func jsonTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonTagResourceInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.ResourceArn == "" {
		return snsJSONErr(service.ErrValidation("ResourceArn is required."))
	}

	tags := make(map[string]string, len(input.Tags))
	for _, t := range input.Tags {
		tags[t.Key] = t.Value
	}

	if !store.TagResource(input.ResourceArn, tags) {
		return snsJSONErr(service.NewAWSError("NotFound",
			"Resource does not exist.", http.StatusNotFound))
	}

	return snsJSONOK(map[string]interface{}{})
}

// ---- UntagResource ----

type jsonUntagResourceInput struct {
	ResourceArn string   `json:"ResourceArn"`
	TagKeys     []string `json:"TagKeys"`
}

func jsonUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var input jsonUntagResourceInput
	if err := snsParseJSONBody(ctx, &input); err != nil {
		return snsJSONErr(service.ErrValidation("Invalid JSON: " + err.Error()))
	}
	if input.ResourceArn == "" {
		return snsJSONErr(service.ErrValidation("ResourceArn is required."))
	}

	if !store.UntagResource(input.ResourceArn, input.TagKeys) {
		return snsJSONErr(service.NewAWSError("NotFound",
			"Resource does not exist.", http.StatusNotFound))
	}

	return snsJSONOK(map[string]interface{}{})
}
