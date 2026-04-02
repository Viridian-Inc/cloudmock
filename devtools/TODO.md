# neureaux-devtools — Comprehensive TODO

> 31 commits · 12 views · 62 frontend files · 9 Rust modules · 3 SDKs

---

## Phase A: Live Service Map

### Node Health & Visualization
- [x] Health state computation (green/yellow/red from error rate + p99 vs SLO)
- [x] Node health coloring on topology canvas
- [x] User-facing impact detection (BFS from Client nodes through sync edges)
- [x] Pulsing red border + banner for user-facing degradation
- [x] Incident badges (red count circles) on affected nodes
- [x] Deploy markers (blue dots) on recently-deployed nodes
- [x] Blast radius dimming (select node → downstream highlighted, others dim)
- [x] Blast radius click-through (dimmed nodes remain clickable)
- [x] Lambda functions surfaced as individual microservice nodes

### Edge Metrics
- [x] Edge labels: callCount req/s + avgLatencyMs
- [x] Edge enrichment from trace data (fallback when request log empty)
- [x] Traffic-discovered edges (auto-create nodes + edges for services with trace traffic)
- [x] Edge labels: show on hover only (reduce visual noise)
- [x] Animated packets along edges (SVG animateMotion, glow filter, traffic-proportional count)

### Graph Layout & Interaction
- [x] ELK.js layered DAG layout
- [x] ELK tuning: wider spacing, 200px nodes, model order
- [x] GitHub Actions-style pill nodes (HTML over SVG edges)
- [x] Collapse/Expand toggle (34 nodes ↔ ~20 category nodes)
- [x] Node drag-to-reposition with pinning (double-click to unpin)
- [x] Domain group bounding boxes (subtle fill + border + label)
- [x] Minimap (150x100px, health-colored dots, viewport rectangle, click to pan)
- [x] Zoom (scroll) + pan (drag)
- [x] Saved layouts — persist node positions + zoom/pan per topology config
  - [x] Save current layout to localStorage with a name
  - [x] "Layouts" dropdown in toolbar to switch between saved layouts
  - [x] Auto-save "last used" layout on navigation away
  - [x] Import/export layout as JSON
  - [x] Reset to auto-layout (ELK) option

### Data Polling
- [x] Metrics polling (10s metrics from traces, 30s deploys/incidents)
- [x] Metrics computed from /api/traces when /api/metrics returns empty
- [x] Deploy normalization (PascalCase → camelCase from cloudmock API)
- [x] Topology data pulled entirely from cloudmock API (zero hardcoded data)
- [x] Topology collapse: external/plugin nodes kept, AWS resources collapsed by service

### Toolbar
- [x] Live indicator (pulsing green dot)
- [x] Incident count badge
- [x] Last deploy indicator
- [x] "📋 Services" toggle for service browser sidebar
- [x] "📦 Collapse / 🔍 Expand" toggle
- [x] "🗺 Minimap" toggle
- [x] "Unpin all (N)" button when nodes are pinned

---

## Phase B: Inspector Tabs

### Metrics Tab
- [x] Sparkline charts (req/s, p99 latency, error rate) via SVG polyline
- [x] Sparkline component with filled area, label, current value
- [x] Metrics history ring buffer (20 data points per service)
- [x] SLO status row (OK / Warning / Breach)
- [x] Routing toggle (Local / Cloud) in metrics tab

### Endpoints Tab
- [x] Routes from service-manifest.json grouped by resource prefix
- [x] Method badges (GET=green, POST=blue, PUT=yellow, DELETE=red)
- [x] 33 services, 880+ routes loaded from manifest
- [x] OpenAPI spec file scanning
  - [x] Configurable scan paths (e.g., `services/*/openapi.yaml`)
  - [x] Parse specs and map to topology nodes by service name
  - [x] Show parameter schemas, response types, descriptions
  - [ ] Version tracking (diff spec versions on deploy)

