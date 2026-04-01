package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/neureaux/cloudmock/pkg/auth"
	"github.com/neureaux/cloudmock/pkg/audit"
	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/cost"
	errs "github.com/neureaux/cloudmock/pkg/errors"
	"github.com/neureaux/cloudmock/pkg/logstore"
	"github.com/neureaux/cloudmock/pkg/rum"
	"github.com/neureaux/cloudmock/pkg/dataplane"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/iac"
	"github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/incident"
	"github.com/neureaux/cloudmock/pkg/monitor"
	"github.com/neureaux/cloudmock/pkg/notify"
	"github.com/neureaux/cloudmock/pkg/plugin"
	"github.com/neureaux/cloudmock/pkg/profiling"
	"github.com/neureaux/cloudmock/pkg/regression"
	"github.com/neureaux/cloudmock/pkg/report"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/saas/clerk"
	saasstripe "github.com/neureaux/cloudmock/pkg/saas/stripe"
	"github.com/neureaux/cloudmock/pkg/saas/tenant"
	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/neureaux/cloudmock/pkg/tracecompare"
	"github.com/neureaux/cloudmock/pkg/traffic"
	"github.com/neureaux/cloudmock/pkg/uptime"
	"github.com/neureaux/cloudmock/pkg/webhook"
	"github.com/neureaux/cloudmock/services/lambda"
	"github.com/neureaux/cloudmock/services/ses"
)

// Version and BuildTime are set via ldflags at build time.
var Version = "dev"
var BuildTime = "unknown"

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
	Status    string          `json:"status"`
	Services  map[string]bool `json:"services"`
	DataPlane string          `json:"dataplane,omitempty"`
}

// SavedView represents a named filter preset that users can save and recall.
type SavedView struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Filters   map[string]string `json:"filters"`
	CreatedBy string            `json:"created_by"`
	CreatedAt string            `json:"created_at"`
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
	iacTopology      *IaCTopologyConfig
	iacTopologyMu    sync.RWMutex
	iacMicroservices []iac.MicroserviceDef
	deploys        []DeployEvent
	deploysMu      sync.RWMutex
	sloEngine      *gateway.SLOEngine
	views          []SavedView
	viewsMu        sync.RWMutex
	regressionEngine *regression.Engine
	traceComparer    *tracecompare.Comparer
	costEngine       *cost.Engine
	incidentService  *incident.Service
	profilingEngine  *profiling.Engine
	symbolizer       *profiling.Symbolizer
	auditLogger      audit.Logger
	userStore        auth.UserStore
	authSecret       []byte
	webhookDispatcher *webhook.Dispatcher
	reportGenerator   *report.Generator
	pluginManager     *plugin.Manager
	mux              *http.ServeMux
	dp               *dataplane.DataPlane
	prefsMu          sync.RWMutex
	prefs            map[string]map[string]json.RawMessage
	sourceServer     *SourceServer
	tenantStore      tenant.Store
	clerkWebhook     *clerk.WebhookHandler
	stripeWebhook    *saasstripe.WebhookHandler
	dashboards       []Dashboard
	dashboardsMu     sync.RWMutex
	rumEngine        *rum.Engine
	uptimeEngine     *uptime.Engine
	monitorService   *monitor.Service
	trafficEngine    *traffic.Engine
	errorStore       errs.ErrorStore
	logStore         logstore.LogStore
	notifyRouter     *notify.Router
	scm              *scmState
}

// SetRequestLog sets the direct in-memory request log and stats on the API.
// This is needed for topology edge enrichment even when DataPlane is used,
// because the DataPlane stores requests but the topology enrichment reads
// from the direct RequestLog.
func (a *API) SetRequestLog(log *gateway.RequestLog, stats *gateway.RequestStats) {
	a.log = log
	a.stats = stats
}

// SetSourceServer wires the source server for HTTP event ingestion.
func (a *API) SetSourceServer(ss *SourceServer) {
	a.sourceServer = ss
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

	a.mux.HandleFunc("/api/version", a.handleVersion)
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
	a.mux.HandleFunc("/api/metrics/query", a.handleMetricQuery)
	a.mux.HandleFunc("/api/dashboards", a.handleDashboards)
	a.mux.HandleFunc("/api/dashboards/", a.handleDashboardByID)
	a.mux.HandleFunc("/api/slo", a.handleSLO)
	a.mux.HandleFunc("/api/blast-radius", a.handleBlastRadius)
	a.mux.HandleFunc("/api/tenants", a.handleTenants)
	a.mux.HandleFunc("/api/tenants/export", a.handleTenantExport)
	a.mux.HandleFunc("/api/shadow", a.handleShadowTest)
	a.mux.HandleFunc("/api/cost", a.handleCost)
	a.mux.HandleFunc("/api/compare", a.handleCompare)
	a.mux.HandleFunc("/api/deploys", a.handleDeploys)
	a.mux.HandleFunc("/api/chaos", a.handleChaos)
	a.mux.HandleFunc("/api/explain/", a.handleExplainRequest)
	a.mux.HandleFunc("/api/chaos/", a.handleChaosRule)
	a.mux.HandleFunc("/api/views", a.handleViews)
	a.mux.HandleFunc("/api/regressions", a.handleRegressions)
	a.mux.HandleFunc("/api/regressions/", a.handleRegressions)
	a.mux.HandleFunc("/api/monitors", a.handleMonitors)
	a.mux.HandleFunc("/api/monitors/", a.handleMonitors)
	a.mux.HandleFunc("/api/alerts", a.handleAlerts)
	a.mux.HandleFunc("/api/alerts/", a.handleAlerts)
	a.mux.HandleFunc("/api/audit", a.handleAudit)
	a.mux.HandleFunc("/api/auth/login", a.handleAuthLogin)
	a.mux.HandleFunc("/api/auth/register", a.handleAuthRegister)
	a.mux.HandleFunc("/api/auth/me", a.handleAuthMe)
	a.mux.HandleFunc("/api/users", a.handleUsers)
	a.mux.HandleFunc("/api/users/", a.handleUserByID)
	a.mux.HandleFunc("/api/preferences", a.handlePreferences)
	a.mux.HandleFunc("/api/plugins", a.handlePlugins)
	a.mux.HandleFunc("/api/plugins/", a.handlePluginByName)
	a.mux.HandleFunc("/api/source/events", a.handleSourceEvents)
	a.mux.HandleFunc("/api/source/status", a.handleSourceStatus)

	// SaaS hosted-tier endpoints
	a.mux.HandleFunc("/api/saas/tenants", a.handleTenantsSaaS)
	a.mux.HandleFunc("/api/usage", a.handleUsage)
	a.mux.HandleFunc("/api/subscription", a.handleSubscription)
	a.mux.HandleFunc("/api/webhooks/clerk", a.handleClerkWebhook)
	a.mux.HandleFunc("/api/webhooks/stripe", a.handleStripeWebhook)

	// RUM (Real User Monitoring) endpoints
	a.mux.HandleFunc("/api/rum/events", a.handleRUMIngest)
	a.mux.HandleFunc("/api/rum/vitals", a.handleRUMVitals)
	a.mux.HandleFunc("/api/rum/pages", a.handleRUMPages)
	a.mux.HandleFunc("/api/rum/errors", a.handleRUMErrors)
	a.mux.HandleFunc("/api/rum/sessions", a.handleRUMSessions)
	a.mux.HandleFunc("/api/rum/clicks", a.handleRUMClicks)
	a.mux.HandleFunc("/api/rum/journeys/", a.handleRUMJourneys)

	// Uptime monitoring endpoints
	a.mux.HandleFunc("/api/uptime/checks", a.handleUptimeChecks)
	a.mux.HandleFunc("/api/uptime/checks/", a.handleUptimeCheckByID)
	a.mux.HandleFunc("/api/uptime/status", a.handleUptimeStatus)

	// Structured error tracking endpoints
	a.mux.HandleFunc("/api/errors", a.handleErrors)
	a.mux.HandleFunc("/api/errors/ingest", a.handleErrorIngest)
	a.mux.HandleFunc("/api/errors/", a.handleErrorByID)

	// SCM / GitHub integration endpoints
	a.mux.HandleFunc("/api/source/context", a.handleSourceContext)
	a.mux.HandleFunc("/api/source/suspects", a.handleSourceSuspects)
	a.mux.HandleFunc("/api/scm/config", a.handleSCMConfig)

	// Log management endpoints
	a.mux.HandleFunc("/api/logs", a.handleLogs)
	a.mux.HandleFunc("/api/logs/stream", a.handleLogStream)
	a.mux.HandleFunc("/api/logs/ingest", a.handleLogIngest)
	a.mux.HandleFunc("/api/logs/services", a.handleLogServices)
	a.mux.HandleFunc("/api/logs/levels", a.handleLogLevels)

	// Notification routing endpoints
	a.mux.HandleFunc("/api/notify/routes", a.handleNotifyRoutes)
	a.mux.HandleFunc("/api/notify/routes/", a.handleNotifyRouteByID)
	a.mux.HandleFunc("/api/notify/channels", a.handleNotifyChannels)
	a.mux.HandleFunc("/api/notify/test", a.handleNotifyTest)
	a.mux.HandleFunc("/api/notify/history", a.handleNotifyHistory)

	a.seedDefaultDashboard()

	return a
}

