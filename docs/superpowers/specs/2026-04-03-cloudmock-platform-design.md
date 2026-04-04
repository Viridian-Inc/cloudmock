# CloudMock Platform -- SaaS Design Spec

## Overview

A separate public repo (`cloudmock-platform`) that provides a hosted, API-first cloud testing platform built on CloudMock. Customers create isolated AWS emulation environments ("apps"), access them via API keys, and pay per request. The platform is HIPAA-compliant, source-available under BSL, and designed for teams that need shared cloud testing infrastructure without managing their own CloudMock instances.

**Goal:** Let any developer `export AWS_ENDPOINT_URL=https://abc123.cloudmock.io` and start testing against 100 AWS services with zero setup, usage-based billing, and enterprise-grade compliance.

## Architecture

Two public BSL repos:
- `cloudmock` (existing) -- the Go emulator, 100 AWS services, devtools, `pkg/saas/*` shared libraries
- `cloudmock-platform` (new) -- Next.js dashboard + Go API service

### System Components

```
User's AWS SDK
    │
    ▼
Fly Edge (TLS)
    │
    ▼
Go API Service (Fly)
  ├── Auth (Clerk JWT or API key)
  ├── HIPAA audit log (append-only)
  ├── Usage metering (Stripe billing meter)
  ├── Quota check (free tier cap)
  └── Proxy ──▶ CloudMock instance
                  ├── Shared (free tier, tenant-scoped)
                  └── Dedicated (paid, isolated Fly machine)

Next.js Dashboard (Vercel)
  ├── Clerk auth (sign-up, SSO, org management)
  ├── Role-based views (Admin, Developer, Viewer)
  ├── Stripe billing (usage-based, Customer Portal)
  └── Calls Go API for all data operations
```

### Tech Stack

- **Frontend:** Next.js 15, React, Tailwind CSS, deployed on Vercel
- **Auth:** Clerk (Next.js SDK, org management, RBAC)
- **Billing:** Stripe (billing meters for usage, Customer Portal for invoices)
- **Backend API:** Go, deployed on Fly, imports `cloudmock/pkg/saas/*`
- **Database:** Postgres on Fly (encrypted at rest, RLS for tenant isolation)
- **DNS:** Cloudflare (CNAME per app endpoint)
- **CloudMock instances:** Fly Machines (shared or dedicated per tenant)

## Product Model

### API-First Cloud Testing

The product is an API, not a GUI. Customers interact primarily through their AWS SDKs pointed at a hosted endpoint. The dashboard is a management layer for creating apps, managing keys, monitoring usage, and handling compliance.

**User flow:**
1. Sign up at cloudmock.io (Clerk)
2. Create an "app" (isolated CloudMock environment)
3. Get an endpoint (`https://abc123.cloudmock.io`) and API key
4. Point AWS SDK at the endpoint
5. Use any of 100 AWS services -- they just work
6. Pay $0.50 per 10K requests after 1K free

### Multi-Tenancy (Hybrid)

- **Shared infrastructure (free tier):** One CloudMock process per Fly region handles multiple tenants. The Go API sets `X-Tenant-ID` before proxying. CloudMock's `pkg/tenantscope` isolates data. Cheap to operate -- one machine serves thousands of free users.
- **Dedicated infrastructure (paid option):** Go API provisions a dedicated Fly machine via `pkg/saas/provisioning`. DNS routes `{slug}.cloudmock.io` to the machine. Full resource isolation. The Go API still fronts all requests for auth, metering, and audit.

### Pricing

Usage-based, one model:
- **Free:** 1K requests/month, no credit card required, hard cap
- **Pay-as-you-go:** $0.50 per 10K requests after 1K free, credit card required

No tiers or subscriptions. Stripe billing meter aggregates usage hourly and charges at end of billing cycle. Dedicated infrastructure is an add-on at $7/mo per dedicated app (covers Fly machine cost), charged as a separate Stripe line item when the app is created.

**Note on existing code:** The `pkg/saas/stripe` package uses tier-based subscription webhooks (free/pro/team). The platform's Go API will adapt this to usage-based billing with Stripe billing meters instead. The Clerk webhook handler and provisioning orchestrator can be reused with minimal changes. The Stripe webhook handler needs a new implementation focused on `billing_meter.usage_reported` and `invoice.paid` events rather than subscription lifecycle events.

## Repository Structure

