package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/neureaux/cloudmock/services/lambda"
	"github.com/neureaux/cloudmock/services/ses"
)

// Resettable is an optional interface that services can implement to support state reset.
type Resettable interface {
	Reset()
}

// ServiceInfo describes a registered service for the admin API.
type ServiceInfo struct {
	Name        string `json:"name"`
	ActionCount int    `json:"action_count"`
	Healthy     bool   `json:"healthy"`
}

// HealthResponse is the response body for the /api/health endpoint.
type HealthResponse struct {
	Status   string          `json:"status"`
	Services map[string]bool `json:"services"`
}

// API is the admin HTTP handler.
type API struct {
	cfg         *config.Config
	registry    *routing.Registry
	log         *gateway.RequestLog
	stats       *gateway.RequestStats
	broadcaster *EventBroadcaster
	lambdaLogs  *lambda.LogBuffer
	iamEngine   *iam.Engine
	sesStore    *ses.Store
	mux         *http.ServeMux
}

// New creates an admin API handler wired to the given registry, config, and request log/stats.
func New(cfg *config.Config, registry *routing.Registry, log *gateway.RequestLog, stats *gateway.RequestStats) *API {
	a := &API{
		cfg:         cfg,
		registry:    registry,
		log:         log,
		stats:       stats,
		broadcaster: NewEventBroadcaster(),
		mux:         http.NewServeMux(),
	}

	a.mux.HandleFunc("/api/services", a.handleServices)
	a.mux.HandleFunc("/api/services/", a.handleServiceByName)
	a.mux.HandleFunc("/api/reset", a.handleResetAll)
	a.mux.HandleFunc("/api/health", a.handleHealth)
	a.mux.HandleFunc("/api/config", a.handleConfig)
	a.mux.HandleFunc("/api/stats", a.handleStats)
	a.mux.HandleFunc("/api/requests", a.handleRequests)
	a.mux.HandleFunc("/api/stream", a.handleStream)
	a.mux.HandleFunc("/api/lambda/logs", a.handleLambdaLogs)
	a.mux.HandleFunc("/api/lambda/logs/stream", a.handleLambdaLogStream)
	a.mux.HandleFunc("/api/requests/", a.handleRequestByID)
	a.mux.HandleFunc("/api/iam/evaluate", a.handleIAMEvaluate)
	a.mux.HandleFunc("/api/ses/emails", a.handleSESEmails)
	a.mux.HandleFunc("/api/ses/emails/", a.handleSESEmailByID)
	a.mux.HandleFunc("/api/topology", a.handleTopology)
	a.mux.HandleFunc("/api/resources/", a.handleResources)

	return a
}

// Broadcaster returns the event broadcaster for use by middleware.
func (a *API) Broadcaster() *EventBroadcaster {
	return a.broadcaster
}

// SetLambdaLogs sets the Lambda log buffer for the admin API to serve.
func (a *API) SetLambdaLogs(logs *lambda.LogBuffer) {
	a.lambdaLogs = logs
	// Wire up the log buffer to broadcast lambda_log events.
	logs.SetOnEmit(func(entry lambda.LambdaLogEntry) {
		a.broadcaster.Broadcast("lambda_log", entry)
	})
}

// ServeHTTP implements http.Handler.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func (a *API) handleServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	svcs := a.registry.List()
	infos := make([]ServiceInfo, 0, len(svcs))
	for _, svc := range svcs {
		healthy := svc.HealthCheck() == nil
		infos = append(infos, ServiceInfo{
			Name:        svc.Name(),
			ActionCount: len(svc.Actions()),
			Healthy:     healthy,
		})
	}

	writeJSON(w, http.StatusOK, infos)
}

