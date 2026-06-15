package notify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Router matches incoming notifications against routing rules and dispatches
// them to the appropriate channels. It is safe for concurrent use.
type Router struct {
	mu          sync.RWMutex
	routes      []Route
	channels    map[string]Channel // keyed by "type:name"
	history     []DeliveryRecord
	maxHist     int
	windows     []MaintenanceWindow
	teams       map[string]Team   // team name → team
	serviceTeam map[string]string // service → owning team name
}

// NewRouter creates a notification router.
func NewRouter() *Router {
	return &Router{
		channels:    make(map[string]Channel),
		maxHist:     500,
		teams:       make(map[string]Team),
		serviceTeam: make(map[string]string),
	}
}

// RegisterChannel adds a pre-built channel to the router.
func (r *Router) RegisterChannel(name string, ch Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := ch.Type() + ":" + name
	r.channels[key] = ch
}

// Notify evaluates all enabled routes against the notification and dispatches
// to matching channels. Errors from individual channels are logged but do not
// prevent delivery to other channels (graceful degradation).
func (r *Router) Notify(ctx context.Context, n Notification) error {
	r.mu.RLock()
	routes := make([]Route, len(r.routes))
	copy(routes, r.routes)
	windows := make([]MaintenanceWindow, len(r.windows))
	copy(windows, r.windows)
	team, hasTeam := r.teams[r.serviceTeam[n.Service]]
	r.mu.RUnlock()

	// Maintenance-window suppression: if an active window covers this
	// notification, record it as suppressed and dispatch to nothing.
	now := time.Now()
	for _, w := range windows {
		if w.suppresses(n, now) {
			r.recordDelivery(n, ChannelRef{Type: "suppressed", Name: w.Name}, "window:"+w.ID,
				"suppressed", "maintenance window active")
			return nil
		}
	}

	var firstErr error
	sent := make(map[string]bool) // channel keys already dispatched (for team dedup)

	for _, route := range routes {
		if !route.Enabled || !matchesRoute(route.Match, n) {
			continue
		}
		for _, ref := range route.Channels {
			r.dispatch(ctx, n, ref, route.ID, &firstErr)
			sent[ref.Type+":"+ref.Name] = true
		}
	}

	// Team-tier routing: also deliver to the owning team's channels that a
	// direct route didn't already cover.
	if hasTeam {
		for _, ref := range team.Channels {
			if sent[ref.Type+":"+ref.Name] {
				continue
			}
			r.dispatch(ctx, n, ref, "team:"+team.Name, &firstErr)
			sent[ref.Type+":"+ref.Name] = true
		}
	}

	return firstErr
}

// dispatch resolves a channel ref and sends the notification, recording the
// delivery outcome. *firstErr captures the first delivery error.
func (r *Router) dispatch(ctx context.Context, n Notification, ref ChannelRef, routeID string, firstErr *error) {
	ch := r.resolveChannel(ref)
	if ch == nil {
		slog.Warn("notify: channel not found, building on-demand", "type", ref.Type, "name", ref.Name)
		ch = r.buildChannel(ref)
		if ch == nil {
			r.recordDelivery(n, ref, routeID, "failed", "channel not configured")
			return
		}
	}
	if err := ch.Send(ctx, n); err != nil {
		slog.Warn("notify: channel delivery failed", "type", ref.Type, "name", ref.Name, "error", err)
		r.recordDelivery(n, ref, routeID, "failed", err.Error())
		if *firstErr == nil {
			*firstErr = err
		}
		return
	}
	r.recordDelivery(n, ref, routeID, "success", "")
}

// --- Maintenance windows ---

// AddWindow registers a maintenance window, generating an ID if empty.
func (r *Router) AddWindow(w MaintenanceWindow) MaintenanceWindow {
	r.mu.Lock()
	defer r.mu.Unlock()
	if w.ID == "" {
		w.ID = generateID()
	}
	r.windows = append(r.windows, w)
	return w
}

// RemoveWindow deletes a maintenance window by ID; reports whether it existed.
func (r *Router) RemoveWindow(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, w := range r.windows {
		if w.ID == id {
			r.windows = append(r.windows[:i], r.windows[i+1:]...)
			return true
		}
	}
	return false
}

// ListWindows returns a copy of the configured maintenance windows.
func (r *Router) ListWindows() []MaintenanceWindow {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MaintenanceWindow, len(r.windows))
	copy(out, r.windows)
	return out
}

// LoadWindows replaces all maintenance windows (from config).
func (r *Router) LoadWindows(ws []MaintenanceWindow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.windows = make([]MaintenanceWindow, len(ws))
	for i, w := range ws {
		if w.ID == "" {
			w.ID = generateID()
		}
		r.windows[i] = w
	}
}

// --- Team-tier routing ---

// SetTeam registers/updates a team and the services it owns. Services map to
// the team's channels in addition to any direct service→channel routes.
func (r *Router) SetTeam(team Team, services []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.teams[team.Name] = team
	for _, s := range services {
		r.serviceTeam[s] = team.Name
	}
}

