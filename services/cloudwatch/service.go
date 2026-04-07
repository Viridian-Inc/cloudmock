package cloudwatch

import (
	"net/http"
	"net/url"

	"github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// CloudWatchService is the cloudmock implementation of the AWS CloudWatch Metrics API.
type CloudWatchService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new CloudWatchService for the given AWS account ID and region.
func New(accountID, region string) *CloudWatchService {
	return &CloudWatchService{
		store: NewStore(accountID, region),
	}
}

// SetLocator sets the service locator for cross-service delivery (alarm → SNS).
func (s *CloudWatchService) SetLocator(locator ServiceLocator) {
	s.locator = locator
}

// GetAllAlarms returns all alarms for topology queries.
func (s *CloudWatchService) GetAllAlarms() []*Alarm {
	return s.store.DescribeAlarms(nil)
}

// GetAlarmActionsSummary returns parallel slices of alarm names and action ARNs for topology.
func (s *CloudWatchService) GetAlarmActionsSummary() (alarmNames, actionArns []string) {
	alarms := s.store.DescribeAlarms(nil)
	for _, a := range alarms {
		for _, action := range a.AlarmActions {
			alarmNames = append(alarmNames, a.AlarmName)
			actionArns = append(actionArns, action)
		}
	}
	return alarmNames, actionArns
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

// ResourceSchemas returns the schema for CloudWatch resource types.
func (s *CloudWatchService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "cloudwatch",
			ResourceType:  "aws_cloudwatch_metric_alarm",
			TerraformType: "cloudmock_cloudwatch_metric_alarm",
			AWSType:       "AWS::CloudWatch::Alarm",
			CreateAction:  "PutMetricAlarm",
			ReadAction:    "DescribeAlarms",
			DeleteAction:  "DeleteAlarms",
			ImportID:      "alarm_name",
			Attributes: []schema.AttributeSchema{
				{Name: "alarm_name", Type: "string", Required: true, ForceNew: true},
				{Name: "comparison_operator", Type: "string", Required: true},
				{Name: "evaluation_periods", Type: "int", Required: true},
				{Name: "metric_name", Type: "string"},
				{Name: "namespace", Type: "string"},
				{Name: "period", Type: "int"},
				{Name: "statistic", Type: "string"},
				{Name: "threshold", Type: "float"},
				{Name: "alarm_description", Type: "string"},
				{Name: "alarm_actions", Type: "list"},
				{Name: "ok_actions", Type: "list"},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

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
		return handleSetAlarmState(ctx, s.store, s.locator)
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
