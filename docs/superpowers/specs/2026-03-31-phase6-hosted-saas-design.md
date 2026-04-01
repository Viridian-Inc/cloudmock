# Phase 6: Hosted SaaS Design

## Context

Phases 1-5 shipped CloudMock as a free local tool: CLI, browser devtools, docs, homepage. Phase 6 adds a paid hosted tier at cloudmock.io where developers get a persistent endpoint (`{org}.cloudmock.io`) without installing anything locally.

## Architecture

**Auth**: Clerk handles signup, login, org/team management, API keys. CloudMock's existing JWT system validates Clerk-issued tokens.

**Billing**: Stripe handles subscriptions, usage metering, invoices, customer portal. Usage data comes from CloudMock's existing `pkg/cost/` engine.

**Provisioning**: Each org gets an isolated CloudMock instance as a Fly Machine. Instances auto-suspend after 30 min idle, auto-wake on request. DNS via Cloudflare wildcard `*.cloudmock.io`.

**Data**: Fly Postgres per tenant. 30-day retention default.

```
User browser
  → cloudmock.io (homepage/docs on Cloudflare Pages)
  → {org}.cloudmock.io (Cloudflare DNS wildcard → Fly proxy)
    → Fly Machine (per-tenant CloudMock instance)
      → Fly Postgres (per-tenant database)

Clerk webhook → /api/webhooks/clerk → create/delete tenant
Stripe webhook → /api/webhooks/stripe → update subscription status
```

## Existing Infrastructure to Reuse

| Component | Package | What it does | Phase 6 use |
|-----------|---------|-------------|-------------|
| Tenant isolation | `pkg/tenantscope/` | Scopes traces/requests by tenant_id | Inject Clerk org ID as tenant_id |
| JWT auth + RBAC | `pkg/auth/` | User/Claims/Roles, JWT middleware | Validate Clerk-issued JWTs |
| User store | `pkg/auth/postgres/` | PostgreSQL user table | Store user→tenant mapping |
| Cost engine | `pkg/cost/` | Per-tenant request/cost aggregation | Feed Stripe usage metering |
| Webhook dispatch | `pkg/webhook/` | Outbound HTTP with formatters | Stripe billing events |
| Config system | `pkg/config/` | YAML + env var config | Add Clerk/Stripe/Fly config |
| Docker image | `Dockerfile` | Multi-stage Go + SPA build | Deploy to Fly Machines |
| Health endpoint | `/api/health` | Liveness check | Fly Machine health probe |

## Tiers

| | Free | Pro | Team |
|---|---|---|---|
| Price | $0 | $29/mo | $99/mo |
| Hosted endpoint | No | `{org}.cloudmock.io` | `{org}.cloudmock.io` |
| Services | 25 (local) | 25 | 25 |
| Requests/mo | Unlimited (local) | 1M | 10M |
| Seats | 1 | 1 | 10 |
| Data retention | N/A | 7 days | 30 days |
| Support | Community | Email | Priority |

## New Packages

### `pkg/saas/clerk/`
- Clerk webhook handler (user.created, org.created, org.deleted)
- JWT verification using Clerk's JWKS endpoint
- Sync Clerk users → CloudMock user store
- Extract org_id from Clerk session → inject as tenant_id

### `pkg/saas/stripe/`
- Stripe webhook handler (checkout.session.completed, invoice.paid, customer.subscription.updated/deleted)
- Usage metering: POST usage records to Stripe every hour
- Subscription status cache (active/past_due/canceled)
- Quota enforcement middleware (429 when exceeded)

### `pkg/saas/provisioning/`
- Fly Machines API client (create, start, stop, destroy)
- Tenant→Machine mapping (PostgreSQL table)
- Auto-suspend after idle timeout (30 min default)
- Auto-wake on first request (Fly proxy handles this)
- DNS: Cloudflare API for `{org}.cloudmock.io` CNAME records

## Config Additions

```yaml
saas:
  enabled: false  # toggle SaaS mode
  clerk:
    secret_key: ${CLERK_SECRET_KEY}
    webhook_secret: ${CLERK_WEBHOOK_SECRET}
  stripe:
    secret_key: ${STRIPE_SECRET_KEY}
    webhook_secret: ${STRIPE_WEBHOOK_SECRET}
    pro_price_id: "price_xxx"
    team_price_id: "price_yyy"
  provisioning:
    fly_api_token: ${FLY_API_TOKEN}
    fly_org: "cloudmock-saas"
    fly_region: "iad"
    image: "ghcr.io/neureaux/cloudmock:latest"
    idle_timeout_minutes: 30
    data_retention_days: 30
  cloudflare:
    api_token: ${CLOUDFLARE_API_TOKEN}
    zone_id: ${CLOUDFLARE_ZONE_ID}
```

