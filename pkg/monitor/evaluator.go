package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ServiceSnapshot holds the current metric values for a single service,
// collected by the MetricsProvider during each evaluation tick.
type ServiceSnapshot struct {
	Service    string
	ErrorRate  float64 // 0..1
	LatencyP50 float64 // milliseconds
	LatencyP95 float64 // milliseconds
	LatencyP99 float64 // milliseconds
	Throughput float64 // requests per second
}

// MetricsProvider is the interface the evaluator calls each tick to collect
// current metric values. Implementations may read from the gateway stats,
// DataPlane, or Prometheus.
type MetricsProvider interface {
	Snapshot(ctx context.Context) ([]ServiceSnapshot, error)
}

// Evaluator periodically checks all enabled monitors against live metrics,
// creating alert events when thresholds are breached and resolving them
// on recovery.
type Evaluator struct {
	monitors    MonitorStore
	alerts      AlertStore
	provider    MetricsProvider
	incidents   IncidentCreator   // optional
	webhooks    WebhookDispatcher // optional
	interval    time.Duration

	mu       sync.Mutex
	cancel   context.CancelFunc
	running  bool
}

// NewEvaluator creates an evaluator that ticks at the given interval.
func NewEvaluator(
	monitors MonitorStore,
	alerts AlertStore,
	provider MetricsProvider,
	interval time.Duration,
) *Evaluator {
	return &Evaluator{
		monitors: monitors,
		alerts:   alerts,
		provider: provider,
		interval: interval,
	}
}

// SetIncidentCreator configures the optional incident integration.
func (e *Evaluator) SetIncidentCreator(ic IncidentCreator) {
	e.incidents = ic
}

// SetWebhookDispatcher configures the optional webhook integration.
func (e *Evaluator) SetWebhookDispatcher(wd WebhookDispatcher) {
	e.webhooks = wd
}

// Start begins the evaluation loop in a background goroutine.
func (e *Evaluator) Start(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return
	}

	evalCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.running = true

	go e.loop(evalCtx)
}

// Stop cancels the background evaluation loop.
func (e *Evaluator) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.running = false
}

// loop runs the evaluation on each tick until the context is cancelled.
func (e *Evaluator) loop(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	// Run once immediately on start.
	e.evaluate(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.evaluate(ctx)
		}
	}
}

// evaluate collects a metrics snapshot and checks every enabled monitor.
func (e *Evaluator) evaluate(ctx context.Context) {
	snapshots, err := e.provider.Snapshot(ctx)
	if err != nil {
		slog.Warn("monitor evaluator: failed to collect metrics", "error", err)
		return
	}

	// Index by service name for fast lookup.
	index := make(map[string]ServiceSnapshot, len(snapshots))
	for _, snap := range snapshots {
		index[snap.Service] = snap
	}

	monitors, err := e.monitors.ListEnabled(ctx)
	if err != nil {
		slog.Warn("monitor evaluator: failed to list monitors", "error", err)
		return
	}

	now := time.Now()
	for _, mon := range monitors {
		// Skip muted monitors.
		if mon.MutedUntil != nil && now.Before(*mon.MutedUntil) {
			continue
		}

		snap, ok := index[mon.Service]
		if !ok && mon.Service != "*" {
			// If the service has no data, mark as no_data (but do not alert).
			if mon.Status != MonitorStatusNoData {
				mon.Status = MonitorStatusNoData
				mon.LastChecked = &now
				e.monitors.Update(ctx, &mon) //nolint:errcheck
			}
			continue
		}

		// For wildcard monitors, evaluate against each service.
		if mon.Service == "*" {
			e.evaluateWildcard(ctx, &mon, snapshots, now)
			continue
		}

		value := extractValue(mon.Type, snap)
		e.checkMonitor(ctx, &mon, value, snap.Service, now)
	}
}

