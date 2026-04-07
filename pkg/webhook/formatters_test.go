package webhook_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Viridian-Inc/cloudmock/pkg/incident"
	"github.com/Viridian-Inc/cloudmock/pkg/webhook"
)

func sampleIncident() *incident.Incident {
	now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	return &incident.Incident{
		ID:               "inc-001",
		Status:           "active",
		Severity:         "critical",
		Title:            "Latency spike in svc-auth",
		AffectedServices: []string{"svc-auth", "svc-api"},
		AffectedTenants:  []string{"tenant-a"},
		AlertCount:       3,
		FirstSeen:        now.Add(-10 * time.Minute),
		LastSeen:         now,
	}
}

func TestFormatSlack_Created(t *testing.T) {
	inc := sampleIncident()
	b, err := webhook.FormatSlack("incident.created", inc)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	text, ok := m["text"].(string)
	require.True(t, ok)
	assert.Contains(t, text, "Latency spike in svc-auth")
	assert.Contains(t, text, "created")

	blocks, ok := m["blocks"].([]any)
	require.True(t, ok)
	assert.Len(t, blocks, 2)
}

func TestFormatSlack_Resolved(t *testing.T) {
	inc := sampleIncident()
	now := time.Now()
	inc.Status = "resolved"
	inc.ResolvedAt = &now

	b, err := webhook.FormatSlack("incident.resolved", inc)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	text, _ := m["text"].(string)
	assert.Contains(t, text, "resolved")
}

func TestFormatPagerDuty_Trigger(t *testing.T) {
	inc := sampleIncident()
	b, err := webhook.FormatPagerDuty("incident.created", inc)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Equal(t, "trigger", m["event_action"])
	assert.Equal(t, "inc-001", m["dedup_key"])

	payload, ok := m["payload"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Latency spike in svc-auth", payload["summary"])
	assert.Equal(t, "critical", payload["severity"])
	assert.Equal(t, "cloudmock", payload["source"])
}

func TestFormatPagerDuty_Resolve(t *testing.T) {
	inc := sampleIncident()
	b, err := webhook.FormatPagerDuty("incident.resolved", inc)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Equal(t, "resolve", m["event_action"])
}

func TestFormatGeneric(t *testing.T) {
	inc := sampleIncident()
	b, err := webhook.FormatGeneric("incident.created", inc)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Equal(t, "incident.created", m["event"])
	assert.NotEmpty(t, m["timestamp"])

	incData, ok := m["incident"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "inc-001", incData["id"])
	assert.Equal(t, "Latency spike in svc-auth", incData["title"])
}

func TestFormatSlack_ValuePayload(t *testing.T) {
	// Ensure value (non-pointer) incident works too.
	inc := *sampleIncident()
	b, err := webhook.FormatSlack("incident.created", inc)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}