```
cloudmock-platform/
├── apps/
│   └── web/                        # Next.js (Vercel)
│       ├── app/
│       │   ├── (auth)/             # Clerk sign-in/sign-up
│       │   ├── (dashboard)/
│       │   │   ├── apps/           # App list + detail
│       │   │   ├── keys/           # API key management
│       │   │   ├── usage/          # Usage metrics + charts
│       │   │   ├── team/           # Member management
│       │   │   ├── billing/        # Stripe Customer Portal
│       │   │   ├── audit/          # HIPAA audit log viewer
│       │   │   ├── settings/       # Org settings, data retention
│       │   │   └── layout.tsx      # Role-based layout switcher
│       │   └── api/
│       │       ├── webhooks/clerk/
│       │       ├── webhooks/stripe/
│       │       └── v1/             # Public API proxy to Go API
│       ├── components/
│       │   ├── dashboard/          # Charts, tables, stat cards
│       │   ├── devtools/           # Embedded topology viewer
│       │   └── shared/             # Buttons, modals, layouts
│       └── lib/
│           ├── api.ts              # Go API client
│           └── clerk.ts            # Clerk helpers
├── services/
│   └── api/                        # Go API (Fly)
│       ├── cmd/api/main.go
│       ├── handlers/
│       │   ├── apps.go
│       │   ├── keys.go
│       │   ├── proxy.go
│       │   ├── audit.go
│       │   └── webhooks.go
│       ├── middleware/
│       │   ├── auth.go             # Clerk JWT verification
│       │   ├── audit.go            # HIPAA audit logging
│       │   ├── quota.go            # Usage metering + enforcement
│       │   └── tenant.go           # Tenant scoping
│       ├── store/
│       │   ├── tenants.go
│       │   ├── apps.go
│       │   ├── keys.go
│       │   ├── audit.go
│       │   └── migrations/
│       └── provisioning/
├── packages/
│   └── shared/                     # Shared types
├── docker-compose.yml              # Local dev
├── fly.toml
├── vercel.json
└── LICENSE                         # BSL 1.1
```

## Data Model

### tenants
| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| clerk_org_id | text UNIQUE | From Clerk webhook |
| name | text | |
| slug | text UNIQUE | |
| status | text | active, suspended, canceled |
| has_payment_method | bool | Distinguishes free (hard cap) from PAYG |
| stripe_customer_id | text | Nullable, set when payment method added |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### apps
| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| tenant_id | uuid FK | |
| name | text | "staging", "ci-tests" |
| slug | text | Unique per tenant |
| endpoint | text | https://abc123.cloudmock.io |
| infra_type | text | "shared" or "dedicated" |
| fly_app_name | text | Null if shared |
| fly_machine_id | text | Null if shared |
| status | text | running, stopped, provisioning |
| created_at | timestamptz | |

### api_keys
| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| tenant_id | uuid FK | |
| app_id | uuid FK | |
| key_hash | text | bcrypt, never store plaintext |
| prefix | text | "cm_live_abc" for display |
| name | text | "CI key", "local dev" |
| role | text | admin, developer, viewer |
| last_used_at | timestamptz | |
| expires_at | timestamptz | Nullable |
| revoked_at | timestamptz | Nullable |
| created_at | timestamptz | |

### usage_records
| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| tenant_id | uuid FK | |
| app_id | uuid FK | |
| period_start | timestamptz | |
| period_end | timestamptz | |
| request_count | bigint | |
| reported_to_stripe | bool | |
| created_at | timestamptz | |

### audit_log
| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| tenant_id | uuid FK | |
| actor_id | text | Clerk user ID or API key prefix |
| actor_type | text | "user" or "api_key" |
| action | text | "app.create", "key.rotate", "aws.request" |
| resource_type | text | "app", "key", "tenant", "request" |
| resource_id | text | |
| ip_address | inet | |
| user_agent | text | |
| metadata | jsonb | Action-specific details |
| created_at | timestamptz | Immutable, append-only |

### data_retention
| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| tenant_id | uuid FK | |
| resource_type | text | "audit_log", "request_log", "state_snapshot" |
| retention_days | int | 90, 365, etc. |
| updated_at | timestamptz | |

**Constraints:**
- `audit_log` is append-only: no UPDATE or DELETE, enforced at Postgres role level
- `api_keys.key_hash` stores bcrypt hashes only; plaintext shown once at creation
- All queries filter by `tenant_id`; Postgres RLS as a second isolation layer
- `data_retention` lets each org set retention policies per resource type (HIPAA requirement)