// evaluateWildcard checks a wildcard monitor against every service snapshot,
// using the worst-case value.
func (e *Evaluator) evaluateWildcard(ctx context.Context, mon *Monitor, snapshots []ServiceSnapshot, now time.Time) {
	var worstValue float64
	var worstService string
	first := true

	for _, snap := range snapshots {
		value := extractValue(mon.Type, snap)
		if first || isWorse(mon.Operator, value, worstValue) {
			worstValue = value
			worstService = snap.Service
			first = true // reset after first assignment
		}
		first = false
	}

	if worstService != "" {
		e.checkMonitor(ctx, mon, worstValue, worstService, now)
	}
}

// checkMonitor evaluates a single monitor against a metric value and handles
// threshold transitions.
func (e *Evaluator) checkMonitor(ctx context.Context, mon *Monitor, value float64, service string, now time.Time) {
	prevStatus := mon.Status
	mon.LastValue = value
	mon.LastChecked = &now

	var newStatus MonitorStatus
	if breaches(mon.Operator, value, mon.Critical) {
		newStatus = MonitorStatusCritical
	} else if breaches(mon.Operator, value, mon.Warning) {
		newStatus = MonitorStatusWarning
	} else {
		newStatus = MonitorStatusOK
	}

	mon.Status = newStatus
	e.monitors.Update(ctx, mon) //nolint:errcheck

	// Only create alert events on transitions.
	if newStatus == prevStatus {
		return
	}

	alert := AlertEvent{
		MonitorID:   mon.ID,
		MonitorName: mon.Name,
		Status:      newStatus,
		Value:       value,
		Service:     service,
		Action:      mon.Action,
		CreatedAt:   now,
	}

	switch newStatus {
	case MonitorStatusCritical:
		alert.Threshold = mon.Critical
		alert.Message = fmt.Sprintf("%s breached critical threshold (%.4f %s %.4f)", mon.Name, value, mon.Operator, mon.Critical)
	case MonitorStatusWarning:
		alert.Threshold = mon.Warning
		alert.Message = fmt.Sprintf("%s breached warning threshold (%.4f %s %.4f)", mon.Name, value, mon.Operator, mon.Warning)
	case MonitorStatusOK:
		alert.Message = fmt.Sprintf("%s recovered (value: %.4f)", mon.Name, value)
	}

	if err := e.alerts.SaveAlert(ctx, &alert); err != nil {
		slog.Warn("monitor evaluator: failed to save alert", "monitor", mon.ID, "error", err)
		return
	}

	// Fire webhook on transition.
	if e.webhooks != nil {
		e.webhooks.Fire(ctx, "monitor.alert", alert) //nolint:errcheck
	}

	// Create incident on critical breach.
	if newStatus == MonitorStatusCritical && e.incidents != nil {
		if err := e.incidents.CreateFromMonitor(ctx, alert); err != nil {
			slog.Warn("monitor evaluator: failed to create incident", "monitor", mon.ID, "error", err)
		}
	}
}

// extractValue pulls the relevant metric from a snapshot based on monitor type.
func extractValue(typ MonitorType, snap ServiceSnapshot) float64 {
	switch typ {
	case MonitorTypeErrorRate:
		return snap.ErrorRate
	case MonitorTypeLatencyP50:
		return snap.LatencyP50
	case MonitorTypeLatencyP95:
		return snap.LatencyP95
	case MonitorTypeLatencyP99:
		return snap.LatencyP99
	case MonitorTypeThroughput:
		return snap.Throughput
	default:
		return 0
	}
}

// breaches returns true if value crosses the threshold in the direction
// indicated by the operator.
func breaches(operator string, value, threshold float64) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	default:
		return value > threshold
	}
}

// isWorse returns true if candidate is worse than current for the operator.
func isWorse(operator string, candidate, current float64) bool {
	switch operator {
	case "gt", "gte":
		return candidate > current
	case "lt", "lte":
		return candidate < current
	default:
		return candidate > current
	}
}
