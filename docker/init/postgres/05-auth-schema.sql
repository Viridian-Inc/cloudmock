CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'viewer',
    tenant_id     TEXT,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT now()
);
