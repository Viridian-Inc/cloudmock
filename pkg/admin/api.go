package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
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

// TopologyNode describes a service in the topology graph.
type TopologyNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// TopologyEdge describes a connection between services.
type TopologyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

// TopologyResponse is the topology graph.
type TopologyResponse struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

func (a *API) handleTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build topology from registered services and known inter-service connections.
	svcs := a.registry.List()
	nodes := make([]TopologyNode, 0, len(svcs))
	svcSet := make(map[string]bool)
	for _, svc := range svcs {
		nodes = append(nodes, TopologyNode{
			ID:   svc.Name(),
			Name: svc.Name(),
			Type: "service",
		})
		svcSet[svc.Name()] = true
	}

	// Known AWS cross-service integrations.
	knownEdges := []TopologyEdge{
		{Source: "s3", Target: "sqs", Label: "event notification"},
		{Source: "s3", Target: "sns", Label: "event notification"},
		{Source: "s3", Target: "lambda", Label: "event notification"},
		{Source: "sns", Target: "sqs", Label: "subscription"},
		{Source: "sns", Target: "lambda", Label: "subscription"},
		{Source: "eventbridge", Target: "sqs", Label: "target"},
		{Source: "eventbridge", Target: "sns", Label: "target"},
		{Source: "eventbridge", Target: "lambda", Label: "target"},
		{Source: "dynamodb", Target: "lambda", Label: "streams"},
		{Source: "kinesis", Target: "lambda", Label: "event source"},
		{Source: "sqs", Target: "lambda", Label: "event source"},
		{Source: "apigateway", Target: "lambda", Label: "integration"},
		{Source: "cognito", Target: "lambda", Label: "triggers"},
		{Source: "ses", Target: "sns", Label: "notifications"},
		{Source: "cloudwatch", Target: "sns", Label: "alarm actions"},
	}

	edges := make([]TopologyEdge, 0)
	for _, e := range knownEdges {
		if svcSet[e.Source] && svcSet[e.Target] {
			edges = append(edges, e)
		}
	}

	writeJSON(w, http.StatusOK, TopologyResponse{Nodes: nodes, Edges: edges})
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
