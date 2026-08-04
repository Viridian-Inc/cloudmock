package servicebus

import (
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

const serviceBusAPIVersion = "2024-01-01"
const serviceBusAtomContentType = "application/atom+xml;charset=utf-8"
const serviceBusAtomNamespace = "http://schemas.microsoft.com/netservices/2010/10/servicebus/connect"

// ServiceBusService implements first-slice Azure Service Bus control-plane APIs.
type ServiceBusService struct {
	mu                   sync.RWMutex
	namespaces           map[string]Namespace
	authRules            map[string]AuthorizationRule
	accessKeys           map[string]AccessKeys
	queues               map[string]Queue
	topics               map[string]Topic
	subscriptions        map[string]Subscription
	rules                map[string]Rule
	runtimeQueues        map[string][]runtimeMessage
	runtimeQueueDLQs     map[string][]runtimeMessage
	runtimeTopics        map[string]bool
	runtimeSubscriptions map[string][]runtimeMessage
	runtimeSubDLQs       map[string][]runtimeMessage
	runtimeSeq           uint64
	keyGeneration        uint64
}

type runtimeMessage struct {
	ID               string
	Sequence         uint64
	Body             []byte
	ContentType      string
	UserProperties   map[string]string
	BrokerProperties map[string]any
	EnqueuedAt       time.Time
	DeliveryCount    int
	DeadLetterReason string
	LockToken        string
	LockedUntil      time.Time
	Locked           bool
}

type runtimeMessageProperties struct {
	ContentType string
	User        map[string]string
	Broker      map[string]any
}

type runtimeOutboundMessage struct {
	Body       []byte
	Properties runtimeMessageProperties
}

type runtimeTarget struct {
	QueueName        string
	TopicName        string
	SubscriptionName string
	DeadLetter       bool
}

func (t runtimeTarget) Path() string {
	var path string
	if t.SubscriptionName != "" {
		path = t.TopicName + "/subscriptions/" + t.SubscriptionName
	} else {
		path = t.QueueName
	}
	if t.DeadLetter {
		path += "/$deadletterqueue"
	}
	return path
}

func New() *ServiceBusService {
	return &ServiceBusService{
		namespaces:           make(map[string]Namespace),
		authRules:            make(map[string]AuthorizationRule),
		accessKeys:           make(map[string]AccessKeys),
		queues:               make(map[string]Queue),
		topics:               make(map[string]Topic),
		subscriptions:        make(map[string]Subscription),
		rules:                make(map[string]Rule),
		runtimeQueues:        make(map[string][]runtimeMessage),
		runtimeQueueDLQs:     make(map[string][]runtimeMessage),
		runtimeTopics:        make(map[string]bool),
		runtimeSubscriptions: make(map[string][]runtimeMessage),
		runtimeSubDLQs:       make(map[string][]runtimeMessage),
	}
}

func (s *ServiceBusService) Name() string { return "Microsoft.ServiceBus" }

func (s *ServiceBusService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateNamespace", Method: http.MethodPut, IAMAction: "azure:Microsoft.ServiceBus/namespaces/write"},
		{Name: "GetNamespace", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/read"},
		{Name: "ListNamespaces", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/read"},
		{Name: "DeleteNamespace", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/namespaces/delete"},
		{Name: "CreateOrUpdateNamespaceAuthorizationRule", Method: http.MethodPut, IAMAction: "azure:Microsoft.ServiceBus/namespaces/AuthorizationRules/write"},
		{Name: "GetNamespaceAuthorizationRule", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/AuthorizationRules/read"},
		{Name: "ListNamespaceAuthorizationRules", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/AuthorizationRules/read"},
		{Name: "DeleteNamespaceAuthorizationRule", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/namespaces/AuthorizationRules/delete"},
		{Name: "ListNamespaceKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/namespaces/AuthorizationRules/listKeys/action"},
		{Name: "RegenerateNamespaceKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/namespaces/AuthorizationRules/regenerateKeys/action"},
		{Name: "CreateOrUpdateQueue", Method: http.MethodPut, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/write"},
		{Name: "GetQueue", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/read"},
		{Name: "ListQueues", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/read"},
		{Name: "DeleteQueue", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/delete"},
		{Name: "CreateOrUpdateQueueAuthorizationRule", Method: http.MethodPut, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/AuthorizationRules/write"},
		{Name: "GetQueueAuthorizationRule", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/AuthorizationRules/read"},
		{Name: "ListQueueAuthorizationRules", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/AuthorizationRules/read"},
		{Name: "DeleteQueueAuthorizationRule", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/AuthorizationRules/delete"},
		{Name: "ListQueueKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/AuthorizationRules/listKeys/action"},
		{Name: "RegenerateQueueKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/namespaces/queues/AuthorizationRules/regenerateKeys/action"},
		{Name: "CreateOrUpdateTopic", Method: http.MethodPut, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/write"},
		{Name: "GetTopic", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/read"},
		{Name: "ListTopics", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/read"},
		{Name: "DeleteTopic", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/delete"},
		{Name: "CreateOrUpdateTopicAuthorizationRule", Method: http.MethodPut, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/AuthorizationRules/write"},
		{Name: "GetTopicAuthorizationRule", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/AuthorizationRules/read"},
		{Name: "ListTopicAuthorizationRules", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/AuthorizationRules/read"},
		{Name: "DeleteTopicAuthorizationRule", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/AuthorizationRules/delete"},
		{Name: "ListTopicKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/AuthorizationRules/listKeys/action"},
		{Name: "RegenerateTopicKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/AuthorizationRules/regenerateKeys/action"},
		{Name: "CreateOrUpdateSubscription", Method: http.MethodPut, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/subscriptions/write"},
		{Name: "GetSubscription", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/subscriptions/read"},
		{Name: "ListSubscriptions", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/subscriptions/read"},
		{Name: "DeleteSubscription", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/subscriptions/delete"},
		{Name: "CreateOrUpdateRule", Method: http.MethodPut, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/subscriptions/rules/write"},
		{Name: "GetRule", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/subscriptions/rules/read"},
		{Name: "ListRules", Method: http.MethodGet, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/subscriptions/rules/read"},
		{Name: "DeleteRule", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/namespaces/topics/subscriptions/rules/delete"},
		{Name: "SendMessage", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/runtime/send"},
		{Name: "SendMessageBatch", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/runtime/send"},
		{Name: "PeekLockMessage", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/runtime/receive"},
		{Name: "ReceiveAndDeleteMessage", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/runtime/receive"},
		{Name: "CompleteMessage", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ServiceBus/runtime/complete"},
		{Name: "UnlockMessage", Method: http.MethodPut, IAMAction: "azure:Microsoft.ServiceBus/runtime/lock"},
		{Name: "RenewLockMessage", Method: http.MethodPost, IAMAction: "azure:Microsoft.ServiceBus/runtime/lock"},
	}
}

func (s *ServiceBusService) HealthCheck() error { return nil }

func (s *ServiceBusService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.ServiceBus/namespaces", APIVersion: serviceBusAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.ServiceBus/runtime", APIVersion: ""},
	}
}

func (s *ServiceBusService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces") ||
		strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/AuthorizationRules") ||
		strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/queues") ||
		strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/queues/AuthorizationRules") ||
		strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/topics") ||
		strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/topics/AuthorizationRules") ||
		strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/topics/subscriptions") ||
		strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/topics/subscriptions/rules")
}

func (s *ServiceBusService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Service Bus template resource type %q", stringValue(resource["type"]))
	}
	resourceType := stringValue(resource["type"])
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Service Bus template resource is missing name")
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
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces"):
		resp, err = s.createOrUpdateNamespace(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/AuthorizationRules"):
		namespaceName, ruleName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Service Bus namespace authorization rule template resource name must be {namespace}/{authorizationRule}")
		}
		resp, err = s.createOrUpdateNamespaceAuthorizationRule(subscriptionID, resourceGroup, namespaceName, ruleName, data)
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/queues"):
		namespaceName, queueName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Service Bus queue template resource name must be {namespace}/{queue}")
		}
		resp, err = s.createOrUpdateQueue(subscriptionID, resourceGroup, namespaceName, queueName, data)
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/queues/AuthorizationRules"):
		namespaceName, queueName, ruleName, ok := splitTripleNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Service Bus queue authorization rule template resource name must be {namespace}/{queue}/{authorizationRule}")
		}
		resp, err = s.createOrUpdateQueueAuthorizationRule(subscriptionID, resourceGroup, namespaceName, queueName, ruleName, data)
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/topics"):
		namespaceName, topicName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Service Bus topic template resource name must be {namespace}/{topic}")
		}
		resp, err = s.createOrUpdateTopic(subscriptionID, resourceGroup, namespaceName, topicName, data)
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/topics/AuthorizationRules"):
		namespaceName, topicName, ruleName, ok := splitTripleNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Service Bus topic authorization rule template resource name must be {namespace}/{topic}/{authorizationRule}")
		}
		resp, err = s.createOrUpdateTopicAuthorizationRule(subscriptionID, resourceGroup, namespaceName, topicName, ruleName, data)
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/topics/subscriptions"):
		namespaceName, topicName, subscriptionName, ok := splitTripleNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Service Bus subscription template resource name must be {namespace}/{topic}/{subscription}")
		}
		resp, err = s.createOrUpdateSubscription(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, data)
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/topics/subscriptions/rules"):
		namespaceName, topicName, subscriptionName, ruleName, ok := splitQuadNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Service Bus rule template resource name must be {namespace}/{topic}/{subscription}/{rule}")
		}
		resp, err = s.createOrUpdateRule(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName, data)
	default:
		err = fmt.Errorf("unsupported Service Bus template resource type %q", resourceType)
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

func (s *ServiceBusService) TemplateListOperationResult(subscriptionID, resourceGroup string, resource map[string]any, operation string) (any, bool) {
	if !strings.EqualFold(operation, "listKeys") {
		return nil, false
	}

	resourceType := stringValue(resource["type"])
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch {
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/AuthorizationRules"):
		namespaceName, ruleName, ok := splitNestedName(stringValue(resource["name"]))
		if !ok {
			return nil, false
		}
		keys, ok := s.accessKeys[namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)]
		if !ok {
			return nil, false
		}
		return serviceBusAccessKeysTemplateMap(keys), true
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/queues/AuthorizationRules"):
		namespaceName, queueName, ruleName, ok := splitTripleNestedName(stringValue(resource["name"]))
		if !ok {
			return nil, false
		}
		keys, ok := s.accessKeys[queueAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, queueName, ruleName)]
		if !ok {
			return nil, false
		}
		return serviceBusAccessKeysTemplateMap(keys), true
	case strings.EqualFold(resourceType, "Microsoft.ServiceBus/namespaces/topics/AuthorizationRules"):
		namespaceName, topicName, ruleName, ok := splitTripleNestedName(stringValue(resource["name"]))
		if !ok {
			return nil, false
		}
		keys, ok := s.accessKeys[topicAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, topicName, ruleName)]
		if !ok {
			return nil, false
		}
		return serviceBusAccessKeysTemplateMap(keys), true
	default:
		return nil, false
	}
}

func serviceBusAccessKeysTemplateMap(keys AccessKeys) map[string]any {
	return map[string]any{
		"keyName":                   keys.KeyName,
		"primaryConnectionString":   keys.PrimaryConnectionString,
		"primaryKey":                keys.PrimaryKey,
		"secondaryConnectionString": keys.SecondaryConnectionString,
		"secondaryKey":              keys.SecondaryKey,
	}
}

func (s *ServiceBusService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	if accountName, parts, ok := flociServiceBusAccountAndPath(ctx.RawRequest); ok {
		return s.handleFlociAdminRequest(ctx, accountName, parts)
	}

	if namespaceName, parts, ok := dataPlaneNamespaceAndPath(ctx.RawRequest); ok {
		return s.handleRuntimeRequest(ctx, namespaceName, parts)
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Service Bus route is not implemented.")
	}
	if !strings.EqualFold(route.ResourceType, "namespaces") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Service Bus route is not implemented.")
	}
	if route.ChildType != "" {
		switch {
		case strings.EqualFold(route.ChildType, "AuthorizationRules"):
			return s.handleAuthorizationRuleRequest(ctx, route)
		case strings.EqualFold(route.ChildType, "queues") && strings.EqualFold(route.GrandchildType, "AuthorizationRules"):
			return s.handleQueueAuthorizationRuleRequest(ctx, route)
		case strings.EqualFold(route.ChildType, "queues") && route.GrandchildType == "":
			return s.handleQueueRequest(ctx, route)
		case strings.EqualFold(route.ChildType, "topics") && strings.EqualFold(route.GrandchildType, "AuthorizationRules"):
			return s.handleTopicAuthorizationRuleRequest(ctx, route)
		case strings.EqualFold(route.ChildType, "topics") && route.GrandchildType == "":
			return s.handleTopicRequest(ctx, route)
		case strings.EqualFold(route.ChildType, "topics") && strings.EqualFold(route.GrandchildType, "subscriptions") && route.RuleType != "":
			return s.handleRuleRequest(ctx, route)
		case strings.EqualFold(route.ChildType, "topics") && strings.EqualFold(route.GrandchildType, "subscriptions"):
			return s.handleSubscriptionRequest(ctx, route)
		default:
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Service Bus route is not implemented.")
		}
	}
	return s.handleNamespaceRequest(ctx, route)
}

func (s *ServiceBusService) handleNamespaceRequest(ctx *service.RequestContext, route serviceBusRoute) (*service.Response, error) {
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

func (s *ServiceBusService) handleAuthorizationRuleRequest(ctx *service.RequestContext, route serviceBusRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listNamespaceAuthorizationRules(route.SubscriptionID, route.ResourceGroup, route.NamespaceName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	if route.GrandchildType != "" {
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.GrandchildType, "listKeys") {
			return s.listNamespaceKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
		}
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.GrandchildType, "regenerateKeys") {
			return s.regenerateNamespaceKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, ctx.Body)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Service Bus route is not implemented.")
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

func (s *ServiceBusService) handleQueueRequest(ctx *service.RequestContext, route serviceBusRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listQueues(route.SubscriptionID, route.ResourceGroup, route.NamespaceName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateQueue(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getQueue(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
	case http.MethodDelete:
		return s.deleteQueue(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ServiceBusService) handleQueueAuthorizationRuleRequest(ctx *service.RequestContext, route serviceBusRoute) (*service.Response, error) {
	if route.GrandchildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listQueueAuthorizationRules(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.RuleType != "" {
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.RuleType, "listKeys") {
			return s.listQueueKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName)
		}
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.RuleType, "regenerateKeys") {
			return s.regenerateQueueKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName, ctx.Body)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Service Bus route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateQueueAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName, ctx.Body)
	case http.MethodGet:
		return s.getQueueAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName)
	case http.MethodDelete:
		return s.deleteQueueAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ServiceBusService) handleTopicRequest(ctx *service.RequestContext, route serviceBusRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listTopics(route.SubscriptionID, route.ResourceGroup, route.NamespaceName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateTopic(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getTopic(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
	case http.MethodDelete:
		return s.deleteTopic(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ServiceBusService) handleTopicAuthorizationRuleRequest(ctx *service.RequestContext, route serviceBusRoute) (*service.Response, error) {
	if route.GrandchildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listTopicAuthorizationRules(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.RuleType != "" {
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.RuleType, "listKeys") {
			return s.listTopicKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName)
		}
		if ctx.RawRequest.Method == http.MethodPost && strings.EqualFold(route.RuleType, "regenerateKeys") {
			return s.regenerateTopicKeys(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName, ctx.Body)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Service Bus route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateTopicAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName, ctx.Body)
	case http.MethodGet:
		return s.getTopicAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName)
	case http.MethodDelete:
		return s.deleteTopicAuthorizationRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ServiceBusService) handleSubscriptionRequest(ctx *service.RequestContext, route serviceBusRoute) (*service.Response, error) {
	if route.GrandchildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listSubscriptions(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateSubscription(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName, ctx.Body)
	case http.MethodGet:
		return s.getSubscription(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName)
	case http.MethodDelete:
		return s.deleteSubscription(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ServiceBusService) handleRuleRequest(ctx *service.RequestContext, route serviceBusRoute) (*service.Response, error) {
	if !strings.EqualFold(route.RuleType, "rules") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Service Bus route is not implemented.")
	}
	if route.RuleName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listRules(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName, route.RuleName, ctx.Body)
	case http.MethodGet:
		return s.getRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName, route.RuleName)
	case http.MethodDelete:
		return s.deleteRule(route.SubscriptionID, route.ResourceGroup, route.NamespaceName, route.ChildName, route.GrandchildName, route.RuleName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ServiceBusService) handleRuntimeRequest(ctx *service.RequestContext, namespaceName string, parts []string) (*service.Response, error) {
	if len(parts) < 2 {
		return serviceBusRuntimeError(http.StatusNotFound, "The Service Bus runtime route is not implemented.")
	}

	if strings.EqualFold(parts[len(parts)-1], "messages") && ctx.RawRequest.Method == http.MethodPost {
		entityPath := strings.Join(parts[:len(parts)-1], "/")
		return s.sendRuntimeMessage(namespaceName, entityPath, ctx.RawRequest, ctx.Body)
	}
	if len(parts) >= 3 && strings.EqualFold(parts[len(parts)-2], "messages") && strings.EqualFold(parts[len(parts)-1], "head") {
		target := parseRuntimeTarget(parts[:len(parts)-2])
		switch ctx.RawRequest.Method {
		case http.MethodPost:
			return s.peekLockRuntimeMessage(namespaceName, target)
		case http.MethodDelete:
			return s.receiveAndDeleteRuntimeMessage(namespaceName, target)
		default:
			return serviceBusRuntimeError(http.StatusMethodNotAllowed, "The method is not allowed for this route.")
		}
	}
	if len(parts) >= 4 && strings.EqualFold(parts[len(parts)-3], "messages") && ctx.RawRequest.Method == http.MethodDelete {
		target := parseRuntimeTarget(parts[:len(parts)-3])
		return s.completeRuntimeMessage(namespaceName, target, parts[len(parts)-2], parts[len(parts)-1])
	}
	if len(parts) >= 4 && strings.EqualFold(parts[len(parts)-3], "messages") && ctx.RawRequest.Method == http.MethodPut {
		target := parseRuntimeTarget(parts[:len(parts)-3])
		return s.unlockRuntimeMessage(namespaceName, target, parts[len(parts)-2], parts[len(parts)-1])
	}
	if len(parts) >= 4 && strings.EqualFold(parts[len(parts)-3], "messages") && ctx.RawRequest.Method == http.MethodPost {
		target := parseRuntimeTarget(parts[:len(parts)-3])
		return s.renewRuntimeMessageLock(namespaceName, target, parts[len(parts)-2], parts[len(parts)-1])
	}
	return serviceBusRuntimeError(http.StatusNotFound, "The Service Bus runtime route is not implemented.")
}

type serviceBusNamespaceRef struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
}

func (s *ServiceBusService) handleFlociAdminRequest(ctx *service.RequestContext, accountName string, parts []string) (*service.Response, error) {
	namespaceRef, ok := s.activeNamespaceRef()
	if !ok {
		namespaceRef = serviceBusNamespaceRef{
			SubscriptionID: serviceBusLocalSubscriptionID(ctx),
			ResourceGroup:  "local",
			Name:           accountName,
		}
	}

	if len(parts) == 1 && strings.EqualFold(parts[0], "$namespaceinfo") && ctx.RawRequest.Method == http.MethodGet {
		return serviceBusAtomResponse(http.StatusOK, serviceBusNamespaceInfoAtom(namespaceRef.Name))
	}
	if len(parts) == 2 && strings.EqualFold(parts[0], "$Resources") && ctx.RawRequest.Method == http.MethodGet {
		switch strings.ToLower(parts[1]) {
		case "queues":
			return serviceBusAtomResponse(http.StatusOK, s.flociQueueFeed(namespaceRef))
		case "topics":
			return serviceBusAtomResponse(http.StatusOK, s.flociTopicFeed(namespaceRef))
		default:
			return serviceBusAtomNotFound("Unknown Service Bus entity collection: " + parts[1])
		}
	}
	if len(parts) == 1 {
		return s.handleFlociEntityRequest(ctx, namespaceRef, parts[0])
	}
	if len(parts) >= 2 && strings.EqualFold(parts[1], "subscriptions") {
		return s.handleFlociSubscriptionRequest(ctx, namespaceRef, parts)
	}
	return serviceBusAtomNotFound("The Service Bus local administration route is not implemented.")
}

func (s *ServiceBusService) handleFlociEntityRequest(ctx *service.RequestContext, namespaceRef serviceBusNamespaceRef, entityName string) (*service.Response, error) {
	switch ctx.RawRequest.Method {
	case http.MethodGet:
		return s.getFlociEntity(namespaceRef, entityName)
	case http.MethodPut, http.MethodPost:
		return s.createOrUpdateFlociEntity(namespaceRef, entityName, ctx.Body)
	case http.MethodDelete:
		return s.deleteFlociEntity(namespaceRef, entityName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ServiceBusService) getFlociEntity(namespaceRef serviceBusNamespaceRef, entityName string) (*service.Response, error) {
	s.mu.RLock()
	queue, queueOK := s.queues[queueKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName)]
	topic, topicOK := s.topics[topicKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName)]
	s.mu.RUnlock()
	if queueOK {
		return serviceBusAtomResponse(http.StatusOK, serviceBusQueueAtomEntry(namespaceRef.Name, queue))
	}
	if topicOK {
		return serviceBusAtomResponse(http.StatusOK, serviceBusTopicAtomEntry(namespaceRef.Name, topic))
	}
	return serviceBusAtomNotFound(entityName + " not found")
}

func (s *ServiceBusService) createOrUpdateFlociEntity(namespaceRef serviceBusNamespaceRef, entityName string, body []byte) (*service.Response, error) {
	if strings.Contains(string(body), "TopicDescription") {
		data, err := gojson.Marshal(map[string]any{"properties": serviceBusTopicPropertiesFromAtom(body)})
		if err != nil {
			return nil, err
		}
		resp, err := s.createOrUpdateTopic(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName, data)
		if err != nil {
			return resp, err
		}
		s.mu.RLock()
		topic := s.topics[topicKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName)]
		s.mu.RUnlock()
		return serviceBusAtomResponse(resp.StatusCode, serviceBusTopicAtomEntry(namespaceRef.Name, topic))
	}

	data, err := gojson.Marshal(map[string]any{"properties": serviceBusQueuePropertiesFromAtom(body)})
	if err != nil {
		return nil, err
	}
	resp, err := s.createOrUpdateQueue(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName, data)
	if err != nil {
		return resp, err
	}
	s.mu.RLock()
	queue := s.queues[queueKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName)]
	s.mu.RUnlock()
	return serviceBusAtomResponse(resp.StatusCode, serviceBusQueueAtomEntry(namespaceRef.Name, queue))
}

func (s *ServiceBusService) deleteFlociEntity(namespaceRef serviceBusNamespaceRef, entityName string) (*service.Response, error) {
	s.mu.RLock()
	_, queueOK := s.queues[queueKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName)]
	_, topicOK := s.topics[topicKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName)]
	s.mu.RUnlock()
	if queueOK {
		if _, err := s.deleteQueue(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName); err != nil {
			return nil, err
		}
		return &service.Response{StatusCode: http.StatusOK}, nil
	}
	if topicOK {
		if _, err := s.deleteTopic(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, entityName); err != nil {
			return nil, err
		}
		return &service.Response{StatusCode: http.StatusOK}, nil
	}
	return serviceBusAtomNotFound(entityName + " not found")
}

func (s *ServiceBusService) handleFlociSubscriptionRequest(ctx *service.RequestContext, namespaceRef serviceBusNamespaceRef, parts []string) (*service.Response, error) {
	if len(parts) >= 4 && strings.EqualFold(parts[3], "rules") {
		return s.handleFlociRuleRequest(ctx, namespaceRef, parts)
	}
	if len(parts) == 2 {
		if ctx.RawRequest.Method == http.MethodGet {
			return serviceBusAtomResponse(http.StatusOK, s.flociSubscriptionFeed(namespaceRef, parts[0]))
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if len(parts) == 3 {
		switch ctx.RawRequest.Method {
		case http.MethodGet:
			return s.getFlociSubscription(namespaceRef, parts[0], parts[2])
		case http.MethodPut, http.MethodPost:
			return s.createOrUpdateFlociSubscription(namespaceRef, parts[0], parts[2], ctx.Body)
		case http.MethodDelete:
			return s.deleteFlociSubscription(namespaceRef, parts[0], parts[2])
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	return serviceBusAtomNotFound("The Service Bus local subscription route is not implemented.")
}

func (s *ServiceBusService) handleFlociRuleRequest(ctx *service.RequestContext, namespaceRef serviceBusNamespaceRef, parts []string) (*service.Response, error) {
	if len(parts) == 4 {
		if ctx.RawRequest.Method == http.MethodGet {
			return serviceBusAtomResponse(http.StatusOK, s.flociRuleFeed(namespaceRef, parts[0], parts[2]))
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if len(parts) == 5 {
		switch ctx.RawRequest.Method {
		case http.MethodGet:
			return s.getFlociRule(namespaceRef, parts[0], parts[2], parts[4])
		case http.MethodPut, http.MethodPost:
			return s.createOrUpdateFlociRule(namespaceRef, parts[0], parts[2], parts[4], ctx.Body)
		case http.MethodDelete:
			return s.deleteFlociRule(namespaceRef, parts[0], parts[2], parts[4])
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	return serviceBusAtomNotFound("The Service Bus local rule route is not implemented.")
}

func (s *ServiceBusService) getFlociSubscription(namespaceRef serviceBusNamespaceRef, topicName, subscriptionName string) (*service.Response, error) {
	s.mu.RLock()
	subscription, ok := s.subscriptions[subscriptionKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName)]
	s.mu.RUnlock()
	if !ok {
		return serviceBusAtomNotFound("Subscription not found: " + subscriptionName)
	}
	return serviceBusAtomResponse(http.StatusOK, serviceBusSubscriptionAtomEntry(namespaceRef.Name, topicName, subscription))
}

func (s *ServiceBusService) createOrUpdateFlociSubscription(namespaceRef serviceBusNamespaceRef, topicName, subscriptionName string, body []byte) (*service.Response, error) {
	data, err := gojson.Marshal(map[string]any{"properties": serviceBusSubscriptionPropertiesFromAtom(body)})
	if err != nil {
		return nil, err
	}
	resp, err := s.createOrUpdateSubscription(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName, data)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return resp, nil
	}
	s.mu.RLock()
	subscription := s.subscriptions[subscriptionKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName)]
	s.mu.RUnlock()
	return serviceBusAtomResponse(resp.StatusCode, serviceBusSubscriptionAtomEntry(namespaceRef.Name, topicName, subscription))
}

func (s *ServiceBusService) deleteFlociSubscription(namespaceRef serviceBusNamespaceRef, topicName, subscriptionName string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.subscriptions[subscriptionKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName)]
	s.mu.RUnlock()
	if !ok {
		return serviceBusAtomNotFound("Subscription not found: " + subscriptionName)
	}
	if _, err := s.deleteSubscription(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName); err != nil {
		return nil, err
	}
	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *ServiceBusService) getFlociRule(namespaceRef serviceBusNamespaceRef, topicName, subscriptionName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	rule, ok := s.rules[ruleKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return serviceBusAtomNotFound("Rule not found: " + ruleName)
	}
	return serviceBusAtomResponse(http.StatusOK, serviceBusRuleAtomEntry(namespaceRef.Name, topicName, subscriptionName, rule))
}

func (s *ServiceBusService) createOrUpdateFlociRule(namespaceRef serviceBusNamespaceRef, topicName, subscriptionName, ruleName string, body []byte) (*service.Response, error) {
	properties := serviceBusRulePropertiesFromAtom(body)
	ruleStoreKey := ruleKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName, ruleName)
	s.mu.RLock()
	_, existed := s.rules[ruleStoreKey]
	s.mu.RUnlock()

	data, err := gojson.Marshal(map[string]any{"properties": properties})
	if err != nil {
		return nil, err
	}
	resp, err := s.createOrUpdateRule(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName, ruleName, data)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return resp, nil
	}
	s.mu.RLock()
	rule := s.rules[ruleStoreKey]
	s.mu.RUnlock()
	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return serviceBusAtomResponse(status, serviceBusRuleAtomEntry(namespaceRef.Name, topicName, subscriptionName, rule))
}

func (s *ServiceBusService) deleteFlociRule(namespaceRef serviceBusNamespaceRef, topicName, subscriptionName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.rules[ruleKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return serviceBusAtomNotFound("Rule not found: " + ruleName)
	}
	if _, err := s.deleteRule(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName, ruleName); err != nil {
		return nil, err
	}
	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *ServiceBusService) sendRuntimeMessage(namespaceName, entityPath string, req *http.Request, body []byte) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	outboundMessages, err := runtimeOutboundMessagesFromRequest(req, body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}
	if len(outboundMessages) == 0 && !s.runtimeEntityExistsLocked(namespaceName, entityPath) {
		return serviceBusRuntimeError(http.StatusGone, fmt.Sprintf("Service Bus entity %q could not be found.", entityPath))
	}

	for _, outbound := range outboundMessages {
		if !s.enqueueRuntimeMessageLocked(namespaceName, entityPath, outbound.Body, outbound.Properties) {
			return serviceBusRuntimeError(http.StatusGone, fmt.Sprintf("Service Bus entity %q could not be found.", entityPath))
		}
	}
	return serviceBusSendResponse(req), nil
}

func (s *ServiceBusService) enqueueRuntimeMessageLocked(namespaceName, entityPath string, body []byte, messageProperties runtimeMessageProperties) bool {
	queueKey := runtimeQueueKey(namespaceName, entityPath)
	if _, ok := s.runtimeQueues[queueKey]; ok {
		msg := s.newRuntimeMessageLocked(body, messageProperties)
		s.runtimeQueues[queueKey] = append(s.runtimeQueues[queueKey], msg)
		return true
	}
	topicKey := runtimeTopicKey(namespaceName, entityPath)
	if _, ok := s.runtimeTopics[topicKey]; !ok {
		return false
	}
	prefix := topicKey + "/"
	for subKey := range s.runtimeSubscriptions {
		if strings.HasPrefix(subKey, prefix) {
			subscriptionName := strings.TrimPrefix(subKey, prefix)
			if !s.runtimeSubscriptionAcceptsMessageLocked(namespaceName, entityPath, subscriptionName, messageProperties) {
				continue
			}
			s.runtimeSubscriptions[subKey] = append(s.runtimeSubscriptions[subKey], s.newRuntimeMessageLocked(body, messageProperties))
		}
	}
	return true
}

func (s *ServiceBusService) runtimeEntityExistsLocked(namespaceName, entityPath string) bool {
	if _, ok := s.runtimeQueues[runtimeQueueKey(namespaceName, entityPath)]; ok {
		return true
	}
	_, ok := s.runtimeTopics[runtimeTopicKey(namespaceName, entityPath)]
	return ok
}

func (s *ServiceBusService) peekLockRuntimeMessage(namespaceName string, target runtimeTarget) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages, ok := s.runtimeMessagesForTargetLocked(namespaceName, target)
	if !ok {
		return serviceBusRuntimeError(http.StatusGone, fmt.Sprintf("Service Bus entity %q could not be found.", target.Path()))
	}
	now := time.Now().UTC()
	messages = s.expireRuntimeLocksForTargetLocked(namespaceName, target, messages, now)
	messages = s.expireRuntimeMessagesForTargetLocked(namespaceName, target, messages, now)
	for i := range messages {
		if messages[i].Locked {
			continue
		}
		messages[i].Locked = true
		messages[i].LockToken = fmt.Sprintf("lock-%016x", messages[i].Sequence)
		messages[i].LockedUntil = now.Add(s.lockDurationForRuntimeTargetLocked(namespaceName, target))
		s.setRuntimeMessagesForTargetLocked(namespaceName, target, messages)
		return serviceBusMessageResponse(http.StatusCreated, namespaceName, target.Path(), messages[i], true)
	}
	return &service.Response{StatusCode: http.StatusNoContent}, nil
}

func (s *ServiceBusService) receiveAndDeleteRuntimeMessage(namespaceName string, target runtimeTarget) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages, ok := s.runtimeMessagesForTargetLocked(namespaceName, target)
	if !ok {
		return serviceBusRuntimeError(http.StatusGone, fmt.Sprintf("Service Bus entity %q could not be found.", target.Path()))
	}
	now := time.Now().UTC()
	messages = s.expireRuntimeLocksForTargetLocked(namespaceName, target, messages, now)
	messages = s.expireRuntimeMessagesForTargetLocked(namespaceName, target, messages, now)
	for i, msg := range messages {
		if msg.Locked {
			continue
		}
		s.setRuntimeMessagesForTargetLocked(namespaceName, target, append(messages[:i], messages[i+1:]...))
		return serviceBusMessageResponse(http.StatusOK, namespaceName, target.Path(), msg, false)
	}
	return &service.Response{StatusCode: http.StatusNoContent}, nil
}

func (s *ServiceBusService) completeRuntimeMessage(namespaceName string, target runtimeTarget, messageID, lockToken string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages, ok := s.runtimeMessagesForTargetLocked(namespaceName, target)
	if !ok {
		return serviceBusRuntimeError(http.StatusGone, fmt.Sprintf("Service Bus entity %q could not be found.", target.Path()))
	}
	messages = s.expireRuntimeLocksForTargetLocked(namespaceName, target, messages, time.Now().UTC())
	for i, msg := range messages {
		if msg.ID == messageID && msg.LockToken == lockToken {
			s.setRuntimeMessagesForTargetLocked(namespaceName, target, append(messages[:i], messages[i+1:]...))
			return serviceBusEmptyRuntimeSuccessResponse(), nil
		}
	}
	return serviceBusRuntimeError(http.StatusNotFound, "No message was found with the specified MessageId or LockToken.")
}

func (s *ServiceBusService) unlockRuntimeMessage(namespaceName string, target runtimeTarget, messageID, lockToken string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages, ok := s.runtimeMessagesForTargetLocked(namespaceName, target)
	if !ok {
		return serviceBusRuntimeError(http.StatusGone, fmt.Sprintf("Service Bus entity %q could not be found.", target.Path()))
	}
	messages = s.expireRuntimeLocksForTargetLocked(namespaceName, target, messages, time.Now().UTC())
	for i := range messages {
		if messages[i].ID == messageID && messages[i].LockToken == lockToken {
			if !target.DeadLetter {
				if messages[i].DeliveryCount == 0 {
					messages[i].DeliveryCount = 1
				}
				messages[i].DeliveryCount++
				if messages[i].DeliveryCount > s.maxDeliveryCountForRuntimeTargetLocked(namespaceName, target) {
					messages[i].Locked = false
					messages[i].LockToken = ""
					messages[i].DeadLetterReason = "MaxDeliveryCountExceeded"
					s.appendRuntimeDeadLetterLocked(namespaceName, target, messages[i])
					s.setRuntimeMessagesForTargetLocked(namespaceName, target, append(messages[:i], messages[i+1:]...))
					return serviceBusEmptyRuntimeSuccessResponse(), nil
				}
			}
			messages[i].Locked = false
			messages[i].LockToken = ""
			messages[i].LockedUntil = time.Time{}
			s.setRuntimeMessagesForTargetLocked(namespaceName, target, messages)
			return serviceBusEmptyRuntimeSuccessResponse(), nil
		}
	}
	return serviceBusRuntimeError(http.StatusNotFound, "No message was found with the specified MessageId or LockToken.")
}

func (s *ServiceBusService) renewRuntimeMessageLock(namespaceName string, target runtimeTarget, messageID, lockToken string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages, ok := s.runtimeMessagesForTargetLocked(namespaceName, target)
	if !ok {
		return serviceBusRuntimeError(http.StatusGone, fmt.Sprintf("Service Bus entity %q could not be found.", target.Path()))
	}
	messages = s.expireRuntimeLocksForTargetLocked(namespaceName, target, messages, time.Now().UTC())
	for i := range messages {
		if messages[i].ID == messageID && messages[i].LockToken == lockToken && messages[i].Locked {
			messages[i].LockedUntil = time.Now().UTC().Add(s.lockDurationForRuntimeTargetLocked(namespaceName, target))
			s.setRuntimeMessagesForTargetLocked(namespaceName, target, messages)
			return serviceBusEmptyRuntimeSuccessResponse(), nil
		}
	}
	return serviceBusRuntimeError(http.StatusNotFound, "No message was found with the specified MessageId or LockToken.")
}

func (s *ServiceBusService) createOrUpdateNamespace(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
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
		Type:       "Microsoft.ServiceBus/namespaces",
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

func (s *ServiceBusService) getNamespace(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	namespace, ok := s.namespaces[namespaceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus namespace %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, namespace)
}

func (s *ServiceBusService) listNamespaces(subscriptionID, resourceGroup string) (*service.Response, error) {
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

func (s *ServiceBusService) deleteNamespace(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := namespaceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.namespaces[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus namespace %q could not be found.", name))
	}
	delete(s.namespaces, key)
	queuePrefix := key + "/"
	for queueKey := range s.queues {
		if strings.HasPrefix(queueKey, queuePrefix) {
			delete(s.queues, queueKey)
		}
	}
	for topicKey := range s.topics {
		if strings.HasPrefix(topicKey, queuePrefix) {
			delete(s.topics, topicKey)
		}
	}
	for subscriptionKey := range s.subscriptions {
		if strings.HasPrefix(subscriptionKey, queuePrefix) {
			delete(s.subscriptions, subscriptionKey)
		}
	}
	for ruleKey := range s.rules {
		if strings.HasPrefix(ruleKey, queuePrefix) {
			delete(s.rules, ruleKey)
		}
	}
	for ruleKey := range s.authRules {
		if strings.HasPrefix(ruleKey, queuePrefix) {
			delete(s.authRules, ruleKey)
		}
	}
	for accessKey := range s.accessKeys {
		if strings.HasPrefix(accessKey, queuePrefix) {
			delete(s.accessKeys, accessKey)
		}
	}
	runtimePrefix := strings.ToLower(name) + "/"
	for runtimeKey := range s.runtimeQueues {
		if strings.HasPrefix(runtimeKey, runtimePrefix) {
			delete(s.runtimeQueues, runtimeKey)
		}
	}
	for runtimeKey := range s.runtimeQueueDLQs {
		if strings.HasPrefix(runtimeKey, runtimePrefix) {
			delete(s.runtimeQueueDLQs, runtimeKey)
		}
	}
	for runtimeKey := range s.runtimeTopics {
		if strings.HasPrefix(runtimeKey, runtimePrefix) {
			delete(s.runtimeTopics, runtimeKey)
		}
	}
	for runtimeKey := range s.runtimeSubscriptions {
		if strings.HasPrefix(runtimeKey, runtimePrefix) {
			delete(s.runtimeSubscriptions, runtimeKey)
		}
	}
	for runtimeKey := range s.runtimeSubDLQs {
		if strings.HasPrefix(runtimeKey, runtimePrefix) {
			delete(s.runtimeSubDLQs, runtimeKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *ServiceBusService) createOrUpdateNamespaceAuthorizationRule(subscriptionID, resourceGroup, namespaceName, ruleName string, body []byte) (*service.Response, error) {
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
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus namespace %q could not be found.", namespaceName))
	}

	rule := AuthorizationRule{
		ID:         namespaceAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, ruleName),
		Name:       ruleName,
		Type:       "Microsoft.ServiceBus/Namespaces/AuthorizationRules",
		Properties: input.Properties,
	}
	key := namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)
	s.authRules[key] = rule
	if _, ok := s.accessKeys[key]; !ok {
		s.accessKeys[key] = s.newNamespaceAccessKeysLocked(namespaceName, ruleName, key)
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *ServiceBusService) getNamespaceAuthorizationRule(subscriptionID, resourceGroup, namespaceName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	rule, ok := s.authRules[namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *ServiceBusService) listNamespaceAuthorizationRules(subscriptionID, resourceGroup, namespaceName string) (*service.Response, error) {
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
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus namespace %q could not be found.", namespaceName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ServiceBusService) deleteNamespaceAuthorizationRule(subscriptionID, resourceGroup, namespaceName, ruleName string) (*service.Response, error) {
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

func authorizationRulePropertiesFromBody(body []byte) (map[string]any, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return nil, err
		}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	if _, ok := input.Properties["rights"]; !ok {
		input.Properties["rights"] = []string{"Listen", "Send"}
	}
	return input.Properties, nil
}

func (s *ServiceBusService) listNamespaceKeys(subscriptionID, resourceGroup, namespaceName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	keys, ok := s.accessKeys[namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *ServiceBusService) regenerateNamespaceKeys(subscriptionID, resourceGroup, namespaceName, ruleName string, body []byte) (*service.Response, error) {
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
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus authorization rule %q could not be found.", ruleName))
	}
	nextKey := strings.TrimSpace(input.Key)
	if nextKey == "" {
		nextKey = s.generateAccessKeyLocked(key, input.KeyType)
	}
	if keyType == "primarykey" {
		keys.PrimaryKey = nextKey
		keys.PrimaryConnectionString = serviceBusConnectionString(namespaceName, ruleName, nextKey)
	} else {
		keys.SecondaryKey = nextKey
		keys.SecondaryConnectionString = serviceBusConnectionString(namespaceName, ruleName, nextKey)
	}
	s.accessKeys[key] = keys
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *ServiceBusService) createOrUpdateQueue(subscriptionID, resourceGroup, namespaceName, queueName string, body []byte) (*service.Response, error) {
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
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus namespace %q could not be found.", namespaceName))
	}

	queue := Queue{
		ID:         queueID(subscriptionID, resourceGroup, namespaceName, queueName),
		Name:       namespaceName + "/" + queueName,
		Type:       "Microsoft.ServiceBus/namespaces/queues",
		Properties: input.Properties,
	}
	key := queueKey(subscriptionID, resourceGroup, namespaceName, queueName)
	_, existed := s.queues[key]
	s.queues[key] = queue
	if _, ok := s.runtimeQueues[runtimeQueueKey(namespaceName, queueName)]; !ok {
		s.runtimeQueues[runtimeQueueKey(namespaceName, queueName)] = nil
	}
	if _, ok := s.runtimeQueueDLQs[runtimeQueueKey(namespaceName, queueName)]; !ok {
		s.runtimeQueueDLQs[runtimeQueueKey(namespaceName, queueName)] = nil
	}

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, queue)
}

func (s *ServiceBusService) getQueue(subscriptionID, resourceGroup, namespaceName, queueName string) (*service.Response, error) {
	s.mu.RLock()
	queue, ok := s.queues[queueKey(subscriptionID, resourceGroup, namespaceName, queueName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus queue %q could not be found.", queueName))
	}
	return azurearm.JSONResponse(http.StatusOK, queue)
}

func (s *ServiceBusService) listQueues(subscriptionID, resourceGroup, namespaceName string) (*service.Response, error) {
	nsKey := namespaceKey(subscriptionID, resourceGroup, namespaceName)

	s.mu.RLock()
	_, namespaceExists := s.namespaces[nsKey]
	values := make([]Queue, 0)
	prefix := nsKey + "/"
	for key, queue := range s.queues {
		if strings.HasPrefix(key, prefix) {
			values = append(values, queue)
		}
	}
	s.mu.RUnlock()

	if !namespaceExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus namespace %q could not be found.", namespaceName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ServiceBusService) deleteQueue(subscriptionID, resourceGroup, namespaceName, queueName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := queueKey(subscriptionID, resourceGroup, namespaceName, queueName)
	if _, ok := s.queues[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus queue %q could not be found.", queueName))
	}
	delete(s.queues, key)
	delete(s.runtimeQueues, runtimeQueueKey(namespaceName, queueName))
	delete(s.runtimeQueueDLQs, runtimeQueueKey(namespaceName, queueName))
	authRulePrefix := queueAuthorizationRulePrefix(subscriptionID, resourceGroup, namespaceName, queueName)
	for ruleKey := range s.authRules {
		if strings.HasPrefix(ruleKey, authRulePrefix) {
			delete(s.authRules, ruleKey)
			delete(s.accessKeys, ruleKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *ServiceBusService) createOrUpdateQueueAuthorizationRule(subscriptionID, resourceGroup, namespaceName, queueName, ruleName string, body []byte) (*service.Response, error) {
	properties, err := authorizationRulePropertiesFromBody(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.queues[queueKey(subscriptionID, resourceGroup, namespaceName, queueName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus queue %q could not be found.", queueName))
	}

	rule := AuthorizationRule{
		ID:         queueAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, queueName, ruleName),
		Name:       ruleName,
		Type:       "Microsoft.ServiceBus/Namespaces/Queues/AuthorizationRules",
		Properties: properties,
	}
	key := queueAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, queueName, ruleName)
	s.authRules[key] = rule
	if _, ok := s.accessKeys[key]; !ok {
		s.accessKeys[key] = s.newEntityAccessKeysLocked(namespaceName, queueName, ruleName, key)
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *ServiceBusService) getQueueAuthorizationRule(subscriptionID, resourceGroup, namespaceName, queueName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	rule, ok := s.authRules[queueAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, queueName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus queue authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *ServiceBusService) listQueueAuthorizationRules(subscriptionID, resourceGroup, namespaceName, queueName string) (*service.Response, error) {
	parentQueueKey := queueKey(subscriptionID, resourceGroup, namespaceName, queueName)

	s.mu.RLock()
	_, queueExists := s.queues[parentQueueKey]
	values := make([]AuthorizationRule, 0)
	prefix := queueAuthorizationRulePrefix(subscriptionID, resourceGroup, namespaceName, queueName)
	for key, rule := range s.authRules {
		if strings.HasPrefix(key, prefix) {
			values = append(values, rule)
		}
	}
	s.mu.RUnlock()

	if !queueExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus queue %q could not be found.", queueName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ServiceBusService) deleteQueueAuthorizationRule(subscriptionID, resourceGroup, namespaceName, queueName, ruleName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := queueAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, queueName, ruleName)
	if _, ok := s.authRules[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.authRules, key)
	delete(s.accessKeys, key)
	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *ServiceBusService) listQueueKeys(subscriptionID, resourceGroup, namespaceName, queueName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	keys, ok := s.accessKeys[queueAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, queueName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus queue authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *ServiceBusService) regenerateQueueKeys(subscriptionID, resourceGroup, namespaceName, queueName, ruleName string, body []byte) (*service.Response, error) {
	return s.regenerateEntityKeys(queueAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, queueName, ruleName), namespaceName, queueName, ruleName, body, "Service Bus queue authorization rule")
}

func (s *ServiceBusService) createOrUpdateTopic(subscriptionID, resourceGroup, namespaceName, topicName string, body []byte) (*service.Response, error) {
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
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus namespace %q could not be found.", namespaceName))
	}

	topic := Topic{
		ID:         topicID(subscriptionID, resourceGroup, namespaceName, topicName),
		Name:       namespaceName + "/" + topicName,
		Type:       "Microsoft.ServiceBus/namespaces/topics",
		Properties: input.Properties,
	}
	key := topicKey(subscriptionID, resourceGroup, namespaceName, topicName)
	_, existed := s.topics[key]
	s.topics[key] = topic
	s.runtimeTopics[runtimeTopicKey(namespaceName, topicName)] = true

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, topic)
}

func (s *ServiceBusService) getTopic(subscriptionID, resourceGroup, namespaceName, topicName string) (*service.Response, error) {
	s.mu.RLock()
	topic, ok := s.topics[topicKey(subscriptionID, resourceGroup, namespaceName, topicName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus topic %q could not be found.", topicName))
	}
	return azurearm.JSONResponse(http.StatusOK, topic)
}

func (s *ServiceBusService) listTopics(subscriptionID, resourceGroup, namespaceName string) (*service.Response, error) {
	nsKey := namespaceKey(subscriptionID, resourceGroup, namespaceName)

	s.mu.RLock()
	_, namespaceExists := s.namespaces[nsKey]
	values := make([]Topic, 0)
	prefix := nsKey + "/"
	for key, topic := range s.topics {
		if strings.HasPrefix(key, prefix) {
			values = append(values, topic)
		}
	}
	s.mu.RUnlock()

	if !namespaceExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus namespace %q could not be found.", namespaceName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ServiceBusService) deleteTopic(subscriptionID, resourceGroup, namespaceName, topicName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := topicKey(subscriptionID, resourceGroup, namespaceName, topicName)
	if _, ok := s.topics[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus topic %q could not be found.", topicName))
	}
	delete(s.topics, key)
	delete(s.runtimeTopics, runtimeTopicKey(namespaceName, topicName))
	subscriptionPrefix := key + "/"
	for subscriptionKey := range s.subscriptions {
		if strings.HasPrefix(subscriptionKey, subscriptionPrefix) {
			delete(s.subscriptions, subscriptionKey)
		}
	}
	for ruleKey := range s.rules {
		if strings.HasPrefix(ruleKey, subscriptionPrefix) {
			delete(s.rules, ruleKey)
		}
	}
	runtimeSubscriptionPrefix := runtimeTopicKey(namespaceName, topicName) + "/"
	for runtimeKey := range s.runtimeSubscriptions {
		if strings.HasPrefix(runtimeKey, runtimeSubscriptionPrefix) {
			delete(s.runtimeSubscriptions, runtimeKey)
		}
	}
	for runtimeKey := range s.runtimeSubDLQs {
		if strings.HasPrefix(runtimeKey, runtimeSubscriptionPrefix) {
			delete(s.runtimeSubDLQs, runtimeKey)
		}
	}
	authRulePrefix := topicAuthorizationRulePrefix(subscriptionID, resourceGroup, namespaceName, topicName)
	for ruleKey := range s.authRules {
		if strings.HasPrefix(ruleKey, authRulePrefix) {
			delete(s.authRules, ruleKey)
			delete(s.accessKeys, ruleKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *ServiceBusService) createOrUpdateTopicAuthorizationRule(subscriptionID, resourceGroup, namespaceName, topicName, ruleName string, body []byte) (*service.Response, error) {
	properties, err := authorizationRulePropertiesFromBody(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.topics[topicKey(subscriptionID, resourceGroup, namespaceName, topicName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus topic %q could not be found.", topicName))
	}

	rule := AuthorizationRule{
		ID:         topicAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, topicName, ruleName),
		Name:       ruleName,
		Type:       "Microsoft.ServiceBus/Namespaces/Topics/AuthorizationRules",
		Properties: properties,
	}
	key := topicAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, topicName, ruleName)
	s.authRules[key] = rule
	if _, ok := s.accessKeys[key]; !ok {
		s.accessKeys[key] = s.newEntityAccessKeysLocked(namespaceName, topicName, ruleName, key)
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *ServiceBusService) getTopicAuthorizationRule(subscriptionID, resourceGroup, namespaceName, topicName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	rule, ok := s.authRules[topicAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, topicName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus topic authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *ServiceBusService) listTopicAuthorizationRules(subscriptionID, resourceGroup, namespaceName, topicName string) (*service.Response, error) {
	parentTopicKey := topicKey(subscriptionID, resourceGroup, namespaceName, topicName)

	s.mu.RLock()
	_, topicExists := s.topics[parentTopicKey]
	values := make([]AuthorizationRule, 0)
	prefix := topicAuthorizationRulePrefix(subscriptionID, resourceGroup, namespaceName, topicName)
	for key, rule := range s.authRules {
		if strings.HasPrefix(key, prefix) {
			values = append(values, rule)
		}
	}
	s.mu.RUnlock()

	if !topicExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus topic %q could not be found.", topicName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ServiceBusService) deleteTopicAuthorizationRule(subscriptionID, resourceGroup, namespaceName, topicName, ruleName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := topicAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, topicName, ruleName)
	if _, ok := s.authRules[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.authRules, key)
	delete(s.accessKeys, key)
	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *ServiceBusService) listTopicKeys(subscriptionID, resourceGroup, namespaceName, topicName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	keys, ok := s.accessKeys[topicAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, topicName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus topic authorization rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *ServiceBusService) regenerateTopicKeys(subscriptionID, resourceGroup, namespaceName, topicName, ruleName string, body []byte) (*service.Response, error) {
	return s.regenerateEntityKeys(topicAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, topicName, ruleName), namespaceName, topicName, ruleName, body, "Service Bus topic authorization rule")
}

func (s *ServiceBusService) createOrUpdateSubscription(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName string, body []byte) (*service.Response, error) {
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

	if _, ok := s.topics[topicKey(subscriptionID, resourceGroup, namespaceName, topicName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus topic %q could not be found.", topicName))
	}

	subscription := Subscription{
		ID:         subscriptionIDForResource(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName),
		Name:       namespaceName + "/" + topicName + "/" + subscriptionName,
		Type:       "Microsoft.ServiceBus/namespaces/topics/subscriptions",
		Properties: input.Properties,
	}
	key := subscriptionKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName)
	_, existed := s.subscriptions[key]
	s.subscriptions[key] = subscription
	if _, ok := s.runtimeSubscriptions[runtimeSubscriptionKey(namespaceName, topicName, subscriptionName)]; !ok {
		s.runtimeSubscriptions[runtimeSubscriptionKey(namespaceName, topicName, subscriptionName)] = nil
	}
	if _, ok := s.runtimeSubDLQs[runtimeSubscriptionKey(namespaceName, topicName, subscriptionName)]; !ok {
		s.runtimeSubDLQs[runtimeSubscriptionKey(namespaceName, topicName, subscriptionName)] = nil
	}

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, subscription)
}

func (s *ServiceBusService) getSubscription(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName string) (*service.Response, error) {
	s.mu.RLock()
	subscription, ok := s.subscriptions[subscriptionKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus subscription %q could not be found.", subscriptionName))
	}
	return azurearm.JSONResponse(http.StatusOK, subscription)
}

func (s *ServiceBusService) listSubscriptions(subscriptionID, resourceGroup, namespaceName, topicName string) (*service.Response, error) {
	parentTopicKey := topicKey(subscriptionID, resourceGroup, namespaceName, topicName)

	s.mu.RLock()
	_, topicExists := s.topics[parentTopicKey]
	values := make([]Subscription, 0)
	prefix := parentTopicKey + "/"
	for key, subscription := range s.subscriptions {
		if strings.HasPrefix(key, prefix) {
			values = append(values, subscription)
		}
	}
	s.mu.RUnlock()

	if !topicExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus topic %q could not be found.", topicName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ServiceBusService) deleteSubscription(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := subscriptionKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName)
	if _, ok := s.subscriptions[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus subscription %q could not be found.", subscriptionName))
	}
	delete(s.subscriptions, key)
	rulePrefix := key + "/"
	for ruleKey := range s.rules {
		if strings.HasPrefix(ruleKey, rulePrefix) {
			delete(s.rules, ruleKey)
		}
	}
	delete(s.runtimeSubscriptions, runtimeSubscriptionKey(namespaceName, topicName, subscriptionName))
	delete(s.runtimeSubDLQs, runtimeSubscriptionKey(namespaceName, topicName, subscriptionName))
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *ServiceBusService) createOrUpdateRule(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	properties := normalizeRuleProperties(input.Properties)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subscriptions[subscriptionKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus subscription %q could not be found.", subscriptionName))
	}

	rule := Rule{
		ID:         ruleID(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName),
		Name:       ruleName,
		Type:       "Microsoft.ServiceBus/Namespaces/Topics/Subscriptions/Rules",
		Properties: properties,
	}
	s.rules[ruleKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName)] = rule
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *ServiceBusService) getRule(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	rule, ok := s.rules[ruleKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *ServiceBusService) listRules(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName string) (*service.Response, error) {
	parentSubscriptionKey := subscriptionKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName)

	s.mu.RLock()
	_, subscriptionExists := s.subscriptions[parentSubscriptionKey]
	values := make([]Rule, 0)
	prefix := parentSubscriptionKey + "/"
	for key, rule := range s.rules {
		if strings.HasPrefix(key, prefix) {
			values = append(values, rule)
		}
	}
	s.mu.RUnlock()

	if !subscriptionExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Service Bus subscription %q could not be found.", subscriptionName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ServiceBusService) deleteRule(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := ruleKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName)
	if _, ok := s.rules[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.rules, key)
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

func (s *ServiceBusService) ensureRootAuthorizationRuleLocked(subscriptionID, resourceGroup, namespaceName string) {
	const ruleName = "RootManageSharedAccessKey"
	key := namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName)
	if _, ok := s.authRules[key]; ok {
		return
	}
	s.authRules[key] = AuthorizationRule{
		ID:   namespaceAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, ruleName),
		Name: ruleName,
		Type: "Microsoft.ServiceBus/Namespaces/AuthorizationRules",
		Properties: map[string]any{
			"rights": []string{"Listen", "Manage", "Send"},
		},
	}
	s.accessKeys[key] = s.newNamespaceAccessKeysLocked(namespaceName, ruleName, key)
}

func (s *ServiceBusService) newNamespaceAccessKeysLocked(namespaceName, ruleName, key string) AccessKeys {
	primary := s.generateAccessKeyLocked(key, "PrimaryKey")
	secondary := s.generateAccessKeyLocked(key, "SecondaryKey")
	return AccessKeys{
		KeyName:                   ruleName,
		PrimaryKey:                primary,
		SecondaryKey:              secondary,
		PrimaryConnectionString:   serviceBusConnectionString(namespaceName, ruleName, primary),
		SecondaryConnectionString: serviceBusConnectionString(namespaceName, ruleName, secondary),
	}
}

func (s *ServiceBusService) newEntityAccessKeysLocked(namespaceName, entityPath, ruleName, key string) AccessKeys {
	primary := s.generateAccessKeyLocked(key, "PrimaryKey")
	secondary := s.generateAccessKeyLocked(key, "SecondaryKey")
	return AccessKeys{
		KeyName:                   ruleName,
		PrimaryKey:                primary,
		SecondaryKey:              secondary,
		PrimaryConnectionString:   serviceBusEntityConnectionString(namespaceName, entityPath, ruleName, primary),
		SecondaryConnectionString: serviceBusEntityConnectionString(namespaceName, entityPath, ruleName, secondary),
	}
}

func (s *ServiceBusService) regenerateEntityKeys(key, namespaceName, entityPath, ruleName string, body []byte, missingResourceName string) (*service.Response, error) {
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

	keys, ok := s.accessKeys[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("%s %q could not be found.", missingResourceName, ruleName))
	}
	nextKey := strings.TrimSpace(input.Key)
	if nextKey == "" {
		nextKey = s.generateAccessKeyLocked(key, input.KeyType)
	}
	if keyType == "primarykey" {
		keys.PrimaryKey = nextKey
		keys.PrimaryConnectionString = serviceBusEntityConnectionString(namespaceName, entityPath, ruleName, nextKey)
	} else {
		keys.SecondaryKey = nextKey
		keys.SecondaryConnectionString = serviceBusEntityConnectionString(namespaceName, entityPath, ruleName, nextKey)
	}
	s.accessKeys[key] = keys
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *ServiceBusService) generateAccessKeyLocked(key, keyType string) string {
	s.keyGeneration++
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", key, keyType, s.keyGeneration)))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func serviceBusConnectionString(namespaceName, ruleName, key string) string {
	return "Endpoint=sb://" + namespaceName + ".servicebus.windows.net/;SharedAccessKeyName=" + ruleName + ";SharedAccessKey=" + key
}

func serviceBusEntityConnectionString(namespaceName, entityPath, ruleName, key string) string {
	return serviceBusConnectionString(namespaceName, ruleName, key) + ";EntityPath=" + entityPath
}

func (s *ServiceBusService) activeNamespaceRef() (serviceBusNamespaceRef, bool) {
	s.mu.RLock()
	refs := make([]serviceBusNamespaceRef, 0, len(s.namespaces))
	for _, namespace := range s.namespaces {
		parts := splitPath(namespace.ID)
		if len(parts) < 8 {
			continue
		}
		refs = append(refs, serviceBusNamespaceRef{
			SubscriptionID: parts[1],
			ResourceGroup:  parts[3],
			Name:           namespace.Name,
		})
	}
	s.mu.RUnlock()
	if len(refs) == 0 {
		return serviceBusNamespaceRef{}, false
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Name < refs[j].Name })
	return refs[0], true
}

func serviceBusLocalSubscriptionID(ctx *service.RequestContext) string {
	if ctx != nil && strings.TrimSpace(ctx.AccountID) != "" {
		return ctx.AccountID
	}
	return "sub-1"
}

type serviceBusRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	NamespaceName  string
	ChildType      string
	ChildName      string
	GrandchildType string
	GrandchildName string
	RuleType       string
	RuleName       string
}

func parseRoute(escapedPath string) (serviceBusRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.ServiceBus") {
		return serviceBusRoute{}, false
	}
	route := serviceBusRoute{
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
		route.GrandchildType = parts[10]
		return route, true
	case 12:
		route.NamespaceName = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandchildType = parts[10]
		route.GrandchildName = parts[11]
		return route, true
	case 13:
		route.NamespaceName = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandchildType = parts[10]
		route.GrandchildName = parts[11]
		route.RuleType = parts[12]
		return route, true
	case 14:
		route.NamespaceName = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandchildType = parts[10]
		route.GrandchildName = parts[11]
		route.RuleType = parts[12]
		route.RuleName = parts[13]
		return route, true
	default:
		return serviceBusRoute{}, false
	}
}

func namespaceID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ServiceBus/namespaces/" + name
}

func queueID(subscriptionID, resourceGroup, namespaceName, queueName string) string {
	return namespaceID(subscriptionID, resourceGroup, namespaceName) + "/queues/" + queueName
}

func topicID(subscriptionID, resourceGroup, namespaceName, topicName string) string {
	return namespaceID(subscriptionID, resourceGroup, namespaceName) + "/topics/" + topicName
}

func subscriptionIDForResource(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName string) string {
	return topicID(subscriptionID, resourceGroup, namespaceName, topicName) + "/subscriptions/" + subscriptionName
}

func ruleID(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName string) string {
	return subscriptionIDForResource(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName) + "/rules/" + ruleName
}

func namespaceAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, ruleName string) string {
	return namespaceID(subscriptionID, resourceGroup, namespaceName) + "/AuthorizationRules/" + ruleName
}

func queueAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, queueName, ruleName string) string {
	return queueID(subscriptionID, resourceGroup, namespaceName, queueName) + "/AuthorizationRules/" + ruleName
}

func topicAuthorizationRuleID(subscriptionID, resourceGroup, namespaceName, topicName, ruleName string) string {
	return topicID(subscriptionID, resourceGroup, namespaceName, topicName) + "/AuthorizationRules/" + ruleName
}

func namespaceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func namespaceAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, ruleName string) string {
	return namespaceKey(subscriptionID, resourceGroup, namespaceName) + "/authorizationrules/" + strings.ToLower(ruleName)
}

func queueAuthorizationRulePrefix(subscriptionID, resourceGroup, namespaceName, queueName string) string {
	return namespaceKey(subscriptionID, resourceGroup, namespaceName) + "/queues/" + strings.ToLower(queueName) + "/authorizationrules/"
}

func queueAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, queueName, ruleName string) string {
	return queueAuthorizationRulePrefix(subscriptionID, resourceGroup, namespaceName, queueName) + strings.ToLower(ruleName)
}

func topicAuthorizationRulePrefix(subscriptionID, resourceGroup, namespaceName, topicName string) string {
	return namespaceKey(subscriptionID, resourceGroup, namespaceName) + "/topics/" + strings.ToLower(topicName) + "/authorizationrules/"
}

func topicAuthorizationRuleKey(subscriptionID, resourceGroup, namespaceName, topicName, ruleName string) string {
	return topicAuthorizationRulePrefix(subscriptionID, resourceGroup, namespaceName, topicName) + strings.ToLower(ruleName)
}

func queueKey(subscriptionID, resourceGroup, namespaceName, queueName string) string {
	return namespaceKey(subscriptionID, resourceGroup, namespaceName) + "/" + strings.ToLower(queueName)
}

func topicKey(subscriptionID, resourceGroup, namespaceName, topicName string) string {
	return namespaceKey(subscriptionID, resourceGroup, namespaceName) + "/" + strings.ToLower(topicName)
}

func subscriptionKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName string) string {
	return topicKey(subscriptionID, resourceGroup, namespaceName, topicName) + "/" + strings.ToLower(subscriptionName)
}

func ruleKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName, ruleName string) string {
	return subscriptionKey(subscriptionID, resourceGroup, namespaceName, topicName, subscriptionName) + "/" + strings.ToLower(ruleName)
}

func flociServiceBusAccountAndPath(req *http.Request) (string, []string, bool) {
	parts := splitPath(req.URL.EscapedPath())
	if len(parts) == 0 {
		return "", nil, false
	}
	root := strings.ToLower(parts[0])
	if !strings.HasSuffix(root, "-servicebus") {
		return "", nil, false
	}
	accountName := strings.TrimSuffix(parts[0], "-servicebus")
	if accountName == "" {
		return "", nil, false
	}
	return accountName, parts[1:], true
}

func serviceBusAtomResponse(status int, body string) (*service.Response, error) {
	return &service.Response{
		StatusCode:     status,
		RawBody:        []byte(body),
		RawContentType: serviceBusAtomContentType,
	}, nil
}

func serviceBusAtomNotFound(message string) (*service.Response, error) {
	return serviceBusAtomResponse(http.StatusNotFound, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>404</Code><Detail>`+serviceBusXMLEscape(message)+`</Detail></Error>`)
}

func serviceBusNamespaceInfoAtom(namespaceName string) string {
	now := serviceBusAtomTimestamp()
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<entry xmlns="http://www.w3.org/2005/Atom">` +
		`<id>https://localhost/$namespaceinfo</id>` +
		`<title type="text">` + serviceBusXMLEscape(namespaceName) + `</title>` +
		`<updated>` + now + `</updated>` +
		`<content type="application/xml">` +
		`<NamespaceInfo xmlns="` + serviceBusAtomNamespace + `">` +
		`<Name>` + serviceBusXMLEscape(namespaceName) + `</Name>` +
		`<MessagingSKU>Standard</MessagingSKU>` +
		`<NamespaceType>Messaging</NamespaceType>` +
		`<CreatedTime>` + now + `</CreatedTime>` +
		`<ModifiedTime>` + now + `</ModifiedTime>` +
		`</NamespaceInfo>` +
		`</content>` +
		`</entry>`
}

func (s *ServiceBusService) flociQueueFeed(namespaceRef serviceBusNamespaceRef) string {
	prefix := namespaceKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name) + "/"
	s.mu.RLock()
	values := make([]Queue, 0)
	for key, queue := range s.queues {
		if strings.HasPrefix(key, prefix) {
			values = append(values, queue)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })

	now := serviceBusAtomTimestamp()
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	builder.WriteString(`<feed xmlns="http://www.w3.org/2005/Atom">`)
	builder.WriteString(`<title type="text">Queues</title>`)
	builder.WriteString(`<id>https://localhost/`)
	builder.WriteString(serviceBusXMLEscape(namespaceRef.Name))
	builder.WriteString(`/$Resources/queues</id>`)
	builder.WriteString(`<updated>`)
	builder.WriteString(now)
	builder.WriteString(`</updated>`)
	for _, queue := range values {
		builder.WriteString(serviceBusQueueAtomEntry(namespaceRef.Name, queue))
	}
	builder.WriteString(`</feed>`)
	return builder.String()
}

func (s *ServiceBusService) flociTopicFeed(namespaceRef serviceBusNamespaceRef) string {
	prefix := namespaceKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name) + "/"
	s.mu.RLock()
	values := make([]Topic, 0)
	for key, topic := range s.topics {
		if strings.HasPrefix(key, prefix) {
			values = append(values, topic)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })

	now := serviceBusAtomTimestamp()
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	builder.WriteString(`<feed xmlns="http://www.w3.org/2005/Atom">`)
	builder.WriteString(`<title type="text">Topics</title>`)
	builder.WriteString(`<id>https://localhost/`)
	builder.WriteString(serviceBusXMLEscape(namespaceRef.Name))
	builder.WriteString(`/$Resources/topics</id>`)
	builder.WriteString(`<updated>`)
	builder.WriteString(now)
	builder.WriteString(`</updated>`)
	for _, topic := range values {
		builder.WriteString(serviceBusTopicAtomEntry(namespaceRef.Name, topic))
	}
	builder.WriteString(`</feed>`)
	return builder.String()
}

func (s *ServiceBusService) flociSubscriptionFeed(namespaceRef serviceBusNamespaceRef, topicName string) string {
	prefix := subscriptionKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, "")
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix != "" {
		prefix += "/"
	}

	s.mu.RLock()
	values := make([]Subscription, 0)
	for key, subscription := range s.subscriptions {
		if strings.HasPrefix(key, prefix) {
			values = append(values, subscription)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })

	now := serviceBusAtomTimestamp()
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	builder.WriteString(`<feed xmlns="http://www.w3.org/2005/Atom">`)
	builder.WriteString(`<title type="text">Subscriptions</title>`)
	builder.WriteString(`<id>https://localhost/`)
	builder.WriteString(serviceBusXMLEscape(namespaceRef.Name))
	builder.WriteString(`/topics/`)
	builder.WriteString(serviceBusXMLEscape(topicName))
	builder.WriteString(`/subscriptions</id>`)
	builder.WriteString(`<updated>`)
	builder.WriteString(now)
	builder.WriteString(`</updated>`)
	for _, subscription := range values {
		builder.WriteString(serviceBusSubscriptionAtomEntry(namespaceRef.Name, topicName, subscription))
	}
	builder.WriteString(`</feed>`)
	return builder.String()
}

func (s *ServiceBusService) flociRuleFeed(namespaceRef serviceBusNamespaceRef, topicName, subscriptionName string) string {
	prefix := ruleKey(namespaceRef.SubscriptionID, namespaceRef.ResourceGroup, namespaceRef.Name, topicName, subscriptionName, "")
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix != "" {
		prefix += "/"
	}

	s.mu.RLock()
	values := make([]Rule, 0)
	for key, rule := range s.rules {
		if strings.HasPrefix(key, prefix) {
			values = append(values, rule)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })

	now := serviceBusAtomTimestamp()
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	builder.WriteString(`<feed xmlns="http://www.w3.org/2005/Atom">`)
	builder.WriteString(`<title type="text">Rules</title>`)
	builder.WriteString(`<id>https://localhost/`)
	builder.WriteString(serviceBusXMLEscape(namespaceRef.Name))
	builder.WriteString(`/topics/`)
	builder.WriteString(serviceBusXMLEscape(topicName))
	builder.WriteString(`/subscriptions/`)
	builder.WriteString(serviceBusXMLEscape(subscriptionName))
	builder.WriteString(`/rules</id>`)
	builder.WriteString(`<updated>`)
	builder.WriteString(now)
	builder.WriteString(`</updated>`)
	for _, rule := range values {
		builder.WriteString(serviceBusRuleAtomEntry(namespaceRef.Name, topicName, subscriptionName, rule))
	}
	builder.WriteString(`</feed>`)
	return builder.String()
}

func serviceBusQueueAtomEntry(namespaceName string, queue Queue) string {
	queueName := serviceBusLeafName(queue.Name)
	now := serviceBusAtomTimestamp()
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<entry xmlns="http://www.w3.org/2005/Atom">` +
		`<id>https://localhost/` + serviceBusXMLEscape(namespaceName) + `/queues/` + serviceBusXMLEscape(queueName) + `</id>` +
		`<title type="text">` + serviceBusXMLEscape(queueName) + `</title>` +
		`<published>` + now + `</published>` +
		`<updated>` + now + `</updated>` +
		`<content type="application/xml">` +
		serviceBusQueueDescriptionXML(queue) +
		`</content>` +
		`</entry>`
}

func serviceBusTopicAtomEntry(namespaceName string, topic Topic) string {
	topicName := serviceBusLeafName(topic.Name)
	now := serviceBusAtomTimestamp()
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<entry xmlns="http://www.w3.org/2005/Atom">` +
		`<id>https://localhost/` + serviceBusXMLEscape(namespaceName) + `/topics/` + serviceBusXMLEscape(topicName) + `</id>` +
		`<title type="text">` + serviceBusXMLEscape(topicName) + `</title>` +
		`<published>` + now + `</published>` +
		`<updated>` + now + `</updated>` +
		`<content type="application/xml">` +
		serviceBusTopicDescriptionXML(topic) +
		`</content>` +
		`</entry>`
}

func serviceBusSubscriptionAtomEntry(namespaceName, topicName string, subscription Subscription) string {
	subscriptionName := serviceBusLeafName(subscription.Name)
	now := serviceBusAtomTimestamp()
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<entry xmlns="http://www.w3.org/2005/Atom">` +
		`<id>https://localhost/` + serviceBusXMLEscape(namespaceName) + `/topics/` + serviceBusXMLEscape(topicName) + `/subscriptions/` + serviceBusXMLEscape(subscriptionName) + `</id>` +
		`<title type="text">` + serviceBusXMLEscape(subscriptionName) + `</title>` +
		`<published>` + now + `</published>` +
		`<updated>` + now + `</updated>` +
		`<content type="application/xml">` +
		serviceBusSubscriptionDescriptionXML(subscription) +
		`</content>` +
		`</entry>`
}

func serviceBusRuleAtomEntry(namespaceName, topicName, subscriptionName string, rule Rule) string {
	now := serviceBusAtomTimestamp()
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<entry xmlns="http://www.w3.org/2005/Atom">` +
		`<id>https://localhost/` + serviceBusXMLEscape(namespaceName) + `/topics/` + serviceBusXMLEscape(topicName) + `/subscriptions/` + serviceBusXMLEscape(subscriptionName) + `/rules/` + serviceBusXMLEscape(rule.Name) + `</id>` +
		`<title type="text">` + serviceBusXMLEscape(rule.Name) + `</title>` +
		`<published>` + now + `</published>` +
		`<updated>` + now + `</updated>` +
		`<content type="application/xml">` +
		serviceBusRuleDescriptionXML(rule) +
		`</content>` +
		`</entry>`
}

func serviceBusQueueDescriptionXML(queue Queue) string {
	props := queue.Properties
	return `<QueueDescription xmlns="` + serviceBusAtomNamespace + `">` +
		`<MaxSizeInMegabytes>` + serviceBusIntProperty(props, "maxSizeInMegabytes", 1024) + `</MaxSizeInMegabytes>` +
		`<DefaultMessageTimeToLive>` + serviceBusStringProperty(props, "defaultMessageTimeToLive", "P14D") + `</DefaultMessageTimeToLive>` +
		`<LockDuration>` + serviceBusStringProperty(props, "lockDuration", "PT30S") + `</LockDuration>` +
		`<MaxDeliveryCount>` + serviceBusIntProperty(props, "maxDeliveryCount", 10) + `</MaxDeliveryCount>` +
		`<RequiresDuplicateDetection>` + serviceBusBoolProperty(props, "requiresDuplicateDetection", false) + `</RequiresDuplicateDetection>` +
		`<RequiresSession>` + serviceBusBoolProperty(props, "requiresSession", false) + `</RequiresSession>` +
		`<DeadLetteringOnMessageExpiration>` + serviceBusBoolProperty(props, "deadLetteringOnMessageExpiration", false) + `</DeadLetteringOnMessageExpiration>` +
		`<EnableBatchedOperations>` + serviceBusBoolProperty(props, "enableBatchedOperations", true) + `</EnableBatchedOperations>` +
		`<AutoDeleteOnIdle>` + serviceBusStringProperty(props, "autoDeleteOnIdle", "P10675199DT2H48M5.4775807S") + `</AutoDeleteOnIdle>` +
		`<Status>Active</Status>` +
		`<EntityAvailabilityStatus>Available</EntityAvailabilityStatus>` +
		serviceBusOptionalStringElement(props, "userMetadata", "UserMetadata") +
		`</QueueDescription>`
}

func serviceBusTopicDescriptionXML(topic Topic) string {
	props := topic.Properties
	return `<TopicDescription xmlns="` + serviceBusAtomNamespace + `">` +
		`<MaxSizeInMegabytes>` + serviceBusIntProperty(props, "maxSizeInMegabytes", 1024) + `</MaxSizeInMegabytes>` +
		`<DefaultMessageTimeToLive>` + serviceBusStringProperty(props, "defaultMessageTimeToLive", "P14D") + `</DefaultMessageTimeToLive>` +
		`<RequiresDuplicateDetection>` + serviceBusBoolProperty(props, "requiresDuplicateDetection", false) + `</RequiresDuplicateDetection>` +
		`<EnableBatchedOperations>` + serviceBusBoolProperty(props, "enableBatchedOperations", true) + `</EnableBatchedOperations>` +
		`<SupportOrdering>` + serviceBusBoolProperty(props, "supportOrdering", false) + `</SupportOrdering>` +
		`<AutoDeleteOnIdle>` + serviceBusStringProperty(props, "autoDeleteOnIdle", "P10675199DT2H48M5.4775807S") + `</AutoDeleteOnIdle>` +
		`<Status>Active</Status>` +
		`<EntityAvailabilityStatus>Available</EntityAvailabilityStatus>` +
		serviceBusOptionalStringElement(props, "userMetadata", "UserMetadata") +
		`</TopicDescription>`
}

func serviceBusSubscriptionDescriptionXML(subscription Subscription) string {
	props := subscription.Properties
	return `<SubscriptionDescription xmlns="` + serviceBusAtomNamespace + `">` +
		`<LockDuration>` + serviceBusStringProperty(props, "lockDuration", "PT30S") + `</LockDuration>` +
		`<MaxDeliveryCount>` + serviceBusIntProperty(props, "maxDeliveryCount", 10) + `</MaxDeliveryCount>` +
		`<RequiresSession>` + serviceBusBoolProperty(props, "requiresSession", false) + `</RequiresSession>` +
		`<DefaultMessageTimeToLive>` + serviceBusStringProperty(props, "defaultMessageTimeToLive", "P14D") + `</DefaultMessageTimeToLive>` +
		`<DeadLetteringOnMessageExpiration>` + serviceBusBoolProperty(props, "deadLetteringOnMessageExpiration", false) + `</DeadLetteringOnMessageExpiration>` +
		`<DeadLetteringOnFilterEvaluationExceptions>` + serviceBusBoolProperty(props, "deadLetteringOnFilterEvaluationExceptions", true) + `</DeadLetteringOnFilterEvaluationExceptions>` +
		`<EnableBatchedOperations>` + serviceBusBoolProperty(props, "enableBatchedOperations", true) + `</EnableBatchedOperations>` +
		`<AutoDeleteOnIdle>` + serviceBusStringProperty(props, "autoDeleteOnIdle", "P10675199DT2H48M5.4775807S") + `</AutoDeleteOnIdle>` +
		`<Status>Active</Status>` +
		`<EntityAvailabilityStatus>Available</EntityAvailabilityStatus>` +
		serviceBusOptionalStringElement(props, "userMetadata", "UserMetadata") +
		`</SubscriptionDescription>`
}

func serviceBusRuleDescriptionXML(rule Rule) string {
	return `<RuleDescription xmlns="` + serviceBusAtomNamespace + `">` +
		serviceBusRuleFilterXML(rule) +
		serviceBusRuleActionXML(nestedAnyMap(rule.Properties["action"])) +
		`</RuleDescription>`
}

func serviceBusRuleFilterXML(rule Rule) string {
	if strings.EqualFold(stringValue(rule.Properties["filterType"]), "CorrelationFilter") {
		return serviceBusCorrelationFilterXML(nestedAnyMap(rule.Properties["correlationFilter"]))
	}
	sqlFilter := nestedAnyMap(rule.Properties["sqlFilter"])
	sqlExpression := stringValue(sqlFilter["sqlExpression"])
	if sqlExpression == "" {
		sqlExpression = "1=1"
	}
	return `<Filter xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="SqlFilter">` +
		`<SqlExpression>` + serviceBusXMLEscape(sqlExpression) + `</SqlExpression>` +
		`<CompatibilityLevel>` + serviceBusIntProperty(sqlFilter, "compatibilityLevel", 20) + `</CompatibilityLevel>` +
		`</Filter>`
}

func serviceBusCorrelationFilterXML(filter map[string]any) string {
	out := `<Filter xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="CorrelationFilter">`
	for _, field := range []struct {
		property string
		tag      string
	}{
		{"correlationId", "CorrelationId"},
		{"contentType", "ContentType"},
		{"messageId", "MessageId"},
		{"replyTo", "ReplyTo"},
		{"replyToSessionId", "ReplyToSessionId"},
		{"sessionId", "SessionId"},
		{"subject", "Subject"},
		{"label", "Label"},
		{"to", "To"},
	} {
		out += serviceBusOptionalStringElement(filter, field.property, field.tag)
	}
	out += serviceBusCorrelationPropertiesXML(nestedAnyMap(filter["properties"]))
	out += `</Filter>`
	return out
}

func serviceBusCorrelationPropertiesXML(properties map[string]any) string {
	if len(properties) == 0 {
		return ""
	}
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := `<Properties>`
	for _, key := range keys {
		value := fmt.Sprint(properties[key])
		out += `<KeyValueOfstringanyType><Key>` + serviceBusXMLEscape(key) + `</Key><Value>` + serviceBusXMLEscape(value) + `</Value></KeyValueOfstringanyType>`
	}
	out += `</Properties>`
	return out
}

func serviceBusRuleActionXML(action map[string]any) string {
	sqlExpression := stringValue(action["sqlExpression"])
	if sqlExpression == "" {
		return ""
	}
	return `<Action xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="SqlRuleAction">` +
		`<SqlExpression>` + serviceBusXMLEscape(sqlExpression) + `</SqlExpression>` +
		`<CompatibilityLevel>` + serviceBusIntProperty(action, "compatibilityLevel", 20) + `</CompatibilityLevel>` +
		`</Action>`
}

func serviceBusRulePropertiesFromAtom(body []byte) map[string]any {
	bodyText := string(body)
	filterBlock := serviceBusXMLBlock(bodyText, "Filter")
	if serviceBusXMLStartTagHasType(bodyText, "Filter", "CorrelationFilter") {
		properties := map[string]any{
			"filterType":        "CorrelationFilter",
			"correlationFilter": serviceBusCorrelationFilterFromAtom(filterBlock),
		}
		actionBlock := serviceBusXMLBlock(bodyText, "Action")
		if actionExpression := serviceBusXMLTagValue(actionBlock, "SqlExpression"); strings.TrimSpace(actionExpression) != "" {
			properties["action"] = map[string]any{
				"sqlExpression":      actionExpression,
				"compatibilityLevel": 20,
			}
		}
		return properties
	}
	sqlExpression := serviceBusXMLTagValue(filterBlock, "SqlExpression")
	if strings.TrimSpace(sqlExpression) == "" {
		sqlExpression = serviceBusXMLTagValue(bodyText, "SqlExpression")
	}
	if strings.TrimSpace(sqlExpression) == "" {
		sqlExpression = "1=1"
	}
	properties := map[string]any{
		"filterType": "SqlFilter",
		"sqlFilter": map[string]any{
			"sqlExpression":      sqlExpression,
			"compatibilityLevel": 20,
		},
	}
	actionBlock := serviceBusXMLBlock(bodyText, "Action")
	if actionExpression := serviceBusXMLTagValue(actionBlock, "SqlExpression"); strings.TrimSpace(actionExpression) != "" {
		properties["action"] = map[string]any{
			"sqlExpression":      actionExpression,
			"compatibilityLevel": 20,
		}
	}
	return properties
}

func serviceBusCorrelationFilterFromAtom(filterBlock string) map[string]any {
	filter := map[string]any{}
	for _, field := range []struct {
		tag      string
		property string
	}{
		{"CorrelationId", "correlationId"},
		{"ContentType", "contentType"},
		{"MessageId", "messageId"},
		{"ReplyTo", "replyTo"},
		{"ReplyToSessionId", "replyToSessionId"},
		{"SessionId", "sessionId"},
		{"Subject", "subject"},
		{"Label", "label"},
		{"To", "to"},
	} {
		if value := serviceBusXMLTagValue(filterBlock, field.tag); strings.TrimSpace(value) != "" {
			filter[field.property] = value
		}
	}
	if properties := serviceBusCorrelationFilterPropertiesFromAtom(serviceBusXMLBlock(filterBlock, "Properties")); len(properties) > 0 {
		filter["properties"] = properties
	}
	return filter
}

func serviceBusCorrelationFilterPropertiesFromAtom(propertiesBlock string) map[string]any {
	properties := map[string]any{}
	for _, entry := range serviceBusXMLBlocks(propertiesBlock, "KeyValueOfstringanyType") {
		key := serviceBusXMLTagValue(entry, "Key")
		if strings.TrimSpace(key) == "" {
			continue
		}
		properties[key] = serviceBusXMLTagValue(entry, "Value")
	}
	return properties
}

func serviceBusQueuePropertiesFromAtom(body []byte) map[string]any {
	properties := map[string]any{
		"lockDuration":            "PT30S",
		"maxDeliveryCount":        10,
		"requiresSession":         false,
		"maxSizeInMegabytes":      1024,
		"enableBatchedOperations": true,
	}
	serviceBusApplyStringAtomProperty(properties, body, "DefaultMessageTimeToLive", "defaultMessageTimeToLive")
	serviceBusApplyStringAtomProperty(properties, body, "LockDuration", "lockDuration")
	serviceBusApplyIntAtomProperty(properties, body, "MaxDeliveryCount", "maxDeliveryCount")
	serviceBusApplyIntAtomProperty(properties, body, "MaxSizeInMegabytes", "maxSizeInMegabytes")
	serviceBusApplyBoolAtomProperty(properties, body, "RequiresDuplicateDetection", "requiresDuplicateDetection")
	serviceBusApplyBoolAtomProperty(properties, body, "RequiresSession", "requiresSession")
	serviceBusApplyBoolAtomProperty(properties, body, "DeadLetteringOnMessageExpiration", "deadLetteringOnMessageExpiration")
	serviceBusApplyBoolAtomProperty(properties, body, "EnableBatchedOperations", "enableBatchedOperations")
	serviceBusApplyStringAtomProperty(properties, body, "AutoDeleteOnIdle", "autoDeleteOnIdle")
	serviceBusApplyStringAtomProperty(properties, body, "UserMetadata", "userMetadata")
	return properties
}

func serviceBusTopicPropertiesFromAtom(body []byte) map[string]any {
	properties := map[string]any{
		"maxSizeInMegabytes":      1024,
		"enableBatchedOperations": true,
	}
	serviceBusApplyStringAtomProperty(properties, body, "DefaultMessageTimeToLive", "defaultMessageTimeToLive")
	serviceBusApplyIntAtomProperty(properties, body, "MaxSizeInMegabytes", "maxSizeInMegabytes")
	serviceBusApplyBoolAtomProperty(properties, body, "RequiresDuplicateDetection", "requiresDuplicateDetection")
	serviceBusApplyBoolAtomProperty(properties, body, "EnableBatchedOperations", "enableBatchedOperations")
	serviceBusApplyBoolAtomProperty(properties, body, "SupportOrdering", "supportOrdering")
	serviceBusApplyStringAtomProperty(properties, body, "AutoDeleteOnIdle", "autoDeleteOnIdle")
	serviceBusApplyStringAtomProperty(properties, body, "UserMetadata", "userMetadata")
	return properties
}

func serviceBusSubscriptionPropertiesFromAtom(body []byte) map[string]any {
	properties := map[string]any{
		"lockDuration":                              "PT30S",
		"maxDeliveryCount":                          10,
		"requiresSession":                           false,
		"enableBatchedOperations":                   true,
		"deadLetteringOnFilterEvaluationExceptions": true,
	}
	serviceBusApplyStringAtomProperty(properties, body, "LockDuration", "lockDuration")
	serviceBusApplyIntAtomProperty(properties, body, "MaxDeliveryCount", "maxDeliveryCount")
	serviceBusApplyBoolAtomProperty(properties, body, "RequiresSession", "requiresSession")
	serviceBusApplyStringAtomProperty(properties, body, "DefaultMessageTimeToLive", "defaultMessageTimeToLive")
	serviceBusApplyBoolAtomProperty(properties, body, "DeadLetteringOnMessageExpiration", "deadLetteringOnMessageExpiration")
	serviceBusApplyBoolAtomProperty(properties, body, "DeadLetteringOnFilterEvaluationExceptions", "deadLetteringOnFilterEvaluationExceptions")
	serviceBusApplyBoolAtomProperty(properties, body, "EnableBatchedOperations", "enableBatchedOperations")
	serviceBusApplyStringAtomProperty(properties, body, "AutoDeleteOnIdle", "autoDeleteOnIdle")
	serviceBusApplyStringAtomProperty(properties, body, "UserMetadata", "userMetadata")
	return properties
}

func serviceBusApplyStringAtomProperty(properties map[string]any, body []byte, xmlTag, propertyName string) {
	if value := serviceBusXMLTagValue(string(body), xmlTag); strings.TrimSpace(value) != "" {
		properties[propertyName] = value
	}
}

func serviceBusApplyIntAtomProperty(properties map[string]any, body []byte, xmlTag, propertyName string) {
	value := strings.TrimSpace(serviceBusXMLTagValue(string(body), xmlTag))
	if value == "" {
		return
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		properties[propertyName] = parsed
	}
}

func serviceBusApplyBoolAtomProperty(properties map[string]any, body []byte, xmlTag, propertyName string) {
	value := strings.TrimSpace(serviceBusXMLTagValue(string(body), xmlTag))
	switch strings.ToLower(value) {
	case "true":
		properties[propertyName] = true
	case "false":
		properties[propertyName] = false
	}
}

func serviceBusXMLTagValue(body, tag string) string {
	return serviceBusXMLUnescape(serviceBusXMLBlock(body, tag))
}

func serviceBusXMLStartTagHasType(body, tag, typeName string) bool {
	start := strings.Index(body, "<"+tag)
	if start < 0 {
		return false
	}
	openEnd := strings.Index(body[start:], ">")
	if openEnd < 0 {
		return false
	}
	openingTag := body[start : start+openEnd+1]
	return strings.Contains(openingTag, `type="`+typeName+`"`) ||
		strings.Contains(openingTag, `type='`+typeName+`'`)
}

func serviceBusXMLBlock(body, tag string) string {
	start := strings.Index(body, "<"+tag)
	if start < 0 {
		return ""
	}
	tagEnd := start + len(tag) + 1
	if tagEnd < len(body) {
		next := body[tagEnd]
		if next != '>' && next != ' ' && next != '\t' && next != '\n' && next != '\r' && next != '/' {
			return ""
		}
	}
	openEnd := strings.Index(body[start:], ">")
	if openEnd < 0 {
		return ""
	}
	openEnd += start
	if openEnd > start && body[openEnd-1] == '/' {
		return ""
	}
	closeTag := "</" + tag + ">"
	closeStart := strings.Index(body[openEnd+1:], closeTag)
	if closeStart < 0 {
		return ""
	}
	return body[openEnd+1 : openEnd+1+closeStart]
}

func serviceBusXMLBlocks(body, tag string) []string {
	blocks := []string{}
	remaining := body
	for {
		block, rest, ok := serviceBusXMLBlockWithRest(remaining, tag)
		if !ok {
			return blocks
		}
		blocks = append(blocks, block)
		remaining = rest
	}
}

func serviceBusXMLBlockWithRest(body, tag string) (string, string, bool) {
	start := strings.Index(body, "<"+tag)
	if start < 0 {
		return "", "", false
	}
	tagEnd := start + len(tag) + 1
	if tagEnd < len(body) {
		next := body[tagEnd]
		if next != '>' && next != ' ' && next != '\t' && next != '\n' && next != '\r' && next != '/' {
			return "", "", false
		}
	}
	openEnd := strings.Index(body[start:], ">")
	if openEnd < 0 {
		return "", "", false
	}
	openEnd += start
	closeTag := "</" + tag + ">"
	closeStart := strings.Index(body[openEnd+1:], closeTag)
	if closeStart < 0 {
		return "", "", false
	}
	blockEnd := openEnd + 1 + closeStart + len(closeTag)
	return body[openEnd+1 : openEnd+1+closeStart], body[blockEnd:], true
}

func serviceBusAtomTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func serviceBusLeafName(name string) string {
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		return name[slash+1:]
	}
	return name
}

func serviceBusStringProperty(props map[string]any, name, fallback string) string {
	if value, ok := props[name].(string); ok && value != "" {
		return serviceBusXMLEscape(value)
	}
	return fallback
}

func serviceBusIntProperty(props map[string]any, name string, fallback int) string {
	switch value := props[name].(type) {
	case int:
		return fmt.Sprintf("%d", value)
	case int64:
		return fmt.Sprintf("%d", value)
	case float64:
		return fmt.Sprintf("%.0f", value)
	case gojson.Number:
		return string(value)
	default:
		return fmt.Sprintf("%d", fallback)
	}
}

func serviceBusBoolProperty(props map[string]any, name string, fallback bool) string {
	if value, ok := props[name].(bool); ok {
		if value {
			return "true"
		}
		return "false"
	}
	if fallback {
		return "true"
	}
	return "false"
}

func serviceBusOptionalStringElement(props map[string]any, name, tag string) string {
	value, ok := props[name].(string)
	if !ok || value == "" {
		return ""
	}
	return "<" + tag + ">" + serviceBusXMLEscape(value) + "</" + tag + ">"
}

func serviceBusXMLEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}

func serviceBusXMLUnescape(value string) string {
	replacer := strings.NewReplacer(
		"&apos;", "'",
		"&quot;", `"`,
		"&gt;", ">",
		"&lt;", "<",
		"&amp;", "&",
	)
	return replacer.Replace(value)
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

func runtimeQueueKey(namespaceName, queueName string) string {
	return strings.ToLower(namespaceName) + "/" + strings.ToLower(queueName)
}

func runtimeTopicKey(namespaceName, topicName string) string {
	return strings.ToLower(namespaceName) + "/" + strings.ToLower(topicName)
}

func runtimeSubscriptionKey(namespaceName, topicName, subscriptionName string) string {
	return runtimeTopicKey(namespaceName, topicName) + "/" + strings.ToLower(subscriptionName)
}

func parseRuntimeTarget(parts []string) runtimeTarget {
	deadLetter := false
	if len(parts) > 0 && strings.EqualFold(parts[len(parts)-1], "$deadletterqueue") {
		deadLetter = true
		parts = parts[:len(parts)-1]
	}
	if len(parts) >= 3 && strings.EqualFold(parts[len(parts)-2], "subscriptions") {
		return runtimeTarget{
			TopicName:        strings.Join(parts[:len(parts)-2], "/"),
			SubscriptionName: parts[len(parts)-1],
			DeadLetter:       deadLetter,
		}
	}
	return runtimeTarget{QueueName: strings.Join(parts, "/"), DeadLetter: deadLetter}
}

func (s *ServiceBusService) newRuntimeMessageLocked(body []byte, properties runtimeMessageProperties) runtimeMessage {
	s.runtimeSeq++
	return runtimeMessage{
		ID:               fmt.Sprintf("msg-%016x", s.runtimeSeq),
		Sequence:         s.runtimeSeq,
		Body:             append([]byte(nil), body...),
		ContentType:      properties.ContentType,
		UserProperties:   cloneStringMap(properties.User),
		BrokerProperties: cloneAnyMap(properties.Broker),
		EnqueuedAt:       time.Now().UTC(),
		DeliveryCount:    1,
	}
}

func (s *ServiceBusService) runtimeSubscriptionAcceptsMessageLocked(namespaceName, topicName, subscriptionName string, properties runtimeMessageProperties) bool {
	rules := s.rulesForRuntimeSubscriptionLocked(namespaceName, topicName, subscriptionName)
	if len(rules) == 0 {
		return true
	}
	for _, rule := range rules {
		if serviceBusRuleMatches(rule, properties) {
			return true
		}
	}
	return false
}

func (s *ServiceBusService) rulesForRuntimeSubscriptionLocked(namespaceName, topicName, subscriptionName string) []Rule {
	subscriptionResourceName := namespaceName + "/" + topicName + "/" + subscriptionName
	rules := make([]Rule, 0)
	for key, subscription := range s.subscriptions {
		if !strings.EqualFold(subscription.Name, subscriptionResourceName) {
			continue
		}
		prefix := key + "/"
		for ruleKey, rule := range s.rules {
			if strings.HasPrefix(ruleKey, prefix) {
				rules = append(rules, rule)
			}
		}
	}
	return rules
}

func (s *ServiceBusService) runtimeMessagesForTargetLocked(namespaceName string, target runtimeTarget) ([]runtimeMessage, bool) {
	if target.SubscriptionName != "" {
		if target.DeadLetter {
			messages, ok := s.runtimeSubDLQs[runtimeSubscriptionKey(namespaceName, target.TopicName, target.SubscriptionName)]
			return messages, ok
		}
		messages, ok := s.runtimeSubscriptions[runtimeSubscriptionKey(namespaceName, target.TopicName, target.SubscriptionName)]
		return messages, ok
	}
	if target.DeadLetter {
		messages, ok := s.runtimeQueueDLQs[runtimeQueueKey(namespaceName, target.QueueName)]
		return messages, ok
	}
	messages, ok := s.runtimeQueues[runtimeQueueKey(namespaceName, target.QueueName)]
	return messages, ok
}

func (s *ServiceBusService) setRuntimeMessagesForTargetLocked(namespaceName string, target runtimeTarget, messages []runtimeMessage) {
	if target.SubscriptionName != "" {
		if target.DeadLetter {
			s.runtimeSubDLQs[runtimeSubscriptionKey(namespaceName, target.TopicName, target.SubscriptionName)] = messages
			return
		}
		s.runtimeSubscriptions[runtimeSubscriptionKey(namespaceName, target.TopicName, target.SubscriptionName)] = messages
		return
	}
	if target.DeadLetter {
		s.runtimeQueueDLQs[runtimeQueueKey(namespaceName, target.QueueName)] = messages
		return
	}
	s.runtimeQueues[runtimeQueueKey(namespaceName, target.QueueName)] = messages
}

func (s *ServiceBusService) expireRuntimeMessagesForTargetLocked(namespaceName string, target runtimeTarget, messages []runtimeMessage, now time.Time) []runtimeMessage {
	if target.DeadLetter {
		return messages
	}
	remaining := make([]runtimeMessage, 0, len(messages))
	changed := false
	for _, msg := range messages {
		if msg.Locked || !serviceBusRuntimeMessageExpired(msg, now) {
			remaining = append(remaining, msg)
			continue
		}
		changed = true
		if s.deadLettersExpiredRuntimeMessagesLocked(namespaceName, target) {
			msg.Locked = false
			msg.LockToken = ""
			msg.DeadLetterReason = "TTLExpiredException"
			s.appendRuntimeDeadLetterLocked(namespaceName, target, msg)
		}
	}
	if changed {
		s.setRuntimeMessagesForTargetLocked(namespaceName, target, remaining)
		return remaining
	}
	return messages
}

func (s *ServiceBusService) expireRuntimeLocksForTargetLocked(namespaceName string, target runtimeTarget, messages []runtimeMessage, now time.Time) []runtimeMessage {
	remaining := make([]runtimeMessage, 0, len(messages))
	changed := false
	for _, msg := range messages {
		if !serviceBusRuntimeLockExpired(msg, now) {
			remaining = append(remaining, msg)
			continue
		}
		changed = true
		msg.Locked = false
		msg.LockToken = ""
		msg.LockedUntil = time.Time{}
		if !target.DeadLetter {
			if msg.DeliveryCount == 0 {
				msg.DeliveryCount = 1
			}
			msg.DeliveryCount++
			if msg.DeliveryCount > s.maxDeliveryCountForRuntimeTargetLocked(namespaceName, target) {
				msg.DeadLetterReason = "MaxDeliveryCountExceeded"
				s.appendRuntimeDeadLetterLocked(namespaceName, target, msg)
				continue
			}
		}
		remaining = append(remaining, msg)
	}
	if changed {
		s.setRuntimeMessagesForTargetLocked(namespaceName, target, remaining)
		return remaining
	}
	return messages
}

func serviceBusRuntimeLockExpired(msg runtimeMessage, now time.Time) bool {
	return msg.Locked && !msg.LockedUntil.IsZero() && !msg.LockedUntil.After(now)
}

func serviceBusRuntimeMessageExpired(msg runtimeMessage, now time.Time) bool {
	ttlSeconds, ok := serviceBusRuntimeMessageTTLSeconds(msg.BrokerProperties)
	if !ok {
		return false
	}
	if ttlSeconds <= 0 {
		return true
	}
	enqueuedAt := msg.EnqueuedAt
	if enqueuedAt.IsZero() {
		return false
	}
	return !enqueuedAt.Add(time.Duration(ttlSeconds * float64(time.Second))).After(now)
}

func serviceBusRuntimeMessageTTLSeconds(properties map[string]any) (float64, bool) {
	switch value := properties["TimeToLive"].(type) {
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case float64:
		return value, true
	case gojson.Number:
		parsed, err := strconv.ParseFloat(string(value), 64)
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(value, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func (s *ServiceBusService) appendRuntimeDeadLetterLocked(namespaceName string, target runtimeTarget, msg runtimeMessage) {
	deadLetterTarget := target
	deadLetterTarget.DeadLetter = true
	messages, _ := s.runtimeMessagesForTargetLocked(namespaceName, deadLetterTarget)
	messages = append(messages, msg)
	s.setRuntimeMessagesForTargetLocked(namespaceName, deadLetterTarget, messages)
}

func (s *ServiceBusService) maxDeliveryCountForRuntimeTargetLocked(namespaceName string, target runtimeTarget) int {
	if target.SubscriptionName != "" {
		resourceName := namespaceName + "/" + target.TopicName + "/" + target.SubscriptionName
		for _, subscription := range s.subscriptions {
			if strings.EqualFold(subscription.Name, resourceName) {
				return serviceBusMaxDeliveryCount(subscription.Properties)
			}
		}
		return 10
	}
	resourceName := namespaceName + "/" + target.QueueName
	for _, queue := range s.queues {
		if strings.EqualFold(queue.Name, resourceName) {
			return serviceBusMaxDeliveryCount(queue.Properties)
		}
	}
	return 10
}

func (s *ServiceBusService) lockDurationForRuntimeTargetLocked(namespaceName string, target runtimeTarget) time.Duration {
	if target.SubscriptionName != "" {
		resourceName := namespaceName + "/" + target.TopicName + "/" + target.SubscriptionName
		for _, subscription := range s.subscriptions {
			if strings.EqualFold(subscription.Name, resourceName) {
				return serviceBusDurationValue(subscription.Properties["lockDuration"], time.Minute)
			}
		}
		return time.Minute
	}
	resourceName := namespaceName + "/" + target.QueueName
	for _, queue := range s.queues {
		if strings.EqualFold(queue.Name, resourceName) {
			return serviceBusDurationValue(queue.Properties["lockDuration"], time.Minute)
		}
	}
	return time.Minute
}

func (s *ServiceBusService) deadLettersExpiredRuntimeMessagesLocked(namespaceName string, target runtimeTarget) bool {
	if target.SubscriptionName != "" {
		resourceName := namespaceName + "/" + target.TopicName + "/" + target.SubscriptionName
		for _, subscription := range s.subscriptions {
			if strings.EqualFold(subscription.Name, resourceName) {
				return serviceBusBoolValue(subscription.Properties["deadLetteringOnMessageExpiration"])
			}
		}
		return false
	}
	resourceName := namespaceName + "/" + target.QueueName
	for _, queue := range s.queues {
		if strings.EqualFold(queue.Name, resourceName) {
			return serviceBusBoolValue(queue.Properties["deadLetteringOnMessageExpiration"])
		}
	}
	return false
}

func serviceBusDurationValue(value any, fallback time.Duration) time.Duration {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	parsed, ok := serviceBusParseISODuration(text)
	if !ok {
		return fallback
	}
	return parsed
}

func serviceBusParseISODuration(value string) (time.Duration, bool) {
	text := strings.ToUpper(strings.TrimSpace(value))
	if !strings.HasPrefix(text, "P") {
		return 0, false
	}
	text = strings.TrimPrefix(text, "P")
	inTime := false
	var total time.Duration
	var number strings.Builder
	for _, r := range text {
		if r >= '0' && r <= '9' {
			number.WriteRune(r)
			continue
		}
		if r == 'T' {
			inTime = true
			continue
		}
		if number.Len() == 0 {
			return 0, false
		}
		amount, err := strconv.Atoi(number.String())
		if err != nil {
			return 0, false
		}
		number.Reset()
		switch r {
		case 'D':
			if inTime {
				return 0, false
			}
			total += time.Duration(amount) * 24 * time.Hour
		case 'H':
			if !inTime {
				return 0, false
			}
			total += time.Duration(amount) * time.Hour
		case 'M':
			if !inTime {
				return 0, false
			}
			total += time.Duration(amount) * time.Minute
		case 'S':
			if !inTime {
				return 0, false
			}
			total += time.Duration(amount) * time.Second
		default:
			return 0, false
		}
	}
	if number.Len() != 0 {
		return 0, false
	}
	return total, true
}

func serviceBusBoolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	default:
		return false
	}
}

func serviceBusMaxDeliveryCount(properties map[string]any) int {
	switch value := properties["maxDeliveryCount"].(type) {
	case int:
		if value > 0 {
			return value
		}
	case int64:
		if value > 0 {
			return int(value)
		}
	case float64:
		if value > 0 {
			return int(value)
		}
	case gojson.Number:
		if parsed, err := strconv.Atoi(string(value)); err == nil && parsed > 0 {
			return parsed
		}
	case string:
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 10
}

func serviceBusSendResponse(req *http.Request) *service.Response {
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/xml; charset=utf-8"
	}
	return &service.Response{
		StatusCode:     http.StatusCreated,
		RawContentType: contentType,
		Headers:        map[string]string{"Content-Type": contentType},
	}
}

func serviceBusEmptyRuntimeSuccessResponse() *service.Response {
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawContentType: "application/xml; charset=utf-8",
		Headers:        map[string]string{"Content-Type": "application/xml; charset=utf-8"},
	}
}

func serviceBusMessageResponse(status int, namespaceName, entityPath string, msg runtimeMessage, locked bool) (*service.Response, error) {
	contentType := msg.ContentType
	if contentType == "" {
		contentType = "application/atom+xml;type=entry;charset=utf-8"
	}
	broker := cloneAnyMap(msg.BrokerProperties)
	if broker == nil {
		broker = map[string]any{}
	}
	if _, ok := broker["MessageId"]; !ok {
		broker["MessageId"] = msg.ID
	}
	deliveryCount := msg.DeliveryCount
	if deliveryCount == 0 {
		deliveryCount = 1
	}
	responseBrokerProperties := map[string]any{
		"DeliveryCount":          deliveryCount,
		"EnqueuedSequenceNumber": 0,
		"EnqueuedTimeUtc":        "Wed, 02 Jul 2014 01:32:27 GMT",
		"SequenceNumber":         msg.Sequence,
	}
	for key, value := range responseBrokerProperties {
		broker[key] = value
	}
	defaultBrokerProperties := map[string]any{
		"State":      "Active",
		"TimeToLive": 922337203685,
	}
	for key, value := range defaultBrokerProperties {
		if _, ok := broker[key]; !ok {
			broker[key] = value
		}
	}
	if msg.DeadLetterReason != "" {
		broker["DeadLetterReason"] = msg.DeadLetterReason
	}
	headers := map[string]string{"Content-Type": contentType}
	for key, value := range msg.UserProperties {
		headers[key] = value
	}
	if locked {
		broker["LockToken"] = msg.LockToken
		if msg.LockedUntil.IsZero() {
			broker["LockedUntilUtc"] = "Wed, 02 Jul 2014 01:33:27 GMT"
		} else {
			broker["LockedUntilUtc"] = msg.LockedUntil.UTC().Format(http.TimeFormat)
		}
		headers["Location"] = "https://" + namespaceName + ".servicebus.windows.net/" + entityPath + "/messages/" + msg.ID + "/" + msg.LockToken
	}
	data, err := gojson.Marshal(broker)
	if err != nil {
		return nil, err
	}
	headers["BrokerProperties"] = string(data)
	return &service.Response{
		StatusCode:     status,
		RawBody:        append([]byte(nil), msg.Body...),
		RawContentType: contentType,
		Headers:        headers,
	}, nil
}

func serviceBusRuntimeError(status int, message string) (*service.Response, error) {
	return azurearm.ErrorResponse(status, "ServiceBusRuntimeError", message)
}

func runtimeOutboundMessagesFromRequest(req *http.Request, body []byte) ([]runtimeOutboundMessage, error) {
	if !isServiceBusBatchContentType(req) {
		return []runtimeOutboundMessage{{
			Body:       append([]byte(nil), body...),
			Properties: runtimeMessagePropertiesFromRequest(req),
		}}, nil
	}

	var batch []struct {
		Body             any            `json:"Body"`
		BrokerProperties map[string]any `json:"BrokerProperties"`
		UserProperties   map[string]any `json:"UserProperties"`
	}
	if err := gojson.Unmarshal(body, &batch); err != nil {
		return nil, err
	}
	out := make([]runtimeOutboundMessage, 0, len(batch))
	for _, item := range batch {
		messageBody, err := serviceBusBatchMessageBody(item.Body)
		if err != nil {
			return nil, err
		}
		out = append(out, runtimeOutboundMessage{
			Body: messageBody,
			Properties: runtimeMessageProperties{
				User:   stringifyAnyMap(item.UserProperties),
				Broker: cloneAnyMap(item.BrokerProperties),
			},
		})
	}
	return out, nil
}

func isServiceBusBatchContentType(req *http.Request) bool {
	if req == nil {
		return false
	}
	contentType := strings.ToLower(req.Header.Get("Content-Type"))
	if semicolon := strings.IndexByte(contentType, ';'); semicolon >= 0 {
		contentType = contentType[:semicolon]
	}
	return strings.TrimSpace(contentType) == "application/vnd.microsoft.servicebus.json"
}

func serviceBusBatchMessageBody(value any) ([]byte, error) {
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

func runtimeMessagePropertiesFromRequest(req *http.Request) runtimeMessageProperties {
	properties := runtimeMessageProperties{
		User:   map[string]string{},
		Broker: map[string]any{},
	}
	if req == nil {
		return properties
	}
	properties.ContentType = strings.TrimSpace(req.Header.Get("Content-Type"))
	for key, values := range req.Header {
		if len(values) == 0 || isServiceBusReservedMessageHeader(key) {
			continue
		}
		properties.User[key] = values[0]
	}
	if rawBroker := req.Header.Get("BrokerProperties"); rawBroker != "" {
		var broker map[string]any
		if err := gojson.Unmarshal([]byte(rawBroker), &broker); err == nil {
			properties.Broker = broker
		}
	}
	return properties
}

func isServiceBusReservedMessageHeader(header string) bool {
	switch strings.ToLower(header) {
	case "accept", "authorization", "brokerproperties", "connection", "content-length", "content-type", "date", "expect", "host", "user-agent", "x-ms-retrypolicy":
		return true
	default:
		return strings.HasPrefix(strings.ToLower(header), "x-ms-")
	}
}

func serviceBusRuleMatches(rule Rule, properties runtimeMessageProperties) bool {
	filterType := stringValue(rule.Properties["filterType"])
	switch {
	case filterType == "", strings.EqualFold(filterType, "SqlFilter"):
		sqlFilter := nestedAnyMap(rule.Properties["sqlFilter"])
		return serviceBusSQLFilterMatches(stringValue(sqlFilter["sqlExpression"]), properties)
	case strings.EqualFold(filterType, "CorrelationFilter"):
		return serviceBusCorrelationFilterMatches(nestedAnyMap(rule.Properties["correlationFilter"]), properties)
	default:
		return false
	}
}

func serviceBusSQLFilterMatches(expression string, properties runtimeMessageProperties) bool {
	expression = strings.TrimSpace(expression)
	compact := strings.ReplaceAll(expression, " ", "")
	switch {
	case expression == "", strings.EqualFold(compact, "1=1"), strings.EqualFold(expression, "true"):
		return true
	case strings.EqualFold(compact, "1=0"), strings.EqualFold(expression, "false"):
		return false
	}

	clauses := splitServiceBusSQLAnd(expression)
	if len(clauses) == 0 {
		return false
	}
	for _, clause := range clauses {
		if !serviceBusSQLClauseMatches(clause, properties) {
			return false
		}
	}
	return true
}

func splitServiceBusSQLAnd(expression string) []string {
	parts := make([]string, 0, 1)
	remaining := expression
	for {
		lower := strings.ToLower(remaining)
		idx := strings.Index(lower, " and ")
		if idx < 0 {
			break
		}
		if part := strings.TrimSpace(remaining[:idx]); part != "" {
			parts = append(parts, part)
		}
		remaining = remaining[idx+5:]
	}
	if part := strings.TrimSpace(remaining); part != "" {
		parts = append(parts, part)
	}
	return parts
}

func serviceBusSQLClauseMatches(clause string, properties runtimeMessageProperties) bool {
	clause = strings.TrimSpace(clause)
	for strings.HasPrefix(clause, "(") && strings.HasSuffix(clause, ")") {
		clause = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(clause, "("), ")"))
	}
	compact := strings.ReplaceAll(clause, " ", "")
	switch {
	case strings.EqualFold(compact, "1=1"):
		return true
	case strings.EqualFold(compact, "1=0"):
		return false
	}

	idx := strings.Index(clause, "=")
	if idx < 0 {
		return false
	}
	left := strings.TrimSpace(clause[:idx])
	right := strings.TrimSpace(clause[idx+1:])
	expected := trimServiceBusSQLLiteral(right)
	if isServiceBusSQLLiteral(left) {
		return trimServiceBusSQLLiteral(left) == expected
	}

	propertyName, systemProperty := normalizeServiceBusSQLProperty(left)
	actual, ok := "", false
	if systemProperty {
		actual, ok = lookupAnyStringFold(properties.Broker, propertyName)
	} else {
		actual, ok = lookupStringFold(properties.User, propertyName)
		if !ok {
			actual, ok = lookupAnyStringFold(properties.Broker, propertyName)
		}
	}
	return ok && actual == expected
}

func normalizeServiceBusSQLProperty(property string) (string, bool) {
	property = strings.TrimSpace(property)
	property = strings.Trim(property, "[]`\"")
	systemProperty := false
	if strings.HasPrefix(strings.ToLower(property), "sys.") {
		systemProperty = true
		property = property[4:]
	}
	if strings.HasPrefix(strings.ToLower(property), "user.") {
		property = property[5:]
	}
	return strings.Trim(property, "[]`\""), systemProperty
}

func trimServiceBusSQLLiteral(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
			value = value[1 : len(value)-1]
		}
	}
	return strings.ReplaceAll(value, "''", "'")
}

func isServiceBusSQLLiteral(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if _, ok := normalizeServiceBusSQLNumber(value); ok {
		return true
	}
	return (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) || (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\""))
}

func normalizeServiceBusSQLNumber(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	for i, r := range value {
		if r >= '0' && r <= '9' {
			continue
		}
		if i == 0 && (r == '-' || r == '+') {
			continue
		}
		if r == '.' {
			continue
		}
		return "", false
	}
	return value, true
}

func serviceBusCorrelationFilterMatches(filter map[string]any, properties runtimeMessageProperties) bool {
	if len(filter) == 0 {
		return true
	}
	if expected := stringValue(filter["contentType"]); expected != "" && properties.ContentType != expected {
		return false
	}
	for filterName, brokerName := range map[string]string{
		"correlationId":    "CorrelationId",
		"label":            "Label",
		"messageId":        "MessageId",
		"replyTo":          "ReplyTo",
		"replyToSessionId": "ReplyToSessionId",
		"sessionId":        "SessionId",
		"subject":          "Label",
		"to":               "To",
	} {
		expected := stringValue(filter[filterName])
		if expected == "" {
			continue
		}
		actual, ok := lookupAnyStringFold(properties.Broker, brokerName)
		if !ok || actual != expected {
			return false
		}
	}
	for key, expectedValue := range nestedAnyMap(filter["properties"]) {
		expected := fmt.Sprint(expectedValue)
		actual, ok := lookupStringFold(properties.User, key)
		if !ok || actual != expected {
			return false
		}
	}
	return true
}

func nestedAnyMap(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return typed
}

func lookupStringFold(values map[string]string, key string) (string, bool) {
	for candidate, value := range values {
		if strings.EqualFold(candidate, key) {
			return value, true
		}
	}
	return "", false
}

func lookupAnyStringFold(values map[string]any, key string) (string, bool) {
	for candidate, value := range values {
		if strings.EqualFold(candidate, key) {
			return fmt.Sprint(value), true
		}
	}
	return "", false
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

func normalizeRuleProperties(input map[string]any) map[string]any {
	properties := cloneMap(input)
	if properties == nil {
		properties = map[string]any{}
	}

	filterType := stringValue(properties["filterType"])
	if filterType == "" {
		switch {
		case properties["correlationFilter"] != nil:
			filterType = "CorrelationFilter"
		default:
			filterType = "SqlFilter"
		}
		properties["filterType"] = filterType
	}

	if strings.EqualFold(filterType, "SqlFilter") {
		sqlFilter := cloneNestedMap(properties["sqlFilter"])
		if sqlFilter == nil {
			sqlFilter = map[string]any{}
		}
		if _, ok := sqlFilter["sqlExpression"]; !ok {
			sqlFilter["sqlExpression"] = "1=1"
		}
		if _, ok := sqlFilter["compatibilityLevel"]; !ok {
			sqlFilter["compatibilityLevel"] = 20
		}
		properties["sqlFilter"] = sqlFilter
	}

	action := cloneNestedMap(properties["action"])
	if action == nil {
		action = map[string]any{}
	}
	if _, hasExpression := action["sqlExpression"]; hasExpression {
		if _, ok := action["compatibilityLevel"]; !ok {
			action["compatibilityLevel"] = 20
		}
	}
	properties["action"] = action

	return properties
}

func cloneNestedMap(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return cloneMap(typed)
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneAnyMap(input map[string]any) map[string]any {
	return cloneMap(input)
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
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

func splitQuadNestedName(name string) (string, string, string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		return "", "", "", "", false
	}
	return parts[0], parts[1], parts[2], parts[3], true
}
