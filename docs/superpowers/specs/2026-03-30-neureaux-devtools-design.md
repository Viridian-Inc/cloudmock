# neureaux-devtools — Cross-Platform Developer Tools

**Date:** 2026-03-30
**Status:** Design
**Location:** New top-level project at `neureaux-devtools/` in the monorepo

## Problem

Developing across multiple platforms (iOS, Android, Web, backend) requires juggling separate tools: Safari DevTools for iOS, Chrome DevTools for Android/Web, framework-specific debuggers (Flutter DevTools, React Native Debugger), and the cloudmock dashboard for cloud service inspection. There is no single tool that unifies debugging, logging, network inspection, and cloud service observability across all platforms — especially since Meta deprecated Flipper in 2024.

## Vision

**cloudmock as a developer platform** (modeled after Kong's platform approach):

| Layer | Component | Role |
|-------|-----------|------|
| Runtime | **cloudmock gateway** (open-source) | 101-service cloud emulator with plugin system. Runs locally, hosted, or in CI. |
| Platform | **cloudmock console** (already built) | Observability, traces, SLOs, AI explain, incidents, profiling, chaos engineering. |
| Desktop tool | **neureaux devtools** (this spec) | Tauri desktop app — unified debugging across all platforms + cloud services. |
| Control plane | **cloudmock.io** (future) | Hosted instances, team management, governance, billing. Marketing site + web console. |
| CLI | **cloudmock CLI** (existing) | `cloudmock start/stop/status` + tool wrappers (`cloudmock-aws`, `cloudmock-cdk`, etc.) |

The gateway stays open-source. Monetization is in the hosted platform (cloudmock.io) and potentially premium desktop features.

## Product: neureaux devtools

A **Tauri v2** desktop application with a purpose-built **Preact + Vite + TypeScript** frontend. Not a wrapper around the existing cloudmock dashboard — a new UI designed for desktop from the ground up, talking to cloudmock's admin API.

### Architecture

```
┌─────────────────────────────────────────────────┐
│  neureaux devtools (Tauri v2)                   │
│                                                 │
│  ┌───────────────────────────────────────────┐  │
│  │  Frontend (Preact + Vite + TypeScript)    │  │
│  │  Activity · Topology · Services · Traces  │  │
│  │  Metrics · SLOs · Incidents · Profiler    │  │
│  │  Chaos · AI Debug                         │  │
│  └───────────────────────────────────────────┘  │
│                                                 │
│  ┌───────────────────────────────────────────┐  │
│  │  Rust Backend (Tauri Commands)             │  │
│  │  Process Manager · Source Server (WS)     │  │
│  │  BLE Scanner · System Tray                │  │
│  └───────────────────────────────────────────┘  │
└─────────────────────┬───────────────────────────┘
                      │ IPC + HTTP + WebSocket
        ┌─────────────┼─────────────────┐
        ▼             ▼                 ▼
  ┌──────────┐  ┌──────────┐  ┌──────────────┐
  │cloudmock │  │ App SDKs │  │ BLE Devices  │
  │ gateway  │  │ (sources)│  │ (mesh topo)  │
  │ (Go)     │  │          │  │              │
  │ :4566    │  │ :4580 WS │  │ CoreBluetooth│
  │ :4599    │  │          │  │ / Android BLE│
  └──────────┘  └──────────┘  └──────────────┘
```

**Key decisions:**

- **Headless cloudmock** — the Go binary runs without its dashboard (port 4500 disabled via `dashboard.enabled: false` in cloudmock.yml, or a `--headless` flag if needed as a Phase 1 prerequisite). The Tauri frontend talks directly to the admin API on :4599.
- **Same tech stack as existing dashboard** (Preact + Vite + TS) so existing dashboard components can be migrated as needed. API client code is directly reusable.
- **Rust-native backends** — process management, WebSocket source server, and BLE scanning live in Rust. The frontend gets data via Tauri IPC commands and events.
- **Event-driven** — Tauri v2 events push real-time data (log lines, device state, network requests, BLE topology changes) to the frontend. No polling.

### Connection Model

On first launch, the app presents a connection picker:

1. **Local Instance** — starts the cloudmock Go binary on the user's machine. Free, no account needed, full 100 services. Auto-starts on next launch.
2. **cloudmock.io** — connects to a hosted instance. Requires API key from the web console. Team sharing, persistent state, CI integration.
3. **Custom Endpoint** — any cloudmock instance by URL (self-hosted, teammate's machine, etc.).

Connections are saved as profiles. The status bar shows the active connection and allows switching. The app is endpoint-agnostic — it doesn't know or care whether it's talking to a local Go binary or a hosted container.

### UI Layout: Workspace Panels (Flipper Model)

The app uses a **resizable split-panel layout** with an icon rail, inspired by Meta's Flipper:

- **Icon rail** (left, 56px) — switches between views. Icons represent views, not platforms.
- **Connected sources bar** (top) — shows all connected apps as color-coded chips with runtime + app-name (e.g., "Node · autotend-bff", "Swift · autotend-ios").
- **Resizable panels** (center) — each view's content area. Panels can be split, resized, and rearranged.
- **Inspector panel** (right, collapsible) — detail view for the selected item. Shows both client-side and server-side data.
- **Status bar** (bottom) — connection info, region, source count, event rate.

### Views (Icon Rail)

Every view shows data from ALL connected sources. There are no "cloud-only" or "device-only" screens.

| Icon | View | Description |
|------|------|-------------|
| ⚡ | **Activity** | Unified chronological stream from all sources. Logs, network requests, errors, cloudmock API calls interleaved. Filter by source, level, type. Click any event → inspector. |
| 🗺️ | **Topology** | Service graph. Nodes = apps + cloudmock services + BLE devices. Edges = traffic (HTTP) and connections (BLE mesh). Health-colored nodes, traffic-sized edges. Auto-discovered from request correlation and BLE scanning. |
| ☁️ | **Services** | Resource browser for cloud services — S3 buckets, DDB tables, Lambda functions, SQS queues, Cognito pools, K8s namespaces, ArgoCD apps. |
| 🔍 | **Traces** | Distributed trace waterfall. Includes spans from client apps — full journey from button tap → HTTP request → Lambda → DynamoDB → response. |
| 📊 | **Metrics** | Request volume, latency percentiles, error rates. Includes client-side metrics (app launch time, frame rate, crash rate) per source. |
| 🎯 | **SLOs** | SLO burn rate dashboard, error budgets. Extended for client-side SLOs (e.g., "95% of Swift API calls complete under 500ms"). |
| 🚨 | **Incidents** | Alert grouping, incident timeline, regression detection. Client-side crashes surface alongside server-side incidents. |
| 🔥 | **Profiler** | Flame graphs, CPU/heap profiling. |
| 🧪 | **Chaos** | Inject latency, errors, throttling into cloudmock services. Test how apps handle degraded backends. |
| 🤖 | **AI Debug** | Select any request, trace, or error → get a narrative explanation with full cross-stack context. |
| ⚙️ | **Settings** | Connection profiles, config, appearance, SDK setup instructions. |

### Source Integration: Framework-Agnostic

The devtools works at the **runtime level**, not the framework level. Three SDKs cover all platforms:

| SDK | Hooks | Covers |
|-----|-------|--------|
| **cloudmock/node** | `http`/`https` module, `fetch` global, `console.*`, `process.on('uncaughtException')` | React, React Native, Next.js, Express, Fastify, Hono, Remix, Svelte, Vue, Angular, Bun, Deno, Electron — anything JS/TS |
| **cloudmock/swift** | `URLSession`, `os.Logger`/`NSLog`, `NSSetUncaughtExceptionHandler`, `MetricKit` | SwiftUI, UIKit, AppKit, Vapor, Hummingbird — any Swift app (iOS, macOS, tvOS, watchOS, server-side) |
| **cloudmock/kotlin** | `OkHttp` interceptor, `HttpURLConnection`, `Log.*`/`Timber`, `Thread.setDefaultUncaughtExceptionHandler` | Jetpack Compose, XML Views, Kotlin Multiplatform, Ktor, Spring Boot — any Kotlin/JVM app |

A **Dart SDK** can be added as a 4th runtime if there's demand (hooks `HttpClient`, `dart:developer`, zone error handling).

**What SDKs capture:**
- **Network requests** — URL, method, headers, body, timing, status. Auto-correlates with cloudmock's server-side log via `X-CloudMock-Source` header injection.
- **Console logs** — level, timestamp, source location (file:line). Platform-native interception.
- **Errors & crashes** — uncaught exceptions with stack traces.
- **Performance** — app launch time, frame rate (mobile), memory usage, navigation timing.

**SDK properties:**
- **Dev-only** — wrapped in `kDebugMode`/`__DEV__`/`process.env.NODE_ENV` checks. Tree-shaken in production. Zero overhead in release builds.
- **Auto-discovery** — SDKs find the devtools app via mDNS/Bonjour on the local network. Falls back to `CLOUDMOCK_DEVTOOLS_HOST` environment variable or manual config for environments where mDNS is blocked (corporate networks, CI).
- **Graceful degradation** — if devtools isn't running, the SDK silently no-ops.
- **Automatic correlation** — `X-CloudMock-Source` header links client-side requests to cloudmock's server-side log entries by request ID.

**No-SDK alternatives:**
- **Proxy mode** — route traffic through devtools (localhost:4580) instead of cloudmock directly. Captures network traffic with zero app changes. No logs/crashes.
- **cloudmock-only** — cloudmock's own request log shows all API calls. Loses client-side perspective.

Sources identify as **runtime · app-name** in the UI (e.g., "Node · autotend-bff", "Swift · autotend-ios"). The devtools doesn't know or care what framework is running.

### BLE Mesh Topology

The Topology view includes **Bluetooth Low Energy mesh visualization** for phone/device networks:

- **BLE scanning** in Rust via CoreBluetooth (macOS/iOS) and platform APIs.
- **Device nodes** — phones, peripherals, beacons shown on the topology graph alongside cloud services.
- **BLE edges** — connections between devices with RSSI signal strength as edge weight.
- **Mesh topology** — visualize how devices discover each other, connection state, data flow between mesh nodes.
- **Unified graph** — a phone node can have both BLE edges (to other devices) and HTTP edges (to cloudmock services) in the same topology view.

### Project Structure

```
neureaux-devtools/
├── src-tauri/               # Rust backend
│   ├── src/
│   │   ├── main.rs          # Tauri app entry
│   │   ├── commands/        # IPC command handlers
│   │   │   ├── process.rs   # cloudmock process management
│   │   │   ├── sources.rs   # WebSocket source server
│   │   │   ├── ble.rs       # BLE scanning + mesh topology
│   │   │   └── config.rs    # Config read/write
│   │   ├── bridge/          # cloudmock admin API client
│   │   └── tray.rs          # System tray
│   ├── Cargo.toml
│   └── tauri.conf.json
├── src/                     # Preact frontend
│   ├── app.tsx              # Root app component
│   ├── views/               # One per icon rail view
│   │   ├── activity/        # Unified activity stream
│   │   ├── topology/        # Service + BLE mesh graph
│   │   ├── services/        # Cloud resource browser
│   │   ├── traces/          # Distributed trace waterfall
│   │   ├── metrics/         # Metrics dashboard
│   │   ├── slos/            # SLO burn rates
│   │   ├── incidents/       # Incident management
│   │   ├── profiler/        # Flame graphs
│   │   ├── chaos/           # Chaos engineering
│   │   ├── ai-debug/        # AI explain
│   │   └── settings/        # Config + connections
│   ├── components/          # Shared UI components
│   │   ├── panels/          # Resizable panel system
│   │   ├── inspector/       # Detail inspector
│   │   ├── source-bar/      # Connected sources bar
│   │   └── icon-rail/       # Navigation rail
│   ├── hooks/               # Tauri IPC + SSE hooks
│   └── lib/                 # Admin API client, types
├── sdk/                     # Client SDKs (or separate repos)
│   ├── node/                # @cloudmock/node
│   ├── swift/               # CloudMockSDK (Swift package)
│   └── kotlin/              # cloudmock-sdk (Maven/Gradle)
├── package.json             # Vite + Preact deps
├── vite.config.ts
└── tsconfig.json
```

### Phasing

**Phase 1 — Desktop App MVP**
- Tauri shell with icon rail, resizable panels, system tray
- Rust process manager (spawn/stop/restart cloudmock Go binary, health monitoring)
- Connection picker (local only)
- Activity view (unified stream from cloudmock SSE + placeholder for SDK sources)
- Services view (resource browser via admin API)
- Settings view (connection profiles, cloudmock config)

**Phase 2 — Source SDKs + Full Observability**
- Node SDK (@cloudmock/node) — HTTP interception, console capture, error handling
- WebSocket source server in Rust (localhost:4580)
- mDNS/Bonjour auto-discovery
- Activity view with real multi-source data
- Topology view (service graph + app nodes)
- Traces view (distributed waterfall with client spans)
- Metrics view with client-side metrics

**Phase 3 — Native SDKs + BLE**
- Swift SDK (CloudMockSDK)
- Kotlin SDK (cloudmock-sdk)
- BLE mesh scanning and topology visualization
- SLOs, Incidents, Profiler, Chaos, AI Debug views
- Proxy mode (no-SDK network capture)

**Phase 4 — cloudmock.io + Hosted Platform**
- cloudmock.io marketing site + web console
- Hosted cloudmock instances (container orchestration)
- API key auth, desktop app connects to hosted endpoint
- Team sharing, billing, usage dashboards

**Phase 5 — Developer Portal + Ecosystem**
- Service catalog with API docs browser
- Plugin marketplace
- Community plugins (GCP, Azure emulation)
- Third-party integrations (Datadog, PagerDuty, Slack)
