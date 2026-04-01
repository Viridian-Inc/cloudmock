# V1 Feature Roadmap — Prioritized Todo List

**Date:** 2026-04-01
**Goal:** Ship a competitive v1 that differentiates on local-first development + unified observability

---

## Priority Definitions
- **P0** — Must have for v1 launch. Without this, the product is not viable.
- **P1** — Should have for v1. Significantly increases adoption and competitiveness.
- **P2** — Nice to have. Differentiates but not blocking.
- **P3** — Future. Post-launch enhancement.

---

## Category 1: Error Tracking & Debugging

### 1.1 Structured Error Tracking with Grouping & Dedup
**Priority:** P0
**Summary:** Capture JS/backend errors with stack traces, group identical errors, track frequency, and assign to releases.
**Description:** Currently errors appear in the request log but aren't treated as first-class entities. Need: error inbox, fingerprinting for grouping, occurrence counting, first/last seen timestamps, affected users/sessions count, link to source code.
**Acceptance Criteria:**
- [ ] Errors ingested from RUM SDK (browser) and source SDK (server)
- [ ] Errors grouped by fingerprint (stack trace + message hash)
- [ ] Error list view with count, first/last seen, trend sparkline
- [ ] Error detail view with full stack trace, breadcrumbs, request context
- [ ] Errors linked to releases/deploys
**Justification:** Sentry's core value prop. Every developer needs this. Without it, we're not a real monitoring tool — just a fancy request logger.
**Opportunity:** This is what makes users choose us over "just use CloudWatch." Table-stakes for any observability product.

### 1.2 AI-Powered Root Cause Analysis
**Priority:** P1
**Summary:** When an error or regression occurs, automatically analyze traces, logs, and recent deploys to suggest the root cause.
**Description:** The `/api/explain` endpoint already generates narrative debug reports. Extend this to automatically run on new errors and regressions, producing actionable suggestions (specific file, function, or config change to investigate).
**Acceptance Criteria:**
- [ ] Auto-explain triggered on new error groups with >5 occurrences
- [ ] Explain output includes: probable cause, affected services, related deploys, suggested fix
- [ ] Displayed in error detail and incident views
**Justification:** New Relic AI and Datadog Watchdog both offer this. Our `/api/explain` is already close — just needs automation and better UX.
**Opportunity:** "AI that actually helps debug" is a strong differentiator if done well. Most competitors' AI features are surface-level.

---

## Category 2: Log Management

### 2.1 Unified Log Viewer
**Priority:** P0
**Summary:** Centralized log view that shows application logs, Lambda execution logs, and SDK-captured console output in one searchable timeline.
**Description:** CloudWatch Logs service exists and stores logs, but there's no dedicated log viewer in the DevTools. Need: full-text search, log level filtering (info/warn/error), source filtering (service name), time range selection, log-to-trace correlation (click a log line → see the trace).
**Acceptance Criteria:**
- [ ] Logs page in DevTools with live tail (SSE)
- [ ] Full-text search across all log sources
- [ ] Filter by level, service, time range
- [ ] Click log entry → jump to related trace
- [ ] Logs correlated with requests via trace ID
**Justification:** Every competitor has centralized logs. It's the #1 reason Platform Engineers (Persona 3) can't adopt us for production use. Without logs, we're a development tool only.
**Opportunity:** Logs + traces + metrics = the "three pillars." We have traces and metrics. Logs complete the story and unlock the production use case.

### 2.2 Log Forwarding from Source SDKs
**Priority:** P1
**Summary:** Node.js/Swift/Kotlin SDKs capture console.log/NSLog/Log.d output and forward to CloudMock.
**Description:** Part of Phase 2-3 SDK work. The Source SDK should intercept logging calls and forward them as structured events to the CloudMock source server.
**Acceptance Criteria:**
- [ ] Node.js SDK captures console.log/warn/error with context
- [ ] Logs appear in unified log viewer within 1 second
- [ ] Structured fields: timestamp, level, message, service, trace_id
**Justification:** This is what makes the "unified view" promise real. Without it, developers still need to check terminal output separately.
**Opportunity:** Sentry doesn't have logs. Honeycomb doesn't have logs. This is a chance to be more complete than both.

---

## Category 3: Performance Monitoring & RUM

