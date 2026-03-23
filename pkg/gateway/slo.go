package gateway

import (
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/config"
)

// SLOEngine evaluates requests against configured SLO thresholds
// and tracks error budgets with burn rate calculation.
type SLOEngine struct {
	mu      sync.RWMutex
	rules   []config.SLORule
	windows map[string]*SLOWindow // key: "service:action"
}

// SLOWindow tracks SLO compliance over a rolling window.
type SLOWindow struct {
	Service    string  `json:"service"`
	Action     string  `json:"action"`
	Total      int64   `json:"total"`
	Violations int64   `json:"violations"` // requests exceeding SLO
	Errors     int64   `json:"errors"`
	P50Target  float64 `json:"p50_target_ms"`
	P95Target  float64 `json:"p95_target_ms"`
	P99Target  float64 `json:"p99_target_ms"`
	ErrorTarget float64 `json:"error_target"`

	// Rolling latency samples for percentile calculation
	latencies []float64
	lastReset time.Time
}

// SLOStatus is the current state of all SLO windows.
type SLOStatus struct {
	Windows   []SLOWindowStatus `json:"windows"`
	Healthy   bool              `json:"healthy"`
	Alerts    []SLOAlert        `json:"alerts"`
}

// SLOWindowStatus is the status of a single SLO window.
type SLOWindowStatus struct {
	Service       string  `json:"service"`
	Action        string  `json:"action"`
	Total         int64   `json:"total"`
	Violations    int64   `json:"violations"`
	Errors        int64   `json:"errors"`
	ErrorRate     float64 `json:"error_rate"`
	BudgetUsed    float64 `json:"budget_used"`    // 0-1, >1 = budget exhausted
	BurnRate      float64 `json:"burn_rate"`       // current burn rate (1.0 = normal)
	P50Ms         float64 `json:"p50_ms"`
	P95Ms         float64 `json:"p95_ms"`
	P99Ms         float64 `json:"p99_ms"`
	P50Target     float64 `json:"p50_target_ms"`
	P95Target     float64 `json:"p95_target_ms"`
	P99Target     float64 `json:"p99_target_ms"`
	Breaching     bool    `json:"breaching"`
}

// SLOAlert represents an active SLO alert.
type SLOAlert struct {
	Severity string `json:"severity"` // "warning", "critical"
	Service  string `json:"service"`
	Action   string `json:"action"`
	Message  string `json:"message"`
}

// NewSLOEngine creates an SLO engine with the given rules.
func NewSLOEngine(rules []config.SLORule) *SLOEngine {
	return &SLOEngine{
		rules:   rules,
		windows: make(map[string]*SLOWindow),
	}
}

// Record evaluates a request against SLO rules.
func (e *SLOEngine) Record(service, action string, latencyMs float64, statusCode int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rule := e.findRule(service, action)
	if rule == nil {
		return
	}

	key := service + ":" + action
	w, ok := e.windows[key]
	if !ok {
		w = &SLOWindow{
			Service:     service,
			Action:      action,
			P50Target:   rule.P50Ms,
			P95Target:   rule.P95Ms,
			P99Target:   rule.P99Ms,
			ErrorTarget: rule.ErrorRate,
			lastReset:   time.Now(),
		}
		e.windows[key] = w
	}

	// Reset window every 5 minutes
	if time.Since(w.lastReset) > 5*time.Minute {
		w.Total = 0
		w.Violations = 0
		w.Errors = 0
		w.latencies = w.latencies[:0]
		w.lastReset = time.Now()
	}

	w.Total++
	if latencyMs > rule.P99Ms {
		w.Violations++
	}
	if statusCode >= 400 {
		w.Errors++
	}

	// Keep last 1000 latency samples for percentile calculation
	if len(w.latencies) >= 1000 {
		w.latencies = w.latencies[1:]
	}
	w.latencies = append(w.latencies, latencyMs)
}

