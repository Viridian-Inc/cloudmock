# CloudMock v1 SaaS Launch Design

## Overview

Ship CloudMock as a dual-mode product: a free local CLI that developers install and run instantly, plus a paid hosted SaaS tier at cloudmock.io. 25 Tier 1 AWS services work flawlessly (24 existing + AppSync promoted from Tier 2, making 25 Tier 1 + 73 Tier 2 = 98 total). Browser-based devtools embedded in the Go binary (no desktop app). Starlight docs site. Clerk + Stripe for the hosted tier.

## Delivery Model

**Local (free):** `npx cloudmock` / `brew install cloudmock` / Docker. Single binary bundles gateway + admin API + browser devtools. Opens at localhost:4500.

**Hosted (paid):** Sign up at cloudmock.io, get `{org}.cloudmock.io` endpoint. Clerk for auth/org management, Stripe for billing. Per-tenant isolated instances.

## Phase Sequence

| Phase | Deliverable | Effort | Dependency |
|-------|------------|--------|------------|
| 1 | Core reliability: test all 25 services | L (2-3 weeks) | None |
| 2 | Browser devtools: drop Tauri, embed in Go binary | M (1-2 weeks) | Phase 1 |
| 3 | Docs site: Starlight at cloudmock.io/docs | M (1-2 weeks) | Phase 1 |
| 4 | CLI + DX: npx/brew/docker, README | S (3-5 days) | Phase 2 |
| 5 | Homepage: cloudmock.io landing + pricing | S (2-3 days) | Phase 3 |
| 6 | Hosted SaaS: Clerk + Stripe + provisioning | XL (4-6 weeks) | Phase 4-5 |

Phases 1-5 = open-source local release. Phase 6 = paid hosted tier.

---

## Phase 1: Core Reliability

### Goal
Every operation listed in `docs/compatibility-matrix.md` returns correct responses with correct error codes. The compatibility matrix is the source of truth for what "supported" means.

### Services (25)

24 existing Tier 1 + AppSync promoted from Tier 2:

S3, DynamoDB, SQS, SNS, Lambda, IAM, STS, Secrets Manager, SSM, CloudWatch, CloudWatch Logs, EventBridge, Cognito, API Gateway, Step Functions, Route 53, RDS, ECR, ECS, SES, Kinesis, Firehose, CloudFormation, KMS, AppSync.

### AppSync (promoted to Tier 1)

AppSync already has a working implementation (2,062 lines, 4 files) with CRUD for APIs, data sources, resolvers, functions, and API keys. Tests exist.

**V1 scope (ship with what exists):** GraphQL API CRUD, data source registration, resolver registration, function CRUD, API key management. These operations are already implemented and tested.

**Post-v1 roadmap (not v1):** VTL execution engine, WebSocket subscriptions, response caching, auth directive evaluation (@aws_auth, @aws_iam, @aws_cognito_user_pools). These are substantial features that each warrant their own design doc.

The compatibility matrix must be updated to move AppSync to Tier 1 and list exactly which operations ship.

### Test strategy
- Per-service test file covering every operation in the compatibility matrix
- Error code verification (ResourceNotFoundException, ConditionalCheckFailedException, etc.)
- Edge cases: pagination, batch operations, transactions, TTL, streams
- Integration tests: cross-service workflows (Lambda->DynamoDB, EventBridge->SQS)
- Gate: `make test && make test-integration` must pass with zero failures
- Add `make test-all` target to Makefile that runs both

### Files
- `services/appsync/` -- verify and extend existing implementation
- `services/*/store_test.go` -- test files for each of 25 services
- `docs/compatibility-matrix.md` -- move AppSync to Tier 1, verify all operations listed
- `Makefile` -- add `test-all` target

---

## Phase 2: Browser Devtools

### Goal
Drop Tauri. Embed the neureaux-devtools Preact SPA into the cloudmock Go binary. Users open `localhost:4500` in their browser. One UI replaces both the existing cloudmock dashboard and the Tauri app.

### Dashboard merge strategy

