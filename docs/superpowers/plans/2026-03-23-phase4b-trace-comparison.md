# Phase 4b: Trace Comparison Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add side-by-side trace comparison with span alignment, per-span deltas, and route baseline synthesis — all storage-agnostic via the TraceReader interface.

**Architecture:** A stateless `Comparer` in `pkg/tracecompare/` takes a `TraceReader`, fetches traces, aligns spans by `(service, action)`, computes latency deltas and metadata diffs. Baseline mode synthesizes a virtual trace from P50 latencies of recent similar traces. One API endpoint with a `baseline` flag.

**Tech Stack:** Go 1.26, existing `dataplane.TraceReader` interface

---

## File Structure

```
pkg/tracecompare/
├── types.go          # TraceComparison, SpanMatch, SpanStats, ComparisonSummary
├── comparer.go       # Comparer struct, Compare(), CompareBaseline(), alignment logic
└── comparer_test.go  # unit tests with mock TraceReader
```

**Files modified:**
- `pkg/admin/api.go` — add `handleTraceCompare` handler, `SetTraceComparer` setter
- `cmd/gateway/main.go` — create Comparer, wire to admin API

---

## Task 1: Types & Comparer

**Files:**
- Create: `pkg/tracecompare/types.go`
- Create: `pkg/tracecompare/comparer.go`
- Create: `pkg/tracecompare/comparer_test.go`

- [ ] **Step 1: Create package directory**

Run: `mkdir -p pkg/tracecompare`

- [ ] **Step 2: Write types.go**

```go
package tracecompare

type TraceComparison struct {
    TraceA   string            `json:"trace_a"`
    TraceB   string            `json:"trace_b"`
    Matches  []SpanMatch       `json:"matches"`
    OnlyInA  []SpanSummary     `json:"only_in_a"`
    OnlyInB  []SpanSummary     `json:"only_in_b"`
    Summary  ComparisonSummary `json:"summary"`
}

type SpanMatch struct {
    Service      string              `json:"service"`
    Action       string              `json:"action"`
    A            SpanStats           `json:"a"`
    B            SpanStats           `json:"b"`
    LatencyDelta float64             `json:"latency_delta_ms"`
    LatencyPct   float64             `json:"latency_pct"`
    StatusChange bool                `json:"status_change"`
    MetadataDiff map[string][2]string `json:"metadata_diff,omitempty"`
}

type SpanStats struct {
    DurationMs float64           `json:"duration_ms"`
    StatusCode int               `json:"status_code"`
    Error      string            `json:"error,omitempty"`
    Metadata   map[string]string `json:"metadata,omitempty"`
}

type SpanSummary struct {
    Service    string  `json:"service"`
    Action     string  `json:"action"`
    DurationMs float64 `json:"duration_ms"`
    StatusCode int     `json:"status_code"`
}

type ComparisonSummary struct {
    TotalLatencyA float64 `json:"total_latency_a_ms"`
    TotalLatencyB float64 `json:"total_latency_b_ms"`
    LatencyDelta  float64 `json:"latency_delta_ms"`
    SlowerSpans   int     `json:"slower_spans"`
    FasterSpans   int     `json:"faster_spans"`
    AddedSpans    int     `json:"added_spans"`
    RemovedSpans  int     `json:"removed_spans"`
    CriticalPath  string  `json:"critical_path,omitempty"`
}
```

- [ ] **Step 3: Write comparer tests**

Create `pkg/tracecompare/comparer_test.go` with a mock `TraceReader` and these test cases:

```go
// Mock TraceReader that returns canned traces
type mockTraceReader struct {
    traces map[string]*dataplane.TraceContext
    recent []dataplane.TraceSummary
}

func TestCompare_IdenticalTraces(t *testing.T) {
    // Two identical traces → all matches, zero deltas, no added/removed
}

func TestCompare_DifferentLatencies(t *testing.T) {
    // Same structure but trace B has higher latencies
    // Verify LatencyDelta, LatencyPct, SlowerSpans count
}

func TestCompare_AddedAndRemovedSpans(t *testing.T) {
    // Trace A has spans [bff, dynamodb], trace B has [bff, lambda]
    // dynamodb → OnlyInA, lambda → OnlyInB, bff → Matches
}

func TestCompare_DuplicateSpans(t *testing.T) {
    // Trace A has two dynamodb:Query spans (at different times)
    // Trace B has two dynamodb:Query spans
    // Matched in order of appearance
}

func TestCompare_MetadataDiff(t *testing.T) {
    // Same spans but different metadata values
    // Verify MetadataDiff captures the differences
}

func TestCompare_StatusChange(t *testing.T) {
    // Span in A has 200, same span in B has 500
    // Verify StatusChange=true
}

func TestCompare_TraceNotFound(t *testing.T) {
    // Request nonexistent trace → dataplane.ErrNotFound
}

func TestCompare_Summary(t *testing.T) {
    // Verify CriticalPath is the service with largest absolute slowdown
}

func TestCompareBaseline(t *testing.T) {
    // Mock TraceReader.Search returns 5 similar traces
    // Mock TraceReader.Get returns each with known latencies
    // Verify baseline is synthesized as P50 per (service, action)
    // Verify comparison against baseline produces correct deltas
}

func TestCompareBaseline_NoSimilarTraces(t *testing.T) {
    // Search returns empty → meaningful error
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./pkg/tracecompare/ -v`
Expected: FAIL — comparer.go doesn't exist.

- [ ] **Step 5: Write comparer.go**

