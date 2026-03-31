CREATE TABLE IF NOT EXISTS tenants (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clerk_org_id          TEXT UNIQUE NOT NULL,
    name                  TEXT NOT NULL,
    slug                  TEXT UNIQUE NOT NULL,
    stripe_customer_id    TEXT,
    stripe_subscription_id TEXT,
    tier                  TEXT NOT NULL DEFAULT 'free',
    status                TEXT NOT NULL DEFAULT 'active',
    fly_machine_id        TEXT,
    fly_app_name          TEXT,
    request_count         BIGINT DEFAULT 0,
    request_limit         BIGINT DEFAULT 0,
    data_retention_days   INT DEFAULT 30,
    created_at            TIMESTAMPTZ DEFAULT now(),
    updated_at            TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS usage_records (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID REFERENCES tenants(id) ON DELETE CASCADE,
    period_start       TIMESTAMPTZ NOT NULL,
    period_end         TIMESTAMPTZ NOT NULL,
    request_count      BIGINT NOT NULL,
    total_cost         NUMERIC(10,6),
    reported_to_stripe BOOLEAN DEFAULT false,
    created_at         TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_usage_records_tenant ON usage_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_period ON usage_records(period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_clerk_org ON tenants(clerk_org_id);