func (a *API) handleServiceByName(w http.ResponseWriter, r *http.Request) {
	// Parse: /api/services/{name} or /api/services/{name}/reset
	path := strings.TrimPrefix(r.URL.Path, "/api/services/")
	parts := strings.SplitN(path, "/", 2)
	name := parts[0]

	if name == "" {
		http.NotFound(w, r)
		return
	}

	// /api/services/{name}/reset
	if len(parts) == 2 && parts[1] == "reset" {
		a.handleServiceReset(w, r, name)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	svc, err := a.registry.Lookup(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	info := ServiceInfo{
		Name:        svc.Name(),
		ActionCount: len(svc.Actions()),
		Healthy:     svc.HealthCheck() == nil,
	}
	writeJSON(w, http.StatusOK, info)
}

func (a *API) handleServiceReset(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	svc, err := a.registry.Lookup(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if resettable, ok := svc.(Resettable); ok {
		resettable.Reset()
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "reset", "service": svc.Name()})
}

func (a *API) handleResetAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	svcs := a.registry.List()
	var resetNames []string
	for _, svc := range svcs {
		if resettable, ok := svc.(Resettable); ok {
			resettable.Reset()
			resetNames = append(resetNames, svc.Name())
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "reset", "services": resetNames})
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	svcs := a.registry.List()
	services := make(map[string]bool, len(svcs))
	allHealthy := true
	for _, svc := range svcs {
		healthy := svc.HealthCheck() == nil
		services[svc.Name()] = healthy
		if !healthy {
			allHealthy = false
		}
	}

	status := "healthy"
	if !allHealthy {
		status = "degraded"
	}

	resp := HealthResponse{
		Status:   status,
		Services: services,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *API) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, a.cfg)
}

func (a *API) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, a.stats.Snapshot())
}

func (a *API) handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	svcFilter := r.URL.Query().Get("service")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	entries := a.log.Recent(svcFilter, limit)
	writeJSON(w, http.StatusOK, entries)
}

// handleStream is the SSE endpoint that pushes real-time events to the dashboard.
func (a *API) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := a.broadcaster.Subscribe()
	defer a.broadcaster.Unsubscribe(ch)

	// Send an initial connected event.
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// handleLambdaLogs returns recent Lambda execution logs.
func (a *API) handleLambdaLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if a.lambdaLogs == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	functionFilter := r.URL.Query().Get("function")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	entries := a.lambdaLogs.Recent(functionFilter, limit)
	writeJSON(w, http.StatusOK, entries)
}

// handleLambdaLogStream is an SSE endpoint dedicated to Lambda logs.
func (a *API) handleLambdaLogStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := a.broadcaster.Subscribe()
	defer a.broadcaster.Unsubscribe(ch)

	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			// Only forward lambda_log events on this endpoint.
			if strings.Contains(msg, `"type":"lambda_log"`) {
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

// SetIAMEngine sets the IAM engine for the admin API to use for policy evaluation.
func (a *API) SetIAMEngine(engine *iam.Engine) {
	a.iamEngine = engine
}

// SetSESStore sets the SES store for the admin API to expose captured emails.
func (a *API) SetSESStore(store *ses.Store) {
	a.sesStore = store
}

// handleRequestByID returns the full detail of a single request entry.
func (a *API) handleRequestByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/requests/")

	if r.Method == http.MethodPost && strings.HasSuffix(id, "/replay") {
		// Replay not yet implemented — return a stub.
		id = strings.TrimSuffix(id, "/replay")
		entry := a.log.GetByID(id)
		if entry == nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "replayed", "id": id})
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entry := a.log.GetByID(id)
	if entry == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

