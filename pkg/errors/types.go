package errors

import "time"

// ErrorGroup represents a class of errors grouped by fingerprint.
type ErrorGroup struct {
	ID        string            `json:"id"`        // fingerprint hash
	Message   string            `json:"message"`   // error message
	Type      string            `json:"type"`      // "TypeError", "ValidationError", etc.
	Source    string            `json:"source"`    // file:line or service name
	Stack     string            `json:"stack"`     // full stack trace
	Count     int               `json:"count"`     // total occurrences
	Sessions  int               `json:"sessions"`  // unique sessions affected
	FirstSeen time.Time         `json:"first_seen"`
	LastSeen  time.Time         `json:"last_seen"`
	Status    string            `json:"status"`  // "unresolved", "resolved", "ignored"
	Release   string            `json:"release"` // release/deploy where first seen
	Tags      map[string]string `json:"tags"`
}

// ErrorEvent represents a single error occurrence.
type ErrorEvent struct {
	ID          string         `json:"id"`
	GroupID     string         `json:"group_id"`  // fingerprint
	Timestamp   time.Time      `json:"timestamp"`
	SessionID   string         `json:"session_id"`
	URL         string         `json:"url"`
	UserAgent   string         `json:"user_agent"`
	Message     string         `json:"message"`
	Stack       string         `json:"stack"`
	Breadcrumbs []Breadcrumb   `json:"breadcrumbs"` // events leading up to error
	Context     map[string]any `json:"context"`     // request headers, body, etc.
	Release     string         `json:"release"`
	Service     string         `json:"service"`
	TraceID     string         `json:"trace_id"` // link to distributed trace
}

// Breadcrumb records an event leading up to an error.
type Breadcrumb struct {
	Timestamp time.Time      `json:"timestamp"`
	Category  string         `json:"category"` // "http", "console", "navigation", "ui"
	Message   string         `json:"message"`
	Level     string         `json:"level"` // "info", "warning", "error"
	Data      map[string]any `json:"data"`
}
