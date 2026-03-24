package webhook

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/neureaux/cloudmock/pkg/incident"
)

// slackPayload is the JSON structure sent to a Slack incoming webhook.
type slackPayload struct {
	Text   string       `json:"text"`
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type string      `json:"type"`
	Text *slackText  `json:"text,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// FormatSlack formats an incident event as a Slack incoming-webhook payload.
func FormatSlack(event string, payload any) ([]byte, error) {
	inc, err := toIncident(payload)
	if err != nil {
		return nil, err
	}

	emoji := "🔥"
	action := "created"
	if event == "incident.resolved" {
		emoji = "✅"
		action = "resolved"
	}

	text := fmt.Sprintf("%s Incident %s: %s (severity: %s)", emoji, action, inc.Title, inc.Severity)

	detail := fmt.Sprintf("*Services*: %v\n*Status*: %s\n*First seen*: %s",
		inc.AffectedServices, inc.Status, inc.FirstSeen.UTC().Format(time.RFC3339))
	if inc.ResolvedAt != nil {
		detail += fmt.Sprintf("\n*Resolved at*: %s", inc.ResolvedAt.UTC().Format(time.RFC3339))
	}

	p := slackPayload{
		Text: text,
		Blocks: []slackBlock{
			{
				Type: "section",
				Text: &slackText{Type: "mrkdwn", Text: "*" + text + "*"},
			},
			{
				Type: "section",
				Text: &slackText{Type: "mrkdwn", Text: detail},
			},
		},
	}
	return json.Marshal(p)
}

// pagerDutyPayload is the PagerDuty Events API v2 structure.
type pagerDutyPayload struct {
	RoutingKey  string           `json:"routing_key"`
	EventAction string           `json:"event_action"` // "trigger" or "resolve"
	DedupKey    string           `json:"dedup_key"`
	Payload     pagerDutyDetails `json:"payload"`
}

type pagerDutyDetails struct {
	Summary   string `json:"summary"`
	Severity  string `json:"severity"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

// FormatPagerDuty formats an incident event as a PagerDuty Events API v2 payload.
func FormatPagerDuty(event string, payload any) ([]byte, error) {
	inc, err := toIncident(payload)
	if err != nil {
		return nil, err
	}

	action := "trigger"
	if event == "incident.resolved" {
		action = "resolve"
	}

	severity := inc.Severity
	// PagerDuty accepts: critical, error, warning, info
	if severity == "" {
		severity = "error"
	}

	p := pagerDutyPayload{
		RoutingKey:  "", // callers may set via headers / routing key field
		EventAction: action,
		DedupKey:    inc.ID,
		Payload: pagerDutyDetails{
			Summary:   inc.Title,
			Severity:  severity,
			Source:    "cloudmock",
			Timestamp: inc.FirstSeen.UTC().Format(time.RFC3339),
		},
	}
	return json.Marshal(p)
}

// genericPayload is the envelope used for generic webhooks.
type genericPayload struct {
	Event     string             `json:"event"`
	Timestamp string             `json:"timestamp"`
	Incident  incident.Incident  `json:"incident"`
}

// FormatGeneric wraps the incident in a simple envelope and marshals it to JSON.
func FormatGeneric(event string, payload any) ([]byte, error) {
	inc, err := toIncident(payload)
	if err != nil {
		return nil, err
	}

	p := genericPayload{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Incident:  *inc,
	}
	return json.Marshal(p)
}

// toIncident coerces the payload to *incident.Incident.
// It accepts *incident.Incident or incident.Incident directly, and falls back
// to a JSON round-trip for other types (e.g. map[string]any).
func toIncident(payload any) (*incident.Incident, error) {
	switch v := payload.(type) {
	case *incident.Incident:
		return v, nil
	case incident.Incident:
		return &v, nil
	default:
		// JSON round-trip for flexibility
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("webhook: marshal payload: %w", err)
		}
		var inc incident.Incident
		if err := json.Unmarshal(b, &inc); err != nil {
			return nil, fmt.Errorf("webhook: unmarshal payload as incident: %w", err)
		}
		return &inc, nil
	}
}
