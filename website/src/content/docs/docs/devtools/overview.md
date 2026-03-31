---
title: Devtools Overview
description: CloudMock devtools -- a desktop observability console for your local AWS emulator
---

The CloudMock devtools are a desktop application (built on Tauri + Preact) that connects to your running CloudMock instance and provides real-time visibility into every AWS API call, trace, and metric flowing through the emulator.

## Accessing the devtools

The devtools web UI is served by the CloudMock dashboard on port **4500** by default. Once CloudMock is running, open your browser:

```
http://localhost:4500
```

Alternatively, install the standalone desktop app from the `neureaux-devtools` package, which connects to CloudMock's admin API on port 4599.

## The 12 views

The left-hand icon rail provides access to every view in the devtools:

| View | Purpose |
|------|---------|
| **Activity** | Real-time request stream. Every AWS API call appears as it happens, with service, action, status code, and latency. Supports SSE streaming and polling fallback. |
| **Topology** | Live service map. Nodes represent your services and the AWS resources they use. Edges show traffic flow with call counts and latency. |
| **Services** | Browse all registered CloudMock services, their health status, action counts, and resource inventories. |
| **Traces** | Distributed tracing. Waterfall and flamegraph views for every request. Compare two traces side by side. |
| **Metrics** | Per-service metrics dashboard. Request volume, latency percentiles (P50/P95/P99), error rates, and time-series charts. |
| **SLOs** | Service-level objective monitoring. Define SLO rules, view compliance windows, and track violations. |
| **Incidents** | Incident tracking. View active incidents, acknowledge them, generate reports, and correlate with deploys. |
| **Profiler** | CPU and heap profiling. Capture profiles for any service and view them as flamegraphs or download as pprof. |
| **Chaos** | Chaos engineering. Inject latency, errors, and throttling into any service or action. Timer-based auto-disable. |
| **AI Debug** | AI-assisted debugging. Send a request ID and get an AI-generated narrative explaining what happened and why. |
| **Routing** | Inspect the internal routing table. See how CloudMock maps AWS service names and X-Amz-Target headers to service implementations. |
| **Settings** | Configure the devtools connection, theme, language, and other preferences. |

## Architecture

The devtools communicate with CloudMock through two channels:

1. **Admin API** (port 4599) -- RESTful endpoints for fetching topology, metrics, traces, chaos rules, and more. The full API reference lists 46+ endpoints across 15 categories.

2. **SSE stream** (`/api/stream`) -- A Server-Sent Events endpoint that pushes every request to the devtools in real time as it flows through the gateway. The Activity view uses this as its primary data source, with polling as a fallback.

## Connection picker

When multiple CloudMock instances are running (for example, one per microservice or one per environment), the connection picker in the status bar lets you switch between them. Each connection is identified by its admin API URL.

## Keyboard shortcuts

The devtools support keyboard navigation for switching between views and interacting with panels. View-specific shortcuts are documented on each view's page.
