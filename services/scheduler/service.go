package scheduler

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// SchedulerService is the cloudmock implementation of the AWS EventBridge Scheduler API.
type SchedulerService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new SchedulerService for the given AWS account ID and region.
func New(accountID, region string) *SchedulerService {
	return &SchedulerService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *SchedulerService) Name() string { return "scheduler" }

// Actions returns the list of Scheduler API actions supported by this service.
func (s *SchedulerService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateSchedule", Method: http.MethodPost, IAMAction: "scheduler:CreateSchedule"},
		{Name: "GetSchedule", Method: http.MethodGet, IAMAction: "scheduler:GetSchedule"},
		{Name: "ListSchedules", Method: http.MethodGet, IAMAction: "scheduler:ListSchedules"},
		{Name: "UpdateSchedule", Method: http.MethodPut, IAMAction: "scheduler:UpdateSchedule"},
		{Name: "DeleteSchedule", Method: http.MethodDelete, IAMAction: "scheduler:DeleteSchedule"},
		{Name: "CreateScheduleGroup", Method: http.MethodPost, IAMAction: "scheduler:CreateScheduleGroup"},
		{Name: "GetScheduleGroup", Method: http.MethodGet, IAMAction: "scheduler:GetScheduleGroup"},
		{Name: "ListScheduleGroups", Method: http.MethodGet, IAMAction: "scheduler:ListScheduleGroups"},
		{Name: "DeleteScheduleGroup", Method: http.MethodDelete, IAMAction: "scheduler:DeleteScheduleGroup"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "scheduler:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "scheduler:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "scheduler:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *SchedulerService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Scheduler request to the appropriate handler.
func (s *SchedulerService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateSchedule":
		return handleCreateSchedule(ctx, s.store)
	case "GetSchedule":
		return handleGetSchedule(ctx, s.store)
	case "ListSchedules":
		return handleListSchedules(ctx, s.store)
	case "UpdateSchedule":
		return handleUpdateSchedule(ctx, s.store)
	case "DeleteSchedule":
		return handleDeleteSchedule(ctx, s.store)
	case "CreateScheduleGroup":
		return handleCreateScheduleGroup(ctx, s.store)
	case "GetScheduleGroup":
		return handleGetScheduleGroup(ctx, s.store)
	case "ListScheduleGroups":
		return handleListScheduleGroups(ctx, s.store)
	case "DeleteScheduleGroup":
		return handleDeleteScheduleGroup(ctx, s.store)
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
