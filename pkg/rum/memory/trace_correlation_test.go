package memory

import (
	"testing"
	"time"

	rum "github.com/Viridian-Inc/cloudmock/pkg/rum"
)

// A RUM JS error group should carry the most-recent occurrence's backend trace
// id so it can be correlated to a distributed trace.
func TestErrorGroups_CarriesTraceID(t *testing.T) {
	s := NewStore(100)

	older := makeEvent(rum.EventJSError, "s1")
	older.Timestamp = time.Now().Add(-time.Minute)
	older.TraceID = "trace-old"
	older.JSError = &rum.JSErrorEvent{Message: "boom", Fingerprint: "fp1"}

	newer := makeEvent(rum.EventJSError, "s2")
	newer.Timestamp = time.Now()
	newer.TraceID = "trace-new"
	newer.JSError = &rum.JSErrorEvent{Message: "boom", Fingerprint: "fp1"}

	if err := s.WriteEvent(older); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteEvent(newer); err != nil {
		t.Fatal(err)
	}

	groups, err := s.ErrorGroups()
	if err != nil {
		t.Fatalf("ErrorGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	if groups[0].Count != 2 {
		t.Errorf("count = %d, want 2", groups[0].Count)
	}
	// Representative trace is the most recent occurrence's.
	if groups[0].TraceID != "trace-new" {
		t.Errorf("TraceID = %q, want trace-new (most recent)", groups[0].TraceID)
	}
}
