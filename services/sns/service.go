package sns

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// SNSService is the cloudmock implementation of the AWS Simple Notification Service API.
type SNSService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new SNSService for the given AWS account ID and region.
func New(accountID, region string) *SNSService {
	return &SNSService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns an SNSService that can deliver messages to other services.
func NewWithLocator(accountID, region string, locator ServiceLocator) *SNSService {
	return &SNSService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service delivery.
// This allows setting the locator after construction (needed when services
// and registry have a circular dependency).
func (s *SNSService) SetLocator(locator ServiceLocator) {
	s.locator = locator
}

// Name returns the AWS service name used for routing.
func (s *SNSService) Name() string { return "sns" }

// Actions returns the list of SNS API actions supported by this service.
func (s *SNSService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateTopic", Method: http.MethodPost, IAMAction: "sns:CreateTopic"},
		{Name: "DeleteTopic", Method: http.MethodPost, IAMAction: "sns:DeleteTopic"},
		{Name: "ListTopics", Method: http.MethodPost, IAMAction: "sns:ListTopics"},
		{Name: "GetTopicAttributes", Method: http.MethodPost, IAMAction: "sns:GetTopicAttributes"},
		{Name: "SetTopicAttributes", Method: http.MethodPost, IAMAction: "sns:SetTopicAttributes"},
		{Name: "Subscribe", Method: http.MethodPost, IAMAction: "sns:Subscribe"},
		{Name: "Unsubscribe", Method: http.MethodPost, IAMAction: "sns:Unsubscribe"},
		{Name: "ListSubscriptions", Method: http.MethodPost, IAMAction: "sns:ListSubscriptions"},
		{Name: "ListSubscriptionsByTopic", Method: http.MethodPost, IAMAction: "sns:ListSubscriptionsByTopic"},
		{Name: "Publish", Method: http.MethodPost, IAMAction: "sns:Publish"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "sns:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "sns:UntagResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *SNSService) HealthCheck() error { return nil }

// PublishDirect publishes a message to a topic by name (not ARN) without going
// through the HTTP/form-parsing path. Used for cross-service delivery
// (e.g., EventBridge → SNS).
func (s *SNSService) PublishDirect(topicName, message, subject string) bool {
	topicArn := s.store.topicARN(topicName)
	msgID, ok := s.store.Publish(topicArn, message, subject, nil)
	if !ok {
		return false
	}

	// Also deliver to SQS subscriptions if we have a locator.
	if s.locator != nil {
		deliverToSQSSubscriptions(s.store, s.locator, topicArn, msgID, message, subject)
	}
	return true
}

// HandleRequest routes an incoming SNS request to the appropriate handler.
// SNS uses form-encoded POST bodies; the Action may appear in the query string
// (already parsed into ctx.Params) or in the form-encoded body.
func (s *SNSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "CreateTopic":
		return handleCreateTopic(ctx, s.store)
	case "DeleteTopic":
		return handleDeleteTopic(ctx, s.store)
	case "ListTopics":
		return handleListTopics(ctx, s.store)
	case "GetTopicAttributes":
		return handleGetTopicAttributes(ctx, s.store)
	case "SetTopicAttributes":
		return handleSetTopicAttributes(ctx, s.store)
	case "Subscribe":
		return handleSubscribe(ctx, s.store)
	case "Unsubscribe":
		return handleUnsubscribe(ctx, s.store)
	case "ListSubscriptions":
		return handleListSubscriptions(ctx, s.store)
	case "ListSubscriptionsByTopic":
		return handleListSubscriptionsByTopic(ctx, s.store)
	case "Publish":
		return handlePublish(ctx, s.store, s.locator)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
