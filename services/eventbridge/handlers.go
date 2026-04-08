package eventbridge

import (
	"crypto/rand"
	gojson "github.com/goccy/go-json"
	"fmt"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

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

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func emptyOK() (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       struct{}{},
		Format:     service.FormatJSON,
	}, nil
}

// newUUID returns a random UUID-shaped identifier.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---- CreateEventBus ----

type createEventBusRequest struct {
	Name string            `json:"Name"`
	Tags map[string]string `json:"Tags"`
}

type createEventBusResponse struct {
	EventBusArn string `json:"EventBusArn"`
}

func handleCreateEventBus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createEventBusRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	bus, ok := store.CreateEventBus(req.Name, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceAlreadyExistsException",
			"Event bus "+req.Name+" already exists.", http.StatusConflict))
	}

	return jsonOK(&createEventBusResponse{EventBusArn: bus.ARN})
}

// ---- DeleteEventBus ----

type deleteEventBusRequest struct {
	Name string `json:"Name"`
}

func handleDeleteEventBus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteEventBusRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if req.Name == "default" {
		return jsonErr(service.NewAWSError("ValidationException",
			"Cannot delete the default event bus.", http.StatusBadRequest))
	}

	if !store.DeleteEventBus(req.Name) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Event bus "+req.Name+" does not exist.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- DescribeEventBus ----

type describeEventBusRequest struct {
	Name string `json:"Name"`
}

type describeEventBusResponse struct {
	Name   string `json:"Name"`
	Arn    string `json:"Arn"`
	Policy string `json:"Policy,omitempty"`
}

func handleDescribeEventBus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeEventBusRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := req.Name
	if name == "" {
		name = "default"
	}

	bus, ok := store.GetEventBus(name)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Event bus "+name+" does not exist.", http.StatusNotFound))
	}

	return jsonOK(&describeEventBusResponse{
		Name:   bus.Name,
		Arn:    bus.ARN,
		Policy: bus.Policy,
	})
}

// ---- ListEventBuses ----

type listEventBusesRequest struct {
	NamePrefix string `json:"NamePrefix"`
	Limit      int    `json:"Limit"`
}

type eventBusEntry struct {
	Name   string `json:"Name"`
	Arn    string `json:"Arn"`
	Policy string `json:"Policy,omitempty"`
}

type listEventBusesResponse struct {
	EventBuses []eventBusEntry `json:"EventBuses"`
}

func handleListEventBuses(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listEventBusesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	buses := store.ListEventBuses()
	entries := make([]eventBusEntry, 0, len(buses))
	for _, bus := range buses {
		if req.NamePrefix != "" {
			if len(bus.Name) < len(req.NamePrefix) || bus.Name[:len(req.NamePrefix)] != req.NamePrefix {
				continue
			}
		}
		entries = append(entries, eventBusEntry{
			Name:   bus.Name,
			Arn:    bus.ARN,
			Policy: bus.Policy,
		})
	}

	return jsonOK(&listEventBusesResponse{EventBuses: entries})
}

// ---- PutRule ----

type putRuleRequest struct {
	Name               string            `json:"Name"`
	EventBusName       string            `json:"EventBusName"`
	EventPattern       string            `json:"EventPattern"`
	ScheduleExpression string            `json:"ScheduleExpression"`
	State              string            `json:"State"`
	Description        string            `json:"Description"`
	Tags               map[string]string `json:"Tags"`
}

type putRuleResponse struct {
	RuleArn string `json:"RuleArn"`
}

func handlePutRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putRuleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	busName := req.EventBusName
	if busName == "" {
		busName = "default"
	}

	arn, ok := store.PutRule(busName, req.Name, req.EventPattern, req.ScheduleExpression,
		req.State, req.Description, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Event bus "+busName+" does not exist.", http.StatusNotFound))
	}

	return jsonOK(&putRuleResponse{RuleArn: arn})
}

