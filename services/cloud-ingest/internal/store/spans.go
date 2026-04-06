package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Span is a single unit of traced work.
type Span struct {
	Time         time.Time      `json:"time"`
	TraceID      string         `json:"trace_id"`
	SpanID       string         `json:"span_id"`
	ParentSpanID string         `json:"parent_span_id,omitempty"`
	OrgID        string         `json:"org_id"`
	AppID        string         `json:"app_id,omitempty"`
	Environment  string         `json:"environment"`
	Source       string         `json:"source"`
	Service      string         `json:"service"`
	Action       string         `json:"action"`
	Region       string         `json:"region,omitempty"`
	AccountID    string         `json:"account_id,omitempty"`
	RequestID    string         `json:"request_id,omitempty"`
	DurationMs   float64        `json:"duration_ms"`
	StatusCode   int            `json:"status_code,omitempty"`
	ErrorCode    string         `json:"error_code,omitempty"`
	Attributes   map[string]any `json:"attributes,omitempty"`
}

// MetricPoint is one bucket of aggregated service metrics.
type MetricPoint struct {
	Bucket       time.Time `json:"bucket"`
	Service      string    `json:"service"`
	Action       string    `json:"action"`
	RequestCount int64     `json:"request_count"`
	AvgMs        float64   `json:"avg_ms"`
	MaxMs        float64   `json:"max_ms"`
	ErrorCount   int64     `json:"error_count"`
}