### 3.1 Enhanced RUM with Error Correlation
**Priority:** P1
**Summary:** Extend RUM beyond web vitals to capture user interactions, errors, and correlate with backend traces.
**Description:** Current RUM collects LCP/FID/CLS/TTFB/FCP. Need: click tracking, rage click detection, user journey paths, error-to-RUM correlation (click a RUM error → see the backend trace), custom event tracking.
**Acceptance Criteria:**
- [ ] RUM SDK captures user clicks with element selector
- [ ] Rage click detection (3+ clicks on same element in 1s)
- [ ] User journey visualization (page flow)
- [ ] RUM error → backend trace linking
- [ ] Performance by route/page breakdown
**Justification:** Datadog RUM and Sentry Performance both offer this. Our RUM is currently vitals-only — useful but not actionable.
**Opportunity:** "See exactly what the user did, then see exactly what the server did" is a killer feature when it works end-to-end.

### 3.2 Session Replay
**Priority:** P2
**Summary:** Record and replay user browser sessions to reproduce bugs visually.
**Description:** DOM-based recording of user interactions (clicks, scrolls, input, navigation) with playback in DevTools. Privacy controls for masking sensitive fields.
**Acceptance Criteria:**
- [ ] RUM SDK records DOM mutations and user events
- [ ] Session replay player in DevTools
- [ ] Privacy: mask inputs, configurable selectors
- [ ] Link to errors and performance events
- [ ] Replay includes network waterfall
**Justification:** Sentry, Datadog, and Rollbar all have this. QA engineers (Persona 4) and mobile developers (Persona 2) love it for bug reproduction.
**Opportunity:** High-value feature but large effort. Consider post-v1 unless a simpler approach (screen recording via SDK) is viable.

### 3.3 Uptime / Endpoint Monitoring
**Priority:** P1
**Summary:** Periodic HTTP checks against configured endpoints, with alerting on failures.
**Description:** Simple synthetic monitoring: configure URL + expected status code + interval. Alert when check fails. Show uptime percentage and response time history.
**Acceptance Criteria:**
- [ ] Configure endpoint checks (URL, method, expected status, interval)
- [ ] Check results stored with response time
- [ ] Uptime percentage calculation (24h, 7d, 30d)
- [ ] Alert on consecutive failures
- [ ] Status page view showing all checks
**Justification:** Sentry just launched this. It's a low-effort high-value feature that Platform Engineers expect.
**Opportunity:** Quick win. Can be built on top of existing worker pool infrastructure (pkg/worker).

---

## Category 4: Alerting & Incident Management

### 4.1 Smart Alert Routing (Slack/PagerDuty/Email)
**Priority:** P0
**Summary:** Send alerts to Slack channels, PagerDuty, or email based on severity and service ownership.
**Description:** Webhook integration exists but needs proper Slack App integration (rich formatting, interactive buttons), PagerDuty events API v2 integration, and email delivery. Route alerts based on: service → team → channel mapping.
**Acceptance Criteria:**
- [ ] Slack App with rich message formatting and "Acknowledge" button
- [ ] PagerDuty Events API v2 integration (trigger/acknowledge/resolve)
- [ ] Email alerts via SES or SMTP
- [ ] Service → team → notification channel routing
- [ ] Alert suppression (mute during maintenance windows)
**Justification:** Every competitor has this. Without it, alerts only exist inside the dashboard — nobody will see them.
**Opportunity:** This is what turns a development tool into a production monitoring tool. Critical for Persona 3 and 5 adoption.

### 4.2 Anomaly Detection (ML-Powered)
**Priority:** P2
**Summary:** Automatically detect unusual patterns in metrics without manually-configured thresholds.
**Description:** Extend the existing regression detection engine (6 algorithms) with baseline learning: track normal patterns per service, alert when behavior deviates significantly. Similar to Datadog Watchdog / Honeycomb BubbleUp.
**Acceptance Criteria:**
- [ ] Automatic baseline learning per service (7-day rolling window)
- [ ] Anomaly alerts for latency, error rate, throughput deviations
- [ ] "What changed?" analysis when anomaly detected
- [ ] Low false-positive rate (<5% after warm-up)
**Justification:** Already have regression detection as a foundation. This is the natural evolution.
**Opportunity:** "Zero-config alerting" is a strong pitch for teams without dedicated SREs.

---

## Category 5: Integrations

