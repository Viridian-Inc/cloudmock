package scheduler

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

// ---- CreateSchedule ----

type flexibleTimeWindowJSON struct {
	Mode                   string `json:"Mode"`
	MaximumWindowInMinutes int    `json:"MaximumWindowInMinutes,omitempty"`
}

type targetJSON struct {
	Arn     string `json:"Arn"`
	RoleArn string `json:"RoleArn"`
	Input   string `json:"Input,omitempty"`
}

type createScheduleRequest struct {
	Name                       string                  `json:"Name"`
	GroupName                  string                  `json:"GroupName"`
	Description                string                  `json:"Description"`
	ScheduleExpression         string                  `json:"ScheduleExpression"`
	ScheduleExpressionTimezone string                  `json:"ScheduleExpressionTimezone"`
	State                      string                  `json:"State"`
	FlexibleTimeWindow         *flexibleTimeWindowJSON `json:"FlexibleTimeWindow"`
	Target                     *targetJSON             `json:"Target"`
	KmsKeyArn                  string                  `json:"KmsKeyArn"`
	Tags                       map[string]string       `json:"Tags"`
}

type createScheduleResponse struct {
	ScheduleArn string `json:"ScheduleArn"`
}

func handleCreateSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" || req.ScheduleExpression == "" {
		return jsonErr(service.ErrValidation("Name and ScheduleExpression are required."))
	}
	var fw *FlexibleTimeWindow
	if req.FlexibleTimeWindow != nil {
		fw = &FlexibleTimeWindow{Mode: req.FlexibleTimeWindow.Mode, MaximumWindowInMinutes: req.FlexibleTimeWindow.MaximumWindowInMinutes}
	}
	var tgt *Target
	if req.Target != nil {
		tgt = &Target{Arn: req.Target.Arn, RoleArn: req.Target.RoleArn, Input: req.Target.Input}
	}
	sched, ok := store.CreateSchedule(req.Name, req.GroupName, req.Description, req.ScheduleExpression, req.ScheduleExpressionTimezone, req.State, fw, tgt, req.KmsKeyArn, nil, nil, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("ConflictException", "Schedule "+req.Name+" already exists or group not found.", http.StatusConflict))
	}
	return jsonOK(&createScheduleResponse{ScheduleArn: sched.ARN})
}

// ---- GetSchedule ----

type getScheduleRequest struct {
	Name      string `json:"Name"`
	GroupName string `json:"GroupName"`
}

type scheduleResponse struct {
	Arn                        string                  `json:"Arn"`
	Name                       string                  `json:"Name"`
	GroupName                  string                  `json:"GroupName"`
	Description                string                  `json:"Description,omitempty"`
	ScheduleExpression         string                  `json:"ScheduleExpression"`
	ScheduleExpressionTimezone string                  `json:"ScheduleExpressionTimezone,omitempty"`
	State                      string                  `json:"State"`
	FlexibleTimeWindow         *flexibleTimeWindowJSON `json:"FlexibleTimeWindow,omitempty"`
	Target                     *targetJSON             `json:"Target,omitempty"`
	KmsKeyArn                  string                  `json:"KmsKeyArn,omitempty"`
	CreationDate               string                  `json:"CreationDate"`
	LastModificationDate       string                  `json:"LastModificationDate"`
	LastInvocationTime         string                  `json:"LastInvocationTime,omitempty"`
	InvocationCount            int                     `json:"InvocationCount,omitempty"`
}

func schedToResponse(sched *Schedule) *scheduleResponse {
	r := &scheduleResponse{
		Arn: sched.ARN, Name: sched.Name, GroupName: sched.GroupName,
		Description: sched.Description, ScheduleExpression: sched.ScheduleExpression,
		ScheduleExpressionTimezone: sched.ScheduleExpressionTimezone,
		State: sched.State, KmsKeyArn: sched.KmsKeyArn,
		CreationDate:         sched.CreationDate.Format("2006-01-02T15:04:05Z"),
		LastModificationDate: sched.LastModificationDate.Format("2006-01-02T15:04:05Z"),
		InvocationCount:      len(sched.InvocationHistory),
	}
	if len(sched.InvocationHistory) > 0 {
		r.LastInvocationTime = sched.InvocationHistory[len(sched.InvocationHistory)-1].Time.Format("2006-01-02T15:04:05Z")
	}
	if sched.FlexibleTimeWindow != nil {
		r.FlexibleTimeWindow = &flexibleTimeWindowJSON{Mode: sched.FlexibleTimeWindow.Mode, MaximumWindowInMinutes: sched.FlexibleTimeWindow.MaximumWindowInMinutes}
	}
	if sched.Target != nil {
		r.Target = &targetJSON{Arn: sched.Target.Arn, RoleArn: sched.Target.RoleArn, Input: sched.Target.Input}
	}
	return r
}

func handleGetSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// For REST-JSON, name may come from path params
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	groupName := req.GroupName
	if groupName == "" {
		groupName = ctx.Params["GroupName"]
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	sched, ok := store.GetSchedule(name, groupName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Schedule "+name+" not found.", http.StatusNotFound))
	}
	return jsonOK(schedToResponse(sched))
}

// ---- ListSchedules ----

type listSchedulesRequest struct {
	GroupName  string `json:"GroupName"`
	State      string `json:"State"`
	NamePrefix string `json:"NamePrefix"`
}

type scheduleSummaryJSON struct {
	Arn       string `json:"Arn"`
	Name      string `json:"Name"`
	GroupName string `json:"GroupName"`
	State     string `json:"State"`
}

type listSchedulesResponse struct {
	Schedules []scheduleSummaryJSON `json:"Schedules"`
}