## API Design

### Public API (authenticated via API key or Clerk JWT)

```
# Apps
POST   /v1/apps                         Create app
GET    /v1/apps                         List apps (tenant-scoped)
GET    /v1/apps/:id                     Get app detail
PATCH  /v1/apps/:id                     Update app
DELETE /v1/apps/:id                     Delete app (deprovisions infra)

# API Keys
POST   /v1/apps/:id/keys                Create key (returns plaintext once)
GET    /v1/apps/:id/keys                List keys (prefix + metadata only)
DELETE /v1/apps/:id/keys/:key_id        Revoke key
POST   /v1/apps/:id/keys/:key_id/rotate Rotate key

# Usage
GET    /v1/usage                        Org-wide usage summary
GET    /v1/apps/:id/usage               Per-app usage breakdown

# Audit Log (admin only)
GET    /v1/audit                        Query audit log
GET    /v1/audit/export                 Export as CSV

# Team (wraps Clerk org API)
GET    /v1/team/members                 List members + roles
POST   /v1/team/invites                 Invite member
PATCH  /v1/team/members/:id             Change role

# Settings
GET    /v1/settings                     Org settings
PATCH  /v1/settings                     Update settings
GET    /v1/settings/retention           Data retention policies
PATCH  /v1/settings/retention           Update retention

# Data Management (HIPAA)
POST   /v1/apps/:id/snapshots           Export state snapshot
GET    /v1/apps/:id/snapshots           List snapshots
POST   /v1/apps/:id/snapshots/:sid/restore  Restore snapshot
DELETE /v1/apps/:id/data                Purge all app data

# AWS Proxy (programmatic API access)
ANY    /v1/apps/:id/aws/*               Proxy to CloudMock instance
```

### Subdomain-Based AWS Access (SDK usage)

SDK users set `AWS_ENDPOINT_URL=https://abc123.cloudmock.io` and make normal AWS SDK calls. The Go API resolves the subdomain to an app, authenticates via `X-Api-Key` header, and proxies to the correct CloudMock instance. This is the primary access pattern -- no `/v1/` prefix needed.

The `/v1/apps/:id/aws/*` path is an alternative for programmatic access (e.g., scripts that manage multiple apps).

### Webhook Endpoints (signature-verified, no auth)

```
POST   /webhooks/clerk                  Clerk org/user events (Svix sig)
POST   /webhooks/stripe                 Stripe billing events (HMAC sig)
```

### Authentication

- **Dashboard:** Clerk JWT in `Authorization: Bearer <jwt>`, verified via JWKS
- **SDK/API:** API key in `X-Api-Key: cm_live_abc123...`, bcrypt hash lookup
- **Webhooks:** Svix signatures (Clerk), HMAC-SHA256 (Stripe)
- Every authenticated request generates an audit log entry

## Request Flow

### SDK Request Path

```
aws s3 mb s3://my-bucket --endpoint https://abc123.cloudmock.io
    │
    ▼
Fly Edge (TLS termination)
    │
    ▼
Go API
  1. Extract API key from X-Api-Key header
  2. Hash and look up in api_keys table → get app_id, tenant_id, role
  3. Check quota (free tier: reject if > 1K; paid: pass through)
  4. Write audit log entry (async, non-blocking)
  5. Increment usage counter (async)
  6. Route to CloudMock:
     - Shared: set X-Tenant-ID, proxy to regional shared instance
     - Dedicated: proxy to app's Fly machine
  7. Return response to caller
```

### Shared vs Dedicated Routing

```go
switch app.InfraType {
case "shared":
    r.Header.Set("X-Tenant-ID", app.TenantID)
    sharedProxy.ServeHTTP(w, r)
case "dedicated":
    dedicatedProxy(app.FlyAppName).ServeHTTP(w, r)
}
```

## Dashboard Pages

### Role-Based Access

| Page | Admin | Developer | Viewer |
|------|-------|-----------|--------|
| Overview (usage-first) | Home | -- | -- |
| Apps (apps-first) | Yes | Home | Read-only |
| App detail + devtools | Full | Full | Read-only |
| API keys | Full | Full | Hidden |
| Usage & billing | Full | Read-only | Read-only |
| Team management | Full | Read-only | Read-only |
| Audit log | Full | Hidden | Hidden |
| Data retention | Full | Hidden | Hidden |
| Settings | Full | Hidden | Hidden |

