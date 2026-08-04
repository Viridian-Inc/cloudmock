package eventgrid

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const (
	eventGridAPIVersion        = "2025-02-15"
	eventGridPublishAPIVersion = "2018-01-01"
	eventGridMaxPublishBytes   = 1024 * 1024
	eventGridMaxPublishEvents  = 5000
)

// EventGridService implements Azure Event Grid control-plane and custom topic publish APIs.
type EventGridService struct {
	mu              sync.RWMutex
	topics          map[string]Topic
	subscriptions   map[string]EventSubscription
	topicKeys       map[string]topicSharedAccessKeys
	publishedEvents map[string][]map[string]any
	keyGeneration   uint64
}

func New() *EventGridService {
	return &EventGridService{
		topics:          make(map[string]Topic),
		subscriptions:   make(map[string]EventSubscription),
		topicKeys:       make(map[string]topicSharedAccessKeys),
		publishedEvents: make(map[string][]map[string]any),
	}
}

func (s *EventGridService) Name() string { return "Microsoft.EventGrid" }

func (s *EventGridService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateTopic", Method: http.MethodPut, IAMAction: "azure:Microsoft.EventGrid/topics/write"},
		{Name: "GetTopic", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventGrid/topics/read"},
		{Name: "ListTopics", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventGrid/topics/read"},
		{Name: "DeleteTopic", Method: http.MethodDelete, IAMAction: "azure:Microsoft.EventGrid/topics/delete"},
		{Name: "ListSharedAccessKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.EventGrid/topics/listKeys/action"},
		{Name: "RegenerateKey", Method: http.MethodPost, IAMAction: "azure:Microsoft.EventGrid/topics/regenerateKey/action"},
		{Name: "CreateOrUpdateTopicEventSubscription", Method: http.MethodPut, IAMAction: "azure:Microsoft.EventGrid/topics/eventSubscriptions/write"},
		{Name: "GetTopicEventSubscription", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventGrid/topics/eventSubscriptions/read"},
		{Name: "ListTopicEventSubscriptions", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventGrid/topics/eventSubscriptions/read"},
		{Name: "DeleteTopicEventSubscription", Method: http.MethodDelete, IAMAction: "azure:Microsoft.EventGrid/topics/eventSubscriptions/delete"},
		{Name: "GetDeliveryAttributes", Method: http.MethodPost, IAMAction: "azure:Microsoft.EventGrid/topics/eventSubscriptions/getDeliveryAttributes/action"},
		{Name: "PublishEvents", Method: http.MethodPost, IAMAction: "azure:Microsoft.EventGrid/topics/publish/action"},
	}
}

func (s *EventGridService) HealthCheck() error { return nil }

func (s *EventGridService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.EventGrid/topics", APIVersion: eventGridAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.EventGrid/publish", APIVersion: eventGridPublishAPIVersion},
	}
}

func (s *EventGridService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.EventGrid/topics") ||
		strings.EqualFold(resourceType, "Microsoft.EventGrid/topics/eventSubscriptions")
}

func (s *EventGridService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Event Grid template resource type %q", stringValue(resource["type"]))
	}
	resourceType := stringValue(resource["type"])
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Event Grid template resource is missing name")
	}

	body := map[string]any{
		"location":   resource["location"],
		"tags":       resource["tags"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}

	var resp *service.Response
	switch {
	case strings.EqualFold(resourceType, "Microsoft.EventGrid/topics"):
		resp, err = s.createOrUpdateTopic(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.EventGrid/topics/eventSubscriptions"):
		topicName, subscriptionName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Event Grid event subscription template resource name must be {topic}/{eventSubscription}")
		}
		resp, err = s.createOrUpdateEventSubscription(subscriptionID, resourceGroup, topicName, subscriptionName, data)
	default:
		err = fmt.Errorf("unsupported Event Grid template resource type %q", resourceType)
	}
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *EventGridService) TemplateListOperationResult(subscriptionID, resourceGroup string, resource map[string]any, operation string) (any, bool) {
	if !strings.EqualFold(operation, "listKeys") || !strings.EqualFold(stringValue(resource["type"]), "Microsoft.EventGrid/topics") {
		return nil, false
	}

	name := stringValue(resource["name"])
	if name == "" {
		return nil, false
	}

	s.mu.RLock()
	keys, ok := s.topicKeys[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return map[string]any{"key1": keys.Key1, "key2": keys.Key2}, true
}

func (s *EventGridService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}
	if isEventGridPublishRequest(ctx.RawRequest) {
		return s.publishEvents(ctx)
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "topics") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Grid route is not implemented.")
	}
	if route.ChildType != "" {
		if route.ChildName == "" && strings.EqualFold(route.ChildType, "listKeys") {
			return s.listSharedAccessKeys(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		if route.ChildName == "" && strings.EqualFold(route.ChildType, "regenerateKey") {
			return s.regenerateKey(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
		}
		if strings.EqualFold(route.ChildType, "eventSubscriptions") {
			return s.handleEventSubscriptionRequest(ctx, route)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Grid route is not implemented.")
	}
	return s.handleTopicRequest(ctx, route)
}

func (s *EventGridService) handleTopicRequest(ctx *service.RequestContext, route eventGridRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listTopics(ctx.RawRequest, route.SubscriptionID, route.ResourceGroup, route.SubscriptionScope)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateTopic(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getTopic(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteTopic(ctx.Region, route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *EventGridService) handleEventSubscriptionRequest(ctx *service.RequestContext, route eventGridRoute) (*service.Response, error) {
	if route.ChildAction != "" {
		if strings.EqualFold(route.ChildAction, "getDeliveryAttributes") && ctx.RawRequest.Method == http.MethodPost {
			return s.getEventSubscriptionDeliveryAttributes(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Grid route is not implemented.")
	}
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listEventSubscriptions(ctx.RawRequest, route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateEventSubscription(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getEventSubscription(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deleteEventSubscription(ctx.Region, route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *EventGridService) createOrUpdateTopic(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "eastus"
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	applyTopicDefaults(input.Properties)
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["endpoint"]; !ok {
		input.Properties["endpoint"] = eventGridTopicEndpoint(name, input.Location)
	}

	topic := Topic{
		ID:         topicID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.EventGrid/topics",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.topics[key]
	s.topics[key] = topic
	if _, ok := s.topicKeys[key]; !ok {
		s.topicKeys[key] = topicSharedAccessKeys{
			Key1: s.generateTopicKeyLocked(key, "key1"),
			Key2: s.generateTopicKeyLocked(key, "key2"),
		}
	}
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, topic)
}

func applyTopicDefaults(properties map[string]any) {
	if _, ok := properties["disableLocalAuth"]; !ok {
		properties["disableLocalAuth"] = false
	}
	if _, ok := properties["inputSchema"]; !ok {
		properties["inputSchema"] = "EventGridSchema"
	}
	if _, ok := properties["publicNetworkAccess"]; !ok {
		properties["publicNetworkAccess"] = "Enabled"
	}
}

func (s *EventGridService) getTopic(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	topic, ok := s.topics[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Grid topic %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, topic)
}

func (s *EventGridService) listTopics(req *http.Request, subscriptionID, resourceGroup string, subscriptionScope bool) (*service.Response, error) {
	matches, top, skip, err := eventGridTopicListOptions(req)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadRequest", err.Error())
	}
	prefix := strings.ToLower(subscriptionID) + "/"
	if !subscriptionScope {
		prefix += strings.ToLower(resourceGroup) + "/"
	}

	s.mu.RLock()
	values := make([]Topic, 0)
	for key, topic := range s.topics {
		if strings.HasPrefix(key, prefix) && matches(topic) {
			values = append(values, topic)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	page, nextLink := pageEventGridList(values, top, skip, req)
	result := map[string]any{"value": page}
	if nextLink != "" {
		result["nextLink"] = nextLink
	}
	return azurearm.JSONResponse(http.StatusOK, result)
}

func (s *EventGridService) deleteTopic(region, subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.topics[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.topics, key)
	childPrefix := key + "/"
	for subscriptionKey := range s.subscriptions {
		if strings.HasPrefix(subscriptionKey, childPrefix) {
			delete(s.subscriptions, subscriptionKey)
		}
	}
	delete(s.topicKeys, key)
	delete(s.publishedEvents, key)
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers:    eventGridDeleteOperationHeaders(region, subscriptionID, resourceGroup, topicID(subscriptionID, resourceGroup, name)),
	}, nil
}

func (s *EventGridService) createOrUpdateEventSubscription(subscriptionID, resourceGroup, topicName, subscriptionName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	applyEventSubscriptionDefaults(input.Properties, topicID(subscriptionID, resourceGroup, topicName))
	input.Properties["provisioningState"] = "Succeeded"

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.topics[resourceKey(subscriptionID, resourceGroup, topicName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Grid topic %q could not be found.", topicName))
	}

	eventSubscription := EventSubscription{
		ID:         eventSubscriptionID(subscriptionID, resourceGroup, topicName, subscriptionName),
		Name:       subscriptionName,
		Type:       "Microsoft.EventGrid/topics/eventSubscriptions",
		Properties: input.Properties,
	}
	key := childKey(subscriptionID, resourceGroup, topicName, subscriptionName)
	_, existed := s.subscriptions[key]
	s.subscriptions[key] = eventSubscription

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, eventSubscription)
}

func (s *EventGridService) getEventSubscription(subscriptionID, resourceGroup, topicName, subscriptionName string) (*service.Response, error) {
	s.mu.RLock()
	eventSubscription, ok := s.subscriptions[childKey(subscriptionID, resourceGroup, topicName, subscriptionName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Grid event subscription %q could not be found.", subscriptionName))
	}
	return azurearm.JSONResponse(http.StatusOK, eventSubscription)
}

func (s *EventGridService) getEventSubscriptionDeliveryAttributes(subscriptionID, resourceGroup, topicName, subscriptionName string) (*service.Response, error) {
	s.mu.RLock()
	eventSubscription, ok := s.subscriptions[childKey(subscriptionID, resourceGroup, topicName, subscriptionName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Grid event subscription %q could not be found.", subscriptionName))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": eventGridDeliveryAttributeMappings(eventSubscription.Properties)})
}

func (s *EventGridService) listEventSubscriptions(req *http.Request, subscriptionID, resourceGroup, topicName string) (*service.Response, error) {
	matchesName, top, skip, err := eventGridTopicListOptions(req)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadRequest", err.Error())
	}
	topicKey := resourceKey(subscriptionID, resourceGroup, topicName)

	s.mu.RLock()
	_, topicExists := s.topics[topicKey]
	values := make([]EventSubscription, 0)
	prefix := topicKey + "/"
	for key, eventSubscription := range s.subscriptions {
		if strings.HasPrefix(key, prefix) && matchesName(Topic{Name: eventSubscription.Name}) {
			values = append(values, eventSubscription)
		}
	}
	s.mu.RUnlock()

	if !topicExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Grid topic %q could not be found.", topicName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	page, nextLink := pageEventGridList(values, top, skip, req)
	result := map[string]any{"value": page}
	if nextLink != "" {
		result["nextLink"] = nextLink
	}
	return azurearm.JSONResponse(http.StatusOK, result)
}

func (s *EventGridService) deleteEventSubscription(region, subscriptionID, resourceGroup, topicName, subscriptionName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := childKey(subscriptionID, resourceGroup, topicName, subscriptionName)
	if _, ok := s.subscriptions[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.subscriptions, key)
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers:    eventGridDeleteOperationHeaders(region, subscriptionID, resourceGroup, eventSubscriptionID(subscriptionID, resourceGroup, topicName, subscriptionName)),
	}, nil
}

func (s *EventGridService) listSharedAccessKeys(subscriptionID, resourceGroup, topicName string) (*service.Response, error) {
	s.mu.RLock()
	keys, ok := s.topicKeys[resourceKey(subscriptionID, resourceGroup, topicName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Grid topic %q could not be found.", topicName))
	}
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *EventGridService) regenerateKey(subscriptionID, resourceGroup, topicName string, body []byte) (*service.Response, error) {
	var input struct {
		KeyName string `json:"keyName"`
	}
	if err := gojson.Unmarshal(body, &input); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadRequest", "The key regeneration request content was invalid.")
	}
	input.KeyName = strings.ToLower(strings.TrimSpace(input.KeyName))
	if input.KeyName != "key1" && input.KeyName != "key2" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadRequest", "The keyName field must be key1 or key2.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, topicName)
	keys, ok := s.topicKeys[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Grid topic %q could not be found.", topicName))
	}
	if input.KeyName == "key1" {
		keys.Key1 = s.generateTopicKeyLocked(key, "key1")
	} else {
		keys.Key2 = s.generateTopicKeyLocked(key, "key2")
	}
	s.topicKeys[key] = keys
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *EventGridService) publishEvents(ctx *service.RequestContext) (*service.Response, error) {
	if len(ctx.Body) > eventGridMaxPublishBytes {
		return azurearm.ErrorResponse(http.StatusRequestEntityTooLarge, "RequestEntityTooLarge", "Array or event exceeds size limits.")
	}

	var events []map[string]any
	if err := gojson.Unmarshal(ctx.Body, &events); err != nil || len(events) == 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadRequest", "Event data has incorrect format.")
	}
	if len(events) > eventGridMaxPublishEvents {
		return azurearm.ErrorResponse(http.StatusRequestEntityTooLarge, "RequestEntityTooLarge", "Array or event exceeds size limits.")
	}

	topicKey, topic, ok := s.topicForEndpoint(ctx.RawRequest)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "Incorrect endpoint.")
	}
	if !s.canPublish(ctx.RawRequest, topicKey, topic) {
		return azurearm.ErrorResponse(http.StatusUnauthorized, "Unauthorized", "Invalid access key.")
	}
	inputSchema := stringValue(topic.Properties["inputSchema"])
	for _, event := range events {
		if !isValidEventGridPublishEvent(event, inputSchema, topic.ID) {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadRequest", "Event data has incorrect format.")
		}
		normalizeEventGridPublishEvent(event, inputSchema, topic.ID)
	}

	s.mu.Lock()
	s.publishedEvents[topicKey] = append(s.publishedEvents[topicKey], cloneEventGridEvents(events)...)
	s.mu.Unlock()

	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *EventGridService) topicForEndpoint(req *http.Request) (string, Topic, bool) {
	endpoint := eventGridRequestEndpoint(req)

	s.mu.RLock()
	defer s.mu.RUnlock()
	for key, topic := range s.topics {
		if strings.EqualFold(strings.TrimRight(stringValue(topic.Properties["endpoint"]), "/"), endpoint) {
			return key, topic, true
		}
	}
	return "", Topic{}, false
}

func (s *EventGridService) canPublish(req *http.Request, topicKey string, topic Topic) bool {
	if eventGridBoolValue(topic.Properties["disableLocalAuth"]) {
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(req.Header.Get("Authorization"))), "bearer ")
	}

	s.mu.RLock()
	keys, ok := s.topicKeys[topicKey]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	key := eventGridAccessKey(req)
	if key != "" && (key == keys.Key1 || key == keys.Key2) {
		return true
	}
	return eventGridSASAuthorizes(req, keys)
}

func eventGridAccessKey(req *http.Request) string {
	if key := strings.TrimSpace(req.Header.Get("aeg-sas-key")); key != "" {
		return key
	}
	return strings.TrimSpace(req.URL.Query().Get("aeg-sas-key"))
}

func eventGridBoolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func eventGridSASAuthorizes(req *http.Request, keys topicSharedAccessKeys) bool {
	token := strings.TrimSpace(req.Header.Get("aeg-sas-token"))
	if token == "" {
		const prefix = "sharedaccesssignature "
		auth := strings.TrimSpace(req.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), prefix) {
			token = strings.TrimSpace(auth[len(prefix):])
		}
	}
	if token == "" {
		return false
	}

	values, err := url.ParseQuery(token)
	if err != nil {
		return false
	}
	resource := strings.TrimRight(values.Get("r"), "/")
	expiration := values.Get("e")
	signature := values.Get("s")
	if resource == "" || expiration == "" || signature == "" {
		return false
	}
	if !strings.HasPrefix(strings.ToLower(eventGridRequestEndpoint(req)), strings.ToLower(resource)) {
		return false
	}
	expiry, ok := parseEventGridSASExpiration(expiration)
	if !ok || !time.Now().UTC().Before(expiry) {
		return false
	}

	unsigned := "r=" + url.QueryEscape(resource) + "&e=" + url.QueryEscape(expiration)
	return eventGridSASSignatureMatches(unsigned, signature, keys.Key1) ||
		eventGridSASSignatureMatches(unsigned, signature, keys.Key2)
}

func parseEventGridSASExpiration(value string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, "1/2/2006 3:04:05 PM", "1/2/2006 3:04:05 pm"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func eventGridSASSignatureMatches(unsigned, signature, key string) bool {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return false
	}
	expectedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, keyBytes)
	_, _ = mac.Write([]byte(unsigned))
	return hmac.Equal(mac.Sum(nil), expectedSignature)
}

func (s *EventGridService) generateTopicKeyLocked(topicKey, keyName string) string {
	s.keyGeneration++
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", topicKey, keyName, s.keyGeneration)))
	return base64.StdEncoding.EncodeToString(sum[:])
}

type eventGridRoute struct {
	SubscriptionID    string
	ResourceGroup     string
	SubscriptionScope bool
	ResourceType      string
	Name              string
	ChildType         string
	ChildName         string
	ChildAction       string
}

func parseRoute(escapedPath string) (eventGridRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 5 || !strings.EqualFold(parts[0], "subscriptions") {
		return eventGridRoute{}, false
	}

	if strings.EqualFold(parts[2], "providers") && strings.EqualFold(parts[3], "Microsoft.EventGrid") {
		route := eventGridRoute{
			SubscriptionID:    parts[1],
			SubscriptionScope: true,
			ResourceType:      parts[4],
		}
		switch len(parts) {
		case 5:
			return route, true
		case 6:
			route.Name = parts[5]
			return route, true
		default:
			return eventGridRoute{}, false
		}
	}

	if len(parts) < 7 ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.EventGrid") {
		return eventGridRoute{}, false
	}
	route := eventGridRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
		ResourceType:   parts[6],
	}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.Name = parts[7]
		return route, true
	case 9:
		route.Name = parts[7]
		route.ChildType = parts[8]
		return route, true
	case 10:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		return route, true
	case 11:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.ChildAction = parts[10]
		return route, true
	default:
		return eventGridRoute{}, false
	}
}