## New API Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| POST | `/api/webhooks/clerk` | Clerk signature | User/org lifecycle events |
| POST | `/api/webhooks/stripe` | Stripe signature | Billing events |
| GET | `/api/tenants` | Admin | List all tenants |
| POST | `/api/tenants` | Admin | Provision new tenant |
| GET | `/api/tenants/{id}` | Owner/Admin | Tenant details + usage |
| DELETE | `/api/tenants/{id}` | Admin | Deprovision tenant |
| GET | `/api/usage` | Owner | Current billing period usage |
| GET | `/api/subscription` | Owner | Subscription status |

## Database Schema (new tables)

```sql
CREATE TABLE tenants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clerk_org_id    TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    slug            TEXT UNIQUE NOT NULL,  -- used in {slug}.cloudmock.io
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    tier            TEXT NOT NULL DEFAULT 'free',  -- free/pro/team
    status          TEXT NOT NULL DEFAULT 'active', -- active/suspended/canceled
    fly_machine_id  TEXT,
    fly_app_name    TEXT,
    request_count   BIGINT DEFAULT 0,
    request_limit   BIGINT DEFAULT 0,  -- 0 = unlimited
    data_retention_days INT DEFAULT 30,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE usage_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id),
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    request_count BIGINT NOT NULL,
    total_cost   NUMERIC(10,6),
    reported_to_stripe BOOLEAN DEFAULT false,
    created_at   TIMESTAMPTZ DEFAULT now()
);
```

## Provisioning Flow

```
1. User signs up on cloudmock.io → Clerk handles auth
2. User creates org in Clerk → webhook fires
3. POST /api/webhooks/clerk { type: "organization.created", data: { id, name, slug } }
4. Handler creates tenant record in PostgreSQL
5. User selects Pro/Team → Stripe Checkout
6. POST /api/webhooks/stripe { type: "checkout.session.completed" }
7. Handler updates tenant tier, calls provisioning:
   a. Fly Machines API: create machine from ghcr.io/neureaux/cloudmock
   b. Configure env: CLOUDMOCK_AUTH_ENABLED=true, DB_URL=...
   c. Cloudflare API: add CNAME {slug}.cloudmock.io → {app}.fly.dev
8. User gets endpoint: https://{slug}.cloudmock.io
9. Point AWS SDK at https://{slug}.cloudmock.io:4566
```

## Quota Enforcement

Middleware in the gateway checks tenant's request_count vs request_limit:
- At 80%: add `X-CloudMock-Usage-Warning: approaching limit` header
- At 100%: return 429 with `X-CloudMock-Usage-Exceeded: true`
- Reset count at billing period start (Stripe subscription cycle)

## Open Questions (decide during implementation)

1. Fly Postgres vs Neon — try Fly Postgres first (simpler billing)
2. Per-tenant volumes vs shared DB with row isolation — start with isolated for security
3. Custom domain support (`aws.mycompany.dev`) — post-v1 feature
4. WebSocket support for devtools on hosted tier — verify Fly proxy supports it
5. Backup strategy — Fly Postgres has built-in daily snapshots

## Implementation Order

1. Config additions (Clerk, Stripe, provisioning structs)
2. Database schema (tenants, usage_records tables)
3. Clerk integration (webhook handler, JWT verification)
4. Stripe integration (webhook handler, metering, checkout)
5. Provisioning (Fly Machines API, DNS)
6. Quota enforcement middleware
7. Devtools connection picker → Clerk login flow
8. Pricing page → functional Stripe checkout
9. End-to-end testing

## Verification

1. Sign up via Clerk → org created → tenant record in DB
2. Subscribe via Stripe → payment succeeds → Fly Machine provisioned
3. Point SDK at `{slug}.cloudmock.io` → requests work
4. Usage metered → Stripe invoice reflects usage
5. Cancel subscription → machine deprovisioned → endpoint returns 402
6. Devtools connection picker → "cloudmock.io" → Clerk login → connects to hosted instance
