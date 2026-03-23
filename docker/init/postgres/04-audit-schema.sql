CREATE TABLE audit_log (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor     TEXT NOT NULL,
    action    TEXT NOT NULL,
    resource  TEXT NOT NULL,
    details   JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_actor ON audit_log(actor, timestamp DESC);
CREATE INDEX idx_audit_action ON audit_log(action, timestamp DESC);