func topicID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.EventGrid/topics/" + name
}

func eventSubscriptionID(subscriptionID, resourceGroup, topicName, subscriptionName string) string {
	return topicID(subscriptionID, resourceGroup, topicName) + "/eventSubscriptions/" + subscriptionName
}

func resourceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func childKey(subscriptionID, resourceGroup, parentName, childName string) string {
	return resourceKey(subscriptionID, resourceGroup, parentName) + "/" + strings.ToLower(childName)
}

func eventGridTopicListOptions(req *http.Request) (func(Topic) bool, int, int, error) {
	query := req.URL.Query()
	top := 20
	if rawTop := strings.TrimSpace(query.Get("$top")); rawTop != "" {
		parsed, err := strconv.Atoi(rawTop)
		if err != nil || parsed < 1 || parsed > 100 {
			return nil, 0, 0, fmt.Errorf("The $top query parameter must be an integer from 1 to 100.")
		}
		top = parsed
	}
	skip := 0
	if rawSkip := strings.TrimSpace(query.Get("$skipToken")); rawSkip != "" {
		parsed, err := strconv.Atoi(rawSkip)
		if err != nil || parsed < 0 {
			return nil, 0, 0, fmt.Errorf("The $skipToken query parameter is invalid.")
		}
		skip = parsed
	}

	matchesName, err := eventGridTopicNameFilter(query.Get("$filter"))
	if err != nil {
		return nil, 0, 0, err
	}
	return func(topic Topic) bool { return matchesName(topic.Name) }, top, skip, nil
}

