package applicationautoscaling

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ApplicationAutoScalingService is the cloudmock implementation of the AWS Application Auto Scaling API.
type ApplicationAutoScalingService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ApplicationAutoScalingService for the given AWS account ID and region.
func New(accountID, region string) *ApplicationAutoScalingService {
	return &ApplicationAutoScalingService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *ApplicationAutoScalingService) Name() string { return "application-autoscaling" }

// Actions returns the list of Application Auto Scaling API actions supported by this service.
func (s *ApplicationAutoScalingService) Actions() []service.Action {
	return []service.Action{
		{Name: "RegisterScalableTarget", Method: http.MethodPost, IAMAction: "application-autoscaling:RegisterScalableTarget"},
		{Name: "DescribeScalableTargets", Method: http.MethodPost, IAMAction: "application-autoscaling:DescribeScalableTargets"},
		{Name: "DeregisterScalableTarget", Method: http.MethodPost, IAMAction: "application-autoscaling:DeregisterScalableTarget"},
		{Name: "PutScalingPolicy", Method: http.MethodPost, IAMAction: "application-autoscaling:PutScalingPolicy"},
		{Name: "DescribeScalingPolicies", Method: http.MethodPost, IAMAction: "application-autoscaling:DescribeScalingPolicies"},
		{Name: "DeleteScalingPolicy", Method: http.MethodPost, IAMAction: "application-autoscaling:DeleteScalingPolicy"},
		{Name: "PutScheduledAction", Method: http.MethodPost, IAMAction: "application-autoscaling:PutScheduledAction"},
		{Name: "DescribeScheduledActions", Method: http.MethodPost, IAMAction: "application-autoscaling:DescribeScheduledActions"},
		{Name: "DeleteScheduledAction", Method: http.MethodPost, IAMAction: "application-autoscaling:DeleteScheduledAction"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "application-autoscaling:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "application-autoscaling:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "application-autoscaling:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *ApplicationAutoScalingService) HealthCheck() error { return nil }

// HandleRequest routes an incoming request to the appropriate handler.
func (s *ApplicationAutoScalingService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "RegisterScalableTarget":
		return handleRegisterScalableTarget(ctx, s.store)
	case "DescribeScalableTargets":
		return handleDescribeScalableTargets(ctx, s.store)
	case "DeregisterScalableTarget":
		return handleDeregisterScalableTarget(ctx, s.store)
	case "PutScalingPolicy":
		return handlePutScalingPolicy(ctx, s.store)
	case "DescribeScalingPolicies":
		return handleDescribeScalingPolicies(ctx, s.store)
	case "DeleteScalingPolicy":
		return handleDeleteScalingPolicy(ctx, s.store)
	case "PutScheduledAction":
		return handlePutScheduledAction(ctx, s.store)
	case "DescribeScheduledActions":
		return handleDescribeScheduledActions(ctx, s.store)
	case "DeleteScheduledAction":
		return handleDeleteScheduledAction(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
