package stepfunctions

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

type createStateMachineRequest struct {
	Name       string            `json:"name"`
	Definition string            `json:"definition"`
	RoleArn    string            `json:"roleArn"`
	Type       string            `json:"type"`
	Tags       []tagEntry        `json:"tags"`
}

type createStateMachineResponse struct {
	StateMachineArn string  `json:"stateMachineArn"`
	CreationDate    float64 `json:"creationDate"`
}

type deleteStateMachineRequest struct {
	StateMachineArn string `json:"stateMachineArn"`
}

type describeStateMachineRequest struct {
	StateMachineArn string `json:"stateMachineArn"`
}

type describeStateMachineResponse struct {
	StateMachineArn string  `json:"stateMachineArn"`
	Name            string  `json:"name"`
	Definition      string  `json:"definition"`
	RoleArn         string  `json:"roleArn"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	CreationDate    float64 `json:"creationDate"`
}

type listStateMachinesResponse struct {
	StateMachines []stateMachineListEntry `json:"stateMachines"`
}

type stateMachineListEntry struct {
	StateMachineArn string  `json:"stateMachineArn"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	CreationDate    float64 `json:"creationDate"`
}

type updateStateMachineRequest struct {
	StateMachineArn string `json:"stateMachineArn"`
	Definition      string `json:"definition"`
	RoleArn         string `json:"roleArn"`
}

type updateStateMachineResponse struct {
	UpdateDate float64 `json:"updateDate"`
}

type startExecutionRequest struct {
	StateMachineArn string `json:"stateMachineArn"`
	Name            string `json:"name"`
	Input           string `json:"input"`
}

type startExecutionResponse struct {
	ExecutionArn string  `json:"executionArn"`
	StartDate    float64 `json:"startDate"`
}

type describeExecutionRequest struct {
	ExecutionArn string `json:"executionArn"`
}

type describeExecutionResponse struct {
	ExecutionArn    string   `json:"executionArn"`
	StateMachineArn string   `json:"stateMachineArn"`
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	Input           string   `json:"input"`
	Output          string   `json:"output,omitempty"`
	StartDate       float64  `json:"startDate"`
	StopDate        *float64 `json:"stopDate,omitempty"`
}

type stopExecutionRequest struct {
	ExecutionArn string `json:"executionArn"`
	Cause        string `json:"cause"`
	Error        string `json:"error"`
}

type stopExecutionResponse struct {
	StopDate float64 `json:"stopDate"`
}

type listExecutionsRequest struct {
	StateMachineArn string `json:"stateMachineArn"`
}

type listExecutionsResponse struct {
	Executions []executionListEntry `json:"executions"`
}