func pageEventGridList[T any](values []T, top, skip int, req *http.Request) ([]T, string) {
	if skip > len(values) {
		return []T{}, ""
	}
	end := skip + top
	if end > len(values) {
		end = len(values)
	}
	page := values[skip:end]
	if end >= len(values) {
		return page, ""
	}
	return page, eventGridNextLink(req, end)
}

func eventGridNextLink(req *http.Request, skipToken int) string {
	next := *req.URL
	query := next.Query()
	query.Set("$skipToken", strconv.Itoa(skipToken))
	next.RawQuery = query.Encode()
	if next.Scheme == "" {
		next.Scheme = "https"
	}
	if next.Host == "" {
		next.Host = "management.azure.com"
	}
	return next.String()
}

func eventGridTopicNameFilter(filter string) (func(string) bool, error) {
	if strings.TrimSpace(filter) == "" {
		return func(string) bool { return true }, nil
	}
	parser, err := newEventGridTopicFilterParser(filter)
	if err != nil {
		return nil, err
	}
	matches, err := parser.parse()
	if err != nil {
		return nil, err
	}
	return matches, nil
}

type eventGridTopicFilterToken struct {
	kind  string
	value string
}

type eventGridTopicFilterParser struct {
	tokens []eventGridTopicFilterToken
	pos    int
}

