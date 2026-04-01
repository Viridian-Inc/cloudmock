package monitor

import (
	"context"
	"time"
)

// MonitorType identifies the kind of metric a monitor watches.
type MonitorType string

const (
	MonitorTypeErrorRate    MonitorType = "error_rate"
	MonitorTypeLatencyP50  MonitorType = "latency_p50"
	MonitorTypeLatencyP95  MonitorType = "latency_p95"
	MonitorTypeLatencyP99  MonitorType = "latency_p99"
	MonitorTypeThroughput  MonitorType = "throughput"
	MonitorTypeCustom      MonitorType = "custom"
)

// MonitorStatus represents the current evaluation state of a monitor.
type MonitorStatus string

const (
	MonitorStatusOK       MonitorStatus = "ok"
	MonitorStatusWarning  MonitorStatus = "warning"
	MonitorStatusCritical MonitorStatus = "critical"
	MonitorStatusMuted    MonitorStatus = "muted"
	MonitorStatusNoData   MonitorStatus = "no_data"
)

// Monitor defines a threshold-based alert rule over a service metric.
type Monitor struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Type        MonitorType   `json:"type"`
	Service     string        `json:"service"`               // target service (e.g. "dynamodb"), "*" for all
	Action      string        `json:"action,omitempty"`       // target action (e.g. "Query"), "*" or empty for all
	Operator    string        `json:"operator"`               // "gt", "lt", "gte", "lte"
	Warning     float64       `json:"warning"`                // warning threshold
	Critical    float64       `json:"critical"`               // critical threshold
	Enabled     bool          `json:"enabled"`
	Status      MonitorStatus `json:"status"`
	LastValue   float64       `json:"last_value"`
	MutedUntil  *time.Time    `json:"muted_until,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Message     string        `json:"message,omitempty"`      // optional description / runbook
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	LastChecked *time.Time    `json:"last_checked,omitempty"`
}

// AlertEvent records a single alert transition (trigger or recovery).
type AlertEvent struct {
	ID         string        `json:"id"`
	MonitorID  string        `json:"monitor_id"`
	MonitorName string       `json:"monitor_name"`
	Status     MonitorStatus `json:"status"`
	Value      float64       `json:"value"`
	Threshold  float64       `json:"threshold"`
	Service    string        `json:"service"`
	Action     string        `json:"action,omitempty"`
	Message    string        `json:"message,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
}

// MonitorFilter controls which monitors are returned by List.
type MonitorFilter struct {
	Service string
	Type    MonitorType
	Status  MonitorStatus
	Enabled *bool
	Limit   int
}

// AlertFilter controls which alert events are returned by ListAlerts.
type AlertFilter struct {
	MonitorID string
	Status    MonitorStatus
	Service   string
	Limit     int
}

// IncidentCreator is the interface the monitor system uses to create
// incidents when a critical threshold is breached.
type IncidentCreator interface {
	CreateFromMonitor(ctx context.Context, alert AlertEvent) error
}

// WebhookDispatcher is the interface for firing outbound webhooks.
type WebhookDispatcher interface {
	Fire(ctx context.Context, event string, payload any) error
}