// Status returns the current SLO status across all windows.
func (e *SLOEngine) Status() SLOStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := SLOStatus{Healthy: true}

	for _, w := range e.windows {
		ws := SLOWindowStatus{
			Service:   w.Service,
			Action:    w.Action,
			Total:     w.Total,
			Violations: w.Violations,
			Errors:    w.Errors,
			P50Target: w.P50Target,
			P95Target: w.P95Target,
			P99Target: w.P99Target,
		}

		if w.Total > 0 {
			ws.ErrorRate = float64(w.Errors) / float64(w.Total)

			// Calculate percentiles from samples
			if len(w.latencies) > 0 {
				sorted := make([]float64, len(w.latencies))
				copy(sorted, w.latencies)
				sortF64(sorted)
				ws.P50Ms = pctl(sorted, 50)
				ws.P95Ms = pctl(sorted, 95)
				ws.P99Ms = pctl(sorted, 99)
			}

			// Budget: how much of the error budget has been consumed
			// Error budget = 1 - SLO target (e.g., 99% SLO = 1% error budget)
			if w.ErrorTarget > 0 {
				ws.BudgetUsed = ws.ErrorRate / w.ErrorTarget
			}

			// Burn rate: how fast the budget is being consumed
			// burn_rate = 1.0 means consuming at exactly the budget rate
			// burn_rate > 1.0 means budget will be exhausted before window ends
			windowMinutes := time.Since(w.lastReset).Minutes()
			if windowMinutes > 0 && w.ErrorTarget > 0 {
				expectedErrors := w.ErrorTarget * float64(w.Total)
				if expectedErrors > 0 {
					ws.BurnRate = float64(w.Errors) / expectedErrors
				}
			}

			// Check breaching
			ws.Breaching = ws.P99Ms > w.P99Target || ws.ErrorRate > w.ErrorTarget
			if ws.Breaching {
				status.Healthy = false
			}
		}

		status.Windows = append(status.Windows, ws)

		// Generate alerts
		if ws.Breaching && w.Total > 10 {
			if ws.BurnRate > 10 {
				status.Alerts = append(status.Alerts, SLOAlert{
					Severity: "critical",
					Service:  w.Service,
					Action:   w.Action,
					Message:  "Error budget exhausting rapidly — burn rate " + formatFloat(ws.BurnRate) + "x",
				})
			} else if ws.BurnRate > 2 {
				status.Alerts = append(status.Alerts, SLOAlert{
					Severity: "warning",
					Service:  w.Service,
					Action:   w.Action,
					Message:  "SLO at risk — burn rate " + formatFloat(ws.BurnRate) + "x, P99 " + formatFloat(ws.P99Ms) + "ms vs target " + formatFloat(w.P99Target) + "ms",
				})
			}
		}
	}

	return status
}

// Rules returns the configured SLO rules.
func (e *SLOEngine) Rules() []config.SLORule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rules
}

// SetRules updates the SLO rules.
func (e *SLOEngine) SetRules(rules []config.SLORule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = rules
	// Reset windows when rules change
	e.windows = make(map[string]*SLOWindow)
}

func (e *SLOEngine) findRule(service, action string) *config.SLORule {
	// Most specific match first
	for i, r := range e.rules {
		if r.Service == service && r.Action == action {
			return &e.rules[i]
		}
	}
	for i, r := range e.rules {
		if r.Service == service && r.Action == "*" {
			return &e.rules[i]
		}
	}
	for i, r := range e.rules {
		if r.Service == "*" && r.Action == "*" {
			return &e.rules[i]
		}
	}
	return nil
}

func sortF64(s []float64) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func pctl(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func formatFloat(f float64) string {
	if f < 1 {
		return "<1"
	}
	if f == float64(int(f)) {
		return string(rune('0'+int(f)%10)) + string([]byte{})
	}
	// Simple formatting without fmt
	whole := int(f)
	frac := int((f - float64(whole)) * 10)
	buf := make([]byte, 0, 8)
	buf = appendInt(buf, whole)
	buf = append(buf, '.')
	buf = appendInt(buf, frac)
	return string(buf)
}

func appendInt(buf []byte, n int) []byte {
	if n == 0 {
		return append(buf, '0')
	}
	if n < 0 {
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		buf = append(buf, digits[i])
	}
	return buf
}
