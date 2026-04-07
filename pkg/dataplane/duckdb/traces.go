package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
)

// TraceStore implements dataplane.TraceReader and dataplane.TraceWriter
// backed by DuckDB.
type TraceStore struct {
	db *sql.DB
}

// NewTraceStore creates a TraceStore using the given Client's database.
func NewTraceStore(c *Client) *TraceStore {
	return &TraceStore{db: c.DB()}
}

// WriteSpans inserts spans into the spans table.
func (s *TraceStore) WriteSpans(ctx context.Context, spans []*dataplane.Span) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO spans (
		trace_id, span_id, parent_span_id,
		start_time, end_time, duration_ns,
		service_name, action, method, path,
		status_code, error,
		tenant_id, org_id, user_id,
		mem_alloc_kb, goroutines,
		metadata, request_headers,
		request_body, response_body
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, sp := range spans {
		durationNs := sp.DurationNs
		if durationNs == 0 {
			durationNs = uint64(sp.EndTime.Sub(sp.StartTime).Nanoseconds())
		}

		metaJSON, err := marshalMap(sp.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		headersJSON, err := marshalMap(sp.ReqHeaders)
		if err != nil {
			return fmt.Errorf("marshal headers: %w", err)
		}

		if _, err := stmt.ExecContext(ctx,
			sp.TraceID, sp.SpanID, sp.ParentSpanID,
			sp.StartTime, sp.EndTime, int64(durationNs),
			sp.Service, sp.Action, sp.Method, sp.Path,
			sp.StatusCode, sp.Error,
			sp.TenantID, sp.OrgID, sp.UserID,
			sp.MemAllocKB, sp.Goroutines,
			metaJSON, headersJSON,
			sp.ReqBody, sp.RespBody,
		); err != nil {
			return fmt.Errorf("insert span: %w", err)
		}
	}
	return tx.Commit()
}

// selectSpans queries spans for a given trace_id ordered by start_time.
func (s *TraceStore) selectSpans(ctx context.Context, traceID string) ([]spanRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT trace_id, span_id, parent_span_id,
			start_time, end_time, duration_ns,
			service_name, action, method, path,
			status_code, error,
			tenant_id, org_id, user_id,
			mem_alloc_kb, goroutines,
			metadata, request_headers,
			request_body, response_body
		FROM spans WHERE trace_id = ? ORDER BY start_time`, traceID)
	if err != nil {
		return nil, fmt.Errorf("query spans: %w", err)
	}
	defer rows.Close()

	var result []spanRow
	for rows.Next() {
		var r spanRow
		var metaJSON, headersJSON sql.NullString
		if err := rows.Scan(
			&r.TraceID, &r.SpanID, &r.ParentSpanID,
			&r.StartTime, &r.EndTime, &r.DurationNs,
			&r.ServiceName, &r.Action, &r.Method, &r.Path,
			&r.StatusCode, &r.Error,
			&r.TenantID, &r.OrgID, &r.UserID,
			&r.MemAllocKB, &r.Goroutines,
			&metaJSON, &headersJSON,
			&r.RequestBody, &r.ResponseBody,
		); err != nil {
			return nil, fmt.Errorf("scan span: %w", err)
		}
		r.Metadata = unmarshalMap(metaJSON.String)
		r.RequestHeaders = unmarshalMap(headersJSON.String)
		result = append(result, r)
	}
	return result, rows.Err()
}

// Get retrieves all spans for a trace and assembles a parent/child tree.
func (s *TraceStore) Get(ctx context.Context, traceID string) (*dataplane.TraceContext, error) {
	spanRows, err := s.selectSpans(ctx, traceID)
	if err != nil {
		return nil, err
	}
	if len(spanRows) == 0 {
		return nil, dataplane.ErrNotFound
	}

	// Build map of spanID -> TraceContext.
	nodeMap := make(map[string]*dataplane.TraceContext, len(spanRows))
	for _, r := range spanRows {
		dur := time.Duration(r.DurationNs)
		tc := &dataplane.TraceContext{
			TraceID:      r.TraceID,
			SpanID:       r.SpanID,
			ParentSpanID: r.ParentSpanID,
			Service:      r.ServiceName,
			Action:       r.Action,
			Method:       r.Method,
			Path:         r.Path,
			StartTime:    r.StartTime,
			EndTime:      r.EndTime,
			Duration:     dur,
			DurationMs:   float64(r.DurationNs) / 1e6,
			StatusCode:   int(r.StatusCode),
			Error:        r.Error,
			Metadata:     r.Metadata,
		}
		nodeMap[r.SpanID] = tc
	}

	// Link children to parents; find root.
	var root *dataplane.TraceContext
	for _, tc := range nodeMap {
		if tc.ParentSpanID == "" {
			root = tc
			continue
		}
		if parent, ok := nodeMap[tc.ParentSpanID]; ok {
			parent.Children = append(parent.Children, tc)
		}
	}

	// Fallback: if no span has empty parent, pick the first span.
	if root == nil {
		root = nodeMap[spanRows[0].SpanID]
	}

	return root, nil
}

// Search returns TraceSummary entries matching the filter.
func (s *TraceStore) Search(ctx context.Context, filter dataplane.TraceFilter) ([]dataplane.TraceSummary, error) {
	var (
		where []string
		args  []any
	)
	if filter.Service != "" {
		where = append(where, "service_name = ?")
		args = append(args, filter.Service)
	}
	if filter.HasError != nil && *filter.HasError {
		where = append(where, "error != ''")
	}
	if filter.TenantID != "" {
		where = append(where, "tenant_id = ?")
		args = append(args, filter.TenantID)
	}

	query := "SELECT trace_id, span_id, parent_span_id, start_time, duration_ns, service_name, action, method, path, status_code, error FROM spans"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY start_time DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	type traceAgg struct {
		summary   dataplane.TraceSummary
		spanCount int
		seen      bool
	}
	traceOrder := []string{}
	traces := map[string]*traceAgg{}

	for rows.Next() {
		var (
			tid, sid, pid string
			startTime     time.Time
			durationNs    int64
			svc, act      string
			method, path  string
			statusCode    int
			errStr        string
		)
		if err := rows.Scan(&tid, &sid, &pid, &startTime, &durationNs, &svc, &act, &method, &path, &statusCode, &errStr); err != nil {
			return nil, fmt.Errorf("scan search row: %w", err)
		}

		agg, ok := traces[tid]
		if !ok {
			agg = &traceAgg{}
			traces[tid] = agg
			traceOrder = append(traceOrder, tid)
		}
		agg.spanCount++
		// Use root span (empty parent) to set summary fields.
		if pid == "" && !agg.seen {
			agg.seen = true
			agg.summary = dataplane.TraceSummary{
				TraceID:     tid,
				RootService: svc,
				RootAction:  act,
				Method:      method,
				Path:        path,
				DurationMs:  float64(durationNs) / 1e6,
				StatusCode:  statusCode,
				HasError:    errStr != "",
				StartTime:   startTime,
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	var results []dataplane.TraceSummary
	for _, tid := range traceOrder {
		agg := traces[tid]
		agg.summary.SpanCount = agg.spanCount
		// If no root span was found, fill in trace ID at minimum.
		if !agg.seen {
			agg.summary.TraceID = tid
		}
		results = append(results, agg.summary)
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

// Timeline returns a flat waterfall view of spans in a trace.
func (s *TraceStore) Timeline(ctx context.Context, traceID string) ([]dataplane.TimelineSpan, error) {
	spanRows, err := s.selectSpans(ctx, traceID)
	if err != nil {
		return nil, err
	}
	if len(spanRows) == 0 {
		return nil, dataplane.ErrNotFound
	}

	// Earliest start time is the baseline.
	earliest := spanRows[0].StartTime
	for _, r := range spanRows[1:] {
		if r.StartTime.Before(earliest) {
			earliest = r.StartTime
		}
	}

	// Build parent->children map for depth computation.
	childMap := make(map[string][]string)
	for _, r := range spanRows {
		if r.ParentSpanID != "" {
			childMap[r.ParentSpanID] = append(childMap[r.ParentSpanID], r.SpanID)
		}
	}

	// Compute depth for each span via BFS from roots.
	depthMap := make(map[string]int)
	for _, r := range spanRows {
		if r.ParentSpanID == "" {
			depthMap[r.SpanID] = 0
		}
	}
	// Iterative depth assignment.
	changed := true
	for changed {
		changed = false
		for _, r := range spanRows {
			if _, ok := depthMap[r.SpanID]; ok {
				continue
			}
			if parentDepth, ok := depthMap[r.ParentSpanID]; ok {
				depthMap[r.SpanID] = parentDepth + 1
				changed = true
			}
		}
	}

	result := make([]dataplane.TimelineSpan, len(spanRows))
	for i, r := range spanRows {
		result[i] = dataplane.TimelineSpan{
			SpanID:        r.SpanID,
			ParentSpanID:  r.ParentSpanID,
			Service:       r.ServiceName,
			Action:        r.Action,
			StartOffsetMs: float64(r.StartTime.Sub(earliest).Nanoseconds()) / 1e6,
			DurationMs:    float64(r.DurationNs) / 1e6,
			StatusCode:    int(r.StatusCode),
			Error:         r.Error,
			Depth:         depthMap[r.SpanID],
			Metadata:      r.Metadata,
		}
	}
	return result, nil
}

// Compile-time interface checks.
var (
	_ dataplane.TraceReader = (*TraceStore)(nil)
	_ dataplane.TraceWriter = (*TraceStore)(nil)
)
