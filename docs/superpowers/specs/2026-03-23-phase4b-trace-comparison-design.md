# Phase 4b: Trace Comparison — Design Specification

**Date:** 2026-03-23
**Status:** Approved
**Phase:** 4b of 6 (CloudMock Console — Intelligence Layer, sub-project 2 of 5)
**Depends on:** Phase 3 (Production Data Plane)

---

## Overview

Side-by-side trace comparison that aligns spans by `(service, action)`, computes per-span latency deltas and metadata diffs, and identifies added/removed spans. Supports two modes: trace-vs-trace and trace-vs-route-baseline. Entirely storage-agnostic — uses only the `TraceReader` interface, works in both local and production modes.

### Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Comparison output | Structural diff + per-span stats | Shows both shape changes and magnitude changes |
| Baseline computation | On-demand via TraceReader | No ClickHouse-specific code; works in local + production |
| API scope | Single endpoint with baseline flag | YAGNI on standalone baseline endpoint |
| Architecture | Pure computation, no stored state | Stateless, single package, 3 files |

---

## 1. Data Model

```go
// pkg/tracecompare/types.go

type TraceComparison struct {
    TraceA       string            // trace ID (or "baseline")
    TraceB       string            // trace ID
    Matches      []SpanMatch       // aligned spans present in both
    OnlyInA      []SpanSummary     // spans in A but not B
    OnlyInB      []SpanSummary     // spans in B but not A
    Summary      ComparisonSummary
}

type SpanMatch struct {
    Service      string
    Action       string
    A            SpanStats
    B            SpanStats
    LatencyDelta float64           // B - A in ms (positive = slower)
    LatencyPct   float64           // percent change
    StatusChange bool              // status codes differ
    MetadataDiff map[string][2]string // key → [A value, B value]
}

type SpanStats struct {
    DurationMs   float64
    StatusCode   int
    Error        string
    Metadata     map[string]string
}

type SpanSummary struct {
    Service    string
    Action     string
    DurationMs float64
    StatusCode int
}

type ComparisonSummary struct {
    TotalLatencyA float64  // total trace duration A
    TotalLatencyB float64  // total trace duration B
    LatencyDelta  float64  // B - A
    SlowerSpans   int      // count of spans that got slower
    FasterSpans   int      // count that got faster
    AddedSpans    int      // in B only
    RemovedSpans  int      // in A only
    CriticalPath  string   // service with largest absolute slowdown
}
```

---

## 2. Span Alignment Algorithm

Spans are flattened from the trace tree and matched by `(service, action)` tuple. If a trace has multiple spans with the same `(service, action)` — e.g., two DynamoDB Query calls — they're matched in order of appearance (by start time). Unmatched spans go to `OnlyInA` or `OnlyInB`.

For each matched pair, compute:
- `LatencyDelta = B.DurationMs - A.DurationMs`
- `LatencyPct = (B.DurationMs - A.DurationMs) / A.DurationMs * 100`
- `StatusChange = A.StatusCode != B.StatusCode`
- `MetadataDiff` — keys where values differ between A and B

---

## 3. Baseline Computation

For baseline mode, "trace A" is synthesized from recent historical traces for the same route:

1. Get the target trace via `TraceReader.Get(traceID)` to determine its root `service`, `action`, `method`
2. Use `TraceReader.Search(filter)` to find recent traces matching that root (limit 20)
3. Fetch each via `TraceReader.Get()` and flatten their spans
4. For each unique `(service, action)`, compute P50 latency and most common status code
5. Synthesize a virtual trace from these P50 spans
6. Run the same alignment/diff logic against this virtual trace

This works in both local mode (in-memory TraceStore) and production mode (ClickHouse via TraceReader). No storage-specific code needed.

---

## 4. Comparer

```go
// pkg/tracecompare/comparer.go

type Comparer struct {
    traces dataplane.TraceReader
}

func New(traces dataplane.TraceReader) *Comparer

func (c *Comparer) Compare(ctx context.Context, traceA, traceB string) (*TraceComparison, error)

func (c *Comparer) CompareBaseline(ctx context.Context, traceID string) (*TraceComparison, error)
```

`Compare` fetches both traces, flattens spans, aligns, computes deltas.

`CompareBaseline` fetches the target trace, queries for similar recent traces, synthesizes a baseline, then runs the same comparison logic.

Both return `dataplane.ErrNotFound` if a trace doesn't exist.

---

## 5. API

```
GET /api/traces/compare?a={traceID}&b={traceID}     — compare two traces
GET /api/traces/compare?a={traceID}&baseline=true    — compare against route baseline
```

Returns `TraceComparison` as JSON. Errors:
- 400 if `a` param missing
- 400 if neither `b` nor `baseline=true` provided
- 404 if trace not found

### Wiring

- Add `handleTraceCompare` handler to admin API in `pkg/admin/api.go`
- Register route in `NewWithDataPlane()`: `/api/traces/compare`
- Create `Comparer` in `cmd/gateway/main.go` with `dp.Traces`
- Pass to admin API via `SetTraceComparer()`

---

## 6. Testing

**Unit tests (`comparer_test.go`):**
- Two identical traces → zero deltas, no added/removed
- Two different traces → correct deltas, matches, OnlyInA/OnlyInB
- Trace with duplicate `(service, action)` spans → matched in order
- Completely disjoint traces → all in OnlyInA/OnlyInB, no matches
- Baseline mode → mock TraceReader returning 20 similar traces, verify P50 synthesis
- Summary stats computed correctly (SlowerSpans, FasterSpans, CriticalPath)
- Missing trace → ErrNotFound

Tests use a mock `TraceReader` with canned traces — no testcontainers needed.

---

## 7. File Layout

```
pkg/tracecompare/
├── types.go          # TraceComparison, SpanMatch, SpanStats, ComparisonSummary
├── comparer.go       # Comparer struct, Compare(), CompareBaseline(), alignment logic
└── comparer_test.go  # unit tests with mock TraceReader
```

**Files modified:**
- `pkg/admin/api.go` — add `handleTraceCompare` handler, `SetTraceComparer` setter
- `cmd/gateway/main.go` — create Comparer, wire to admin API
