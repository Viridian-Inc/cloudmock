CREATE TABLE data_retention (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    resource_type  TEXT NOT NULL CHECK (resource_type IN ('audit_log', 'request_log', 'state_snapshot')),
    retention_days INT NOT NULL DEFAULT 365,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, resource_type)
);

CREATE INDEX idx_data_retention_tenant_id ON data_retention(tenant_id);
