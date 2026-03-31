# Phase 6: Hosted SaaS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Paid hosted tier at cloudmock.io. Clerk auth, Stripe billing, per-tenant Fly Machine provisioning. Users sign up, subscribe, get `{org}.cloudmock.io` endpoint.

**Architecture:** New `pkg/saas/` package with clerk/, stripe/, provisioning/ sub-packages. PostgreSQL tables for tenants and usage. Fly Machines API for per-tenant instances. Cloudflare DNS for wildcard routing.

**Tech Stack:** Go, Clerk SDK, Stripe Go SDK, Fly Machines API, PostgreSQL, Cloudflare API

**Spec:** `docs/superpowers/specs/2026-03-31-phase6-hosted-saas-design.md`

---

## Task 1: Add SaaS config structs

**Files:**
- Modify: `pkg/config/config.go`

- [ ] **Step 1: Read current config.go**
- [ ] **Step 2: Add SaaS config structs** (ClerkConfig, StripeConfig, ProvisioningConfig, CloudflareConfig) under a `SaaS` field in Config
- [ ] **Step 3: Add env var overrides** (CLOUDMOCK_SAAS_ENABLED, CLERK_SECRET_KEY, STRIPE_SECRET_KEY, etc.)
- [ ] **Step 4: Build:** `go build ./cmd/gateway/`
- [ ] **Step 5: Commit**

---

## Task 2: Create tenants database schema

**Files:**
- Create: `docker/init/postgres/10-saas-schema.sql`

- [ ] **Step 1: Write SQL** for `tenants` and `usage_records` tables (from design doc)
- [ ] **Step 2: Commit**

---

## Task 3: Create tenant store

**Files:**
- Create: `pkg/saas/tenant/types.go`
- Create: `pkg/saas/tenant/store.go`
- Create: `pkg/saas/tenant/postgres.go`

- [ ] **Step 1: Define Tenant and UsageRecord types**
- [ ] **Step 2: Define TenantStore interface** (Create, Get, GetBySlug, List, Update, Delete, RecordUsage)
- [ ] **Step 3: Implement PostgreSQL store**
- [ ] **Step 4: Write tests:** `pkg/saas/tenant/store_test.go`
- [ ] **Step 5: Build and commit**

---

## Task 4: Clerk webhook handler

**Files:**
- Create: `pkg/saas/clerk/webhook.go`
- Create: `pkg/saas/clerk/jwt.go`

- [ ] **Step 1: Implement webhook handler** for `organization.created`, `organization.deleted`, `user.created`
- [ ] **Step 2: Implement Clerk JWT verification** using JWKS endpoint
- [ ] **Step 3: Wire into admin API:** `POST /api/webhooks/clerk`
- [ ] **Step 4: Write tests**
- [ ] **Step 5: Build and commit**

---

## Task 5: Stripe webhook handler

**Files:**
- Create: `pkg/saas/stripe/webhook.go`
- Create: `pkg/saas/stripe/metering.go`

- [ ] **Step 1: Implement webhook handler** for `checkout.session.completed`, `invoice.paid`, `customer.subscription.updated`, `customer.subscription.deleted`
- [ ] **Step 2: Implement usage metering** — hourly export of per-tenant request counts to Stripe
- [ ] **Step 3: Wire into admin API:** `POST /api/webhooks/stripe`
- [ ] **Step 4: Write tests**
- [ ] **Step 5: Build and commit**

---

## Task 6: Fly Machines provisioning

**Files:**
- Create: `pkg/saas/provisioning/fly.go`
- Create: `pkg/saas/provisioning/dns.go`

- [x] **Step 1: Implement Fly Machines API client** (create, start, stop, destroy machine)
- [x] **Step 2: Implement Cloudflare DNS client** (add/remove CNAME record for `{slug}.cloudmock.io`)
- [x] **Step 3: Implement provisioning orchestrator** — called by Stripe webhook after payment, creates machine + DNS
- [ ] **Step 4: Write tests**
- [x] **Step 5: Build and commit**

---

## Task 7: Tenant API endpoints

**Files:**
- Modify: `pkg/admin/api.go`

- [ ] **Step 1: Add endpoints:** GET/POST /api/tenants, GET/DELETE /api/tenants/{id}, GET /api/usage, GET /api/subscription
- [ ] **Step 2: Wire tenant store and provisioning into admin API**
- [ ] **Step 3: Write tests**
- [ ] **Step 4: Build and commit**

---

## Task 8: Quota enforcement middleware

**Files:**
- Create: `pkg/saas/quota/middleware.go`

- [ ] **Step 1: Implement middleware** that checks tenant request_count vs request_limit
- [ ] **Step 2: Add warning header at 80%, return 429 at 100%**
- [ ] **Step 3: Wire into gateway** when SaaS mode is enabled
- [ ] **Step 4: Write tests**
- [ ] **Step 5: Build and commit**

---

## Task 9: Wire SaaS mode into gateway main.go

**Files:**
- Modify: `cmd/gateway/main.go`

- [ ] **Step 1: Add `--saas` flag** or read from config `saas.enabled`
- [ ] **Step 2: When SaaS enabled:** initialize Clerk JWT verifier, Stripe client, provisioning orchestrator, quota middleware
- [ ] **Step 3: Register webhook routes** (/api/webhooks/clerk, /api/webhooks/stripe)
- [ ] **Step 4: Build and commit**

---

## Task 10: Update devtools connection picker for cloudmock.io

**Files:**
- Modify: `neureaux-devtools/src/components/connection-picker/`

- [ ] **Step 1: Add "cloudmock.io" connection option**
- [ ] **Step 2: When selected:** redirect to Clerk login, then connect to `{org}.cloudmock.io`
- [ ] **Step 3: Store connection in localStorage**
- [ ] **Step 4: Build and test frontend**
- [ ] **Step 5: Commit**

---

## Task 11: Upgrade pricing page to functional Stripe checkout

**Files:**
- Modify: `website/src/pages/pricing.astro`

- [ ] **Step 1: Replace "Coming Soon" buttons** with Stripe Checkout links for Pro and Team tiers
- [ ] **Step 2: Use Stripe's hosted checkout** (redirect to checkout.stripe.com)
- [ ] **Step 3: Build and commit**

---

## Task 12: Create fly.toml and deploy config

**Files:**
- Create: `fly.toml`
- Create: `scripts/provision-tenant.sh`

- [ ] **Step 1: Write fly.toml** for the SaaS control plane instance
- [ ] **Step 2: Write provisioning script** for manual tenant creation (useful for testing)
- [ ] **Step 3: Commit**

---

## Task 13: End-to-end verification

- [ ] **Step 1: Build:** `go build ./cmd/gateway/`
- [ ] **Step 2: Run tests:** `make test-all`
- [ ] **Step 3: Verify webhook handlers accept test payloads**
- [ ] **Step 4: Verify tenant CRUD via API**
- [ ] **Step 5: Verify quota enforcement returns 429**
- [ ] **Step 6: Commit**

---

## Verification

1. `go build ./cmd/gateway/` — compiles with SaaS packages
2. `make test-all` — all tests pass
3. Clerk webhook creates tenant record
4. Stripe webhook updates subscription status
5. Provisioning creates Fly Machine + DNS record
6. Quota middleware returns 429 at limit
7. Devtools connects to hosted instance via Clerk login
8. Pricing page redirects to Stripe checkout
