CREATE TABLE IF NOT EXISTS preferences (
    namespace  TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      JSONB NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (namespace, key)
);