### Deploys Tab
- [x] Recent deploys with relative timestamp, commit, author, branch
- [x] PascalCase normalization for cloudmock deploy API response
- [x] Click deploy → opens deploy detail overlay

### Incidents Tab
- [x] Severity badges (critical/high/medium/low)
- [x] Status, first/last seen, alert count
- [x] Filtered by affected_services matching selected node

### Connections Tab
- [x] Inbound/outbound edges with callCount and latency
- [x] Blast radius count

### Deployment Drill-Down (overlay)
- [x] Service name, commit, author, branch, timestamp
- [x] Status badge (success/rolling/failed)
- [x] Container/pod section (simulated from deploy data)
- [x] Before/after deploy metrics comparison (error rate, p99, volume)
- [x] Real container/pod data from K8s plugin
  - [x] Fetch pods from cloudmock K8s API (`/api/v1/namespaces/*/pods`)
  - [x] Show pod status, restart count, readiness, age
  - [x] Fall back to simulated data with "Simulated" badge when no K8s pods running
  - [ ] Container logs (tail last N lines)
- [x] Real ECS task data
  - [x] Fetch tasks/services from cloudmock ECS (`/api/resources/ecs`)
  - [x] Task status, age, readiness
  - [ ] Task definition version
- [x] Rollback button (trigger redeploy to previous version)

---

## Phase C: Timeline + Time Travel

### Timeline Bar
- [x] Horizontal timeline at bottom of topology view
- [x] Deploy events as blue dots, incident events as red dots
- [x] Proportional positioning along time range
- [x] Hover tooltips with event details
- [x] Click event → selects node / opens deploy detail

### Time Range Selector
- [x] Presets: Live, 15m, 1h, 6h, 24h
- [x] Custom date range picker (datetime-local inputs)
- [x] Live mode: real-time polling with pulsing green dot
- [x] Historical mode: stops polling, filters traces by time window
- [x] "Paused — viewing Xm ago" indicator with "Now" snap-back button

### Playhead Scrubbing
- [x] Draggable vertical line on timeline
- [x] Map updates health states to playhead timestamp
- [x] Playhead time label
- [x] Auto-switches to historical mode when dragged in live mode

### Persistence
- [x] SQLite database (rusqlite) at ~/.config/neureaux-devtools/history.db
- [x] Tables: metrics_snapshots, deploy_events, incident_history
- [x] Tauri commands: save_metrics_snapshot, query_metrics_history
- [x] Auto-cleanup (7-day metrics, 30-day events)
- [x] Auto-persist metrics from polling loop
  - [x] Frontend calls `save_metrics_snapshot` Tauri command every 5 minutes
  - [x] Batch current ServiceMetrics into snapshot records
- [x] Query from SQLite when time range exceeds cloudmock memory window
  - [x] Detect when requested time range is older than cloudmock's trace buffer
  - [x] Fall back to SQLite query for historical data
  - [x] Merge SQLite + live data for overlapping windows

---

## Phase D: Service Browser + Routing

### Service Browser Sidebar
- [x] Services grouped by domain from service-domains.json
- [x] Health dots, route counts, incident badges per service
- [x] Collapsible domain headers
- [x] Search/filter input
- [x] Click service → selects node on map + opens inspector

### Routing View (🔀 icon rail tab)
- [x] Per-service toggle: Local (cloudmock) ↔ Cloud (dev/staging/prod)
- [x] Grouped by: API, Microservices, BFF Modules, AWS Services
- [x] Per-group "All Local" / "All Cloud" bulk toggles
- [x] Environment selector (dev / staging / prod)
- [x] Toggle switches with green (local) / teal (cloud) styling
- [x] Health check dots for local HTTP services (30s polling)
- [x] Search filter, endpoint display
- [x] Persisted to localStorage
- [x] Wire routing toggles to cloudmock proxy
  - [x] POST route changes to cloudmock's proxy config API
  - [x] Or inject AWS_ENDPOINT_URL per-service via env vars
  - [x] Visual confirmation that traffic is actually routing to selected target