func handleListSchedules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listSchedulesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	groupName := req.GroupName
	if groupName == "" {
		groupName = ctx.Params["GroupName"]
	}
	schedules := store.ListSchedules(groupName, req.State, req.NamePrefix)
	items := make([]scheduleSummaryJSON, 0, len(schedules))
	for _, s := range schedules {
		items = append(items, scheduleSummaryJSON{Arn: s.ARN, Name: s.Name, GroupName: s.GroupName, State: s.State})
	}
	return jsonOK(&listSchedulesResponse{Schedules: items})
}

// ---- UpdateSchedule ----

type updateScheduleRequest struct {
	Name                       string                  `json:"Name"`
	GroupName                  string                  `json:"GroupName"`
	Description                string                  `json:"Description"`
	ScheduleExpression         string                  `json:"ScheduleExpression"`
	ScheduleExpressionTimezone string                  `json:"ScheduleExpressionTimezone"`
	State                      string                  `json:"State"`
	FlexibleTimeWindow         *flexibleTimeWindowJSON `json:"FlexibleTimeWindow"`
	Target                     *targetJSON             `json:"Target"`
}

type updateScheduleResponse struct {
	ScheduleArn string `json:"ScheduleArn"`
}

func handleUpdateSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	var fw *FlexibleTimeWindow
	if req.FlexibleTimeWindow != nil {
		fw = &FlexibleTimeWindow{Mode: req.FlexibleTimeWindow.Mode, MaximumWindowInMinutes: req.FlexibleTimeWindow.MaximumWindowInMinutes}
	}
	var tgt *Target
	if req.Target != nil {
		tgt = &Target{Arn: req.Target.Arn, RoleArn: req.Target.RoleArn, Input: req.Target.Input}
	}
	sched, ok := store.UpdateSchedule(name, req.GroupName, req.Description, req.ScheduleExpression, req.ScheduleExpressionTimezone, req.State, fw, tgt)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Schedule "+name+" not found.", http.StatusNotFound))
	}
	return jsonOK(&updateScheduleResponse{ScheduleArn: sched.ARN})
}

// ---- DeleteSchedule ----

type deleteScheduleRequest struct {
	Name      string `json:"Name"`
	GroupName string `json:"GroupName"`
}

func handleDeleteSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	groupName := req.GroupName
	if groupName == "" {
		groupName = ctx.Params["GroupName"]
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if !store.DeleteSchedule(name, groupName) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Schedule "+name+" not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- CreateScheduleGroup ----

type createScheduleGroupRequest struct {
	Name string            `json:"Name"`
	Tags map[string]string `json:"Tags"`
}

type createScheduleGroupResponse struct {
	ScheduleGroupArn string `json:"ScheduleGroupArn"`
}

func handleCreateScheduleGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createScheduleGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	group, ok := store.CreateScheduleGroup(req.Name, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("ConflictException", "Schedule group "+req.Name+" already exists.", http.StatusConflict))
	}
	return jsonOK(&createScheduleGroupResponse{ScheduleGroupArn: group.ARN})
}

// ---- GetScheduleGroup ----

type getScheduleGroupRequest struct {
	Name string `json:"Name"`
}

type scheduleGroupResponse struct {
	Arn                  string `json:"Arn"`
	Name                 string `json:"Name"`
	State                string `json:"State"`
	CreationDate         string `json:"CreationDate"`
	LastModificationDate string `json:"LastModificationDate"`
}

func handleGetScheduleGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getScheduleGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	group, ok := store.GetScheduleGroup(name)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Schedule group "+name+" not found.", http.StatusNotFound))
	}
	return jsonOK(&scheduleGroupResponse{
		Arn: group.ARN, Name: group.Name, State: group.State,
		CreationDate:         group.CreationDate.Format("2006-01-02T15:04:05Z"),
		LastModificationDate: group.LastModificationDate.Format("2006-01-02T15:04:05Z"),
	})
}

// ---- ListScheduleGroups ----

type listScheduleGroupsRequest struct {
	NamePrefix string `json:"NamePrefix"`
}

type scheduleGroupSummaryJSON struct {
	Arn   string `json:"Arn"`
	Name  string `json:"Name"`
	State string `json:"State"`
}

type listScheduleGroupsResponse struct {
	ScheduleGroups []scheduleGroupSummaryJSON `json:"ScheduleGroups"`
}

func handleListScheduleGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listScheduleGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	groups := store.ListScheduleGroups(req.NamePrefix)
	items := make([]scheduleGroupSummaryJSON, 0, len(groups))
	for _, g := range groups {
		items = append(items, scheduleGroupSummaryJSON{Arn: g.ARN, Name: g.Name, State: g.State})
	}
	return jsonOK(&listScheduleGroupsResponse{ScheduleGroups: items})
}

// ---- DeleteScheduleGroup ----

type deleteScheduleGroupRequest struct {
	Name string `json:"Name"`
}

func handleDeleteScheduleGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteScheduleGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if !store.DeleteScheduleGroup(name) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Schedule group "+name+" not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceArn string            `json:"ResourceArn"`
	Tags        map[string]string `json:"Tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	if !store.TagResource(req.ResourceArn, req.Tags) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceArn string   `json:"ResourceArn"`
	TagKeys     []string `json:"TagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	if !store.UntagResource(req.ResourceArn, req.TagKeys) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- ListTagsForResource ----

type listTagsRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

type listTagsResponse struct {
	Tags map[string]string `json:"Tags"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["ResourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags, ok := store.ListTagsForResource(arn)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(&listTagsResponse{Tags: tags})
}
