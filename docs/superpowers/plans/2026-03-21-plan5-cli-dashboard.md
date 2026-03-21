# CLI + Dashboard Implementation Plan

> **For agentic workers:** This is a retrospective document. All tasks listed below were already executed. Checkboxes are marked completed.

**Goal:** Build the operator-facing control plane: a request logging middleware baked into the gateway, a JSON admin API for introspection and control, a self-contained web dashboard served as a single-page application, and a `cloudmock` CLI binary for managing the gateway from the terminal.

**Status:** COMPLETED

---

## Overview

Plan 5 added the full operator interface to cloudmock. The gateway now logs every request to a circular in-memory buffer and tracks per-service request counts. The admin API (default port 4599) exposes this data over HTTP/JSON. A web dashboard (default port 4598) renders it as a live-refreshing SPA. The `cloudmock` CLI binary wraps the admin API for terminal use.

All three components (logging middleware, admin API, dashboard) are built in pure Go with no external dependencies.

---

## Chunk 1: Request Logging Middleware

### Task 1: Request Log and Stats (`pkg/gateway/logging.go`)

- [x] Define `RequestEntry` struct capturing: `Timestamp`, `Service`, `Action`, `Method`, `Path`, `StatusCode`, `Latency`, `CallerID`, `Error`
- [x] Implement `RequestLog` — thread-safe circular buffer backed by a fixed-size slice
  - [x] `NewRequestLog(capacity)` — capacity defaults to 1000
  - [x] `Add(entry)` — writes at `pos`, advances pos with wraparound
  - [x] `Recent(service, limit)` — returns entries newest-first, optionally filtered by service name
- [x] Implement `RequestStats` — per-service atomic counters
  - [x] `NewRequestStats()` — empty counter map
  - [x] `Increment(svcName)` — double-checked locking to avoid write lock on hot path
  - [x] `Snapshot()` — returns `map[string]int64` copy under read lock
- [x] Implement `responseRecorder` — wraps `http.ResponseWriter` to capture status code
- [x] Implement `LoggingMiddleware(next, log, stats)` — wraps any handler, records timing and entry after response
- [x] Implement `detectServiceFromRequest(r)` — extracts service name from `Authorization` credential scope or `X-Amz-Target` header (without importing `routing` to avoid circular imports)
- [x] Implement `detectActionFromRequest(r)` — extracts action from `X-Amz-Target` or `?Action` query param
- [x] Implement `extractCallerID(r)` — extracts access key ID from `Authorization` header
- [x] Wire `LoggingMiddleware` around the gateway in `cmd/gateway/main.go`

**Files created/modified:**
- `pkg/gateway/logging.go`
- `cmd/gateway/main.go`

---

## Chunk 2: Admin API

### Task 2: Admin HTTP Handler (`pkg/admin/api.go`)

- [x] Define `Resettable` interface — optional interface services can implement to support `Reset()`
- [x] Define `ServiceInfo` struct: `Name`, `ActionCount`, `Healthy`
- [x] Define `HealthResponse` struct: `Status`, `Services`
- [x] Implement `API` struct holding `cfg`, `registry`, `log`, `stats`, and an internal `*http.ServeMux`
- [x] Implement `New(cfg, registry, log, stats)` — registers all routes on the mux
- [x] Implement route handlers:
  - [x] `GET /api/services` — list all registered services with health and action count
  - [x] `GET /api/services/{name}` — get a single service by name
  - [x] `POST /api/services/{name}/reset` — reset a named service (calls `Resettable.Reset()` if implemented)
  - [x] `POST /api/reset` — reset all resettable services
  - [x] `GET /api/health` — aggregate health: `"healthy"` if all services pass, `"degraded"` otherwise
  - [x] `GET /api/config` — return the running `*config.Config` as JSON
  - [x] `GET /api/stats` — return per-service request count snapshot
  - [x] `GET /api/requests` — return recent request log entries, supports `?service=` and `?limit=` query params
- [x] Implement `writeJSON(w, status, v)` helper
- [x] Wire `admin.New(...)` into `cmd/gateway/main.go`, serve on admin port in a goroutine

**Files created/modified:**
- `pkg/admin/api.go`
- `cmd/gateway/main.go`

---

## Chunk 3: Web Dashboard

### Task 3: Dashboard Handler (`pkg/dashboard/dashboard.go`)

- [x] Implement `Handler` struct holding pre-rendered HTML bytes
- [x] Implement `New(adminPort)` — builds the HTML template with the admin base URL substituted
- [x] Implement `ServeHTTP` — serves the single HTML file for all requests
- [x] Build self-contained SPA as a Go string constant (`htmlTemplate`)

**Dashboard features:**
- [x] Sticky header with `cloudmock` brand and live health badge (green/orange/red dot)
- [x] **Services table** — lists all registered services with health status and request count badge, auto-populated from `/api/services` + `/api/stats`
- [x] **Request log table** — shows last 50 requests with time, service, action, status code (color-coded), and latency
- [x] **Service filter dropdown** — filters the request log to a single service
- [x] Auto-refresh every 5 seconds via `setInterval`
- [x] No external dependencies — all CSS and JS inlined in the HTML template
- [x] Wire dashboard into `cmd/gateway/main.go` — enabled when `cfg.Dashboard.Enabled` is true, serves on `cfg.Dashboard.Port`

**Files created/modified:**
- `pkg/dashboard/dashboard.go`
- `cmd/gateway/main.go`

---

## Chunk 4: CLI Binary

### Task 4: `cloudmock` CLI (`cmd/cloudmock/main.go`)

The `cloudmock` binary is the control-plane client. It communicates with the running gateway via the admin API. Default admin address is `http://localhost:4599`, overridable via `CLOUDMOCK_ADMIN_ADDR`.

- [x] Implement `main()` with command dispatch
- [x] Implement `cloudmock start` — locates the `gateway` binary, sets env vars for profile/services overrides, exec's it
  - [x] Flags: `-config` (default `cloudmock.yml`), `-profile` (minimal/standard/full), `-services` (comma-separated)
  - [x] `findGatewayBinary()` — searches `./bin/gateway`, `bin/gateway`, then `PATH`
- [x] Implement `cloudmock stop` — prints instructions (Ctrl+C or kill)
- [x] Implement `cloudmock status` — calls `GET /api/health`, renders a tabwriter table of service health
- [x] Implement `cloudmock reset [--service name]` — calls `POST /api/reset` or `POST /api/services/{name}/reset`
- [x] Implement `cloudmock services` — calls `GET /api/services`, renders name/action count/health table
- [x] Implement `cloudmock version` — prints `cloudmock version 0.1.0`
- [x] Implement `cloudmock config` — calls `GET /api/config`, pretty-prints JSON
- [x] Implement `cloudmock help` — prints usage

**Files created/modified:**
- `cmd/cloudmock/main.go`

---

## Verification

- [x] `LoggingMiddleware` records entries for every request including service, action, status, and latency
- [x] `RequestStats.Increment` is safe under concurrent load
- [x] Admin API returns correct JSON for all endpoints
- [x] `/api/health` returns `"degraded"` if any service is unhealthy
- [x] Dashboard HTML renders without JavaScript errors
- [x] Dashboard auto-refresh updates the services table and request log
- [x] `cloudmock status` connects to running gateway and displays service health
- [x] `cloudmock reset` resets all resettable services
- [x] `cloudmock services` lists all 98 registered services
- [x] Admin API tests pass (`pkg/admin/api_test.go`)
- [x] Dashboard tests pass (`pkg/dashboard/dashboard_test.go`)
- [x] Gateway logging tests pass (`pkg/gateway/logging_test.go`)
