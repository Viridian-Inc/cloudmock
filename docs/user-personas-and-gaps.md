# Target User Personas & Gap Analysis

**Date:** 2026-04-01

---

## Target User Personas

### Persona 1: Solo Full-Stack Developer / Indie Hacker
**Name:** Alex | **Age:** 24-32 | **Company:** 1-5 person startup
**Role:** Builds everything — frontend, backend, infra

**Context:**
- Building on AWS (Lambda, DynamoDB, S3, Cognito)
- Can't afford $200+/mo for separate monitoring tools
- Needs local dev environment that "just works"
- Ships fast, breaks things, needs to debug quickly
- Probably uses Sentry free tier for errors

**Pain Points:**
- LocalStack is finicky and incomplete for their stack
- Can't see what's happening inside their Lambda functions locally
- Sentry catches production errors but doesn't help during development
- No budget for Datadog/New Relic
- Context switching between 4+ tools (LocalStack, Sentry, CloudWatch, Postman)

**Decision Criteria:** Free/cheap, fast setup, covers dev + basic prod monitoring

**What They'd Pay:** $0 locally, $29/mo for hosted if it replaces 2+ tools

**Our Fit:** ✓✓ Perfect. Free local, replaces LocalStack + basic monitoring. The "single pane of glass" they can't afford elsewhere.

---

### Persona 2: Frontend/Mobile Engineer at a Startup
**Name:** Jordan | **Age:** 25-35 | **Company:** 10-50 person startup
**Role:** React Native / Expo developer, touches backend occasionally

**Context:**
- Building cross-platform mobile app (iOS + Android + Web)
- Flipper deprecated, no good replacement
- Needs to debug network requests, see API responses, understand errors
- Cares about app performance (launch time, frame rate)
- Backend team uses Datadog but they don't have access

**Pain Points:**
- Flipper is gone — no unified mobile devtools
- React Native debugging is fragmented (Reactotron, Chrome DevTools, Xcode)
- Can't see what the backend is doing when their API call fails
- No way to test against AWS services without deploying
- Performance issues are hard to reproduce

**Decision Criteria:** Works with React Native/Expo, unified view, team-friendly

**What They'd Pay:** Free for local, team pays $99/mo if it replaces Flipper + gives backend visibility

**Our Fit:** ✓✓ Excellent. Desktop devtools replaces Flipper. Source SDKs (Phase 2-3) give cross-platform visibility. BLE mesh topology is unique differentiator for mobile teams.

---

### Persona 3: Platform / DevOps Engineer
**Name:** Sam | **Age:** 28-40 | **Company:** 50-500 person company
**Role:** Manages infrastructure, CI/CD, observability stack

**Context:**
- Runs AWS infrastructure (ECS, Lambda, DynamoDB, etc.)
- Currently uses Datadog or New Relic ($5K-50K/yr)
- Responsible for SLOs, incident response, cost optimization
- Evaluates tools for the engineering org
- Cares about OpenTelemetry, vendor lock-in, total cost

**Pain Points:**
- Datadog bills are unpredictable and growing
- Engineers don't use the monitoring tools because they're too complex
- Local dev environments don't match production
- No way to test failure scenarios before they happen in prod
- Alert fatigue — too many noisy alerts

**Decision Criteria:** Cost savings, ease of adoption, production-grade reliability, OTel support

**What They'd Pay:** $99/mo team plan if it replaces part of Datadog bill

**Our Fit:** ✓ Good for dev, ○ needs work for production. Missing: OpenTelemetry ingestion, log management, production-grade data retention. Chaos engineering is a unique sell. IaC topology is compelling.

---

### Persona 4: QA / Test Engineer
**Name:** Riley | **Age:** 26-35 | **Company:** 20-200 person company
**Role:** Writes E2E tests, validates features, investigates bugs

**Context:**
- Runs Maestro/Detox for mobile E2E tests
- Needs to reproduce bugs reported by users
- Wants to understand what happened when a test fails
- Uses Sentry to see production errors, tries to reproduce locally

**Pain Points:**
- "Works on my machine" — can't reproduce the AWS environment
- E2E test failures are black boxes — no visibility into what services did
- Manual testing against real AWS is slow and costs money
- No session replay for mobile
- Can't inject failures to test error handling

**Decision Criteria:** Reproducibility, visibility into test runs, failure injection

**What They'd Pay:** Free for local (included in team's subscription)

**Our Fit:** ✓✓ Excellent. Local AWS emulation for reproducible tests. Chaos engineering for failure testing. Trace visibility shows exactly what happened. Missing: session replay would be huge for this persona.

