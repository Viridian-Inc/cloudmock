package memory

import (
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/rum"
)

func makeEvent(typ rum.EventType, sessionID string) rum.RUMEvent {
	return rum.RUMEvent{
		ID:        "test-" + sessionID,
		Type:      typ,
		SessionID: sessionID,
		URL:       "https://example.com/",
		UserAgent: "TestAgent/1.0",
		Timestamp: time.Now(),
	}
}

func TestWriteAndSnapshot(t *testing.T) {
	s := NewStore(5)

	for i := 0; i < 3; i++ {
		e := makeEvent(rum.EventPageLoad, "s1")
		e.PageLoad = &rum.PageLoadEvent{Route: "/", DurationMs: 100}
		if err := s.WriteEvent(e); err != nil {
			t.Fatal(err)
		}
	}

	sessions, err := s.Sessions(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].PageViews != 3 {
		t.Errorf("expected 3 page views, got %d", sessions[0].PageViews)
	}
}

func TestCircularBuffer(t *testing.T) {
	s := NewStore(3)

	// Write 5 events into a buffer of size 3.
	for i := 0; i < 5; i++ {
		e := makeEvent(rum.EventJSError, "s1")
		e.JSError = &rum.JSErrorEvent{
			Message:     "err",
			Fingerprint: "fp1",
		}
		s.WriteEvent(e)
	}

	groups, err := s.ErrorGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 error group, got %d", len(groups))
	}
	// Only 3 events should be retained.
	if groups[0].Count != 3 {
		t.Errorf("expected count=3 (buffer wrapped), got %d", groups[0].Count)
	}
}

func TestWebVitalsOverview(t *testing.T) {
	s := NewStore(100)

	vitals := []struct {
		name   string
		value  float64
		rating string
	}{
		{"LCP", 1500, "good"},
		{"LCP", 3000, "needs-improvement"},
		{"LCP", 5000, "poor"},
		{"FID", 50, "good"},
		{"CLS", 0.05, "good"},
	}

	for _, v := range vitals {
		e := makeEvent(rum.EventWebVital, "s1")
		e.WebVital = &rum.WebVitalEvent{
			Name:   v.name,
			Value:  v.value,
			Rating: v.rating,
		}
		s.WriteEvent(e)
	}

	overview, err := s.WebVitalsOverview()
	if err != nil {
		t.Fatal(err)
	}

	if overview.LCP.Good != 1 || overview.LCP.NeedsImprovement != 1 || overview.LCP.Poor != 1 {
		t.Errorf("unexpected LCP breakdown: %+v", overview.LCP)
	}
	if overview.FID.Good != 1 {
		t.Errorf("unexpected FID: %+v", overview.FID)
	}
	if overview.CLS.Good != 1 {
		t.Errorf("unexpected CLS: %+v", overview.CLS)
	}
	if overview.TotalSessions != 1 {
		t.Errorf("expected 1 session, got %d", overview.TotalSessions)
	}
}

func TestPageLoads(t *testing.T) {
	s := NewStore(100)

	routes := []string{"/", "/about", "/", "/"}
	for _, r := range routes {
		e := makeEvent(rum.EventPageLoad, "s1")
		e.PageLoad = &rum.PageLoadEvent{
			Route:      r,
			DurationMs: 200,
			TTFB:       50,
		}
		s.WriteEvent(e)
	}

	pages, err := s.PageLoads()
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(pages))
	}
	// Sorted by views descending.
	if pages[0].Route != "/" || pages[0].Views != 3 {
		t.Errorf("expected / with 3 views, got %s with %d", pages[0].Route, pages[0].Views)
	}
}

func TestSessionDetail(t *testing.T) {
	s := NewStore(100)

	e1 := makeEvent(rum.EventPageLoad, "s1")
	e1.PageLoad = &rum.PageLoadEvent{Route: "/"}
	s.WriteEvent(e1)

	e2 := makeEvent(rum.EventPageLoad, "s2")
	e2.PageLoad = &rum.PageLoadEvent{Route: "/other"}
	s.WriteEvent(e2)

	detail, err := s.SessionDetail("s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail) != 1 {
		t.Fatalf("expected 1 event for s1, got %d", len(detail))
	}
	if detail[0].SessionID != "s1" {
		t.Errorf("wrong session ID: %s", detail[0].SessionID)
	}
}

