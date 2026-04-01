package notify

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockChannel records notifications for testing.
type mockChannel struct {
	mu       sync.Mutex
	sent     []Notification
	failWith error
	typ      string
}

func newMockChannel(typ string) *mockChannel {
	return &mockChannel{typ: typ}
}

func (m *mockChannel) Type() string { return m.typ }

func (m *mockChannel) Send(_ context.Context, n Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failWith != nil {
		return m.failWith
	}
	m.sent = append(m.sent, n)
	return nil
}

func (m *mockChannel) sentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

func TestRouterBasicRouting(t *testing.T) {
	r := NewRouter()
	ch := newMockChannel("slack")
	r.RegisterChannel("engineering", ch)

	err := r.AddRoute(Route{
		ID:      "r1",
		Name:    "all-to-slack",
		Enabled: true,
		Match:   RouteMatch{},
		Channels: []ChannelRef{
			{Type: "slack", Name: "engineering"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	n := Notification{
		Title:     "Test alert",
		Severity:  "warning",
		Service:   "api-gateway",
		Timestamp: time.Now(),
	}

	if err := r.Notify(context.Background(), n); err != nil {
		t.Fatal(err)
	}

	if ch.sentCount() != 1 {
		t.Fatalf("expected 1 notification, got %d", ch.sentCount())
	}
}

func TestRouterSeverityFilter(t *testing.T) {
	r := NewRouter()
	ch := newMockChannel("pagerduty")
	r.RegisterChannel("oncall", ch)

	err := r.AddRoute(Route{
		ID:      "crit-only",
		Name:    "critical-to-pagerduty",
		Enabled: true,
		Match: RouteMatch{
			Severities: []string{"critical"},
		},
		Channels: []ChannelRef{
			{Type: "pagerduty", Name: "oncall"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Warning should NOT match
	r.Notify(context.Background(), Notification{
		Title:     "Warning alert",
		Severity:  "warning",
		Timestamp: time.Now(),
	})
	if ch.sentCount() != 0 {
		t.Fatalf("expected 0 notifications for warning, got %d", ch.sentCount())
	}

	// Critical should match
	r.Notify(context.Background(), Notification{
		Title:     "Critical alert",
		Severity:  "critical",
		Timestamp: time.Now(),
	})
	if ch.sentCount() != 1 {
		t.Fatalf("expected 1 notification for critical, got %d", ch.sentCount())
	}
}

func TestRouterServiceFilter(t *testing.T) {
	r := NewRouter()
	ch := newMockChannel("slack")
	r.RegisterChannel("team", ch)

	r.AddRoute(Route{
		ID:      "svc-filter",
		Name:    "api-alerts",
		Enabled: true,
		Match: RouteMatch{
			Services: []string{"api-gateway", "auth-service"},
		},
		Channels: []ChannelRef{
			{Type: "slack", Name: "team"},
		},
	})

	// Matching service
	r.Notify(context.Background(), Notification{
		Title:     "API alert",
		Severity:  "warning",
		Service:   "api-gateway",
		Timestamp: time.Now(),
	})
	if ch.sentCount() != 1 {
		t.Fatalf("expected 1 notification for api-gateway, got %d", ch.sentCount())
	}

	// Non-matching service
	r.Notify(context.Background(), Notification{
		Title:     "DB alert",
		Severity:  "warning",
		Service:   "database",
		Timestamp: time.Now(),
	})
	if ch.sentCount() != 1 {
		t.Fatalf("expected still 1 notification, got %d", ch.sentCount())
	}
}

func TestRouterTypeFilter(t *testing.T) {
	r := NewRouter()
	ch := newMockChannel("slack")
	r.RegisterChannel("ops", ch)

	r.AddRoute(Route{
		ID:      "inc-only",
		Name:    "incidents-only",
		Enabled: true,
		Match: RouteMatch{
			Types: []string{"incident"},
		},
		Channels: []ChannelRef{
			{Type: "slack", Name: "ops"},
		},
	})

	// Regression type should not match
	r.Notify(context.Background(), Notification{
		Title:     "Regression detected",
		Type:      "regression",
		Severity:  "warning",
		Timestamp: time.Now(),
	})
	if ch.sentCount() != 0 {
		t.Fatal("regression should not match incident-only route")
	}

	// Incident type should match
	r.Notify(context.Background(), Notification{
		Title:     "Incident created",
		Type:      "incident",
		Severity:  "critical",
		Timestamp: time.Now(),
	})
	if ch.sentCount() != 1 {
		t.Fatal("incident should match incident-only route")
	}
}

func TestRouterDisabledRoute(t *testing.T) {
	r := NewRouter()
	ch := newMockChannel("slack")
	r.RegisterChannel("team", ch)

	r.AddRoute(Route{
		ID:      "disabled",
		Name:    "disabled-route",
		Enabled: false,
		Match:   RouteMatch{},
		Channels: []ChannelRef{
			{Type: "slack", Name: "team"},
		},
	})

	r.Notify(context.Background(), Notification{
		Title:     "Test",
		Severity:  "info",
		Timestamp: time.Now(),
	})
	if ch.sentCount() != 0 {
		t.Fatal("disabled route should not send notifications")
	}
}

func TestRouterMultipleChannels(t *testing.T) {
	r := NewRouter()
	slack := newMockChannel("slack")
	pd := newMockChannel("pagerduty")
	r.RegisterChannel("eng", slack)
	r.RegisterChannel("oncall", pd)

	r.AddRoute(Route{
		ID:      "multi",
		Name:    "critical-everywhere",
		Enabled: true,
		Match: RouteMatch{
			Severities: []string{"critical"},
		},
		Channels: []ChannelRef{
			{Type: "slack", Name: "eng"},
			{Type: "pagerduty", Name: "oncall"},
		},
	})

	r.Notify(context.Background(), Notification{
		Title:     "Outage",
		Severity:  "critical",
		Timestamp: time.Now(),
	})

	if slack.sentCount() != 1 {
		t.Fatalf("slack: expected 1, got %d", slack.sentCount())
	}
	if pd.sentCount() != 1 {
		t.Fatalf("pagerduty: expected 1, got %d", pd.sentCount())
	}
}

func TestRouterGracefulDegradation(t *testing.T) {
	r := NewRouter()
	failing := newMockChannel("slack")
	failing.failWith = errTestDelivery
	working := newMockChannel("pagerduty")
	r.RegisterChannel("broken", failing)
	r.RegisterChannel("oncall", working)

	r.AddRoute(Route{
		ID:      "degrade",
		Name:    "multi",
		Enabled: true,
		Match:   RouteMatch{},
		Channels: []ChannelRef{
			{Type: "slack", Name: "broken"},
			{Type: "pagerduty", Name: "oncall"},
		},
	})

	err := r.Notify(context.Background(), Notification{
		Title:     "Test",
		Severity:  "info",
		Timestamp: time.Now(),
	})

	// Should return an error but still deliver to working channel
	if err == nil {
		t.Fatal("expected error from failing channel")
	}
	if working.sentCount() != 1 {
		t.Fatal("working channel should still receive notification")
	}
}

var errTestDelivery = fmt.Errorf("test delivery error")

func TestRouterCRUD(t *testing.T) {
	r := NewRouter()

	// Add
	err := r.AddRoute(Route{ID: "a", Name: "route-a", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	// Duplicate ID
	err = r.AddRoute(Route{ID: "a", Name: "duplicate"})
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}

	// List
	routes := r.ListRoutes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	// Update
	err = r.UpdateRoute(Route{ID: "a", Name: "updated", Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	routes = r.ListRoutes()
	if routes[0].Name != "updated" {
		t.Fatal("route not updated")
	}

	// Remove
	err = r.RemoveRoute("a")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.ListRoutes()) != 0 {
		t.Fatal("route not removed")
	}

	// Remove non-existent
	err = r.RemoveRoute("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent route")
	}
}

func TestRouterHistory(t *testing.T) {
	r := NewRouter()
	ch := newMockChannel("slack")
	r.RegisterChannel("team", ch)

	r.AddRoute(Route{
		ID:      "h1",
		Name:    "hist",
		Enabled: true,
		Match:   RouteMatch{},
		Channels: []ChannelRef{
			{Type: "slack", Name: "team"},
		},
	})

	for i := 0; i < 5; i++ {
		r.Notify(context.Background(), Notification{
			Title:     "Alert",
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}

	history := r.History(3)
	if len(history) != 3 {
		t.Fatalf("expected 3 history records, got %d", len(history))
	}

	// Most recent first
	allHistory := r.History(0)
	if len(allHistory) != 5 {
		t.Fatalf("expected 5 total history records, got %d", len(allHistory))
	}
}

func TestRouterOnDemandChannelBuild(t *testing.T) {
	r := NewRouter()

	// Don't register channel ahead of time; let router build it from config
	r.AddRoute(Route{
		ID:      "ondemand",
		Name:    "on-demand-slack",
		Enabled: true,
		Match:   RouteMatch{},
		Channels: []ChannelRef{
			{
				Type: "slack",
				Name: "dynamic",
				Config: map[string]string{
					"webhook_url": "https://hooks.slack.com/test",
				},
			},
		},
	})

	// This will try to build the channel on-demand; it will fail to actually
	// send (no real server) but the channel should be created
	_ = r.Notify(context.Background(), Notification{
		Title:     "Test",
		Severity:  "info",
		Timestamp: time.Now(),
	})

	// Check that a channel was cached
	r.mu.RLock()
	_, exists := r.channels["slack:dynamic"]
	r.mu.RUnlock()
	if !exists {
		t.Fatal("expected on-demand channel to be cached")
	}
}

func TestRouterConcurrency(t *testing.T) {
	r := NewRouter()
	ch := newMockChannel("slack")
	r.RegisterChannel("team", ch)

	r.AddRoute(Route{
		ID:      "conc",
		Name:    "concurrent",
		Enabled: true,
		Match:   RouteMatch{},
		Channels: []ChannelRef{
			{Type: "slack", Name: "team"},
		},
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Notify(context.Background(), Notification{
				Title:     "Concurrent",
				Severity:  "info",
				Timestamp: time.Now(),
			})
		}()
	}
	wg.Wait()

	if ch.sentCount() != 50 {
		t.Fatalf("expected 50 notifications, got %d", ch.sentCount())
	}
}

func TestMatchesRoute(t *testing.T) {
	tests := []struct {
		name  string
		match RouteMatch
		notif Notification
		want  bool
	}{
		{
			name:  "empty match matches all",
			match: RouteMatch{},
			notif: Notification{Severity: "info", Service: "foo", Type: "incident"},
			want:  true,
		},
		{
			name:  "severity match",
			match: RouteMatch{Severities: []string{"critical", "warning"}},
			notif: Notification{Severity: "warning"},
			want:  true,
		},
		{
			name:  "severity mismatch",
			match: RouteMatch{Severities: []string{"critical"}},
			notif: Notification{Severity: "info"},
			want:  false,
		},
		{
			name:  "combined match",
			match: RouteMatch{Services: []string{"api"}, Severities: []string{"critical"}},
			notif: Notification{Service: "api", Severity: "critical"},
			want:  true,
		},
		{
			name:  "combined partial mismatch",
			match: RouteMatch{Services: []string{"api"}, Severities: []string{"critical"}},
			notif: Notification{Service: "api", Severity: "warning"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesRoute(tt.match, tt.notif)
			if got != tt.want {
				t.Errorf("matchesRoute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadRoutes(t *testing.T) {
	r := NewRouter()
	r.LoadRoutes([]Route{
		{Name: "a", Enabled: true},
		{ID: "b", Name: "b", Enabled: true},
	})

	routes := r.ListRoutes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	// First should have auto-generated ID
	if routes[0].ID == "" {
		t.Fatal("expected auto-generated ID")
	}
	if routes[1].ID != "b" {
		t.Fatal("expected preserved ID 'b'")
	}
}
