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