### Admin Home (Usage-First)

Shows: monthly request count, estimated cost, active apps count, team size, 30-day usage chart, recent audit trail.

### Developer Home (Apps-First)

Shows: app cards with name, status (running/shared/dedicated), endpoint, request count, active services tags, and a "+ New App" card.

### App Detail Page

Tabs: Overview, Services, API Keys, Devtools (embedded topology viewer from CloudMock), Snapshots, Settings.

Overview tab shows: endpoint with copy button, quick-start snippet (`export AWS_ENDPOINT_URL=...`), active services with resource counts.

## HIPAA Compliance

### Technical Safeguards

1. **Encryption in transit:** TLS everywhere. Fly handles termination. Internal traffic between Go API and CloudMock instances uses Fly's private networking (WireGuard).

2. **Encryption at rest:** Postgres on Fly with encrypted volumes. API key hashes stored as bcrypt. No plaintext secrets in the database.

3. **Audit logging:** Append-only `audit_log` table. Every API call, dashboard action, and AWS proxy request is logged with: actor, action, resource, IP, user agent, timestamp. No UPDATE or DELETE on the table, enforced at the Postgres role level. Exportable as CSV for compliance audits.

4. **Access controls:** Three roles (Admin, Developer, Viewer) enforced at both the Go API and Next.js middleware layers. Clerk handles SSO and MFA. API keys are scoped to apps with role-based permissions.

5. **Tenant isolation:** All database queries filter by tenant_id. Postgres Row-Level Security as a second layer. Shared CloudMock instances use `pkg/tenantscope` for data isolation. Dedicated instances are fully isolated Fly machines.

6. **Data retention:** Configurable per-org retention policies for audit logs, request logs, and state snapshots. Automated purge job runs daily. Right-to-deletion endpoint (`DELETE /v1/apps/:id/data`) for complete data removal.

7. **BAA readiness:** Fly, Clerk, and Stripe all offer Business Associate Agreements. The platform architecture ensures PHI (if any) never leaves BAA-covered infrastructure.

### Administrative Safeguards

- Audit log export for compliance reviews
- Role-based access prevents unauthorized data access
- Data retention policies documented and configurable per org
- Incident response: audit log provides forensic trail

## External Service Configuration

### Clerk
- Create Clerk application with org support enabled
- Configure webhook endpoint: `https://api.cloudmock.io/webhooks/clerk`
- Events: `organization.created`, `organization.deleted`, `user.created`
- Enable RBAC with roles: `org:admin`, `org:developer`, `org:viewer`
- Enable SSO for enterprise customers

### Stripe
- Create Stripe billing meter named `api_requests` (event_name: `api_requests`)
- Configure webhook endpoint: `https://api.cloudmock.io/webhooks/stripe`
- Events: `invoice.paid`, `customer.subscription.updated`
- No subscription products needed -- pure usage-based via billing meters
- Customer Portal for payment method management and invoice history

### Fly
- Org: `viridian-inc`
- Shared CloudMock app per region (e.g., `cm-shared-iad` for us-east)
- Dedicated apps named `cm-{tenant-slug}`
- Image: latest CloudMock Docker image
- Machine size: shared-cpu-1x, 256MB for free/basic; performance-2x for dedicated

### Cloudflare
- Zone: `cloudmock.io`
- Proxied CNAME records: `{app-slug}.cloudmock.io` pointing to Fly apps
- SSL: Full (strict) mode

## Local Development

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: cloudmock_platform
      POSTGRES_PASSWORD: dev
    ports: ["5432:5432"]

  cloudmock:
    image: ghcr.io/viridian-inc/cloudmock:latest
    ports:
      - "4566:4566"  # AWS endpoint
      - "4500:4500"  # Devtools

  api:
    build: ./services/api
    environment:
      DATABASE_URL: postgres://postgres:dev@postgres:5432/cloudmock_platform
      CLOUDMOCK_SHARED_URL: http://cloudmock:4566
      CLERK_SECRET_KEY: ${CLERK_SECRET_KEY}
      STRIPE_API_KEY: ${STRIPE_API_KEY}
    ports: ["8080:8080"]
    depends_on: [postgres, cloudmock]
```

Next.js dev server runs locally with `npm run dev`, configured to call the Go API at `localhost:8080`.
