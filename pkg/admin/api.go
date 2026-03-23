package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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

// IaCTopologyConfig holds the topology graph pushed from the IaC layer.
type IaCTopologyConfig struct {
	Nodes []TopologyNodeV2 `json:"nodes"`
	Edges []TopologyEdgeV2 `json:"edges"`
}

// API is the admin HTTP handler.
type API struct {
	cfg            *config.Config
	registry       *routing.Registry
	log            *gateway.RequestLog
	stats          *gateway.RequestStats
	broadcaster    *EventBroadcaster
	lambdaLogs     *lambda.LogBuffer
	iamEngine      *iam.Engine
	sesStore       *ses.Store
	traceStore     *gateway.TraceStore
	chaosEngine    *gateway.ChaosEngine
	iacTopology    *IaCTopologyConfig
	iacTopologyMu  sync.RWMutex
	sloEngine      *gateway.SLOEngine
	mux            *http.ServeMux
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
	a.mux.HandleFunc("/api/topology/config", a.handleTopologyConfig)
	a.mux.HandleFunc("/api/resources/", a.handleResources)
	a.mux.HandleFunc("/api/traces", a.handleTraces)
	a.mux.HandleFunc("/api/traces/", a.handleTraceByID)
	a.mux.HandleFunc("/api/metrics", a.handleMetrics)
	a.mux.HandleFunc("/api/metrics/timeline", a.handleMetricsTimeline)
	a.mux.HandleFunc("/api/slo", a.handleSLO)
	a.mux.HandleFunc("/api/blast-radius", a.handleBlastRadius)
	a.mux.HandleFunc("/api/tenants", a.handleTenants)
	a.mux.HandleFunc("/api/cost", a.handleCost)
	a.mux.HandleFunc("/api/compare", a.handleCompare)
	a.mux.HandleFunc("/api/chaos", a.handleChaos)
	a.mux.HandleFunc("/api/explain/", a.handleExplainRequest)
	a.mux.HandleFunc("/api/chaos/", a.handleChaosRule)

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

	q := r.URL.Query()
	limit := 100
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	filter := gateway.RequestFilter{
		Service:   q.Get("service"),
		Path:      q.Get("path"),
		Method:    q.Get("method"),
		CallerID:  q.Get("caller_id"),
		Action:    q.Get("action"),
		ErrorOnly: q.Get("error") == "true",
		TraceID:   q.Get("trace_id"),
		Limit:     limit,
	}

	entries := a.log.RecentFiltered(filter)
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
		id = strings.TrimSuffix(id, "/replay")
		entry := a.log.GetByID(id)
		if entry == nil {
			http.NotFound(w, r)
			return
		}
		result := a.replayRequest(entry)
		writeJSON(w, http.StatusOK, result)
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

// ReplayResult captures the result of replaying a captured request.
type ReplayResult struct {
	OriginalID     string `json:"original_id"`
	OriginalStatus int    `json:"original_status"`
	OriginalMs     float64 `json:"original_latency_ms"`
	ReplayStatus   int    `json:"replay_status"`
	ReplayMs       float64 `json:"replay_latency_ms"`
	ReplayBody     string `json:"replay_response_body"`
	Match          bool   `json:"match"` // status codes match
	LatencyDelta   float64 `json:"latency_delta_ms"` // replay - original
}

// replayRequest re-executes a captured request against the gateway.
func (a *API) replayRequest(entry *gateway.RequestEntry) ReplayResult {
	gwPort := a.cfg.Gateway.Port
	gwURL := fmt.Sprintf("http://localhost:%d%s", gwPort, entry.Path)

	var body io.Reader
	if entry.RequestBody != "" {
		body = strings.NewReader(entry.RequestBody)
	}

	req, err := http.NewRequest(entry.Method, gwURL, body)
	if err != nil {
		return ReplayResult{OriginalID: entry.ID, ReplayStatus: 0, ReplayBody: "failed to create request: " + err.Error()}
	}

	// Restore original headers
	for k, v := range entry.RequestHeaders {
		req.Header.Set(k, v)
	}
	// Mark as replay so it shows in the request log
	req.Header.Set("X-Cloudmock-Replay", entry.ID)

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	replayMs := float64(time.Since(start).Nanoseconds()) / 1e6
	if err != nil {
		return ReplayResult{
			OriginalID: entry.ID, OriginalStatus: entry.StatusCode, OriginalMs: entry.LatencyMs,
			ReplayStatus: 0, ReplayMs: replayMs, ReplayBody: "request failed: " + err.Error(),
		}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respStr := string(respBody)
	if len(respStr) > 10240 {
		respStr = respStr[:10240]
	}

	return ReplayResult{
		OriginalID:     entry.ID,
		OriginalStatus: entry.StatusCode,
		OriginalMs:     entry.LatencyMs,
		ReplayStatus:   resp.StatusCode,
		ReplayMs:       replayMs,
		ReplayBody:     respStr,
		Match:          resp.StatusCode == entry.StatusCode,
		LatencyDelta:   replayMs - entry.LatencyMs,
	}
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

// handleTopologyConfig accepts (PUT) or returns (GET) the IaC-derived topology config.
func (a *API) handleTopologyConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		var cfg IaCTopologyConfig
		if err := json.Unmarshal(body, &cfg); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		a.iacTopologyMu.Lock()
		a.iacTopology = &cfg
		a.iacTopologyMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "ok",
			"nodes":  len(cfg.Nodes),
			"edges":  len(cfg.Edges),
		})
	case http.MethodGet:
		a.iacTopologyMu.RLock()
		cfg := a.iacTopology
		a.iacTopologyMu.RUnlock()
		if cfg == nil {
			writeJSON(w, http.StatusOK, IaCTopologyConfig{})
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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

// SetTraceStore sets the trace store for the admin API.
func (a *API) SetTraceStore(ts *gateway.TraceStore) {
	a.traceStore = ts
}

// handleTraces returns recent traces.
func (a *API) handleTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if a.traceStore == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	svcFilter := r.URL.Query().Get("service")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	var hasErrorFilter *bool
	if ef := r.URL.Query().Get("error"); ef == "true" {
		v := true
		hasErrorFilter = &v
	} else if ef == "false" {
		v := false
		hasErrorFilter = &v
	}

	traces := a.traceStore.Recent(svcFilter, hasErrorFilter, limit)
	writeJSON(w, http.StatusOK, traces)
}

// handleTraceByID returns a single trace or its timeline.
func (a *API) handleTraceByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/traces/")
	parts := strings.SplitN(path, "/", 2)
	traceID := parts[0]

	if traceID == "" {
		http.NotFound(w, r)
		return
	}

	if a.traceStore == nil {
		http.NotFound(w, r)
		return
	}

	// /api/traces/:traceId/timeline
	if len(parts) == 2 && parts[1] == "timeline" {
		spans := a.traceStore.Timeline(traceID)
		if spans == nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, spans)
		return
	}

	trace := a.traceStore.Get(traceID)
	if trace == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