func newEventGridTopicFilterParser(filter string) (*eventGridTopicFilterParser, error) {
	tokens := make([]eventGridTopicFilterToken, 0)
	for i := 0; i < len(filter); {
		switch ch := filter[i]; {
		case ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n':
			i++
		case ch == '(':
			tokens = append(tokens, eventGridTopicFilterToken{kind: "lparen", value: "("})
			i++
		case ch == ')':
			tokens = append(tokens, eventGridTopicFilterToken{kind: "rparen", value: ")"})
			i++
		case ch == ',':
			tokens = append(tokens, eventGridTopicFilterToken{kind: "comma", value: ","})
			i++
		case ch == '\'':
			value, next, err := readEventGridFilterString(filter, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, eventGridTopicFilterToken{kind: "string", value: value})
			i = next
		case isEventGridFilterIdentByte(ch):
			start := i
			for i < len(filter) && isEventGridFilterIdentByte(filter[i]) {
				i++
			}
			tokens = append(tokens, eventGridTopicFilterToken{kind: "ident", value: filter[start:i]})
		default:
			return nil, fmt.Errorf("The $filter query parameter is invalid.")
		}
	}
	return &eventGridTopicFilterParser{tokens: tokens}, nil
}

func readEventGridFilterString(filter string, start int) (string, int, error) {
	var builder strings.Builder
	for i := start + 1; i < len(filter); i++ {
		if filter[i] != '\'' {
			builder.WriteByte(filter[i])
			continue
		}
		if i+1 < len(filter) && filter[i+1] == '\'' {
			builder.WriteByte('\'')
			i++
			continue
		}
		return builder.String(), i + 1, nil
	}
	return "", 0, fmt.Errorf("The $filter query parameter is invalid.")
}

