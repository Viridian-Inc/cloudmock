package applicationautoscaling

import (
	gojson "github.com/goccy/go-json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
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
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

// ---- RegisterScalableTarget ----

type registerScalableTargetRequest struct {
	ServiceNamespace  string         `json:"ServiceNamespace"`
	ResourceId        string         `json:"ResourceId"`
	ScalableDimension string         `json:"ScalableDimension"`
	MinCapacity       int            `json:"MinCapacity"`
	MaxCapacity       int            `json:"MaxCapacity"`
	RoleARN           string         `json:"RoleARN"`
	SuspendedState    *suspendedJSON `json:"SuspendedState"`
	Tags              map[string]string `json:"Tags"`
}

type suspendedJSON struct {
	DynamicScalingInSuspended  bool `json:"DynamicScalingInSuspended"`
	DynamicScalingOutSuspended bool `json:"DynamicScalingOutSuspended"`
	ScheduledScalingSuspended  bool `json:"ScheduledScalingSuspended"`
}

type registerScalableTargetResponse struct {
	ScalableTargetARN string `json:"ScalableTargetARN"`
}

func handleRegisterScalableTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req registerScalableTargetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceNamespace == "" || req.ResourceId == "" || req.ScalableDimension == "" {
		return jsonErr(service.ErrValidation("ServiceNamespace, ResourceId, and ScalableDimension are required."))
	}
	validNamespaces := map[string]bool{
		"ecs": true, "dynamodb": true, "ec2": true, "rds": true,
		"sagemaker": true, "custom-resource": true, "comprehend": true,
		"lambda": true, "cassandra": true, "kafka": true, "elasticache": true, "neptune": true,
	}
	if !validNamespaces[req.ServiceNamespace] {
		return jsonErr(service.ErrValidation("ServiceNamespace must be one of: ecs, dynamodb, ec2, rds, sagemaker, custom-resource, comprehend, lambda, cassandra, kafka, elasticache, neptune."))
	}
	var suspended *SuspendedState
	if req.SuspendedState != nil {
		suspended = &SuspendedState{
			DynamicScalingInSuspended:  req.SuspendedState.DynamicScalingInSuspended,
			DynamicScalingOutSuspended: req.SuspendedState.DynamicScalingOutSuspended,
			ScheduledScalingSuspended:  req.SuspendedState.ScheduledScalingSuspended,
		}
	}
	store.RegisterScalableTarget(req.ServiceNamespace, req.ResourceId, req.ScalableDimension, req.MinCapacity, req.MaxCapacity, req.RoleARN, suspended, req.Tags)
	return emptyOK()
}

// ---- DescribeScalableTargets ----

type describeScalableTargetsRequest struct {
	ServiceNamespace  string   `json:"ServiceNamespace"`
	ResourceIds       []string `json:"ResourceIds"`
	ScalableDimension string   `json:"ScalableDimension"`
}

type scalableTargetJSON struct {
	ServiceNamespace  string         `json:"ServiceNamespace"`
	ResourceId        string         `json:"ResourceId"`
	ScalableDimension string         `json:"ScalableDimension"`
	MinCapacity       int            `json:"MinCapacity"`
	MaxCapacity       int            `json:"MaxCapacity"`
	RoleARN           string         `json:"RoleARN,omitempty"`
	CreationTime      float64        `json:"CreationTime"`
	SuspendedState    *suspendedJSON `json:"SuspendedState,omitempty"`
}

type describeScalableTargetsResponse struct {
	ScalableTargets []scalableTargetJSON `json:"ScalableTargets"`
}

func handleDescribeScalableTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeScalableTargetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceNamespace == "" {
		return jsonErr(service.ErrValidation("ServiceNamespace is required."))
	}
	targets := store.DescribeScalableTargets(req.ServiceNamespace, req.ResourceIds, req.ScalableDimension)
	items := make([]scalableTargetJSON, 0, len(targets))
	for _, t := range targets {
		item := scalableTargetJSON{
			ServiceNamespace:  t.ServiceNamespace,
			ResourceId:        t.ResourceID,
			ScalableDimension: t.ScalableDimension,
			MinCapacity:       t.MinCapacity,
			MaxCapacity:       t.MaxCapacity,
			RoleARN:           t.RoleARN,
			CreationTime:      float64(t.CreationTime.Unix()),
		}
		if t.SuspendedState != nil {
			item.SuspendedState = &suspendedJSON{
				DynamicScalingInSuspended:  t.SuspendedState.DynamicScalingInSuspended,
				DynamicScalingOutSuspended: t.SuspendedState.DynamicScalingOutSuspended,
				ScheduledScalingSuspended:  t.SuspendedState.ScheduledScalingSuspended,
			}
		}
		items = append(items, item)
	}
	return jsonOK(&describeScalableTargetsResponse{ScalableTargets: items})
}

// ---- DeregisterScalableTarget ----

type deregisterScalableTargetRequest struct {
	ServiceNamespace  string `json:"ServiceNamespace"`
	ResourceId        string `json:"ResourceId"`
	ScalableDimension string `json:"ScalableDimension"`
}

func handleDeregisterScalableTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deregisterScalableTargetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceNamespace == "" || req.ResourceId == "" || req.ScalableDimension == "" {
		return jsonErr(service.ErrValidation("ServiceNamespace, ResourceId, and ScalableDimension are required."))
	}
	if !store.DeregisterScalableTarget(req.ServiceNamespace, req.ResourceId, req.ScalableDimension) {
		return jsonErr(service.NewAWSError("ObjectNotFoundException", "No scalable target found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- PutScalingPolicy ----

type putScalingPolicyRequest struct {
	PolicyName                          string         `json:"PolicyName"`
	ServiceNamespace                    string         `json:"ServiceNamespace"`
	ResourceId                          string         `json:"ResourceId"`
	ScalableDimension                   string         `json:"ScalableDimension"`
	PolicyType                          string         `json:"PolicyType"`
	TargetTrackingScalingPolicyConfiguration map[string]any `json:"TargetTrackingScalingPolicyConfiguration"`
	StepScalingPolicyConfiguration      map[string]any `json:"StepScalingPolicyConfiguration"`
}

type putScalingPolicyResponse struct {
	PolicyARN string `json:"PolicyARN"`
	Alarms    []any  `json:"Alarms"`
}

func handlePutScalingPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putScalingPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.PolicyName == "" || req.ServiceNamespace == "" || req.ResourceId == "" || req.ScalableDimension == "" {
		return jsonErr(service.ErrValidation("PolicyName, ServiceNamespace, ResourceId, and ScalableDimension are required."))
	}
	if req.PolicyType == "TargetTrackingScaling" {
		if req.TargetTrackingScalingPolicyConfiguration == nil {
			return jsonErr(service.ErrValidation("TargetTrackingScalingPolicyConfiguration is required for TargetTrackingScaling policy type."))
		}
		if _, ok := req.TargetTrackingScalingPolicyConfiguration["TargetValue"]; !ok {
			return jsonErr(service.ErrValidation("TargetValue is required in TargetTrackingScalingPolicyConfiguration."))
		}
	}
	policy := store.PutScalingPolicy(req.ServiceNamespace, req.ResourceId, req.ScalableDimension, req.PolicyName, req.PolicyType, req.TargetTrackingScalingPolicyConfiguration, req.StepScalingPolicyConfiguration, nil)
	return jsonOK(&putScalingPolicyResponse{PolicyARN: policy.PolicyARN, Alarms: []any{}})
}

// ---- DescribeScalingPolicies ----

type describeScalingPoliciesRequest struct {
	ServiceNamespace  string   `json:"ServiceNamespace"`
	ResourceId        string   `json:"ResourceId"`
	ScalableDimension string   `json:"ScalableDimension"`
	PolicyNames       []string `json:"PolicyNames"`
}

type scalingPolicyJSON struct {
	PolicyARN         string  `json:"PolicyARN"`
	PolicyName        string  `json:"PolicyName"`
	ServiceNamespace  string  `json:"ServiceNamespace"`
	ResourceId        string  `json:"ResourceId"`
	ScalableDimension string  `json:"ScalableDimension"`
	PolicyType        string  `json:"PolicyType"`
	CreationTime      float64 `json:"CreationTime"`
}

type describeScalingPoliciesResponse struct {
	ScalingPolicies []scalingPolicyJSON `json:"ScalingPolicies"`
}

func handleDescribeScalingPolicies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeScalingPoliciesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceNamespace == "" {
		return jsonErr(service.ErrValidation("ServiceNamespace is required."))
	}
	policies := store.DescribeScalingPolicies(req.ServiceNamespace, req.ResourceId, req.ScalableDimension, req.PolicyNames)
	items := make([]scalingPolicyJSON, 0, len(policies))
	for _, p := range policies {
		items = append(items, scalingPolicyJSON{
			PolicyARN:         p.PolicyARN,
			PolicyName:        p.PolicyName,
			ServiceNamespace:  p.ServiceNamespace,
			ResourceId:        p.ResourceID,
			ScalableDimension: p.ScalableDimension,
			PolicyType:        p.PolicyType,
			CreationTime:      float64(p.CreationTime.Unix()),
		})
	}
	return jsonOK(&describeScalingPoliciesResponse{ScalingPolicies: items})
}

// ---- DeleteScalingPolicy ----

type deleteScalingPolicyRequest struct {
	PolicyName        string `json:"PolicyName"`
	ServiceNamespace  string `json:"ServiceNamespace"`
	ResourceId        string `json:"ResourceId"`
	ScalableDimension string `json:"ScalableDimension"`
}

func handleDeleteScalingPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteScalingPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.PolicyName == "" || req.ServiceNamespace == "" || req.ResourceId == "" || req.ScalableDimension == "" {
		return jsonErr(service.ErrValidation("PolicyName, ServiceNamespace, ResourceId, and ScalableDimension are required."))
	}
	if !store.DeleteScalingPolicy(req.ServiceNamespace, req.ResourceId, req.ScalableDimension, req.PolicyName) {
		return jsonErr(service.NewAWSError("ObjectNotFoundException", "No scaling policy found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- PutScheduledAction ----

type putScheduledActionRequest struct {
	ScheduledActionName  string             `json:"ScheduledActionName"`
	ServiceNamespace     string             `json:"ServiceNamespace"`
	ResourceId           string             `json:"ResourceId"`
	ScalableDimension    string             `json:"ScalableDimension"`
	Schedule             string             `json:"Schedule"`
	Timezone             string             `json:"Timezone"`
	StartTime            *float64           `json:"StartTime"`
	EndTime              *float64           `json:"EndTime"`
	ScalableTargetAction *targetActionJSON  `json:"ScalableTargetAction"`
}

type targetActionJSON struct {
	MinCapacity int `json:"MinCapacity"`
	MaxCapacity int `json:"MaxCapacity"`
}

func handlePutScheduledAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putScheduledActionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ScheduledActionName == "" || req.ServiceNamespace == "" || req.ResourceId == "" || req.ScalableDimension == "" {
		return jsonErr(service.ErrValidation("ScheduledActionName, ServiceNamespace, ResourceId, and ScalableDimension are required."))
	}
	var startTime, endTime *time.Time
	if req.StartTime != nil {
		t := time.Unix(int64(*req.StartTime), 0).UTC()
		startTime = &t
	}
	if req.EndTime != nil {
		t := time.Unix(int64(*req.EndTime), 0).UTC()
		endTime = &t
	}
	var targetAction *ScalableTargetAction
	if req.ScalableTargetAction != nil {
		targetAction = &ScalableTargetAction{MinCapacity: req.ScalableTargetAction.MinCapacity, MaxCapacity: req.ScalableTargetAction.MaxCapacity}
	}
	store.PutScheduledAction(req.ServiceNamespace, req.ResourceId, req.ScalableDimension, req.ScheduledActionName, req.Schedule, req.Timezone, startTime, endTime, targetAction, nil)
	return emptyOK()
}

// ---- DescribeScheduledActions ----

type describeScheduledActionsRequest struct {
	ServiceNamespace     string   `json:"ServiceNamespace"`
	ResourceId           string   `json:"ResourceId"`
	ScalableDimension    string   `json:"ScalableDimension"`
	ScheduledActionNames []string `json:"ScheduledActionNames"`
}

type scheduledActionJSON struct {
	ScheduledActionARN   string            `json:"ScheduledActionARN"`
	ScheduledActionName  string            `json:"ScheduledActionName"`
	ServiceNamespace     string            `json:"ServiceNamespace"`
	ResourceId           string            `json:"ResourceId"`
	ScalableDimension    string            `json:"ScalableDimension"`
	Schedule             string            `json:"Schedule"`
	Timezone             string            `json:"Timezone,omitempty"`
	CreationTime         float64           `json:"CreationTime"`
	ScalableTargetAction *targetActionJSON `json:"ScalableTargetAction,omitempty"`
}

type describeScheduledActionsResponse struct {
	ScheduledActions []scheduledActionJSON `json:"ScheduledActions"`
}

func handleDescribeScheduledActions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeScheduledActionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceNamespace == "" {
		return jsonErr(service.ErrValidation("ServiceNamespace is required."))
	}
	actions := store.DescribeScheduledActions(req.ServiceNamespace, req.ResourceId, req.ScalableDimension, req.ScheduledActionNames)
	items := make([]scheduledActionJSON, 0, len(actions))
	for _, a := range actions {
		item := scheduledActionJSON{
			ScheduledActionARN:  a.ScheduledActionARN,
			ScheduledActionName: a.ScheduledActionName,
			ServiceNamespace:    a.ServiceNamespace,
			ResourceId:          a.ResourceID,
			ScalableDimension:   a.ScalableDimension,
			Schedule:            a.Schedule,
			Timezone:            a.Timezone,
			CreationTime:        float64(a.CreationTime.Unix()),
		}
		if a.ScalableTargetAction != nil {
			item.ScalableTargetAction = &targetActionJSON{MinCapacity: a.ScalableTargetAction.MinCapacity, MaxCapacity: a.ScalableTargetAction.MaxCapacity}
		}
		items = append(items, item)
	}
	return jsonOK(&describeScheduledActionsResponse{ScheduledActions: items})
}

// ---- DeleteScheduledAction ----

type deleteScheduledActionRequest struct {
	ScheduledActionName string `json:"ScheduledActionName"`
	ServiceNamespace    string `json:"ServiceNamespace"`
	ResourceId          string `json:"ResourceId"`
	ScalableDimension   string `json:"ScalableDimension"`
}

func handleDeleteScheduledAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteScheduledActionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ScheduledActionName == "" || req.ServiceNamespace == "" || req.ResourceId == "" || req.ScalableDimension == "" {
		return jsonErr(service.ErrValidation("ScheduledActionName, ServiceNamespace, ResourceId, and ScalableDimension are required."))
	}
	if !store.DeleteScheduledAction(req.ServiceNamespace, req.ResourceId, req.ScalableDimension, req.ScheduledActionName) {
		return jsonErr(service.NewAWSError("ObjectNotFoundException", "No scheduled action found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceARN string            `json:"ResourceARN"`
	Tags        map[string]string `json:"Tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	if !store.TagResource(req.ResourceARN, req.Tags) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found: "+req.ResourceARN, http.StatusNotFound))
	}
	return emptyOK()
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceARN string   `json:"ResourceARN"`
	TagKeys     []string `json:"TagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	if !store.UntagResource(req.ResourceARN, req.TagKeys) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found: "+req.ResourceARN, http.StatusNotFound))
	}
	return emptyOK()
}

// ---- ListTagsForResource ----

type listTagsRequest struct {
	ResourceARN string `json:"ResourceARN"`
}

type listTagsResponse struct {
	Tags map[string]string `json:"Tags"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	tags, ok := store.ListTagsForResource(req.ResourceARN)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found: "+req.ResourceARN, http.StatusNotFound))
	}
	return jsonOK(&listTagsResponse{Tags: tags})
}
