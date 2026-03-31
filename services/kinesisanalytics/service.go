package kinesisanalytics

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// KinesisAnalyticsService is the cloudmock implementation of the AWS Kinesis Analytics API.
type KinesisAnalyticsService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new KinesisAnalyticsService.
func New(accountID, region string) *KinesisAnalyticsService {
	return &KinesisAnalyticsService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *KinesisAnalyticsService) Name() string { return "kinesisanalytics" }

// Actions returns the list of API actions supported.
func (s *KinesisAnalyticsService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateApplication", Method: http.MethodPost, IAMAction: "kinesisanalytics:CreateApplication"},
		{Name: "DescribeApplication", Method: http.MethodPost, IAMAction: "kinesisanalytics:DescribeApplication"},
		{Name: "ListApplications", Method: http.MethodPost, IAMAction: "kinesisanalytics:ListApplications"},
		{Name: "DeleteApplication", Method: http.MethodPost, IAMAction: "kinesisanalytics:DeleteApplication"},
		{Name: "UpdateApplication", Method: http.MethodPost, IAMAction: "kinesisanalytics:UpdateApplication"},
		{Name: "StartApplication", Method: http.MethodPost, IAMAction: "kinesisanalytics:StartApplication"},
		{Name: "StopApplication", Method: http.MethodPost, IAMAction: "kinesisanalytics:StopApplication"},
		{Name: "AddApplicationInput", Method: http.MethodPost, IAMAction: "kinesisanalytics:AddApplicationInput"},
		{Name: "AddApplicationOutput", Method: http.MethodPost, IAMAction: "kinesisanalytics:AddApplicationOutput"},
		{Name: "DeleteApplicationOutput", Method: http.MethodPost, IAMAction: "kinesisanalytics:DeleteApplicationOutput"},
		{Name: "CreateApplicationSnapshot", Method: http.MethodPost, IAMAction: "kinesisanalytics:CreateApplicationSnapshot"},
		{Name: "ListApplicationSnapshots", Method: http.MethodPost, IAMAction: "kinesisanalytics:ListApplicationSnapshots"},
		{Name: "DeleteApplicationSnapshot", Method: http.MethodPost, IAMAction: "kinesisanalytics:DeleteApplicationSnapshot"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "kinesisanalytics:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "kinesisanalytics:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "kinesisanalytics:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *KinesisAnalyticsService) HealthCheck() error { return nil }

// HandleRequest routes an incoming request to the appropriate handler.
func (s *KinesisAnalyticsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateApplication":
		return handleCreateApplication(ctx, s.store)
	case "DescribeApplication":
		return handleDescribeApplication(ctx, s.store)
	case "ListApplications":
		return handleListApplications(ctx, s.store)
	case "DeleteApplication":
		return handleDeleteApplication(ctx, s.store)
	case "UpdateApplication":
		return handleUpdateApplication(ctx, s.store)
	case "StartApplication":
		return handleStartApplication(ctx, s.store)
	case "StopApplication":
		return handleStopApplication(ctx, s.store)
	case "AddApplicationInput":
		return handleAddApplicationInput(ctx, s.store)
	case "AddApplicationOutput":
		return handleAddApplicationOutput(ctx, s.store)
	case "DeleteApplicationOutput":
		return handleDeleteApplicationOutput(ctx, s.store)
	case "CreateApplicationSnapshot":
		return handleCreateApplicationSnapshot(ctx, s.store)
	case "ListApplicationSnapshots":
		return handleListApplicationSnapshots(ctx, s.store)
	case "DeleteApplicationSnapshot":
		return handleDeleteApplicationSnapshot(ctx, s.store)
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
