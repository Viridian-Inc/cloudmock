---
title: Devtools Overview
description: CloudMock devtools -- browser-based observability console for local and hosted AWS emulation
---

The CloudMock devtools is a browser-based console that connects to any CloudMock instance (local or hosted) and provides real-time visibility into every AWS API call, trace, and metric flowing through the emulator.

## Accessing the devtools

The devtools UI is embedded in the CloudMock binary and served on port **4500**:

```
http://localhost:4500
```

This works for both local and hosted instances. No separate app install required.

## Connection modes

On first launch, the connection picker offers three options:

| Mode | Use case | Auth |
|------|----------|------|
| **Local Instance** | Development on your machine. Free, no account. | None |
| **CloudMock Cloud** | Hosted endpoint for CI/CD, staging, shared envs. Pay per use. | API key |
| **Custom Endpoint** | Any CloudMock instance by URL (self-hosted, teammate's machine). | Optional |

Connections are saved to localStorage. The status bar shows the active connection and environment.

## The views

The left-hand icon rail provides access to every view:

| View | Purpose |
|------|---------|
| **Activity** | Real-time request stream. Every AWS API call as it happens, with service, action, status, and latency. SSE streaming with polling fallback. |
| **Topology** | Live service map. Nodes = your services + AWS resources. Edges = traffic flow with call counts and latency. |
| **Services** | Browse all CloudMock services, health status, action counts, and resource inventories. |
| **Traces** | Distributed tracing. Waterfall and flamegraph views. Compare two traces side by side. |
| **Metrics** | Per-service dashboard. Request volume, latency percentiles (P50/P95/P99), error rates, time-series charts. |
| **Dashboards** | Custom metric dashboards. Build your own views with drag-and-drop widgets. |
| **S3 Browser** | Browse S3 buckets and objects visually. |
| **DynamoDB** | Browse DynamoDB tables, scan items, run queries. |
| **SQS Browser** | View SQS queues, messages, and dead-letter queues. |
| **Cognito** | Manage Cognito user pools and users. |
| **Lambda** | View Lambda function configs, logs, and invocations. |
| **IAM** | Browse IAM users, roles, and policies. |
| **Mail** | View SES emails sent through CloudMock. |
| **SLOs** | Service-level objective monitoring with compliance windows and error budgets. |
| **Incidents** | Incident tracking, acknowledgement, reports, and deploy correlation. |
| **Monitors** | Uptime and endpoint monitoring with alert thresholds. |
| **Profiler** | CPU and heap profiling with flamegraph visualization. |
| **Chaos** | Chaos engineering. Inject latency, errors, throttling into any service. |
| **Regressions** | Detect performance regressions across deploys. |
| **AI Debug** | AI-assisted debugging. Send a request ID, get an explanation of what happened and why. |
| **Routing** | Inspect the gateway routing table. |
| **Traffic** | Record and replay traffic patterns. |
| **RUM** | Real User Monitoring — web vitals, session data, browser errors. |
| **Settings** | Connection, theme, appearance, and preferences. |

### Platform views (CloudMock Cloud)

These views appear when connected to a hosted instance:

| View | Purpose |
|------|---------|
| **Apps** | Manage applications within your organization. |
| **API Keys** | Create and manage API keys for programmatic access. |
| **Usage** | View request counts, costs, and billing for the current period. |
| **Audit Log** | Timestamped log of all administrative actions. |
| **Settings** | Organization name, plan tier, and team management. |

## Architecture

The devtools communicate with CloudMock through two channels:

1. **Admin API** (port 4500, path `/api/*`) — RESTful endpoints for topology, metrics, traces, chaos rules, and more.

2. **SSE stream** (`/api/stream`) — Server-Sent Events push every request to the devtools in real time. The Activity view uses this as its primary data source, with polling as a fallback.

In local mode, the UI and API are served on the same origin (port 4500), so no CORS is needed.

## Keyboard shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd+1` through `Cmd+0` | Switch views |
| `Cmd+K` | Open command palette |
| `Cmd+L` | Snap to live mode (topology) |
| `Escape` | Deselect current node/event |

## Using with autotend

CloudMock is the default local AWS backend for autotend development. The autotend local dev setup (`autotend-infra/local/`) runs `pulumi up` against CloudMock to provision 40+ DynamoDB tables, Cognito pools, SQS queues, and Lambda functions. The devtools automatically discover these resources and display them in the topology view.

```bash
# Start cloudmock + autotend local dev
cd autotend-infra/local && pnpm dev:local
# Open devtools
open http://localhost:4500
```

All autotend services (BFF, API, GraphQL) appear as source nodes in the topology, with edges showing traffic flow to AWS resources.