// ---- DeleteRule ----

type deleteRuleRequest struct {
	Name         string `json:"Name"`
	EventBusName string `json:"EventBusName"`
}

func handleDeleteRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteRuleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	busName := req.EventBusName
	if busName == "" {
		busName = "default"
	}

	if !store.DeleteRule(busName, req.Name) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Rule "+req.Name+" does not exist.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- DescribeRule ----

type describeRuleRequest struct {
	Name         string `json:"Name"`
	EventBusName string `json:"EventBusName"`
}

type describeRuleResponse struct {
	Name               string `json:"Name"`
	Arn                string `json:"Arn"`
	EventBusName       string `json:"EventBusName"`
	EventPattern       string `json:"EventPattern,omitempty"`
	ScheduleExpression string `json:"ScheduleExpression,omitempty"`
	State              string `json:"State"`
	Description        string `json:"Description,omitempty"`
}

func handleDescribeRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeRuleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	busName := req.EventBusName
	if busName == "" {
		busName = "default"
	}

	rule, ok := store.GetRule(busName, req.Name)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Rule "+req.Name+" does not exist.", http.StatusNotFound))
	}

	return jsonOK(&describeRuleResponse{
		Name:               rule.Name,
		Arn:                rule.ARN,
		EventBusName:       rule.EventBusName,
		EventPattern:       rule.EventPattern,
		ScheduleExpression: rule.ScheduleExpression,
		State:              rule.State,
		Description:        rule.Description,
	})
}

// ---- ListRules ----

type listRulesRequest struct {
	EventBusName string `json:"EventBusName"`
	NamePrefix   string `json:"NamePrefix"`
}

type ruleEntry struct {
	Name               string `json:"Name"`
	Arn                string `json:"Arn"`
	EventBusName       string `json:"EventBusName"`
	EventPattern       string `json:"EventPattern,omitempty"`
	ScheduleExpression string `json:"ScheduleExpression,omitempty"`
	State              string `json:"State"`
	Description        string `json:"Description,omitempty"`
}

type listRulesResponse struct {
	Rules []ruleEntry `json:"Rules"`
}

func handleListRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listRulesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	busName := req.EventBusName
	if busName == "" {
		busName = "default"
	}

	rules, ok := store.ListRules(busName, req.NamePrefix)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Event bus "+busName+" does not exist.", http.StatusNotFound))
	}

	entries := make([]ruleEntry, 0, len(rules))
	for _, r := range rules {
		entries = append(entries, ruleEntry{
			Name:               r.Name,
			Arn:                r.ARN,
			EventBusName:       r.EventBusName,
			EventPattern:       r.EventPattern,
			ScheduleExpression: r.ScheduleExpression,
			State:              r.State,
			Description:        r.Description,
		})
	}

	return jsonOK(&listRulesResponse{Rules: entries})
}

// ---- PutTargets ----

type targetJSON struct {
	Id        string `json:"Id"`
	Arn       string `json:"Arn"`
	Input     string `json:"Input,omitempty"`
	InputPath string `json:"InputPath,omitempty"`
}

type putTargetsRequest struct {
	Rule         string       `json:"Rule"`
	EventBusName string       `json:"EventBusName"`
	Targets      []targetJSON `json:"Targets"`
}

type failedEntry struct {
	TargetId     string `json:"TargetId"`
	ErrorCode    string `json:"ErrorCode"`
	ErrorMessage string `json:"ErrorMessage"`
}

type putTargetsResponse struct {
	FailedEntryCount int           `json:"FailedEntryCount"`
	FailedEntries    []failedEntry `json:"FailedEntries"`
}

func handlePutTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putTargetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Rule == "" {
		return jsonErr(service.ErrValidation("Rule is required."))
	}

	busName := req.EventBusName
	if busName == "" {
		busName = "default"
	}

	targets := make([]Target, 0, len(req.Targets))
	for _, t := range req.Targets {
		targets = append(targets, Target{
			Id:        t.Id,
			Arn:       t.Arn,
			Input:     t.Input,
			InputPath: t.InputPath,
		})
	}

	_, ok := store.PutTargets(busName, req.Rule, targets)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Rule "+req.Rule+" does not exist.", http.StatusNotFound))
	}

	return jsonOK(&putTargetsResponse{
		FailedEntryCount: 0,
		FailedEntries:    []failedEntry{},
	})
}

// ---- RemoveTargets ----

type removeTargetsRequest struct {
	Rule         string   `json:"Rule"`
	EventBusName string   `json:"EventBusName"`
	Ids          []string `json:"Ids"`
}

type removeTargetsFailedEntry struct {
	TargetId     string `json:"TargetId"`
	ErrorCode    string `json:"ErrorCode"`
	ErrorMessage string `json:"ErrorMessage"`
}

type removeTargetsResponse struct {
	FailedEntryCount int                        `json:"FailedEntryCount"`
	FailedEntries    []removeTargetsFailedEntry `json:"FailedEntries"`
}

func handleRemoveTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req removeTargetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Rule == "" {
		return jsonErr(service.ErrValidation("Rule is required."))
	}

	busName := req.EventBusName
	if busName == "" {
		busName = "default"
	}

	notFound, ok := store.RemoveTargets(busName, req.Rule, req.Ids)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Rule "+req.Rule+" does not exist.", http.StatusNotFound))
	}

	failed := make([]removeTargetsFailedEntry, 0, len(notFound))
	for _, id := range notFound {
		failed = append(failed, removeTargetsFailedEntry{
			TargetId:     id,
			ErrorCode:    "TargetNotFoundException",
			ErrorMessage: "Target " + id + " not found.",
		})
	}

	return jsonOK(&removeTargetsResponse{
		FailedEntryCount: len(failed),
		FailedEntries:    failed,
	})
}

// ---- ListTargetsByRule ----

type listTargetsByRuleRequest struct {
	Rule         string `json:"Rule"`
	EventBusName string `json:"EventBusName"`
}

type listTargetsByRuleResponse struct {
	Targets []targetJSON `json:"Targets"`
}

func handleListTargetsByRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTargetsByRuleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Rule == "" {
		return jsonErr(service.ErrValidation("Rule is required."))
	}

	busName := req.EventBusName
	if busName == "" {
		busName = "default"
	}

	targets, ok := store.ListTargetsByRule(busName, req.Rule)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Rule "+req.Rule+" does not exist.", http.StatusNotFound))
	}

	entries := make([]targetJSON, 0, len(targets))
	for _, t := range targets {
		entries = append(entries, targetJSON{
			Id:        t.Id,
			Arn:       t.Arn,
			Input:     t.Input,
			InputPath: t.InputPath,
		})
	}

	return jsonOK(&listTargetsByRuleResponse{Targets: entries})
}

// ---- PutEvents ----

type putEventEntry struct {
	Source       string   `json:"Source"`
	DetailType   string   `json:"DetailType"`
	Detail       string   `json:"Detail"`
	EventBusName string   `json:"EventBusName"`
	Resources    []string `json:"Resources"`
	Time         any      `json:"Time,omitempty"` // string (ISO8601) or number (epoch seconds)
}

type putEventsRequest struct {
	Entries []putEventEntry `json:"Entries"`
}

type putEventResultEntry struct {
	EventId string `json:"EventId,omitempty"`
}

type putEventsResponse struct {
	FailedEntryCount int                   `json:"FailedEntryCount"`
	Entries          []putEventResultEntry `json:"Entries"`
}

