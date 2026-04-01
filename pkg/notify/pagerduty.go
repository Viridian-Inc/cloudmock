package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const pagerDutyEventsURL = "https://events.pagerduty.com/v2/enqueue"

// PagerDutyChannel sends notifications via PagerDuty Events API v2.
type PagerDutyChannel struct {
	RoutingKey string
	Name       string
	client     *http.Client
}

// NewPagerDutyChannel creates a PagerDuty notification channel.
func NewPagerDutyChannel(name, routingKey string) *PagerDutyChannel {
	return &PagerDutyChannel{
		RoutingKey: routingKey,
		Name:       name,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *PagerDutyChannel) Type() string { return "pagerduty" }

// Send delivers a notification to PagerDuty Events API v2.
func (p *PagerDutyChannel) Send(ctx context.Context, n Notification) error {
	payload := p.buildPayload(n)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("pagerduty: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pagerDutyEventsURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pagerduty: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("pagerduty: http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("pagerduty: http status %d", resp.StatusCode)
	}
	return nil
}

// pdEvent is the PagerDuty Events API v2 payload.
type pdEvent struct {
	RoutingKey  string    `json:"routing_key"`
	EventAction string    `json:"event_action"` // "trigger", "resolve"
	DedupKey    string    `json:"dedup_key,omitempty"`
	Payload     pdPayload `json:"payload"`
	Links       []pdLink  `json:"links,omitempty"`
}

type pdPayload struct {
	Summary       string            `json:"summary"`
	Severity      string            `json:"severity"` // critical, error, warning, info
	Source        string            `json:"source"`
	Component     string            `json:"component,omitempty"`
	Group         string            `json:"group,omitempty"`
	Class         string            `json:"class,omitempty"`
	Timestamp     string            `json:"timestamp"`
	CustomDetails map[string]string `json:"custom_details,omitempty"`
}

type pdLink struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

// mapSeverity converts our severity levels to PagerDuty's accepted values.
// PagerDuty accepts: critical, error, warning, info.
func mapPDSeverity(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "warning":
		return "warning"
	case "info":
		return "info"
	default:
		return "error"
	}
}

func (p *PagerDutyChannel) buildPayload(n Notification) pdEvent {
	// Determine event action: default to trigger; resolve if message contains "resolved"
	action := "trigger"

	dedupKey := n.DedupKey
	if dedupKey == "" {
		dedupKey = n.Service + ":" + n.Type + ":" + n.Title
	}

	summary := n.Title
	if n.Message != "" {
		summary = n.Title + " - " + n.Message
	}
	// PagerDuty summary max 1024 chars
	if len(summary) > 1024 {
		summary = summary[:1021] + "..."
	}

	event := pdEvent{
		RoutingKey:  p.RoutingKey,
		EventAction: action,
		DedupKey:    dedupKey,
		Payload: pdPayload{
			Summary:       summary,
			Severity:      mapPDSeverity(n.Severity),
			Source:        "cloudmock",
			Component:     n.Service,
			Group:         n.Type,
			Class:         n.Severity,
			Timestamp:     n.Timestamp.UTC().Format(time.RFC3339),
			CustomDetails: n.Fields,
		},
	}

	if n.URL != "" {
		event.Links = []pdLink{
			{Href: n.URL, Text: "View in DevTools"},
		}
	}

	return event
}

// SendResolve sends a resolve event to PagerDuty for a given dedup key.
func (p *PagerDutyChannel) SendResolve(ctx context.Context, dedupKey string) error {
	event := pdEvent{
		RoutingKey:  p.RoutingKey,
		EventAction: "resolve",
		DedupKey:    dedupKey,
		Payload: pdPayload{
			Summary:   "Resolved",
			Severity:  "info",
			Source:    "cloudmock",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("pagerduty: marshal resolve: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pagerDutyEventsURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pagerduty: build resolve request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("pagerduty: resolve http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("pagerduty: resolve http status %d", resp.StatusCode)
	}
	return nil
}

// FormatPagerDutyPayload formats a notification as a PagerDuty Events API v2 payload (exported for testing).
func FormatPagerDutyPayload(n Notification, routingKey string) ([]byte, error) {
	ch := &PagerDutyChannel{RoutingKey: routingKey}
	return json.Marshal(ch.buildPayload(n))
}