// IAMEvalRequest is the request body for the IAM evaluate endpoint.
type IAMEvalRequest struct {
	Principal string `json:"principal"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
}

// IAMEvalResponse is the response for the IAM evaluate endpoint.
type IAMEvalResponse struct {
	Decision         string         `json:"decision"`
	Reason           string         `json:"reason"`
	MatchedStatement *iam.Statement `json:"matched_statement,omitempty"`
}

func (a *API) handleIAMEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if a.iamEngine == nil {
		writeJSON(w, http.StatusOK, IAMEvalResponse{
			Decision: "DENY",
			Reason:   "IAM engine not configured",
		})
		return
	}

	var req IAMEvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result := a.iamEngine.Evaluate(&iam.EvalRequest{
		Principal: req.Principal,
		Action:    req.Action,
		Resource:  req.Resource,
	})

	decision := "DENY"
	if result.Decision == iam.Allow {
		decision = "ALLOW"
	}

	resp := IAMEvalResponse{
		Decision: decision,
		Reason:   result.Reason,
	}
	if result.MatchedStatement != nil {
		resp.MatchedStatement = result.MatchedStatement
	}

	writeJSON(w, http.StatusOK, resp)
}

// SESEmailSummary is a summary of a captured email for listing.
type SESEmailSummary struct {
	MessageId string   `json:"message_id"`
	Source    string    `json:"source"`
	To       []string  `json:"to"`
	Subject  string    `json:"subject"`
	Timestamp string   `json:"timestamp"`
}

func (a *API) handleSESEmails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if a.sesStore == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	emails := a.sesStore.GetEmails()
	summaries := make([]SESEmailSummary, 0, len(emails))
	for i := len(emails) - 1; i >= 0; i-- {
		e := emails[i]
		summaries = append(summaries, SESEmailSummary{
			MessageId: e.MessageId,
			Source:    e.Source,
			To:        e.ToAddresses,
			Subject:   e.Subject,
			Timestamp: e.Timestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, summaries)
}

func (a *API) handleSESEmailByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/ses/emails/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	if a.sesStore == nil {
		http.NotFound(w, r)
		return
	}

	emails := a.sesStore.GetEmails()
	for _, e := range emails {
		if e.MessageId == id {
			writeJSON(w, http.StatusOK, e)
			return
		}
	}

	http.NotFound(w, r)
}

func (a *API) handleTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	topo := a.buildDynamicTopology()
	writeJSON(w, http.StatusOK, topo)
}

// ResourcesResponse is the response body for the /api/resources/:service endpoint.
type ResourcesResponse struct {
	Service   string      `json:"service"`
	Resources interface{} `json:"resources"`
}

// listActions maps service name → action used to enumerate resources.
// Empty string means the service uses REST-based routing with no Action parameter.
var listActions = map[string]string{
	"s3":             "", // REST GET /
	"dynamodb":       "ListTables",
	"sqs":            "ListQueues",
	"sns":            "ListTopics",
	"cognito-idp":    "ListUserPools",
	"lambda":         "", // REST GET /2015-03-31/functions
	"kms":            "ListKeys",
	"secretsmanager": "ListSecrets",
	"ssm":            "DescribeParameters",
	"ec2":            "DescribeVpcs",
	"rds":            "DescribeDBInstances",
	"ecs":            "ListClusters",
	"ecr":            "DescribeRepositories",
	"route53":        "", // REST GET /2013-04-01/hostedzone
	"monitoring":     "DescribeAlarms",
	"events":         "ListEventBuses",
	"states":         "ListStateMachines",
	"cloudformation": "ListStacks",
	"logs":           "DescribeLogGroups",
	"ses":            "ListIdentities",
	"kinesis":        "ListStreams",
	"firehose":       "ListDeliveryStreams",
	"sts":            "GetCallerIdentity",
}

// jsonServices is the set of services that use the X-Amz-Target / JSON protocol.
var jsonServices = map[string]bool{
	"dynamodb":       true,
	"kms":            true,
	"secretsmanager": true,
	"ssm":            true,
	"cognito-idp":    true,
	"ecs":            true,
	"ecr":            true,
	"events":         true,
	"states":         true,
	"kinesis":        true,
	"firehose":       true,
	"logs":           true,
}

// amzTargetPrefix maps service name → X-Amz-Target prefix (e.g. "DynamoDB_20120810").
var amzTargetPrefix = map[string]string{
	"dynamodb":       "DynamoDB_20120810",
	"kms":            "TrentService",
	"secretsmanager": "secretsmanager",
	"ssm":            "AmazonSSM",
	"cognito-idp":    "AWSCognitoIdentityProviderService",
	"ecs":            "AmazonEC2ContainerServiceV20141113",
	"ecr":            "AmazonEC2ContainerRegistry_V20150921",
	"events":         "AmazonEventBridgeV2",
	"states":         "AWSStepFunctions",
	"kinesis":        "Kinesis_20131202",
	"firehose":       "Firehose_20150804",
	"logs":           "Logs_20140328",
}

// restServices is the set of services that use REST path-based routing.
var restServices = map[string]bool{
	"s3":      true,
	"lambda":  true,
	"route53": true,
}

// handleResources handles GET /api/resources/:service — lists resources for a service
// by making an internal call to the service's HandleRequest method.
func (a *API) handleResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serviceName := strings.TrimPrefix(r.URL.Path, "/api/resources/")
	// Strip any trailing path segments — only the service name is accepted.
	if idx := strings.Index(serviceName, "/"); idx >= 0 {
		serviceName = serviceName[:idx]
	}
	if serviceName == "" {
		http.NotFound(w, r)
		return
	}

	svc, err := a.registry.Lookup(serviceName)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	action, actionKnown := listActions[serviceName]
	if !actionKnown {
		// Service is registered but we don't have a list action for it; return empty.
		writeJSON(w, http.StatusOK, ResourcesResponse{Service: serviceName, Resources: []interface{}{}})
		return
	}

	ctx, fakeReq := buildListRequestContext(a.cfg, serviceName, action)

	// For REST services, override the RawRequest path.
	if restServices[serviceName] {
		fakeReq = buildRESTRequest(serviceName)
		ctx.RawRequest = fakeReq
	}

	resp, svcErr := svc.HandleRequest(ctx)
	if svcErr != nil {
		// Return empty resource list on service errors rather than propagating AWS errors.
		writeJSON(w, http.StatusOK, ResourcesResponse{Service: serviceName, Resources: []interface{}{}})
		return
	}

	if resp == nil || resp.Body == nil {
		writeJSON(w, http.StatusOK, ResourcesResponse{Service: serviceName, Resources: []interface{}{}})
		return
	}

	// Marshal the response body to JSON. Regardless of whether the underlying
	// service uses XML or JSON protocol, the Body field is a Go struct that can
	// be JSON-encoded for the dashboard.
	writeJSON(w, http.StatusOK, ResourcesResponse{Service: serviceName, Resources: resp.Body})
}

// buildListRequestContext builds a service.RequestContext for the given list action.
// It also returns the *http.Request embedded in the context.
func buildListRequestContext(cfg *config.Config, serviceName, action string) (*service.RequestContext, *http.Request) {
	var fakeReq *http.Request

	if jsonServices[serviceName] {
		// JSON protocol: action is parsed from X-Amz-Target.
		fakeReq, _ = http.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("{}")))
		prefix := amzTargetPrefix[serviceName]
		if prefix == "" {
			prefix = serviceName
		}
		fakeReq.Header.Set("X-Amz-Target", prefix+"."+action)
		fakeReq.Header.Set("Content-Type", "application/x-amz-json-1.1")
	} else {
		// Query/form protocol: action is in the form body and ctx.Params.
		formBody := "Action=" + action
		if action != "" {
			fakeReq, _ = http.NewRequest(http.MethodPost, "/", strings.NewReader(formBody))
			fakeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			fakeReq, _ = http.NewRequest(http.MethodGet, "/", nil)
		}
	}

	params := map[string]string{}
	if action != "" {
		params["Action"] = action
	}

	ctx := &service.RequestContext{
		Action:     action,
		Region:     cfg.Region,
		AccountID:  cfg.AccountID,
		Service:    serviceName,
		Identity:   &service.CallerIdentity{IsRoot: true, AccountID: cfg.AccountID},
		Params:     params,
		Body:       []byte("{}"),
		RawRequest: fakeReq,
	}

	if !jsonServices[serviceName] && action != "" {
		ctx.Body = []byte("Action=" + action)
	}

	return ctx, fakeReq
}

// buildRESTRequest constructs a path-appropriate *http.Request for REST services.
func buildRESTRequest(serviceName string) *http.Request {
	var path string
	switch serviceName {
	case "s3":
		path = "/"
	case "lambda":
		path = "/2015-03-31/functions"
	case "route53":
		path = "/2013-04-01/hostedzone"
	default:
		path = "/"
	}
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	return req
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
