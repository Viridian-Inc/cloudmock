package memory

import (
	"testing"
	"time"

	errs "github.com/Viridian-Inc/cloudmock/pkg/errors"
)

func TestIngestAndGetGroups(t *testing.T) {
	s := NewStore(100)

	ev := errs.ErrorEvent{
		Message:   "cannot read property 'foo' of undefined",
		Stack:     "at App.render (app.js:42)\nat React.render (react.js:100)\nat mount (react-dom.js:50)",
		Service:   "frontend",
		SessionID: "sess-1",
		Timestamp: time.Now(),
	}

	if err := s.IngestError(ev); err != nil {
		t.Fatalf("IngestError: %v", err)
	}

	// Ingest a second occurrence of the same error.
	ev.SessionID = "sess-2"
	if err := s.IngestError(ev); err != nil {
		t.Fatalf("IngestError: %v", err)
	}

	groups, err := s.GetGroups("", 10)
	if err != nil {
		t.Fatalf("GetGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Count != 2 {
		t.Errorf("expected count=2, got %d", groups[0].Count)
	}
	if groups[0].Status != "unresolved" {
		t.Errorf("expected status=unresolved, got %q", groups[0].Status)
	}
}

func TestGetGroupNotFound(t *testing.T) {
	s := NewStore(100)
	_, err := s.GetGroup("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing group")
	}
}

func TestGetEvents(t *testing.T) {
	s := NewStore(100)

	ev1 := errs.ErrorEvent{
		Message:   "error A",
		Stack:     "frame1\nframe2\nframe3",
		Service:   "svc-a",
		Timestamp: time.Now().Add(-time.Second),
	}
	ev2 := errs.ErrorEvent{
		Message:   "error A",
		Stack:     "frame1\nframe2\nframe3",
		Service:   "svc-a",
		Timestamp: time.Now(),
	}

	s.IngestError(ev1)
	s.IngestError(ev2)

	fp := errs.Fingerprint(ev1.Message, ev1.Stack)
	events, err := s.GetEvents(fp, 10)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestUpdateGroupStatus(t *testing.T) {
	s := NewStore(100)

	s.IngestError(errs.ErrorEvent{
		Message:   "test error",
		Stack:     "at foo (bar.js:1)",
		Timestamp: time.Now(),
	})

	groups, _ := s.GetGroups("", 10)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	if err := s.UpdateGroupStatus(groups[0].ID, "resolved"); err != nil {
		t.Fatalf("UpdateGroupStatus: %v", err)
	}

	g, _ := s.GetGroup(groups[0].ID)
	if g.Status != "resolved" {
		t.Errorf("expected resolved, got %q", g.Status)
	}

	// Filter by status.
	resolved, _ := s.GetGroups("resolved", 10)
	if len(resolved) != 1 {
		t.Errorf("expected 1 resolved group, got %d", len(resolved))
	}
	unresolved, _ := s.GetGroups("unresolved", 10)
	if len(unresolved) != 0 {
		t.Errorf("expected 0 unresolved groups, got %d", len(unresolved))
	}
}

func TestCircularBufferOverflow(t *testing.T) {
	s := NewStore(3)

	for i := 0; i < 5; i++ {
		s.IngestError(errs.ErrorEvent{
			Message:   "overflow test",
			Stack:     "at x (y.js:1)",
			Timestamp: time.Now(),
		})
	}

	fp := errs.Fingerprint("overflow test", "at x (y.js:1)")
	events, _ := s.GetEvents(fp, 10)
	// Only 3 should be retained (buffer capacity).
	if len(events) != 3 {
		t.Errorf("expected 3 events after overflow, got %d", len(events))
	}
}

func TestDifferentErrorsDifferentGroups(t *testing.T) {
	s := NewStore(100)

	s.IngestError(errs.ErrorEvent{
		Message:   "error A",
		Stack:     "at a (a.js:1)",
		Timestamp: time.Now(),
	})
	s.IngestError(errs.ErrorEvent{
		Message:   "error B",
		Stack:     "at b (b.js:1)",
		Timestamp: time.Now(),
	})

	groups, _ := s.GetGroups("", 10)
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}