func isEventGridFilterIdentByte(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}

func (p *eventGridTopicFilterParser) parse() (func(string) bool, error) {
	matches, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.tokens) {
		return nil, fmt.Errorf("The $filter query parameter is invalid.")
	}
	return matches, nil
}

func (p *eventGridTopicFilterParser) parseOr() (func(string) bool, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.matchIdent("or") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		previous := left
		left = func(name string) bool { return previous(name) || right(name) }
	}
	return left, nil
}

func (p *eventGridTopicFilterParser) parseAnd() (func(string) bool, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.matchIdent("and") {
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		previous := left
		left = func(name string) bool { return previous(name) && right(name) }
	}
	return left, nil
}

func (p *eventGridTopicFilterParser) parseNot() (func(string) bool, error) {
	if p.matchIdent("not") {
		inner, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return func(name string) bool { return !inner(name) }, nil
	}
	return p.parsePrimary()
}

func (p *eventGridTopicFilterParser) parsePrimary() (func(string) bool, error) {
	if p.matchKind("lparen") {
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if !p.matchKind("rparen") {
			return nil, fmt.Errorf("The $filter query parameter is invalid.")
		}
		return inner, nil
	}
	if p.matchIdent("contains") {
		if !p.matchKind("lparen") {
			return nil, fmt.Errorf("The $filter query parameter is invalid.")
		}
		property, ok := p.consumeIdent()
		if !ok || !strings.EqualFold(property, "name") || !p.matchKind("comma") {
			return nil, fmt.Errorf("Filtering is permitted on the 'name' property only.")
		}
		pattern, ok := p.consumeString()
		if !ok || !p.matchKind("rparen") {
			return nil, fmt.Errorf("The $filter query parameter is invalid.")
		}
		pattern = strings.ToLower(pattern)
		return func(name string) bool { return strings.Contains(strings.ToLower(name), pattern) }, nil
	}

	property, ok := p.consumeIdent()
	if !ok {
		return nil, fmt.Errorf("The $filter query parameter is invalid.")
	}
	if !strings.EqualFold(property, "name") {
		return nil, fmt.Errorf("Filtering is permitted on the 'name' property only.")
	}
	switch {
	case p.matchIdent("eq"):
		expected, ok := p.consumeString()
		if !ok {
			return nil, fmt.Errorf("The $filter query parameter is invalid.")
		}
		return func(name string) bool { return strings.EqualFold(name, expected) }, nil
	case p.matchIdent("ne"):
		expected, ok := p.consumeString()
		if !ok {
			return nil, fmt.Errorf("The $filter query parameter is invalid.")
		}
		return func(name string) bool { return !strings.EqualFold(name, expected) }, nil
	default:
		return nil, fmt.Errorf("The $filter query parameter is invalid.")
	}
}

