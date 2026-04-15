package admin

import (
	"net/http"
	"strings"
)

// registerBrowserRoutes wires the per-service "browser" endpoints consumed by
// the devtools views (SNS, EventBridge, API Gateway, Route 53) and the shape
// wrappers over the anomaly and lambda log lists. These endpoints return data
// in the shape the UI wants rather than the raw AWS response, so view code
// stays thin.
func (a *API) registerBrowserRoutes() {
	a.mux.HandleFunc("/api/sns/topics", a.handleBrowserSNS)
	a.mux.HandleFunc("/api/eventbridge/buses", a.handleBrowserEventBridge)
	a.mux.HandleFunc("/api/apigateway/apis", a.handleBrowserAPIGateway)
	a.mux.HandleFunc("/api/route53/zones", a.handleBrowserRoute53)
	a.mux.HandleFunc("/api/browser/anomalies", a.handleBrowserAnomalies)
	a.mux.HandleFunc("/api/browser/logs", a.handleBrowserLogs)
}

// browserInspector is the anonymous interface each service implements to feed
// its state to the admin "browser" endpoints. Using an anonymous interface
// avoids a hard import cycle from admin back into services/*.
type browserInspector interface {
	BrowserInspect() []map[string]any
}

// ── SNS ──────────────────────────────────────────────────────────────────────

func (a *API) handleBrowserSNS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	svc, err := a.registry.Lookup("sns")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"topics": []any{}})
		return
	}

	if insp, ok := svc.(browserInspector); ok {
		writeJSON(w, http.StatusOK, map[string]any{"topics": insp.BrowserInspect()})
		return
	}

	// Fallback: synthesize a list from the minimum interface contract.
	if topicsIface, ok := svc.(interface{ GetAllTopics() []string }); ok {
		arns := topicsIface.GetAllTopics()
		counts := map[string]int{}
		if subsIface, ok := svc.(interface {
			GetSubscriptionsSummary() (topicArns, protocols, endpoints []string)
		}); ok {
			topicArns, _, _ := subsIface.GetSubscriptionsSummary()
			for _, ta := range topicArns {
				counts[ta]++
			}
		}
		topics := make([]map[string]any, 0, len(arns))
		for _, arn := range arns {
			topics = append(topics, map[string]any{
				"topicArn":          arn,
				"name":              arnLastPart(arn),
				"subscriptionCount": counts[arn],
				"recentMessages":    []string{},
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"topics": topics})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"topics": []any{}})
}

// ── EventBridge ──────────────────────────────────────────────────────────────

func (a *API) handleBrowserEventBridge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	svc, err := a.registry.Lookup("events")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"buses": []any{}})
		return
	}

	if insp, ok := svc.(browserInspector); ok {
		writeJSON(w, http.StatusOK, map[string]any{"buses": insp.BrowserInspect()})
		return
	}

	if busesIface, ok := svc.(interface{ GetAllEventBuses() []string }); ok {
		busNames := busesIface.GetAllEventBuses()
		buses := make([]map[string]any, 0, len(busNames))
		for _, name := range busNames {
			buses = append(buses, map[string]any{
				"name":  name,
				"arn":   "arn:aws:events:us-east-1:000000000000:event-bus/" + name,
				"rules": []any{},
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"buses": buses})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"buses": []any{}})
}

// ── API Gateway ──────────────────────────────────────────────────────────────

func (a *API) handleBrowserAPIGateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, err := a.registry.Lookup("apigateway")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"apis": []any{}})
		return
	}
	if insp, ok := svc.(browserInspector); ok {
		writeJSON(w, http.StatusOK, map[string]any{"apis": insp.BrowserInspect()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"apis": []any{}})
}

// ── Route 53 ────────────────────────────────────────────────────────────────

func (a *API) handleBrowserRoute53(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	svc, err := a.registry.Lookup("route53")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"zones": []any{}})
		return
	}
	if insp, ok := svc.(browserInspector); ok {
		writeJSON(w, http.StatusOK, map[string]any{"zones": insp.BrowserInspect()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"zones": []any{}})
}

// ── Anomalies (shape wrapper) ───────────────────────────────────────────────

func (a *API) handleBrowserAnomalies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.anomalyDetector == nil {
		writeJSON(w, http.StatusOK, map[string]any{"anomalies": []any{}})
		return
	}

	items := a.anomalyDetector.GetAnomalies(60)
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		var kind string
		switch it.Metric {
		case "latency_p50", "latency_p95", "latency_p99", "latency":
			kind = "latency_spike"
		case "error_rate", "errors":
			kind = "error_spike"
		case "throughput", "request_rate":
			kind = "throughput_drop"
		default:
			kind = it.Metric
		}
		sev := it.Severity
		if sev == "info" {
			sev = "warning"
		}
		out = append(out, map[string]any{
			"id":         it.ID,
			"type":       kind,
			"severity":   sev,
			"service":    it.Service,
			"message":    it.Description,
			"detectedAt": it.DetectedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"anomalies": out})
}

// ── Logs (shape wrapper) ────────────────────────────────────────────────────

func (a *API) handleBrowserLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.lambdaLogs == nil {
		writeJSON(w, http.StatusOK, map[string]any{"logs": []any{}})
		return
	}

	entries := a.lambdaLogs.Recent(r.URL.Query().Get("function"), 200)
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		severity := "INFO"
		if e.Stream == "stderr" {
			severity = "ERROR"
		}
		low := strings.ToLower(e.Message)
		switch {
		case strings.Contains(low, "error"):
			severity = "ERROR"
		case strings.Contains(low, "warn"):
			severity = "WARN"
		case strings.Contains(low, "debug"):
			severity = "DEBUG"
		}
		out = append(out, map[string]any{
			"timestamp": e.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
			"service":   e.FunctionName,
			"severity":  severity,
			"message":   e.Message,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": out})
}

