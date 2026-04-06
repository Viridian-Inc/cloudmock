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