- [x] Environment-specific config
  - [x] Load cloud endpoints from Pulumi stack outputs or env file
  - [x] Support multiple cloud environments per service
  - [x] Show which environment is "active" in the status bar

### Domain Config
- [x] service-domains.json with explicit domain groupings
- [x] User-facing overrides
- [x] Routing endpoints (local/cloud)
- [x] Edit domains from UI (settings or inline)
  - [x] Add/remove/rename domain groups
  - [x] Drag services between domains
  - [x] Save back to service-domains.json

---

## Round-Trip Waterfall (Traces View)

- [x] Full round-trip visualization with timing at each hop
- [x] Spans positioned by StartTime, width proportional to DurationMs
- [x] Color-coded by service
- [x] Parent/child indentation with connector lines
- [x] Critical path highlighting (yellow border on longest chain)
- [x] Gap detection (lighter bars for network/queuing time)
- [x] Status code coloring (green=2xx, yellow=4xx, red=5xx)
- [x] Click span → detail panel with metadata
- [x] Breakdown summary (per-service aggregate, bottleneck, upstream/downstream split)
- [x] Compare two traces side-by-side
  - [x] Select two traces from the list → split view
  - [x] Highlight differences in span timing
  - [x] Useful for before/after deploy comparison
- [x] Span flamegraph view (alternative to waterfall)
  - [x] Toggle between waterfall and flamegraph layout
  - [x] Flamegraph shows time on x-axis, call depth on y-axis

---

## Saved Layouts

- [x] Save current topology layout
  - [x] Capture: pinned node positions, zoom level, pan offset, collapse state
  - [x] Name the layout (e.g., "Incident Triage", "Full Architecture", "Data Flow")
  - [x] Store in localStorage keyed by topology config hash
- [x] "Layouts" dropdown in topology toolbar
  - [x] List saved layouts with preview thumbnails (optional)
  - [x] Click to apply: restore node positions, zoom, pan, collapse state
  - [x] Star/default layout (auto-loads on topology open)
- [x] Auto-save "last used" layout
  - [x] On navigate away from topology, save current positions
  - [x] On return, restore last used layout
  - [x] Differentiate from explicitly saved layouts
- [x] Import/export layout as JSON
  - [x] Export button → downloads layout.json
  - [x] Import button → file picker to load layout
  - [x] Shareable between team members
- [x] Reset to auto-layout
  - [x] "Reset Layout" button in toolbar
  - [x] Clears all pinned positions, reverts to ELK-computed layout
  - [x] Confirms before reset if there are pinned nodes

---

## Activity View

- [x] Live SSE stream from cloudmock
- [x] SSE auto-fallback: tries 3 URLs, falls back to polling /api/traces every 3s
- [x] Event merging with deduplication
- [x] Pause/resume toggle
- [x] Search filter (action, path, service)
- [x] Service dropdown filter
- [x] Status code category filter (2xx/3xx/4xx/5xx)
- [x] Event detail inspector (headers, body, timing)
- [x] Copy as curl button with clipboard feedback
- [x] Filter by source (SDK source chips as toggleable filters)
- [x] Request detail: show full headers and body (SyntaxHighlightedBody + collapsible sections)
- [x] Request diff: compare two requests side-by-side (RequestDiff component)
- [x] Request replay: re-send the same request to cloudmock
- [x] Export filtered requests as HAR file

---

## Other Views

### Metrics View (📊)
- [x] Per-service request count, avg latency, error rate computed from traces
- [x] SLO window data (P50/P95/P99) from /api/slo
- [x] Time-series charts (line charts over time, not just current values)
- [x] Service comparison (select 2+ services to compare side-by-side)

### SLOs View (🎯)
- [x] Health badge, rules table, compliance windows
- [x] Error budget burn-down chart (ErrorBudgetSection with LineChart)
- [x] SLO rule editor (create/edit rules from UI)

### Incidents View (🚨)
- [x] List with severity/status, acknowledge/resolve buttons
- [x] Incident timeline (horizontal timeline of alert grouping)
- [x] Related traces: auto-link traces from the incident time window
- [x] Root cause suggestions (from AI Debug)