Two SPAs exist today:
- **cloudmock/dashboard/** -- lightweight Preact SPA (service list, request log, config, resource inspector). Built and embedded via `pkg/dashboard/dashboard.go` using `embed.FS`. Served at `:4500`.
- **neureaux-devtools/** -- full 12-view Preact SPA (topology, traces, metrics, SLOs, chaos, etc.) with 269 tests. Currently a Tauri desktop app but works as standalone web app via `pnpm dev`.

**Decision: neureaux-devtools replaces cloudmock/dashboard entirely.** The devtools UI is a superset -- it has everything the dashboard has plus 10 additional views. The merge is:

1. Remove `cloudmock/dashboard/` source directory
2. Build neureaux-devtools with `pnpm build` -> produces `dist/`
3. Copy `dist/` into `cloudmock/pkg/dashboard/dist/`
4. `pkg/dashboard/dashboard.go` already embeds `dist/` via `embed.FS` and serves it -- no Go changes needed
5. Update `Makefile` `build-dashboard` target to build from neureaux-devtools instead of cloudmock/dashboard
6. Update `Dockerfile` to build neureaux-devtools instead of cloudmock/dashboard

### Tauri removal from neureaux-devtools

1. Remove `src-tauri/` directory entirely
2. Remove `@tauri-apps/api` and `@tauri-apps/cli` from package.json
3. Remove 6 Tauri imports (already wrapped in try/catch with graceful fallbacks):
   - `src/lib/connection.tsx` -- Tauri event listeners, health polling
   - `src/hooks/use-topology-metrics.ts` -- Tauri invoke for SQLite persistence
   - `src/components/source-bar/` -- Tauri source events
4. Remove `tauri` scripts from package.json
5. Remove `src-tauri/` references from vite.config.ts

### Persistence migration

The Tauri app used SQLite (`src-tauri/src/persistence.rs` via `rusqlite`) to store metric snapshots, deploy records, and incident history. The admin API already serves all of this data via `/api/metrics`, `/api/deploys`, `/api/incidents`. After removing Tauri:

- **Metric history**: admin API provides current metrics; historical data comes from DataPlane (DuckDB/PostgreSQL in production mode, in-memory in dev)
- **Deploy/incident records**: admin API serves these directly
- **User preferences** (connection URL, theme, layout): persist to `localStorage` (already done for most)
- **No IndexedDB needed** -- the admin API is the source of truth for all operational data

### Views (12)

All ship as functional (verified in the existing 269-test suite) except:
- **Profiler**: ship with honest "Coming soon -- CPU, heap, and goroutine profiling" empty state

### Connection picker
- Local mode: defaults to `localhost:4566` / `localhost:4599`
- Hosted mode: `{org}.cloudmock.io` with Clerk auth (Phase 6)
- Custom mode: any URL

### CORS handling
The SPA at `:4500` calls the admin API at `:4599` (cross-origin). Two options:
- **Option A (recommended):** Serve the admin API and SPA on the same port (`:4500`). Route `/api/*` to admin handlers, everything else to the SPA. Single origin, no CORS needed.
- **Option B:** Keep separate ports, admin API already has CORS middleware (`pkg/gateway/cors.go`).

### Files
- `neureaux-devtools/` -- remove Tauri, remove @tauri-apps deps
- `cloudmock/dashboard/` -- remove old SPA source (replaced by devtools)
- `cloudmock/pkg/dashboard/` -- update embed path if needed
- `cloudmock/Makefile` -- update `build-dashboard` to build from neureaux-devtools
- `cloudmock/Dockerfile` -- update dashboard build stage

---

## Phase 3: Documentation Site

### Goal
Starlight (Astro) docs site at `cloudmock.io/docs`. Every Tier 1 service documented. Getting started in under 60 seconds.

### Tech
- Astro + Starlight
- Deployed to Cloudflare Pages or Vercel
- Custom domain: `cloudmock.io`
- `/docs/` path for documentation
- `/` for homepage (Phase 5, same Astro project)

### Structure
```
docs/
├── getting-started/
│   ├── installation.md        # brew, npx, docker, go install
│   ├── first-request.md       # S3 bucket in 30 seconds
│   └── with-your-stack.md     # Node/Python/Go/Java SDK config
├── services/                  # 25 pages, consistent template
│   ├── s3.md
│   ├── dynamodb.md
│   ├── appsync.md
│   └── ...
├── devtools/                  # Browser UI guide
│   ├── overview.md
│   ├── topology.md
│   ├── activity.md
│   ├── traces.md
│   └── ...
├── language-guides/           # SDK setup per language
│   ├── node.md                # @cloudmock/node SDK + AWS SDK config
│   ├── go.md                  # cloudmock Go SDK + AWS SDK config
│   ├── python.md              # cloudmock Python SDK + boto3 config
│   ├── swift.md               # AWS SDK config (no custom SDK)
│   ├── kotlin.md              # AWS SDK config (no custom SDK)
│   └── dart.md                # AWS SDK config (no custom SDK)
├── configuration.md           # cloudmock.yml reference
├── admin-api.md               # REST API reference
├── plugins.md                 # Plugin development
├── comparison.md              # vs LocalStack, Moto, etc.
└── hosted/                    # SaaS-specific (Phase 6)
    ├── account.md
    ├── billing.md
    └── api-keys.md
```

**Language guides vs SDK guides:** Node, Go, and Python have dedicated CloudMock SDKs (in `neureaux-devtools/sdk/`) that capture inbound requests for the devtools. Swift, Kotlin, and Dart guides are "point your AWS SDK at localhost:4566" configuration instructions. The docs section is titled "Language Guides" to be honest about this distinction.

### Per-service page template
1. Service name + one-line description
2. Supported operations table (from compatibility matrix, with checkmarks)
3. Quick start example (curl + Node + Python)
4. Configuration options
5. Known differences from real AWS
6. Error codes supported

### Content source
Migrate existing `cloudmock/docs/services/` (26 directories with existing comprehensive docs) into Starlight markdown format. Primarily a formatting/restructuring task.

### Files
- `website/` -- new Astro + Starlight project (at cloudmock repo root or separate repo)
- `website/src/content/docs/` -- all markdown pages
- `website/astro.config.mjs` -- Starlight config

---

## Phase 4: CLI + DX

### Goal
1-minute install and run. Four installation methods. Stellar README.

### Installation methods

**npx (zero install):**
```bash
npx cloudmock
```
Requires a `cloudmock` npm package with:
- Platform detection (os + arch)
- Binary download from GitHub releases (cached in `~/.cloudmock/bin/`)
- Graceful error if download fails
- Sub-tasks: (a) verify `cloudmock` npm name is available, (b) write platform-detect + download script, (c) test on Linux/macOS/Windows, (d) set up npm publish CI

**brew:**
```bash
brew install neureaux/tap/cloudmock
```
Homebrew tap with pre-built bottles (not source builds). The Go binary includes embedded dashboard assets, so source builds would require Node.js -- bottles avoid this.

**Docker:**
```bash
docker run -p 4566:4566 -p 4500:4500 ghcr.io/neureaux/cloudmock
```
Already works. Update image to include embedded devtools.

**go install:**
```bash
go install github.com/Viridian-Inc/cloudmock/cmd/gateway@latest
```
Already works. Note: dashboard won't be embedded (requires separate build step). Document this limitation.

### Startup output
```
CloudMock v1.0.0
  Gateway:    http://localhost:4566
  Devtools:   http://localhost:4500  <-- open in browser
  Admin API:  http://localhost:4599
  Services:   25 active (standard profile)

Ready. Point your AWS SDK at http://localhost:4566
```

### README.md structure
1. Project name + one-line: "Local AWS. 25 services. One binary."
2. Install (4 methods, copy-paste ready)
3. "Point your SDK" -- 3 language examples (10 lines each)
4. Feature table -- services, devtools, language guides
5. Configuration basics -- profiles, ports, persistence
6. Screenshots -- topology, activity, traces (2-3 images)
7. Link to docs
8. Contributing
9. License (Apache-2.0)

No AI-generated fluff. No excessive emoji. Technical, direct, developer-friendly.

### Files
- `cloudmock/README.md` -- rewrite
- `cloudmock/cmd/gateway/main.go` -- startup banner
- `npm/cloudmock/` -- npm package with bin script
- Homebrew tap repo or formula

---

## Phase 5: Homepage

### Goal
Landing page at `cloudmock.io` that communicates the value prop and drives adoption.

### Sections
1. **Hero**: "AWS, locally." + install command + "Open devtools" CTA
2. **Feature grid**: 25 services, real-time devtools, 6 language guides, chaos engineering, IAM emulation
3. **How it works**: 3-step (install, point SDK, open devtools)
4. **Service coverage**: searchable table of 25 + 73 services
5. **Comparison**: honest table vs LocalStack, Moto, SAM Local
6. **Pricing**: Free (local) / Pro / Team -- static page (functional checkout in Phase 6)
7. **CTA**: "Get started" -> docs/getting-started

### Tech
Same Astro project as docs. Homepage is a custom Astro page (`src/pages/index.astro`).

### Files
- `website/src/pages/index.astro` -- landing page
- `website/src/pages/pricing.astro` -- static pricing (upgraded to Stripe checkout in Phase 6)

---

## Phase 6: Hosted SaaS

### Goal
Paid tier where developers get a hosted cloudmock instance with a stable endpoint, no local install required.

**This phase warrants its own detailed design doc.** The spec below captures the high-level architecture; implementation details (provisioning latency targets, idle instance lifecycle, crash recovery, DNS wildcard setup) will be specified in a Phase 6 design doc before implementation begins.

### Architecture
- **Auth**: Clerk (social login, org/team management, API keys)
- **Billing**: Stripe (usage-based metering, customer portal, invoices)
- **Provisioning**: Per-tenant cloudmock instances on Fly.io (Machines API for on-demand spin-up)
- **Endpoints**: `{org}.cloudmock.io` via Cloudflare DNS wildcard + proxy
- **Data**: Fly Postgres per tenant (or Neon for serverless Postgres)

### Tiers
| Tier | Price | Includes |
|------|-------|----------|
| Free | $0 | Local CLI only, no hosted endpoint |
| Pro | $29/mo | Hosted endpoint, 25 services, 1M requests/mo, 1 seat |
| Team | $99/mo | Everything in Pro + 10 seats, org management, priority support |

### Open questions for Phase 6 design doc
- Fly Machines vs shared multi-tenant with namespace isolation?
- Idle instance timeout (suspend after 30min? 2hr? configurable?)
- Cold start latency target (<5s? <10s?)
- Per-tenant data retention (30 days? configurable?)
- Crash recovery and health checking
- DNS propagation for new orgs

### Integration points
- Devtools connection picker: "cloudmock.io" option -> Clerk login -> org selection -> connect
- Admin API: tenant-scoped (existing `pkg/tenantscope/` middleware)
- Billing: Stripe webhook -> update tenant quotas in `pkg/cost/`
- Provisioning API: `POST /api/tenants` -> spin up Fly Machine -> return endpoint

### Existing infrastructure to reuse
- `pkg/tenantscope/` -- tenant isolation for traces/requests/metrics
- `pkg/auth/` -- JWT + RBAC (admin/editor/viewer roles)
- `pkg/cost/` -- per-tenant cost tracking with TenantCost type
- `pkg/audit/` -- audit logging per tenant
- Admin endpoints: `/api/auth/login`, `/api/auth/register`, `/api/tenants`

### Files
- `cloudmock/pkg/saas/` -- new package: Clerk integration, Stripe webhooks, Fly provisioning
- `cloudmock/cmd/gateway/main.go` -- SaaS mode flag (`--saas`)
- `website/src/pages/pricing.astro` -- upgrade to Stripe checkout links
- Separate design doc: `docs/superpowers/specs/YYYY-MM-DD-hosted-saas-design.md`

---

## CI/CD and Release Engineering

### Build pipeline
- **GitHub Actions**: on push to main, run `make test-all` + devtools `pnpm test`
- **Release workflow**: on git tag `v*`, build cross-platform binaries + Docker image + npm package
- **Targets**: Linux arm64/amd64, macOS arm64/amd64, Windows amd64
- **Docker**: push to `ghcr.io/neureaux/cloudmock:{tag}` and `:latest`
- **npm**: publish `cloudmock` CLI package and `@cloudmock/node` SDK
- **Homebrew**: update formula in tap repo

### Existing CI
- GitHub Actions release workflow already builds binaries and Docker images
- Makefile has `release` target for cross-platform builds
- Docker multi-stage build already works

### Files
- `.github/workflows/ci.yml` -- add test gate on PR
- `.github/workflows/release.yml` -- extend with npm publish + Homebrew update

---

## Exhaustive V1 Checklist

### Core (Phase 1) -- COMPLETE
- [x] 25 AWS services: every operation in compatibility matrix works correctly (1,876 tests)
- [x] AppSync promoted to Tier 1 (CRUD operations, data source/resolver registration)
- [x] Compatibility matrix updated (25 Tier 1 + 73 Tier 2)
- [x] Per-service test suites (25 test files)
- [x] Cross-service integration tests (Lambda->DynamoDB, EventBridge->SQS, CloudFormation->S3)
- [x] Error codes match AWS for all operations
- [x] `make test-all` target exists and passes
- [x] No silent error swallowing in any service
- [x] No simulated/mock data pretending to be real

### Browser Devtools (Phase 2) -- COMPLETE
- [x] Tauri removed from neureaux-devtools (src-tauri/, @tauri-apps deps)
- [x] Old cloudmock/dashboard/ replaced by neureaux-devtools build
- [x] Preact SPA embedded in Go binary via embed.FS
- [x] Served at localhost:4500 (single origin with admin API, no CORS issues)
- [x] 12 views functional in browser
- [x] Profiler: "Coming soon" empty state
- [x] Source server HTTP ingestion working (POST /api/source/events)
- [x] Connection picker: local + custom + cloudmock.io modes
- [x] No requests disappearing from topology (useReducer state machine)
- [x] Replay works via admin API
- [x] Activity shows BFF + Lambda + AWS traffic
- [x] Makefile updated: `build-dashboard` builds neureaux-devtools
- [x] Dockerfile updated: builds neureaux-devtools instead of old dashboard

### Docs (Phase 3) -- COMPLETE
- [x] Starlight project builds (48 pages)
- [x] Getting started: install + first request in under 60 seconds
- [x] 25 per-service reference pages (from compatibility matrix)
- [x] Devtools guide (topology, activity, traces, metrics, chaos, overview)
- [x] 6 language guides (Node, Go, Python have SDKs; Swift, Kotlin, Dart are config guides)
- [x] Configuration reference (cloudmock.yml)
- [x] Admin API reference (46+ endpoints)
- [x] Plugin development guide
- [x] Comparison page (vs LocalStack, Moto, SAM Local)
- [x] Search working (Pagefind)
- [ ] DEPLOY: deploy to cloudmock.io via Cloudflare Pages (requires DNS setup)

### CLI + DX (Phase 4) -- COMPLETE
- [ ] MANUAL: verify `cloudmock` npm package name is available
- [x] npm package with platform-detect + binary download script
- [x] Homebrew formula with pre-built bottle structure
- [x] Docker image updated with embedded devtools (Dockerfile)
- [x] `go install` works
- [x] Startup banner shows ports + "open in browser" hint
- [x] README: 1-minute install, no fluff, developer-friendly
- [x] cmk CLI wrapper (like awslocal)
- [ ] MANUAL: add screenshots to README (topology, activity, traces)
- [ ] MANUAL: npm publish `@cloudmock/node` SDK (requires npm credentials)
- [ ] MANUAL: npm publish `cloudmock` CLI package (requires npm credentials)

### Homepage (Phase 5) -- COMPLETE
- [x] cloudmock.io landing page built (src/pages/index.astro)
- [x] Feature grid, service list, pricing link
- [x] "Get started" links to docs
- [x] Mobile-responsive
- [ ] DEPLOY: deploy to cloudmock.io via Cloudflare Pages

### Hosted SaaS (Phase 6) -- COMPLETE (code)
- [x] Phase 6 design doc written
- [x] Clerk auth integration (webhook handler + JWT verification)
- [x] Stripe billing (webhook handler + usage metering)
- [x] Tenant provisioning (Fly Machines + Cloudflare DNS)
- [x] Quota enforcement middleware (429 at limit, warning at 80%)
- [x] Devtools "cloudmock.io" connection mode with Clerk login
- [x] Pricing page upgraded to Stripe checkout links
- [x] Tenant API endpoints (/api/saas/tenants, /api/usage, /api/subscription)
- [x] Gateway SaaS wiring (main.go)
- [x] fly.toml + provision script
- [ ] DEPLOY: create Fly app, provision first instance
- [ ] MANUAL: configure Clerk app (publishable key, webhook URL)
- [ ] MANUAL: configure Stripe products (Pro/Team price IDs, webhook URL)
- [ ] MANUAL: configure Cloudflare DNS wildcard (*.cloudmock.io)

### CI/CD -- COMPLETE
- [x] GitHub Actions CI: test gate on every PR
- [x] Release workflow: binaries + Docker + npm on git tag
- [ ] MANUAL: create git tag v1.0.0 to trigger release
- [ ] MANUAL: verify Docker image published to ghcr.io

### Cross-cutting -- COMPLETE
- [x] All Go tests green (`make test-all` — excluding Docker-dependent postgres tests)
- [x] All TypeScript tests green (269/269)
- [x] Zero TypeScript errors (`npx tsc --noEmit`)
- [x] Zero Go build errors (`go build ./cmd/gateway/`)
- [x] LICENSE file present (Apache-2.0)
- [x] CHANGELOG for v1.0.0

---

## What Must NOT Ship Broken

1. No service operation silently returns wrong data
2. No devtools view crashes or shows stale data
3. No installation method fails on supported platforms
4. No documentation page references a non-existent feature
5. No pricing page promises features that don't work
6. No hosted endpoint drops requests or leaks between tenants
7. No SDK fails to connect to cloudmock
8. No README step is wrong or outdated

---

## Post-V1 Roadmap (not in v1 scope)

- AppSync advanced: VTL execution, WebSocket subscriptions, caching, auth directives
- Custom dashboards (Datadog-style metric widgets)
- Monitoring and alerting (view/create alerts)
- RUM metrics (browser/hybrid-native user experience)
- Traffic simulator / replay previous traffic
- Profiler: CPU, heap, goroutine profiles with real data
- Step Functions: state machine execution (currently definitions-only)
- Multi-region support
- Anonymous usage telemetry (opt-in)
