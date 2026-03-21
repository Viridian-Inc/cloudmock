package cloudwatch

import (
	"net/http"
	"net/url"

	"github.com/neureaux/cloudmock/pkg/service"
)

// CloudWatchService is the cloudmock implementation of the AWS CloudWatch Metrics API.
type CloudWatchService struct {
	store *Store
}

// New returns a new CloudWatchService for the given AWS account ID and region.
func New(accountID, region string) *CloudWatchService {
	return &CloudWatchService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
// CloudWatch Metrics uses "monitoring" in the credential scope.
func (s *CloudWatchService) Name() string { return "monitoring" }

// Actions returns the list of CloudWatch API actions supported by this service.
func (s *CloudWatchService) Actions() []service.Action {
	return []service.Action{
		{Name: "PutMetricData", Method: http.MethodPost, IAMAction: "cloudwatch:PutMetricData"},
		{Name: "GetMetricData", Method: http.MethodPost, IAMAction: "cloudwatch:GetMetricData"},
		{Name: "ListMetrics", Method: http.MethodPost, IAMAction: "cloudwatch:ListMetrics"},
		{Name: "PutMetricAlarm", Method: http.MethodPost, IAMAction: "cloudwatch:PutMetricAlarm"},
		{Name: "DescribeAlarms", Method: http.MethodPost, IAMAction: "cloudwatch:DescribeAlarms"},
		{Name: "DeleteAlarms", Method: http.MethodPost, IAMAction: "cloudwatch:DeleteAlarms"},
		{Name: "SetAlarmState", Method: http.MethodPost, IAMAction: "cloudwatch:SetAlarmState"},
		{Name: "DescribeAlarmsForMetric", Method: http.MethodPost, IAMAction: "cloudwatch:DescribeAlarmsForMetric"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "cloudwatch:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "cloudwatch:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "cloudwatch:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *CloudWatchService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CloudWatch request to the appropriate handler.
// CloudWatch uses form-encoded POST bodies; the Action is found in the form body.
func (s *CloudWatchService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "PutMetricData":
		return handlePutMetricData(ctx, s.store)
	case "GetMetricData":
		return handleGetMetricData(ctx, s.store)
	case "ListMetrics":
		return handleListMetrics(ctx, s.store)
	case "PutMetricAlarm":
		return handlePutMetricAlarm(ctx, s.store)
	case "DescribeAlarms":
		return handleDescribeAlarms(ctx, s.store)
	case "DeleteAlarms":
		return handleDeleteAlarms(ctx, s.store)
	case "SetAlarmState":
		return handleSetAlarmState(ctx, s.store)
	case "DescribeAlarmsForMetric":
		return handleDescribeAlarmsForMetric(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}

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
