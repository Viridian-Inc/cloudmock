CREATE TABLE webhooks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url        TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'generic',
    events     TEXT[] NOT NULL DEFAULT '{}',
    headers    JSONB DEFAULT '{}',
    active     BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_webhooks_active ON webhooks(active) WHERE active = true;
