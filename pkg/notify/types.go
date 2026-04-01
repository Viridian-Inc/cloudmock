package notify

import (
	"context"
	"time"
)

// Channel is the interface that notification channels must implement.
type Channel interface {
	Send(ctx context.Context, notification Notification) error
	Type() string
}

// Notification is the payload sent to notification channels.
type Notification struct {
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Severity  string            `json:"severity"` // "critical", "warning", "info"
	Service   string            `json:"service"`
	URL       string            `json:"url"`       // link back to DevTools
	Timestamp time.Time         `json:"timestamp"`
	Fields    map[string]string `json:"fields,omitempty"`  // extra key-value pairs
	Actions   []Action          `json:"actions,omitempty"` // interactive buttons
	Type      string            `json:"type"`              // "incident", "regression", "slo_breach", "error"
	DedupKey  string            `json:"dedup_key,omitempty"`
}

// Action represents an interactive button in a notification.
type Action struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Style string `json:"style"` // "primary", "danger"
}

// DeliveryRecord tracks a single notification delivery attempt.
type DeliveryRecord struct {
	ID           string    `json:"id"`
	Notification Notification `json:"notification"`
	ChannelType  string    `json:"channel_type"`
	ChannelName  string    `json:"channel_name"`
	RouteID      string    `json:"route_id"`
	Status       string    `json:"status"` // "success", "failed"
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// Route defines a routing rule that matches notifications to channels.
type Route struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Match    RouteMatch  `json:"match"`
	Channels []ChannelRef `json:"channels"`
	Enabled  bool        `json:"enabled"`
}

// RouteMatch defines the conditions that a notification must match.
type RouteMatch struct {
	Services   []string `json:"services,omitempty"`   // match specific services (empty = all)
	Severities []string `json:"severities,omitempty"` // match severities (empty = all)
	Types      []string `json:"types,omitempty"`      // "incident", "regression", "slo_breach", "error"
}

// ChannelRef references a configured channel with optional overrides.
type ChannelRef struct {
	Type   string            `json:"type"`   // "slack", "pagerduty", "email"
	Name   string            `json:"name"`   // human-readable name
	Config map[string]string `json:"config"` // webhook_url, routing_key, etc.
}

// ChannelSchema describes a channel type's configuration schema for the API.
type ChannelSchema struct {
	Type        string              `json:"type"`
	Description string              `json:"description"`
	Fields      []ChannelFieldSchema `json:"fields"`
}

// ChannelFieldSchema describes a single config field for a channel type.
type ChannelFieldSchema struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // "string", "url", "email", "number"
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Placeholder string `json:"placeholder,omitempty"`
}