// SetSLOEngine sets the SLO engine for the admin API.
func (a *API) SetSLOEngine(engine *gateway.SLOEngine) {
	a.sloEngine = engine
}

// handleSLO returns the current SLO status or updates rules.
func (a *API) handleSLO(w http.ResponseWriter, r *http.Request) {
	if a.sloEngine == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"enabled": false})
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := a.sloEngine.Status()
		writeJSON(w, http.StatusOK, status)
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var rules []config.SLORule
		if err := json.Unmarshal(body, &rules); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		a.sloEngine.SetRules(rules)
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "rules": len(rules)})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBlastRadius computes which services would be affected if a given
// node fails. Traces upstream/downstream through the topology graph.
// GET /api/blast-radius?node=dynamodb:attendance
func (a *API) handleBlastRadius(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node")
	if nodeID == "" {
		http.Error(w, "node parameter required", http.StatusBadRequest)
		return
	}

	topo := a.buildDynamicTopology()

	// Build adjacency: both directions
	upstream := make(map[string][]string)   // target → sources
	downstream := make(map[string][]string) // source → targets
	for _, e := range topo.Edges {
		downstream[e.Source] = append(downstream[e.Source], e.Target)
		upstream[e.Target] = append(upstream[e.Target], e.Source)
	}

	// BFS downstream: what breaks if this node fails
	affected := bfsNodes(nodeID, upstream) // nodes that depend on this node
	dependsOn := bfsNodes(nodeID, downstream) // nodes this node depends on

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"node":       nodeID,
		"affected":   affected,
		"depends_on": dependsOn,
		"blast_radius": len(affected),
	})
}