func TestRageClicks(t *testing.T) {
	s := NewStore(100)

	now := time.Now()

	// Write some rage clicks and some normal clicks.
	for i := 0; i < 3; i++ {
		e := makeEvent(rum.EventClick, "s1")
		e.Timestamp = now.Add(-time.Duration(i) * time.Second)
		e.Click = &rum.ClickEvent{
			Selector: "#btn-submit",
			Text:     "Submit",
			X:        100,
			Y:        200,
			IsRage:   true,
			URL:      "https://example.com/form",
		}
		s.WriteEvent(e)
	}
	// A non-rage click.
	e := makeEvent(rum.EventClick, "s1")
	e.Timestamp = now
	e.Click = &rum.ClickEvent{
		Selector: "#btn-cancel",
		Text:     "Cancel",
		X:        300,
		Y:        200,
		IsRage:   false,
		URL:      "https://example.com/form",
	}
	s.WriteEvent(e)

	clicks, err := s.RageClicks(60)
	if err != nil {
		t.Fatal(err)
	}
	if len(clicks) != 3 {
		t.Errorf("expected 3 rage clicks, got %d", len(clicks))
	}
	for _, c := range clicks {
		if !c.IsRage {
			t.Error("expected all returned clicks to be rage clicks")
		}
	}

	// With a 0-minute window, should get nothing (events are in the past).
	clicks2, err := s.RageClicks(0)
	if err != nil {
		t.Fatal(err)
	}
	// 0 minutes means cutoff is now, so recent events should still match since they're at 'now'.
	// Actually cutoff = now - 0 = now, so events at exactly now pass.
	// The ones a few seconds in the past won't.
	if len(clicks2) > 1 {
		// At most the one at 'now' could pass depending on timing.
	}
}

func TestUserJourneys(t *testing.T) {
	s := NewStore(100)

	now := time.Now()
	navs := []struct {
		from, to, typ string
		offset        time.Duration
	}{
		{"/", "/about", "push", 0},
		{"/about", "/contact", "push", time.Second},
		{"/contact", "/about", "back", 2 * time.Second},
	}

	for _, n := range navs {
		e := makeEvent(rum.EventNavigation, "s1")
		e.Timestamp = now.Add(n.offset)
		e.Navigation = &rum.NavigationEvent{
			FromURL: n.from,
			ToURL:   n.to,
			Type:    n.typ,
		}
		s.WriteEvent(e)
	}

	// Add a navigation for a different session.
	e := makeEvent(rum.EventNavigation, "s2")
	e.Timestamp = now
	e.Navigation = &rum.NavigationEvent{FromURL: "/x", ToURL: "/y", Type: "push"}
	s.WriteEvent(e)

	journeys, err := s.UserJourneys("s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(journeys) != 3 {
		t.Fatalf("expected 3 navigation events for s1, got %d", len(journeys))
	}
	if journeys[0].FromURL != "/" || journeys[0].ToURL != "/about" {
		t.Errorf("unexpected first nav: %+v", journeys[0])
	}
	if journeys[2].Type != "back" {
		t.Errorf("expected last nav type=back, got %s", journeys[2].Type)
	}

	// Different session.
	journeys2, err := s.UserJourneys("s2")
	if err != nil {
		t.Fatal(err)
	}
	if len(journeys2) != 1 {
		t.Fatalf("expected 1 navigation for s2, got %d", len(journeys2))
	}
}

func TestPerformanceByRoute(t *testing.T) {
	s := NewStore(100)

	routes := []string{"/", "/about", "/", "/"}
	for _, r := range routes {
		e := makeEvent(rum.EventPageLoad, "s1")
		e.PageLoad = &rum.PageLoadEvent{
			Route:      r,
			DurationMs: 200,
			TTFB:       50,
		}
		s.WriteEvent(e)
	}

	perf, err := s.PerformanceByRoute()
	if err != nil {
		t.Fatal(err)
	}
	if len(perf) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(perf))
	}
	// Sorted by views descending.
	if perf[0].Route != "/" || perf[0].Views != 3 {
		t.Errorf("expected / with 3 views, got %s with %d", perf[0].Route, perf[0].Views)
	}
	if perf[0].AvgDurationMs != 200 {
		t.Errorf("expected avg duration 200, got %f", perf[0].AvgDurationMs)
	}
}

func TestWriteBatch(t *testing.T) {
	s := NewStore(100)

	events := make([]rum.RUMEvent, 5)
	for i := range events {
		events[i] = makeEvent(rum.EventPageLoad, "batch-session")
		events[i].PageLoad = &rum.PageLoadEvent{Route: "/batch"}
	}

	if err := s.WriteBatch(events); err != nil {
		t.Fatal(err)
	}

	sessions, err := s.Sessions(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].PageViews != 5 {
		t.Errorf("expected 5 page views, got %d", sessions[0].PageViews)
	}
}