// ListTeams returns the configured teams.
func (r *Router) ListTeams() []Team {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Team, 0, len(r.teams))
	for _, t := range r.teams {
		out = append(out, t)
	}
	return out
}

// AddRoute adds a routing rule. If the ID is empty, one is generated.
func (r *Router) AddRoute(route Route) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if route.ID == "" {
		route.ID = generateID()
	}

	// Check for duplicate ID
	for _, existing := range r.routes {
		if existing.ID == route.ID {
			return fmt.Errorf("route with id %q already exists", route.ID)
		}
	}

	r.routes = append(r.routes, route)
	return nil
}

// UpdateRoute replaces a route by ID.
func (r *Router) UpdateRoute(route Route) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, existing := range r.routes {
		if existing.ID == route.ID {
			r.routes[i] = route
			return nil
		}
	}
	return fmt.Errorf("route %q not found", route.ID)
}

// RemoveRoute removes a routing rule by ID.
func (r *Router) RemoveRoute(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, route := range r.routes {
		if route.ID == id {
			r.routes = append(r.routes[:i], r.routes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("route %q not found", id)
}

// ListRoutes returns a copy of all routing rules.
func (r *Router) ListRoutes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Route, len(r.routes))
	copy(result, r.routes)
	return result
}

// History returns recent delivery records.
func (r *Router) History(limit int) []DeliveryRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > len(r.history) {
		limit = len(r.history)
	}

	// Return most recent first
	result := make([]DeliveryRecord, limit)
	for i := 0; i < limit; i++ {
		result[i] = r.history[len(r.history)-1-i]
	}
	return result
}

// resolveChannel looks up a channel by type:name, falling back to type-only match.
func (r *Router) resolveChannel(ref ChannelRef) Channel {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try exact match first
	key := ref.Type + ":" + ref.Name
	if ch, ok := r.channels[key]; ok {
		return ch
	}

	// Fall back to any channel of this type
	for k, ch := range r.channels {
		if strings.HasPrefix(k, ref.Type+":") {
			return ch
		}
	}
	return nil
}

// buildChannel creates a channel on-demand from a ChannelRef config.
func (r *Router) buildChannel(ref ChannelRef) Channel {
	var ch Channel

	switch ref.Type {
	case "slack":
		url := ref.Config["webhook_url"]
		if url == "" {
			return nil
		}
		ch = NewSlackChannel(ref.Name, url)
	case "pagerduty":
		key := ref.Config["routing_key"]
		if key == "" {
			return nil
		}
		ch = NewPagerDutyChannel(ref.Name, key)
	case "email":
		host := ref.Config["smtp_host"]
		portStr := ref.Config["smtp_port"]
		from := ref.Config["from"]
		toStr := ref.Config["to"]
		if host == "" || from == "" || toStr == "" {
			return nil
		}
		port, _ := strconv.Atoi(portStr)
		if port == 0 {
			port = 587
		}
		to := strings.Split(toStr, ",")
		for i := range to {
			to[i] = strings.TrimSpace(to[i])
		}
		ch = NewEmailChannel(ref.Name, host, port, ref.Config["username"], ref.Config["password"], from, to)
	default:
		return nil
	}

	// Cache it
	r.mu.Lock()
	r.channels[ref.Type+":"+ref.Name] = ch
	r.mu.Unlock()

	return ch
}

func (r *Router) recordDelivery(n Notification, ref ChannelRef, routeID, status, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := DeliveryRecord{
		ID:           generateID(),
		Notification: n,
		ChannelType:  ref.Type,
		ChannelName:  ref.Name,
		RouteID:      routeID,
		Status:       status,
		Error:        errMsg,
		Timestamp:    time.Now(),
	}

	r.history = append(r.history, rec)
	if len(r.history) > r.maxHist {
		r.history = r.history[len(r.history)-r.maxHist:]
	}
}

// matchesRoute checks if a notification matches a route's conditions.
func matchesRoute(m RouteMatch, n Notification) bool {
	// Check services filter
	if len(m.Services) > 0 {
		found := false
		for _, s := range m.Services {
			if s == n.Service {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check severities filter
	if len(m.Severities) > 0 {
		found := false
		for _, s := range m.Severities {
			if s == n.Severity {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check types filter
	if len(m.Types) > 0 {
		found := false
		for _, t := range m.Types {
			if t == n.Type {
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

// generateID creates a short random ID.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// LoadRoutes loads routes from config, replacing any existing routes.
func (r *Router) LoadRoutes(routes []Route) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = make([]Route, len(routes))
	for i, route := range routes {
		if route.ID == "" {
			route.ID = generateID()
		}
		r.routes[i] = route
	}
}

// LoadChannels creates and registers channels from ChannelRef configs.
func (r *Router) LoadChannels(refs []ChannelRef) {
	for _, ref := range refs {
		ch := r.buildChannel(ref)
		if ch != nil {
			r.RegisterChannel(ref.Name, ch)
		}
	}
}
