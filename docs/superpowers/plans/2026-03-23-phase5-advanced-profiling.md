# Phase 5: Advanced Profiling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add function call stack capture, CPU/heap/goroutine profiling, flame graph rendering, trace-to-profile linking, and source map symbolication.

**Architecture:** A `profiling.Engine` captures and stores pprof profiles as files, converts to folded stacks for flame graphs. `CaptureStack()` captures frames at key execution points via `runtime.Callers()`. `Symbolizer` resolves Node/TS frames via source maps. All exposed via REST API.

**Tech Stack:** Go 1.26, `runtime/pprof`, `runtime.Callers`, pprof protobuf format, Source Map v3

---

## File Structure

```
pkg/profiling/
├── types.go          # StackFrame, SpanStack, Profile
├── stack.go          # CaptureStack()
├── stack_test.go
├── engine.go         # Engine: Capture, Get, List, FoldedStacks, FindRelevant
├── engine_test.go
├── folded.go         # pprof → folded stacks converter
├── folded_test.go
├── sourcemap.go      # Symbolizer
└── sourcemap_test.go
```

**Files modified:**
- `pkg/gateway/logging.go` — capture stack at handler entry
- `pkg/admin/api.go` — profiling API endpoints
- `cmd/gateway/main.go` — create engine, wire

---

## Task 1: Stack Capture

**Files:**
- Create: `pkg/profiling/types.go`
- Create: `pkg/profiling/stack.go`
- Create: `pkg/profiling/stack_test.go`

- [ ] **Step 1:** `mkdir -p pkg/profiling`

- [ ] **Step 2: Write types.go**

`StackFrame` (Function, File, Line, Module), `SpanStack` (Point, Frames), `Profile` (ID, Service, Type, FilePath, CapturedAt, Duration, Size).

- [ ] **Step 3: Write stack tests**

```go
func TestCaptureStack(t *testing.T) {
    stack := CaptureStack("handler_entry", 0)
    if stack.Point != "handler_entry" { t.Error(...) }
    if len(stack.Frames) == 0 { t.Error("expected frames") }
    // First frame should be this test function or CaptureStack
    // Verify no runtime.goexit or runtime.main frames
}

func TestCaptureStack_SkipsRuntime(t *testing.T) {
    stack := CaptureStack("test", 0)
    for _, f := range stack.Frames {
        if strings.HasPrefix(f.Function, "runtime.") { t.Error("should skip runtime frames") }
    }
}

func TestCaptureStack_MaxFrames(t *testing.T) {
    stack := CaptureStack("test", 0)
    if len(stack.Frames) > 32 { t.Error("should cap at 32 frames") }
}
```

- [ ] **Step 4: Write stack.go**

```go
func CaptureStack(point string, skip int) SpanStack {
    var pcs [32]uintptr
    n := runtime.Callers(skip+2, pcs[:]) // +2 to skip Callers + CaptureStack
    frames := runtime.CallersFrames(pcs[:n])
    var result []StackFrame
    for {
        frame, more := frames.Next()
        // Skip runtime, reflect, testing internals
        if strings.HasPrefix(frame.Function, "runtime.") ||
           strings.HasPrefix(frame.Function, "reflect.") ||
           strings.HasPrefix(frame.Function, "testing.") {
            if !more { break }
            continue
        }
        result = append(result, StackFrame{
            Function: frame.Function,
            File:     frame.File,
            Line:     frame.Line,
        })
        if !more { break }
    }
    return SpanStack{Point: point, Frames: result}
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./pkg/profiling/ -v -run TestCaptureStack`

- [ ] **Step 6: Commit**

```bash
git add pkg/profiling/
git commit -m "feat(profiling): add stack frame capture via runtime.Callers

CaptureStack() captures up to 32 frames, skipping runtime/reflect
internals. SpanStack stored in trace metadata."
```

---

## Task 2: Profiling Engine

**Files:**
- Create: `pkg/profiling/engine.go`
- Create: `pkg/profiling/engine_test.go`

- [ ] **Step 1: Write engine tests**

