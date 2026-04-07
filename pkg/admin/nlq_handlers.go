package admin

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/nlq"
)

// handleAsk handles POST /api/ask — natural language query.
func (a *API) handleAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Question == "" {
		writeError(w, http.StatusBadRequest, "question is required")
		return
	}

	query, err := nlq.Parse(req.Question)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"question": req.Question,
			"error":    err.Error(),
			"hint":     "Try asking about errors, requests, deploys, metrics, or system health",
		})
		return
	}

	// Build response with the parsed query and a human-readable description.
	resp := map[string]any{
		"question":    req.Question,
		"description": query.Describe(),
		"query":       query,
	}

	// Execute the query against available stores.
	results := a.executeNLQuery(query)
	if results != nil {
		resp["results"] = results
	}

	writeJSON(w, http.StatusOK, resp)
}

// executeNLQuery runs a parsed query against the appropriate store.
func (a *API) executeNLQuery(query *nlq.Query) any {
	switch query.Type {
	case "health":
		return a.nlqHealth()
	case "errors":
		return a.nlqErrors(query)
	case "requests":
		return a.nlqRequests(query)
	case "deploys":
		return a.nlqDeploys(query)
	case "metrics":
		return map[string]string{"note": "Use /api/metrics for detailed metric queries"}
	default:
		return nil
	}
}

func (a *API) nlqHealth() map[string]any {
	svcs := a.registry.List()
	healthy := 0
	for _, svc := range svcs {
		if svc.HealthCheck() == nil {
			healthy++
		}
	}
	result := map[string]any{
		"services_total":   len(svcs),
		"services_healthy": healthy,
		"status":           "healthy",
	}
	if healthy < len(svcs) {
		result["status"] = "degraded"
	}
	return result
}

func (a *API) nlqErrors(query *nlq.Query) any {
	if a.errorStore == nil {
		return map[string]string{"note": "Error store not available"}
	}
	groups, err := a.errorStore.GetGroups("", 20)
	if err != nil {
		return map[string]string{"error": err.Error()}
	}
	return map[string]any{
		"count":  len(groups),
		"groups": groups,
	}
}

func (a *API) nlqRequests(query *nlq.Query) any {
	if a.log != nil {
		entries := a.log.Recent(query.Service, 20)
		return map[string]any{
			"count":    len(entries),
			"requests": entries,
		}
	}
	return map[string]string{"note": "Request log not available"}
}

func (a *API) nlqDeploys(_ *nlq.Query) any {
	a.deploysMu.RLock()
	defer a.deploysMu.RUnlock()
	limit := 10
	if len(a.deploys) < limit {
		limit = len(a.deploys)
	}
	// Return most recent deploys.
	if limit == 0 {
		return map[string]any{"count": 0, "deploys": []any{}}
	}
	recent := a.deploys[len(a.deploys)-limit:]
	return map[string]any{
		"count":   len(recent),
		"deploys": recent,
	}
}