func (p *eventGridTopicFilterParser) matchIdent(value string) bool {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].kind != "ident" || !strings.EqualFold(p.tokens[p.pos].value, value) {
		return false
	}
	p.pos++
	return true
}

func (p *eventGridTopicFilterParser) matchKind(kind string) bool {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].kind != kind {
		return false
	}
	p.pos++
	return true
}

func (p *eventGridTopicFilterParser) consumeIdent() (string, bool) {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].kind != "ident" {
		return "", false
	}
	value := p.tokens[p.pos].value
	p.pos++
	return value, true
}

func (p *eventGridTopicFilterParser) consumeString() (string, bool) {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].kind != "string" {
		return "", false
	}
	value := p.tokens[p.pos].value
	p.pos++
	return value, true
}

func eventGridDeleteOperationHeaders(region, subscriptionID, resourceGroup, resourceID string) map[string]string {
	if strings.TrimSpace(region) == "" {
		region = "eastus"
	}
	sum := sha256.Sum256([]byte(resourceID))
	operationID := fmt.Sprintf("%x", sum[:16])
	base := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.EventGrid/locations/%s", subscriptionID, resourceGroup, region)
	return map[string]string{
		"Location":             fmt.Sprintf("%s/operationStatus/default/operationId/%s?api-version=%s", base, operationID, eventGridAPIVersion),
		"Azure-AsyncOperation": fmt.Sprintf("%s/operationResults/%s/Spring/default?api-version=%s", base, operationID, eventGridAPIVersion),
	}
}

