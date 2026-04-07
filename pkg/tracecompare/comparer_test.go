package tracecompare

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
)

// --- mock TraceReader ---

type mockTraceReader struct {
	traces        map[string]*dataplane.TraceContext
	searchResults []dataplane.TraceSummary
}

func (m *mockTraceReader) Get(_ context.Context, traceID string) (*dataplane.TraceContext, error) {
	tc, ok := m.traces[traceID]
	if !ok {
		return nil, dataplane.ErrNotFound
	}
	return tc, nil
}

func (m *mockTraceReader) Search(_ context.Context, _ dataplane.TraceFilter) ([]dataplane.TraceSummary, error) {
	return m.searchResults, nil
}

func (m *mockTraceReader) Timeline(_ context.Context, _ string) ([]dataplane.TimelineSpan, error) {
	return nil, nil
}

// --- helpers ---

func makeTrace(traceID, service, action string, durationMs float64, statusCode int, children ...*dataplane.TraceContext) *dataplane.TraceContext {
	return &dataplane.TraceContext{
		TraceID:    traceID,
		SpanID:     traceID + "-span",
		Service:    service,
		Action:     action,
		DurationMs: durationMs,
		StatusCode: statusCode,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Duration(durationMs) * time.Millisecond),
		Duration:   time.Duration(durationMs) * time.Millisecond,
		Children:   children,
	}
}

func makeSpan(service, action string, durationMs float64, statusCode int, children ...*dataplane.TraceContext) *dataplane.TraceContext {
	return &dataplane.TraceContext{
		SpanID:     service + "-" + action,
		Service:    service,
		Action:     action,
		DurationMs: durationMs,
		StatusCode: statusCode,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(time.Duration(durationMs) * time.Millisecond),
		Duration:   time.Duration(durationMs) * time.Millisecond,
		Children:   children,
	}
}

func makeSpanWithMeta(service, action string, durationMs float64, statusCode int, meta map[string]string) *dataplane.TraceContext {
	tc := makeSpan(service, action, durationMs, statusCode)
	tc.Metadata = meta
	return tc
}

func makeSpanWithError(service, action string, durationMs float64, statusCode int, errMsg string) *dataplane.TraceContext {
	tc := makeSpan(service, action, durationMs, statusCode)
	tc.Error = errMsg
	return tc
}

// --- tests ---

func TestCompare_IdenticalTraces(t *testing.T) {
	traceA := makeTrace("t1", "gateway", "handle", 100, 200,
		makeSpan("auth", "verify", 20, 200),
		makeSpan("db", "query", 50, 200),
	)
	traceB := makeTrace("t2", "gateway", "handle", 100, 200,
		makeSpan("auth", "verify", 20, 200),
		makeSpan("db", "query", 50, 200),
	)

	reader := &mockTraceReader{traces: map[string]*dataplane.TraceContext{"t1": traceA, "t2": traceB}}
	cmp := New(reader)

	result, err := cmp.Compare(context.Background(), "t1", "t2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(result.Matches))
	}
	if len(result.OnlyInA) != 0 || len(result.OnlyInB) != 0 {
		t.Fatalf("expected no unmatched spans, got onlyA=%d onlyB=%d", len(result.OnlyInA), len(result.OnlyInB))
	}
	for _, m := range result.Matches {
		if m.LatencyDelta != 0 {
			t.Errorf("expected zero delta for %s:%s, got %f", m.Service, m.Action, m.LatencyDelta)
		}
		if m.LatencyPct != 0 {
			t.Errorf("expected zero pct for %s:%s, got %f", m.Service, m.Action, m.LatencyPct)
		}
	}
}