```go
func TestCapture_Heap(t *testing.T) {
    dir := t.TempDir()
    e := New(dir, 10)
    p, err := e.Capture("gateway", "heap", 0)
    // Verify profile created, file exists, non-zero size
}

func TestCapture_Goroutine(t *testing.T) {
    dir := t.TempDir()
    e := New(dir, 10)
    p, err := e.Capture("gateway", "goroutine", 0)
    // Verify profile created
}

func TestCapture_CPU(t *testing.T) {
    dir := t.TempDir()
    e := New(dir, 10)
    p, err := e.Capture("gateway", "cpu", 100*time.Millisecond)
    // Short duration for test. Verify file created.
}

func TestList(t *testing.T) {
    // Capture 3 profiles for different services
    // List("gateway") → only gateway profiles
    // List("") → all
}

func TestCircularBuffer(t *testing.T) {
    dir := t.TempDir()
    e := New(dir, 3)
    // Capture 5 profiles → only 3 remain, oldest files deleted
}

func TestFindRelevant(t *testing.T) {
    // Capture profile, then FindRelevant with timestamp ±5m → found
    // FindRelevant with timestamp 1h away → not found
}
```

- [ ] **Step 2: Write engine.go**

Engine with `profileDir`, circular `[]Profile` buffer, mutex.

`Capture`: generate UUID-based ID, create file at `{dir}/{id}.pprof`, write profile using `runtime/pprof`. For CPU: `StartCPUProfile`, sleep duration, `StopCPUProfile`. For heap: `WriteHeapProfile`. For goroutine: `Lookup("goroutine").WriteTo`.

`Get`: find by ID. `List`: filter by service. `FilePath`: return file path for download. `FindRelevant`: return profiles within ±5min of timestamp.

Circular buffer: when len(profiles) > maxProfiles, remove oldest, delete its file.

- [ ] **Step 3: Run tests**

Run: `go test ./pkg/profiling/ -v -run TestCapture -cover`

- [ ] **Step 4: Commit**

```bash
git add pkg/profiling/engine.go pkg/profiling/engine_test.go
git commit -m "feat(profiling): add profiling engine with CPU/heap/goroutine capture

File-based pprof storage with circular buffer. Supports capture,
list, get, and find-relevant-for-trace-linking."
```

---

## Task 3: Folded Stacks Converter

**Files:**
- Create: `pkg/profiling/folded.go`
- Create: `pkg/profiling/folded_test.go`

- [ ] **Step 1: Write folded tests**

```go
func TestFoldedStacks_FromHeapProfile(t *testing.T) {
    // Capture a real heap profile
    // Convert to folded stacks
    // Verify output format: "func1;func2;func3 count\n"
    // Verify at least some lines present
}

func TestFoldedStacks_InvalidFile(t *testing.T) {
    // Non-pprof file → error
}
```

- [ ] **Step 2: Write folded.go**

Parse pprof protobuf format. The pprof format stores:
- `string_table`: array of strings
- `sample`: array of {location_id[], value[]}
- `location`: array of {id, line[{function_id, line}]}
- `function`: array of {id, name (index into string_table), filename}

Walk samples, resolve each location → function name chain, emit folded format.

Use `google.golang.org/protobuf` to parse. The pprof proto is defined at `google/pprof/profile.proto`. You'll need to either use the `github.com/google/pprof/profile` package (which has `Parse()`) or parse manually.

**Recommended:** Use `github.com/google/pprof/profile` package which provides `profile.Parse(reader)` and gives you `Profile.Sample`, `Profile.Location`, `Profile.Function` directly.

`go get github.com/google/pprof`

```go
func FoldedStacks(r io.Reader) (string, error) {
    p, err := profile.Parse(r)
    // For each sample: resolve location → function names, join with ";", append count
}
```

Wire into Engine: `engine.FoldedStacks(id)` opens the pprof file and calls this.

- [ ] **Step 3: Run tests**

Run: `go test ./pkg/profiling/ -v -run TestFolded`

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum pkg/profiling/folded.go pkg/profiling/folded_test.go
git commit -m "feat(profiling): add pprof to folded stacks converter for flame graphs

Parses pprof protobuf, resolves function names, emits Brendan Gregg
folded stack format for d3-flame-graph rendering."
```

---

## Task 4: Source Map Symbolication

**Files:**
- Create: `pkg/profiling/sourcemap.go`
- Create: `pkg/profiling/sourcemap_test.go`

- [ ] **Step 1: Write sourcemap tests**

```go
func TestSymbolicate_WithMap(t *testing.T) {
    s := NewSymbolizer()
    // Load a simple source map (hand-crafted or from test fixture)
    s.LoadMap("bundle.js", []byte(`{"version":3,"sources":["src/app.ts"],...}`))
    frames := []StackFrame{{Function: "anonymous", File: "bundle.js", Line: 1, Module: ""}}
    result := s.Symbolicate(frames)
    // Verify original file/line resolved
}