func splitPath(escapedPath string) []string {
	trimmed := strings.Trim(escapedPath, "/")
	if trimmed == "" {
		return nil
	}
	rawParts := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part == "" {
			continue
		}
		decoded, err := url.PathUnescape(part)
		if err != nil {
			decoded = part
		}
		parts = append(parts, decoded)
	}
	return parts
}

func stringifyTags(tags map[string]any) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for key, value := range tags {
		out[key] = fmt.Sprint(value)
	}
	return out
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func applyEventSubscriptionDefaults(properties map[string]any, topicResourceID string) {
	if _, ok := properties["topic"]; !ok {
		properties["topic"] = topicResourceID
	}
	if _, ok := properties["eventDeliverySchema"]; !ok {
		properties["eventDeliverySchema"] = "EventGridSchema"
	}
	if _, ok := properties["retryPolicy"]; !ok {
		properties["retryPolicy"] = map[string]any{
			"maxDeliveryAttempts":      30,
			"eventTimeToLiveInMinutes": 1440,
		}
	}
	projectEventSubscriptionWebhookDestination(properties["destination"])
}

func projectEventSubscriptionWebhookDestination(destination any) {
	destinationMap, ok := destination.(map[string]any)
	if !ok || !strings.EqualFold(stringValue(destinationMap["endpointType"]), "WebHook") {
		return
	}
	destinationProperties, ok := destinationMap["properties"].(map[string]any)
	if !ok {
		return
	}
	if _, ok := destinationProperties["endpointBaseUrl"]; ok {
		return
	}
	if endpointURL := stringValue(destinationProperties["endpointUrl"]); endpointURL != "" {
		destinationProperties["endpointBaseUrl"] = endpointURL
	}
}

func eventGridDeliveryAttributeMappings(properties map[string]any) []any {
	if attrs := eventGridDeliveryAttributeMappingsFromDestination(properties["destination"]); attrs != nil {
		return attrs
	}
	if deliveryWithIdentity, ok := properties["deliveryWithResourceIdentity"].(map[string]any); ok {
		if attrs := eventGridDeliveryAttributeMappingsFromDestination(deliveryWithIdentity["destination"]); attrs != nil {
			return attrs
		}
	}
	return []any{}
}

