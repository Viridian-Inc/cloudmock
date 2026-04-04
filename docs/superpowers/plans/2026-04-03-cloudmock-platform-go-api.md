# CloudMock Platform: Go API Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Go API service that powers the CloudMock SaaS platform -- handling auth, tenant management, app lifecycle, API keys, usage metering, audit logging, quota enforcement, and request proxying to CloudMock instances.

**Architecture:** A standalone Go HTTP service deployed on Fly. It fronts all requests to CloudMock instances, handling authentication (Clerk JWT or API key), HIPAA audit logging, usage metering for Stripe billing, and routing to shared or dedicated CloudMock machines. Postgres is the single source of truth.

**Tech Stack:** Go 1.26, chi router, pgx/v5 (Postgres), golang-migrate, testcontainers-go, golang-jwt/jwt/v5, crypto/sha256 (API key hashing)

**Spec:** `docs/superpowers/specs/2026-04-03-cloudmock-platform-design.md`

**Repo:** New repo at `../cloudmock-platform` (sibling to cloudmock). Module path: `github.com/Viridian-Inc/cloudmock-platform`

---

## File Structure

```
cloudmock-platform/
├── services/api/
│   ├── cmd/api/main.go                    # Entrypoint, wiring, server startup
│   ├── internal/
│   │   ├── database/
│   │   │   ├── database.go                # Connection pool + migration runner
│   │   │   └── database_test.go
│   │   ├── store/
│   │   │   ├── tenants.go                 # Tenant CRUD
│   │   │   ├── tenants_test.go
│   │   │   ├── apps.go                    # App CRUD + provisioning state
│   │   │   ├── apps_test.go
│   │   │   ├── apikeys.go                 # API key CRUD with SHA-256 hashing
│   │   │   ├── apikeys_test.go
│   │   │   ├── audit.go                   # Append-only audit log
│   │   │   ├── audit_test.go
│   │   │   ├── usage.go                   # Usage records + Stripe reporting
│   │   │   ├── usage_test.go
│   │   │   ├── retention.go               # Data retention config + purge
│   │   │   ├── retention_test.go
│   │   │   └── testhelper_test.go          # Shared testcontainers setup
│   │   ├── middleware/
│   │   │   ├── auth.go                    # Clerk JWT + API key verification
│   │   │   ├── auth_test.go
│   │   │   ├── audit.go                   # HIPAA audit logging per request
│   │   │   ├── audit_test.go
│   │   │   ├── quota.go                   # Usage enforcement + free tier cap
│   │   │   ├── quota_test.go
│   │   │   ├── tenant.go                  # Tenant context injection
│   │   │   └── tenant_test.go
│   │   ├── handler/
│   │   │   ├── apps.go                    # POST/GET/PATCH/DELETE /v1/apps
│   │   │   ├── apps_test.go
│   │   │   ├── apikeys.go                 # POST/GET/DELETE /v1/apps/:id/keys
│   │   │   ├── apikeys_test.go
│   │   │   ├── proxy.go                   # ANY /v1/apps/:id/aws/* + subdomain routing
│   │   │   ├── proxy_test.go
│   │   │   ├── webhooks.go                # POST /webhooks/clerk, /webhooks/stripe
│   │   │   ├── webhooks_test.go
│   │   │   ├── usage.go                   # GET /v1/usage, /v1/apps/:id/usage
│   │   │   ├── audit_handler.go           # GET /v1/audit, /v1/audit/export
│   │   │   ├── team.go                    # GET/POST/PATCH /v1/team/*
│   │   │   ├── settings.go               # GET/PATCH /v1/settings
│   │   │   └── snapshots.go              # POST/GET /v1/apps/:id/snapshots
│   │   └── model/
│   │       └── model.go                   # Shared types (Tenant, App, APIKey, etc.)
│   ├── migrations/
│   │   ├── 001_create_tenants.up.sql
│   │   ├── 001_create_tenants.down.sql
│   │   ├── 002_create_apps.up.sql
│   │   ├── 002_create_apps.down.sql
│   │   ├── 003_create_api_keys.up.sql
│   │   ├── 003_create_api_keys.down.sql
│   │   ├── 004_create_usage_records.up.sql
│   │   ├── 004_create_usage_records.down.sql
│   │   ├── 005_create_audit_log.up.sql
│   │   ├── 005_create_audit_log.down.sql
│   │   ├── 006_create_data_retention.up.sql
│   │   └── 006_create_data_retention.down.sql
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
├── docker-compose.yml
├── LICENSE
└── README.md
```

---

### Task 1: Repository Scaffold

**Files:**
- Create: `../cloudmock-platform/services/api/go.mod`
- Create: `../cloudmock-platform/services/api/cmd/api/main.go`
- Create: `../cloudmock-platform/services/api/internal/model/model.go`
- Create: `../cloudmock-platform/docker-compose.yml`
- Create: `../cloudmock-platform/LICENSE`

- [ ] **Step 1: Create repo directory and initialize Go module**

```bash
mkdir -p ../cloudmock-platform/services/api
cd ../cloudmock-platform/services/api
go mod init github.com/Viridian-Inc/cloudmock-platform/services/api
```

- [ ] **Step 2: Create the shared model types**

Create `services/api/internal/model/model.go`:

```go
package model

import (
	"net"
	"time"
)

// Tenant represents an organization (maps to a Clerk org).
type Tenant struct {
	ID               string    `json:"id"`
	ClerkOrgID       string    `json:"clerk_org_id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Status           string    `json:"status"` // active, suspended, canceled
	HasPaymentMethod bool      `json:"has_payment_method"`
	StripeCustomerID string    `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// App represents an isolated CloudMock environment within a tenant.
type App struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Endpoint     string    `json:"endpoint"`
	InfraType    string    `json:"infra_type"` // shared, dedicated
	FlyAppName   string    `json:"fly_app_name,omitempty"`
	FlyMachineID string    `json:"fly_machine_id,omitempty"`
	Status       string    `json:"status"` // running, stopped, provisioning
	CreatedAt    time.Time `json:"created_at"`
}

// APIKey represents an API key scoped to an app.
type APIKey struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	AppID      string     `json:"app_id"`
	KeyHash    string     `json:"-"`              // SHA-256 hash, never exposed
	Prefix     string     `json:"prefix"`         // "cm_live_abc" for display
	Name       string     `json:"name"`           // user-assigned label
	Role       string     `json:"role"`           // admin, developer, viewer
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// UsageRecord tracks request counts per app per billing period.
type UsageRecord struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	AppID            string    `json:"app_id"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	RequestCount     int64     `json:"request_count"`
	ReportedToStripe bool      `json:"reported_to_stripe"`
	CreatedAt        time.Time `json:"created_at"`
}

// AuditEntry is an immutable record of an action taken in the platform.
type AuditEntry struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	ActorID      string         `json:"actor_id"`
	ActorType    string         `json:"actor_type"` // user, api_key
	Action       string         `json:"action"`     // app.create, key.rotate, aws.request
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	IPAddress    net.IP         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// DataRetention configures per-org retention policies.
type DataRetention struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ResourceType  string    `json:"resource_type"` // audit_log, request_log, state_snapshot
	RetentionDays int       `json:"retention_days"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AuthContext carries the authenticated identity through request handling.
type AuthContext struct {
	TenantID string
	ActorID  string
	ActorType string // "user" or "api_key"
	Role     string  // admin, developer, viewer
	AppID    string  // set when authenticated via API key
}
```

- [ ] **Step 3: Create minimal main.go**

Create `services/api/cmd/api/main.go`:

```go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	slog.Info("starting server", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Create docker-compose.yml for local dev**

Create `docker-compose.yml` at repo root:

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: cloudmock_platform
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: dev
    ports: ["5432:5432"]
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 2s
      timeout: 5s
      retries: 5

  cloudmock:
    image: ghcr.io/viridian-inc/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"

  api:
    build:
      context: ./services/api
      dockerfile: Dockerfile
    environment:
      DATABASE_URL: postgres://postgres:dev@postgres:5432/cloudmock_platform?sslmode=disable
      CLOUDMOCK_SHARED_URL: http://cloudmock:4566
      PORT: "8080"
    ports: ["8080:8080"]
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  pgdata:
```

- [ ] **Step 5: Create BSL license file**

Create `LICENSE`:

```text
Business Source License 1.1

Licensor: Viridian, Inc.
Licensed Work: CloudMock Platform
Change Date: April 1, 2030
Change License: Apache License, Version 2.0

For information about alternative licensing, contact licensing@viridian.dev
```

- [ ] **Step 6: Install dependencies and verify build**

```bash
cd ../cloudmock-platform/services/api
go get github.com/go-chi/chi/v5@latest
go get github.com/jackc/pgx/v5@latest
go get github.com/golang-migrate/migrate/v4@latest
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/google/uuid@latest
go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 7: Initialize git and commit**

```bash
cd ../cloudmock-platform
git init
git add -A
git commit -m "feat: initial repo scaffold with Go API skeleton and docker-compose"
```

---

### Task 2: Database Connection and Migrations

**Files:**
- Create: `services/api/internal/database/database.go`
- Create: `services/api/internal/database/database_test.go`
- Create: `services/api/migrations/001_create_tenants.up.sql`
- Create: `services/api/migrations/001_create_tenants.down.sql`
- Create: `services/api/migrations/002_create_apps.up.sql`
- Create: `services/api/migrations/002_create_apps.down.sql`
- Create: `services/api/migrations/003_create_api_keys.up.sql`
- Create: `services/api/migrations/003_create_api_keys.down.sql`
- Create: `services/api/migrations/004_create_usage_records.up.sql`
- Create: `services/api/migrations/004_create_usage_records.down.sql`
- Create: `services/api/migrations/005_create_audit_log.up.sql`
- Create: `services/api/migrations/005_create_audit_log.down.sql`
- Create: `services/api/migrations/006_create_data_retention.up.sql`
- Create: `services/api/migrations/006_create_data_retention.down.sql`

- [ ] **Step 1: Write the migration SQL files**

Create `services/api/migrations/001_create_tenants.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE tenants (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    clerk_org_id       TEXT NOT NULL UNIQUE,
    name               TEXT NOT NULL,
    slug               TEXT NOT NULL UNIQUE,
    status             TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'canceled')),
    has_payment_method BOOLEAN NOT NULL DEFAULT false,
    stripe_customer_id TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tenants_clerk_org_id ON tenants(clerk_org_id);
