package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// TraceStore implements dataplane.TraceReader and dataplane.TraceWriter
// backed by ClickHouse.
type TraceStore struct {
	conn driver.Conn
}

// NewTraceStore creates a TraceStore using the given Client's connection.
func NewTraceStore(c *Client) *TraceStore {
	return &TraceStore{conn: c.Conn()}
}

// spanRow is the internal representation matching the ClickHouse spans table.
type spanRow struct {
	TraceID        string            `ch:"trace_id"`
	SpanID         string            `ch:"span_id"`
	ParentSpanID   string            `ch:"parent_span_id"`
	StartTime      time.Time         `ch:"start_time"`
	EndTime        time.Time         `ch:"end_time"`
	DurationNs     uint64            `ch:"duration_ns"`
	ServiceName    string            `ch:"service_name"`
	Action         string            `ch:"action"`
	Method         string            `ch:"method"`
	Path           string            `ch:"path"`
	StatusCode     uint16            `ch:"status_code"`
	Error          string            `ch:"error"`
	TenantID       string            `ch:"tenant_id"`
	OrgID          string            `ch:"org_id"`
	UserID         string            `ch:"user_id"`
	MemAllocKB     float64           `ch:"mem_alloc_kb"`
	Goroutines     uint32            `ch:"goroutines"`
	Metadata       map[string]string `ch:"metadata"`
	RequestHeaders map[string]string `ch:"request_headers"`
	RequestBody    string            `ch:"request_body"`
	ResponseBody   string            `ch:"response_body"`
}

// trimNulls strips null bytes that ClickHouse FixedString columns pad with.
func trimNulls(s string) string {
	return strings.TrimRight(s, "\x00")
}

// WriteSpans batch-inserts spans into the spans table.
func (s *TraceStore) WriteSpans(ctx context.Context, spans []*dataplane.Span) error {
	batch, err := s.conn.PrepareBatch(ctx, `INSERT INTO spans (
		trace_id, span_id, parent_span_id,
		start_time, end_time, duration_ns,
		service_name, action, method, path,
		status_code, error,
		tenant_id, org_id, user_id,
		mem_alloc_kb, goroutines,
		metadata, request_headers,
		request_body, response_body
	)`)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}
	for _, sp := range spans {
		durationNs := sp.DurationNs
		if durationNs == 0 {
			durationNs = uint64(sp.EndTime.Sub(sp.StartTime).Nanoseconds())
		}
		meta := sp.Metadata
		if meta == nil {
			meta = map[string]string{}
		}
		headers := sp.ReqHeaders
		if headers == nil {
			headers = map[string]string{}
		}
		if err := batch.Append(
			sp.TraceID, sp.SpanID, sp.ParentSpanID,
			sp.StartTime, sp.EndTime, durationNs,
			sp.Service, sp.Action, sp.Method, sp.Path,
			uint16(sp.StatusCode), sp.Error,
			sp.TenantID, sp.OrgID, sp.UserID,
			sp.MemAllocKB, sp.Goroutines,
			meta, headers,
			sp.ReqBody, sp.RespBody,
		); err != nil {
			return fmt.Errorf("append span: %w", err)
		}
	}
	return batch.Send()
}

// selectSpans queries spans for a given trace_id ordered by start_time.
func (s *TraceStore) selectSpans(ctx context.Context, traceID string) ([]spanRow, error) {
	rows, err := s.conn.Query(ctx,
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
		if err := rows.ScanStruct(&r); err != nil {
			return nil, fmt.Errorf("scan span: %w", err)
		}
		r.TraceID = trimNulls(r.TraceID)
		r.SpanID = trimNulls(r.SpanID)
		r.ParentSpanID = trimNulls(r.ParentSpanID)
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
		args  []interface{}
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
	// We don't LIMIT at the SQL level because we need to group by trace_id.
	// Instead, we'll limit the output after grouping.

	rows, err := s.conn.Query(ctx, query, args...)
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
			tid, sid, pid  string
			startTime      time.Time
			durationNs     uint64
			svc, act       string
			method, path   string
			statusCode     uint16
			errStr         string
		)
		if err := rows.Scan(&tid, &sid, &pid, &startTime, &durationNs, &svc, &act, &method, &path, &statusCode, &errStr); err != nil {
			return nil, fmt.Errorf("scan search row: %w", err)
		}
		tid = trimNulls(tid)
		pid = trimNulls(pid)

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
				StatusCode:  int(statusCode),
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