func eventGridDeliveryAttributeMappingsFromDestination(destination any) []any {
	destinationMap, ok := destination.(map[string]any)
	if !ok {
		return nil
	}
	destinationProperties, ok := destinationMap["properties"].(map[string]any)
	if !ok {
		return nil
	}
	attrs, ok := destinationProperties["deliveryAttributeMappings"].([]any)
	if !ok {
		return nil
	}
	return append([]any(nil), attrs...)
}

func splitNestedName(name string) (string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func eventGridTopicEndpoint(name, location string) string {
	region := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(location), " ", ""))
	if region == "" {
		region = "eastus"
	}
	return "https://" + name + "." + region + "-1.eventgrid.azure.net/api/events"
}

func isEventGridPublishRequest(req *http.Request) bool {
	host := strings.ToLower(req.Host)
	if host == "" {
		host = strings.ToLower(req.Header.Get("Host"))
	}
	if colon := strings.IndexByte(host, ':'); colon >= 0 {
		host = host[:colon]
	}
	if !strings.HasSuffix(host, ".eventgrid.azure.net") && !strings.HasSuffix(host, ".eventgrid.azure.us") && !strings.HasSuffix(host, ".eventgrid.azure.cn") {
		return false
	}
	if req.URL.Query().Get("api-version") != eventGridPublishAPIVersion {
		return false
	}
	parts := splitPath(req.URL.EscapedPath())
	return req.Method == http.MethodPost && len(parts) == 2 && strings.EqualFold(parts[0], "api") && strings.EqualFold(parts[1], "events")
}

func eventGridRequestEndpoint(req *http.Request) string {
	scheme := req.URL.Scheme
	if scheme == "" {
		scheme = "https"
	}
	host := strings.ToLower(req.Host)
	if host == "" {
		host = strings.ToLower(req.Header.Get("Host"))
	}
	return strings.TrimRight(scheme+"://"+host+req.URL.EscapedPath(), "/")
}

func isValidEventGridEvent(event map[string]any, topicID string) bool {
	requiredStrings := []string{"id", "eventType", "subject"}
	for _, field := range requiredStrings {
		value, ok := event[field].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return false
		}
	}
	if dataVersion, ok := event["dataVersion"]; ok {
		if _, ok := dataVersion.(string); !ok {
			return false
		}
	}
	if metadataVersion, ok := event["metadataVersion"]; ok {
		value, ok := metadataVersion.(string)
		if !ok || value != "1" {
			return false
		}
	}
	if topic, ok := event["topic"]; ok {
		value, ok := topic.(string)
		if !ok || value != topicID {
			return false
		}
	}
	if !isEventGridDateTime(event["eventTime"]) {
		return false
	}
	_, hasData := event["data"].(map[string]any)
	return hasData
}

func isEventGridDateTime(value any) bool {
	raw, ok := value.(string)
	if !ok {
		return false
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05"} {
		if _, err := time.Parse(layout, raw); err == nil {
			return true
		}
	}
	return false
}

func isValidEventGridPublishEvent(event map[string]any, inputSchema, topicID string) bool {
	if strings.EqualFold(inputSchema, "CloudEventSchemaV1_0") {
		return isValidCloudEventGridEvent(event)
	}
	return isValidEventGridEvent(event, topicID)
}

func normalizeEventGridPublishEvent(event map[string]any, inputSchema, topicID string) {
	if strings.EqualFold(inputSchema, "CloudEventSchemaV1_0") {
		return
	}
	if _, ok := event["topic"]; !ok {
		event["topic"] = topicID
	}
	if _, ok := event["dataVersion"]; !ok {
		event["dataVersion"] = ""
	}
	if _, ok := event["metadataVersion"]; !ok {
		event["metadataVersion"] = "1"
	}
}

func isValidCloudEventGridEvent(event map[string]any) bool {
	specversion, ok := event["specversion"].(string)
	if !ok || strings.TrimSpace(specversion) != "1.0" {
		return false
	}
	requiredStrings := []string{"type", "source", "id"}
	for _, field := range requiredStrings {
		value, ok := event[field].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return false
		}
	}
	if !isEventGridDateTime(event["time"]) {
		return false
	}
	_, hasData := event["data"]
	return hasData
}

func cloneEventGridEvents(events []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(events))
	for _, event := range events {
		cloned := make(map[string]any, len(event))
		for key, value := range event {
			cloned[key] = value
		}
		out = append(out, cloned)
	}
	return out
}
