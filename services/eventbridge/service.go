package eventbridge

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/schema"
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
		{Name: "CreateArchive", Method: http.MethodPost, IAMAction: "events:CreateArchive"},
		{Name: "DescribeArchive", Method: http.MethodPost, IAMAction: "events:DescribeArchive"},
		{Name: "ListArchives", Method: http.MethodPost, IAMAction: "events:ListArchives"},
		{Name: "DeleteArchive", Method: http.MethodPost, IAMAction: "events:DeleteArchive"},
		{Name: "StartReplay", Method: http.MethodPost, IAMAction: "events:StartReplay"},
		{Name: "DescribeReplay", Method: http.MethodPost, IAMAction: "events:DescribeReplay"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *EventBridgeService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for EventBridge resource types.
func (s *EventBridgeService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "eventbridge",
			ResourceType:  "aws_cloudwatch_event_bus",
			TerraformType: "cloudmock_cloudwatch_event_bus",
			AWSType:       "AWS::Events::EventBus",
			CreateAction:  "CreateEventBus",
			ReadAction:    "DescribeEventBus",
			DeleteAction:  "DeleteEventBus",
			ListAction:    "ListEventBuses",
			ImportID:      "name",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "eventbridge",
			ResourceType:  "aws_cloudwatch_event_rule",
			TerraformType: "cloudmock_cloudwatch_event_rule",
			AWSType:       "AWS::Events::Rule",
			CreateAction:  "PutRule",
			ReadAction:    "DescribeRule",
			DeleteAction:  "DeleteRule",
			ListAction:    "ListRules",
			ImportID:      "name",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true, ForceNew: true},
				{Name: "event_bus_name", Type: "string", Default: "default"},
				{Name: "schedule_expression", Type: "string"},
				{Name: "event_pattern", Type: "string"},
				{Name: "description", Type: "string"},
				{Name: "is_enabled", Type: "bool", Default: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// RuleWithTargets pairs a rule with its targets for topology queries.
type RuleWithTargets struct {
	Rule    *Rule
	Targets []Target
}

// GetAllRulesWithTargets returns all rules across all event buses with their targets.
func (s *EventBridgeService) GetAllRulesWithTargets() []RuleWithTargets {
	buses := s.store.ListEventBuses()
	var result []RuleWithTargets
	for _, bus := range buses {
		rules, ok := s.store.ListRules(bus.Name, "")
		if !ok {
			continue
		}
		for _, rule := range rules {
			targets, _ := s.store.ListTargetsByRule(bus.Name, rule.Name)
			result = append(result, RuleWithTargets{Rule: rule, Targets: targets})
		}
	}
	return result
}

// GetAllEventBuses returns all event bus names for topology queries.
func (s *EventBridgeService) GetAllEventBuses() []string {
	buses := s.store.ListEventBuses()
	names := make([]string, 0, len(buses))
	for _, bus := range buses {
		names = append(names, bus.Name)
	}
	return names
}

// GetRuleTargetsSummary returns parallel slices of rule names and target ARNs for topology.
func (s *EventBridgeService) GetRuleTargetsSummary() (ruleNames, targetArns []string) {
	rwts := s.GetAllRulesWithTargets()
	for _, rwt := range rwts {
		for _, t := range rwt.Targets {
			ruleNames = append(ruleNames, rwt.Rule.Name)
			targetArns = append(targetArns, t.Arn)
		}
	}
	return ruleNames, targetArns
}

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
	case "CreateArchive":
		return handleCreateArchive(ctx, s.store)
	case "DescribeArchive":
		return handleDescribeArchive(ctx, s.store)
	case "ListArchives":
		return handleListArchives(ctx, s.store)
	case "DeleteArchive":
		return handleDeleteArchive(ctx, s.store)
	case "StartReplay":
		return handleStartReplay(ctx, s.store)
	case "DescribeReplay":
		return handleDescribeReplay(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
