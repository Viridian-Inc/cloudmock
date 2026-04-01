package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackChannel sends notifications via Slack Incoming Webhooks using Block Kit.
type SlackChannel struct {
	WebhookURL string
	Name       string
	client     *http.Client
}

// NewSlackChannel creates a Slack notification channel.
func NewSlackChannel(name, webhookURL string) *SlackChannel {
	return &SlackChannel{
		WebhookURL: webhookURL,
		Name:       name,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *SlackChannel) Type() string { return "slack" }

// Send delivers a notification to Slack using Block Kit formatting.
func (s *SlackChannel) Send(ctx context.Context, n Notification) error {
	payload := s.buildPayload(n)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("slack: http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack: http status %d", resp.StatusCode)
	}
	return nil
}

// slackMessage is the top-level Slack message payload.
type slackMessage struct {
	Text        string       `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
	Blocks      []slackBlock `json:"blocks,omitempty"`
}

type slackAttachment struct {
	Color  string       `json:"color"`
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type     string          `json:"type"`
	Text     *slackTextObj   `json:"text,omitempty"`
	Fields   []slackTextObj  `json:"fields,omitempty"`
	Elements []slackElement  `json:"elements,omitempty"`
}

type slackTextObj struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type slackElement struct {
	Type  string       `json:"type"`
	Text  *slackTextObj `json:"text,omitempty"`
	URL   string       `json:"url,omitempty"`
	Style string       `json:"style,omitempty"`
}

// severityColor maps severity levels to Slack sidebar colors.
func severityColor(severity string) string {
	switch severity {
	case "critical":
		return "#E01E5A" // red
	case "warning":
		return "#ECB22E" // orange
	case "info":
		return "#2EB67D" // blue-green
	default:
		return "#36C5F0" // blue
	}
}

// severityEmoji maps severity levels to emoji indicators.
func severityEmoji(severity string) string {
	switch severity {
	case "critical":
		return "\xf0\x9f\x94\xb4" // red circle
	case "warning":
		return "\xf0\x9f\x9f\xa0" // orange circle
	case "info":
		return "\xf0\x9f\x94\xb5" // blue circle
	default:
		return "\xe2\xac\x9c"     // white circle
	}
}

func (s *SlackChannel) buildPayload(n Notification) slackMessage {
	emoji := severityEmoji(n.Severity)
	fallbackText := fmt.Sprintf("%s [%s] %s", emoji, n.Severity, n.Title)

	// Header block
	headerBlock := slackBlock{
		Type: "section",
		Text: &slackTextObj{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s %s*", emoji, n.Title),
		},
	}

	// Fields block with service, severity, type
	fields := []slackTextObj{
		{Type: "mrkdwn", Text: fmt.Sprintf("*Severity:*\n%s", n.Severity)},
	}
	if n.Service != "" {
		fields = append(fields, slackTextObj{Type: "mrkdwn", Text: fmt.Sprintf("*Service:*\n%s", n.Service)})
	}
	if n.Type != "" {
		fields = append(fields, slackTextObj{Type: "mrkdwn", Text: fmt.Sprintf("*Type:*\n%s", n.Type)})
	}
	// Add extra fields
	for k, v := range n.Fields {
		fields = append(fields, slackTextObj{Type: "mrkdwn", Text: fmt.Sprintf("*%s:*\n%s", k, v)})
	}
	fieldsBlock := slackBlock{
		Type:   "section",
		Fields: fields,
	}

	// Message block
	var blocks []slackBlock
	blocks = append(blocks, headerBlock, fieldsBlock)

	if n.Message != "" {
		blocks = append(blocks, slackBlock{
			Type: "section",
			Text: &slackTextObj{Type: "mrkdwn", Text: n.Message},
		})
	}

	// Action buttons
	if len(n.Actions) > 0 || n.URL != "" {
		var elements []slackElement

		if n.URL != "" {
			elements = append(elements, slackElement{
				Type: "button",
				Text: &slackTextObj{Type: "plain_text", Text: "View in DevTools"},
				URL:  n.URL,
			})
		}

		for _, a := range n.Actions {
			elem := slackElement{
				Type: "button",
				Text: &slackTextObj{Type: "plain_text", Text: a.Label},
				URL:  a.URL,
			}
			if a.Style == "danger" {
				elem.Style = "danger"
			} else if a.Style == "primary" {
				elem.Style = "primary"
			}
			elements = append(elements, elem)
		}

		if len(elements) > 0 {
			blocks = append(blocks, slackBlock{
				Type:     "actions",
				Elements: elements,
			})
		}
	}

	// Context with timestamp
	blocks = append(blocks, slackBlock{
		Type: "context",
		Elements: []slackElement{
			{
				Type: "mrkdwn",
				Text: &slackTextObj{Type: "mrkdwn", Text: fmt.Sprintf("CloudMock | %s", n.Timestamp.UTC().Format(time.RFC3339))},
			},
		},
	})

	return slackMessage{
		Text: fallbackText,
		Attachments: []slackAttachment{
			{
				Color:  severityColor(n.Severity),
				Blocks: blocks,
			},
		},
	}
}

// FormatSlackPayload formats a notification as a Slack Block Kit payload (exported for testing).
func FormatSlackPayload(n Notification) ([]byte, error) {
	ch := &SlackChannel{}
	return json.Marshal(ch.buildPayload(n))
}

// AvailableChannelSchemas returns the config schemas for all built-in channel types.
func AvailableChannelSchemas() []ChannelSchema {
	return []ChannelSchema{
		{
			Type:        "slack",
			Description: "Slack incoming webhook notifications with Block Kit formatting",
			Fields: []ChannelFieldSchema{
				{Name: "webhook_url", Label: "Webhook URL", Type: "url", Required: true, Placeholder: "https://hooks.slack.com/services/..."},
			},
		},
		{
			Type:        "pagerduty",
			Description: "PagerDuty Events API v2 for incident alerting and on-call escalation",
			Fields: []ChannelFieldSchema{
				{Name: "routing_key", Label: "Routing Key", Type: "string", Required: true, Secret: true, Placeholder: "Your PagerDuty integration key"},
			},
		},
		{
			Type:        "email",
			Description: "Email notifications via SMTP",
			Fields: []ChannelFieldSchema{
				{Name: "smtp_host", Label: "SMTP Host", Type: "string", Required: true, Placeholder: "smtp.gmail.com"},
				{Name: "smtp_port", Label: "SMTP Port", Type: "number", Required: true, Placeholder: "587"},
				{Name: "username", Label: "Username", Type: "string", Required: false},
				{Name: "password", Label: "Password", Type: "string", Required: false, Secret: true},
				{Name: "from", Label: "From Address", Type: "email", Required: true, Placeholder: "alerts@myapp.com"},
				{Name: "to", Label: "To Addresses (comma-separated)", Type: "string", Required: true, Placeholder: "team@myapp.com"},
			},
		},
	}
}
