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
