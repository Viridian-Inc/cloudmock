CREATE TABLE regressions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    algorithm       TEXT NOT NULL,
    severity        TEXT NOT NULL,
    confidence      INT NOT NULL,
    service         TEXT NOT NULL,
    action          TEXT,
    deploy_id       UUID REFERENCES deploys(id),
    tenant_id       TEXT,
    title           TEXT NOT NULL,
    before_value    DOUBLE PRECISION NOT NULL,
    after_value     DOUBLE PRECISION NOT NULL,
    change_percent  DOUBLE PRECISION NOT NULL,
    sample_size     BIGINT NOT NULL,
    detected_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    window_before   TSTZRANGE NOT NULL,
    window_after    TSTZRANGE NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active',
    resolved_at     TIMESTAMPTZ
);

CREATE INDEX idx_regressions_service ON regressions(service, detected_at DESC);
CREATE INDEX idx_regressions_deploy ON regressions(deploy_id);
CREATE INDEX idx_regressions_status ON regressions(status) WHERE status = 'active';
