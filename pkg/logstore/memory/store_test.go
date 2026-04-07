package memory

import (
	"context"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/logstore"
)

func TestWriteAndQuery(t *testing.T) {
	s := NewStore(100)

	now := time.Now()
	s.Write(logstore.LogEntry{
		ID:        "1",
		Timestamp: now.Add(-2 * time.Second),
		Level:     "info",
		Message:   "starting service",
		Service:   "api",
	})
	s.Write(logstore.LogEntry{
		ID:        "2",
		Timestamp: now.Add(-time.Second),
		Level:     "error",
		Message:   "connection refused",
		Service:   "api",
	})
	s.Write(logstore.LogEntry{
		ID:        "3",
		Timestamp: now,
		Level:     "info",
		Message:   "healthy check ok",
		Service:   "worker",
	})

	// Query all.
	all, err := s.Query(logstore.QueryOpts{Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}

	// Filter by level.
	errors, _ := s.Query(logstore.QueryOpts{Level: "error", Limit: 10})
	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}

	// Filter by service.
	worker, _ := s.Query(logstore.QueryOpts{Service: "worker", Limit: 10})
	if len(worker) != 1 {
		t.Errorf("expected 1 worker log, got %d", len(worker))
	}

	// Full-text search.
	search, _ := s.Query(logstore.QueryOpts{Search: "refused", Limit: 10})
	if len(search) != 1 {
		t.Errorf("expected 1 search match, got %d", len(search))
	}
}

func TestWriteBatch(t *testing.T) {
	s := NewStore(100)

	entries := []logstore.LogEntry{
		{ID: "a", Level: "info", Message: "one", Timestamp: time.Now()},
		{ID: "b", Level: "warn", Message: "two", Timestamp: time.Now()},
	}
	if err := s.WriteBatch(entries); err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}

	all, _ := s.Query(logstore.QueryOpts{Limit: 10})
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}

func TestQueryTimeRange(t *testing.T) {
	s := NewStore(100)

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	s.Write(logstore.LogEntry{ID: "1", Timestamp: base, Level: "info", Message: "old"})
	s.Write(logstore.LogEntry{ID: "2", Timestamp: base.Add(time.Hour), Level: "info", Message: "mid"})
	s.Write(logstore.LogEntry{ID: "3", Timestamp: base.Add(2 * time.Hour), Level: "info", Message: "new"})

	results, _ := s.Query(logstore.QueryOpts{
		StartTime: base.Add(30 * time.Minute),
		EndTime:   base.Add(90 * time.Minute),
		Limit:     10,
	})
	if len(results) != 1 {
		t.Errorf("expected 1 result in time range, got %d", len(results))
	}
}

func TestTail(t *testing.T) {
	s := NewStore(100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := s.Tail(ctx, logstore.TailFilter{Level: "error"})

	// Write some entries after subscribing.
	s.Write(logstore.LogEntry{ID: "1", Level: "info", Message: "skip me", Timestamp: time.Now()})
	s.Write(logstore.LogEntry{ID: "2", Level: "error", Message: "catch me", Timestamp: time.Now()})

	select {
	case entry := <-ch:
		if entry.Message != "catch me" {
			t.Errorf("expected 'catch me', got %q", entry.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tail entry")
	}
}

func TestServices(t *testing.T) {
	s := NewStore(100)

	s.Write(logstore.LogEntry{ID: "1", Service: "api", Level: "info", Message: "a", Timestamp: time.Now()})
	s.Write(logstore.LogEntry{ID: "2", Service: "worker", Level: "info", Message: "b", Timestamp: time.Now()})
	s.Write(logstore.LogEntry{ID: "3", Service: "api", Level: "info", Message: "c", Timestamp: time.Now()})

	svcs, err := s.Services()
	if err != nil {
		t.Fatalf("Services: %v", err)
	}
	if len(svcs) != 2 {
		t.Errorf("expected 2 services, got %d", len(svcs))
	}
}

func TestLevelCounts(t *testing.T) {
	s := NewStore(100)

	s.Write(logstore.LogEntry{ID: "1", Level: "info", Message: "a", Timestamp: time.Now()})
	s.Write(logstore.LogEntry{ID: "2", Level: "error", Message: "b", Timestamp: time.Now()})
	s.Write(logstore.LogEntry{ID: "3", Level: "info", Message: "c", Timestamp: time.Now()})

	counts, err := s.LevelCounts()
	if err != nil {
		t.Fatalf("LevelCounts: %v", err)
	}
	if counts["info"] != 2 {
		t.Errorf("expected 2 info, got %d", counts["info"])
	}
	if counts["error"] != 1 {
		t.Errorf("expected 1 error, got %d", counts["error"])
	}
}

func TestCircularBufferOverflow(t *testing.T) {
	s := NewStore(3)

	for i := 0; i < 5; i++ {
		s.Write(logstore.LogEntry{
			ID:        "x",
			Level:     "info",
			Message:   "msg",
			Timestamp: time.Now(),
		})
	}

	all, _ := s.Query(logstore.QueryOpts{Limit: 10})
	if len(all) != 3 {
		t.Errorf("expected 3 after overflow, got %d", len(all))
	}
}
