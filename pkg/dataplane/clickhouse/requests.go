package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// RequestStore implements dataplane.RequestReader and dataplane.RequestWriter
// backed by ClickHouse. Request entries are stored in the denormalized spans
// table.
type RequestStore struct {
	conn driver.Conn
}

// NewRequestStore creates a RequestStore using the given Client's connection.
func NewRequestStore(c *Client) *RequestStore {
	return &RequestStore{conn: c.Conn()}
}

// Write inserts a single request entry as a span row.
func (s *RequestStore) Write(ctx context.Context, entry dataplane.RequestEntry) error {
	durationNs := uint64(entry.Latency.Nanoseconds())
	if durationNs == 0 && entry.LatencyMs > 0 {
		durationNs = uint64(entry.LatencyMs * 1e6)
	}

	startTime := entry.Timestamp
	endTime := startTime.Add(entry.Latency)

	spanID := entry.SpanID
	if spanID == "" {
		spanID = entry.ID
	}

	meta := map[string]string{}
	if entry.CallerID != "" {
		meta["caller_id"] = entry.CallerID
	}
	if entry.Level != "" {
		meta["level"] = entry.Level
	}

	headers := entry.RequestHeaders
	if headers == nil {
		headers = map[string]string{}
	}

	return s.conn.Exec(ctx, `INSERT INTO spans (
		trace_id, span_id, parent_span_id,
		start_time, end_time, duration_ns,
		service_name, action, method, path,
		status_code, error,
		tenant_id, org_id, user_id,
		mem_alloc_kb, goroutines,
		metadata, request_headers,
		request_body, response_body
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.TraceID, spanID, "",
		startTime, endTime, durationNs,
		entry.Service, entry.Action, entry.Method, entry.Path,
		uint16(entry.StatusCode), entry.Error,
		entry.TenantID, entry.OrgID, entry.UserID,
		entry.MemAllocKB, uint32(entry.Goroutines),
		meta, headers,
		entry.RequestBody, entry.ResponseBody,
	)
}

// Query returns request entries matching the filter.
func (s *RequestStore) Query(ctx context.Context, filter dataplane.RequestFilter) ([]dataplane.RequestEntry, error) {
	var (
		where []string
		args  []interface{}
	)

	if filter.Service != "" {
		where = append(where, "service_name = ?")
		args = append(args, filter.Service)
	}
	if filter.Path != "" {
		where = append(where, "path = ?")
		args = append(args, filter.Path)
	}
	if filter.Method != "" {
		where = append(where, "method = ?")
		args = append(args, filter.Method)
	}
	if filter.Action != "" {
		where = append(where, "action = ?")
		args = append(args, filter.Action)
	}
	if filter.ErrorOnly {
		where = append(where, "error != ''")
	}
	if filter.TraceID != "" {
		where = append(where, "trace_id = ?")
		args = append(args, filter.TraceID)
	}
	if filter.TenantID != "" {
		where = append(where, "tenant_id = ?")
		args = append(args, filter.TenantID)
	}
	if filter.OrgID != "" {
		where = append(where, "org_id = ?")
		args = append(args, filter.OrgID)
	}
	if filter.UserID != "" {
		where = append(where, "user_id = ?")
		args = append(args, filter.UserID)
	}
	if filter.MinLatencyMs > 0 {
		where = append(where, "duration_ns >= ?")
		args = append(args, uint64(filter.MinLatencyMs*1e6))
	}
	if filter.MaxLatencyMs > 0 {
		where = append(where, "duration_ns <= ?")
		args = append(args, uint64(filter.MaxLatencyMs*1e6))
	}
	if !filter.From.IsZero() {
		where = append(where, "start_time >= ?")
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		where = append(where, "start_time <= ?")
		args = append(args, filter.To)
	}

	query := `SELECT trace_id, span_id, parent_span_id,
		start_time, end_time, duration_ns,
		service_name, action, method, path,
		status_code, error,
		tenant_id, org_id, user_id,
		mem_alloc_kb, goroutines,
		metadata, request_headers,
		request_body, response_body
	FROM spans`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY start_time DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query requests: %w", err)
	}
	defer rows.Close()

	var results []dataplane.RequestEntry
	for rows.Next() {
		var r spanRow
		if err := rows.ScanStruct(&r); err != nil {
			return nil, fmt.Errorf("scan request: %w", err)
		}
		results = append(results, spanRowToRequestEntry(r))
	}
	return results, rows.Err()
}

// GetByID returns a single request entry by span_id.
func (s *RequestStore) GetByID(ctx context.Context, id string) (*dataplane.RequestEntry, error) {
	rows, err := s.conn.Query(ctx, `SELECT trace_id, span_id, parent_span_id,
		start_time, end_time, duration_ns,
		service_name, action, method, path,
		status_code, error,
		tenant_id, org_id, user_id,
		mem_alloc_kb, goroutines,
		metadata, request_headers,
		request_body, response_body
	FROM spans WHERE span_id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("get by id: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, dataplane.ErrNotFound
	}
	var r spanRow
	if err := rows.ScanStruct(&r); err != nil {
		return nil, fmt.Errorf("scan request: %w", err)
	}
	entry := spanRowToRequestEntry(r)
	return &entry, rows.Err()
}

// spanRowToRequestEntry converts an internal spanRow to a dataplane.RequestEntry.
func spanRowToRequestEntry(r spanRow) dataplane.RequestEntry {
	r.TraceID = trimNulls(r.TraceID)
	r.SpanID = trimNulls(r.SpanID)
	r.ParentSpanID = trimNulls(r.ParentSpanID)

	latency := time.Duration(r.DurationNs)
	entry := dataplane.RequestEntry{
		ID:             r.SpanID,
		TraceID:        r.TraceID,
		SpanID:         r.SpanID,
		Timestamp:      r.StartTime,
		Service:        r.ServiceName,
		Action:         r.Action,
		Method:         r.Method,
		Path:           r.Path,
		StatusCode:     int(r.StatusCode),
		Latency:        latency,
		LatencyMs:      float64(r.DurationNs) / 1e6,
		Error:          r.Error,
		MemAllocKB:     r.MemAllocKB,
		Goroutines:     int(r.Goroutines),
		RequestHeaders: r.RequestHeaders,
		RequestBody:    r.RequestBody,
		ResponseBody:   r.ResponseBody,
		TenantID:       r.TenantID,
		OrgID:          r.OrgID,
		UserID:         r.UserID,
	}
	// Restore CallerID and Level from metadata.
	if r.Metadata != nil {
		entry.CallerID = r.Metadata["caller_id"]
		entry.Level = r.Metadata["level"]
	}
	return entry
}

// Compile-time interface checks.
var (
	_ dataplane.RequestReader = (*RequestStore)(nil)
	_ dataplane.RequestWriter = (*RequestStore)(nil)
)