### 5.1 OpenTelemetry (OTLP) Ingestion
**Priority:** P0
**Summary:** Accept traces, metrics, and logs via the OpenTelemetry Protocol (OTLP/gRPC and OTLP/HTTP).
**Description:** OpenTelemetry is the industry standard. Platform Engineers (Persona 3) won't consider a tool that doesn't support it. Need: OTLP/gRPC endpoint on port 4317, OTLP/HTTP endpoint on port 4318, trace/metric/log mapping to our internal data model.
**Acceptance Criteria:**
- [ ] OTLP/gRPC endpoint accepting traces
- [ ] OTLP/HTTP endpoint accepting traces
- [ ] OTLP metrics ingestion
- [ ] OTLP logs ingestion
- [ ] Traces appear in existing trace viewer
- [ ] Works with otel-collector and direct SDK export
**Justification:** New Relic, Datadog, Sentry, and Honeycomb all support OTLP. It's the minimum bar for production adoption. Without it, we're development-only.
**Opportunity:** "Works with your existing OpenTelemetry instrumentation" means zero migration cost. Huge for adoption.

### 5.2 GitHub / GitLab Integration
**Priority:** P1
**Summary:** Link errors and incidents to source code, commits, and PRs.
**Description:** Connect to GitHub/GitLab to: show source code context in stack traces, identify suspect commits (git blame on error stack frames), link to PRs, show deployment history from CI.
**Acceptance Criteria:**
- [ ] GitHub OAuth app for repository access
- [ ] Stack trace → source code context (show surrounding lines)
- [ ] Suspect commits analysis
- [ ] Deploy events from GitHub Actions / GitLab CI
**Justification:** Sentry's suspect commits feature is one of their most loved. It turns "here's an error" into "here's who probably caused it and when."
**Opportunity:** Strong differentiator combined with IaC topology — "see the architecture AND the code."

### 5.3 Slack App (Rich Integration)
**Priority:** P1
**Summary:** Full Slack App with interactive messages, slash commands, and bot.
**Description:** Beyond webhook notifications: rich error/incident cards with "Acknowledge" / "Resolve" buttons, `/cloudmock status` slash command, alert channel routing.
**Acceptance Criteria:**
- [ ] Slack App (not just incoming webhook)
- [ ] Interactive message buttons for incidents
- [ ] Slash command for status queries
- [ ] Channel routing per service/team
**Justification:** Slack is where developers live. Actionable alerts in Slack > alerts in a dashboard nobody checks.
**Opportunity:** Low effort, high impact. Most teams evaluate tools partially on Slack integration quality.

---

## Category 6: Developer Experience

### 6.1 One-Command Bootstrap (`cmk` CLI)
**Priority:** P0
**Summary:** `npx cloudmock` or `cmk start` should boot everything with zero config.
**Description:** Currently requires `go run ./cmd/gateway/ --iac /path/to/pulumi --iac-env dev` with correct flags. Need: `cmk` CLI wrapper that auto-discovers IaC, reads `.cloudmock.yaml` config, and starts with sensible defaults. Similar to `localstack start`.
**Acceptance Criteria:**
- [ ] `npx cloudmock` boots gateway with default config
- [ ] Auto-discovers Pulumi/Terraform in parent directories
- [ ] `.cloudmock.yaml` for persistent config
- [ ] `cmk start`, `cmk stop`, `cmk status`, `cmk logs`
- [ ] Port configuration via env vars or config
**Justification:** User feedback: "must be as easy as LocalStack — one command, zero config." This is the #1 adoption barrier.
**Opportunity:** If setup takes >2 minutes, developers will abandon. This is existential for growth.

### 6.2 Source SDK for Node.js
**Priority:** P0
**Summary:** `@cloudmock/node` SDK that auto-instruments HTTP requests, captures logs, and forwards to DevTools.
**Description:** Phase 2 deliverable. Intercepts: fetch, http/https modules, console.* calls. Adds `X-CloudMock-Source` header for request correlation. Auto-discovers CloudMock via mDNS or localhost fallback.
**Acceptance Criteria:**
- [ ] `npm install @cloudmock/node` + one-line setup
- [ ] HTTP/fetch interception (outgoing requests)
- [ ] Console.log/warn/error capture
- [ ] Request → trace correlation
- [ ] Auto-discovery of CloudMock endpoint
**Justification:** Without a source SDK, DevTools can only observe CloudMock's side of the conversation. The SDK completes the story: "what your app did" + "what AWS did."
**Opportunity:** This is what makes us a *platform* not just a mock server. The SDK is our distribution channel.