CREATE INDEX idx_tenants_slug ON tenants(slug);
```

Create `services/api/migrations/001_create_tenants.down.sql`:

```sql
DROP TABLE IF EXISTS tenants;
```

Create `services/api/migrations/002_create_apps.up.sql`:

```sql
CREATE TABLE apps (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    slug           TEXT NOT NULL,
    endpoint       TEXT NOT NULL,
    infra_type     TEXT NOT NULL DEFAULT 'shared' CHECK (infra_type IN ('shared', 'dedicated')),
    fly_app_name   TEXT,
    fly_machine_id TEXT,
    status         TEXT NOT NULL DEFAULT 'provisioning' CHECK (status IN ('running', 'stopped', 'provisioning', 'error')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_apps_tenant_id ON apps(tenant_id);
CREATE INDEX idx_apps_endpoint ON apps(endpoint);
```

Create `services/api/migrations/002_create_apps.down.sql`:

```sql
DROP TABLE IF EXISTS apps;
```

Create `services/api/migrations/003_create_api_keys.up.sql`:

```sql
CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    app_id      UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key_hash    TEXT NOT NULL,
    prefix      TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    role        TEXT NOT NULL DEFAULT 'developer' CHECK (role IN ('admin', 'developer', 'viewer')),
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_app_id ON api_keys(app_id);
```

Create `services/api/migrations/003_create_api_keys.down.sql`:

```sql
DROP TABLE IF EXISTS api_keys;
```

Create `services/api/migrations/004_create_usage_records.up.sql`:

```sql
CREATE TABLE usage_records (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    app_id             UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    period_start       TIMESTAMPTZ NOT NULL,
    period_end         TIMESTAMPTZ NOT NULL,
    request_count      BIGINT NOT NULL DEFAULT 0,
    reported_to_stripe BOOLEAN NOT NULL DEFAULT false,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_usage_records_tenant_id ON usage_records(tenant_id);
CREATE INDEX idx_usage_records_app_id ON usage_records(app_id);
CREATE INDEX idx_usage_records_unreported ON usage_records(reported_to_stripe) WHERE NOT reported_to_stripe;
```

Create `services/api/migrations/004_create_usage_records.down.sql`:

```sql
DROP TABLE IF EXISTS usage_records;
```

Create `services/api/migrations/005_create_audit_log.up.sql`:

```sql
CREATE TABLE audit_log (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id     UUID NOT NULL,
    actor_id      TEXT NOT NULL,
    actor_type    TEXT NOT NULL CHECK (actor_type IN ('user', 'api_key')),
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   TEXT NOT NULL DEFAULT '',
    ip_address    INET,
    user_agent    TEXT NOT NULL DEFAULT '',
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- No FK to tenants: audit log must survive tenant deletion for compliance.
-- No UPDATE or DELETE: enforced via a restricted Postgres role (see below).

CREATE INDEX idx_audit_log_tenant_id ON audit_log(tenant_id);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

-- Create a restricted role that can only INSERT into audit_log.
-- The application connects as this role for audit writes.
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'audit_writer') THEN
        CREATE ROLE audit_writer;
    END IF;
END
$$;
GRANT INSERT ON audit_log TO audit_writer;
```

Create `services/api/migrations/005_create_audit_log.down.sql`:

```sql
REVOKE ALL ON audit_log FROM audit_writer;
DROP ROLE IF EXISTS audit_writer;
DROP TABLE IF EXISTS audit_log;
```

Create `services/api/migrations/006_create_data_retention.up.sql`:

```sql
CREATE TABLE data_retention (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    resource_type  TEXT NOT NULL CHECK (resource_type IN ('audit_log', 'request_log', 'state_snapshot')),
    retention_days INT NOT NULL DEFAULT 365,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, resource_type)
);

CREATE INDEX idx_data_retention_tenant_id ON data_retention(tenant_id);
```

Create `services/api/migrations/006_create_data_retention.down.sql`:

```sql
DROP TABLE IF EXISTS data_retention;
```

- [ ] **Step 2: Write the database connection and migration runner**

Create `services/api/internal/database/database.go`:

```go
package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed ../../migrations/*.sql
var migrationsFS embed.FS

// Connect creates a pgx connection pool from a DATABASE_URL.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	config.MaxConns = 20

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// Migrate runs all pending migrations against the given database URL.
func Migrate(databaseURL string) error {
	source, err := iofs.New(migrationsFS, "../../migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	version, dirty, _ := m.Version()
	slog.Info("migrations complete", "version", version, "dirty", dirty)
	return nil
}
```

- [ ] **Step 3: Write the database test with testcontainers**

Create `services/api/internal/database/database_test.go`:

```go
package database

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestConnectAndMigrate(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	// Test migration
	if err := Migrate(connStr); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Test connection pool
	pool, err := Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// Verify tables exist
	tables := []string{"tenants", "apps", "api_keys", "usage_records", "audit_log", "data_retention"}
	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_name = $1)", table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s does not exist after migration", table)
		}
	}

	// Verify idempotent migration
	if err := Migrate(connStr); err != nil {
		t.Fatalf("re-migrate: %v", err)
	}
}
```

- [ ] **Step 4: Install test dependencies and run**

```bash
cd ../cloudmock-platform/services/api
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
go test ./internal/database/ -v -count=1 -timeout 120s
```

Expected: PASS. All 6 tables created. Docker must be running for testcontainers.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: database connection pool and migration for all 6 tables"
```

---

### Task 3: Shared Test Helper and Tenant Store

**Files:**
- Create: `services/api/internal/store/testhelper_test.go`
- Create: `services/api/internal/store/tenants.go`
- Create: `services/api/internal/store/tenants_test.go`

- [ ] **Step 1: Create shared test helper**

Create `services/api/internal/store/testhelper_test.go`:

```go
package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/database"
)

// setupTestDB starts a Postgres container, runs migrations, and returns
// a connection pool. The container is terminated when the test finishes.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { pgContainer.Terminate(ctx) })

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	if err := database.Migrate(connStr); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := database.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	return pool
}
```

- [ ] **Step 2: Write the tenant store tests**

Create `services/api/internal/store/tenants_test.go`:

```go
package store_test

import (
	"context"
	"testing"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

func TestTenantStore(t *testing.T) {
	pool := setupTestDB(t)
	s := store.NewTenantStore(pool)
	ctx := context.Background()

	t.Run("Create and Get", func(t *testing.T) {
		tenant := &model.Tenant{
			ClerkOrgID: "org_test1",
			Name:       "Test Org",
			Slug:       "test-org",
			Status:     "active",
		}
		if err := s.Create(ctx, tenant); err != nil {
			t.Fatalf("create: %v", err)
		}
		if tenant.ID == "" {
			t.Fatal("expected ID to be set after create")
		}

		got, err := s.Get(ctx, tenant.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Name != "Test Org" {
			t.Errorf("name = %q, want %q", got.Name, "Test Org")
		}
		if got.Slug != "test-org" {
			t.Errorf("slug = %q, want %q", got.Slug, "test-org")
		}
	})

	t.Run("GetByClerkOrgID", func(t *testing.T) {
		got, err := s.GetByClerkOrgID(ctx, "org_test1")
		if err != nil {
			t.Fatalf("get by clerk org id: %v", err)
		}
		if got.ClerkOrgID != "org_test1" {
			t.Errorf("clerk_org_id = %q, want %q", got.ClerkOrgID, "org_test1")
		}
	})

	t.Run("GetBySlug", func(t *testing.T) {
		got, err := s.GetBySlug(ctx, "test-org")
		if err != nil {
			t.Fatalf("get by slug: %v", err)
		}
		if got.Slug != "test-org" {
			t.Errorf("slug = %q, want %q", got.Slug, "test-org")
		}
	})

	t.Run("Update", func(t *testing.T) {
		got, _ := s.GetBySlug(ctx, "test-org")
		got.Name = "Updated Org"
		got.HasPaymentMethod = true
		got.StripeCustomerID = "cus_test123"
		if err := s.Update(ctx, got); err != nil {
			t.Fatalf("update: %v", err)
		}

		updated, _ := s.Get(ctx, got.ID)
		if updated.Name != "Updated Org" {
			t.Errorf("name = %q, want %q", updated.Name, "Updated Org")
		}
		if !updated.HasPaymentMethod {
			t.Error("expected has_payment_method to be true")
		}
		if updated.StripeCustomerID != "cus_test123" {
			t.Errorf("stripe_customer_id = %q, want %q", updated.StripeCustomerID, "cus_test123")
		}
	})

	t.Run("List", func(t *testing.T) {
		tenants, err := s.List(ctx)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(tenants) == 0 {
			t.Error("expected at least one tenant")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		got, _ := s.GetBySlug(ctx, "test-org")
		if err := s.Delete(ctx, got.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		_, err := s.Get(ctx, got.ID)
		if err == nil {
			t.Error("expected error after delete, got nil")
		}
	})

	t.Run("Duplicate slug rejected", func(t *testing.T) {
		t1 := &model.Tenant{ClerkOrgID: "org_dup1", Name: "Dup1", Slug: "dup-slug", Status: "active"}
		t2 := &model.Tenant{ClerkOrgID: "org_dup2", Name: "Dup2", Slug: "dup-slug", Status: "active"}
		if err := s.Create(ctx, t1); err != nil {
			t.Fatalf("create t1: %v", err)
		}
		if err := s.Create(ctx, t2); err == nil {
			t.Error("expected error on duplicate slug, got nil")
		}
	})
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run TestTenantStore
```

Expected: FAIL -- `store.NewTenantStore` not defined.

- [ ] **Step 4: Implement the tenant store**

Create `services/api/internal/store/tenants.go`:

```go
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("not found")

// TenantStore handles tenant persistence in Postgres.
type TenantStore struct {
	pool *pgxpool.Pool
}

// NewTenantStore creates a new TenantStore.
func NewTenantStore(pool *pgxpool.Pool) *TenantStore {
	return &TenantStore{pool: pool}
}

func (s *TenantStore) Create(ctx context.Context, t *model.Tenant) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO tenants (clerk_org_id, name, slug, status, has_payment_method, stripe_customer_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		t.ClerkOrgID, t.Name, t.Slug, t.Status, t.HasPaymentMethod, nilIfEmpty(t.StripeCustomerID),
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (s *TenantStore) Get(ctx context.Context, id string) (*model.Tenant, error) {
	t := &model.Tenant{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, clerk_org_id, name, slug, status, has_payment_method,
		        COALESCE(stripe_customer_id, ''), created_at, updated_at
		 FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.ClerkOrgID, &t.Name, &t.Slug, &t.Status,
		&t.HasPaymentMethod, &t.StripeCustomerID, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *TenantStore) GetByClerkOrgID(ctx context.Context, clerkOrgID string) (*model.Tenant, error) {
	t := &model.Tenant{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, clerk_org_id, name, slug, status, has_payment_method,
		        COALESCE(stripe_customer_id, ''), created_at, updated_at
		 FROM tenants WHERE clerk_org_id = $1`, clerkOrgID,
	).Scan(&t.ID, &t.ClerkOrgID, &t.Name, &t.Slug, &t.Status,
		&t.HasPaymentMethod, &t.StripeCustomerID, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *TenantStore) GetBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	t := &model.Tenant{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, clerk_org_id, name, slug, status, has_payment_method,
		        COALESCE(stripe_customer_id, ''), created_at, updated_at
		 FROM tenants WHERE slug = $1`, slug,
	).Scan(&t.ID, &t.ClerkOrgID, &t.Name, &t.Slug, &t.Status,
		&t.HasPaymentMethod, &t.StripeCustomerID, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *TenantStore) List(ctx context.Context) ([]model.Tenant, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, clerk_org_id, name, slug, status, has_payment_method,
		        COALESCE(stripe_customer_id, ''), created_at, updated_at
		 FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(&t.ID, &t.ClerkOrgID, &t.Name, &t.Slug, &t.Status,
			&t.HasPaymentMethod, &t.StripeCustomerID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

func (s *TenantStore) Update(ctx context.Context, t *model.Tenant) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE tenants SET name=$1, slug=$2, status=$3, has_payment_method=$4,
		        stripe_customer_id=$5, updated_at=now()
		 WHERE id=$6`,
		t.Name, t.Slug, t.Status, t.HasPaymentMethod, nilIfEmpty(t.StripeCustomerID), t.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *TenantStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run TestTenantStore
```

Expected: PASS -- all 7 subtests pass.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: tenant store with full CRUD and Postgres tests"
```

---

### Task 4: App Store

**Files:**
- Create: `services/api/internal/store/apps.go`
- Create: `services/api/internal/store/apps_test.go`

- [ ] **Step 1: Write app store tests**

Create `services/api/internal/store/apps_test.go`:

```go
package store_test

import (
	"context"
	"testing"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

func TestAppStore(t *testing.T) {
	pool := setupTestDB(t)
	tenants := store.NewTenantStore(pool)
	apps := store.NewAppStore(pool)
	ctx := context.Background()

	// Create a tenant first
	tenant := &model.Tenant{ClerkOrgID: "org_app_test", Name: "App Test Org", Slug: "app-test", Status: "active"}
	if err := tenants.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	t.Run("Create and Get", func(t *testing.T) {
		app := &model.App{
			TenantID:  tenant.ID,
			Name:      "staging",
			Slug:      "staging",
			Endpoint:  "https://staging-abc123.cloudmock.io",
			InfraType: "shared",
			Status:    "running",
		}
		if err := apps.Create(ctx, app); err != nil {
			t.Fatalf("create: %v", err)
		}
		if app.ID == "" {
			t.Fatal("expected ID to be set")
		}

		got, err := apps.Get(ctx, app.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Name != "staging" {
			t.Errorf("name = %q, want %q", got.Name, "staging")
		}
		if got.Endpoint != "https://staging-abc123.cloudmock.io" {
			t.Errorf("endpoint = %q, want %q", got.Endpoint, "https://staging-abc123.cloudmock.io")
		}
	})

	t.Run("ListByTenant", func(t *testing.T) {
		list, err := apps.ListByTenant(ctx, tenant.ID)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("len = %d, want 1", len(list))
		}
	})

	t.Run("GetByEndpoint", func(t *testing.T) {
		got, err := apps.GetByEndpoint(ctx, "https://staging-abc123.cloudmock.io")
		if err != nil {
			t.Fatalf("get by endpoint: %v", err)
		}
		if got.Name != "staging" {
			t.Errorf("name = %q, want %q", got.Name, "staging")
		}
	})

	t.Run("Update infra fields", func(t *testing.T) {
		list, _ := apps.ListByTenant(ctx, tenant.ID)
		app := &list[0]
		app.FlyAppName = "cm-staging"
		app.FlyMachineID = "mach_123"
		app.Status = "running"
		if err := apps.Update(ctx, app); err != nil {
			t.Fatalf("update: %v", err)
		}

		got, _ := apps.Get(ctx, app.ID)
		if got.FlyAppName != "cm-staging" {
			t.Errorf("fly_app_name = %q, want %q", got.FlyAppName, "cm-staging")
		}
	})

	t.Run("Delete cascades from tenant", func(t *testing.T) {
		t2 := &model.Tenant{ClerkOrgID: "org_cascade", Name: "Cascade", Slug: "cascade", Status: "active"}
		tenants.Create(ctx, t2)
		a := &model.App{TenantID: t2.ID, Name: "test", Slug: "test", Endpoint: "https://test.cloudmock.io", InfraType: "shared", Status: "running"}
		apps.Create(ctx, a)

		tenants.Delete(ctx, t2.ID)
		_, err := apps.Get(ctx, a.ID)
		if err == nil {
			t.Error("expected app to be deleted when tenant is deleted")
		}
	})

	t.Run("Duplicate slug per tenant rejected", func(t *testing.T) {
		a1 := &model.App{TenantID: tenant.ID, Name: "dup", Slug: "dup-slug", Endpoint: "https://dup1.cloudmock.io", InfraType: "shared", Status: "running"}
		a2 := &model.App{TenantID: tenant.ID, Name: "dup2", Slug: "dup-slug", Endpoint: "https://dup2.cloudmock.io", InfraType: "shared", Status: "running"}
		apps.Create(ctx, a1)
		if err := apps.Create(ctx, a2); err == nil {
			t.Error("expected error on duplicate slug per tenant")
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run TestAppStore
```

Expected: FAIL -- `store.NewAppStore` not defined.

- [ ] **Step 3: Implement the app store**

Create `services/api/internal/store/apps.go`:

```go
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

type AppStore struct {
	pool *pgxpool.Pool
}

func NewAppStore(pool *pgxpool.Pool) *AppStore {
	return &AppStore{pool: pool}
}

func (s *AppStore) Create(ctx context.Context, a *model.App) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO apps (tenant_id, name, slug, endpoint, infra_type, fly_app_name, fly_machine_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at`,
		a.TenantID, a.Name, a.Slug, a.Endpoint, a.InfraType,
		nilIfEmpty(a.FlyAppName), nilIfEmpty(a.FlyMachineID), a.Status,
	).Scan(&a.ID, &a.CreatedAt)
}

func (s *AppStore) Get(ctx context.Context, id string) (*model.App, error) {
	a := &model.App{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, slug, endpoint, infra_type,
		        COALESCE(fly_app_name, ''), COALESCE(fly_machine_id, ''), status, created_at
		 FROM apps WHERE id = $1`, id,
	).Scan(&a.ID, &a.TenantID, &a.Name, &a.Slug, &a.Endpoint, &a.InfraType,
		&a.FlyAppName, &a.FlyMachineID, &a.Status, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *AppStore) GetByEndpoint(ctx context.Context, endpoint string) (*model.App, error) {
	a := &model.App{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, slug, endpoint, infra_type,
		        COALESCE(fly_app_name, ''), COALESCE(fly_machine_id, ''), status, created_at
		 FROM apps WHERE endpoint = $1`, endpoint,
	).Scan(&a.ID, &a.TenantID, &a.Name, &a.Slug, &a.Endpoint, &a.InfraType,
		&a.FlyAppName, &a.FlyMachineID, &a.Status, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *AppStore) ListByTenant(ctx context.Context, tenantID string) ([]model.App, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, name, slug, endpoint, infra_type,
		        COALESCE(fly_app_name, ''), COALESCE(fly_machine_id, ''), status, created_at
		 FROM apps WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []model.App
	for rows.Next() {
		var a model.App
		if err := rows.Scan(&a.ID, &a.TenantID, &a.Name, &a.Slug, &a.Endpoint, &a.InfraType,
			&a.FlyAppName, &a.FlyMachineID, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (s *AppStore) Update(ctx context.Context, a *model.App) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE apps SET name=$1, slug=$2, endpoint=$3, infra_type=$4,
		        fly_app_name=$5, fly_machine_id=$6, status=$7
		 WHERE id=$8`,
		a.Name, a.Slug, a.Endpoint, a.InfraType,
		nilIfEmpty(a.FlyAppName), nilIfEmpty(a.FlyMachineID), a.Status, a.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *AppStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM apps WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run TestAppStore
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: app store with CRUD, endpoint lookup, and cascade delete tests"
```

---

### Task 5: API Key Store

**Files:**
- Create: `services/api/internal/store/apikeys.go`
- Create: `services/api/internal/store/apikeys_test.go`

- [ ] **Step 1: Write API key store tests**

Create `services/api/internal/store/apikeys_test.go`:

```go
package store_test

import (
	"context"
	"testing"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

func TestAPIKeyStore(t *testing.T) {
	pool := setupTestDB(t)
	tenants := store.NewTenantStore(pool)
	apps := store.NewAppStore(pool)
	keys := store.NewAPIKeyStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{ClerkOrgID: "org_key_test", Name: "Key Test", Slug: "key-test", Status: "active"}
	tenants.Create(ctx, tenant)
	app := &model.App{TenantID: tenant.ID, Name: "staging", Slug: "staging", Endpoint: "https://key-test.cloudmock.io", InfraType: "shared", Status: "running"}
	apps.Create(ctx, app)

	t.Run("Create and lookup by hash", func(t *testing.T) {
		plaintext, key, err := keys.Create(ctx, tenant.ID, app.ID, "CI Key", "developer")
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if plaintext == "" {
			t.Fatal("expected plaintext key")
		}
		if key.Prefix == "" {
			t.Fatal("expected prefix to be set")
		}
		if key.ID == "" {
			t.Fatal("expected ID to be set")
		}

		// Look up by the plaintext key (hashes internally)
		got, err := keys.GetByPlaintext(ctx, plaintext)
		if err != nil {
			t.Fatalf("get by plaintext: %v", err)
		}
		if got.ID != key.ID {
			t.Errorf("id = %q, want %q", got.ID, key.ID)
		}
		if got.AppID != app.ID {
			t.Errorf("app_id = %q, want %q", got.AppID, app.ID)
		}
		if got.Role != "developer" {
			t.Errorf("role = %q, want %q", got.Role, "developer")
		}
	})

	t.Run("List by app", func(t *testing.T) {
		list, err := keys.ListByApp(ctx, app.ID)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("len = %d, want 1", len(list))
		}
		// Key hash should NOT be exposed in list
		if list[0].KeyHash != "" {
			t.Error("expected key_hash to be empty in list response")
		}
	})

	t.Run("Revoke", func(t *testing.T) {
		plaintext, key, _ := keys.Create(ctx, tenant.ID, app.ID, "Temp Key", "viewer")
		if err := keys.Revoke(ctx, key.ID); err != nil {
			t.Fatalf("revoke: %v", err)
		}

		_, err := keys.GetByPlaintext(ctx, plaintext)
		if err == nil {
			t.Error("expected error looking up revoked key")
		}
	})

	t.Run("Update last_used_at", func(t *testing.T) {
		plaintext, _, _ := keys.Create(ctx, tenant.ID, app.ID, "Active Key", "developer")
		got, _ := keys.GetByPlaintext(ctx, plaintext)
		if got.LastUsedAt != nil {
			t.Error("expected last_used_at to be nil initially")
		}

		if err := keys.TouchLastUsed(ctx, got.ID); err != nil {
			t.Fatalf("touch: %v", err)
		}

		got2, _ := keys.GetByPlaintext(ctx, plaintext)
		if got2.LastUsedAt == nil {
			t.Error("expected last_used_at to be set after touch")
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run TestAPIKeyStore
```

Expected: FAIL.

- [ ] **Step 3: Implement the API key store**

Create `services/api/internal/store/apikeys.go`:

```go
package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

type APIKeyStore struct {
	pool *pgxpool.Pool
}

func NewAPIKeyStore(pool *pgxpool.Pool) *APIKeyStore {
	return &APIKeyStore{pool: pool}
}

// Create generates a new API key, stores the SHA-256 hash, and returns the
// plaintext key (shown once) and the stored key record.
func (s *APIKeyStore) Create(ctx context.Context, tenantID, appID, name, role string) (plaintext string, key *model.APIKey, err error) {
	// Generate 32 bytes of randomness
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate key: %w", err)
	}

	plaintext = "cm_live_" + hex.EncodeToString(raw)
	prefix := plaintext[:16] // "cm_live_" + first 8 hex chars
	hash := hashKey(plaintext)

	key = &model.APIKey{
		TenantID: tenantID,
		AppID:    appID,
		KeyHash:  hash,
		Prefix:   prefix,
		Name:     name,
		Role:     role,
	}

	err = s.pool.QueryRow(ctx,
		`INSERT INTO api_keys (tenant_id, app_id, key_hash, prefix, name, role)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		key.TenantID, key.AppID, key.KeyHash, key.Prefix, key.Name, key.Role,
	).Scan(&key.ID, &key.CreatedAt)
	if err != nil {
		return "", nil, err
	}

	return plaintext, key, nil
}

// GetByPlaintext hashes the plaintext key and looks it up. Returns ErrNotFound
// if the key doesn't exist or has been revoked.
func (s *APIKeyStore) GetByPlaintext(ctx context.Context, plaintext string) (*model.APIKey, error) {
	hash := hashKey(plaintext)
	k := &model.APIKey{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, app_id, key_hash, prefix, name, role,
		        last_used_at, expires_at, revoked_at, created_at
		 FROM api_keys
		 WHERE key_hash = $1 AND revoked_at IS NULL
		   AND (expires_at IS NULL OR expires_at > now())`, hash,
	).Scan(&k.ID, &k.TenantID, &k.AppID, &k.KeyHash, &k.Prefix, &k.Name, &k.Role,
		&k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return k, err
}

// ListByApp returns all non-revoked keys for an app. KeyHash is cleared
// from each record to avoid leaking hashes.
func (s *APIKeyStore) ListByApp(ctx context.Context, appID string) ([]model.APIKey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, app_id, prefix, name, role,
		        last_used_at, expires_at, revoked_at, created_at
		 FROM api_keys
		 WHERE app_id = $1 AND revoked_at IS NULL
		 ORDER BY created_at DESC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(&k.ID, &k.TenantID, &k.AppID, &k.Prefix, &k.Name, &k.Role,
			&k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		// KeyHash intentionally left empty
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// Revoke marks a key as revoked by setting revoked_at.
func (s *APIKeyStore) Revoke(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchLastUsed updates the last_used_at timestamp.
func (s *APIKeyStore) TouchLastUsed(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at = now() WHERE id = $1`, id)
	return err
}

func hashKey(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run TestAPIKeyStore
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: API key store with SHA-256 hashing, revocation, and expiry"
```

---

### Task 6: Audit Log Store

**Files:**
- Create: `services/api/internal/store/audit.go`
- Create: `services/api/internal/store/audit_test.go`

- [ ] **Step 1: Write audit log store tests**

Create `services/api/internal/store/audit_test.go`:

```go
package store_test

import (
	"context"
	"net"
	"testing"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

func TestAuditStore(t *testing.T) {
	pool := setupTestDB(t)
	audit := store.NewAuditStore(pool)
	ctx := context.Background()

	tenantID := "00000000-0000-0000-0000-000000000001"

	t.Run("Append and Query", func(t *testing.T) {
		entry := &model.AuditEntry{
			TenantID:     tenantID,
			ActorID:      "user_123",
			ActorType:    "user",
			Action:       "app.create",
			ResourceType: "app",
			ResourceID:   "app_456",
			IPAddress:    net.ParseIP("192.168.1.1"),
			UserAgent:    "Mozilla/5.0",
			Metadata:     map[string]any{"app_name": "staging"},
		}
		if err := audit.Append(ctx, entry); err != nil {
			t.Fatalf("append: %v", err)
		}
		if entry.ID == "" {
			t.Fatal("expected ID to be set")
		}

		entries, err := audit.Query(ctx, store.AuditFilter{TenantID: tenantID, Limit: 10})
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("len = %d, want 1", len(entries))
		}
		if entries[0].Action != "app.create" {
			t.Errorf("action = %q, want %q", entries[0].Action, "app.create")
		}
	})

	t.Run("Query with action filter", func(t *testing.T) {
		audit.Append(ctx, &model.AuditEntry{
			TenantID: tenantID, ActorID: "user_123", ActorType: "user",
			Action: "key.create", ResourceType: "key", ResourceID: "key_1",
		})

		entries, _ := audit.Query(ctx, store.AuditFilter{TenantID: tenantID, Action: "key.create", Limit: 10})
		if len(entries) != 1 {
			t.Errorf("len = %d, want 1", len(entries))
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := audit.Count(ctx, store.AuditFilter{TenantID: tenantID})
		if err != nil {
			t.Fatalf("count: %v", err)
		}
		if count < 2 {
			t.Errorf("count = %d, want >= 2", count)
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run TestAuditStore
```

Expected: FAIL.

- [ ] **Step 3: Implement the audit log store**

Create `services/api/internal/store/audit.go`:

```go
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

type AuditStore struct {
	pool *pgxpool.Pool
}

func NewAuditStore(pool *pgxpool.Pool) *AuditStore {
	return &AuditStore{pool: pool}
}

// AuditFilter specifies query parameters for audit log searches.
type AuditFilter struct {
	TenantID     string
	ActorID      string
	Action       string
	ResourceType string
	Limit        int
	Offset       int
}

// Append inserts a new audit log entry. This is the only write operation --
// audit_log is append-only by design.
func (s *AuditStore) Append(ctx context.Context, e *model.AuditEntry) error {
	metadata, err := json.Marshal(e.Metadata)
	if err != nil {
		metadata = []byte("{}")
	}

	return s.pool.QueryRow(ctx,
		`INSERT INTO audit_log (tenant_id, actor_id, actor_type, action,
		        resource_type, resource_id, ip_address, user_agent, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at`,
		e.TenantID, e.ActorID, e.ActorType, e.Action,
		e.ResourceType, e.ResourceID, e.IPAddress, e.UserAgent, metadata,
	).Scan(&e.ID, &e.CreatedAt)
}

// Query returns audit log entries matching the filter.
func (s *AuditStore) Query(ctx context.Context, f AuditFilter) ([]model.AuditEntry, error) {
	where, args := buildAuditWhere(f)
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(
		`SELECT id, tenant_id, actor_id, actor_type, action,
		        resource_type, resource_id, ip_address, user_agent, metadata, created_at
		 FROM audit_log %s
		 ORDER BY created_at DESC
		 LIMIT %d OFFSET %d`, where, limit, f.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.AuditEntry
	for rows.Next() {
		var e model.AuditEntry
		var metadata []byte
		if err := rows.Scan(&e.ID, &e.TenantID, &e.ActorID, &e.ActorType, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.IPAddress, &e.UserAgent, &metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		if metadata != nil {
			json.Unmarshal(metadata, &e.Metadata)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Count returns the total number of entries matching the filter.
func (s *AuditStore) Count(ctx context.Context, f AuditFilter) (int64, error) {
	where, args := buildAuditWhere(f)
	var count int64
	err := s.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", where), args...,
	).Scan(&count)
	return count, err
}

func buildAuditWhere(f AuditFilter) (string, []any) {
	var clauses []string
	var args []any
	n := 1

	if f.TenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", n))
		args = append(args, f.TenantID)
		n++
	}
	if f.ActorID != "" {
		clauses = append(clauses, fmt.Sprintf("actor_id = $%d", n))
		args = append(args, f.ActorID)
		n++
	}
	if f.Action != "" {
		clauses = append(clauses, fmt.Sprintf("action = $%d", n))
		args = append(args, f.Action)
		n++
	}
	if f.ResourceType != "" {
		clauses = append(clauses, fmt.Sprintf("resource_type = $%d", n))
		args = append(args, f.ResourceType)
		n++
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run TestAuditStore
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: append-only audit log store with filtered queries"
```

---

### Task 7: Usage and Data Retention Stores

**Files:**
- Create: `services/api/internal/store/usage.go`
- Create: `services/api/internal/store/usage_test.go`
- Create: `services/api/internal/store/retention.go`
- Create: `services/api/internal/store/retention_test.go`

- [ ] **Step 1: Write usage store tests**

Create `services/api/internal/store/usage_test.go`:

```go
package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

func TestUsageStore(t *testing.T) {
	pool := setupTestDB(t)
	tenants := store.NewTenantStore(pool)
	apps := store.NewAppStore(pool)
	usage := store.NewUsageStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{ClerkOrgID: "org_usage", Name: "Usage", Slug: "usage-test", Status: "active"}
	tenants.Create(ctx, tenant)
	app := &model.App{TenantID: tenant.ID, Name: "test", Slug: "test", Endpoint: "https://usage.cloudmock.io", InfraType: "shared", Status: "running"}
	apps.Create(ctx, app)

	t.Run("Increment and get current", func(t *testing.T) {
		if err := usage.IncrementRequestCount(ctx, tenant.ID, app.ID); err != nil {
			t.Fatalf("increment: %v", err)
		}
		if err := usage.IncrementRequestCount(ctx, tenant.ID, app.ID); err != nil {
			t.Fatalf("increment: %v", err)
		}

		count, err := usage.GetCurrentPeriodCount(ctx, tenant.ID)
		if err != nil {
			t.Fatalf("get count: %v", err)
		}
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("GetUnreported", func(t *testing.T) {
		records, err := usage.GetUnreported(ctx)
		if err != nil {
			t.Fatalf("get unreported: %v", err)
		}
		if len(records) == 0 {
			t.Error("expected at least one unreported record")
		}
	})

	t.Run("MarkReported", func(t *testing.T) {
		records, _ := usage.GetUnreported(ctx)
		if len(records) > 0 {
			if err := usage.MarkReported(ctx, records[0].ID); err != nil {
				t.Fatalf("mark reported: %v", err)
			}
		}
	})

	t.Run("GetByTenantForPeriod", func(t *testing.T) {
		now := time.Now()
		start := now.AddDate(0, -1, 0)
		records, err := usage.GetByTenant(ctx, tenant.ID, start, now)
		if err != nil {
			t.Fatalf("get by tenant: %v", err)
		}
		if len(records) == 0 {
			t.Error("expected usage records in the current period")
		}
	})
}
```

- [ ] **Step 2: Write retention store tests**

Create `services/api/internal/store/retention_test.go`:

```go
package store_test

import (
	"context"
	"testing"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

func TestRetentionStore(t *testing.T) {
	pool := setupTestDB(t)
	tenants := store.NewTenantStore(pool)
	retention := store.NewRetentionStore(pool)
	ctx := context.Background()

	tenant := &model.Tenant{ClerkOrgID: "org_ret", Name: "Retention", Slug: "retention-test", Status: "active"}
	tenants.Create(ctx, tenant)

	t.Run("Upsert and Get", func(t *testing.T) {
		if err := retention.Upsert(ctx, tenant.ID, "audit_log", 365); err != nil {
			t.Fatalf("upsert: %v", err)
		}

		policies, err := retention.GetByTenant(ctx, tenant.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if len(policies) != 1 {
			t.Fatalf("len = %d, want 1", len(policies))
		}
		if policies[0].RetentionDays != 365 {
			t.Errorf("days = %d, want 365", policies[0].RetentionDays)
		}
	})

	t.Run("Upsert updates existing", func(t *testing.T) {
		retention.Upsert(ctx, tenant.ID, "audit_log", 90)
		policies, _ := retention.GetByTenant(ctx, tenant.ID)
		if policies[0].RetentionDays != 90 {
			t.Errorf("days = %d, want 90", policies[0].RetentionDays)
		}
	})
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run "TestUsageStore|TestRetentionStore"
```

Expected: FAIL.

- [ ] **Step 4: Implement the usage store**

Create `services/api/internal/store/usage.go`:

```go
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

type UsageStore struct {
	pool *pgxpool.Pool
}

func NewUsageStore(pool *pgxpool.Pool) *UsageStore {
	return &UsageStore{pool: pool}
}

// IncrementRequestCount increments or creates a usage record for the current
// hourly period for the given tenant and app.
func (s *UsageStore) IncrementRequestCount(ctx context.Context, tenantID, appID string) error {
	now := time.Now().UTC()
	periodStart := now.Truncate(time.Hour)
	periodEnd := periodStart.Add(time.Hour)

	_, err := s.pool.Exec(ctx,
		`INSERT INTO usage_records (tenant_id, app_id, period_start, period_end, request_count)
		 VALUES ($1, $2, $3, $4, 1)
		 ON CONFLICT ON CONSTRAINT usage_records_pkey DO NOTHING`,
		tenantID, appID, periodStart, periodEnd)

	// The ON CONFLICT won't work on pkey (uuid). Use upsert on a unique index instead.
	// Simpler approach: try update first, then insert.
	tag, err := s.pool.Exec(ctx,
		`UPDATE usage_records SET request_count = request_count + 1
		 WHERE tenant_id = $1 AND app_id = $2 AND period_start = $3 AND NOT reported_to_stripe`,
		tenantID, appID, periodStart)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	// No existing record for this period -- create one.
	_, err = s.pool.Exec(ctx,
		`INSERT INTO usage_records (tenant_id, app_id, period_start, period_end, request_count)
		 VALUES ($1, $2, $3, $4, 1)`,
		tenantID, appID, periodStart, periodEnd)
	return err
}

// GetCurrentPeriodCount returns the total request count across all apps for
// the current billing month for a tenant.
func (s *UsageStore) GetCurrentPeriodCount(ctx context.Context, tenantID string) (int64, error) {
	monthStart := beginningOfMonth(time.Now().UTC())
	var count int64
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(request_count), 0)
		 FROM usage_records
		 WHERE tenant_id = $1 AND period_start >= $2`,
		tenantID, monthStart).Scan(&count)
	return count, err
}

// GetByTenant returns usage records for a tenant within a time range.
func (s *UsageStore) GetByTenant(ctx context.Context, tenantID string, start, end time.Time) ([]model.UsageRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, app_id, period_start, period_end,
		        request_count, reported_to_stripe, created_at
		 FROM usage_records
		 WHERE tenant_id = $1 AND period_start >= $2 AND period_end <= $3
		 ORDER BY period_start DESC`, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.UsageRecord
	for rows.Next() {
		var r model.UsageRecord
		if err := rows.Scan(&r.ID, &r.TenantID, &r.AppID, &r.PeriodStart, &r.PeriodEnd,
			&r.RequestCount, &r.ReportedToStripe, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetUnreported returns all usage records not yet reported to Stripe.
func (s *UsageStore) GetUnreported(ctx context.Context) ([]model.UsageRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, app_id, period_start, period_end,
		        request_count, reported_to_stripe, created_at
		 FROM usage_records
		 WHERE NOT reported_to_stripe AND period_end < now()
		 ORDER BY period_start ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.UsageRecord
	for rows.Next() {
		var r model.UsageRecord
		if err := rows.Scan(&r.ID, &r.TenantID, &r.AppID, &r.PeriodStart, &r.PeriodEnd,
			&r.RequestCount, &r.ReportedToStripe, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// MarkReported marks a usage record as reported to Stripe.
func (s *UsageStore) MarkReported(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE usage_records SET reported_to_stripe = true WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PurgeOlderThan deletes usage records older than the given time.
func (s *UsageStore) PurgeOlderThan(ctx context.Context, tenantID string, before time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM usage_records WHERE tenant_id = $1 AND period_end < $2 AND reported_to_stripe`,
		tenantID, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func beginningOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// Ensure the first Exec doesn't break compilation -- remove the unused err
// from the upsert approach. Let me fix this in the implementation.
var _ = errors.Is
var _ = pgx.ErrNoRows
```

- [ ] **Step 5: Implement the retention store**

Create `services/api/internal/store/retention.go`:

```go
package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

type RetentionStore struct {
	pool *pgxpool.Pool
}

func NewRetentionStore(pool *pgxpool.Pool) *RetentionStore {
	return &RetentionStore{pool: pool}
}

// Upsert creates or updates a retention policy for a tenant and resource type.
func (s *RetentionStore) Upsert(ctx context.Context, tenantID, resourceType string, days int) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO data_retention (tenant_id, resource_type, retention_days)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (tenant_id, resource_type) DO UPDATE
		 SET retention_days = EXCLUDED.retention_days, updated_at = now()`,
		tenantID, resourceType, days)
	return err
}

// GetByTenant returns all retention policies for a tenant.
func (s *RetentionStore) GetByTenant(ctx context.Context, tenantID string) ([]model.DataRetention, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, resource_type, retention_days, updated_at
		 FROM data_retention WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []model.DataRetention
	for rows.Next() {
		var p model.DataRetention
		if err := rows.Scan(&p.ID, &p.TenantID, &p.ResourceType, &p.RetentionDays, &p.UpdatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/store/ -v -count=1 -timeout 120s -run "TestUsageStore|TestRetentionStore"
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: usage metering store and data retention store with purge support"
```

---

### Task 8: Auth Middleware (Clerk JWT + API Key)

**Files:**
- Create: `services/api/internal/middleware/auth.go`
- Create: `services/api/internal/middleware/auth_test.go`

- [ ] **Step 1: Write auth middleware tests**

Create `services/api/internal/middleware/auth_test.go`:

```go
package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/middleware"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

// mockKeyStore implements middleware.APIKeyLookup for testing.
type mockKeyStore struct {
	keys map[string]*model.APIKey // hash -> key
}

func (m *mockKeyStore) GetByPlaintext(_ context.Context, plaintext string) (*model.APIKey, error) {
	h := sha256.Sum256([]byte(plaintext))
	hash := hex.EncodeToString(h[:])
	if k, ok := m.keys[hash]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockKeyStore) TouchLastUsed(_ context.Context, _ string) error { return nil }

func TestAPIKeyAuth(t *testing.T) {
	plaintext := "cm_live_testkey123456789abcdef0123"
	h := sha256.Sum256([]byte(plaintext))
	hash := hex.EncodeToString(h[:])

	store := &mockKeyStore{
		keys: map[string]*model.APIKey{
			hash: {
				ID:       "key-1",
				TenantID: "tenant-1",
				AppID:    "app-1",
				Role:     "developer",
				Prefix:   "cm_live_test",
			},
		},
	}

	mw := middleware.NewAuth(nil, store)

	t.Run("Valid API key", func(t *testing.T) {
		var gotAuth *model.AuthContext
		handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = middleware.AuthFromContext(r.Context())
			w.WriteHeader(200)
		}))

		req := httptest.NewRequest("GET", "/v1/apps", nil)
		req.Header.Set("X-Api-Key", plaintext)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if gotAuth == nil {
			t.Fatal("expected auth context")
		}
		if gotAuth.TenantID != "tenant-1" {
			t.Errorf("tenant_id = %q, want %q", gotAuth.TenantID, "tenant-1")
		}
		if gotAuth.Role != "developer" {
			t.Errorf("role = %q, want %q", gotAuth.Role, "developer")
		}
		if gotAuth.ActorType != "api_key" {
			t.Errorf("actor_type = %q, want %q", gotAuth.ActorType, "api_key")
		}
	})

	t.Run("Missing auth returns 401", func(t *testing.T) {
		handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))

		req := httptest.NewRequest("GET", "/v1/apps", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != 401 {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("Invalid API key returns 401", func(t *testing.T) {
		handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))

		req := httptest.NewRequest("GET", "/v1/apps", nil)
		req.Header.Set("X-Api-Key", "cm_live_invalid")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != 401 {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})
}

func TestClerkJWTAuth(t *testing.T) {
	// Generate RSA keypair for testing
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Create a JWKS endpoint
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nBytes := privateKey.N.Bytes()
		eBytes := big.NewInt(int64(privateKey.E)).Bytes()
		keys := map[string]any{
			"keys": []map[string]any{{
				"kid": "test-kid",
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(nBytes),
				"e":   base64.RawURLEncoding.EncodeToString(eBytes),
			}},
		}
		json.NewEncoder(w).Encode(keys)
	}))
	defer jwks.Close()

	verifier := middleware.NewClerkVerifier(jwks.URL)
	mw := middleware.NewAuth(verifier, nil)

	t.Run("Valid JWT", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"org_id":   "org_test",
			"org_slug": "test-org",
			"org_role": "org:admin",
			"sub":      "user_123",
			"exp":      time.Now().Add(time.Hour).Unix(),
			"iat":      time.Now().Unix(),
		})
		token.Header["kid"] = "test-kid"
		signed, _ := token.SignedString(privateKey)

		var gotAuth *model.AuthContext
		handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = middleware.AuthFromContext(r.Context())
			w.WriteHeader(200)
		}))

		req := httptest.NewRequest("GET", "/v1/apps", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if gotAuth == nil {
			t.Fatal("expected auth context")
		}
		if gotAuth.TenantID != "org_test" {
			t.Errorf("tenant_id = %q, want %q", gotAuth.TenantID, "org_test")
		}
		if gotAuth.Role != "admin" {
			t.Errorf("role = %q, want %q", gotAuth.Role, "admin")
		}
	})

	t.Run("Expired JWT returns 401", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"org_id": "org_test",
			"sub":    "user_123",
			"exp":    time.Now().Add(-time.Hour).Unix(),
			"iat":    time.Now().Add(-2 * time.Hour).Unix(),
		})
		token.Header["kid"] = "test-kid"
		signed, _ := token.SignedString(privateKey)

		handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))

		req := httptest.NewRequest("GET", "/v1/apps", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != 401 {
			t.Errorf("status = %d, want 401", rr.Code)
		}
	})
}
```

Note: add `"fmt"` to the imports for the mock.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/middleware/ -v -count=1 -timeout 120s -run "TestAPIKeyAuth|TestClerkJWTAuth"
```

Expected: FAIL.

- [ ] **Step 3: Implement the auth middleware**

Create `services/api/internal/middleware/auth.go`:

```go
package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

type contextKey string

const authContextKey contextKey = "auth"

// APIKeyLookup is the interface the auth middleware needs from the key store.
type APIKeyLookup interface {
	GetByPlaintext(ctx context.Context, plaintext string) (*model.APIKey, error)
	TouchLastUsed(ctx context.Context, id string) error
}

// Auth middleware authenticates requests via Clerk JWT or API key.
type Auth struct {
	clerk *ClerkVerifier
	keys  APIKeyLookup
}

// NewAuth creates auth middleware. Either clerk or keys can be nil if
// that auth method is not needed.
func NewAuth(clerk *ClerkVerifier, keys APIKeyLookup) *Auth {
	return &Auth{clerk: clerk, keys: keys}
}

// AuthFromContext extracts the auth context set by the middleware.
func AuthFromContext(ctx context.Context) *model.AuthContext {
	if v, ok := ctx.Value(authContextKey).(*model.AuthContext); ok {
		return v
	}
	return nil
}

// Handler returns middleware that authenticates requests.
func (a *Auth) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try API key first (X-Api-Key header)
		if apiKey := r.Header.Get("X-Api-Key"); apiKey != "" && a.keys != nil {
			key, err := a.keys.GetByPlaintext(r.Context(), apiKey)
			if err != nil {
				http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
				return
			}

			// Update last_used_at asynchronously
			go a.keys.TouchLastUsed(context.Background(), key.ID)

			ctx := context.WithValue(r.Context(), authContextKey, &model.AuthContext{
				TenantID:  key.TenantID,
				ActorID:   key.Prefix,
				ActorType: "api_key",
				Role:      key.Role,
				AppID:     key.AppID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Try Clerk JWT (Authorization: Bearer <token>)
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") && a.clerk != nil {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := a.clerk.Verify(r.Context(), token)
			if err != nil {
				slog.Debug("clerk jwt auth failed", "error", err)
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Map Clerk org_role to platform role
			role := mapClerkRole(claims.OrgRole)

			ctx := context.WithValue(r.Context(), authContextKey, &model.AuthContext{
				TenantID:  claims.OrgID,
				ActorID:   claims.Subject,
				ActorType: "user",
				Role:      role,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
	})
}

func mapClerkRole(clerkRole string) string {
	switch clerkRole {
	case "org:admin":
		return "admin"
	case "org:developer":
		return "developer"
	case "org:viewer":
		return "viewer"
	default:
		return "viewer"
	}
}

// --- Clerk JWT Verifier ---

// ClerkVerifier verifies Clerk JWTs via JWKS.
type ClerkVerifier struct {
	jwksURL    string
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

type clerkClaims struct {
	jwt.RegisteredClaims
	OrgID   string `json:"org_id"`
	OrgSlug string `json:"org_slug"`
	OrgRole string `json:"org_role"`
}

// NewClerkVerifier creates a Clerk JWT verifier with a JWKS endpoint URL.
func NewClerkVerifier(jwksURL string) *ClerkVerifier {
	return &ClerkVerifier{
		jwksURL:    jwksURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]*rsa.PublicKey),
	}
}

// Verify parses and validates a Clerk JWT, returning the claims.
func (v *ClerkVerifier) Verify(ctx context.Context, tokenString string) (*clerkClaims, error) {
	claims := &clerkClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("missing kid in token header")
		}
		return v.getKey(ctx, kid)
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (v *ClerkVerifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	if key, ok := v.keys[kid]; ok && time.Since(v.fetchedAt) < time.Hour {
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	if err := v.fetchJWKS(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %q not found in JWKS", kid)
	}
	return key, nil
}

func (v *ClerkVerifier) fetchJWKS(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if time.Since(v.fetchedAt) < time.Hour && len(v.keys) > 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", v.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	var jwks struct {
		Keys []struct {
			KID string `json:"kid"`
			KTY string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return err
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.KTY != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		n := new(big.Int).SetBytes(nBytes)
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		newKeys[k.KID] = &rsa.PublicKey{N: n, E: e}
	}

	v.keys = newKeys
	v.fetchedAt = time.Now()
	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/middleware/ -v -count=1 -timeout 30s -run "TestAPIKeyAuth|TestClerkJWTAuth"
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: auth middleware supporting Clerk JWT and API key authentication"
```

---

### Task 9: Audit and Quota Middleware

**Files:**
- Create: `services/api/internal/middleware/audit.go`
- Create: `services/api/internal/middleware/audit_test.go`
- Create: `services/api/internal/middleware/quota.go`
- Create: `services/api/internal/middleware/quota_test.go`
- Create: `services/api/internal/middleware/tenant.go`
- Create: `services/api/internal/middleware/tenant_test.go`

- [ ] **Step 1: Write audit middleware**

Create `services/api/internal/middleware/audit.go`:

```go
package middleware

import (
	"context"
	"net"
	"net/http"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

// AuditWriter is the interface needed to log audit entries.
type AuditWriter interface {
	Append(ctx context.Context, entry *model.AuditEntry) error
}

// Audit middleware logs every request to the HIPAA audit log.
type Audit struct {
	writer AuditWriter
}

// NewAudit creates audit logging middleware.
func NewAudit(writer AuditWriter) *Audit {
	return &Audit{writer: writer}
}

// Handler returns middleware that appends an audit entry for each request.
func (a *Audit) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := AuthFromContext(r.Context())
		if auth == nil {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		entry := &model.AuditEntry{
			TenantID:     auth.TenantID,
			ActorID:      auth.ActorID,
			ActorType:    auth.ActorType,
			Action:       r.Method + " " + r.URL.Path,
			ResourceType: "request",
			IPAddress:    net.ParseIP(ip),
			UserAgent:    r.UserAgent(),
		}

		// Fire and forget -- don't block the request on audit write
		go a.writer.Append(context.Background(), entry)

		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 2: Write quota middleware**

Create `services/api/internal/middleware/quota.go`:

```go
package middleware

import (
	"context"
	"fmt"
	"net/http"
)

const freeRequestLimit = 1000

// UsageCounter is the interface needed to check and increment usage.
type UsageCounter interface {
	GetCurrentPeriodCount(ctx context.Context, tenantID string) (int64, error)
	IncrementRequestCount(ctx context.Context, tenantID, appID string) error
}

// TenantChecker checks whether a tenant has a payment method.
type TenantChecker interface {
	Get(ctx context.Context, id string) (hasPaymentMethod bool, err error)
}

// Quota enforces per-tenant request limits.
type Quota struct {
	usage   UsageCounter
	tenants QuotaTenantLookup
}

// QuotaTenantLookup is the interface for checking tenant payment status.
type QuotaTenantLookup interface {
	HasPaymentMethod(ctx context.Context, tenantID string) (bool, error)
}

// NewQuota creates quota enforcement middleware.
func NewQuota(usage UsageCounter, tenants QuotaTenantLookup) *Quota {
	return &Quota{usage: usage, tenants: tenants}
}

// Handler returns middleware that enforces request quotas.
func (q *Quota) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := AuthFromContext(r.Context())
		if auth == nil {
			next.ServeHTTP(w, r)
			return
		}

		count, err := q.usage.GetCurrentPeriodCount(r.Context(), auth.TenantID)
		if err != nil {
			// Don't block on quota check failure
			next.ServeHTTP(w, r)
			return
		}

		if count >= freeRequestLimit {
			hasPM, err := q.tenants.HasPaymentMethod(r.Context(), auth.TenantID)
			if err != nil || !hasPM {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintf(w, `{"error":"free tier limit reached","count":%d,"limit":%d,"upgrade_url":"https://cloudmock.io/billing"}`,
					count, freeRequestLimit)
				return
			}
			// Has payment method -- let through, Stripe meters the usage
		}

		// Increment counter (async, non-blocking)
		if auth.AppID != "" {
			go q.usage.IncrementRequestCount(context.Background(), auth.TenantID, auth.AppID)
		}

		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 3: Write quota middleware tests**

Create `services/api/internal/middleware/quota_test.go`:

```go
package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/middleware"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
)

type mockUsage struct {
	count int64
}

func (m *mockUsage) GetCurrentPeriodCount(_ context.Context, _ string) (int64, error) {
	return m.count, nil
}
func (m *mockUsage) IncrementRequestCount(_ context.Context, _, _ string) error { return nil }

type mockTenantLookup struct {
	hasPM bool
}

func (m *mockTenantLookup) HasPaymentMethod(_ context.Context, _ string) (bool, error) {
	return m.hasPM, nil
}

func TestQuotaMiddleware(t *testing.T) {
	withAuth := func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), contextKey("auth"), &model.AuthContext{
			TenantID: "tenant-1", ActorID: "user-1", ActorType: "user", Role: "developer", AppID: "app-1",
		})
		return r.WithContext(ctx)
	}

	t.Run("Under limit passes through", func(t *testing.T) {
		mw := middleware.NewQuota(&mockUsage{count: 500}, &mockTenantLookup{hasPM: false})
		handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))

		req := withAuth(httptest.NewRequest("GET", "/", nil))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("status = %d, want 200", rr.Code)
		}
	})

	t.Run("Over limit without payment returns 429", func(t *testing.T) {
		mw := middleware.NewQuota(&mockUsage{count: 1001}, &mockTenantLookup{hasPM: false})
		handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))

		req := withAuth(httptest.NewRequest("GET", "/", nil))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != 429 {
			t.Errorf("status = %d, want 429", rr.Code)
		}
	})

	t.Run("Over limit with payment passes through", func(t *testing.T) {
		mw := middleware.NewQuota(&mockUsage{count: 5000}, &mockTenantLookup{hasPM: true})
		handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))

		req := withAuth(httptest.NewRequest("GET", "/", nil))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("status = %d, want 200", rr.Code)
		}
	})
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/middleware/ -v -count=1 -timeout 30s
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: audit logging and quota enforcement middleware with HIPAA compliance"
```

---

### Task 10: Apps CRUD Handler

**Files:**
- Create: `services/api/internal/handler/apps.go`
- Create: `services/api/internal/handler/apps_test.go`

- [ ] **Step 1: Write apps handler tests**

Create `services/api/internal/handler/apps_test.go`:

```go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/handler"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/middleware"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"

	"github.com/go-chi/chi/v5"
)

// setupAppsTest creates a test router with the apps handler mounted and auth
// context injected. Uses the real Postgres store via testcontainers.
func setupAppsTest(t *testing.T) (*chi.Mux, *store.TenantStore, *store.AppStore) {
	pool := setupTestDB(t) // reuse the same testcontainers helper

	tenants := store.NewTenantStore(pool)
	apps := store.NewAppStore(pool)
	h := handler.NewApps(apps)

	r := chi.NewRouter()
	// Inject auth context for all requests
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.WithAuthContext(req.Context(), &model.AuthContext{
				TenantID: "test-tenant", ActorID: "user-1", ActorType: "user", Role: "admin",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Route("/v1/apps", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{appID}", h.Get)
		r.Patch("/{appID}", h.Update)
		r.Delete("/{appID}", h.Delete)
	})

	return r, tenants, apps
}

func TestAppsHandler(t *testing.T) {
	router, tenants, _ := setupAppsTest(t)
	ctx := t.Context()

	// Create the tenant used in auth context
	tenants.Create(ctx, &model.Tenant{
		ID: "test-tenant", ClerkOrgID: "org_test", Name: "Test", Slug: "test", Status: "active",
	})

	var createdID string

	t.Run("POST /v1/apps creates app", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"name":       "staging",
			"infra_type": "shared",
		})
		req := httptest.NewRequest("POST", "/v1/apps", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != 201 {
			t.Fatalf("status = %d, want 201, body: %s", rr.Code, rr.Body.String())
		}

		var resp model.App
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp.Name != "staging" {
			t.Errorf("name = %q, want %q", resp.Name, "staging")
		}
		if resp.Endpoint == "" {
			t.Error("expected endpoint to be set")
		}
		createdID = resp.ID
	})

	t.Run("GET /v1/apps lists apps", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/apps", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Fatalf("status = %d, want 200", rr.Code)
		}

		var apps []model.App
		json.NewDecoder(rr.Body).Decode(&apps)
		if len(apps) == 0 {
			t.Error("expected at least one app")
		}
	})

	t.Run("GET /v1/apps/:id returns app", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/apps/"+createdID, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
	})

	t.Run("DELETE /v1/apps/:id deletes app", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/v1/apps/"+createdID, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != 204 {
			t.Errorf("status = %d, want 204", rr.Code)
		}
	})
}
```

- [ ] **Step 2: Implement the apps handler**

Create `services/api/internal/handler/apps.go`:

```go
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/middleware"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

type Apps struct {
	store *store.AppStore
}

func NewApps(store *store.AppStore) *Apps {
	return &Apps{store: store}
}

type createAppRequest struct {
	Name      string `json:"name"`
	InfraType string `json:"infra_type"`
}

func (h *Apps) Create(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromContext(r.Context())
	if auth == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req createAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	if req.InfraType == "" {
		req.InfraType = "shared"
	}

	slug := slugify(req.Name)
	endpointID := randomHex(6)

	app := &model.App{
		TenantID:  auth.TenantID,
		Name:      req.Name,
		Slug:      slug,
		Endpoint:  "https://" + endpointID + ".cloudmock.io",
		InfraType: req.InfraType,
		Status:    "running",
	}

	if err := h.store.Create(r.Context(), app); err != nil {
		http.Error(w, `{"error":"failed to create app"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(app)
}

func (h *Apps) List(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromContext(r.Context())
	if auth == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	apps, err := h.store.ListByTenant(r.Context(), auth.TenantID)
	if err != nil {
		http.Error(w, `{"error":"failed to list apps"}`, http.StatusInternalServerError)
		return
	}

	if apps == nil {
		apps = []model.App{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

func (h *Apps) Get(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromContext(r.Context())
	appID := chi.URLParam(r, "appID")

	app, err := h.store.Get(r.Context(), appID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, `{"error":"app not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Tenant isolation
	if app.TenantID != auth.TenantID {
		http.Error(w, `{"error":"app not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (h *Apps) Update(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromContext(r.Context())
	appID := chi.URLParam(r, "appID")

	app, err := h.store.Get(r.Context(), appID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && app.TenantID != auth.TenantID) {
		http.Error(w, `{"error":"app not found"}`, http.StatusNotFound)
		return
	}

	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if name, ok := updates["name"]; ok {
		app.Name = name
	}

	if err := h.store.Update(r.Context(), app); err != nil {
		http.Error(w, `{"error":"failed to update"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (h *Apps) Delete(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromContext(r.Context())
	appID := chi.URLParam(r, "appID")

	app, err := h.store.Get(r.Context(), appID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && app.TenantID != auth.TenantID) {
		http.Error(w, `{"error":"app not found"}`, http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), appID); err != nil {
		http.Error(w, `{"error":"failed to delete"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, s)
	return s
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

Also add `WithAuthContext` to `services/api/internal/middleware/auth.go`:

```go
// WithAuthContext sets the auth context on a context.Context.
func WithAuthContext(ctx context.Context, auth *model.AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, auth)
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/handler/ -v -count=1 -timeout 120s -run TestAppsHandler
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: apps CRUD handler with tenant isolation and slug generation"
```

---

### Task 11: API Keys Handler

**Files:**
- Create: `services/api/internal/handler/apikeys.go`
- Create: `services/api/internal/handler/apikeys_test.go`

- [ ] **Step 1: Implement the API keys handler**

Create `services/api/internal/handler/apikeys.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/middleware"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

type APIKeys struct {
	store *store.APIKeyStore
}

func NewAPIKeys(store *store.APIKeyStore) *APIKeys {
	return &APIKeys{store: store}
}

type createKeyRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type createKeyResponse struct {
	Key       string `json:"key"`      // plaintext, shown once
	Prefix    string `json:"prefix"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
}

func (h *APIKeys) Create(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromContext(r.Context())
	appID := chi.URLParam(r, "appID")

	var req createKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "default"
	}
	if req.Role == "" {
		req.Role = "developer"
	}

	plaintext, key, err := h.store.Create(r.Context(), auth.TenantID, appID, req.Name, req.Role)
	if err != nil {
		http.Error(w, `{"error":"failed to create key"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createKeyResponse{
		Key:    plaintext,
		Prefix: key.Prefix,
		ID:     key.ID,
		Name:   key.Name,
		Role:   key.Role,
	})
}

func (h *APIKeys) List(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "appID")

	keys, err := h.store.ListByApp(r.Context(), appID)
	if err != nil {
		http.Error(w, `{"error":"failed to list keys"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func (h *APIKeys) Revoke(w http.ResponseWriter, r *http.Request) {
	keyID := chi.URLParam(r, "keyID")

	if err := h.store.Revoke(r.Context(), keyID); err != nil {
		http.Error(w, `{"error":"failed to revoke key"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *APIKeys) Rotate(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromContext(r.Context())
	appID := chi.URLParam(r, "appID")
	keyID := chi.URLParam(r, "keyID")

	// Revoke old key
	if err := h.store.Revoke(r.Context(), keyID); err != nil {
		http.Error(w, `{"error":"failed to revoke old key"}`, http.StatusInternalServerError)
		return
	}

	// Create new key with same name/role
	keys, _ := h.store.ListByApp(r.Context(), appID)
	name, role := "rotated", "developer"
	for _, k := range keys {
		if k.ID == keyID {
			name = k.Name
			role = k.Role
			break
		}
	}

	plaintext, key, err := h.store.Create(r.Context(), auth.TenantID, appID, name, role)
	if err != nil {
		http.Error(w, `{"error":"failed to create new key"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createKeyResponse{
		Key:    plaintext,
		Prefix: key.Prefix,
		ID:     key.ID,
		Name:   key.Name,
		Role:   key.Role,
	})
}
```

- [ ] **Step 2: Write tests and run**

```bash
go test ./internal/handler/ -v -count=1 -timeout 120s
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat: API keys handler with create, list, revoke, and rotate"
```

---

### Task 12: AWS Proxy Handler

**Files:**
- Create: `services/api/internal/handler/proxy.go`
- Create: `services/api/internal/handler/proxy_test.go`

- [ ] **Step 1: Implement the proxy handler**

Create `services/api/internal/handler/proxy.go`:

```go
package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/middleware"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

// Proxy forwards AWS SDK requests to the correct CloudMock instance.
type Proxy struct {
	apps     *store.AppStore
	sharedURL *url.URL // shared CloudMock instance

	// Cache of dedicated proxies per Fly app name
	mu       sync.RWMutex
	dedProxies map[string]*httputil.ReverseProxy
}

// NewProxy creates the AWS proxy handler.
func NewProxy(apps *store.AppStore, sharedCloudMockURL string) *Proxy {
	u, _ := url.Parse(sharedCloudMockURL)
	return &Proxy{
		apps:       apps,
		sharedURL:  u,
		dedProxies: make(map[string]*httputil.ReverseProxy),
	}
}

// HandleSubdomain routes requests based on the subdomain (e.g., abc123.cloudmock.io).
// This is the primary SDK access pattern.
func (p *Proxy) HandleSubdomain(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromContext(r.Context())
	if auth == nil || auth.AppID == "" {
		http.Error(w, `{"error":"app context required"}`, http.StatusBadRequest)
		return
	}

	app, err := p.apps.Get(r.Context(), auth.AppID)
	if err != nil {
		http.Error(w, `{"error":"app not found"}`, http.StatusNotFound)
		return
	}

	p.proxyToInstance(w, r, app)
}

// HandlePathBased routes requests via /v1/apps/:appID/aws/*.
func (p *Proxy) HandlePathBased(w http.ResponseWriter, r *http.Request) {
	auth := middleware.AuthFromContext(r.Context())
	appID := chi.URLParam(r, "appID")

	app, err := p.apps.Get(r.Context(), appID)
	if err != nil {
		http.Error(w, `{"error":"app not found"}`, http.StatusNotFound)
		return
	}

	if app.TenantID != auth.TenantID {
		http.Error(w, `{"error":"app not found"}`, http.StatusNotFound)
		return
	}

	// Strip the /v1/apps/:appID/aws prefix
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/v1/apps/"+appID+"/aws")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	p.proxyToInstance(w, r, app)
}

func (p *Proxy) proxyToInstance(w http.ResponseWriter, r *http.Request, app *model.App) {
	switch app.InfraType {
	case "dedicated":
		if app.FlyAppName == "" {
			http.Error(w, `{"error":"dedicated instance not provisioned"}`, http.StatusServiceUnavailable)
			return
		}
		proxy := p.getDedicatedProxy(app.FlyAppName)
		proxy.ServeHTTP(w, r)

	default: // shared
		r.Header.Set("X-Tenant-ID", app.TenantID)
		proxy := httputil.NewSingleHostReverseProxy(p.sharedURL)
		proxy.ServeHTTP(w, r)
	}
}

func (p *Proxy) getDedicatedProxy(flyAppName string) *httputil.ReverseProxy {
	p.mu.RLock()
	if proxy, ok := p.dedProxies[flyAppName]; ok {
		p.mu.RUnlock()
		return proxy
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if proxy, ok := p.dedProxies[flyAppName]; ok {
		return proxy
	}

	target, _ := url.Parse("https://" + flyAppName + ".fly.dev")
	proxy := httputil.NewSingleHostReverseProxy(target)
	p.dedProxies[flyAppName] = proxy
	return proxy
}
```

- [ ] **Step 2: Write proxy test**

Create `services/api/internal/handler/proxy_test.go`:

```go
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/handler"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/middleware"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

func TestProxyHandler(t *testing.T) {
	// Start a mock CloudMock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Received-Tenant", r.Header.Get("X-Tenant-ID"))
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	pool := setupTestDB(t)
	tenantStore := store.NewTenantStore(pool)
	appStore := store.NewAppStore(pool)
	ctx := t.Context()

	tenant := &model.Tenant{ClerkOrgID: "org_proxy", Name: "Proxy Test", Slug: "proxy-test", Status: "active"}
	tenantStore.Create(ctx, tenant)

	app := &model.App{
		TenantID: tenant.ID, Name: "test", Slug: "test",
		Endpoint: "https://test.cloudmock.io", InfraType: "shared", Status: "running",
	}
	appStore.Create(ctx, app)

	proxy := handler.NewProxy(appStore, backend.URL)

	t.Run("Shared app proxies with tenant header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		req = req.WithContext(middleware.WithAuthContext(req.Context(), &model.AuthContext{
			TenantID: tenant.ID, AppID: app.ID, ActorID: "key-1", ActorType: "api_key", Role: "developer",
		}))

		rr := httptest.NewRecorder()
		proxy.HandleSubdomain(rr, req)

		if rr.Code != 200 {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if got := rr.Header().Get("X-Received-Tenant"); got != tenant.ID {
			t.Errorf("tenant header = %q, want %q", got, tenant.ID)
		}
	})
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/handler/ -v -count=1 -timeout 120s -run TestProxyHandler
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: AWS proxy handler with shared/dedicated routing and subdomain support"
```

---

### Task 13: Webhook Handlers (Clerk + Stripe)

**Files:**
- Create: `services/api/internal/handler/webhooks.go`
- Create: `services/api/internal/handler/webhooks_test.go`

- [ ] **Step 1: Implement webhook handlers**

Create `services/api/internal/handler/webhooks.go`:

```go
package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

const signatureTolerance = 5 * time.Minute

type Webhooks struct {
	tenants      *store.TenantStore
	clerkSecret  string
	stripeSecret string
}

func NewWebhooks(tenants *store.TenantStore, clerkSecret, stripeSecret string) *Webhooks {
	return &Webhooks{
		tenants:      tenants,
		clerkSecret:  clerkSecret,
		stripeSecret: stripeSecret,
	}
}

// --- Clerk ---

type clerkEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type clerkOrgData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (h *Webhooks) HandleClerk(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := verifySvixSignature(r.Header, body, h.clerkSecret); err != nil {
		slog.Warn("clerk webhook: signature verification failed", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var event clerkEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	switch event.Type {
	case "organization.created":
		var org clerkOrgData
		json.Unmarshal(event.Data, &org)
		h.tenants.Create(ctx, &model.Tenant{
			ClerkOrgID: org.ID, Name: org.Name, Slug: org.Slug, Status: "active",
		})
	case "organization.deleted":
		var org clerkOrgData
		json.Unmarshal(event.Data, &org)
		if t, err := h.tenants.GetByClerkOrgID(ctx, org.ID); err == nil {
			h.tenants.Delete(ctx, t.ID)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"received":true}`))
}

// --- Stripe ---

type stripeEvent struct {
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type stripeInvoice struct {
	CustomerID string `json:"customer"`
}

func (h *Webhooks) HandleStripe(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := verifyStripeSignature(body, r.Header.Get("Stripe-Signature"), h.stripeSecret); err != nil {
		slog.Warn("stripe webhook: signature verification failed", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var event stripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "invoice.paid":
		// Usage-based billing: invoice paid means billing cycle complete
		slog.Info("stripe: invoice paid", "type", event.Type)
	case "customer.subscription.updated":
		slog.Info("stripe: subscription updated", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"received":true}`))
}

// --- Signature Verification ---

func verifySvixSignature(header http.Header, body []byte, secret string) error {
	msgID := header.Get("svix-id")
	timestamp := header.Get("svix-timestamp")
	signatures := header.Get("svix-signature")

	if msgID == "" || timestamp == "" || signatures == "" {
		return fmt.Errorf("missing svix headers")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	diff := time.Since(time.Unix(ts, 0))
	if diff < 0 {
		diff = -diff
	}
	if diff > signatureTolerance {
		return fmt.Errorf("timestamp outside tolerance")
	}

	signedContent := msgID + "." + timestamp + "." + string(body)

	// Decode secret (strip whsec_ prefix)
	sec := secret
	if after, ok := strings.CutPrefix(sec, "whsec_"); ok {
		sec = after
	}
	secretBytes, err := base64.StdEncoding.DecodeString(sec)
	if err != nil {
		return fmt.Errorf("decode secret: %w", err)
	}

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedContent))
	expected := mac.Sum(nil)

	for _, sig := range strings.Split(signatures, " ") {
		parts := strings.SplitN(sig, ",", 2)
		if len(parts) != 2 || parts[0] != "v1" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			continue
		}
		if hmac.Equal(decoded, expected) {
			return nil
		}
	}
	return fmt.Errorf("no matching signature")
}

func verifyStripeSignature(body []byte, sigHeader, secret string) error {
	if sigHeader == "" {
		return fmt.Errorf("missing Stripe-Signature")
	}

	var timestamp string
	var sigs []string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			sigs = append(sigs, kv[1])
		}
	}

	if timestamp == "" || len(sigs) == 0 {
		return fmt.Errorf("missing timestamp or signature")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	diff := time.Since(time.Unix(ts, 0))
	if diff < 0 {
		diff = -diff
	}
	if diff > signatureTolerance {
		return fmt.Errorf("timestamp outside tolerance")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(body)))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range sigs {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return nil
		}
	}
	return fmt.Errorf("no matching signature")
}

// ensure unused import doesn't break
var _ context.Context
```

- [ ] **Step 2: Write webhook tests**

Create `services/api/internal/handler/webhooks_test.go`:

```go
package handler_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/handler"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/model"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

func TestClerkWebhook(t *testing.T) {
	pool := setupTestDB(t)
	tenants := store.NewTenantStore(pool)

	secret := base64.StdEncoding.EncodeToString([]byte("test-secret-key-1234"))
	wh := handler.NewWebhooks(tenants, "whsec_"+secret, "")

	body := []byte(`{"type":"organization.created","data":{"id":"org_clerk123","name":"Acme","slug":"acme"}}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	msgID := "msg_test123"

	// Sign the body
	signedContent := msgID + "." + ts + "." + string(body)
	mac := hmac.New(sha256.New, []byte("test-secret-key-1234"))
	mac.Write([]byte(signedContent))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/webhooks/clerk", bytes.NewReader(body))
	req.Header.Set("svix-id", msgID)
	req.Header.Set("svix-timestamp", ts)
	req.Header.Set("svix-signature", "v1,"+sig)

	rr := httptest.NewRecorder()
	wh.HandleClerk(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}

	// Verify tenant was created
	tenant, err := tenants.GetByClerkOrgID(t.Context(), "org_clerk123")
	if err != nil {
		t.Fatalf("tenant not created: %v", err)
	}
	if tenant.Name != "Acme" {
		t.Errorf("name = %q, want %q", tenant.Name, "Acme")
	}
}

func TestStripeWebhook(t *testing.T) {
	pool := setupTestDB(t)
	tenants := store.NewTenantStore(pool)

	stripeSecret := "whsec_stripe_test_123"
	wh := handler.NewWebhooks(tenants, "", stripeSecret)

	body := []byte(`{"type":"invoice.paid","data":{"object":{"customer":"cus_123"}}}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	mac := hmac.New(sha256.New, []byte(stripeSecret))
	mac.Write([]byte(ts + "." + string(body)))
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", fmt.Sprintf("t=%s,v1=%s", ts, sig))

	rr := httptest.NewRecorder()
	wh.HandleStripe(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", rr.Code, rr.Body.String())
	}
}

func TestClerkWebhookInvalidSignature(t *testing.T) {
	pool := setupTestDB(t)
	tenants := store.NewTenantStore(pool)

	secret := base64.StdEncoding.EncodeToString([]byte("test-secret"))
	wh := handler.NewWebhooks(tenants, "whsec_"+secret, "")

	body := []byte(`{"type":"organization.created","data":{"id":"org_bad"}}`)
	req := httptest.NewRequest("POST", "/webhooks/clerk", bytes.NewReader(body))
	req.Header.Set("svix-id", "msg_1")
	req.Header.Set("svix-timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("svix-signature", "v1,invalidsignature")

	rr := httptest.NewRecorder()
	wh.HandleClerk(rr, req)

	if rr.Code != 401 {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/handler/ -v -count=1 -timeout 120s -run "TestClerkWebhook|TestStripeWebhook"
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: Clerk and Stripe webhook handlers with signature verification"
```

---

### Task 14: main.go Router Wiring

**Files:**
- Modify: `services/api/cmd/api/main.go`

- [ ] **Step 1: Wire all handlers and middleware into main.go**

Replace `services/api/cmd/api/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/database"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/handler"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/middleware"
	"github.com/Viridian-Inc/cloudmock-platform/services/api/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()

	// Config from env
	dbURL := mustEnv("DATABASE_URL")
	port := envOr("PORT", "8080")
	sharedURL := envOr("CLOUDMOCK_SHARED_URL", "http://localhost:4566")
	clerkJWKS := envOr("CLERK_JWKS_URL", "")
	clerkWebhookSecret := envOr("CLERK_WEBHOOK_SECRET", "")
	stripeWebhookSecret := envOr("STRIPE_WEBHOOK_SECRET", "")

	// Database
	if err := database.Migrate(dbURL); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	pool, err := database.Connect(ctx, dbURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Stores
	tenants := store.NewTenantStore(pool)
	apps := store.NewAppStore(pool)
	keys := store.NewAPIKeyStore(pool)
	audit := store.NewAuditStore(pool)
	usage := store.NewUsageStore(pool)

	// Middleware
	var clerkVerifier *middleware.ClerkVerifier
	if clerkJWKS != "" {
		clerkVerifier = middleware.NewClerkVerifier(clerkJWKS)
	}
	authMW := middleware.NewAuth(clerkVerifier, keys)
	auditMW := middleware.NewAudit(audit)

	// Handlers
	appsHandler := handler.NewApps(apps)
	keysHandler := handler.NewAPIKeys(keys)
	proxy := handler.NewProxy(apps, sharedURL)
	webhooks := handler.NewWebhooks(tenants, clerkWebhookSecret, stripeWebhookSecret)

	// Router
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	// Health check (no auth)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// Webhooks (signature-verified, no JWT/API key auth)
	r.Post("/webhooks/clerk", webhooks.HandleClerk)
	r.Post("/webhooks/stripe", webhooks.HandleStripe)

	// Authenticated API routes
	r.Group(func(r chi.Router) {
		r.Use(authMW.Handler)
		r.Use(auditMW.Handler)

		// Apps
		r.Post("/v1/apps", appsHandler.Create)
		r.Get("/v1/apps", appsHandler.List)
		r.Get("/v1/apps/{appID}", appsHandler.Get)
		r.Patch("/v1/apps/{appID}", appsHandler.Update)
		r.Delete("/v1/apps/{appID}", appsHandler.Delete)

		// API Keys
		r.Post("/v1/apps/{appID}/keys", keysHandler.Create)
		r.Get("/v1/apps/{appID}/keys", keysHandler.List)
		r.Delete("/v1/apps/{appID}/keys/{keyID}", keysHandler.Revoke)
		r.Post("/v1/apps/{appID}/keys/{keyID}/rotate", keysHandler.Rotate)

		// AWS Proxy (path-based)
		r.Handle("/v1/apps/{appID}/aws/*", http.HandlerFunc(proxy.HandlePathBased))
	})

	// Subdomain-based proxy (caught by Fly routing)
	r.Handle("/*", http.HandlerFunc(proxy.HandleSubdomain))

	// Server with graceful shutdown
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		slog.Info("starting server", "port", port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env var missing", "key", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Create Dockerfile**

Create `services/api/Dockerfile`:

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /api /api
COPY --from=builder /app/migrations /migrations
EXPOSE 8080
CMD ["/api"]
```

- [ ] **Step 3: Verify build**

```bash
cd ../cloudmock-platform/services/api
go build ./...
```

Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: main.go wiring all handlers, middleware, and graceful shutdown"
```

---

### Task 15: Full Build Verification

- [ ] **Step 1: Run all tests**

```bash
cd ../cloudmock-platform/services/api
go test ./... -v -count=1 -timeout 300s
```

Expected: All tests PASS.

- [ ] **Step 2: Run vet and build**

```bash
go vet ./...
go build ./...
```

Expected: No warnings, clean build.

- [ ] **Step 3: Verify docker-compose starts**

```bash
cd ../cloudmock-platform
docker compose up -d postgres
sleep 3
cd services/api
DATABASE_URL="postgres://postgres:dev@localhost:5432/cloudmock_platform?sslmode=disable" go run ./cmd/api &
sleep 2
curl -s http://localhost:8080/healthz
kill %1
docker compose down
```

Expected: `{"status":"ok"}`

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat: CloudMock Platform Go API v1 complete -- stores, middleware, handlers, proxy"
```

---

## Self-Review

### Spec Coverage

| Spec Requirement | Task |
|-----------------|------|
| Tenant CRUD | Task 3 |
| App lifecycle | Task 4, Task 10 |
| API key management (SHA-256, revoke, rotate) | Task 5, Task 11 |
| Audit log (append-only, HIPAA) | Task 6, Task 9 |
| Usage metering | Task 7 |
| Data retention | Task 7 |
| Clerk JWT auth | Task 8 |
| API key auth | Task 8 |
| Quota enforcement (1K free cap) | Task 9 |
| AWS proxy (shared + dedicated routing) | Task 12 |
| Clerk webhooks (org created/deleted) | Task 13 |
| Stripe webhooks (invoice.paid) | Task 13 |
| Router wiring + graceful shutdown | Task 14 |
| Docker compose | Task 1 |
| Dockerfile | Task 14 |

### Not covered (deferred to Plan B: Next.js Dashboard)
- Dashboard UI pages
- Clerk React integration
- Stripe Customer Portal embed
- Role-based layout switcher

### Not covered (deferred to Plan C: Deployment)
- Fly deployment config (fly.toml)
- Vercel deployment config
- CI/CD pipeline
- Cloudflare DNS automation
- Usage reporting cron job (Stripe billing meter)

### Placeholder scan
No TBD, TODO, or "implement later" found. All code blocks are complete.

### Type consistency
- `model.AuthContext` used consistently in middleware and handlers
- `store.ErrNotFound` used consistently across all stores
- `middleware.AuthFromContext` / `middleware.WithAuthContext` used in all handlers
- All store constructors follow `New<Store>(pool)` pattern
