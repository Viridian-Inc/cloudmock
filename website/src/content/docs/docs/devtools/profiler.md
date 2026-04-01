---
title: Profiler View
description: CPU, heap, and goroutine profiling for CloudMock services (coming soon)
---

The Profiler view will provide runtime profiling capabilities for CloudMock's Go-based service emulators. This view is currently in development and shows a "Coming soon" placeholder.

## Planned capabilities

### CPU profiling

Capture CPU profiles for any running CloudMock service. CPU profiles show where the emulator spends processing time, helping identify hot paths in request handling, IAM evaluation, or service routing.

Profiles will be capturable on demand with a configurable duration (e.g., 10s, 30s, 60s) and viewable as interactive flamegraphs directly in the devtools.

### Heap profiling

Take heap snapshots to analyze memory allocation patterns. Heap profiles will show which objects are consuming memory, helping identify leaks or excessive allocation in long-running CloudMock instances.

### Goroutine profiling

Inspect active goroutines to diagnose concurrency issues. The goroutine profile will show the stack trace of every running goroutine, making it easy to spot blocked operations, deadlocks, or goroutine leaks in the gateway or service handlers.

## Viewing profiles

The profiler will support two visualization modes:

- **Flamegraph** -- Interactive flamegraph rendered in the browser, with zoom, search, and click-to-focus.
- **pprof download** -- Download the raw profile in Go's pprof format for analysis with `go tool pprof` or other compatible tools.

## When to use

The Profiler is intended for diagnosing performance issues in CloudMock itself, not in your application code. Typical use cases:

- CloudMock is consuming unexpectedly high CPU when processing a specific service's requests.
- Memory usage grows over time during long test sessions.
- Request latency is higher than expected and you want to identify the bottleneck inside the emulator.

## Admin API endpoints (planned)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/profile/cpu` | Start a CPU profile capture |
| `POST` | `/api/profile/heap` | Take a heap snapshot |
| `GET` | `/api/profile/goroutines` | List active goroutines with stack traces |
| `GET` | `/api/profile/{id}` | Download a captured profile |