```go
package tracecompare

import (
    "context"
    "fmt"
    "sort"

    "github.com/neureaux/cloudmock/pkg/dataplane"
)

type Comparer struct {
    traces dataplane.TraceReader
}

func New(traces dataplane.TraceReader) *Comparer {
    return &Comparer{traces: traces}
}
```

Key internal functions:

`flattenTrace(tc *dataplane.TraceContext) []flatSpan` — recursively walks the trace tree, collecting all spans into a flat list sorted by start time. Each `flatSpan` has Service, Action, DurationMs, StatusCode, Error, Metadata.

`alignSpans(a, b []flatSpan) (matches []SpanMatch, onlyA, onlyB []SpanSummary)` — matches spans by `(service, action)` key. Uses a map of `key → []flatSpan` for each side. Pops from the list in order for duplicate handling. Computes deltas for each match.

`Compare(ctx, traceAID, traceBID)`:
1. Fetch both traces via `TraceReader.Get()`
2. Flatten both
3. Align spans
4. Compute summary (total latency from root span duration, count slower/faster, find critical path)
5. Return `TraceComparison`

`CompareBaseline(ctx, traceID)`:
1. Fetch the target trace
2. Get root service, action, method from the trace
3. `TraceReader.Search(filter{Service: rootService, Limit: 20})` to find similar traces
4. Fetch each, flatten, collect all spans grouped by `(service, action)`
5. For each group, compute P50 latency (sort, pick median) and most common status code
6. Build a synthetic `[]flatSpan` from the P50 values
7. Run the same `alignSpans` logic comparing target vs baseline
8. Return with `TraceA: "baseline"` label

Handle edge cases:
- Division by zero in LatencyPct: if A.DurationMs == 0, set LatencyPct to 0
- No similar traces for baseline: return error `fmt.Errorf("no recent traces found for route %s/%s", service, action)`
- Empty trace (no spans): return comparison with empty matches

- [ ] **Step 6: Run tests**

Run: `go test ./pkg/tracecompare/ -v -cover`
Expected: All PASS, >80% coverage.

- [ ] **Step 7: Commit**

```bash
git add pkg/tracecompare/
git commit -m "feat(tracecompare): add trace comparison with span alignment and baseline

Compares two traces by aligning spans on (service, action), computing
per-span latency deltas and metadata diffs. Baseline mode synthesizes
P50 from recent similar traces via TraceReader."
```

---

## Task 2: API Endpoint

**Files:**
- Modify: `pkg/admin/api.go` — add handler and setter

- [ ] **Step 1: Add traceComparer field and setter**

In `pkg/admin/api.go`, add to API struct:
```go
traceComparer *tracecompare.Comparer
```

Add setter:
```go
func (a *API) SetTraceComparer(tc *tracecompare.Comparer) {
    a.traceComparer = tc
}
```

Register route in `NewWithDataPlane()`:
```go
a.mux.HandleFunc("/api/traces/compare", a.handleTraceCompare)
```

- [ ] **Step 2: Write handleTraceCompare handler**

```go
func (a *API) handleTraceCompare(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if a.traceComparer == nil {
        http.Error(w, "trace comparison not available", http.StatusServiceUnavailable)
        return
    }

    traceA := r.URL.Query().Get("a")
    if traceA == "" {
        http.Error(w, "missing required parameter: a", http.StatusBadRequest)
        return
    }

    baseline := r.URL.Query().Get("baseline") == "true"
    traceB := r.URL.Query().Get("b")

    if !baseline && traceB == "" {
        http.Error(w, "must provide parameter b or baseline=true", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    var result *tracecompare.TraceComparison
    var err error

    if baseline {
        result, err = a.traceComparer.CompareBaseline(ctx, traceA)
    } else {
        result, err = a.traceComparer.Compare(ctx, traceA, traceB)
    }

    if errors.Is(err, dataplane.ErrNotFound) {
        http.Error(w, "trace not found", http.StatusNotFound)
        return
    }
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 3: Run existing tests**

Run: `go test ./pkg/admin/ -v -short`
Expected: All existing tests PASS. New handler tested via Task 1's unit tests on the Comparer.

- [ ] **Step 4: Commit**

```bash
git add pkg/admin/api.go
git commit -m "feat(tracecompare): add /api/traces/compare endpoint

GET with a={id}&b={id} for trace-vs-trace, a={id}&baseline=true
for trace-vs-baseline. Returns structured comparison with span
alignment and deltas."
```

---

## Task 3: Gateway Wiring

**Files:**
- Modify: `cmd/gateway/main.go` — create Comparer, wire to admin API

- [ ] **Step 1: Wire Comparer in main.go**

After DataPlane construction and admin API creation, add:

```go
// Trace comparison
tc := tracecompare.New(dp.Traces)
adminAPI.SetTraceComparer(tc)
```

Add import: `"github.com/neureaux/cloudmock/pkg/tracecompare"`

- [ ] **Step 2: Verify full build**

Run: `go build ./...`
Expected: Clean compile.

- [ ] **Step 3: Run all tests**

Run: `go test -short ./pkg/tracecompare/ ./pkg/admin/ ./pkg/gateway/ -v`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/gateway/main.go
git commit -m "feat(tracecompare): wire Comparer into gateway

Creates Comparer with DataPlane TraceReader, passes to admin API.
Works in both local and production modes."
```

---

## Task Summary

| Task | What it builds | Depends on |
|------|---------------|------------|
| 1 | Types + Comparer + tests (core logic) | — |
| 2 | API endpoint in admin API | 1 |
| 3 | Gateway wiring in main.go | 1, 2 |
