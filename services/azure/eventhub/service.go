package eventhub

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

var eventHubAPIVersions = []string{"2026-01-01", "2024-01-01"}

// EventHubService implements first-slice Azure Event Hubs control-plane APIs.
type EventHubService struct {
	mu             sync.RWMutex
	namespaces     map[string]Namespace
	eventHubs      map[string]EventHub
	consumerGroups map[string]ConsumerGroup
	authRules      map[string]AuthorizationRule
	accessKeys     map[string]AccessKeys
	runtimeEvents  map[string][]runtimeEvent
	keyGeneration  uint64
}

type runtimeEvent struct {
	Body           []byte
	UserProperties map[string]string
}

type runtimeOutboundEvent struct {
	Body           []byte
	UserProperties map[string]string
}

func New() *EventHubService {
	return &EventHubService{
		namespaces:     make(map[string]Namespace),
		eventHubs:      make(map[string]EventHub),
		consumerGroups: make(map[string]ConsumerGroup),
		authRules:      make(map[string]AuthorizationRule),
		accessKeys:     make(map[string]AccessKeys),
		runtimeEvents:  make(map[string][]runtimeEvent),
	}
}

func (s *EventHubService) Name() string { return "Microsoft.EventHub" }

func (s *EventHubService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateNamespace", Method: http.MethodPut, IAMAction: "azure:Microsoft.EventHub/namespaces/write"},
		{Name: "GetNamespace", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventHub/namespaces/read"},
		{Name: "ListNamespaces", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventHub/namespaces/read"},
		{Name: "DeleteNamespace", Method: http.MethodDelete, IAMAction: "azure:Microsoft.EventHub/namespaces/delete"},
		{Name: "CreateOrUpdateNamespaceAuthorizationRule", Method: http.MethodPut, IAMAction: "azure:Microsoft.EventHub/namespaces/authorizationRules/write"},
		{Name: "GetNamespaceAuthorizationRule", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventHub/namespaces/authorizationRules/read"},
		{Name: "ListNamespaceAuthorizationRules", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventHub/namespaces/authorizationRules/read"},
		{Name: "DeleteNamespaceAuthorizationRule", Method: http.MethodDelete, IAMAction: "azure:Microsoft.EventHub/namespaces/authorizationRules/delete"},
		{Name: "ListNamespaceKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.EventHub/namespaces/authorizationRules/listKeys/action"},
		{Name: "RegenerateNamespaceKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.EventHub/namespaces/authorizationRules/regenerateKeys/action"},
		{Name: "CreateOrUpdateEventHub", Method: http.MethodPut, IAMAction: "azure:Microsoft.EventHub/namespaces/eventhubs/write"},
		{Name: "GetEventHub", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventHub/namespaces/eventhubs/read"},
		{Name: "ListEventHubs", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventHub/namespaces/eventhubs/read"},
		{Name: "DeleteEventHub", Method: http.MethodDelete, IAMAction: "azure:Microsoft.EventHub/namespaces/eventhubs/delete"},
		{Name: "CreateOrUpdateConsumerGroup", Method: http.MethodPut, IAMAction: "azure:Microsoft.EventHub/namespaces/eventhubs/consumergroups/write"},
		{Name: "GetConsumerGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventHub/namespaces/eventhubs/consumergroups/read"},
		{Name: "ListConsumerGroups", Method: http.MethodGet, IAMAction: "azure:Microsoft.EventHub/namespaces/eventhubs/consumergroups/read"},
		{Name: "DeleteConsumerGroup", Method: http.MethodDelete, IAMAction: "azure:Microsoft.EventHub/namespaces/eventhubs/consumergroups/delete"},
		{Name: "SendEvent", Method: http.MethodPost, IAMAction: "azure:Microsoft.EventHub/runtime/send"},
		{Name: "SendBatchEvents", Method: http.MethodPost, IAMAction: "azure:Microsoft.EventHub/runtime/send"},
	}
}

func (s *EventHubService) HealthCheck() error { return nil }

func (s *EventHubService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(eventHubAPIVersions))
	for _, apiVersion := range eventHubAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.EventHub/namespaces",
			APIVersion: apiVersion,
		})
	}
	keys = append(keys, routing.ServiceKey{
		Provider:   routing.ProviderAzure,
		Service:    "Microsoft.EventHub/runtime",
		APIVersion: "2014-01",
	})
	return keys
}

func (s *EventHubService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces") ||
		strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/authorizationRules") ||
		strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/eventhubs") ||
		strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/eventhubs/authorizationRules") ||
		strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/eventhubs/consumergroups")
}

func (s *EventHubService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Event Hubs template resource type %q", stringValue(resource["type"]))
	}
	resourceType := stringValue(resource["type"])
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Event Hubs template resource is missing name")
	}

	body := map[string]any{
		"location":   resource["location"],
		"sku":        resource["sku"],
		"tags":       resource["tags"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}

	var resp *service.Response
	switch {
	case strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces"):
		resp, err = s.createOrUpdateNamespace(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/authorizationRules"):
		namespaceName, ruleName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Event Hubs namespace authorization rule template resource name must be {namespace}/{authorizationRule}")
		}
		resp, err = s.createOrUpdateNamespaceAuthorizationRule(subscriptionID, resourceGroup, namespaceName, ruleName, data)
	case strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/eventhubs"):
		namespaceName, eventHubName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Event Hubs template resource name must be {namespace}/{eventhub}")
		}
		resp, err = s.createOrUpdateEventHub(subscriptionID, resourceGroup, namespaceName, eventHubName, data)
	case strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/eventhubs/authorizationRules"):
		namespaceName, eventHubName, ruleName, ok := splitTripleNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Event Hubs event hub authorization rule template resource name must be {namespace}/{eventhub}/{authorizationRule}")
		}
		resp, err = s.createOrUpdateEventHubAuthorizationRule(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName, data)
	case strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/eventhubs/consumergroups"):
		namespaceName, eventHubName, consumerGroupName, ok := splitTripleNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Event Hubs consumer group template resource name must be {namespace}/{eventhub}/{consumerGroup}")
		}
		resp, err = s.createOrUpdateConsumerGroup(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName, data)
	default:
		err = fmt.Errorf("unsupported Event Hubs template resource type %q", resourceType)
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