---

### Persona 5: Engineering Manager / CTO at Early-Stage Startup
**Name:** Morgan | **Age:** 30-45 | **Company:** 5-30 person startup
**Role:** Makes tooling decisions, manages budget, sets technical direction

**Context:**
- Evaluated LocalStack, Sentry, Datadog — using some combo
- Paying for 3+ tools that overlap
- Wants to reduce tool sprawl and cost
- Needs something the whole team can use (not just backend)
- Thinking about production readiness and scaling

**Pain Points:**
- Paying $500+/mo across LocalStack Pro + Sentry + basic Datadog
- Engineers waste time context-switching between tools
- No unified view of the system — have to check 4 dashboards
- Hard to justify monitoring spend to investors
- Want observability from day 1, not bolted on later

**Decision Criteria:** Total cost of ownership, team adoption, consolidation, growth path

**What They'd Pay:** $99-199/mo if it truly replaces 2-3 tools

**Our Fit:** ✓✓ Strong. Single platform for dev + observability. Free local, $29-99 hosted. Replaces LocalStack + fills monitoring gaps. The "one tool" story is compelling for budget-conscious CTOs.

---

## Gap Analysis: Feature-by-Feature

### What We're Missing (Gaps)

| Gap | Severity | Who Cares | Competitor Benchmark | Effort |
|-----|----------|-----------|---------------------|--------|
| **Log management / unified log viewer** | P0 | Personas 1,3,5 | Every competitor has centralized logs | Medium |
| **OpenTelemetry (OTLP) ingestion** | P0 | Persona 3 | New Relic, Datadog, Sentry, Honeycomb all support | Medium |
| **Session replay** | P1 | Personas 2,4 | Sentry, Datadog, Rollbar | Large |
| **Error tracking with grouping/dedup** | P1 | Personas 1,2,4 | Sentry is the gold standard | Medium |
| **Synthetic monitoring** | P2 | Persona 3 | New Relic, Datadog | Medium |
| **Mobile crash reporting** | P1 | Persona 2 | Sentry dominates | Large (Phase 3) |
| **Integrations (Slack, PagerDuty, Jira, GitHub)** | P1 | All | 100-800+ integrations on competitors | Medium |
| **CI/CD visibility** | P2 | Persona 3 | Datadog unique offering | Medium |
| **ML-powered anomaly detection** | P2 | Persona 3 | New Relic AIOps, Datadog Watchdog, Honeycomb BubbleUp | Large |
| **Natural language querying** | P3 | All | New Relic AI, Datadog Bits AI | Medium |
| **Uptime / endpoint monitoring** | P1 | Personas 1,3 | Sentry (new), New Relic, Datadog | Small |
| **Database query monitoring** | P2 | Personas 1,3 | Datadog, New Relic | Medium |

### What We Uniquely Have (Advantages)

| Advantage | Value | Who Cares Most |
|-----------|-------|---------------|
| **Local AWS emulation (98 services)** | No competitor offers this | All — eliminates need for AWS account in dev |
| **Desktop devtools (Flipper replacement)** | No equivalent exists | Persona 2 — mobile developers |
| **Built-in chaos engineering** | Competitors need separate tools | Personas 3,4 — reliability & testing |
| **IaC-driven topology** | Automatic architecture visualization from Pulumi/Terraform | Personas 3,5 — infrastructure visibility |
| **Free unlimited local usage** | Competitors charge from first event/host | Personas 1,5 — cost-sensitive |
| **Tenant-aware observability** | Built-in from day 1, not bolted on | Persona 5 — SaaS builders |
| **Unified dev + production** | Same tool for local dev and production monitoring | All — reduces tool sprawl |

### What We Have That's On-Par

| Feature | Our Status | Notes |
|---------|-----------|-------|
| Distributed tracing | ✓ Solid | Span merging, parent/child, waterfall view |
| SLOs / SLIs | ✓ Solid | Burn rate, error budgets, per-tenant |
| Incident management | ✓ Good | Auto-creation, lifecycle, grouping |
| Custom dashboards | ✓ Good | Metric query DSL, widgets |
| Regression detection | ✓ Good | 6 algorithms, deploy correlation |
| Profiling | ✓ Good | CPU/heap, flame graphs, source maps |
| Cost intelligence | ✓ Unique angle | Per-request, per-service, per-tenant |
| RUM (web vitals) | ✓ Basic | LCP, FID, CLS, TTFB, FCP |
| RBAC + audit logging | ✓ Enterprise-ready | JWT, 3 roles, full audit trail |
