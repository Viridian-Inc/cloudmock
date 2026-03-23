CREATE TABLE IF NOT EXISTS spans (
    trace_id         FixedString(32),
    span_id          FixedString(16),
    parent_span_id   FixedString(16),
    start_time       DateTime64(9, 'UTC'),
    end_time         DateTime64(9, 'UTC'),
    duration_ns      UInt64,
    service_name     LowCardinality(String),
    action           LowCardinality(String),
    method           LowCardinality(String),
    path             String,
    status_code      UInt16,
    error            String,
    tenant_id        String,
    org_id           String,
    user_id          String,
    mem_alloc_kb     Float64,
    goroutines       UInt32,
    metadata         Map(String, String),
    request_headers  Map(String, String),
    request_body     String,
    response_body    String,
    _date            Date DEFAULT toDate(start_time)
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(_date))
ORDER BY (service_name, action, start_time, trace_id)
TTL _date + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

ALTER TABLE spans ADD INDEX IF NOT EXISTS idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 4;
ALTER TABLE spans ADD INDEX IF NOT EXISTS idx_error error TYPE tokenbf_v1(10240, 3, 0) GRANULARITY 4;
ALTER TABLE spans ADD INDEX IF NOT EXISTS idx_status status_code TYPE minmax GRANULARITY 4;
ALTER TABLE spans ADD INDEX IF NOT EXISTS idx_user_id user_id TYPE bloom_filter(0.01) GRANULARITY 4;
ALTER TABLE spans ADD INDEX IF NOT EXISTS idx_org_id org_id TYPE bloom_filter(0.01) GRANULARITY 4;