### 6.3 Documentation Site
**Priority:** P0
**Summary:** Comprehensive docs at cloudmock.io/docs with getting started guide, API reference, and SDK guides.
**Description:** Starlight/Astro-based docs site. Must have: 5-minute quickstart, service compatibility matrix, API reference (all 55+ endpoints), SDK setup guides, FAQ.
**Acceptance Criteria:**
- [ ] Getting started guide (< 5 minutes to first request)
- [ ] Service compatibility matrix (98 services)
- [ ] API reference (auto-generated from code)
- [ ] Node.js SDK guide
- [ ] Deployment guide (Docker, npm, brew)
**Justification:** No docs = no adoption. Period.
**Opportunity:** Good docs are a competitive advantage. LocalStack's docs are mediocre. Sentry's are excellent — that's the bar.

---

## Category 7: Platform & Enterprise

### 7.1 Multi-Environment Support
**Priority:** P1
**Summary:** Switch between dev/staging/production environments in DevTools.
**Description:** Connection profiles already exist. Extend to: named environments (dev, staging, prod), environment-specific config, data isolation between environments.
**Acceptance Criteria:**
- [ ] Environment selector in DevTools header
- [ ] Per-environment connection settings
- [ ] Visual indicator of current environment (color coding)
- [ ] Warn when connecting to production
**Justification:** Teams have multiple environments. Without this, DevTools is local-only.
**Opportunity:** This is what unlocks the paid tier — monitoring staging/production.

### 7.2 Team Collaboration Features
**Priority:** P2
**Summary:** Shared dashboards, saved views, annotations, and comments on incidents.
**Description:** Currently single-user. Need: shared dashboard ownership, @mentions in incident comments, saved view sharing, annotations on timeline (mark deploy events, incidents, etc.).
**Acceptance Criteria:**
- [ ] Shared dashboards (team-visible)
- [ ] Incident comments with @mentions
- [ ] Annotations on metric timeline
- [ ] Activity feed (who did what)
**Justification:** Production monitoring is a team sport. Solo features → team features is the natural growth path.
**Opportunity:** Team features drive seat expansion ($29 → $99/mo as team grows).

### 7.3 Data Retention & Export
**Priority:** P1
**Summary:** Configurable data retention with export to S3/CloudWatch/external systems.
**Description:** Currently in-memory (local) or 30-day PostgreSQL (production). Need: configurable retention per data type, export to S3 for long-term storage, CSV/JSON export from UI.
**Acceptance Criteria:**
- [ ] Configurable retention per data type (traces: 7d, metrics: 30d, logs: 14d)
- [ ] S3 export for archival
- [ ] CSV/JSON export from any table/chart in UI
**Justification:** Enterprise compliance requires data retention controls.
**Opportunity:** Data retention is a paid-tier differentiator.

---

## Summary: Priority Order

### P0 — Must Ship for V1
1. One-command bootstrap (`cmk` CLI) — DX
2. Source SDK for Node.js — DX
3. Documentation site — DX
4. Structured error tracking with grouping — Error Tracking
5. Unified log viewer — Log Management
6. OpenTelemetry (OTLP) ingestion — Integrations
7. Smart alert routing (Slack/PagerDuty/Email) — Alerting

### P1 — Ship Within 2 Weeks of V1
8. Enhanced RUM with error correlation — Performance
9. Uptime / endpoint monitoring — Performance
10. GitHub / GitLab integration — Integrations
11. Slack App (rich integration) — Integrations
12. Log forwarding from source SDKs — Log Management
13. AI-powered root cause analysis — Error Tracking
14. Multi-environment support — Platform
15. Data retention & export — Platform

### P2 — Ship Within 1 Month of V1
16. Session replay — Performance
17. Anomaly detection (ML-powered) — Alerting
18. Team collaboration features — Platform
19. CI/CD visibility — Integrations
20. Database query monitoring — Performance

### P3 — Post-V1
21. Mobile crash reporting (Swift/Kotlin SDKs)
22. Natural language querying
23. Synthetic multi-step browser tests
24. Security monitoring
25. Plugin marketplace