// bfsNodes does a BFS from startID through the adjacency map.
func bfsNodes(startID string, adj map[string][]string) []string {
	visited := map[string]bool{startID: true}
	queue := []string{startID}
	var result []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adj[current] {
			if !visited[next] {
				visited[next] = true
				result = append(result, next)
				queue = append(queue, next)
			}
		}
	}
	return result
}

// handleTenants returns per-tenant request stats and filtering.
// GET /api/tenants — list all observed tenants with request counts
// GET /api/tenants?id=CALLER_ID — detail for a specific tenant
func (a *API) handleTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries := a.log.Recent("", 1000)

	tenantID := r.URL.Query().Get("id")
	if tenantID != "" {
		// Filter for specific tenant
		var tenantReqs []gateway.RequestEntry
		for _, e := range entries {
			if e.CallerID == tenantID {
				tenantReqs = append(tenantReqs, e)
			}
		}
		errorCount := 0
		var totalLatency float64
		services := make(map[string]int)
		for _, e := range tenantReqs {
			if e.StatusCode >= 400 {
				errorCount++
			}
			totalLatency += e.LatencyMs
			services[e.Service]++
		}
		avgLatency := 0.0
		if len(tenantReqs) > 0 {
			avgLatency = totalLatency / float64(len(tenantReqs))
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"tenant_id":    tenantID,
			"request_count": len(tenantReqs),
			"error_count":  errorCount,
			"error_rate":   float64(errorCount) / float64(max(len(tenantReqs), 1)),
			"avg_latency_ms": avgLatency,
			"services":     services,
			"requests":     tenantReqs,
		})
		return
	}

	// List all tenants
	type tenantSummary struct {
		ID          string  `json:"id"`
		Requests    int     `json:"requests"`
		Errors      int     `json:"errors"`
		ErrorRate   float64 `json:"error_rate"`
		AvgLatency  float64 `json:"avg_latency_ms"`
		LastSeen    string  `json:"last_seen"`
	}
	tenants := make(map[string]*tenantSummary)
	for _, e := range entries {
		if e.CallerID == "" {
			continue
		}
		t, ok := tenants[e.CallerID]
		if !ok {
			t = &tenantSummary{ID: e.CallerID}
			tenants[e.CallerID] = t
		}
		t.Requests++
		t.AvgLatency += e.LatencyMs
		if e.StatusCode >= 400 {
			t.Errors++
		}
		ts := e.Timestamp.Format(time.RFC3339)
		if ts > t.LastSeen {
			t.LastSeen = ts
		}
	}
	result := make([]tenantSummary, 0, len(tenants))
	for _, t := range tenants {
		if t.Requests > 0 {
			t.AvgLatency /= float64(t.Requests)
			t.ErrorRate = float64(t.Errors) / float64(t.Requests)
		}
		result = append(result, *t)
	}
	writeJSON(w, http.StatusOK, result)
}

