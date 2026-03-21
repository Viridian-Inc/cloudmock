package eventbridge

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// EventBridgeService is the cloudmock implementation of the AWS EventBridge API.
type EventBridgeService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new EventBridgeService for the given AWS account ID and region.
func New(accountID, region string) *EventBridgeService {
	return &EventBridgeService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns an EventBridgeService that can deliver events to targets.
func NewWithLocator(accountID, region string, locator ServiceLocator) *EventBridgeService {
	return &EventBridgeService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service delivery.
func (s *EventBridgeService) SetLocator(locator ServiceLocator) {
	s.locator = locator
}

// Name returns the AWS service name used for routing.
// EventBridge uses "events" as the credential scope service name.
func (s *EventBridgeService) Name() string { return "events" }

// Actions returns the list of EventBridge API actions supported by this service.
func (s *EventBridgeService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateEventBus", Method: http.MethodPost, IAMAction: "events:CreateEventBus"},
		{Name: "DeleteEventBus", Method: http.MethodPost, IAMAction: "events:DeleteEventBus"},
		{Name: "DescribeEventBus", Method: http.MethodPost, IAMAction: "events:DescribeEventBus"},
		{Name: "ListEventBuses", Method: http.MethodPost, IAMAction: "events:ListEventBuses"},
		{Name: "PutRule", Method: http.MethodPost, IAMAction: "events:PutRule"},
		{Name: "DeleteRule", Method: http.MethodPost, IAMAction: "events:DeleteRule"},
		{Name: "DescribeRule", Method: http.MethodPost, IAMAction: "events:DescribeRule"},
		{Name: "ListRules", Method: http.MethodPost, IAMAction: "events:ListRules"},
		{Name: "PutTargets", Method: http.MethodPost, IAMAction: "events:PutTargets"},
		{Name: "RemoveTargets", Method: http.MethodPost, IAMAction: "events:RemoveTargets"},
		{Name: "ListTargetsByRule", Method: http.MethodPost, IAMAction: "events:ListTargetsByRule"},
		{Name: "PutEvents", Method: http.MethodPost, IAMAction: "events:PutEvents"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "events:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "events:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "events:ListTagsForResource"},
		{Name: "EnableRule", Method: http.MethodPost, IAMAction: "events:EnableRule"},
		{Name: "DisableRule", Method: http.MethodPost, IAMAction: "events:DisableRule"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *EventBridgeService) HealthCheck() error { return nil }

// HandleRequest routes an incoming EventBridge request to the appropriate handler.
// EventBridge uses the JSON protocol; the action is parsed from X-Amz-Target by the gateway
// (e.g. "AWSEvents.CreateEventBus" → "CreateEventBus").
func (s *EventBridgeService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateEventBus":
		return handleCreateEventBus(ctx, s.store)
	case "DeleteEventBus":
		return handleDeleteEventBus(ctx, s.store)
	case "DescribeEventBus":
		return handleDescribeEventBus(ctx, s.store)
	case "ListEventBuses":
		return handleListEventBuses(ctx, s.store)
	case "PutRule":
		return handlePutRule(ctx, s.store)
	case "DeleteRule":
		return handleDeleteRule(ctx, s.store)
	case "DescribeRule":
		return handleDescribeRule(ctx, s.store)
	case "ListRules":
		return handleListRules(ctx, s.store)
	case "PutTargets":
		return handlePutTargets(ctx, s.store)
	case "RemoveTargets":
		return handleRemoveTargets(ctx, s.store)
	case "ListTargetsByRule":
		return handleListTargetsByRule(ctx, s.store)
	case "PutEvents":
		return handlePutEvents(ctx, s.store, s.locator)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "EnableRule":
		return handleEnableRule(ctx, s.store)
	case "DisableRule":
		return handleDisableRule(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
