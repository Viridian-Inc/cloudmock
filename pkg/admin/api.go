package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
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
	Status   string                 `json:"status"`
	Services map[string]bool        `json:"services"`
}

// API is the admin HTTP handler.
type API struct {
	cfg      *config.Config
	registry *routing.Registry
	log      *gateway.RequestLog
	stats    *gateway.RequestStats
	mux      *http.ServeMux
}

// New creates an admin API handler wired to the given registry, config, and request log/stats.
func New(cfg *config.Config, registry *routing.Registry, log *gateway.RequestLog, stats *gateway.RequestStats) *API {
	a := &API{
		cfg:      cfg,
		registry: registry,
		log:      log,
		stats:    stats,
		mux:      http.NewServeMux(),
	}

	a.mux.HandleFunc("/api/services", a.handleServices)
	a.mux.HandleFunc("/api/services/", a.handleServiceByName)
	a.mux.HandleFunc("/api/reset", a.handleResetAll)
	a.mux.HandleFunc("/api/health", a.handleHealth)
	a.mux.HandleFunc("/api/config", a.handleConfig)
	a.mux.HandleFunc("/api/stats", a.handleStats)
	a.mux.HandleFunc("/api/requests", a.handleRequests)

	return a
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