// handleCost returns estimated AWS cost breakdown from recent request traffic.
// Prices based on us-east-1 on-demand pricing (approximate).
func (a *API) handleCost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries := a.log.Recent("", 1000)

	// Approximate AWS pricing per operation (us-east-1, USD)
	prices := map[string]float64{
		"dynamodb":       0.00000025, // $0.25 per million read units
		"s3":             0.0000004,  // $0.40 per million GET
		"sqs":            0.0000004,  // $0.40 per million requests
		"sns":            0.0000005,  // $0.50 per million publishes
		"lambda":         0.0000002,  // $0.20 per million invocations + compute
		"cognito-idp":    0.00000550, // $0.0055 per MAU (amortized)
		"ses":            0.0001,     // $0.10 per 1000 emails
		"secretsmanager": 0.00000005, // $0.05 per 10,000 API calls
		"kms":            0.000003,   // $0.03 per 10,000 requests
	}

	type serviceCost struct {
		Service    string  `json:"service"`
		Requests   int     `json:"requests"`
		CostUSD    float64 `json:"cost_usd"`
		PricePerOp float64 `json:"price_per_op_usd"`
	}

	svcCounts := make(map[string]int)
	for _, e := range entries {
		svcCounts[e.Service]++
	}

	var costs []serviceCost
	var totalCost float64
	for svc, count := range svcCounts {
		price := prices[svc]
		if price == 0 {
			price = 0.0000001 // default
		}
		cost := float64(count) * price
		totalCost += cost
		costs = append(costs, serviceCost{
			Service:    svc,
			Requests:   count,
			CostUSD:    cost,
			PricePerOp: price,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_cost_usd": totalCost,
		"request_count":  len(entries),
		"services":       costs,
		"note":           "Estimates based on us-east-1 on-demand pricing. Actual costs vary.",
	})
}

// handleCompare returns a before/after comparison for a service/action.
// GET /api/compare?service=dynamodb&action=Query&window=60
// Splits recent requests into two halves and compares metrics.
func (a *API) handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	service := r.URL.Query().Get("service")
	action := r.URL.Query().Get("action")

	entries := a.log.RecentFiltered(gateway.RequestFilter{
		Service: service,
		Action:  action,
		Limit:   500,
	})

	if len(entries) < 4 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "insufficient data", "count": fmt.Sprintf("%d", len(entries))})
		return
	}

	// Split into two halves: "before" (older) and "after" (newer)
	mid := len(entries) / 2
	after := entries[:mid]  // newer (entries are newest-first)
	before := entries[mid:] // older

	type windowStats struct {
		Count     int     `json:"count"`
		ErrorRate float64 `json:"error_rate"`
		P50Ms     float64 `json:"p50_ms"`
		P95Ms     float64 `json:"p95_ms"`
		P99Ms     float64 `json:"p99_ms"`
		AvgMs     float64 `json:"avg_ms"`
		From      string  `json:"from"`
		To        string  `json:"to"`
	}

	calcStats := func(reqs []gateway.RequestEntry) windowStats {
		if len(reqs) == 0 {
			return windowStats{}
		}
		var totalMs float64
		var errors int
		latencies := make([]float64, len(reqs))
		for i, r := range reqs {
			latencies[i] = r.LatencyMs
			totalMs += r.LatencyMs
			if r.StatusCode >= 400 {
				errors++
			}
		}
		explainSortFloat64s(latencies)
		return windowStats{
			Count:     len(reqs),
			ErrorRate: float64(errors) / float64(len(reqs)),
			P50Ms:     explainPercentile(latencies, 50),
			P95Ms:     explainPercentile(latencies, 95),
			P99Ms:     explainPercentile(latencies, 99),
			AvgMs:     totalMs / float64(len(reqs)),
			From:      reqs[len(reqs)-1].Timestamp.Format(time.RFC3339),
			To:        reqs[0].Timestamp.Format(time.RFC3339),
		}
	}

	beforeStats := calcStats(before)
	afterStats := calcStats(after)

	// Calculate deltas
	p50Delta := afterStats.P50Ms - beforeStats.P50Ms
	p99Delta := afterStats.P99Ms - beforeStats.P99Ms
	errDelta := afterStats.ErrorRate - beforeStats.ErrorRate

	regression := false
	if p99Delta > beforeStats.P99Ms*0.5 && beforeStats.P99Ms > 0 {
		regression = true // P99 increased by >50%
	}
	if errDelta > 0.05 {
		regression = true // error rate increased by >5%
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service":    service,
		"action":     action,
		"before":     beforeStats,
		"after":      afterStats,
		"p50_delta_ms": p50Delta,
		"p99_delta_ms": p99Delta,
		"error_delta":  errDelta,
		"regression":   regression,
	})
}

