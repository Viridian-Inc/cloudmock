package webhook_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neureaux/cloudmock/pkg/incident"
	"github.com/neureaux/cloudmock/pkg/webhook"
	whmemory "github.com/neureaux/cloudmock/pkg/webhook/memory"
)

// captureServer captures the last request body and returns a given status code.
type captureServer struct {
	mu     sync.Mutex
	bodies [][]byte
	status int
}

func (c *captureServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.bodies = append(c.bodies, body)
		c.mu.Unlock()
		w.WriteHeader(c.status)
	})
}

func TestDispatcher_Fire_Slack(t *testing.T) {
	srv := &captureServer{status: http.StatusOK}
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	store := whmemory.NewStore()
	ctx := context.Background()

	cfg := &webhook.Config{
		URL:    ts.URL,
		Type:   "slack",
		Events: []string{"incident.created"},
		Active: true,
	}
	require.NoError(t, store.Save(ctx, cfg))

	d := webhook.NewDispatcher(store)

	inc := sampleIncident()
	require.NoError(t, d.Fire(ctx, "incident.created", inc))

	// Give the HTTP call time to complete (dispatcher is synchronous; this is instant).
	srv.mu.Lock()
	bodies := srv.bodies
	srv.mu.Unlock()

	require.Len(t, bodies, 1)

	var m map[string]any
	require.NoError(t, json.Unmarshal(bodies[0], &m))
	assert.Contains(t, m["text"], "Latency spike in svc-auth")
}

func TestDispatcher_Fire_PagerDuty(t *testing.T) {
	srv := &captureServer{status: http.StatusAccepted}
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	store := whmemory.NewStore()
	ctx := context.Background()

	cfg := &webhook.Config{
		URL:    ts.URL,
		Type:   "pagerduty",
		Events: []string{"incident.created", "incident.resolved"},
		Active: true,
	}
	require.NoError(t, store.Save(ctx, cfg))

	d := webhook.NewDispatcher(store)

	inc := &incident.Incident{
		ID:               "pg-001",
		Title:            "DB overload",
		Severity:         "warning",
		Status:           "active",
		AffectedServices: []string{"svc-db"},
		AffectedTenants:  []string{},
		FirstSeen:        time.Now().UTC(),
		LastSeen:         time.Now().UTC(),
	}
	require.NoError(t, d.Fire(ctx, "incident.created", inc))

	srv.mu.Lock()
	bodies := srv.bodies
	srv.mu.Unlock()
	require.Len(t, bodies, 1)

	var m map[string]any
	require.NoError(t, json.Unmarshal(bodies[0], &m))
	assert.Equal(t, "trigger", m["event_action"])
}

func TestDispatcher_Fire_Generic(t *testing.T) {
	srv := &captureServer{status: http.StatusOK}
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	store := whmemory.NewStore()
	ctx := context.Background()

	cfg := &webhook.Config{
		URL:    ts.URL,
		Type:   "generic",
		Events: []string{"incident.resolved"},
		Active: true,
	}
	require.NoError(t, store.Save(ctx, cfg))

	d := webhook.NewDispatcher(store)

	inc := sampleIncident()
	require.NoError(t, d.Fire(ctx, "incident.resolved", inc))

	srv.mu.Lock()
	bodies := srv.bodies
	srv.mu.Unlock()
	require.Len(t, bodies, 1)

	var m map[string]any
	require.NoError(t, json.Unmarshal(bodies[0], &m))
	assert.Equal(t, "incident.resolved", m["event"])
}

func TestDispatcher_Fire_EventMismatch(t *testing.T) {
	srv := &captureServer{status: http.StatusOK}
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	store := whmemory.NewStore()
	ctx := context.Background()

	// Only subscribed to resolved, not created.
	cfg := &webhook.Config{
		URL:    ts.URL,
		Type:   "generic",
		Events: []string{"incident.resolved"},
		Active: true,
	}
	require.NoError(t, store.Save(ctx, cfg))

	d := webhook.NewDispatcher(store)
	require.NoError(t, d.Fire(ctx, "incident.created", sampleIncident()))

	srv.mu.Lock()
	bodies := srv.bodies
	srv.mu.Unlock()
	assert.Len(t, bodies, 0, "should not fire when event does not match")
}

func TestDispatcher_Fire_InactiveWebhook(t *testing.T) {
	srv := &captureServer{status: http.StatusOK}
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	store := whmemory.NewStore()
	ctx := context.Background()

	cfg := &webhook.Config{
		URL:    ts.URL,
		Type:   "generic",
		Events: []string{"incident.created"},
		Active: false, // inactive
	}
	require.NoError(t, store.Save(ctx, cfg))

	d := webhook.NewDispatcher(store)
	require.NoError(t, d.Fire(ctx, "incident.created", sampleIncident()))

	srv.mu.Lock()
	bodies := srv.bodies
	srv.mu.Unlock()
	assert.Len(t, bodies, 0, "should not fire for inactive webhook")
}

func TestDispatcher_Fire_CustomHeaders(t *testing.T) {
	var receivedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	store := whmemory.NewStore()
	ctx := context.Background()

	cfg := &webhook.Config{
		URL:     ts.URL,
		Type:    "generic",
		Events:  []string{"incident.created"},
		Headers: map[string]string{"Authorization": "Bearer token-xyz"},
		Active:  true,
	}
	require.NoError(t, store.Save(ctx, cfg))

	d := webhook.NewDispatcher(store)
	require.NoError(t, d.Fire(ctx, "incident.created", sampleIncident()))

	assert.Equal(t, "Bearer token-xyz", receivedAuth)
}

func TestDispatcher_Fire_BestEffortOnHTTPError(t *testing.T) {
	// Server returns 500 — dispatcher should not return an error.
	srv := &captureServer{status: http.StatusInternalServerError}
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	store := whmemory.NewStore()
	ctx := context.Background()

	cfg := &webhook.Config{
		URL:    ts.URL,
		Type:   "generic",
		Events: []string{"incident.created"},
		Active: true,
	}
	require.NoError(t, store.Save(ctx, cfg))

	d := webhook.NewDispatcher(store)
	err := d.Fire(ctx, "incident.created", sampleIncident())
	assert.NoError(t, err, "Fire should be best-effort and not propagate HTTP errors")
}