// TopologyEdge represents a service-to-service call relationship.
type TopologyEdge struct {
	ParentService string  `json:"parent_service"`
	ChildService  string  `json:"child_service"`
	CallCount     int64   `json:"call_count"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
}

// SpanStore handles persistence of spans to TimescaleDB / Postgres.
type SpanStore struct {
	pool *pgxpool.Pool
}

// NewSpanStore creates a SpanStore backed by the given connection pool.
func NewSpanStore(pool *pgxpool.Pool) *SpanStore {
	return &SpanStore{pool: pool}
}

// spanColumns is the ordered list of columns used for CopyFrom.
var spanColumns = []string{
	"time", "trace_id", "span_id", "parent_span_id",
	"org_id", "app_id", "environment", "source",
	"service", "action", "region", "account_id", "request_id",
	"duration_ms", "status_code", "error_code", "attributes",
}

// InsertBatch inserts a batch of spans using the high-throughput COPY protocol.
func (s *SpanStore) InsertBatch(ctx context.Context, spans []Span) error {
	if len(spans) == 0 {
		return nil
	}

	rows := make([][]any, 0, len(spans))
	for _, sp := range spans {
		var attrJSON []byte
		if sp.Attributes != nil {
			var err error
			attrJSON, err = json.Marshal(sp.Attributes)
			if err != nil {
				return fmt.Errorf("marshal attributes: %w", err)
			}
		}

		// Normalise defaults so DB constraints are satisfied.
		env := sp.Environment
		if env == "" {
			env = "production"
		}
		src := sp.Source
		if src == "" {
			src = "agent-proxy"
		}
		t := sp.Time
		if t.IsZero() {
			t = time.Now().UTC()
		}

		rows = append(rows, []any{
			t,
			sp.TraceID,
			sp.SpanID,
			nullableString(sp.ParentSpanID),
			sp.OrgID,
			nullableString(sp.AppID),
			env,
			src,
			sp.Service,
			sp.Action,
			nullableString(sp.Region),
			nullableString(sp.AccountID),
			nullableString(sp.RequestID),
			sp.DurationMs,
			nullableInt(sp.StatusCode),
			nullableString(sp.ErrorCode),
			nullableBytes(attrJSON),
		})
	}

	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	_, err = conn.Conn().CopyFrom(
		ctx,
		pgx.Identifier{"spans"},
		spanColumns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy from: %w", err)
	}
	return nil
}

// QueryByOrg returns the most recent spans for an org, optionally filtered by
// environment, since a given time, up to limit rows.
func (s *SpanStore) QueryByOrg(ctx context.Context, orgID, env string, since time.Time, limit int) ([]Span, error) {
	if limit <= 0 || limit > 10000 {
		limit = 100
	}

	query := `
SELECT time, trace_id, span_id, COALESCE(parent_span_id,''),
       org_id, COALESCE(app_id,''), environment, source,
       service, action,
       COALESCE(region,''), COALESCE(account_id,''), COALESCE(request_id,''),
       duration_ms, COALESCE(status_code,0), COALESCE(error_code,''),
       COALESCE(attributes,'{}')
FROM spans
WHERE org_id = $1
  AND ($2 = '' OR environment = $2)
  AND time >= $3
ORDER BY time DESC
LIMIT $4`

	rows, err := s.pool.Query(ctx, query, orgID, env, since, limit)
	if err != nil {
		return nil, fmt.Errorf("query by org: %w", err)
	}
	defer rows.Close()

	return collectSpans(rows)
}

// QueryByTrace returns all spans belonging to the given trace ID.
func (s *SpanStore) QueryByTrace(ctx context.Context, traceID string) ([]Span, error) {
	query := `
SELECT time, trace_id, span_id, COALESCE(parent_span_id,''),
       org_id, COALESCE(app_id,''), environment, source,
       service, action,
       COALESCE(region,''), COALESCE(account_id,''), COALESCE(request_id,''),
       duration_ms, COALESCE(status_code,0), COALESCE(error_code,''),
       COALESCE(attributes,'{}')
FROM spans
WHERE trace_id = $1
ORDER BY time ASC`

	rows, err := s.pool.Query(ctx, query, traceID)
	if err != nil {
		return nil, fmt.Errorf("query by trace: %w", err)
	}
	defer rows.Close()

	return collectSpans(rows)
}

// QueryMetrics reads pre-aggregated per-minute metrics from the TimescaleDB
// continuous aggregate view. Falls back to querying the raw spans table if
// the view does not exist (plain Postgres deployments).
func (s *SpanStore) QueryMetrics(ctx context.Context, orgID, env, service string, start, end time.Time) ([]MetricPoint, error) {
	query := `
SELECT bucket, service, action,
       request_count, avg_ms, max_ms, error_count
FROM service_metrics_1m
WHERE org_id = $1
  AND ($2 = '' OR environment = $2)
  AND ($3 = '' OR service = $3)
  AND bucket >= $4
  AND bucket < $5
ORDER BY bucket ASC`

	rows, err := s.pool.Query(ctx, query, orgID, env, service, start, end)
	if err != nil {
		// Fallback: aggregate directly from spans.
		return s.queryMetricsFallback(ctx, orgID, env, service, start, end)
	}
	defer rows.Close()

	var pts []MetricPoint
	for rows.Next() {
		var p MetricPoint
		if err := rows.Scan(&p.Bucket, &p.Service, &p.Action,
			&p.RequestCount, &p.AvgMs, &p.MaxMs, &p.ErrorCount); err != nil {
			return nil, fmt.Errorf("scan metric point: %w", err)
		}
		pts = append(pts, p)
	}
	return pts, rows.Err()
}

func (s *SpanStore) queryMetricsFallback(ctx context.Context, orgID, env, service string, start, end time.Time) ([]MetricPoint, error) {
	query := `
SELECT
    date_trunc('minute', time) AS bucket,
    service, action,
    COUNT(*) AS request_count,
    AVG(duration_ms) AS avg_ms,
    MAX(duration_ms) AS max_ms,
    COUNT(*) FILTER (WHERE error_code IS NOT NULL AND error_code != '') AS error_count
FROM spans
WHERE org_id = $1
  AND ($2 = '' OR environment = $2)
  AND ($3 = '' OR service = $3)
  AND time >= $4
  AND time < $5
GROUP BY bucket, service, action
ORDER BY bucket ASC`

	rows, err := s.pool.Query(ctx, query, orgID, env, service, start, end)
	if err != nil {
		return nil, fmt.Errorf("query metrics fallback: %w", err)
	}
	defer rows.Close()

	var pts []MetricPoint
	for rows.Next() {
		var p MetricPoint
		if err := rows.Scan(&p.Bucket, &p.Service, &p.Action,
			&p.RequestCount, &p.AvgMs, &p.MaxMs, &p.ErrorCount); err != nil {
			return nil, fmt.Errorf("scan metric fallback: %w", err)
		}
		pts = append(pts, p)
	}
	return pts, rows.Err()
}

// QueryTopology derives service-to-service edges by joining parent and child
// spans within the requested time window.
func (s *SpanStore) QueryTopology(ctx context.Context, orgID, env string, window time.Duration) ([]TopologyEdge, error) {
	since := time.Now().UTC().Add(-window)

	query := `
SELECT
    parent.service AS parent_service,
    child.service  AS child_service,
    COUNT(*)       AS call_count,
    AVG(child.duration_ms) AS avg_latency_ms
FROM spans child
JOIN spans parent
    ON child.parent_span_id = parent.span_id
    AND child.trace_id = parent.trace_id
WHERE child.org_id = $1
  AND ($2 = '' OR child.environment = $2)
  AND child.time >= $3
  AND child.parent_span_id IS NOT NULL
  AND child.parent_span_id != ''
  AND parent.service != child.service
GROUP BY parent.service, child.service
ORDER BY call_count DESC`

	rows, err := s.pool.Query(ctx, query, orgID, env, since)
	if err != nil {
		return nil, fmt.Errorf("query topology: %w", err)
	}
	defer rows.Close()

	var edges []TopologyEdge
	for rows.Next() {
		var e TopologyEdge
		if err := rows.Scan(&e.ParentService, &e.ChildService, &e.CallCount, &e.AvgLatencyMs); err != nil {
			return nil, fmt.Errorf("scan topology edge: %w", err)
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

// collectSpans scans a pgx.Rows result into a []Span slice.
func collectSpans(rows pgx.Rows) ([]Span, error) {
	var spans []Span
	for rows.Next() {
		var sp Span
		var attrRaw []byte
		if err := rows.Scan(
			&sp.Time, &sp.TraceID, &sp.SpanID, &sp.ParentSpanID,
			&sp.OrgID, &sp.AppID, &sp.Environment, &sp.Source,
			&sp.Service, &sp.Action,
			&sp.Region, &sp.AccountID, &sp.RequestID,
			&sp.DurationMs, &sp.StatusCode, &sp.ErrorCode,
			&attrRaw,
		); err != nil {
			return nil, fmt.Errorf("scan span: %w", err)
		}
		if len(attrRaw) > 0 {
			if err := json.Unmarshal(attrRaw, &sp.Attributes); err != nil {
				return nil, fmt.Errorf("unmarshal attributes: %w", err)
			}
		}
		spans = append(spans, sp)
	}
	return spans, rows.Err()
}

// nullableString returns nil for empty strings (maps to SQL NULL).
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullableInt returns nil for zero ints.
func nullableInt(i int) any {
	if i == 0 {
		return nil
	}
	return i
}

// nullableBytes returns nil for empty byte slices.
func nullableBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