### Profiler View (🔥)
- [x] Placeholder with feature cards
- [x] Wire to /api/profiles for real profiling data
- [x] Flame graph renderer (SVG flame-graph.tsx)
- [x] CPU/heap/goroutine profile tabs

### AI Debug View (🤖)
- [x] Request ID input, explain button, markdown renderer
- [x] Wire to working /api/explain endpoint
- [x] Streaming response support
- [x] Auto-suggest from selected trace or incident (recent failed traces)

### Chaos View (🧪)
- [x] Add/remove rules, toggle active, success/error feedback
- [x] Preset chaos scenarios ("Slow database", "Auth failure", "Network partition")
- [x] Scheduled chaos (run for X minutes then auto-disable)

---

## SDKs

### Node SDK (@cloudmock/node)
- [x] TCP JSON-line client (Connection class)
- [x] HTTP interceptor (http.request/get, https.request/get)
- [x] Fetch interceptor (globalThis.fetch)
- [x] Console interceptor (log/warn/error/info/debug)
- [x] Error interceptor (uncaughtException, unhandledRejection)
- [x] Auto-correlation (X-CloudMock-Source header)
- [x] Built with tsup (CJS + ESM + types)
- [x] Verify end-to-end with source server
- [ ] Test with BFF running LOCAL_DB=true
- [ ] Publish to npm

### Swift SDK (sdk/swift/)
- [x] NWConnection TCP client
- [x] URLProtocol HTTP interception
- [x] NSLog capture
- [x] Crash handler (NSSetUncaughtExceptionHandler + signals)
- [x] Verify Swift Package compiles
- [ ] Test with real iOS app

### Kotlin SDK (sdk/kotlin/)
- [x] Socket TCP client
- [x] OkHttp Interceptor
- [x] Log/Timber interceptor
- [x] Uncaught exception handler
- [ ] Verify Gradle build (no gradle/gradlew available; needs Gradle install or wrapper)
- [ ] Test with real Android app

---

## Infrastructure

### Rust Backend
- [x] Process manager (spawn/stop/restart cloudmock Go binary)
- [x] Health monitor (3s polling, Tauri events)
- [x] Admin API bridge (proxy frontend → cloudmock)
- [x] Source server (TCP :4580 for SDK connections)
- [x] System tray (show/hide, start/stop, quit)
- [x] SQLite persistence (history.db)
- [x] Capture gateway stdout/stderr, forward as Tauri events (process.rs emit)
- [x] Exponential backoff on health monitor connection failure (consecutive_failures backoff)
- [ ] BLE scanning module (btleplug crate)

### Frontend Infrastructure
- [x] ConnectionProvider context (configurable URLs)
- [x] ConnectionPicker (first launch modal)
- [x] Vite proxy (/api → localhost:4599)
- [x] SSE with auto-reconnect + multi-URL fallback
- [x] Error boundary (catch rendering errors, show recovery UI)
- [x] Keyboard shortcuts (Cmd+K search, Cmd+1-9 view switching)

### Production Build
- [x] macOS .app bundle (6.2MB binary)
- [x] Fix .dmg bundling (3.9MB DMG produced)
- [x] Generate proper app icon (N lettermark, neureaux brand colors)
- [ ] Code sign for macOS distribution
- [ ] Windows build target
- [ ] Linux AppImage target

---

## Summary

| Category | Done | Remaining |
|----------|------|-----------|
| Phase A: Live Service Map | 40 | 0 |
| Phase B: Inspector Tabs | 32 | 3 |
| Phase C: Timeline + Time Travel | 25 | 0 |
| Phase D: Service Browser + Routing | 28 | 0 |
| Waterfall | 16 | 0 |
| Saved Layouts | 20 | 0 |
| Activity View | 14 | 0 |
| Other Views | 22 | 0 |
| SDKs | 17 | 5 |
| Infrastructure | 17 | 4 |
| **Total** | **231** | **12** |