func (s *EventHubService) TemplateListOperationResult(subscriptionID, resourceGroup string, resource map[string]any, operation string) (any, bool) {
	if !strings.EqualFold(operation, "listKeys") {
		return nil, false
	}

	resourceType := stringValue(resource["type"])
	name := stringValue(resource["name"])
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch {
	case strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/authorizationRules"):
		namespaceName, ruleName, ok := splitNestedName(name)
		if !ok {
			return nil, false
		}
		keys, ok := s.accessKeys[namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)]
		if !ok {
			return nil, false
		}
		return eventHubAccessKeysTemplateMap(keys), true
	case strings.EqualFold(resourceType, "Microsoft.EventHub/namespaces/eventhubs/authorizationRules"):
		namespaceName, eventHubName, ruleName, ok := splitTripleNestedName(name)
		if !ok {
			return nil, false
		}
		keys, ok := s.accessKeys[eventHubAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName)]
		if !ok {
			return nil, false
		}
		return eventHubAccessKeysTemplateMap(keys), true
	default:
		return nil, false
	}
}

func eventHubAccessKeysTemplateMap(keys AccessKeys) map[string]any {
	return map[string]any{
		"keyName":                   keys.KeyName,
		"primaryConnectionString":   keys.PrimaryConnectionString,
		"primaryKey":                keys.PrimaryKey,
		"secondaryConnectionString": keys.SecondaryConnectionString,
		"secondaryKey":              keys.SecondaryKey,
	}
}

func (s *EventHubService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	if namespaceName, parts, ok := dataPlaneNamespaceAndPath(ctx.RawRequest); ok {
		return s.handleRuntimeRequest(ctx, namespaceName, parts)
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Hubs route is not implemented.")
	}
	if !strings.EqualFold(route.ResourceType, "namespaces") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Hubs route is not implemented.")
	}
	if route.ChildType != "" {
		if strings.EqualFold(route.ChildType, "authorizationRules") {
			return s.handleAuthorizationRuleRequest(ctx, route)
		}
		if strings.EqualFold(route.ChildType, "eventhubs") {
			return s.handleEventHubRequest(ctx, route)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Hubs route is not implemented.")
	}
	return s.handleNamespaceRequest(ctx, route)
}

