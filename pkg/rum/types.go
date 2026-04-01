package rum

import "time"

// EventType identifies the kind of RUM event.
type EventType string

const (
	EventPageLoad       EventType = "page_load"
	EventWebVital       EventType = "web_vital"
	EventJSError        EventType = "js_error"
	EventResourceTiming EventType = "resource_timing"
)

// RUMEvent is the envelope for all events sent by the browser SDK.
type RUMEvent struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	SessionID string    `json:"session_id"`
	URL       string    `json:"url"`
	UserAgent string    `json:"user_agent"`
	Timestamp time.Time `json:"timestamp"`

	// Exactly one of these will be populated, depending on Type.
	PageLoad       *PageLoadEvent       `json:"page_load,omitempty"`
	WebVital       *WebVitalEvent       `json:"web_vital,omitempty"`
	JSError        *JSErrorEvent        `json:"js_error,omitempty"`
	ResourceTiming *ResourceTimingEvent `json:"resource_timing,omitempty"`
}

// PageLoadEvent records a full page navigation.
type PageLoadEvent struct {
	Route            string  `json:"route"`
	DurationMs       float64 `json:"duration_ms"`
	TTFB             float64 `json:"ttfb_ms"`
	DOMContentLoaded float64 `json:"dom_content_loaded_ms"`
	Load             float64 `json:"load_ms"`
	TransferSizeKB   float64 `json:"transfer_size_kb"`
}

// WebVitalEvent records a single Core Web Vital measurement.
type WebVitalEvent struct {
	Name  string  `json:"name"`  // LCP, FID, CLS, TTFB, FCP, INP
	Value float64 `json:"value"` // ms for timing metrics, unitless for CLS
	Delta float64 `json:"delta"`
	Rating string `json:"rating"` // "good", "needs-improvement", "poor"
}

// JSErrorEvent records a JavaScript error.
type JSErrorEvent struct {
	Message    string `json:"message"`
	Source     string `json:"source"`
	Lineno     int    `json:"lineno"`
	Colno      int    `json:"colno"`
	Stack      string `json:"stack"`
	Fingerprint string `json:"fingerprint"` // computed server-side
}

// ResourceTimingEvent records the timing of a single resource fetch.
type ResourceTimingEvent struct {
	Name       string  `json:"name"` // URL of the resource
	InitiatorType string `json:"initiator_type"` // fetch, xmlhttprequest, script, css, img
	DurationMs float64 `json:"duration_ms"`
	TransferSizeKB float64 `json:"transfer_size_kb"`
	StatusCode int     `json:"status_code"`
}

// --- Aggregation / query response types ---

// VitalRating groups counts by good/needs-improvement/poor.
type VitalRating struct {
	Good             int     `json:"good"`
	NeedsImprovement int     `json:"needs_improvement"`
	Poor             int     `json:"poor"`
	P75              float64 `json:"p75"`
}

// WebVitalsOverview summarises all core web vitals.
type WebVitalsOverview struct {
	LCP  VitalRating `json:"lcp"`
	FID  VitalRating `json:"fid"`
	CLS  VitalRating `json:"cls"`
	TTFB VitalRating `json:"ttfb"`
	FCP  VitalRating `json:"fcp"`
	INP  VitalRating `json:"inp"`
	TotalSessions int `json:"total_sessions"`
}

// PagePerformance summarises performance for a single page route.
type PagePerformance struct {
	Route            string  `json:"route"`
	Views            int     `json:"views"`
	AvgDurationMs    float64 `json:"avg_duration_ms"`
	P75DurationMs    float64 `json:"p75_duration_ms"`
	AvgTTFB          float64 `json:"avg_ttfb_ms"`
	AvgTransferSizeKB float64 `json:"avg_transfer_size_kb"`
}

// ErrorGroup aggregates JS errors by fingerprint.
type ErrorGroup struct {
	Fingerprint string    `json:"fingerprint"`
	Message     string    `json:"message"`
	Source      string    `json:"source"`
	Count       int       `json:"count"`
	Sessions    int       `json:"sessions"`
	LastSeen    time.Time `json:"last_seen"`
	Stack       string    `json:"stack"`
}

// SessionSummary is a lightweight view of a user session.
type SessionSummary struct {
	SessionID  string    `json:"session_id"`
	StartedAt  time.Time `json:"started_at"`
	LastSeen   time.Time `json:"last_seen"`
	PageViews  int       `json:"page_views"`
	ErrorCount int       `json:"error_count"`
	UserAgent  string    `json:"user_agent"`
}