func handlePutEvents(ctx *service.RequestContext, store *Store, locator ServiceLocator) (*service.Response, error) {
	var req putEventsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(req.Entries) == 0 {
		return jsonErr(service.ErrValidation("Entries is required and must not be empty."))
	}

	storeEntries := make([]PutEvent, 0, len(req.Entries))
	for _, e := range req.Entries {
		busName := e.EventBusName
		if busName == "" {
			busName = "default"
		}
		var t time.Time
		switch v := e.Time.(type) {
		case string:
			if v != "" {
				if parsed, err := time.Parse(time.RFC3339, v); err == nil {
					t = parsed
				}
			}
		case float64:
			sec := int64(v)
			nsec := int64((v - float64(sec)) * 1e9)
			t = time.Unix(sec, nsec)
		}
		storeEntries = append(storeEntries, PutEvent{
			Source:       e.Source,
			DetailType:   e.DetailType,
			Detail:       e.Detail,
			EventBusName: busName,
			Resources:    e.Resources,
			Time:         t,
		})
	}

	ids := store.PutEvents(storeEntries)

	// Deliver events to matching rule targets.
	if locator != nil {
		for i, entry := range storeEntries {
			entry.EventId = ids[i]
			deliverToRuleTargets(store, locator, &entry)
		}
	}

	resultEntries := make([]putEventResultEntry, 0, len(ids))
	for _, id := range ids {
		resultEntries = append(resultEntries, putEventResultEntry{EventId: id})
	}

	return jsonOK(&putEventsResponse{
		FailedEntryCount: 0,
		Entries:          resultEntries,
	})
}

// SQSEnqueuer is an interface for directly enqueuing messages into SQS queues.
type SQSEnqueuer interface {
	EnqueueDirect(queueName, messageBody string) bool
}

// SNSPublisher is an interface for directly publishing messages to SNS topics.
type SNSPublisher interface {
	PublishDirect(topicName, message, subject string) bool
}

// LambdaInvoker is an interface for invoking Lambda functions directly.
type LambdaInvoker interface {
	InvokeDirect(functionName string, event []byte) ([]byte, error)
}

// deliverToRuleTargets checks rules on the event bus for a matching event and
// delivers to each target (SQS queue or SNS topic).
func deliverToRuleTargets(store *Store, locator ServiceLocator, event *PutEvent) {
	rules, ok := store.ListRules(event.EventBusName, "")
	if !ok {
		return
	}

	for _, rule := range rules {
		if rule.State != "ENABLED" {
			continue
		}

		if !matchesEventPattern(rule.EventPattern, event) {
			continue
		}

		for _, target := range rule.Targets {
			deliverToTarget(locator, event, &target)
		}
	}
}

