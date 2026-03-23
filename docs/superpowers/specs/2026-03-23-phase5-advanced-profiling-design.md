# Phase 5: Advanced Profiling — Design Specification

**Date:** 2026-03-23
**Status:** Approved
**Phase:** 5 of 6 (CloudMock Console)
**Depends on:** Phase 3 (Production Data Plane)

---

## Overview

Continuous profiling with function call stack capture at key execution points, CPU/heap/goroutine profiling via `runtime/pprof`, flame graph rendering in folded stack format, trace-to-profile linking for slow spans, and source map symbolication for Node/TS frames. Profiles stored as files in pprof format.

### Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scope | All 5 components including symbolication | Complete Phase 5 as specified |
| Storage | File-based (pprof format in temp dir) | Binary format, works with `go tool pprof`, survives restarts |
| Stack capture | Key points (handler entry, DB, external, error) | Shows actual execution path, matches spec Section 2.6 |
| Flame graph format | Both pprof + folded stacks via format param | Dashboard needs folded stacks, developers want raw pprof |

---

## 1. Stack Capture

```go
// pkg/profiling/types.go

type StackFrame struct {
    Function string `json:"function"`
    File     string `json:"file"`
    Line     int    `json:"line"`
    Module   string `json:"module,omitempty"`
}

type SpanStack struct {
    Point  string       `json:"point"`  // "handler_entry", "db_query", "external_call", "error"
    Frames []StackFrame `json:"frames"`
}
```

### stack.go

`CaptureStack(point string, skip int) SpanStack` — wraps `runtime.Callers()` + `runtime.CallersFrames()`. Skips runtime/reflect internal frames. Returns up to 32 frames.

Stored as JSON in `TraceContext.Metadata["stacks"]` — array of SpanStack objects. The logging middleware captures at handler entry. Other instrumentation code can call `CaptureStack()` at DB query, external call, and error points.

---

## 2. Profiling Engine

```go
// pkg/profiling/engine.go

type Engine struct {
    profileDir  string
    mu          sync.RWMutex
    profiles    []Profile
    maxProfiles int  // default 100, circular
}

type Profile struct {
    ID         string        `json:"id"`
    Service    string        `json:"service"`
    Type       string        `json:"type"`       // "cpu", "heap", "goroutine"
    FilePath   string        `json:"-"`
    CapturedAt time.Time     `json:"captured_at"`
    Duration   time.Duration `json:"duration,omitempty"` // CPU only
    Size       int64         `json:"size_bytes"`
}

func New(profileDir string, maxProfiles int) *Engine
func (e *Engine) Capture(service, profileType string, duration time.Duration) (*Profile, error)
func (e *Engine) Get(id string) (*Profile, error)
func (e *Engine) List(service string) ([]Profile, error)
func (e *Engine) FilePath(id string) (string, error)
func (e *Engine) FoldedStacks(id string) (string, error)
func (e *Engine) FindRelevant(service string, around time.Time) []Profile
```

### Capture behavior

- **CPU:** `pprof.StartCPUProfile(file)`, sleep duration, `pprof.StopCPUProfile()`. Captures the cloudmock gateway process.
- **Heap:** `pprof.WriteHeapProfile(file)`. Point-in-time snapshot.
- **Goroutine:** `pprof.Lookup("goroutine").WriteTo(file, 0)`. All goroutines.

Files stored at `{profileDir}/{id}.pprof`. Circular buffer: when maxProfiles exceeded, oldest profile's file is deleted.

### FoldedStacks

Parse pprof protobuf using `google.golang.org/protobuf` (pprof proto format). Walk samples, resolve function names from string table, emit Brendan Gregg folded format: `func1;func2;func3 count\n`.

### FindRelevant

Returns profiles captured within ±5 minutes of the given timestamp. Used for trace-to-profile linking.

---

## 3. Source Map Symbolication

```go
// pkg/profiling/sourcemap.go

type Symbolizer struct {
    mu   sync.RWMutex
    maps map[string]*sourceMap // key: file path
}

type sourceMap struct {
    mappings []mapping // sorted by generated line/column
}

type mapping struct {
    GeneratedLine   int
    GeneratedColumn int
    OriginalFile    string
    OriginalLine    int
    OriginalColumn  int
    OriginalName    string
}

func NewSymbolizer() *Symbolizer
func (s *Symbolizer) LoadMap(filePath string, mapData []byte) error
func (s *Symbolizer) Symbolicate(frames []StackFrame) []StackFrame
```

Parses Source Map v3 format (VLQ-encoded mappings). Best-effort: if no source map available, frames pass through unchanged. Source maps can be loaded via deploy metadata or a `POST /api/sourcemaps` endpoint.

---

## 4. Trace-to-Profile Linking

When displaying a slow span (via `/api/explain/{id}` or the service inspector), include links to relevant profiles:

- Query `engine.FindRelevant(span.Service, span.StartTime)`
- If profiles found, include them in the response as `related_profiles: [{id, type, captured_at}]`

This is wired into the existing explain endpoint by checking if the profiling engine is available.

---

## 5. API

```
POST /api/profile/{service}?type=cpu&duration=5s              — capture new profile
GET  /api/profile/{service}?type=heap                         — capture heap snapshot
GET  /api/profile/{service}?type=goroutine                    — capture goroutine dump
GET  /api/profiles?service=X                                  — list captured profiles
GET  /api/profiles/{id}?format=pprof                          — download raw pprof
GET  /api/profiles/{id}?format=flamegraph                     — get folded stacks
POST /api/sourcemaps                                          — upload source map
```

CPU profiles use POST because they take `duration` seconds to capture (blocking). Heap and goroutine are instant → GET.

---

## 6. File Layout

```
pkg/profiling/
├── types.go          # StackFrame, SpanStack, Profile
├── stack.go          # CaptureStack() via runtime.Callers
├── stack_test.go
├── engine.go         # Engine: Capture, Get, List, FindRelevant
├── engine_test.go
├── folded.go         # pprof → folded stacks converter
├── folded_test.go
├── sourcemap.go      # Symbolizer, source map v3 parser
└── sourcemap_test.go
```

**Files modified:**
- `pkg/gateway/logging.go` — capture stack at handler entry, store in trace metadata
- `pkg/admin/api.go` — add profiling + sourcemap API endpoints
- `cmd/gateway/main.go` — create profiling engine, wire to admin API
