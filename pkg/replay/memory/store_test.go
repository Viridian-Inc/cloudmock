package memory

import (
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/replay"
)

func makeSession(id string) replay.Session {
	return replay.Session{
		ID:        id,
		URL:       "https://example.com/",
		UserAgent: "TestAgent/1.0",
		StartedAt: time.Now(),
		Duration:  5000,
		Width:     1920,
		Height:    1080,
		Events: []replay.ReplayEvent{
			{Type: "mutation", Timestamp: 0, Data: map[string]any{"html": "<div>test</div>"}},
			{Type: "click", Timestamp: 100, Data: map[string]any{"x": 10, "y": 20}},
		},
	}
}

func TestSaveAndGet(t *testing.T) {
	s := NewStore(10)

	sess := makeSession("sess-1")
	if err := s.SaveSession(sess); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetSession("sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.ID != "sess-1" {
		t.Errorf("expected ID sess-1, got %s", got.ID)
	}
	if len(got.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(got.Events))
	}
}

func TestGetNotFound(t *testing.T) {
	s := NewStore(10)
	got, err := s.GetSession("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestListSessionsNewestFirst(t *testing.T) {
	s := NewStore(10)

	for i := 0; i < 5; i++ {
		sess := makeSession("sess-" + time.Now().Format("150405.000000000"))
		sess.StartedAt = time.Now()
		time.Sleep(time.Millisecond)
		s.SaveSession(sess)
	}

	list, err := s.ListSessions(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(list))
	}
	// Verify newest first.
	if list[0].StartedAt.Before(list[1].StartedAt) {
		t.Error("expected sessions in newest-first order")
	}
}

func TestCircularBufferEviction(t *testing.T) {
	s := NewStore(3)

	// Write 5 sessions into a buffer of size 3.
	for i := 0; i < 5; i++ {
		sess := makeSession("sess-" + string(rune('a'+i)))
		s.SaveSession(sess)
	}

	// Only the last 3 should be accessible.
	list, err := s.ListSessions(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(list))
	}

	// First two should have been evicted.
	got, _ := s.GetSession("sess-a")
	if got != nil {
		t.Error("expected sess-a to be evicted")
	}
	got, _ = s.GetSession("sess-b")
	if got != nil {
		t.Error("expected sess-b to be evicted")
	}

	// Last three should still be present.
	got, _ = s.GetSession("sess-c")
	if got == nil {
		t.Error("expected sess-c to still exist")
	}
	got, _ = s.GetSession("sess-d")
	if got == nil {
		t.Error("expected sess-d to still exist")
	}
	got, _ = s.GetSession("sess-e")
	if got == nil {
		t.Error("expected sess-e to still exist")
	}
}

func TestLinkError(t *testing.T) {
	s := NewStore(10)

	sess := makeSession("sess-1")
	s.SaveSession(sess)

	if err := s.LinkError("sess-1", "err-abc"); err != nil {
		t.Fatal(err)
	}
	// Linking same error again should be idempotent.
	if err := s.LinkError("sess-1", "err-abc"); err != nil {
		t.Fatal(err)
	}

	got, _ := s.GetSession("sess-1")
	if len(got.ErrorIDs) != 1 {
		t.Errorf("expected 1 error ID, got %d", len(got.ErrorIDs))
	}
	if got.ErrorIDs[0] != "err-abc" {
		t.Errorf("expected err-abc, got %s", got.ErrorIDs[0])
	}
}

func TestLinkErrorNotFound(t *testing.T) {
	s := NewStore(10)
	err := s.LinkError("nonexistent", "err-1")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}