func TestCompare_DifferentLatencies(t *testing.T) {
	traceA := makeTrace("t1", "gateway", "handle", 100, 200,
		makeSpan("db", "query", 50, 200),
	)
	traceB := makeTrace("t2", "gateway", "handle", 150, 200,
		makeSpan("db", "query", 80, 200),
	)

	reader := &mockTraceReader{traces: map[string]*dataplane.TraceContext{"t1": traceA, "t2": traceB}}
	result, err := New(reader).Compare(context.Background(), "t1", "t2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the db:query match.
	var dbMatch *SpanMatch
	for i := range result.Matches {
		if result.Matches[i].Service == "db" && result.Matches[i].Action == "query" {
			dbMatch = &result.Matches[i]
			break
		}
	}
	if dbMatch == nil {
		t.Fatal("db:query match not found")
	}
	if dbMatch.LatencyDelta != 30 {
		t.Errorf("expected delta 30, got %f", dbMatch.LatencyDelta)
	}
	expectedPct := (30.0 / 50.0) * 100
	if dbMatch.LatencyPct != expectedPct {
		t.Errorf("expected pct %f, got %f", expectedPct, dbMatch.LatencyPct)
	}
	if result.Summary.SlowerSpans != 2 {
		t.Errorf("expected 2 slower spans, got %d", result.Summary.SlowerSpans)
	}
}

func TestCompare_AddedAndRemovedSpans(t *testing.T) {
	traceA := makeTrace("t1", "gateway", "handle", 100, 200,
		makeSpan("auth", "verify", 20, 200),
	)
	traceB := makeTrace("t2", "gateway", "handle", 100, 200,
		makeSpan("cache", "get", 5, 200),
	)

	reader := &mockTraceReader{traces: map[string]*dataplane.TraceContext{"t1": traceA, "t2": traceB}}
	result, err := New(reader).Compare(context.Background(), "t1", "t2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.OnlyInA) != 1 || result.OnlyInA[0].Service != "auth" {
		t.Errorf("expected auth in OnlyInA, got %+v", result.OnlyInA)
	}
	if len(result.OnlyInB) != 1 || result.OnlyInB[0].Service != "cache" {
		t.Errorf("expected cache in OnlyInB, got %+v", result.OnlyInB)
	}
	if result.Summary.AddedSpans != 1 {
		t.Errorf("expected 1 added span, got %d", result.Summary.AddedSpans)
	}
	if result.Summary.RemovedSpans != 1 {
		t.Errorf("expected 1 removed span, got %d", result.Summary.RemovedSpans)
	}
}

func TestCompare_DuplicateSpans(t *testing.T) {
	traceA := makeTrace("t1", "gateway", "handle", 100, 200,
		makeSpan("db", "query", 10, 200),
		makeSpan("db", "query", 30, 200),
	)
	traceB := makeTrace("t2", "gateway", "handle", 100, 200,
		makeSpan("db", "query", 15, 200),
		makeSpan("db", "query", 40, 200),
	)

	reader := &mockTraceReader{traces: map[string]*dataplane.TraceContext{"t1": traceA, "t2": traceB}}
	result, err := New(reader).Compare(context.Background(), "t1", "t2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should match: gateway:handle (1), db:query x2 = 3 total matches.
	dbMatches := 0
	for _, m := range result.Matches {
		if m.Service == "db" && m.Action == "query" {
			dbMatches++
		}
	}
	if dbMatches != 2 {
		t.Errorf("expected 2 db:query matches, got %d", dbMatches)
	}

	// First db:query: 10 -> 15, second: 30 -> 40.
	var deltas []float64
	for _, m := range result.Matches {
		if m.Service == "db" && m.Action == "query" {
			deltas = append(deltas, m.LatencyDelta)
		}
	}
	if len(deltas) != 2 || deltas[0] != 5 || deltas[1] != 10 {
		t.Errorf("expected deltas [5 10], got %v", deltas)
	}
}

func TestCompare_MetadataDiff(t *testing.T) {
	spanA := makeSpanWithMeta("db", "query", 50, 200, map[string]string{
		"table": "users",
		"rows":  "10",
	})
	spanB := makeSpanWithMeta("db", "query", 50, 200, map[string]string{
		"table": "users",
		"rows":  "25",
		"cache": "miss",
	})

	traceA := makeTrace("t1", "gateway", "handle", 100, 200, spanA)
	traceB := makeTrace("t2", "gateway", "handle", 100, 200, spanB)

	reader := &mockTraceReader{traces: map[string]*dataplane.TraceContext{"t1": traceA, "t2": traceB}}
	result, err := New(reader).Compare(context.Background(), "t1", "t2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var dbMatch *SpanMatch
	for i := range result.Matches {
		if result.Matches[i].Service == "db" {
			dbMatch = &result.Matches[i]
			break
		}
	}
	if dbMatch == nil {
		t.Fatal("db:query match not found")
	}
	if dbMatch.MetadataDiff == nil {
		t.Fatal("expected metadata diff, got nil")
	}
	if diff, ok := dbMatch.MetadataDiff["rows"]; !ok || diff[0] != "10" || diff[1] != "25" {
		t.Errorf("expected rows diff [10 25], got %v", dbMatch.MetadataDiff["rows"])
	}
	if diff, ok := dbMatch.MetadataDiff["cache"]; !ok || diff[0] != "" || diff[1] != "miss" {
		t.Errorf("expected cache diff ['' miss], got %v", dbMatch.MetadataDiff["cache"])
	}
	// "table" should NOT be in the diff since it's identical.
	if _, ok := dbMatch.MetadataDiff["table"]; ok {
		t.Error("table should not be in metadata diff")
	}
}

func TestCompare_StatusChange(t *testing.T) {
	traceA := makeTrace("t1", "gateway", "handle", 100, 200,
		makeSpanWithError("db", "query", 50, 200, ""),
	)
	traceB := makeTrace("t2", "gateway", "handle", 100, 500,
		makeSpanWithError("db", "query", 50, 500, "connection refused"),
	)

	reader := &mockTraceReader{traces: map[string]*dataplane.TraceContext{"t1": traceA, "t2": traceB}}
	result, err := New(reader).Compare(context.Background(), "t1", "t2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statusChanges := 0
	for _, m := range result.Matches {
		if m.StatusChange {
			statusChanges++
		}
	}
	// gateway (200->500) and db (200->500) both changed.
	if statusChanges != 2 {
		t.Errorf("expected 2 status changes, got %d", statusChanges)
	}

	// Check the db match has the error.
	for _, m := range result.Matches {
		if m.Service == "db" {
			if m.B.Error != "connection refused" {
				t.Errorf("expected error 'connection refused', got %q", m.B.Error)
			}
		}
	}
}

func TestCompare_TraceNotFound(t *testing.T) {
	reader := &mockTraceReader{traces: map[string]*dataplane.TraceContext{}}
	_, err := New(reader).Compare(context.Background(), "missing", "also-missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, dataplane.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCompare_Summary(t *testing.T) {
	traceA := makeTrace("t1", "gateway", "handle", 100, 200,
		makeSpan("auth", "verify", 20, 200),
		makeSpan("db", "query", 50, 200),
	)
	// B is slower overall: db:query is 30ms slower, auth is 5ms faster.
	traceB := makeTrace("t2", "gateway", "handle", 125, 200,
		makeSpan("auth", "verify", 15, 200),
		makeSpan("db", "query", 80, 200),
	)

	reader := &mockTraceReader{traces: map[string]*dataplane.TraceContext{"t1": traceA, "t2": traceB}}
	result, err := New(reader).Compare(context.Background(), "t1", "t2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := result.Summary
	if s.TotalLatencyA != 100 {
		t.Errorf("expected totalA=100, got %f", s.TotalLatencyA)
	}
	if s.TotalLatencyB != 125 {
		t.Errorf("expected totalB=125, got %f", s.TotalLatencyB)
	}
	if s.LatencyDelta != 25 {
		t.Errorf("expected delta=25, got %f", s.LatencyDelta)
	}
	// gateway: +25, db: +30 -> critical path is db:query (largest positive delta).
	if s.CriticalPath != "db:query" {
		t.Errorf("expected critical path 'db:query', got %q", s.CriticalPath)
	}
	// gateway +25, db +30 = 2 slower; auth -5 = 1 faster.
	if s.SlowerSpans != 2 {
		t.Errorf("expected 2 slower, got %d", s.SlowerSpans)
	}
	if s.FasterSpans != 1 {
		t.Errorf("expected 1 faster, got %d", s.FasterSpans)
	}
}

func TestCompareBaseline(t *testing.T) {
	// Target trace.
	target := makeTrace("target", "gateway", "handle", 120, 200,
		makeSpan("db", "query", 80, 200),
	)

	// 5 similar traces with varying latencies.
	similar := make(map[string]*dataplane.TraceContext)
	summaries := []dataplane.TraceSummary{}
	durations := []float64{100, 110, 90, 105, 95}
	dbDurations := []float64{60, 70, 50, 65, 55}

	for i := 0; i < 5; i++ {
		id := "sim-" + string(rune('a'+i))
		similar[id] = makeTrace(id, "gateway", "handle", durations[i], 200,
			makeSpan("db", "query", dbDurations[i], 200),
		)
		summaries = append(summaries, dataplane.TraceSummary{
			TraceID:     id,
			RootService: "gateway",
			RootAction:  "handle",
			DurationMs:  durations[i],
		})
	}

	traces := map[string]*dataplane.TraceContext{"target": target}
	for k, v := range similar {
		traces[k] = v
	}

	reader := &mockTraceReader{traces: traces, searchResults: summaries}
	result, err := New(reader).CompareBaseline(context.Background(), "target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TraceA != "baseline" {
		t.Errorf("expected TraceA='baseline', got %q", result.TraceA)
	}
	if result.TraceB != "target" {
		t.Errorf("expected TraceB='target', got %q", result.TraceB)
	}

	// P50 of db durations [50 55 60 65 70] = 60.
	// P50 of gateway durations [90 95 100 105 110] = 100.
	// Target db=80, so delta = 80 - 60 = 20.
	var dbMatch *SpanMatch
	for i := range result.Matches {
		if result.Matches[i].Service == "db" {
			dbMatch = &result.Matches[i]
			break
		}
	}
	if dbMatch == nil {
		t.Fatal("db:query baseline match not found")
	}
	if dbMatch.A.DurationMs != 60 {
		t.Errorf("expected baseline db P50=60, got %f", dbMatch.A.DurationMs)
	}
	if dbMatch.LatencyDelta != 20 {
		t.Errorf("expected db delta=20, got %f", dbMatch.LatencyDelta)
	}
}

func TestCompareBaseline_NoSimilarTraces(t *testing.T) {
	target := makeTrace("target", "gateway", "handle", 100, 200)

	reader := &mockTraceReader{
		traces:        map[string]*dataplane.TraceContext{"target": target},
		searchResults: []dataplane.TraceSummary{},
	}
	_, err := New(reader).CompareBaseline(context.Background(), "target")
	if err == nil {
		t.Fatal("expected error for no similar traces")
	}
	if err.Error() != "no similar traces found for baseline" {
		t.Errorf("unexpected error message: %v", err)
	}
}
