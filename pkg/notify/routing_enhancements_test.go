package notify

import (
	"context"
	"sync"
	"testing"
	"time"
)

type recordChannel struct {
	typ   string
	mu    sync.Mutex
	sends []Notification
}

func (c *recordChannel) Type() string { return c.typ }
func (c *recordChannel) Send(_ context.Context, n Notification) error {
	c.mu.Lock()
	c.sends = append(c.sends, n)
	c.mu.Unlock()
	return nil
}
func (c *recordChannel) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.sends)
}

func routerWithRoute(ch *recordChannel, name string, match RouteMatch) *Router {
	r := NewRouter()
	r.RegisterChannel(name, ch)
	r.AddRoute(Route{
		Name:     "r",
		Match:    match,
		Channels: []ChannelRef{{Type: ch.typ, Name: name}},
		Enabled:  true,
	})
	return r
}

func TestMaintenanceWindow_Suppresses(t *testing.T) {
	ch := &recordChannel{typ: "slack"}
	r := routerWithRoute(ch, "ops", RouteMatch{Services: []string{"api"}})

	// Active window scoped to "api" suppresses delivery.
	now := time.Now()
	r.AddWindow(MaintenanceWindow{
		Name: "deploy", Start: now.Add(-time.Hour), End: now.Add(time.Hour),
		Services: []string{"api"},
	})
	if err := r.Notify(context.Background(), Notification{Service: "api", Severity: "warning"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if ch.count() != 0 {
		t.Errorf("expected suppression (0 sends), got %d", ch.count())
	}
	// A "suppressed" delivery record should exist.
	suppressed := false
	for _, rec := range r.History(10) {
		if rec.Status == "suppressed" {
			suppressed = true
		}
	}
	if !suppressed {
		t.Error("expected a suppressed delivery record")
	}
}

func TestMaintenanceWindow_OutOfScopeAndExpired(t *testing.T) {
	ch := &recordChannel{typ: "slack"}
	r := routerWithRoute(ch, "ops", RouteMatch{Services: []string{"api"}})
	now := time.Now()

	// Expired window must not suppress.
	r.AddWindow(MaintenanceWindow{Name: "old", Start: now.Add(-2 * time.Hour), End: now.Add(-time.Hour)})
	if err := r.Notify(context.Background(), Notification{Service: "api"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if ch.count() != 1 {
		t.Errorf("expired window should not suppress; sends = %d, want 1", ch.count())
	}

	// Window scoped to a different service must not suppress "api".
	r.AddWindow(MaintenanceWindow{Name: "other", Start: now.Add(-time.Hour), End: now.Add(time.Hour), Services: []string{"billing"}})
	if err := r.Notify(context.Background(), Notification{Service: "api"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if ch.count() != 2 {
		t.Errorf("out-of-scope window should not suppress; sends = %d, want 2", ch.count())
	}
}

func TestTeamRouting_DeliversAndDedups(t *testing.T) {
	teamCh := &recordChannel{typ: "slack"}
	r := NewRouter()
	r.RegisterChannel("team-a", teamCh)
	r.SetTeam(Team{Name: "team-a", Channels: []ChannelRef{{Type: "slack", Name: "team-a"}}}, []string{"api"})

	// No direct route — the owning team's channel still receives it.
	if err := r.Notify(context.Background(), Notification{Service: "api"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if teamCh.count() != 1 {
		t.Fatalf("team channel sends = %d, want 1", teamCh.count())
	}

	// A service with no owning team gets nothing from team routing.
	if err := r.Notify(context.Background(), Notification{Service: "unowned"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if teamCh.count() != 1 {
		t.Errorf("unowned service must not hit team channel; sends = %d, want 1", teamCh.count())
	}

	// Dedup: a direct route to the same channel means the team loop skips it (1 send, not 2).
	r.AddRoute(Route{Name: "direct", Match: RouteMatch{Services: []string{"api"}}, Channels: []ChannelRef{{Type: "slack", Name: "team-a"}}, Enabled: true})
	before := teamCh.count()
	if err := r.Notify(context.Background(), Notification{Service: "api"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if got := teamCh.count() - before; got != 1 {
		t.Errorf("route+team to same channel should send once; got %d sends", got)
	}
}
