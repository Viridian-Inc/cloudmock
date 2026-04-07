package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/notify"
)

// handleNotifyRoutes handles GET /api/notify/routes and POST /api/notify/routes.
func (a *API) handleNotifyRoutes(w http.ResponseWriter, r *http.Request) {
	if a.notifyRouter == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "notification routing not configured"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		routes := a.notifyRouter.ListRoutes()
		writeJSON(w, http.StatusOK, routes)

	case http.MethodPost:
		var route notify.Route
		if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := a.notifyRouter.AddRoute(route); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, route)

	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleNotifyRouteByID handles PUT /api/notify/routes/:id and DELETE /api/notify/routes/:id.
func (a *API) handleNotifyRouteByID(w http.ResponseWriter, r *http.Request) {
	if a.notifyRouter == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "notification routing not configured"})
		return
	}

	// Extract ID from path: /api/notify/routes/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/notify/routes/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing route id"})
		return
	}

	switch r.Method {
	case http.MethodPut:
		var route notify.Route
		if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		route.ID = id
		if err := a.notifyRouter.UpdateRoute(route); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, route)

	case http.MethodDelete:
		if err := a.notifyRouter.RemoveRoute(id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		w.Header().Set("Allow", "PUT, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleNotifyChannels handles GET /api/notify/channels — returns available channel type schemas.
func (a *API) handleNotifyChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, notify.AvailableChannelSchemas())
}

// handleNotifyTest handles POST /api/notify/test — sends a test notification.
func (a *API) handleNotifyTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if a.notifyRouter == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "notification routing not configured"})
		return
	}

	var req struct {
		Channel  notify.ChannelRef `json:"channel"`
		Severity string            `json:"severity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if req.Severity == "" {
		req.Severity = "info"
	}

	testNotif := notify.Notification{
		Title:     "CloudMock Test Notification",
		Message:   "This is a test notification to verify your alert channel configuration.",
		Severity:  req.Severity,
		Service:   "cloudmock-test",
		Type:      "test",
		URL:       "http://localhost:4599",
		Timestamp: time.Now(),
		Fields: map[string]string{
			"Channel Type": req.Channel.Type,
			"Channel Name": req.Channel.Name,
		},
		Actions: []notify.Action{
			{Label: "View in DevTools", URL: "http://localhost:4599", Style: "primary"},
		},
	}

	// Build channel from ref and send directly
	var ch notify.Channel
	switch req.Channel.Type {
	case "slack":
		url := req.Channel.Config["webhook_url"]
		if url == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "webhook_url is required"})
			return
		}
		ch = notify.NewSlackChannel(req.Channel.Name, url)
	case "pagerduty":
		key := req.Channel.Config["routing_key"]
		if key == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "routing_key is required"})
			return
		}
		ch = notify.NewPagerDutyChannel(req.Channel.Name, key)
	case "email":
		host := req.Channel.Config["smtp_host"]
		from := req.Channel.Config["from"]
		to := req.Channel.Config["to"]
		if host == "" || from == "" || to == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "smtp_host, from, and to are required"})
			return
		}
		port, _ := strconv.Atoi(req.Channel.Config["smtp_port"])
		if port == 0 {
			port = 587
		}
		toAddrs := strings.Split(to, ",")
		for i := range toAddrs {
			toAddrs[i] = strings.TrimSpace(toAddrs[i])
		}
		ch = notify.NewEmailChannel(req.Channel.Name, host, port, req.Channel.Config["username"], req.Channel.Config["password"], from, toAddrs)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown channel type: " + req.Channel.Type})
		return
	}

	if err := ch.Send(r.Context(), testNotif); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// handleNotifyHistory handles GET /api/notify/history — returns recent delivery log.
func (a *API) handleNotifyHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if a.notifyRouter == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "notification routing not configured"})
		return
	}

	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}

	history := a.notifyRouter.History(limit)
	writeJSON(w, http.StatusOK, history)
}