// NewWithDataPlane creates an admin API handler wired to the given registry,
// config, and DataPlane. When dp is non-nil, handlers use DataPlane stores
// for reads/writes instead of the legacy in-memory fields.
func NewWithDataPlane(cfg *config.Config, registry *routing.Registry, dp *dataplane.DataPlane) *API {
	a := &API{
		cfg:         cfg,
		registry:    registry,
		broadcaster: NewEventBroadcaster(),
		mux:         http.NewServeMux(),
		dp:          dp,
	}

	a.mux.HandleFunc("/api/version", a.handleVersion)
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
	a.mux.HandleFunc("/api/traces/compare", a.handleTraceCompare)
	a.mux.HandleFunc("/api/traces/", a.handleTraceByID)
	a.mux.HandleFunc("/api/metrics", a.handleMetrics)
	a.mux.HandleFunc("/api/metrics/timeline", a.handleMetricsTimeline)
	a.mux.HandleFunc("/api/metrics/query", a.handleMetricQuery)
	a.mux.HandleFunc("/api/dashboards", a.handleDashboards)
	a.mux.HandleFunc("/api/dashboards/", a.handleDashboardByID)
	a.mux.HandleFunc("/api/slo", a.handleSLO)
	a.mux.HandleFunc("/api/blast-radius", a.handleBlastRadius)
	a.mux.HandleFunc("/api/tenants", a.handleTenants)
	a.mux.HandleFunc("/api/tenants/export", a.handleTenantExport)
	a.mux.HandleFunc("/api/shadow", a.handleShadowTest)
	a.mux.HandleFunc("/api/cost", a.handleCost)
	a.mux.HandleFunc("/api/cost/routes", a.handleCostRoutes)
	a.mux.HandleFunc("/api/cost/tenants", a.handleCostTenants)
	a.mux.HandleFunc("/api/cost/trend", a.handleCostTrend)
	a.mux.HandleFunc("/api/compare", a.handleCompare)
	a.mux.HandleFunc("/api/deploys", a.handleDeploys)
	a.mux.HandleFunc("/api/chaos", a.handleChaos)
	a.mux.HandleFunc("/api/explain/", a.handleExplainRequest)
	a.mux.HandleFunc("/api/chaos/", a.handleChaosRule)
	a.mux.HandleFunc("/api/views", a.handleViews)
	a.mux.HandleFunc("/api/regressions", a.handleRegressions)
	a.mux.HandleFunc("/api/regressions/", a.handleRegressions)
	a.mux.HandleFunc("/api/incidents", a.handleIncidents)
	a.mux.HandleFunc("/api/incidents/", a.handleIncidents)
	a.mux.HandleFunc("/api/webhooks", a.handleWebhooks)
	a.mux.HandleFunc("/api/webhooks/", a.handleWebhooks)
	a.mux.HandleFunc("/api/monitors", a.handleMonitors)
	a.mux.HandleFunc("/api/monitors/", a.handleMonitors)
	a.mux.HandleFunc("/api/alerts", a.handleAlerts)
	a.mux.HandleFunc("/api/alerts/", a.handleAlerts)
	a.mux.HandleFunc("/api/profile/", a.handleProfile)
	a.mux.HandleFunc("/api/profiles", a.handleProfiles)
	a.mux.HandleFunc("/api/profiles/", a.handleProfiles)
	a.mux.HandleFunc("/api/sourcemaps", a.handleSourcemaps)
	a.mux.HandleFunc("/api/audit", a.handleAudit)
	a.mux.HandleFunc("/api/auth/login", a.handleAuthLogin)
	a.mux.HandleFunc("/api/auth/register", a.handleAuthRegister)
	a.mux.HandleFunc("/api/auth/me", a.handleAuthMe)
	a.mux.HandleFunc("/api/users", a.handleUsers)
	a.mux.HandleFunc("/api/users/", a.handleUserByID)
	a.mux.HandleFunc("/api/preferences", a.handlePreferences)
	a.mux.HandleFunc("/api/plugins", a.handlePlugins)
	a.mux.HandleFunc("/api/plugins/", a.handlePluginByName)
	a.mux.HandleFunc("/api/source/events", a.handleSourceEvents)
	a.mux.HandleFunc("/api/source/status", a.handleSourceStatus)

	// SaaS hosted-tier endpoints
	a.mux.HandleFunc("/api/saas/tenants", a.handleTenantsSaaS)
	a.mux.HandleFunc("/api/usage", a.handleUsage)
	a.mux.HandleFunc("/api/subscription", a.handleSubscription)
	a.mux.HandleFunc("/api/webhooks/clerk", a.handleClerkWebhook)
	a.mux.HandleFunc("/api/webhooks/stripe", a.handleStripeWebhook)

	// RUM (Real User Monitoring) endpoints
	a.mux.HandleFunc("/api/rum/events", a.handleRUMIngest)
	a.mux.HandleFunc("/api/rum/vitals", a.handleRUMVitals)
	a.mux.HandleFunc("/api/rum/pages", a.handleRUMPages)
	a.mux.HandleFunc("/api/rum/errors", a.handleRUMErrors)
	a.mux.HandleFunc("/api/rum/sessions", a.handleRUMSessions)
	a.mux.HandleFunc("/api/rum/clicks", a.handleRUMClicks)
	a.mux.HandleFunc("/api/rum/journeys/", a.handleRUMJourneys)

	// Uptime monitoring endpoints
	a.mux.HandleFunc("/api/uptime/checks", a.handleUptimeChecks)
	a.mux.HandleFunc("/api/uptime/checks/", a.handleUptimeCheckByID)
	a.mux.HandleFunc("/api/uptime/status", a.handleUptimeStatus)

	// Structured error tracking endpoints
	a.mux.HandleFunc("/api/errors", a.handleErrors)
	a.mux.HandleFunc("/api/errors/ingest", a.handleErrorIngest)
	a.mux.HandleFunc("/api/errors/", a.handleErrorByID)

	// SCM / GitHub integration endpoints
	a.mux.HandleFunc("/api/source/context", a.handleSourceContext)
	a.mux.HandleFunc("/api/source/suspects", a.handleSourceSuspects)
	a.mux.HandleFunc("/api/scm/config", a.handleSCMConfig)

	// Log management endpoints
	a.mux.HandleFunc("/api/logs", a.handleLogs)
	a.mux.HandleFunc("/api/logs/stream", a.handleLogStream)
	a.mux.HandleFunc("/api/logs/ingest", a.handleLogIngest)
	a.mux.HandleFunc("/api/logs/services", a.handleLogServices)
	a.mux.HandleFunc("/api/logs/levels", a.handleLogLevels)

	// Notification routing endpoints
	a.mux.HandleFunc("/api/notify/routes", a.handleNotifyRoutes)
	a.mux.HandleFunc("/api/notify/routes/", a.handleNotifyRouteByID)
	a.mux.HandleFunc("/api/notify/channels", a.handleNotifyChannels)
	a.mux.HandleFunc("/api/notify/test", a.handleNotifyTest)
	a.mux.HandleFunc("/api/notify/history", a.handleNotifyHistory)

	a.seedDefaultDashboard()

	return a
}

// Broadcaster returns the event broadcaster for use by middleware.
func (a *API) Broadcaster() *EventBroadcaster {
	return a.broadcaster
}

// SetMicroservices sets the IaC-extracted microservice definitions for topology.
func (a *API) SetMicroservices(ms []iac.MicroserviceDef) {
	a.iacMicroservices = ms
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	writeJSON(w, http.StatusOK, map[string]any{"status": "reset", "services": resetNames})
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	// Check dataplane connectivity when available.
	if a.dp != nil {
		dpStatus := "ok"
		if _, err := a.dp.Config.GetConfig(r.Context()); err != nil {
			dpStatus = "error"
			status = "degraded"
		}
		resp.DataPlane = dpStatus
		resp.Status = status
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *API) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"version":    Version,
		"build_time": BuildTime,
	})
}

// writeError writes a JSON error response with the given status code and message.
func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (a *API) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, a.cfg)
}

func (a *API) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.dp != nil && a.stats == nil {
		// DataPlane mode without legacy stats — return empty snapshot
		writeJSON(w, http.StatusOK, map[string]int64{})
		return
	}

	writeJSON(w, http.StatusOK, a.stats.Snapshot())
}

func (a *API) handleViews(w http.ResponseWriter, r *http.Request) {
	if a.dp != nil {
		switch r.Method {
		case http.MethodGet:
			views, err := a.dp.Config.ListViews(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, views)
		case http.MethodPost:
			var v dataplane.SavedView
			if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
				return
			}
			v.ID = fmt.Sprintf("view-%d", time.Now().UnixNano())
			v.CreatedAt = time.Now().UTC()
			if err := a.dp.Config.SaveView(r.Context(), v); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			a.auditLog(r.Context(), "view.saved", "view:"+v.ID, map[string]any{"name": v.Name})
			writeJSON(w, http.StatusCreated, v)
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				writeError(w, http.StatusBadRequest, "missing id query parameter")
				return
			}
			if err := a.dp.Config.DeleteView(r.Context(), id); err != nil {
				writeError(w, http.StatusNotFound, "view not found")
				return
			}
			a.auditLog(r.Context(), "view.deleted", "view:"+id, nil)
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.viewsMu.RLock()
		views := make([]SavedView, len(a.views))
		copy(views, a.views)
		a.viewsMu.RUnlock()
		writeJSON(w, http.StatusOK, views)

	case http.MethodPost:
		var v SavedView
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		v.ID = fmt.Sprintf("view-%d", time.Now().UnixNano())
		v.CreatedAt = time.Now().UTC().Format(time.RFC3339)

		a.viewsMu.Lock()
		if len(a.views) >= 50 {
			a.viewsMu.Unlock()
			writeError(w, http.StatusConflict, "maximum of 50 saved views reached")
			return
		}
		a.views = append(a.views, v)
		a.viewsMu.Unlock()
		a.auditLog(r.Context(), "view.saved", "view:"+v.ID, map[string]any{"name": v.Name})
		writeJSON(w, http.StatusCreated, v)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "missing id query parameter")
			return
		}
		a.viewsMu.Lock()
		found := false
		for i, v := range a.views {
			if v.ID == id {
				a.views = append(a.views[:i], a.views[i+1:]...)
				found = true
				break
			}
		}
		a.viewsMu.Unlock()
		if !found {
			writeError(w, http.StatusNotFound, "view not found")
			return
		}
		a.auditLog(r.Context(), "view.deleted", "view:"+id, nil)
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	limit := 100
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	// Default to "app" level to hide infrastructure (AWS SDK) traffic.
	// Use level=all to see everything, level=infra for only AWS calls.
	level := q.Get("level")
	if level == "" {
		level = "app"
	}
	if level == "all" {
		level = ""
	}

	var minLatency, maxLatency float64
	if v := q.Get("min_latency_ms"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minLatency = f
		}
	}
	if v := q.Get("max_latency_ms"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			maxLatency = f
		}
	}

	var from, to time.Time
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		}
	}

	if a.dp != nil {
		dpFilter := dataplane.RequestFilter{
			Service:      q.Get("service"),
			Path:         q.Get("path"),
			Method:       q.Get("method"),
			CallerID:     q.Get("caller_id"),
			Action:       q.Get("action"),
			ErrorOnly:    q.Get("error") == "true",
			TraceID:      q.Get("trace_id"),
			Level:        level,
			Limit:        limit,
			TenantID:     q.Get("tenant_id"),
			OrgID:        q.Get("org_id"),
			UserID:       q.Get("user_id"),
			MinLatencyMs: minLatency,
			MaxLatencyMs: maxLatency,
			From:         from,
			To:           to,
		}
		entries, err := a.dp.Requests.Query(r.Context(), dpFilter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, entries)
		return
	}

	filter := gateway.RequestFilter{
		Service:      q.Get("service"),
		Path:         q.Get("path"),
		Method:       q.Get("method"),
		CallerID:     q.Get("caller_id"),
		Action:       q.Get("action"),
		ErrorOnly:    q.Get("error") == "true",
		TraceID:      q.Get("trace_id"),
		Level:        level,
		Limit:        limit,
		TenantID:     q.Get("tenant_id"),
		OrgID:        q.Get("org_id"),
		UserID:       q.Get("user_id"),
		MinLatencyMs: minLatency,
		MaxLatencyMs: maxLatency,
		From:         from,
		To:           to,
	}

	entries := a.log.RecentFiltered(filter)
	writeJSON(w, http.StatusOK, entries)
}

