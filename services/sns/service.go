package sns

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// SNSService is the cloudmock implementation of the AWS Simple Notification Service API.
type SNSService struct {
	store *Store
}

// New returns a new SNSService for the given AWS account ID and region.
func New(accountID, region string) *SNSService {
	return &SNSService{
		store: NewStore(accountID, region),
	}
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
		return handlePublish(ctx, s.store)
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