func TestSymbolicate_NoMap(t *testing.T) {
    s := NewSymbolizer()
    frames := []StackFrame{{Function: "foo", File: "app.go", Line: 42}}
    result := s.Symbolicate(frames)
    // Frames pass through unchanged
}
```

- [ ] **Step 2: Write sourcemap.go**

Implement Source Map v3 parser. The format has:
- `version`: 3
- `sources`: array of original file paths
- `names`: array of original identifiers
- `mappings`: VLQ-encoded string mapping generated → original positions

Parse VLQ-encoded mappings into a sorted list. `Symbolicate` looks up each frame's `File:Line` in the mappings, resolves to original file/line/name.

**Alternative:** Use `github.com/nicholasgasior/gol` or similar Go source map library if available. If not, implement minimal VLQ decoder (it's ~50 lines).

- [ ] **Step 3: Run tests**

Run: `go test ./pkg/profiling/ -v -run TestSymbolicate`

- [ ] **Step 4: Commit**

```bash
git add pkg/profiling/sourcemap.go pkg/profiling/sourcemap_test.go
git commit -m "feat(profiling): add source map symbolication for Node/TS frames

Parses Source Map v3, resolves generated file:line to original.
Best-effort: frames without maps pass through unchanged."
```

---

## Task 5: API & Wiring

**Files:**
- Modify: `pkg/gateway/logging.go` — add stack capture
- Modify: `pkg/admin/api.go` — add profiling endpoints
- Modify: `cmd/gateway/main.go` — create engine, wire

- [ ] **Step 1: Add stack capture to logging middleware**

In `pkg/gateway/logging.go`, in `LoggingMiddlewareWithOpts`, after capturing the request but before responding, add:

```go
stacks := []profiling.SpanStack{profiling.CaptureStack("handler_entry", 1)}
stackJSON, _ := json.Marshal(stacks)
metadata["stacks"] = string(stackJSON)
```

This adds a "stacks" key to the trace metadata with the handler entry stack.

- [ ] **Step 2: Add profiling API endpoints**

In `pkg/admin/api.go`:
- Add `profilingEngine *profiling.Engine` field + `SetProfilingEngine` setter
- Add `symbolizer *profiling.Symbolizer` field + `SetSymbolizer` setter

Endpoints:
- `POST /api/profile/{service}?type=cpu&duration=5s` — capture CPU profile
- `GET /api/profile/{service}?type=heap` — capture heap
- `GET /api/profile/{service}?type=goroutine` — capture goroutine
- `GET /api/profiles?service=X` — list profiles
- `GET /api/profiles/{id}?format=pprof` — download raw pprof
- `GET /api/profiles/{id}?format=flamegraph` — get folded stacks
- `POST /api/sourcemaps` — upload source map (multipart form: file + path)

Register routes: `/api/profile/`, `/api/profiles`, `/api/profiles/`, `/api/sourcemaps`

- [ ] **Step 3: Wire in main.go**

```go
profileDir := filepath.Join(os.TempDir(), "cloudmock-profiles")
os.MkdirAll(profileDir, 0755)
profEngine := profiling.New(profileDir, 100)
symbolizer := profiling.NewSymbolizer()
adminAPI.SetProfilingEngine(profEngine)
adminAPI.SetSymbolizer(symbolizer)
```

- [ ] **Step 4: Run all tests**

Run: `go test -short ./pkg/profiling/ ./pkg/admin/ ./pkg/gateway/ -v`

- [ ] **Step 5: Commit**

```bash
git add pkg/gateway/logging.go pkg/admin/api.go cmd/gateway/main.go
git commit -m "feat(profiling): wire profiling engine with API endpoints

Stack capture in middleware, profile capture/list/download endpoints,
source map upload, flame graph folded stacks format."
```

---

## Task Summary

| Task | What it builds | Depends on |
|------|---------------|------------|
| 1 | Stack capture (types + runtime.Callers) | — |
| 2 | Profiling engine (capture, store, list) | 1 |
| 3 | Folded stacks converter (pprof → flame graph) | 2 |
| 4 | Source map symbolication | 1 |
| 5 | API endpoints + middleware + wiring | 1-4 |
