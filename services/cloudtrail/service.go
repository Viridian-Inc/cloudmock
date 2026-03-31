package cloudtrail

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// CloudTrailService is the cloudmock implementation of the AWS CloudTrail API.
type CloudTrailService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new CloudTrailService for the given AWS account ID and region.
func New(accountID, region string) *CloudTrailService {
	return &CloudTrailService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *CloudTrailService) Name() string { return "cloudtrail" }

// Actions returns the list of CloudTrail API actions supported by this service.
func (s *CloudTrailService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateTrail", Method: http.MethodPost, IAMAction: "cloudtrail:CreateTrail"},
		{Name: "GetTrail", Method: http.MethodPost, IAMAction: "cloudtrail:GetTrail"},
		{Name: "DescribeTrails", Method: http.MethodPost, IAMAction: "cloudtrail:DescribeTrails"},
		{Name: "DeleteTrail", Method: http.MethodPost, IAMAction: "cloudtrail:DeleteTrail"},
		{Name: "UpdateTrail", Method: http.MethodPost, IAMAction: "cloudtrail:UpdateTrail"},
		{Name: "StartLogging", Method: http.MethodPost, IAMAction: "cloudtrail:StartLogging"},
		{Name: "StopLogging", Method: http.MethodPost, IAMAction: "cloudtrail:StopLogging"},
		{Name: "GetTrailStatus", Method: http.MethodPost, IAMAction: "cloudtrail:GetTrailStatus"},
		{Name: "PutEventSelectors", Method: http.MethodPost, IAMAction: "cloudtrail:PutEventSelectors"},
		{Name: "GetEventSelectors", Method: http.MethodPost, IAMAction: "cloudtrail:GetEventSelectors"},
		{Name: "PutInsightSelectors", Method: http.MethodPost, IAMAction: "cloudtrail:PutInsightSelectors"},
		{Name: "GetInsightSelectors", Method: http.MethodPost, IAMAction: "cloudtrail:GetInsightSelectors"},
		{Name: "LookupEvents", Method: http.MethodPost, IAMAction: "cloudtrail:LookupEvents"},
		{Name: "AddTags", Method: http.MethodPost, IAMAction: "cloudtrail:AddTags"},
		{Name: "RemoveTags", Method: http.MethodPost, IAMAction: "cloudtrail:RemoveTags"},
		{Name: "ListTags", Method: http.MethodPost, IAMAction: "cloudtrail:ListTags"},
	}
}

// HealthCheck always returns nil.
func (s *CloudTrailService) HealthCheck() error { return nil }

// HandleRequest routes an incoming CloudTrail request to the appropriate handler.
func (s *CloudTrailService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateTrail":
		return handleCreateTrail(ctx, s.store)
	case "GetTrail":
		return handleGetTrail(ctx, s.store)
	case "DescribeTrails":
		return handleDescribeTrails(ctx, s.store)
	case "DeleteTrail":
		return handleDeleteTrail(ctx, s.store)
	case "UpdateTrail":
		return handleUpdateTrail(ctx, s.store)
	case "StartLogging":
		return handleStartLogging(ctx, s.store)
	case "StopLogging":
		return handleStopLogging(ctx, s.store)
	case "GetTrailStatus":
		return handleGetTrailStatus(ctx, s.store)
	case "PutEventSelectors":
		return handlePutEventSelectors(ctx, s.store)
	case "GetEventSelectors":
		return handleGetEventSelectors(ctx, s.store)
	case "PutInsightSelectors":
		return handlePutInsightSelectors(ctx, s.store)
	case "GetInsightSelectors":
		return handleGetInsightSelectors(ctx, s.store)
	case "LookupEvents":
		return handleLookupEvents(ctx, s.store)
	case "AddTags":
		return handleAddTags(ctx, s.store)
	case "RemoveTags":
		return handleRemoveTags(ctx, s.store)
	case "ListTags":
		return handleListTags(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