// matchesEventPattern checks if a PutEvent matches a rule's event pattern.
// The event pattern is a JSON object with field-level filtering.
// For simplicity, we match on the "source" field.
func matchesEventPattern(pattern string, event *PutEvent) bool {
	if pattern == "" {
		return true
	}

	// Parse the pattern as JSON.
	var patternMap map[string]any
	if err := gojson.Unmarshal([]byte(pattern), &patternMap); err != nil {
		return false
	}

	// Check "source" filter — AWS EventBridge matches if the event source
	// is in the pattern's source list.
	if sourceFilter, ok := patternMap["source"]; ok {
		sourceList, ok := sourceFilter.([]any)
		if !ok {
			return false
		}
		found := false
		for _, s := range sourceList {
			if str, ok := s.(string); ok && str == event.Source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check "detail-type" filter.
	if dtFilter, ok := patternMap["detail-type"]; ok {
		dtList, ok := dtFilter.([]any)
		if !ok {
			return false
		}
		found := false
		for _, dt := range dtList {
			if str, ok := dt.(string); ok && str == event.DetailType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// deliverToTarget routes an event to a single target based on its ARN.
func deliverToTarget(locator ServiceLocator, event *PutEvent, target *Target) {
	arn := target.Arn

	// Determine message body: use target's Input override if set, otherwise the event detail.
	messageBody := event.Detail
	if target.Input != "" {
		messageBody = target.Input
	}

	// Determine target type from ARN.
	parts := splitARN(arn)
	if len(parts) < 3 {
		return
	}
	svcName := parts[2] // e.g., "sqs", "sns"

	switch svcName {
	case "sqs":
		if len(parts) < 6 {
			return
		}
		queueName := parts[5]
		svc, err := locator.Lookup("sqs")
		if err != nil {
			return
		}
		if enqueuer, ok := svc.(SQSEnqueuer); ok {
			// Wrap in EventBridge envelope.
			envelope := fmt.Sprintf(`{"version":"0","id":"%s","source":"%s","detail-type":"%s","time":"%s","region":"","resources":[],"detail":%s}`,
				event.EventId, event.Source, event.DetailType,
				event.Time.Format(time.RFC3339), messageBody)
			enqueuer.EnqueueDirect(queueName, envelope)
		}

	case "sns":
		if len(parts) < 6 {
			return
		}
		topicName := parts[5]
		svc, err := locator.Lookup("sns")
		if err != nil {
			return
		}
		if publisher, ok := svc.(SNSPublisher); ok {
			publisher.PublishDirect(topicName, messageBody, "EventBridge Notification")
		}

	case "lambda":
		// Lambda target: ARN resource is "function:name"
		if len(parts) < 6 {
			return
		}
		resource := parts[5]
		funcName := resource
		const funcPrefix = "function:"
		if len(resource) > len(funcPrefix) && resource[:len(funcPrefix)] == funcPrefix {
			funcName = resource[len(funcPrefix):]
		}
		// Build EventBridge event payload for Lambda.
		payload := fmt.Sprintf(`{"version":"0","id":"%s","source":"%s","detail-type":"%s","time":"%s","region":"","resources":[],"detail":%s}`,
			event.EventId, event.Source, event.DetailType,
			event.Time.Format(time.RFC3339), messageBody)
		svc, err := locator.Lookup("lambda")
		if err != nil {
			return
		}
		if invoker, ok := svc.(LambdaInvoker); ok {
			invoker.InvokeDirect(funcName, []byte(payload))
		}
	}
}

// splitARN splits an ARN into its components.
func splitARN(arn string) []string {
	result := make([]string, 0, 6)
	s := arn
	for i := 0; i < 5; i++ {
		idx := -1
		for j := 0; j < len(s); j++ {
			if s[j] == ':' {
				idx = j
				break
			}
		}
		if idx < 0 {
			result = append(result, s)
			return result
		}
		result = append(result, s[:idx])
		s = s[idx+1:]
	}
	result = append(result, s)
	return result
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
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Resource "+req.ResourceARN+" does not exist.", http.StatusNotFound))
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
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Resource "+req.ResourceARN+" does not exist.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- ListTagsForResource ----

type listTagsForResourceRequest struct {
	ResourceARN string `json:"ResourceARN"`
}

type listTagsForResourceResponse struct {
	Tags map[string]string `json:"Tags"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}

	tags, ok := store.ListTagsForResource(req.ResourceARN)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Resource "+req.ResourceARN+" does not exist.", http.StatusNotFound))
	}
	return jsonOK(&listTagsForResourceResponse{Tags: tags})
}

// ---- EnableRule ----

type enableRuleRequest struct {
	Name         string `json:"Name"`
	EventBusName string `json:"EventBusName"`
}

func handleEnableRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req enableRuleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	busName := req.EventBusName
	if busName == "" {
		busName = "default"
	}

	if !store.SetRuleState(busName, req.Name, "ENABLED") {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Rule "+req.Name+" does not exist.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- DisableRule ----

type disableRuleRequest struct {
	Name         string `json:"Name"`
	EventBusName string `json:"EventBusName"`
}

func handleDisableRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req disableRuleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	busName := req.EventBusName
	if busName == "" {
		busName = "default"
	}

	if !store.SetRuleState(busName, req.Name, "DISABLED") {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Rule "+req.Name+" does not exist.", http.StatusNotFound))
	}
	return emptyOK()
}