func max(a, b int) int {
	if a > b { return a }
	return b
}

// SetChaosEngine sets the chaos engine for the admin API to manage fault injection rules.
func (a *API) SetChaosEngine(engine *gateway.ChaosEngine) {
	a.chaosEngine = engine
}

// ChaosEngine returns the configured chaos engine.
func (a *API) ChaosEngine() *gateway.ChaosEngine {
	return a.chaosEngine
}

// handleChaos handles GET /api/chaos (list rules) and POST /api/chaos (create rule).
func (a *API) handleChaos(w http.ResponseWriter, r *http.Request) {
	if a.chaosEngine == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	switch r.Method {
	case http.MethodGet:
		rules := a.chaosEngine.Rules()
		if rules == nil {
			rules = []gateway.ChaosRule{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"rules":  rules,
			"active": a.chaosEngine.HasActiveRules(),
		})

	case http.MethodPost:
		var rule gateway.ChaosRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		created := a.chaosEngine.AddRule(rule)
		writeJSON(w, http.StatusCreated, created)

	case http.MethodDelete:
		// DELETE /api/chaos — disable all rules
		a.chaosEngine.DisableAll()
		writeJSON(w, http.StatusOK, map[string]string{"status": "all_disabled"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleChaosRule handles PUT /api/chaos/:id (update) and DELETE /api/chaos/:id (delete).
func (a *API) handleChaosRule(w http.ResponseWriter, r *http.Request) {
	if a.chaosEngine == nil {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/chaos/")
	// Handle /api/metrics/timeline specially
	if id == "timeline" || strings.HasPrefix(id, "timeline") {
		// This shouldn't happen because /api/metrics/ is handled separately,
		// but just in case, redirect to timeline handler.
		http.NotFound(w, r)
		return
	}

	if id == "" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var update gateway.ChaosRule
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		updated, ok := a.chaosEngine.UpdateRule(id, update)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, updated)

	case http.MethodDelete:
		if !a.chaosEngine.DeleteRule(id) {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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

// ExplainContext aggregates all data needed for AI analysis of a request.
type ExplainContext struct {
	Request        *gateway.RequestEntry   `json:"request"`
	Trace          *gateway.TraceContext    `json:"trace,omitempty"`
	Timeline       []gateway.TimelineSpan  `json:"timeline,omitempty"`
	SimilarRecent  []gateway.RequestEntry   `json:"similar_recent"`
	ServiceMetrics interface{}              `json:"service_metrics,omitempty"`
	Topology       *TopologyResponseV2      `json:"topology_context,omitempty"`
	Analysis       ExplainAnalysis          `json:"analysis"`
	Narrative      string                   `json:"narrative"`
}

// ExplainAnalysis contains pre-computed analysis hints.
type ExplainAnalysis struct {
	IsSlow       bool    `json:"is_slow"`
	IsError      bool    `json:"is_error"`
	P50Ms        float64 `json:"p50_ms"`
	P95Ms        float64 `json:"p95_ms"`
	P99Ms        float64 `json:"p99_ms"`
	LatencyRatio float64 `json:"latency_ratio"` // request latency / p50 (>2 = slow)
	ErrorRate    float64 `json:"error_rate"`     // recent error rate for this service
	SpanCount    int     `json:"span_count"`
	SlowestSpan  string  `json:"slowest_span,omitempty"`
	Anomalies    []string `json:"anomalies,omitempty"`
}

// handleExplainRequest returns AI-ready context for a specific request.
// GET /api/explain/{requestId}
func (a *API) handleExplainRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reqID := strings.TrimPrefix(r.URL.Path, "/api/explain/")
	if reqID == "" {
		http.Error(w, "request ID required", http.StatusBadRequest)
		return
	}

	// 1. Get the request
	entry := a.log.GetByID(reqID)
	if entry == nil {
		http.Error(w, "request not found", http.StatusNotFound)
		return
	}

	ctx := ExplainContext{
		Request: entry,
	}

	// 2. Get the trace + timeline
	if entry.TraceID != "" && a.traceStore != nil {
		ctx.Trace = a.traceStore.Get(entry.TraceID)
		ctx.Timeline = a.traceStore.Timeline(entry.TraceID)
	}

	// 3. Get recent similar requests (same service + action)
	similar := a.log.RecentFiltered(gateway.RequestFilter{
		Service: entry.Service,
		Action:  entry.Action,
		Limit:   20,
	})
	ctx.SimilarRecent = similar

	// 4. Compute analysis
	analysis := ExplainAnalysis{
		IsError:   entry.StatusCode >= 400,
		SpanCount: len(ctx.Timeline),
	}

	// Latency analysis from similar requests
	if len(similar) > 0 {
		latencies := make([]float64, len(similar))
		errorCount := 0
		for i, s := range similar {
			latencies[i] = s.LatencyMs
			if s.StatusCode >= 400 {
				errorCount++
			}
		}
		analysis.ErrorRate = float64(errorCount) / float64(len(similar))

		// Sort for percentiles
		explainSortFloat64s(latencies)
		analysis.P50Ms = explainPercentile(latencies, 50)
		analysis.P95Ms = explainPercentile(latencies, 95)
		analysis.P99Ms = explainPercentile(latencies, 99)

		if analysis.P50Ms > 0 {
			analysis.LatencyRatio = entry.LatencyMs / analysis.P50Ms
			analysis.IsSlow = analysis.LatencyRatio > 2.0
		}
	}

	// Find slowest span in trace
	if len(ctx.Timeline) > 0 {
		slowest := ctx.Timeline[0]
		for _, s := range ctx.Timeline[1:] {
			if s.DurationMs > slowest.DurationMs {
				slowest = s
			}
		}
		analysis.SlowestSpan = slowest.Service + "/" + slowest.Action
	}

	// Detect anomalies
	if analysis.IsSlow {
		analysis.Anomalies = append(analysis.Anomalies,
			fmt.Sprintf("Request latency (%.0fms) is %.1fx the p50 (%.0fms)",
				entry.LatencyMs, analysis.LatencyRatio, analysis.P50Ms))
	}
	if analysis.IsError && analysis.ErrorRate < 0.1 {
		analysis.Anomalies = append(analysis.Anomalies,
			fmt.Sprintf("This error is unusual — service error rate is only %.0f%%", analysis.ErrorRate*100))
	}
	if analysis.IsError && analysis.ErrorRate > 0.5 {
		analysis.Anomalies = append(analysis.Anomalies,
			fmt.Sprintf("Service is experiencing high error rate: %.0f%%", analysis.ErrorRate*100))
	}
	if len(ctx.Timeline) > 5 {
		analysis.Anomalies = append(analysis.Anomalies,
			fmt.Sprintf("High span count (%d) — request fans out across multiple services", len(ctx.Timeline)))
	}

	ctx.Analysis = analysis
	ctx.Narrative = buildNarrative(entry, &ctx, &analysis)
	writeJSON(w, http.StatusOK, ctx)
}

// buildNarrative generates a detailed text explanation of a request for debugging.
func buildNarrative(entry *gateway.RequestEntry, ctx *ExplainContext, a *ExplainAnalysis) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("## Request Analysis: %s %s\n\n", entry.Method, entry.Path))
	b.WriteString(fmt.Sprintf("**Request ID:** `%s`\n", entry.ID))
	b.WriteString(fmt.Sprintf("**Timestamp:** %s\n", entry.Timestamp.Format("2006-01-02 15:04:05.000 MST")))
	b.WriteString(fmt.Sprintf("**Service:** %s\n", entry.Service))
	b.WriteString(fmt.Sprintf("**Action:** %s\n", entry.Action))
	b.WriteString(fmt.Sprintf("**Status:** %d\n", entry.StatusCode))
	b.WriteString(fmt.Sprintf("**Latency:** %.2fms\n", entry.LatencyMs))
	if entry.TraceID != "" {
		b.WriteString(fmt.Sprintf("**Trace ID:** `%s`\n", entry.TraceID))
	}
	if entry.CallerID != "" {
		b.WriteString(fmt.Sprintf("**Caller:** %s\n", entry.CallerID))
	}
	b.WriteString("\n")

	// Status assessment
	b.WriteString("### Status\n\n")
	if a.IsError {
		b.WriteString(fmt.Sprintf("This request **failed** with HTTP %d.", entry.StatusCode))
		if entry.Error != "" {
			b.WriteString(fmt.Sprintf(" Error: `%s`", entry.Error))
		}
		b.WriteString("\n\n")
		if a.ErrorRate < 0.1 {
			b.WriteString(fmt.Sprintf("This is **unusual** — the recent error rate for %s/%s is only %.0f%%, suggesting this is an intermittent or new failure rather than a systemic issue.\n\n", entry.Service, entry.Action, a.ErrorRate*100))
		} else if a.ErrorRate > 0.5 {
			b.WriteString(fmt.Sprintf("**Warning:** The %s service is currently experiencing a %.0f%% error rate for this action. This appears to be a systemic issue, not an isolated failure.\n\n", entry.Service, a.ErrorRate*100))
		} else {
			b.WriteString(fmt.Sprintf("The current error rate for this action is %.0f%%.\n\n", a.ErrorRate*100))
		}
	} else {
		b.WriteString(fmt.Sprintf("Request completed **successfully** with HTTP %d.\n\n", entry.StatusCode))
	}

	// Latency analysis
	b.WriteString("### Latency Analysis\n\n")
	b.WriteString(fmt.Sprintf("| Percentile | Value |\n|---|---|\n"))
	b.WriteString(fmt.Sprintf("| This request | **%.2fms** |\n", entry.LatencyMs))
	b.WriteString(fmt.Sprintf("| P50 (median) | %.2fms |\n", a.P50Ms))
	b.WriteString(fmt.Sprintf("| P95 | %.2fms |\n", a.P95Ms))
	b.WriteString(fmt.Sprintf("| P99 | %.2fms |\n\n", a.P99Ms))

	if a.IsSlow {
		b.WriteString(fmt.Sprintf("**This request is slow** — it took %.1fx longer than the median (P50). ", a.LatencyRatio))
		if a.SlowestSpan != "" {
			b.WriteString(fmt.Sprintf("The bottleneck appears to be `%s`.", a.SlowestSpan))
		}
		b.WriteString("\n\n")
	} else if a.P50Ms > 0 {
		b.WriteString(fmt.Sprintf("Latency is **within normal range** at %.1fx the median.\n\n", a.LatencyRatio))
	}

	// Trace walkthrough
	if len(ctx.Timeline) > 0 {
		b.WriteString("### Execution Trace\n\n")
		b.WriteString(fmt.Sprintf("The request executed across **%d spans**:\n\n", len(ctx.Timeline)))
		b.WriteString("| # | Service | Action | Offset | Duration | Status | Depth |\n")
		b.WriteString("|---|---------|--------|--------|----------|--------|-------|\n")

		for i, span := range ctx.Timeline {
			statusStr := fmt.Sprintf("%d", span.StatusCode)
			if span.Error != "" {
				statusStr = fmt.Sprintf("%d \u274C", span.StatusCode)
			}
			indent := ""
			for j := 0; j < span.Depth; j++ {
				indent += "\u2514 "
			}
			b.WriteString(fmt.Sprintf("| %d | %s%s | %s | +%.1fms | %.2fms | %s | %d |\n",
				i+1, indent, span.Service, span.Action, span.StartOffsetMs, span.DurationMs, statusStr, span.Depth))
		}
		b.WriteString("\n")

		// Identify the critical path
		if len(ctx.Timeline) > 1 {
			slowest := ctx.Timeline[0]
			for _, s := range ctx.Timeline[1:] {
				if s.DurationMs > slowest.DurationMs {
					slowest = s
				}
			}
			b.WriteString(fmt.Sprintf("**Critical path:** The slowest span is `%s/%s` at %.2fms (%.0f%% of total request time).\n\n",
				slowest.Service, slowest.Action, slowest.DurationMs,
				(slowest.DurationMs/entry.LatencyMs)*100))
		}
	}

	// Request/response bodies
	if entry.RequestBody != "" {
		b.WriteString("### Request Body\n\n")
		b.WriteString("```json\n")
		b.WriteString(entry.RequestBody)
		if len(entry.RequestBody) > 0 && entry.RequestBody[len(entry.RequestBody)-1] != '\n' {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	}

	if entry.ResponseBody != "" {
		b.WriteString("### Response Body\n\n")
		b.WriteString("```json\n")
		b.WriteString(entry.ResponseBody)
		if len(entry.ResponseBody) > 0 && entry.ResponseBody[len(entry.ResponseBody)-1] != '\n' {
			b.WriteString("```\n\n")
		} else {
			b.WriteString("```\n\n")
		}
	}

	// Request headers
	if len(entry.RequestHeaders) > 0 {
		b.WriteString("### Request Headers\n\n")
		b.WriteString("| Header | Value |\n|---|---|\n")
		for k, v := range entry.RequestHeaders {
			if strings.HasPrefix(strings.ToLower(k), "authorization") {
				v = v[:min(20, len(v))] + "..."
			}
			b.WriteString(fmt.Sprintf("| %s | `%s` |\n", k, v))
		}
		b.WriteString("\n")
	}

	// Similar requests context
	if len(ctx.SimilarRecent) > 1 {
		b.WriteString("### Recent Baseline\n\n")
		b.WriteString(fmt.Sprintf("Based on %d recent similar requests (%s/%s):\n\n", len(ctx.SimilarRecent), entry.Service, entry.Action))
		errCount := 0
		for _, r := range ctx.SimilarRecent {
			if r.StatusCode >= 400 {
				errCount++
			}
		}
		b.WriteString(fmt.Sprintf("- **Success rate:** %.0f%%\n", float64(len(ctx.SimilarRecent)-errCount)/float64(len(ctx.SimilarRecent))*100))
		b.WriteString(fmt.Sprintf("- **Median latency:** %.2fms\n", a.P50Ms))
		b.WriteString(fmt.Sprintf("- **P99 latency:** %.2fms\n", a.P99Ms))
		b.WriteString("\n")
	}

	// Summary
	b.WriteString("### Summary\n\n")
	if len(a.Anomalies) > 0 {
		b.WriteString("**Findings:**\n")
		for _, anom := range a.Anomalies {
			b.WriteString(fmt.Sprintf("- \u26A0 %s\n", anom))
		}
	} else {
		b.WriteString("No anomalies detected. Request completed within normal parameters.\n")
	}

	return b.String()
}

// explainSortFloat64s sorts a slice of float64s in place.
func explainSortFloat64s(s []float64) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// explainPercentile returns the p-th percentile of a sorted slice.
func explainPercentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
