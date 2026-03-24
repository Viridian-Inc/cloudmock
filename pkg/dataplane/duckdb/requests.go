package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// RequestStore implements dataplane.RequestReader and dataplane.RequestWriter
// backed by DuckDB. Request entries are stored in the denormalized spans table.
type RequestStore struct {
	db *sql.DB
}

// NewRequestStore creates a RequestStore using the given Client's database.
func NewRequestStore(c *Client) *RequestStore {
	return &RequestStore{db: c.DB()}
}

// Write inserts a single request entry as a span row.
func (s *RequestStore) Write(ctx context.Context, entry dataplane.RequestEntry) error {
	durationNs := int64(entry.Latency.Nanoseconds())
	if durationNs == 0 && entry.LatencyMs > 0 {
		durationNs = int64(entry.LatencyMs * 1e6)
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

	metaJSON, err := marshalMap(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	headersJSON, err := marshalMap(entry.RequestHeaders)
	if err != nil {
		return fmt.Errorf("marshal headers: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `INSERT INTO spans (
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
		entry.StatusCode, entry.Error,
		entry.TenantID, entry.OrgID, entry.UserID,
		entry.MemAllocKB, entry.Goroutines,
		metaJSON, headersJSON,
		entry.RequestBody, entry.ResponseBody,
	)
	return err
}

// Query returns request entries matching the filter.
func (s *RequestStore) Query(ctx context.Context, filter dataplane.RequestFilter) ([]dataplane.RequestEntry, error) {
	var (
		where []string
		args  []any
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
		args = append(args, int64(filter.MinLatencyMs*1e6))
	}
	if filter.MaxLatencyMs > 0 {
		where = append(where, "duration_ns <= ?")
		args = append(args, int64(filter.MaxLatencyMs*1e6))
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

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query requests: %w", err)
	}
	defer rows.Close()

	var results []dataplane.RequestEntry
	for rows.Next() {
		r, err := scanSpanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan request: %w", err)
		}
		results = append(results, spanRowToRequestEntry(r))
	}
	return results, rows.Err()
}

// GetByID returns a single request entry by span_id.
func (s *RequestStore) GetByID(ctx context.Context, id string) (*dataplane.RequestEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT trace_id, span_id, parent_span_id,
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
	r, err := scanSpanRow(rows)
	if err != nil {
		return nil, fmt.Errorf("scan request: %w", err)
	}
	entry := spanRowToRequestEntry(r)
	return &entry, rows.Err()
}

// scanSpanRow scans a full span row from a *sql.Rows into a spanRow.
func scanSpanRow(rows *sql.Rows) (spanRow, error) {
	var r spanRow
	var metaJSON, headersJSON sql.NullString
	err := rows.Scan(
		&r.TraceID, &r.SpanID, &r.ParentSpanID,
		&r.StartTime, &r.EndTime, &r.DurationNs,
		&r.ServiceName, &r.Action, &r.Method, &r.Path,
		&r.StatusCode, &r.Error,
		&r.TenantID, &r.OrgID, &r.UserID,
		&r.MemAllocKB, &r.Goroutines,
		&metaJSON, &headersJSON,
		&r.RequestBody, &r.ResponseBody,
	)
	if err != nil {
		return r, err
	}
	r.Metadata = unmarshalMap(metaJSON.String)
	r.RequestHeaders = unmarshalMap(headersJSON.String)
	return r, nil
}

// spanRowToRequestEntry converts an internal spanRow to a dataplane.RequestEntry.
func spanRowToRequestEntry(r spanRow) dataplane.RequestEntry {
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
		StatusCode:     r.StatusCode,
		Latency:        latency,
		LatencyMs:      float64(r.DurationNs) / 1e6,
		Error:          r.Error,
		MemAllocKB:     r.MemAllocKB,
		Goroutines:     r.Goroutines,
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
