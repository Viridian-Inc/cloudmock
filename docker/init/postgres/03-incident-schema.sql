CREATE TABLE incidents (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status            TEXT NOT NULL DEFAULT 'active',
    severity          TEXT NOT NULL,
    title             TEXT NOT NULL,
    affected_services TEXT[] DEFAULT '{}',
    affected_tenants  TEXT[] DEFAULT '{}',
    alert_count       INT NOT NULL DEFAULT 1,
    root_cause        TEXT,
    related_deploy_id UUID REFERENCES deploys(id),
    first_seen        TIMESTAMPTZ NOT NULL,
    last_seen         TIMESTAMPTZ NOT NULL,
    resolved_at       TIMESTAMPTZ,
    owner             TEXT
);

CREATE INDEX idx_incidents_status ON incidents(status) WHERE status = 'active';
CREATE INDEX idx_incidents_severity ON incidents(severity, first_seen DESC);
