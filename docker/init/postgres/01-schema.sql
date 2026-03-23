CREATE TABLE services (
    name             TEXT PRIMARY KEY,
    service_type     TEXT NOT NULL,
    group_name       TEXT,
    description      TEXT,
    owner            TEXT,
    repo_url         TEXT,
    created_at       TIMESTAMPTZ DEFAULT now(),
    updated_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE topology_edges (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_service   TEXT NOT NULL REFERENCES services(name),
    target_service   TEXT NOT NULL REFERENCES services(name),
    edge_type        TEXT NOT NULL,
    first_seen       TIMESTAMPTZ NOT NULL,
    last_seen        TIMESTAMPTZ NOT NULL,
    request_count    BIGINT DEFAULT 0,
    UNIQUE (source_service, target_service, edge_type)
);

CREATE TABLE deploys (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service          TEXT NOT NULL REFERENCES services(name),
    version          TEXT NOT NULL,
    commit_sha       TEXT,
    author           TEXT,
    description      TEXT,
    deployed_at      TIMESTAMPTZ DEFAULT now(),
    metadata         JSONB DEFAULT '{}'
);

CREATE TABLE slo_rules (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service          TEXT NOT NULL,  -- no FK, supports wildcard "*"
    action           TEXT NOT NULL DEFAULT '*',
    route            TEXT,
    tenant_tier      TEXT,
    p50_ms           DOUBLE PRECISION,
    p95_ms           DOUBLE PRECISION,
    p99_ms           DOUBLE PRECISION,
    error_rate       DOUBLE PRECISION,
    annotation       TEXT,
    active           BOOLEAN DEFAULT true,
    created_at       TIMESTAMPTZ DEFAULT now(),
    created_by       TEXT
);

CREATE TABLE slo_rule_history (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id          UUID NOT NULL REFERENCES slo_rules(id),
    change_type      TEXT NOT NULL,
    old_values       JSONB,
    new_values       JSONB,
    changed_by       TEXT,
    changed_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE saved_views (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    filters          JSONB NOT NULL,
    created_by       TEXT,
    created_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE config (
    key              TEXT PRIMARY KEY,
    value            JSONB NOT NULL,
    updated_by       TEXT,
    updated_at       TIMESTAMPTZ DEFAULT now()
);