// handleStream is the SSE endpoint that pushes real-time events to the dashboard.
func (a *API) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
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
		// Replay always uses the legacy log (needs gateway.RequestEntry)
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.dp != nil {
		entry, err := a.dp.Requests.GetByID(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, entry)
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusBadRequest, "invalid request body")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
			writeError(w, http.StatusBadRequest, "failed to read body")
			return
		}
		var cfg IaCTopologyConfig
		if err := json.Unmarshal(body, &cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		a.iacTopologyMu.Lock()
		a.iacTopology = &cfg
		a.iacTopologyMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ResourcesResponse is the response body for the /api/resources/:service endpoint.
type ResourcesResponse struct {
	Service   string      `json:"service"`
	Resources any `json:"resources"`
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeJSON(w, http.StatusOK, ResourcesResponse{Service: serviceName, Resources: []any{}})
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
		writeJSON(w, http.StatusOK, ResourcesResponse{Service: serviceName, Resources: []any{}})
		return
	}

	if resp == nil || resp.Body == nil {
		writeJSON(w, http.StatusOK, ResourcesResponse{Service: serviceName, Resources: []any{}})
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	if a.dp != nil {
		summaries, err := a.dp.Traces.Search(r.Context(), dataplane.TraceFilter{
			Service:  svcFilter,
			HasError: hasErrorFilter,
			Limit:    limit,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, summaries)
		return
	}

	if a.traceStore == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	traces := a.traceStore.Recent(svcFilter, hasErrorFilter, limit)
	writeJSON(w, http.StatusOK, traces)
}

// handleTraceByID returns a single trace or its timeline.
func (a *API) handleTraceByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/traces/")
	parts := strings.SplitN(path, "/", 2)
	traceID := parts[0]

	if traceID == "" {
		http.NotFound(w, r)
		return
	}

	if a.dp != nil {
		// /api/traces/:traceId/timeline
		if len(parts) == 2 && parts[1] == "timeline" {
			spans, err := a.dp.Traces.Timeline(r.Context(), traceID)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, http.StatusOK, spans)
			return
		}

		trace, err := a.dp.Traces.Get(r.Context(), traceID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, trace)
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

// AdminAuthMiddleware protects admin endpoints with API key authentication.
// Checks X-Admin-Key header or ?key= query param against the configured key.
// Health and stream endpoints are excluded to allow monitoring.
func AdminAuthMiddleware(next http.Handler, apiKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks and SSE stream
		if r.URL.Path == "/api/health" || r.URL.Path == "/api/stream" {
			next.ServeHTTP(w, r)
			return
		}
		key := r.Header.Get("X-Admin-Key")
		if key == "" {
			key = r.URL.Query().Get("key")
		}
		if key != apiKey {
			writeError(w, http.StatusUnauthorized, "Invalid or missing X-Admin-Key header")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetSLOEngine sets the SLO engine for the admin API.
func (a *API) SetSLOEngine(engine *gateway.SLOEngine) {
	a.sloEngine = engine
}

// handleSLO returns the current SLO status or updates rules.
func (a *API) handleSLO(w http.ResponseWriter, r *http.Request) {
	if a.dp != nil {
		switch r.Method {
		case http.MethodGet:
			status, err := a.dp.SLO.Status(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			rules, _ := a.dp.SLO.Rules(r.Context())
			writeJSON(w, http.StatusOK, map[string]any{
				"windows": status.Windows,
				"healthy": status.Healthy,
				"alerts":  status.Alerts,
				"rules":   rules,
			})
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeError(w, http.StatusBadRequest, "bad request")
				return
			}
			var rules []config.SLORule
			if err := json.Unmarshal(body, &rules); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
				return
			}
			if err := a.dp.SLO.SetRules(r.Context(), rules); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			a.auditLog(r.Context(), "slo.rules.updated", "slo:config", map[string]any{"rule_count": len(rules)})
			writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "rules": len(rules)})
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if a.sloEngine == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := a.sloEngine.Status()
		writeJSON(w, http.StatusOK, status)
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad request")
			return
		}
		var rules []config.SLORule
		if err := json.Unmarshal(body, &rules); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		a.sloEngine.SetRules(rules)
		a.auditLog(r.Context(), "slo.rules.updated", "slo:config", map[string]any{"rule_count": len(rules)})
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "rules": len(rules)})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleBlastRadius computes which services would be affected if a given
// node fails. Traces upstream/downstream through the topology graph.
// GET /api/blast-radius?node=dynamodb:attendance
func (a *API) handleBlastRadius(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "node parameter required")
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

	writeJSON(w, http.StatusOK, map[string]any{
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeJSON(w, http.StatusOK, map[string]any{
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	writeJSON(w, http.StatusOK, map[string]any{
		"total_cost_usd": totalCost,
		"request_count":  len(entries),
		"services":       costs,
		"note":           "Estimates based on us-east-1 on-demand pricing. Actual costs vary.",
	})
}

// SetCostEngine wires the cost engine to the admin API.
func (a *API) SetCostEngine(engine *cost.Engine) {
	a.costEngine = engine
}

// parseDuration parses duration strings including "d" suffix for days.
// Go's time.ParseDuration does not support days.
func parseDuration(s string, fallback time.Duration) time.Duration {
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

// handleCostRoutes returns costs aggregated by service+method+path.
// GET /api/cost/routes?limit=20
func (a *API) handleCostRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.costEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "cost engine not available")
		return
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}
	results, err := a.costEngine.ByRoute(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// handleCostTenants returns costs aggregated by tenant ID.
// GET /api/cost/tenants?limit=20
func (a *API) handleCostTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.costEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "cost engine not available")
		return
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}
	results, err := a.costEngine.ByTenant(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// handleCostTrend returns cost aggregated into time buckets over a window.
// GET /api/cost/trend?window=24h&bucket=1h
func (a *API) handleCostTrend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.costEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "cost engine not available")
		return
	}
	window := parseDuration(r.URL.Query().Get("window"), 24*time.Hour)
	bucket := parseDuration(r.URL.Query().Get("bucket"), time.Hour)
	results, err := a.costEngine.Trend(r.Context(), window, bucket)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// handleCompare returns a before/after comparison for a service/action.
// GET /api/compare?service=dynamodb&action=Query&window=60
// Splits recent requests into two halves and compares metrics.
func (a *API) handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	writeJSON(w, http.StatusOK, map[string]any{
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

// DeployEvent records a deployment for change correlation.
type DeployEvent struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Commit    string `json:"commit"`
	Author    string `json:"author"`
	Message   string `json:"message"`
	Branch    string `json:"branch"`
	PR        string `json:"pr,omitempty"`
}

// handleDeploys manages deploy events for change intelligence.
// GET /api/deploys — list recent deploys
// POST /api/deploys — record a new deploy
func (a *API) handleDeploys(w http.ResponseWriter, r *http.Request) {
	if a.dp != nil {
		switch r.Method {
		case http.MethodGet:
			deploys, err := a.dp.Config.ListDeploys(r.Context(), dataplane.DeployFilter{Limit: 100})
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, deploys)
		case http.MethodPost:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeError(w, http.StatusBadRequest, "bad request")
				return
			}
			var deploy dataplane.DeployEvent
			if err := json.Unmarshal(body, &deploy); err != nil {
				writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
				return
			}
			if deploy.DeployedAt.IsZero() {
				deploy.DeployedAt = time.Now()
			}
			if deploy.ID == "" {
				deploy.ID = fmt.Sprintf("deploy-%d", time.Now().UnixNano())
			}
			if err := a.dp.Config.AddDeploy(r.Context(), deploy); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if a.regressionEngine != nil {
				a.regressionEngine.OnDeploy(deploy)
			}
			a.auditLog(r.Context(), "deploy.created", "deploy:"+deploy.ID, map[string]any{"service": deploy.Service})
			writeJSON(w, http.StatusCreated, deploy)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.deploysMu.RLock()
		result := make([]DeployEvent, len(a.deploys))
		copy(result, a.deploys)
		a.deploysMu.RUnlock()
		writeJSON(w, http.StatusOK, result)

	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad request")
			return
		}
		var deploy DeployEvent
		if err := json.Unmarshal(body, &deploy); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if deploy.Timestamp == "" {
			deploy.Timestamp = time.Now().Format(time.RFC3339)
		}
		if deploy.ID == "" {
			deploy.ID = fmt.Sprintf("deploy-%d", time.Now().UnixNano())
		}

		a.deploysMu.Lock()
		a.deploys = append(a.deploys, deploy)
		if len(a.deploys) > 100 {
			a.deploys = a.deploys[len(a.deploys)-100:]
		}
		a.deploysMu.Unlock()

		if a.regressionEngine != nil {
			ts, _ := time.Parse(time.RFC3339, deploy.Timestamp)
			if ts.IsZero() {
				ts = time.Now()
			}
			a.regressionEngine.OnDeploy(dataplane.DeployEvent{
				ID:          deploy.ID,
				Service:     deploy.Service,
				CommitSHA:   deploy.Commit,
				Author:      deploy.Author,
				Description: deploy.Message,
				DeployedAt:  ts,
			})
		}

		a.auditLog(r.Context(), "deploy.created", "deploy:"+deploy.ID, map[string]any{"service": deploy.Service})
		writeJSON(w, http.StatusCreated, deploy)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleTenantExport exports per-tenant report as CSV.
// GET /api/tenants/export?format=csv
func (a *API) handleTenantExport(w http.ResponseWriter, r *http.Request) {
	entries := a.log.Recent("", 1000)

	type tenantRow struct {
		ID       string
		Requests int
		Errors   int
		AvgMs    float64
	}
	tenants := make(map[string]*tenantRow)
	for _, e := range entries {
		if e.CallerID == "" {
			continue
		}
		t, ok := tenants[e.CallerID]
		if !ok {
			t = &tenantRow{ID: e.CallerID}
			tenants[e.CallerID] = t
		}
		t.Requests++
		t.AvgMs += e.LatencyMs
		if e.StatusCode >= 400 {
			t.Errors++
		}
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=tenant-report.csv")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("tenant_id,requests,errors,error_rate,avg_latency_ms\n"))
	for _, t := range tenants {
		if t.Requests > 0 {
			t.AvgMs /= float64(t.Requests)
		}
		errRate := float64(t.Errors) / float64(max(t.Requests, 1))
		line := fmt.Sprintf("%s,%d,%d,%.4f,%.2f\n", t.ID, t.Requests, t.Errors, errRate, t.AvgMs)
		_, _ = w.Write([]byte(line))
	}
}

// handleShadowTest replays recent traffic against a target URL for synthetic testing.
// POST /api/shadow {"target": "http://localhost:3203", "service": "bff", "limit": 10}
func (a *API) handleShadowTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		Target  string `json:"target"`
		Service string `json:"service"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target URL required")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Get recent requests to replay
	entries := a.log.RecentFiltered(gateway.RequestFilter{
		Service: req.Service,
		Limit:   req.Limit,
		Level:   "app",
	})

	type shadowResult struct {
		OriginalID     string  `json:"original_id"`
		Method         string  `json:"method"`
		Path           string  `json:"path"`
		OriginalStatus int     `json:"original_status"`
		ShadowStatus   int     `json:"shadow_status"`
		OriginalMs     float64 `json:"original_ms"`
		ShadowMs       float64 `json:"shadow_ms"`
		Match          bool    `json:"match"`
		Error          string  `json:"error,omitempty"`
	}

	var results []shadowResult
	client := &http.Client{Timeout: 10 * time.Second}

	for _, entry := range entries {
		targetURL := req.Target + entry.Path
		var reqBody io.Reader
		if entry.RequestBody != "" {
			reqBody = strings.NewReader(entry.RequestBody)
		}
		httpReq, err := http.NewRequest(entry.Method, targetURL, reqBody)
		if err != nil {
			results = append(results, shadowResult{OriginalID: entry.ID, Error: err.Error()})
			continue
		}
		for k, v := range entry.RequestHeaders {
			httpReq.Header.Set(k, v)
		}
		httpReq.Header.Set("X-Cloudmock-Shadow", "true")

		start := time.Now()
		resp, err := client.Do(httpReq)
		shadowMs := float64(time.Since(start).Nanoseconds()) / 1e6

		if err != nil {
			results = append(results, shadowResult{
				OriginalID: entry.ID, Method: entry.Method, Path: entry.Path,
				OriginalStatus: entry.StatusCode, OriginalMs: entry.LatencyMs,
				ShadowMs: shadowMs, Error: err.Error(),
			})
			continue
		}
		resp.Body.Close()

		results = append(results, shadowResult{
			OriginalID:     entry.ID,
			Method:         entry.Method,
			Path:           entry.Path,
			OriginalStatus: entry.StatusCode,
			ShadowStatus:   resp.StatusCode,
			OriginalMs:     entry.LatencyMs,
			ShadowMs:       shadowMs,
			Match:          resp.StatusCode == entry.StatusCode,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"target":  req.Target,
		"count":   len(results),
		"results": results,
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

// SetRegressionEngine sets the regression detection engine for the admin API.
func (a *API) SetRegressionEngine(engine *regression.Engine) {
	a.regressionEngine = engine
}

// SetTraceComparer sets the trace comparer for the /api/traces/compare endpoint.
func (a *API) SetTraceComparer(tc *tracecompare.Comparer) {
	a.traceComparer = tc
}

// SetIncidentService sets the incident service for the admin API.
func (a *API) SetIncidentService(svc *incident.Service) {
	a.incidentService = svc
}

// SetReportGenerator sets the incident report generator for the admin API.
func (a *API) SetReportGenerator(g *report.Generator) {
	a.reportGenerator = g
}

// SetWebhookDispatcher sets the webhook dispatcher for the admin API.
func (a *API) SetWebhookDispatcher(d *webhook.Dispatcher) {
	a.webhookDispatcher = d
}

// SetMonitorService sets the monitor service for the admin API.
func (a *API) SetMonitorService(svc *monitor.Service) {
	a.monitorService = svc
}

// SetNotificationRouter sets the notification router for alert routing.
func (a *API) SetNotificationRouter(nr *notify.Router) {
	a.notifyRouter = nr
}

// SetPluginManager sets the plugin manager for the admin API to expose plugin info.
func (a *API) SetPluginManager(pm *plugin.Manager) {
	a.pluginManager = pm
}

// handlePlugins serves GET /api/plugins — lists all registered plugins.
func (a *API) handlePlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.pluginManager == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}
	writeJSON(w, http.StatusOK, a.pluginManager.List())
}

// handlePluginByName serves GET /api/plugins/{name}/health — plugin health check.
func (a *API) handlePluginByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.pluginManager == nil {
		writeError(w, http.StatusNotFound, "plugin system not enabled")
		return
	}

	// Parse /api/plugins/{name} or /api/plugins/{name}/health
	path := strings.TrimPrefix(r.URL.Path, "/api/plugins/")
	parts := strings.SplitN(path, "/", 2)
	name := parts[0]

	if len(parts) == 2 && parts[1] == "health" {
		results := a.pluginManager.HealthCheckAll(r.Context())
		if result, ok := results[name]; ok {
			writeJSON(w, http.StatusOK, result)
			return
		}
		writeError(w, http.StatusNotFound, fmt.Sprintf("plugin %q not found", name))
		return
	}

	// Return plugin info
	for _, info := range a.pluginManager.List() {
		if info.Name == name {
			writeJSON(w, http.StatusOK, info)
			return
		}
	}
	writeError(w, http.StatusNotFound, fmt.Sprintf("plugin %q not found", name))
}

// SetProfilingEngine sets the profiling engine for the admin API.
func (a *API) SetProfilingEngine(e *profiling.Engine) {
	a.profilingEngine = e
}

// SetSymbolizer sets the source-map symbolizer for the admin API.
func (a *API) SetSymbolizer(s *profiling.Symbolizer) {
	a.symbolizer = s
}

// SetAuditLogger sets the audit logger for recording mutating API actions.
func (a *API) SetAuditLogger(l audit.Logger) {
	a.auditLogger = l
}

// auditLog records an audit entry if the audit logger is configured.
// It is a fire-and-forget helper — errors are silently ignored.
func (a *API) auditLog(ctx context.Context, action, resource string, details map[string]any) {
	if a.auditLogger == nil {
		return
	}
	_ = a.auditLogger.Log(ctx, audit.Entry{
		Actor:    "system",
		Action:   action,
		Resource: resource,
		Details:  details,
	})
}

// handleAudit serves GET /api/audit — returns recent audit log entries.
func (a *API) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.auditLogger == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	filter := audit.Filter{
		Actor:    r.URL.Query().Get("actor"),
		Action:   r.URL.Query().Get("action"),
		Resource: r.URL.Query().Get("resource"),
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	entries, err := a.auditLogger.Query(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []audit.Entry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (a *API) handleTraceCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.traceComparer == nil {
		writeError(w, http.StatusServiceUnavailable, "trace comparison not available")
		return
	}

	traceA := r.URL.Query().Get("a")
	if traceA == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: a")
		return
	}

	baseline := r.URL.Query().Get("baseline") == "true"
	traceB := r.URL.Query().Get("b")

	if !baseline && traceB == "" {
		writeError(w, http.StatusBadRequest, "must provide parameter b or baseline=true")
		return
	}

	ctx := r.Context()
	var result *tracecompare.TraceComparison
	var err error

	if baseline {
		result, err = a.traceComparer.CompareBaseline(ctx, traceA)
	} else {
		result, err = a.traceComparer.Compare(ctx, traceA, traceB)
	}

	if errors.Is(err, dataplane.ErrNotFound) {
		writeError(w, http.StatusNotFound, "trace not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleRegressions handles regression API endpoints:
//
//	GET  /api/regressions          — list regressions with optional filters
//	GET  /api/regressions/{id}     — get a single regression by ID
//	POST /api/regressions/{id}/dismiss — dismiss a regression
func (a *API) handleRegressions(w http.ResponseWriter, r *http.Request) {
	if a.regressionEngine == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	// Strip the base prefix to determine sub-path.
	path := strings.TrimPrefix(r.URL.Path, "/api/regressions")
	path = strings.TrimPrefix(path, "/")

	switch {
	// POST /api/regressions/{id}/dismiss
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/dismiss"):
		id := strings.TrimSuffix(path, "/dismiss")
		if id == "" {
			writeError(w, http.StatusBadRequest, "missing regression id")
			return
		}
		if err := a.regressionEngine.Store().UpdateStatus(r.Context(), id, "dismissed"); err != nil {
			if err == regression.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.auditLog(r.Context(), "regression.dismissed", "regression:"+id, nil)
		writeJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})

	// GET /api/regressions/{id}
	case r.Method == http.MethodGet && path != "":
		reg, err := a.regressionEngine.Store().Get(r.Context(), path)
		if err != nil {
			if err == regression.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, reg)

	// GET /api/regressions
	case r.Method == http.MethodGet:
		filter := regression.RegressionFilter{
			Service:  r.URL.Query().Get("service"),
			DeployID: r.URL.Query().Get("deploy_id"),
			Severity: regression.Severity(r.URL.Query().Get("severity")),
			Status:   r.URL.Query().Get("status"),
		}
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				filter.Limit = l
			}
		}
		results, err := a.regressionEngine.Store().List(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []regression.Regression{}
		}
		writeJSON(w, http.StatusOK, results)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
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
		writeJSON(w, http.StatusOK, map[string]any{
			"rules":  rules,
			"active": a.chaosEngine.HasActiveRules(),
		})

	case http.MethodPost:
		var rule gateway.ChaosRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		created := a.chaosEngine.AddRule(rule)
		writeJSON(w, http.StatusCreated, created)

	case http.MethodDelete:
		// DELETE /api/chaos — disable all rules
		a.chaosEngine.DisableAll()
		writeJSON(w, http.StatusOK, map[string]string{"status": "all_disabled"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
			writeError(w, http.StatusBadRequest, "invalid request body")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
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
	ServiceMetrics any              `json:"service_metrics,omitempty"`
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
// GET  /api/explain/{requestId}
// POST /api/explain/  with JSON body {"request_id": "..."}
func (a *API) handleExplainRequest(w http.ResponseWriter, r *http.Request) {
	var reqID string

	switch r.Method {
	case http.MethodGet:
		reqID = strings.TrimPrefix(r.URL.Path, "/api/explain/")
	case http.MethodPost:
		var body struct {
			RequestID string `json:"request_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		reqID = body.RequestID
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if reqID == "" {
		writeError(w, http.StatusBadRequest, "request ID required")
		return
	}

	// 1. Get the request
	entry := a.log.GetByID(reqID)
	if entry == nil {
		writeError(w, http.StatusNotFound, "request not found")
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

	// Check for recent deploys near the request time
	a.deploysMu.RLock()
	for _, d := range a.deploys {
		deployTime, err := time.Parse(time.RFC3339, d.Timestamp)
		if err != nil {
			continue
		}
		diff := entry.Timestamp.Sub(deployTime)
		if diff > -5*time.Minute && diff < 10*time.Minute {
			analysis.Anomalies = append(analysis.Anomalies,
				fmt.Sprintf("Deploy detected near this request: %s by %s (%s) — commit %s",
					d.Service, d.Author, d.Message, d.Commit))
		}
	}
	a.deploysMu.RUnlock()

	ctx.Analysis = analysis
	ctx.Narrative = buildNarrative(entry, &ctx, &analysis)
	writeJSON(w, http.StatusOK, ctx)
}

// serviceDescription maps AWS service names to human-readable descriptions.
var serviceDescription = map[string]string{
	"dynamodb":       "DynamoDB (NoSQL database)",
	"s3":             "S3 (object storage)",
	"sqs":            "SQS (message queue)",
	"sns":            "SNS (pub/sub messaging)",
	"lambda":         "Lambda (serverless compute)",
	"cognito-idp":    "Cognito (authentication)",
	"ses":            "SES (email service)",
	"secretsmanager": "Secrets Manager (credential store)",
	"kms":            "KMS (key management)",
	"iam":            "IAM (identity & access)",
	"sts":            "STS (security tokens)",
	"events":         "EventBridge (event bus)",
	"logs":           "CloudWatch Logs",
	"monitoring":     "CloudWatch Metrics",
}

// actionDescription maps service+action to plain English.
var actionDescription = map[string]string{
	"dynamodb:Query":           "queried a DynamoDB table using a key condition expression",
	"dynamodb:GetItem":         "fetched a single item from DynamoDB by primary key",
	"dynamodb:PutItem":         "wrote an item to DynamoDB",
	"dynamodb:UpdateItem":      "updated an existing DynamoDB item",
	"dynamodb:DeleteItem":      "deleted an item from DynamoDB",
	"dynamodb:Scan":            "performed a full table scan on DynamoDB (expensive operation)",
	"dynamodb:BatchGetItem":    "fetched multiple items from DynamoDB in a batch",
	"dynamodb:BatchWriteItem":  "wrote multiple items to DynamoDB in a batch",
	"dynamodb:CreateTable":     "created a new DynamoDB table",
	"lambda:Invoke":            "invoked a Lambda function",
	"cognito-idp:InitiateAuth": "initiated an authentication flow with Cognito",
	"cognito-idp:GetUser":      "retrieved user details from Cognito",
	"s3:GetObject":             "downloaded an object from S3",
	"s3:PutObject":             "uploaded an object to S3",
	"sqs:SendMessage":          "sent a message to an SQS queue",
	"sqs:ReceiveMessage":       "polled messages from an SQS queue",
	"sns:Publish":              "published a message to an SNS topic",
	"ses:SendEmail":            "sent an email via SES",
}

// buildNarrative generates a detailed, AI-style text explanation that walks
// through every call in the request, explains what each service did, where
// time was spent, and what went wrong (if anything).
func buildNarrative(entry *gateway.RequestEntry, ctx *ExplainContext, a *ExplainAnalysis) string {
	var b strings.Builder

	// ---- Opening summary ----
	b.WriteString(fmt.Sprintf("## Request Analysis: %s %s\n\n", entry.Method, entry.Path))

	if a.IsError {
		b.WriteString(fmt.Sprintf("This request to **%s** (%s) **failed** with HTTP %d after **%.2fms**.",
			describeService(entry.Service), entry.Action, entry.StatusCode, entry.LatencyMs))
		if entry.Error != "" {
			b.WriteString(fmt.Sprintf(" The error returned was: `%s`.", entry.Error))
		}
	} else {
		b.WriteString(fmt.Sprintf("This request to **%s** completed **successfully** (HTTP %d) in **%.2fms**.",
			describeService(entry.Service), entry.StatusCode, entry.LatencyMs))
	}
	b.WriteString("\n\n")

	// ---- What happened (plain English trace walkthrough) ----
	b.WriteString("### What Happened\n\n")

	if len(ctx.Timeline) == 0 {
		// Single-span request
		desc := describeAction(entry.Service, entry.Action)
		b.WriteString(fmt.Sprintf("The request %s. ", desc))
		b.WriteString(fmt.Sprintf("It was handled directly by %s and took **%.2fms** to complete.\n\n", describeService(entry.Service), entry.LatencyMs))
	} else {
		b.WriteString(fmt.Sprintf("The request executed across **%d operations** in the following sequence:\n\n", len(ctx.Timeline)))

		for i, span := range ctx.Timeline {
			indent := ""
			for j := 0; j < span.Depth; j++ {
				indent += "  "
			}

			stepNum := i + 1
			desc := describeAction(span.Service, span.Action)
			b.WriteString(fmt.Sprintf("%s**Step %d** (+%.1fms): %s — %s",
				indent, stepNum, span.StartOffsetMs, describeService(span.Service), desc))

			if span.DurationMs > 0 {
				b.WriteString(fmt.Sprintf(". Took **%.2fms**", span.DurationMs))
			}
			if span.Error != "" {
				b.WriteString(fmt.Sprintf(". **Failed** with: `%s`", span.Error))
			} else if span.StatusCode >= 400 {
				b.WriteString(fmt.Sprintf(". **Returned HTTP %d**", span.StatusCode))
			}
			b.WriteString(".\n")
		}
		b.WriteString("\n")
	}

	// ---- Time breakdown by service layer ----
	if len(ctx.Timeline) > 1 {
		b.WriteString("### Time Breakdown by Service\n\n")

		type layerTime struct {
			service string
			totalMs float64
			count   int
		}
		layers := make(map[string]*layerTime)
		for _, span := range ctx.Timeline {
			svc := span.Service
			lt, ok := layers[svc]
			if !ok {
				lt = &layerTime{service: svc}
				layers[svc] = lt
			}
			lt.totalMs += span.DurationMs
			lt.count++
		}

		b.WriteString("| Service | Calls | Time | % of Total | Role |\n")
		b.WriteString("|---------|-------|------|------------|------|\n")
		for _, lt := range layers {
			pct := 0.0
			if entry.LatencyMs > 0 {
				pct = (lt.totalMs / entry.LatencyMs) * 100
			}
			role := categorizeService(lt.service)
			b.WriteString(fmt.Sprintf("| %s | %d | %.2fms | %.0f%% | %s |\n",
				describeService(lt.service), lt.count, lt.totalMs, pct, role))
		}
		b.WriteString("\n")

		// Explain what this means
		var slowestLayer *layerTime
		for _, lt := range layers {
			if slowestLayer == nil || lt.totalMs > slowestLayer.totalMs {
				slowestLayer = lt
			}
		}
		if slowestLayer != nil && entry.LatencyMs > 0 {
			pct := (slowestLayer.totalMs / entry.LatencyMs) * 100
			if pct > 50 {
				b.WriteString(fmt.Sprintf("**%s consumed %.0f%% of the total request time.** ", describeService(slowestLayer.service), pct))
				switch categorizeService(slowestLayer.service) {
				case "Data":
					b.WriteString("This suggests the bottleneck is in the data layer. Consider adding caching, optimizing query patterns, or reducing the data payload.\n\n")
				case "Compute":
					b.WriteString("The compute layer is the bottleneck. Check function cold starts, memory allocation, and processing complexity.\n\n")
				case "Auth":
					b.WriteString("Authentication is the bottleneck. Consider caching tokens or reducing auth round-trips.\n\n")
				case "Messaging":
					b.WriteString("Messaging operations are slow. Check queue depth and consumer throughput.\n\n")
				default:
					b.WriteString("\n\n")
				}
			}
		}
	}

	// ---- Bottleneck analysis ----
	b.WriteString("### Bottleneck Analysis\n\n")

	if len(ctx.Timeline) > 1 {
		// Find slowest span
		slowest := ctx.Timeline[0]
		for _, s := range ctx.Timeline[1:] {
			if s.DurationMs > slowest.DurationMs {
				slowest = s
			}
		}
		if entry.LatencyMs > 0 {
			pct := (slowest.DurationMs / entry.LatencyMs) * 100
			b.WriteString(fmt.Sprintf("The **critical path** runs through `%s/%s`, which took **%.2fms** (%.0f%% of total).\n\n",
				slowest.Service, slowest.Action, slowest.DurationMs, pct))
		}

		// Check for sequential vs parallel execution
		if len(ctx.Timeline) > 2 {
			lastEnd := 0.0
			sequential := 0
			for _, span := range ctx.Timeline[1:] {
				if span.StartOffsetMs >= lastEnd-0.1 {
					sequential++
				}
				end := span.StartOffsetMs + span.DurationMs
				if end > lastEnd {
					lastEnd = end
				}
			}
			if sequential > len(ctx.Timeline)/2 {
				b.WriteString(fmt.Sprintf("Most operations ran **sequentially** (%d of %d). Consider parallelizing independent calls to reduce total latency.\n\n", sequential, len(ctx.Timeline)-1))
			} else {
				b.WriteString("Operations appear to have some **parallelism**, which is good for latency.\n\n")
			}
		}
	} else {
		if a.IsSlow {
			b.WriteString(fmt.Sprintf("This single-operation request is **slow** at %.1fx the median. ", a.LatencyRatio))
			b.WriteString("Since there are no downstream calls, the latency is entirely within the service itself.\n\n")
		} else {
			b.WriteString("No downstream calls detected. Latency is within the service itself and within normal range.\n\n")
		}
	}

	// ---- Latency context ----
	b.WriteString("### Latency Context\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n|---|---|\n"))
	b.WriteString(fmt.Sprintf("| This request | **%.2fms** |\n", entry.LatencyMs))
	b.WriteString(fmt.Sprintf("| P50 (typical) | %.2fms |\n", a.P50Ms))
	b.WriteString(fmt.Sprintf("| P95 (slow) | %.2fms |\n", a.P95Ms))
	b.WriteString(fmt.Sprintf("| P99 (very slow) | %.2fms |\n", a.P99Ms))
	if a.P50Ms > 0 {
		b.WriteString(fmt.Sprintf("| Ratio to median | **%.1fx** |\n", a.LatencyRatio))
	}
	b.WriteString("\n")

	// ---- Request details ----
	if entry.RequestBody != "" {
		b.WriteString("### Request Payload\n\n")
		b.WriteString(fmt.Sprintf("The request sent the following payload to %s:\n\n", describeService(entry.Service)))
		b.WriteString("```json\n")
		b.WriteString(entry.RequestBody)
		if len(entry.RequestBody) > 0 && entry.RequestBody[len(entry.RequestBody)-1] != '\n' {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")

		// Analyze the payload
		if entry.Service == "dynamodb" {
			if strings.Contains(entry.RequestBody, "Scan") {
				b.WriteString("**Note:** This request uses a `Scan` operation, which reads every item in the table. This is the most expensive DynamoDB operation and should be avoided in production for large tables.\n\n")
			}
			if tableName := extractTableFromBody(entry.RequestBody); tableName != "" {
				b.WriteString(fmt.Sprintf("**Target table:** `%s`\n\n", tableName))
			}
			if strings.Contains(entry.RequestBody, "FilterExpression") {
				b.WriteString("**Note:** A `FilterExpression` is applied after the query/scan, meaning DynamoDB reads more data than returned. Consider moving filter conditions into `KeyConditionExpression` if possible.\n\n")
			}
		}
	}

	if entry.ResponseBody != "" {
		b.WriteString("### Response\n\n")
		b.WriteString("```json\n")
		body := entry.ResponseBody
		if len(body) > 2000 {
			body = body[:2000] + "\n... (truncated)"
		}
		b.WriteString(body)
		if len(body) > 0 && body[len(body)-1] != '\n' {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	}

	// ---- Baseline comparison ----
	if len(ctx.SimilarRecent) > 1 {
		b.WriteString("### Compared to Recent Traffic\n\n")
		errCount := 0
		for _, r := range ctx.SimilarRecent {
			if r.StatusCode >= 400 {
				errCount++
			}
		}
		successRate := float64(len(ctx.SimilarRecent)-errCount) / float64(len(ctx.SimilarRecent)) * 100
		b.WriteString(fmt.Sprintf("Out of **%d recent** `%s/%s` requests:\n\n", len(ctx.SimilarRecent), entry.Service, entry.Action))
		b.WriteString(fmt.Sprintf("- **%.0f%%** succeeded\n", successRate))
		b.WriteString(fmt.Sprintf("- Typical latency is **%.2fms** (P50)\n", a.P50Ms))
		b.WriteString(fmt.Sprintf("- Worst case is **%.2fms** (P99)\n", a.P99Ms))
		if a.IsSlow {
			b.WriteString(fmt.Sprintf("- This request at **%.2fms** is **slower than %.0f%%** of similar requests\n", entry.LatencyMs, 100-(a.LatencyRatio*50)))
		}
		b.WriteString("\n")
	}

	// ---- Diagnosis & recommendations ----
	b.WriteString("### Diagnosis\n\n")
	if len(a.Anomalies) > 0 {
		for _, anom := range a.Anomalies {
			b.WriteString(fmt.Sprintf("- \u26A0 **%s**\n", anom))
		}
		b.WriteString("\n")
	}

	if a.IsError {
		b.WriteString("**Root cause assessment:** ")
		if entry.StatusCode == 400 {
			b.WriteString("The request was rejected due to invalid input. Check the request payload for missing or malformed fields.\n\n")
		} else if entry.StatusCode == 403 {
			b.WriteString("Access was denied. Verify IAM policies, caller credentials, and resource permissions.\n\n")
		} else if entry.StatusCode == 404 {
			b.WriteString("The requested resource was not found. Verify the resource exists and the identifier is correct.\n\n")
		} else if entry.StatusCode >= 500 {
			b.WriteString("The service returned an internal error. Check service logs, resource limits, and downstream health.\n\n")
		} else {
			b.WriteString(fmt.Sprintf("HTTP %d returned. Review the response body for details.\n\n", entry.StatusCode))
		}
	} else if a.IsSlow {
		b.WriteString("**Performance assessment:** ")
		if a.SlowestSpan != "" {
			b.WriteString(fmt.Sprintf("The primary bottleneck is `%s`. ", a.SlowestSpan))
		}
		if len(ctx.Timeline) > 3 {
			b.WriteString("Consider reducing the number of downstream calls by batching operations or caching results.")
		}
		b.WriteString("\n\n")
	} else {
		b.WriteString("Request completed within normal parameters. No issues detected.\n\n")
	}

	// ---- Metadata ----
	b.WriteString("### Metadata\n\n")
	b.WriteString(fmt.Sprintf("- **Request ID:** `%s`\n", entry.ID))
	b.WriteString(fmt.Sprintf("- **Timestamp:** %s\n", entry.Timestamp.Format("2006-01-02 15:04:05.000 MST")))
	if entry.TraceID != "" {
		b.WriteString(fmt.Sprintf("- **Trace ID:** `%s`\n", entry.TraceID))
	}
	if entry.CallerID != "" {
		b.WriteString(fmt.Sprintf("- **Caller:** `%s`\n", entry.CallerID))
	}
	b.WriteString(fmt.Sprintf("- **Service:** %s\n", entry.Service))
	b.WriteString(fmt.Sprintf("- **Action:** %s\n", entry.Action))

	return b.String()
}

func describeService(svc string) string {
	if desc, ok := serviceDescription[svc]; ok {
		return desc
	}
	return svc
}

func describeAction(svc, action string) string {
	key := svc + ":" + action
	if desc, ok := actionDescription[key]; ok {
		return desc
	}
	return fmt.Sprintf("performed `%s`", action)
}

func categorizeService(svc string) string {
	switch svc {
	case "dynamodb", "s3", "rds":
		return "Data"
	case "lambda":
		return "Compute"
	case "cognito-idp", "iam", "sts", "kms", "secretsmanager":
		return "Auth"
	case "sqs", "sns", "ses", "events":
		return "Messaging"
	default:
		return "Infrastructure"
	}
}

func extractTableFromBody(body string) string {
	idx := strings.Index(body, `"TableName"`)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(`"TableName"`):]
	rest = strings.TrimLeft(rest, " \t\n\r:")
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	end := strings.Index(rest[1:], `"`)
	if end < 0 {
		return ""
	}
	return rest[1 : end+1]
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

// handleIncidents handles incident API endpoints:
//
//	GET  /api/incidents              — list incidents with optional filters
//	GET  /api/incidents/{id}         — get a single incident by ID
//	POST /api/incidents/{id}/acknowledge — acknowledge an incident
//	POST /api/incidents/{id}/resolve     — resolve an incident
func (a *API) handleIncidents(w http.ResponseWriter, r *http.Request) {
	if a.incidentService == nil {
		writeError(w, http.StatusServiceUnavailable, "incident service not available")
		return
	}

	// Strip prefix to get ID
	path := strings.TrimPrefix(r.URL.Path, "/api/incidents")
	path = strings.TrimPrefix(path, "/")

	// GET /api/incidents/{id}/report — export report
	if r.Method == http.MethodGet && strings.HasSuffix(path, "/report") {
		id := strings.TrimSuffix(path, "/report")
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}
		if a.reportGenerator == nil {
			writeError(w, http.StatusServiceUnavailable, "report generator not available")
			return
		}
		content, contentType, err := a.reportGenerator.Generate(r.Context(), id, format)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", contentType)
		if format != "json" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=incident-%s.%s", id, format))
		}
		w.Write(content) //nolint:errcheck
		return
	}

	// GET /api/incidents — list
	if r.Method == http.MethodGet && path == "" {
		filter := incident.IncidentFilter{
			Status:   r.URL.Query().Get("status"),
			Severity: r.URL.Query().Get("severity"),
			Service:  r.URL.Query().Get("service"),
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			filter.Limit, _ = strconv.Atoi(l)
		}
		if filter.Limit == 0 {
			filter.Limit = 50
		}
		results, err := a.incidentService.Store().List(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, results)
		return
	}

	// GET /api/incidents/{id} — detail
	if r.Method == http.MethodGet && path != "" && !strings.Contains(path, "/") {
		inc, err := a.incidentService.Store().Get(r.Context(), path)
		if err != nil {
			if errors.Is(err, incident.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, inc)
		return
	}

	// POST /api/incidents/{id}/acknowledge
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/acknowledge") {
		id := strings.TrimSuffix(path, "/acknowledge")
		var body struct {
			Owner string `json:"owner"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		inc, err := a.incidentService.Store().Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, incident.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		inc.Status = "acknowledged"
		inc.Owner = body.Owner
		if err := a.incidentService.Store().Update(r.Context(), inc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.auditLog(r.Context(), "incident.acknowledged", "incident:"+id, map[string]any{"owner": body.Owner})
		writeJSON(w, http.StatusOK, inc)
		return
	}

	// POST /api/incidents/{id}/resolve
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/resolve") {
		id := strings.TrimSuffix(path, "/resolve")
		inc, err := a.incidentService.Store().Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, incident.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		now := time.Now()
		inc.Status = "resolved"
		inc.ResolvedAt = &now
		if err := a.incidentService.Store().Update(r.Context(), inc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.auditLog(r.Context(), "incident.resolved", "incident:"+id, nil)
		writeJSON(w, http.StatusOK, inc)
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

// handleWebhooks handles webhook CRUD endpoints:
//
//	GET    /api/webhooks              — list all webhooks
//	POST   /api/webhooks              — create a webhook
//	DELETE /api/webhooks/{id}         — delete a webhook
//	POST   /api/webhooks/{id}/test    — send a test payload
func (a *API) handleWebhooks(w http.ResponseWriter, r *http.Request) {
	if a.webhookDispatcher == nil {
		writeError(w, http.StatusServiceUnavailable, "webhook dispatcher not available")
		return
	}

	store := a.webhookDispatcher.Store()

	path := strings.TrimPrefix(r.URL.Path, "/api/webhooks")
	path = strings.TrimPrefix(path, "/")

	// GET /api/webhooks — list
	if r.Method == http.MethodGet && path == "" {
		list, err := store.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if list == nil {
			list = []webhook.Config{}
		}
		writeJSON(w, http.StatusOK, list)
		return
	}

	// POST /api/webhooks — create
	if r.Method == http.MethodPost && path == "" {
		var cfg webhook.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		cfg.ID = "" // always generate a new ID
		if err := store.Save(r.Context(), &cfg); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.auditLog(r.Context(), "webhook.created", "webhook:"+cfg.ID, map[string]any{"url": cfg.URL})
		writeJSON(w, http.StatusCreated, cfg)
		return
	}

	// DELETE /api/webhooks/{id}
	if r.Method == http.MethodDelete && path != "" && !strings.Contains(path, "/") {
		if err := store.Delete(r.Context(), path); err != nil {
			if errors.Is(err, webhook.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.auditLog(r.Context(), "webhook.deleted", "webhook:"+path, nil)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// POST /api/webhooks/{id}/test — send a test payload
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/test") {
		id := strings.TrimSuffix(path, "/test")
		cfg, err := store.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, webhook.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = cfg // fire a test payload via the dispatcher
		testInc := &incident.Incident{
			ID:               "test-" + id,
			Status:           "active",
			Severity:         "info",
			Title:            "Test webhook delivery",
			AffectedServices: []string{"cloudmock"},
			AffectedTenants:  []string{},
			AlertCount:       1,
			FirstSeen:        time.Now(),
			LastSeen:         time.Now(),
		}
		if err := a.webhookDispatcher.FireToConfig(r.Context(), *cfg, "incident.created", testInc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

func (a *API) handleProfile(w http.ResponseWriter, r *http.Request) {
	if a.profilingEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "profiling not available")
		return
	}
	// Parse service from path: /api/profile/{service}
	service := strings.TrimPrefix(r.URL.Path, "/api/profile/")
	profileType := r.URL.Query().Get("type")
	if profileType == "" {
		profileType = "heap"
	}

	var duration time.Duration
	if d := r.URL.Query().Get("duration"); d != "" {
		duration, _ = time.ParseDuration(d)
	}
	if profileType == "cpu" && duration == 0 {
		duration = 5 * time.Second
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "flamegraph"
	}

	p, err := a.profilingEngine.Capture(service, profileType, duration)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if format == "pprof" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pprof", p.ID))
		filePath, _ := a.profilingEngine.FilePath(p.ID)
		http.ServeFile(w, r, filePath)
		return
	}

	// flamegraph format
	folded, err := a.profilingEngine.FoldedStacks(p.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(folded))
}

func (a *API) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if a.profilingEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "profiling not available")
		return
	}

	// GET /api/profiles — list
	path := strings.TrimPrefix(r.URL.Path, "/api/profiles")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		service := r.URL.Query().Get("service")
		profiles, _ := a.profilingEngine.List(service)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profiles)
		return
	}

	// GET /api/profiles/{id}?format=pprof|flamegraph
	format := r.URL.Query().Get("format")
	if format == "pprof" {
		filePath, err := a.profilingEngine.FilePath(path)
		if err != nil {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, filePath)
		return
	}
	folded, err := a.profilingEngine.FoldedStacks(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(folded))
}

func (a *API) handleSourcemaps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || a.symbolizer == nil {
		writeError(w, http.StatusServiceUnavailable, "not available")
		return
	}
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "missing file parameter")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.symbolizer.LoadMap(filePath, body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	a.auditLog(r.Context(), "sourcemap.uploaded", "sourcemap:"+filePath, nil)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// SetUserStore sets the user store for auth endpoints.
func (a *API) SetUserStore(s auth.UserStore) {
	a.userStore = s
}

// SetAuthSecret sets the JWT signing secret.
func (a *API) SetAuthSecret(secret []byte) {
	a.authSecret = secret
}

// SetTenantStore sets the SaaS tenant store for hosted-tier endpoints.
func (a *API) SetTenantStore(s tenant.Store) {
	a.tenantStore = s
}

// SetClerkWebhook sets the Clerk webhook handler for /api/webhooks/clerk.
func (a *API) SetClerkWebhook(h *clerk.WebhookHandler) {
	a.clerkWebhook = h
}

// SetStripeWebhook sets the Stripe webhook handler for /api/webhooks/stripe.
func (a *API) SetStripeWebhook(h *saasstripe.WebhookHandler) {
	a.stripeWebhook = h
}

// SetTrafficEngine sets the traffic recording/replay engine and registers routes.
func (a *API) SetTrafficEngine(e *traffic.Engine) {
	a.trafficEngine = e
	e.SetBroadcaster(a.broadcaster)

	a.mux.HandleFunc("/api/traffic/recordings", a.handleTrafficRecordings)
	a.mux.HandleFunc("/api/traffic/recordings/", a.handleTrafficRecordingByID)
	a.mux.HandleFunc("/api/traffic/record", a.handleTrafficRecordStart)
	a.mux.HandleFunc("/api/traffic/record/stop", a.handleTrafficRecordStop)
	a.mux.HandleFunc("/api/traffic/replay", a.handleTrafficReplayStart)
	a.mux.HandleFunc("/api/traffic/replay/", a.handleTrafficReplayByID)
	a.mux.HandleFunc("/api/traffic/runs", a.handleTrafficRuns)
	a.mux.HandleFunc("/api/traffic/synthetic", a.handleTrafficSynthetic)
	a.mux.HandleFunc("/api/traffic/compare", a.handleTrafficCompare)
}

// handleAuthLogin handles POST /api/auth/login.
func (a *API) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.userStore == nil {
		writeError(w, http.StatusNotFound, "auth not enabled")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := a.userStore.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcryptCompare(user.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.GenerateToken(user, a.authSecret, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}

// handleAuthRegister handles POST /api/auth/register.
func (a *API) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.userStore == nil {
		writeError(w, http.StatusNotFound, "auth not enabled")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "email, password, and name are required")
		return
	}

	role := req.Role
	if role == "" {
		role = auth.RoleViewer
	}
	if !auth.ValidRoles[role] {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}

	// If auth is enabled (middleware is active), require admin to register users.
	if a.cfg.Auth.Enabled {
		caller := auth.UserFromContext(r.Context())
		if caller == nil || caller.Role != auth.RoleAdmin {
			writeError(w, http.StatusForbidden, "admin role required to register users")
			return
		}
	}

	hash, err := bcryptHash(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := &auth.User{
		Email:        req.Email,
		Name:         req.Name,
		Role:         role,
		PasswordHash: hash,
	}
	if err := a.userStore.Create(r.Context(), user); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	a.auditLog(r.Context(), "user.created", "user:"+user.ID, map[string]any{
		"email": user.Email,
		"role":  user.Role,
	})

	writeJSON(w, http.StatusCreated, user)
}

// handleAuthMe handles GET /api/auth/me.
func (a *API) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// handleUsers handles GET /api/users (admin only).
func (a *API) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.userStore == nil {
		writeError(w, http.StatusNotFound, "auth not enabled")
		return
	}

	if a.cfg.Auth.Enabled {
		caller := auth.UserFromContext(r.Context())
		if caller == nil || caller.Role != auth.RoleAdmin {
			writeError(w, http.StatusForbidden, "admin role required")
			return
		}
	}

	users, err := a.userStore.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

// handleUserByID handles PUT /api/users/{id} (admin only, update role).
func (a *API) handleUserByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.userStore == nil {
		writeError(w, http.StatusNotFound, "auth not enabled")
		return
	}

	if a.cfg.Auth.Enabled {
		caller := auth.UserFromContext(r.Context())
		if caller == nil || caller.Role != auth.RoleAdmin {
			writeError(w, http.StatusForbidden, "admin role required")
			return
		}
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "user ID required")
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !auth.ValidRoles[req.Role] {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}

	if err := a.userStore.UpdateRole(r.Context(), id, req.Role); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	a.auditLog(r.Context(), "user.role_updated", "user:"+id, map[string]any{
		"new_role": req.Role,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// bcryptHash hashes a plaintext password using bcrypt.
func bcryptHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// bcryptCompare compares a bcrypt hash with a plaintext password.
func bcryptCompare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// handlePreferences handles CRUD operations for user preferences.
//   - GET /api/preferences?namespace=X          → ListByNamespace
//   - GET /api/preferences?namespace=X&key=Y    → Get single
//   - PUT /api/preferences                      → Set (body: {namespace, key, value})
//   - DELETE /api/preferences?namespace=X&key=Y → Delete
func (a *API) handlePreferences(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		namespace := r.URL.Query().Get("namespace")
		if namespace == "" {
			writeError(w, http.StatusBadRequest, "namespace is required")
			return
		}
		key := r.URL.Query().Get("key")

		if key != "" {
			// Get single preference.
			val, err := a.prefsGet(r.Context(), namespace, key)
			if err != nil {
				if errors.Is(err, dataplane.ErrNotFound) {
					writeError(w, http.StatusNotFound, "not found")
					return
				}
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(val)
			return
		}

		// List all preferences in namespace.
		result, err := a.prefsListByNamespace(r.Context(), namespace)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)

	case http.MethodPut:
		var body struct {
			Namespace string          `json:"namespace"`
			Key       string          `json:"key"`
			Value     json.RawMessage `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if body.Namespace == "" || body.Key == "" {
			writeError(w, http.StatusBadRequest, "namespace and key are required")
			return
		}
		if err := a.prefsSet(r.Context(), body.Namespace, body.Key, body.Value); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		namespace := r.URL.Query().Get("namespace")
		key := r.URL.Query().Get("key")
		if namespace == "" || key == "" {
			writeError(w, http.StatusBadRequest, "namespace and key are required")
			return
		}
		if err := a.prefsDelete(r.Context(), namespace, key); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// prefsGet returns a single preference, using the DataPlane store if
// available or the legacy in-memory fallback.
func (a *API) prefsGet(ctx context.Context, namespace, key string) (json.RawMessage, error) {
	if a.dp != nil && a.dp.Preferences != nil {
		return a.dp.Preferences.Get(ctx, namespace, key)
	}
	a.prefsMu.RLock()
	defer a.prefsMu.RUnlock()
	if a.prefs == nil {
		return nil, dataplane.ErrNotFound
	}
	ns, ok := a.prefs[namespace]
	if !ok {
		return nil, dataplane.ErrNotFound
	}
	v, ok := ns[key]
	if !ok {
		return nil, dataplane.ErrNotFound
	}
	return v, nil
}

// prefsSet upserts a preference.
func (a *API) prefsSet(ctx context.Context, namespace, key string, value json.RawMessage) error {
	if a.dp != nil && a.dp.Preferences != nil {
		return a.dp.Preferences.Set(ctx, namespace, key, value)
	}
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	if a.prefs == nil {
		a.prefs = make(map[string]map[string]json.RawMessage)
	}
	ns, ok := a.prefs[namespace]
	if !ok {
		ns = make(map[string]json.RawMessage)
		a.prefs[namespace] = ns
	}
	ns[key] = value
	return nil
}

// prefsDelete removes a preference.
func (a *API) prefsDelete(ctx context.Context, namespace, key string) error {
	if a.dp != nil && a.dp.Preferences != nil {
		return a.dp.Preferences.Delete(ctx, namespace, key)
	}
	a.prefsMu.Lock()
	defer a.prefsMu.Unlock()
	if a.prefs != nil {
		if ns, ok := a.prefs[namespace]; ok {
			delete(ns, key)
		}
	}
	return nil
}

// prefsListByNamespace returns all preferences for a namespace.
func (a *API) prefsListByNamespace(ctx context.Context, namespace string) (map[string]json.RawMessage, error) {
	if a.dp != nil && a.dp.Preferences != nil {
		return a.dp.Preferences.ListByNamespace(ctx, namespace)
	}
	a.prefsMu.RLock()
	defer a.prefsMu.RUnlock()
	result := make(map[string]json.RawMessage)
	if a.prefs != nil {
		if ns, ok := a.prefs[namespace]; ok {
			for k, v := range ns {
				result[k] = v
			}
		}
	}
	return result, nil
}

// handleSourceEvents accepts SDK events via HTTP POST.
// POST /api/source/events — single event or JSON array of events.
func (a *API) handleSourceEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	var events []sdkEvent
	if err := json.Unmarshal(raw, &events); err != nil {
		var single sdkEvent
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			http.Error(w, "invalid event format", http.StatusBadRequest)
			return
		}
		events = []sdkEvent{single}
	}

	if a.sourceServer != nil {
		for _, evt := range events {
			_ = a.sourceServer.IngestSDKEvent(evt)
		}
	} else {
		for _, evt := range events {
			if evt.Type != "http:inbound" && evt.Type != "http:response" {
				continue
			}
			entry, err := convertSDKEvent(evt)
			if err != nil {
				continue
			}
			if a.log != nil {
				a.log.Add(entry)
			}
			if a.stats != nil {
				a.stats.Increment(entry.Service)
			}
			if a.broadcaster != nil {
				a.broadcaster.Broadcast("request", entry)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"accepted": len(events)})
}

// handleSourceStatus returns connected source info.
// GET /api/source/status
func (a *API) handleSourceStatus(w http.ResponseWriter, r *http.Request) {
	result := map[string]any{
		"tcp_sources":  []connectedSource{},
		"http_sources": []string{},
		"total_events": int64(0),
	}
	if a.sourceServer != nil {
		result["tcp_sources"] = a.sourceServer.ConnectedSources()
		result["http_sources"] = a.sourceServer.HTTPSources()
		result["total_events"] = a.sourceServer.EventCount()
	}
	writeJSON(w, http.StatusOK, result)
}

// --- SaaS hosted-tier handlers ---

// handleTenantsSaaS handles GET/POST /api/saas/tenants for the hosted SaaS tier.
// GET  — list all tenants (admin only).
// POST — create a new tenant.
func (a *API) handleTenantsSaaS(w http.ResponseWriter, r *http.Request) {
	if a.tenantStore == nil {
		writeError(w, http.StatusServiceUnavailable, "SaaS mode is not enabled")
		return
	}

	switch r.Method {
	case http.MethodGet:
		tenants, err := a.tenantStore.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list tenants")
			return
		}
		type tenantResponse struct {
			ID                   string `json:"id"`
			ClerkOrgID           string `json:"clerk_org_id"`
			Name                 string `json:"name"`
			Slug                 string `json:"slug"`
			Tier                 string `json:"tier"`
			Status               string `json:"status"`
			RequestCount         int64  `json:"request_count"`
			RequestLimit         int64  `json:"request_limit"`
			StripeCustomerID     string `json:"stripe_customer_id,omitempty"`
			StripeSubscriptionID string `json:"stripe_subscription_id,omitempty"`
			FlyAppName           string `json:"fly_app_name,omitempty"`
			CreatedAt            string `json:"created_at"`
		}
		result := make([]tenantResponse, 0, len(tenants))
		for _, t := range tenants {
			result = append(result, tenantResponse{
				ID:                   t.ID,
				ClerkOrgID:           t.ClerkOrgID,
				Name:                 t.Name,
				Slug:                 t.Slug,
				Tier:                 t.Tier,
				Status:               t.Status,
				RequestCount:         t.RequestCount,
				RequestLimit:         t.RequestLimit,
				StripeCustomerID:     t.StripeCustomerID,
				StripeSubscriptionID: t.StripeSubscriptionID,
				FlyAppName:           t.FlyAppName,
				CreatedAt:            t.CreatedAt.Format(time.RFC3339),
			})
		}
		writeJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var req struct {
			Name       string `json:"name"`
			Slug       string `json:"slug"`
			ClerkOrgID string `json:"clerk_org_id"`
			Tier       string `json:"tier"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" || req.Slug == "" {
			writeError(w, http.StatusBadRequest, "name and slug are required")
			return
		}
		tier := req.Tier
		if tier == "" {
			tier = "free"
		}
		// Default request limits by tier.
		var requestLimit int64
		switch tier {
		case "pro":
			requestLimit = 100_000
		case "team":
			requestLimit = 1_000_000
		default:
			requestLimit = 1_000
		}

		t := &tenant.Tenant{
			ClerkOrgID:   req.ClerkOrgID,
			Name:         req.Name,
			Slug:         req.Slug,
			Tier:         tier,
			Status:       "active",
			RequestLimit: requestLimit,
		}
		if err := a.tenantStore.Create(r.Context(), t); err != nil {
			writeError(w, http.StatusConflict, "failed to create tenant: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":     t.ID,
			"name":   t.Name,
			"slug":   t.Slug,
			"tier":   t.Tier,
			"status": t.Status,
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleUsage returns current usage for the authenticated user's tenant.
// GET /api/usage
func (a *API) handleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.tenantStore == nil {
		writeError(w, http.StatusServiceUnavailable, "SaaS mode is not enabled")
		return
	}

	// Try to get the tenant ID from the query param or X-Tenant-ID header.
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id query parameter or X-Tenant-ID header is required")
		return
	}

	t, err := a.tenantStore.Get(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":     t.ID,
		"request_count": t.RequestCount,
		"request_limit": t.RequestLimit,
		"tier":          t.Tier,
		"usage_percent": usagePercent(t.RequestCount, t.RequestLimit),
	})
}

// usagePercent returns the usage percentage (0-100), or 0 if limit is 0.
func usagePercent(count, limit int64) float64 {
	if limit <= 0 {
		return 0
	}
	return float64(count) / float64(limit) * 100
}

// handleSubscription returns subscription status for a tenant.
// GET /api/subscription
func (a *API) handleSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.tenantStore == nil {
		writeError(w, http.StatusServiceUnavailable, "SaaS mode is not enabled")
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id query parameter or X-Tenant-ID header is required")
		return
	}

	t, err := a.tenantStore.Get(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":              t.ID,
		"tier":                   t.Tier,
		"status":                 t.Status,
		"stripe_subscription_id": t.StripeSubscriptionID,
		"stripe_customer_id":     t.StripeCustomerID,
	})
}

// handleClerkWebhook delegates POST /api/webhooks/clerk to the Clerk webhook handler.
func (a *API) handleClerkWebhook(w http.ResponseWriter, r *http.Request) {
	if a.clerkWebhook == nil {
		writeError(w, http.StatusServiceUnavailable, "Clerk webhooks not configured")
		return
	}
	a.clerkWebhook.HandleWebhook(w, r)
}

// handleStripeWebhook delegates POST /api/webhooks/stripe to the Stripe webhook handler.
func (a *API) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if a.stripeWebhook == nil {
		writeError(w, http.StatusServiceUnavailable, "Stripe webhooks not configured")
		return
	}
	a.stripeWebhook.HandleWebhook(w, r)
}