type executionListEntry struct {
	ExecutionArn    string  `json:"executionArn"`
	StateMachineArn string  `json:"stateMachineArn"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	StartDate       float64 `json:"startDate"`
	StopDate        *float64 `json:"stopDate,omitempty"`
}

type getExecutionHistoryRequest struct {
	ExecutionArn string `json:"executionArn"`
}

type historyEventJSON struct {
	Timestamp       float64 `json:"timestamp"`
	Type            string  `json:"type"`
	Id              int64   `json:"id"`
	PreviousEventId int64   `json:"previousEventId"`
}

type getExecutionHistoryResponse struct {
	Events []historyEventJSON `json:"events"`
}

type tagResourceRequest struct {
	ResourceArn string     `json:"resourceArn"`
	Tags        []tagEntry `json:"tags"`
}

type untagResourceRequest struct {
	ResourceArn string   `json:"resourceArn"`
	TagKeys     []string `json:"tagKeys"`
}

type listTagsForResourceRequest struct {
	ResourceArn string `json:"resourceArn"`
}

type listTagsForResourceResponse struct {
	Tags []tagEntry `json:"tags"`
}

type tagEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func emptyOK() (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       struct{}{},
		Format:     service.FormatJSON,
	}, nil
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func unixFloat(t time.Time) float64 {
	return float64(t.Unix())
}

func unixFloatPtr(t *time.Time) *float64 {
	if t == nil {
		return nil
	}
	v := float64(t.Unix())
	return &v
}

func tagsFromEntries(entries []tagEntry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Key] = e.Value
	}
	return m
}

func entriesToTags(m map[string]string) []tagEntry {
	entries := make([]tagEntry, 0, len(m))
	for k, v := range m {
		entries = append(entries, tagEntry{Key: k, Value: v})
	}
	return entries
}

// ---- handlers ----

func handleCreateStateMachine(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createStateMachineRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"name is required.", http.StatusBadRequest))
	}
	if req.Definition == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"definition is required.", http.StatusBadRequest))
	}

	tags := tagsFromEntries(req.Tags)
	sm, awsErr := store.CreateStateMachine(req.Name, req.Definition, req.RoleArn, req.Type, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(createStateMachineResponse{
		StateMachineArn: sm.Arn,
		CreationDate:    unixFloat(sm.CreationDate),
	})
}

func handleDeleteStateMachine(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteStateMachineRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StateMachineArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"stateMachineArn is required.", http.StatusBadRequest))
	}
	if awsErr := store.DeleteStateMachine(req.StateMachineArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDescribeStateMachine(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeStateMachineRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StateMachineArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"stateMachineArn is required.", http.StatusBadRequest))
	}

	sm, awsErr := store.GetStateMachine(req.StateMachineArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(describeStateMachineResponse{
		StateMachineArn: sm.Arn,
		Name:            sm.Name,
		Definition:      sm.Definition,
		RoleArn:         sm.RoleArn,
		Type:            sm.Type,
		Status:          string(sm.Status),
		CreationDate:    unixFloat(sm.CreationDate),
	})
}

func handleListStateMachines(_ *service.RequestContext, store *Store) (*service.Response, error) {
	machines := store.ListStateMachines()
	entries := make([]stateMachineListEntry, 0, len(machines))
	for _, sm := range machines {
		entries = append(entries, stateMachineListEntry{
			StateMachineArn: sm.Arn,
			Name:            sm.Name,
			Type:            sm.Type,
			CreationDate:    unixFloat(sm.CreationDate),
		})
	}
	return jsonOK(listStateMachinesResponse{StateMachines: entries})
}

func handleUpdateStateMachine(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateStateMachineRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StateMachineArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"stateMachineArn is required.", http.StatusBadRequest))
	}

	_, awsErr := store.UpdateStateMachine(req.StateMachineArn, req.Definition, req.RoleArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(updateStateMachineResponse{
		UpdateDate: unixFloat(time.Now().UTC()),
	})
}

func handleStartExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StateMachineArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"stateMachineArn is required.", http.StatusBadRequest))
	}

	exec, awsErr := store.StartExecution(req.StateMachineArn, req.Name, req.Input)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(startExecutionResponse{
		ExecutionArn: exec.ExecutionArn,
		StartDate:    unixFloat(exec.StartDate),
	})
}

func handleDescribeExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ExecutionArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"executionArn is required.", http.StatusBadRequest))
	}

	exec, awsErr := store.GetExecution(req.ExecutionArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	resp := describeExecutionResponse{
		ExecutionArn:    exec.ExecutionArn,
		StateMachineArn: exec.StateMachineArn,
		Name:            exec.Name,
		Status:          string(exec.Status),
		Input:           exec.Input,
		StartDate:       unixFloat(exec.StartDate),
		StopDate:        unixFloatPtr(exec.StopDate),
	}
	if exec.Status == executionStatusSucceeded {
		resp.Output = exec.Output
	}

	return jsonOK(resp)
}

func handleStopExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req stopExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ExecutionArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"executionArn is required.", http.StatusBadRequest))
	}

	exec, awsErr := store.StopExecution(req.ExecutionArn, req.Cause, req.Error)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(stopExecutionResponse{
		StopDate: unixFloat(*exec.StopDate),
	})
}

func handleListExecutions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listExecutionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StateMachineArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"stateMachineArn is required.", http.StatusBadRequest))
	}

	execs, awsErr := store.ListExecutions(req.StateMachineArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	entries := make([]executionListEntry, 0, len(execs))
	for _, exec := range execs {
		entries = append(entries, executionListEntry{
			ExecutionArn:    exec.ExecutionArn,
			StateMachineArn: exec.StateMachineArn,
			Name:            exec.Name,
			Status:          string(exec.Status),
			StartDate:       unixFloat(exec.StartDate),
			StopDate:        unixFloatPtr(exec.StopDate),
		})
	}

	return jsonOK(listExecutionsResponse{Executions: entries})
}

func handleGetExecutionHistory(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getExecutionHistoryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ExecutionArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"executionArn is required.", http.StatusBadRequest))
	}

	events, awsErr := store.GetExecutionHistory(req.ExecutionArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	jsonEvents := make([]historyEventJSON, 0, len(events))
	for _, e := range events {
		jsonEvents = append(jsonEvents, historyEventJSON{
			Timestamp:       unixFloat(e.Timestamp),
			Type:            e.Type,
			Id:              e.Id,
			PreviousEventId: e.PreviousEventId,
		})
	}

	return jsonOK(getExecutionHistoryResponse{Events: jsonEvents})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"resourceArn is required.", http.StatusBadRequest))
	}

	tags := tagsFromEntries(req.Tags)
	if awsErr := store.TagResource(req.ResourceArn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"resourceArn is required.", http.StatusBadRequest))
	}

	if awsErr := store.UntagResource(req.ResourceArn, req.TagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"resourceArn is required.", http.StatusBadRequest))
	}

	tags, awsErr := store.ListTagsForResource(req.ResourceArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(listTagsForResourceResponse{Tags: entriesToTags(tags)})
}
