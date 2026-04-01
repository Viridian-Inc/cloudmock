package notify

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFormatSlackPayload(t *testing.T) {
	n := Notification{
		Title:     "SLO burn rate alert: api-gateway/GET (95% budget consumed)",
		Message:   "Error budget is being consumed rapidly. Immediate attention required.",
		Severity:  "critical",
		Service:   "api-gateway",
		Type:      "slo_breach",
		URL:       "http://localhost:4599/incidents/123",
		Timestamp: time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
		Fields: map[string]string{
			"Burn Rate":   "14.2x",
			"Budget Used": "95%",
		},
		Actions: []Action{
			{Label: "Acknowledge", URL: "http://localhost:4599/incidents/123/ack", Style: "danger"},
		},
	}

	body, err := FormatSlackPayload(n)
	if err != nil {
		t.Fatal(err)
	}

	// Parse back to verify structure
	var msg slackMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatal(err)
	}

	// Check fallback text
	if !strings.Contains(msg.Text, "critical") {
		t.Error("fallback text should contain severity")
	}
	if !strings.Contains(msg.Text, "SLO burn rate") {
		t.Error("fallback text should contain title")
	}

	// Check color attachment
	if len(msg.Attachments) == 0 {
		t.Fatal("expected at least one attachment")
	}
	if msg.Attachments[0].Color != "#E01E5A" {
		t.Errorf("expected red color for critical, got %s", msg.Attachments[0].Color)
	}

	// Check blocks exist
	blocks := msg.Attachments[0].Blocks
	if len(blocks) < 3 {
		t.Fatalf("expected at least 3 blocks, got %d", len(blocks))
	}

	// First block should be header with title
	if blocks[0].Text == nil || !strings.Contains(blocks[0].Text.Text, "SLO burn rate") {
		t.Error("first block should contain title")
	}

	// Second block should have fields
	if len(blocks[1].Fields) < 2 {
		t.Errorf("expected at least 2 fields, got %d", len(blocks[1].Fields))
	}
}

func TestFormatSlackPayloadSeverityColors(t *testing.T) {
	tests := []struct {
		severity string
		color    string
	}{
		{"critical", "#E01E5A"},
		{"warning", "#ECB22E"},
		{"info", "#2EB67D"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			n := Notification{
				Title:     "Test",
				Severity:  tt.severity,
				Timestamp: time.Now(),
			}

			body, err := FormatSlackPayload(n)
			if err != nil {
				t.Fatal(err)
			}

			var msg slackMessage
			json.Unmarshal(body, &msg)

			if len(msg.Attachments) == 0 {
				t.Fatal("no attachments")
			}
			if msg.Attachments[0].Color != tt.color {
				t.Errorf("expected color %s, got %s", tt.color, msg.Attachments[0].Color)
			}
		})
	}
}

func TestFormatSlackPayloadWithURL(t *testing.T) {
	n := Notification{
		Title:     "Test with URL",
		Severity:  "info",
		URL:       "http://localhost:4599/incidents/abc",
		Timestamp: time.Now(),
	}

	body, err := FormatSlackPayload(n)
	if err != nil {
		t.Fatal(err)
	}

	raw := string(body)
	if !strings.Contains(raw, "View in DevTools") {
		t.Error("expected 'View in DevTools' button")
	}
	if !strings.Contains(raw, "http://localhost:4599/incidents/abc") {
		t.Error("expected URL in payload")
	}
}

func TestFormatSlackPayloadMinimal(t *testing.T) {
	n := Notification{
		Title:     "Minimal alert",
		Severity:  "info",
		Timestamp: time.Now(),
	}

	body, err := FormatSlackPayload(n)
	if err != nil {
		t.Fatal(err)
	}

	var msg slackMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatal(err)
	}

	if msg.Text == "" {
		t.Error("fallback text should not be empty")
	}
}

func TestPagerDutyPayloadFormat(t *testing.T) {
	n := Notification{
		Title:     "Database connection pool exhausted",
		Severity:  "critical",
		Service:   "user-service",
		Type:      "incident",
		URL:       "http://localhost:4599/incidents/456",
		DedupKey:  "user-service:db-pool",
		Timestamp: time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
		Fields: map[string]string{
			"Pool Size":   "100",
			"Active Conn": "100",
		},
	}

	body, err := FormatPagerDutyPayload(n, "test-routing-key")
	if err != nil {
		t.Fatal(err)
	}

	var event pdEvent
	if err := json.Unmarshal(body, &event); err != nil {
		t.Fatal(err)
	}

	if event.RoutingKey != "test-routing-key" {
		t.Errorf("routing_key = %q, want test-routing-key", event.RoutingKey)
	}
	if event.EventAction != "trigger" {
		t.Errorf("event_action = %q, want trigger", event.EventAction)
	}
	if event.DedupKey != "user-service:db-pool" {
		t.Errorf("dedup_key = %q, want user-service:db-pool", event.DedupKey)
	}
	if event.Payload.Severity != "critical" {
		t.Errorf("severity = %q, want critical", event.Payload.Severity)
	}
	if event.Payload.Source != "cloudmock" {
		t.Errorf("source = %q, want cloudmock", event.Payload.Source)
	}
	if event.Payload.Component != "user-service" {
		t.Errorf("component = %q, want user-service", event.Payload.Component)
	}
	if len(event.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(event.Links))
	}
	if event.Links[0].Href != "http://localhost:4599/incidents/456" {
		t.Error("link href mismatch")
	}
}

func TestPagerDutySeverityMapping(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"critical", "critical"},
		{"warning", "warning"},
		{"info", "info"},
		{"unknown", "error"},
		{"", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapPDSeverity(tt.input)
			if got != tt.want {
				t.Errorf("mapPDSeverity(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPagerDutyAutoDedupKey(t *testing.T) {
	n := Notification{
		Title:     "Test",
		Severity:  "warning",
		Service:   "api",
		Type:      "regression",
		Timestamp: time.Now(),
	}

	body, _ := FormatPagerDutyPayload(n, "key")
	var event pdEvent
	json.Unmarshal(body, &event)

	if event.DedupKey != "api:regression:Test" {
		t.Errorf("auto dedup_key = %q, want api:regression:Test", event.DedupKey)
	}
}