func (s *EventHubService) handleNamespaceRequest(ctx *service.RequestContext, route eventHubRoute) (*service.Response, error) {
	if route.NamespaceName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listNamespaces(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateNamespace(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, ctx.Body)
	case http.MethodGet:
		return s.getNamespace(route.SubscriptionID, route.ResourceGroup, route.NamespaceName)
	case http.MethodDelete:
		return s.deleteNamespace(route.SubscriptionID, route.ResourceGroup, route.NamespaceName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *EventHubService) handleAuthorizationRuleRequest(ctx *service.RequestContext, route eventHubRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listNamespaceAuthorizationRules(route.SubscriptionID, route.ResourceGroup, route.NamespaceName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.GrandChildType != "" {
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.GrandChildType, "listKeys") {
			return s.listNamespaceKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
		}
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.GrandChildType, "regenerateKeys") {
			return s.regenerateNamespaceKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, ctx.Body)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Hubs route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateNamespaceAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getNamespaceAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
	case http.MethodDelete:
		return s.deleteNamespaceAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *EventHubService) handleEventHubRequest(ctx *service.RequestContext, route eventHubRoute) (*service.Response, error) {
	if route.GrandChildType != "" {
		if strings.EqualFold(route.GrandChildType, "consumergroups") {
			return s.handleConsumerGroupRequest(ctx, route)
		}
		if strings.EqualFold(route.GrandChildType, "authorizationRules") {
			return s.handleEventHubAuthorizationRuleRequest(ctx, route)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Hubs route is not implemented.")
	}
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listEventHubs(route.SubscriptionID, route.ResourceGroup, route.NamespaceName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateEventHub(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getEventHub(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
	case http.MethodDelete:
		return s.deleteEventHub(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *EventHubService) handleEventHubAuthorizationRuleRequest(ctx *service.RequestContext, route eventHubRoute) (*service.Response, error) {
	if route.GrandChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listEventHubAuthorizationRules(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.Operation != "" {
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.Operation, "listKeys") {
			return s.listEventHubKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandChildName)
		}
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.Operation, "regenerateKeys") {
			return s.regenerateEventHubKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandChildName, ctx.Body)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Hubs route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateEventHubAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandChildName, ctx.Body)
	case http.MethodGet:
		return s.getEventHubAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandChildName)
	case http.MethodDelete:
		return s.deleteEventHubAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *EventHubService) handleConsumerGroupRequest(ctx *service.RequestContext, route eventHubRoute) (*service.Response, error) {
	if route.GrandChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listConsumerGroups(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateConsumerGroup(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandChildName, ctx.Body)
	case http.MethodGet:
		return s.getConsumerGroup(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandChildName)
	case http.MethodDelete:
		return s.deleteConsumerGroup(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *EventHubService) handleRuntimeRequest(ctx *service.RequestContext, namespaceName string, parts []string) (*service.Response, error) {
	if len(parts) < 2 || !strings.EqualFold(parts[len(parts)-1], "messages") || ctx.RawRequest.Method != http.MethodPost {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Event Hubs runtime route is not implemented.")
	}
	eventHubName := strings.Join(parts[:len(parts)-1], "/")
	return s.sendRuntimeEvent(namespaceName, eventHubName, ctx.RawRequest, ctx.Body)
}

func (s *EventHubService) sendRuntimeEvent(namespaceName, eventHubName string, req *http.Request, body []byte) (*service.Response, error) {
	outboundEvents, err := runtimeOutboundEventsFromRequest(req, body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "EventHubRuntimeError", "The Event Hubs runtime request content was invalid.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.runtimeEventHubExistsLocked(namespaceName, eventHubName) {
		return azurearm.ErrorResponse(http.StatusGone, "EventHubRuntimeError", fmt.Sprintf("Event Hub %q could not be found.", eventHubName))
	}
	key := runtimeEventHubKey(namespaceName, eventHubName)
	for _, outboundEvent := range outboundEvents {
		s.runtimeEvents[key] = append(s.runtimeEvents[key], runtimeEvent{
			Body:           append([]byte(nil), outboundEvent.Body...),
			UserProperties: cloneStringMap(outboundEvent.UserProperties),
		})
	}
	return &service.Response{
		StatusCode:     http.StatusCreated,
		RawContentType: "application/xml; charset=utf-8",
		Headers:        map[string]string{"Content-Type": "application/xml; charset=utf-8"},
	}, nil
}

func (s *EventHubService) createOrUpdateNamespace(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		SKU        map[string]any `json:"sku"`
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
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["status"]; !ok {
		input.Properties["status"] = "Active"
	}
	if _, ok := input.Properties["serviceBusEndpoint"]; !ok {
		input.Properties["serviceBusEndpoint"] = "https://" + name + ".servicebus.windows.net:443/"
	}

	namespace := Namespace{
		ID:         namespaceID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.EventHub/namespaces",
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := namespaceKey(subscriptionID, resourceGroup, name)
	_, existed := s.namespaces[key]
	s.namespaces[key] = namespace
	s.ensureRootAuthorizationRuleLocked(subscriptionID, resourceGroup, name)
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, namespace)
}

func (s *EventHubService) getNamespace(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	namespace, ok := s.namespaces[namespaceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs namespace %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, namespace)
}

func (s *EventHubService) listNamespaces(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]Namespace, 0)
	for key, namespace := range s.namespaces {
		if strings.HasPrefix(key, prefix) {
			values = append(values, namespace)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *EventHubService) deleteNamespace(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := namespaceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.namespaces[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs namespace %q could not be found.", name))
	}
	delete(s.namespaces, key)
	childPrefix := key + "/"
	for eventHubKey := range s.eventHubs {
		if strings.HasPrefix(eventHubKey, childPrefix) {
			delete(s.eventHubs, eventHubKey)
		}
	}
	for consumerGroupKey := range s.consumerGroups {
		if strings.HasPrefix(consumerGroupKey, childPrefix) {
			delete(s.consumerGroups, consumerGroupKey)
		}
	}
	for ruleKey := range s.authRules {
		if strings.HasPrefix(ruleKey, childPrefix) {
			delete(s.authRules, ruleKey)
			delete(s.accessKeys, ruleKey)
		}
	}
	runtimePrefix := strings.ToLower(name) + "/"
	for runtimeKey := range s.runtimeEvents {
		if strings.HasPrefix(runtimeKey, runtimePrefix) {
			delete(s.runtimeEvents, runtimeKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *EventHubService) createOrUpdateNamespaceAuthorizationRule(subscriptionID, resourceGroup, namespaceName, ruleName string, body []byte) (*service.Response, error) {
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
	if _, ok := input.Properties["rights"]; !ok {
		input.Properties["rights"] = []string{"Listen", "Send"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespaceKey(subscriptionID, resourceGroup, namespaceName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs namespace %q could not be found.", namespaceName))
	}

	rule := AuthorizationRule{
		ID:         namespaceAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, ruleName),
		Name:       ruleName,
		Type:       "Microsoft.EventHub/Namespaces/AuthorizationRules",
		Properties: input.Properties,
	}
	key := namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)
	s.authRules[key] = rule
	if _, ok := s.accessKeys[key]; !ok {
		s.accessKeys[key] = s.newNamespaceAccessKeysLocked(namespaceName, ruleName, key)
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *EventHubService) getNamespaceAuthorizationRule(subscriptionID, resourceGroup, namespaceName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	rule, ok := s.authRules[namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *EventHubService) listNamespaceAuthorizationRules(subscriptionID, resourceGroup, namespaceName string) (*service.Response, error) {
	nsKey := namespaceKey(subscriptionID, resourceGroup, namespaceName)

	s.mu.RLock()
	_, namespaceExists := s.namespaces[nsKey]
	values := make([]AuthorizationRule, 0)
	prefix := nsKey + "/authorizationrules/"
	for key, rule := range s.authRules {
		if strings.HasPrefix(key, prefix) {
			values = append(values, rule)
		}
	}
	s.mu.RUnlock()

	if !namespaceExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs namespace %q could not be found.", namespaceName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *EventHubService) deleteNamespaceAuthorizationRule(subscriptionID, resourceGroup, namespaceName, ruleName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)
	if _, ok := s.authRules[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.authRules, key)
	delete(s.accessKeys, key)
	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *EventHubService) listNamespaceKeys(subscriptionID, resourceGroup, namespaceName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	keys, ok := s.accessKeys[namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *EventHubService) regenerateNamespaceKeys(subscriptionID, resourceGroup, namespaceName, ruleName string, body []byte) (*service.Response, error) {
	var input struct {
		KeyType string `json:"keyType"`
		Key     string `json:"key"`
	}
	if err := gojson.Unmarshal(body, &input); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}

	keyType := strings.ToLower(strings.TrimSpace(input.KeyType))
	if keyType != "primarykey" && keyType != "secondarykey" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadRequest", "The keyType field must be PrimaryKey or SecondaryKey.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)
	keys, ok := s.accessKeys[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs authorization rule %q could not be found.", ruleName))
	}
	nextKey := strings.TrimSpace(input.Key)
	if nextKey == "" {
		nextKey = s.generateAccessKeyLocked(key, input.KeyType)
	}
	if keyType == "primarykey" {
		keys.PrimaryKey = nextKey
		keys.PrimaryConnectionString = eventHubConnectionString(namespaceName, ruleName, nextKey)
	} else {
		keys.SecondaryKey = nextKey
		keys.SecondaryConnectionString = eventHubConnectionString(namespaceName, ruleName, nextKey)
	}
	s.accessKeys[key] = keys
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *EventHubService) createOrUpdateEventHub(subscriptionID, resourceGroup, namespaceName, eventHubName string, body []byte) (*service.Response, error) {
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
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["status"]; !ok {
		input.Properties["status"] = "Active"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespaceKey(subscriptionID, resourceGroup, namespaceName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs namespace %q could not be found.", namespaceName))
	}

	eventHub := EventHub{
		ID:         eventHubID(subscriptionID, resourceGroup, namespaceName, eventHubName),
		Name:       namespaceName + "/" + eventHubName,
		Type:       "Microsoft.EventHub/namespaces/eventhubs",
		Properties: input.Properties,
	}
	key := eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName)
	_, existed := s.eventHubs[key]
	s.eventHubs[key] = eventHub
	s.ensureDefaultConsumerGroupLocked(subscriptionID, resourceGroup, namespaceName, eventHubName)

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, eventHub)
}

func (s *EventHubService) getEventHub(subscriptionID, resourceGroup, namespaceName, eventHubName string) (*service.Response, error) {
	s.mu.RLock()
	eventHub, ok := s.eventHubs[eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hub %q could not be found.", eventHubName))
	}
	return azurearm.JSONResponse(http.StatusOK, eventHub)
}

func (s *EventHubService) listEventHubs(subscriptionID, resourceGroup, namespaceName string) (*service.Response, error) {
	nsKey := namespaceKey(subscriptionID, resourceGroup, namespaceName)

	s.mu.RLock()
	_, namespaceExists := s.namespaces[nsKey]
	values := make([]EventHub, 0)
	prefix := nsKey + "/"
	for key, eventHub := range s.eventHubs {
		if strings.HasPrefix(key, prefix) {
			values = append(values, eventHub)
		}
	}
	s.mu.RUnlock()

	if !namespaceExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs namespace %q could not be found.", namespaceName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *EventHubService) deleteEventHub(subscriptionID, resourceGroup, namespaceName, eventHubName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName)
	if _, ok := s.eventHubs[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.eventHubs, key)
	delete(s.runtimeEvents, runtimeEventHubKey(namespaceName, eventHubName))
	childPrefix := key + "/"
	for consumerGroupKey := range s.consumerGroups {
		if strings.HasPrefix(consumerGroupKey, childPrefix) {
			delete(s.consumerGroups, consumerGroupKey)
		}
	}
	for ruleKey := range s.authRules {
		if strings.HasPrefix(ruleKey, childPrefix) {
			delete(s.authRules, ruleKey)
			delete(s.accessKeys, ruleKey)
		}
	}
	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *EventHubService) createOrUpdateEventHubAuthorizationRule(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName string, body []byte) (*service.Response, error) {
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
	if _, ok := input.Properties["rights"]; !ok {
		input.Properties["rights"] = []string{"Listen", "Send"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.eventHubs[eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hub %q could not be found.", eventHubName))
	}

	rule := AuthorizationRule{
		ID:         eventHubAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName),
		Name:       ruleName,
		Type:       "Microsoft.EventHub/Namespaces/EventHubs/AuthorizationRules",
		Properties: input.Properties,
	}
	key := eventHubAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName)
	s.authRules[key] = rule
	if _, ok := s.accessKeys[key]; !ok {
		s.accessKeys[key] = s.newEventHubAccessKeysLocked(namespaceName, eventHubName, ruleName, key)
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *EventHubService) getEventHubAuthorizationRule(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	rule, ok := s.authRules[eventHubAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hub authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *EventHubService) listEventHubAuthorizationRules(subscriptionID, resourceGroup, namespaceName, eventHubName string) (*service.Response, error) {
	hubKey := eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName)

	s.mu.RLock()
	_, eventHubExists := s.eventHubs[hubKey]
	values := make([]AuthorizationRule, 0)
	prefix := hubKey + "/"
	for key, rule := range s.authRules {
		if strings.HasPrefix(key, prefix) {
			values = append(values, rule)
		}
	}
	s.mu.RUnlock()

	if !eventHubExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hub %q could not be found.", eventHubName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *EventHubService) deleteEventHubAuthorizationRule(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := eventHubAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName)
	if _, ok := s.authRules[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.authRules, key)
	delete(s.accessKeys, key)
	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *EventHubService) listEventHubKeys(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	keys, ok := s.accessKeys[eventHubAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hub authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *EventHubService) regenerateEventHubKeys(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName string, body []byte) (*service.Response, error) {
	var input struct {
		KeyType string `json:"keyType"`
		Key     string `json:"key"`
	}
	if err := gojson.Unmarshal(body, &input); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}
	keyType := strings.ToLower(strings.TrimSpace(input.KeyType))
	if keyType != "primarykey" && keyType != "secondarykey" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadRequest", "The keyType field must be PrimaryKey or SecondaryKey.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := eventHubAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName)
	keys, ok := s.accessKeys[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hub authorization rule %q could not be found.", ruleName))
	}
	nextKey := strings.TrimSpace(input.Key)
	if nextKey == "" {
		nextKey = s.generateAccessKeyLocked(key, input.KeyType)
	}
	if keyType == "primarykey" {
		keys.PrimaryKey = nextKey
		keys.PrimaryConnectionString = eventHubEntityConnectionString(namespaceName, eventHubName, ruleName, nextKey)
	} else {
		keys.SecondaryKey = nextKey
		keys.SecondaryConnectionString = eventHubEntityConnectionString(namespaceName, eventHubName, ruleName, nextKey)
	}
	s.accessKeys[key] = keys
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *EventHubService) createOrUpdateConsumerGroup(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName string, body []byte) (*service.Response, error) {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.eventHubs[eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hub %q could not be found.", eventHubName))
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	key := consumerGroupKey(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName)
	if existing, ok := s.consumerGroups[key]; ok {
		if createdAt, ok := existing.Properties["createdAt"]; ok {
			input.Properties["createdAt"] = createdAt
		}
	}
	if _, ok := input.Properties["createdAt"]; !ok {
		input.Properties["createdAt"] = now
	}
	input.Properties["updatedAt"] = now

	consumerGroup := ConsumerGroup{
		ID:         consumerGroupID(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName),
		Name:       consumerGroupName,
		Type:       "Microsoft.EventHub/Namespaces/EventHubs/ConsumerGroups",
		Properties: input.Properties,
	}
	s.consumerGroups[key] = consumerGroup
	return azurearm.JSONResponse(http.StatusOK, consumerGroup)
}

func (s *EventHubService) getConsumerGroup(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName string) (*service.Response, error) {
	s.mu.RLock()
	consumerGroup, ok := s.consumerGroups[consumerGroupKey(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hubs consumer group %q could not be found.", consumerGroupName))
	}
	return azurearm.JSONResponse(http.StatusOK, consumerGroup)
}

func (s *EventHubService) listConsumerGroups(subscriptionID, resourceGroup, namespaceName, eventHubName string) (*service.Response, error) {
	eventHubKey := eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName)

	s.mu.RLock()
	_, eventHubExists := s.eventHubs[eventHubKey]
	values := make([]ConsumerGroup, 0)
	prefix := eventHubKey + "/"
	for key, consumerGroup := range s.consumerGroups {
		if strings.HasPrefix(key, prefix) {
			values = append(values, consumerGroup)
		}
	}
	s.mu.RUnlock()

	if !eventHubExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Event Hub %q could not be found.", eventHubName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *EventHubService) deleteConsumerGroup(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := consumerGroupKey(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName)
	if _, ok := s.consumerGroups[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.consumerGroups, key)
	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *EventHubService) ensureDefaultConsumerGroupLocked(subscriptionID, resourceGroup, namespaceName, eventHubName string) {
	key := consumerGroupKey(subscriptionID, resourceGroup, namespaceName, eventHubName, "$Default")
	if _, ok := s.consumerGroups[key]; ok {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	s.consumerGroups[key] = ConsumerGroup{
		ID:   consumerGroupID(subscriptionID, resourceGroup, namespaceName, eventHubName, "$Default"),
		Name: "$Default",
		Type: "Microsoft.EventHub/Namespaces/EventHubs/ConsumerGroups",
		Properties: map[string]any{
			"createdAt": now,
			"updatedAt": now,
		},
	}
}

func (s *EventHubService) ensureRootAuthorizationRuleLocked(subscriptionID, resourceGroup, namespaceName string) {
	const ruleName = "RootManageSharedAccessKey"
	key := namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)
	if _, ok := s.authRules[key]; ok {
		return
	}
	s.authRules[key] = AuthorizationRule{
		ID:   namespaceAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, ruleName),
		Name: ruleName,
		Type: "Microsoft.EventHub/Namespaces/AuthorizationRules",
		Properties: map[string]any{
			"rights": []string{"Listen", "Manage", "Send"},
		},
	}
	s.accessKeys[key] = s.newNamespaceAccessKeysLocked(namespaceName, ruleName, key)
}

func (s *EventHubService) newNamespaceAccessKeysLocked(namespaceName, ruleName, key string) AccessKeys {
	primary := s.generateAccessKeyLocked(key, "PrimaryKey")
	secondary := s.generateAccessKeyLocked(key, "SecondaryKey")
	return AccessKeys{
		KeyName:                   ruleName,
		PrimaryKey:                primary,
		SecondaryKey:              secondary,
		PrimaryConnectionString:   eventHubConnectionString(namespaceName, ruleName, primary),
		SecondaryConnectionString: eventHubConnectionString(namespaceName, ruleName, secondary),
	}
}

func (s *EventHubService) newEventHubAccessKeysLocked(namespaceName, eventHubName, ruleName, key string) AccessKeys {
	primary := s.generateAccessKeyLocked(key, "PrimaryKey")
	secondary := s.generateAccessKeyLocked(key, "SecondaryKey")
	return AccessKeys{
		KeyName:                   ruleName,
		PrimaryKey:                primary,
		SecondaryKey:              secondary,
		PrimaryConnectionString:   eventHubEntityConnectionString(namespaceName, eventHubName, ruleName, primary),
		SecondaryConnectionString: eventHubEntityConnectionString(namespaceName, eventHubName, ruleName, secondary),
	}
}

func (s *EventHubService) generateAccessKeyLocked(key, keyType string) string {
	s.keyGeneration++
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", key, keyType, s.keyGeneration)))
	return base64.StdEncoding.EncodeToString(sum[:])
}

type eventHubRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	NamespaceName  string
	ChildType      string
	ChildName      string
	GrandChildType string
	GrandChildName string
	Operation      string
}

func parseRoute(escapedPath string) (eventHubRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.EventHub") {
		return eventHubRoute{}, false
	}
	route := eventHubRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
		ResourceType:   parts[6],
	}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.NamespaceName = parts[7]
		return route, true
	case 9:
		route.NamespaceName = parts[7]
		route.ChildType = parts[8]
		return route, true
	case 10:
		route.NamespaceName = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		return route, true
	case 11:
		route.NamespaceName = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandChildType = parts[10]
		return route, true
	case 12:
		route.NamespaceName = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandChildType = parts[10]
		route.GrandChildName = parts[11]
		return route, true
	case 13:
		route.NamespaceName = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandChildType = parts[10]
		route.GrandChildName = parts[11]
		route.Operation = parts[12]
		return route, true
	default:
		return eventHubRoute{}, false
	}
}

func namespaceID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.EventHub/namespaces/" + name
}

func eventHubID(subscriptionID, resourceGroup, namespaceName, eventHubName string) string {
	return namespaceID(subscriptionID, resourceGroup, namespaceName) + "/eventhubs/" + eventHubName
}

func namespaceAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, ruleName string) string {
	return namespaceID(subscriptionID, resourceGroup, namespaceName) + "/authorizationRules/" + ruleName
}

func eventHubAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName string) string {
	return eventHubID(subscriptionID, resourceGroup, namespaceName, eventHubName) + "/authorizationRules/" + ruleName
}

func consumerGroupID(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName string) string {
	return eventHubID(subscriptionID, resourceGroup, namespaceName, eventHubName) + "/consumergroups/" + consumerGroupName
}

func namespaceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName string) string {
	return namespaceKey(subscriptionID, resourceGroup, namespaceName) + "/" + strings.ToLower(eventHubName)
}

func namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName string) string {
	return namespaceKey(subscriptionID, resourceGroup, namespaceName) + "/authorizationrules/" + strings.ToLower(ruleName)
}

func eventHubAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, eventHubName, ruleName string) string {
	return eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName) + "/authorizationrules/" + strings.ToLower(ruleName)
}

func consumerGroupKey(subscriptionID, resourceGroup, namespaceName, eventHubName, consumerGroupName string) string {
	return eventHubKey(subscriptionID, resourceGroup, namespaceName, eventHubName) + "/" + strings.ToLower(consumerGroupName)
}

func eventHubConnectionString(namespaceName, ruleName, key string) string {
	return "Endpoint=sb://" + namespaceName + ".servicebus.windows.net/;SharedAccessKeyName=" + ruleName + ";SharedAccessKey=" + key
}

func eventHubEntityConnectionString(namespaceName, eventHubName, ruleName, key string) string {
	return eventHubConnectionString(namespaceName, ruleName, key) + ";EntityPath=" + eventHubName
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

func dataPlaneNamespaceAndPath(req *http.Request) (string, []string, bool) {
	host := strings.ToLower(req.Host)
	if host == "" {
		host = strings.ToLower(req.Header.Get("Host"))
	}
	if colon := strings.IndexByte(host, ':'); colon >= 0 {
		host = host[:colon]
	}
	for _, suffix := range []string{
		".servicebus.windows.net",
		".servicebus.usgovcloudapi.net",
		".servicebus.chinacloudapi.cn",
		".servicebus.cloudapi.de",
	} {
		if strings.HasSuffix(host, suffix) {
			namespaceName := strings.TrimSuffix(host, suffix)
			return namespaceName, splitPath(req.URL.EscapedPath()), namespaceName != ""
		}
	}
	return "", nil, false
}

func (s *EventHubService) runtimeEventHubExistsLocked(namespaceName, eventHubName string) bool {
	runtimeName := namespaceName + "/" + eventHubName
	for _, eventHub := range s.eventHubs {
		if strings.EqualFold(eventHub.Name, runtimeName) {
			return true
		}
	}
	return false
}

func runtimeEventHubKey(namespaceName, eventHubName string) string {
	return strings.ToLower(namespaceName) + "/" + strings.ToLower(eventHubName)
}

func runtimeOutboundEventsFromRequest(req *http.Request, body []byte) ([]runtimeOutboundEvent, error) {
	if !isEventHubBatchContentType(req) {
		return []runtimeOutboundEvent{{
			Body:           append([]byte(nil), body...),
			UserProperties: runtimeEventUserProperties(req),
		}}, nil
	}

	var batch []struct {
		Body           any            `json:"Body"`
		UserProperties map[string]any `json:"UserProperties"`
	}
	if err := gojson.Unmarshal(body, &batch); err != nil {
		return nil, err
	}
	out := make([]runtimeOutboundEvent, 0, len(batch))
	for _, item := range batch {
		eventBody, err := eventHubBatchEventBody(item.Body)
		if err != nil {
			return nil, err
		}
		out = append(out, runtimeOutboundEvent{
			Body:           eventBody,
			UserProperties: stringifyAnyMap(item.UserProperties),
		})
	}
	return out, nil
}

func isEventHubBatchContentType(req *http.Request) bool {
	if req == nil {
		return false
	}
	contentType := strings.ToLower(req.Header.Get("Content-Type"))
	if semicolon := strings.IndexByte(contentType, ';'); semicolon >= 0 {
		contentType = contentType[:semicolon]
	}
	return strings.TrimSpace(contentType) == "application/vnd.microsoft.servicebus.json"
}

func eventHubBatchEventBody(value any) ([]byte, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case string:
		return []byte(typed), nil
	default:
		data, err := gojson.Marshal(typed)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
}

func runtimeEventUserProperties(req *http.Request) map[string]string {
	if req == nil {
		return nil
	}
	properties := map[string]string{}
	for key, values := range req.Header {
		if len(values) == 0 || isEventHubReservedRuntimeHeader(key) {
			continue
		}
		properties[key] = values[0]
	}
	if len(properties) == 0 {
		return nil
	}
	return properties
}

func isEventHubReservedRuntimeHeader(header string) bool {
	switch strings.ToLower(header) {
	case "accept", "authorization", "connection", "content-length", "content-type", "date", "expect", "host", "user-agent":
		return true
	default:
		return strings.HasPrefix(strings.ToLower(header), "x-ms-")
	}
}

func stringifyAnyMap(values map[string]any) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = fmt.Sprint(value)
	}
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
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

func splitNestedName(name string) (string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func splitTripleNestedName(name string) (string, string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}
