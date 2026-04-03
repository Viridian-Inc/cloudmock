package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neureaux/cloudmock/pkg/dataplane"
	"github.com/neureaux/cloudmock/pkg/profiling"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// memStatsCache caches runtime.MemStats to avoid calling ReadMemStats on every request.
// ReadMemStats is expensive (~1ms) and triggers a STW pause; sampling every N seconds is sufficient.
var (
	memStatsMu       sync.Mutex
	memStatsCache    runtime.MemStats
	memStatsLastRead time.Time
	memStatsInterval = 5 * time.Second
)

func cachedMemAllocKB() int64 {
	memStatsMu.Lock()
	if time.Since(memStatsLastRead) > memStatsInterval {
		runtime.ReadMemStats(&memStatsCache)
		memStatsLastRead = time.Now()
	}
	alloc := int64(memStatsCache.Alloc / 1024)
	memStatsMu.Unlock()
	return alloc
}

// goroutineCountCache caches runtime.NumGoroutine() to avoid the scheduler scan on every request.
var (
	goroutineCountMu       sync.Mutex
	goroutineCountCached   int
	goroutineCountLastRead time.Time
	goroutineCountInterval = 2 * time.Second
)

func cachedNumGoroutine() int {
	goroutineCountMu.Lock()
	if time.Since(goroutineCountLastRead) > goroutineCountInterval {
		goroutineCountCached = runtime.NumGoroutine()
		goroutineCountLastRead = time.Now()
	}
	n := goroutineCountCached
	goroutineCountMu.Unlock()
	return n
}

// maxBodyCapture is the maximum number of bytes captured for request/response bodies.
const maxBodyCapture = 10 * 1024

// RequestEntry holds data about a single request processed by the gateway.
type RequestEntry struct {
	ID             string            `json:"id"`
	TraceID        string            `json:"trace_id,omitempty"`
	SpanID         string            `json:"span_id,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
	Service        string            `json:"service"`
	Action         string            `json:"action"`
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	StatusCode     int               `json:"status_code"`
	Latency        time.Duration     `json:"latency_ns"`
	LatencyMs      float64           `json:"latency_ms"`
	CallerID       string            `json:"caller_id"`
	Error          string            `json:"error,omitempty"`
	Level          string            `json:"level,omitempty"` // "app" (user-facing) or "infra" (AWS SDK calls to cloudmock)
	MemAllocKB     int64             `json:"mem_alloc_kb,omitempty"`   // heap allocation at request time
	Goroutines     int               `json:"goroutines,omitempty"`     // goroutine count at request time
	RequestHeaders map[string]string `json:"request_headers,omitempty"`
	RequestBody    string            `json:"request_body,omitempty"`
	ResponseBody   string            `json:"response_body,omitempty"`
}

// requestIDCounter is a simple monotonic counter for generating unique request IDs.
var requestIDCounter atomic.Int64

// RequestLog is a thread-safe circular buffer of recent request entries.
type RequestLog struct {
	mu      sync.RWMutex
	entries []RequestEntry
	pos     int
	size    int
	count   int
}

// NewRequestLog creates a RequestLog with the given capacity.
func NewRequestLog(capacity int) *RequestLog {
	if capacity <= 0 {
		capacity = 1000
	}
	return &RequestLog{
		entries: make([]RequestEntry, capacity),
		size:    capacity,
	}
}

// Add appends an entry to the circular buffer.
func (rl *RequestLog) Add(entry RequestEntry) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.entries[rl.pos] = entry
	rl.pos = (rl.pos + 1) % rl.size
	if rl.count < rl.size {
		rl.count++
	}
}

// RequestFilter defines filtering criteria for request log queries.
type RequestFilter struct {
	Service      string
	Path         string
	Method       string
	CallerID     string
	Action       string
	ErrorOnly    bool
	TraceID      string
	Level        string // "app" or "infra" — empty means all
	Limit        int
	TenantID     string
	OrgID        string
	UserID       string
	MinLatencyMs float64
	MaxLatencyMs float64
	From         time.Time
	To           time.Time
}

// Recent returns up to limit entries, newest first.
// If service is non-empty, only entries matching that service are returned.
func (rl *RequestLog) Recent(service string, limit int) []RequestEntry {
	return rl.RecentFiltered(RequestFilter{Service: service, Limit: limit})
}

// RecentFiltered returns entries matching all non-empty filter fields.
func (rl *RequestLog) RecentFiltered(f RequestFilter) []RequestEntry {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	limit := f.Limit
	if limit <= 0 {
		limit = rl.count
	}

	var result []RequestEntry
	for i := 0; i < rl.count && len(result) < limit; i++ {
		idx := (rl.pos - 1 - i + rl.size) % rl.size
		e := rl.entries[idx]
		if f.Service != "" && e.Service != f.Service {
			continue
		}
		if f.Path != "" && !strings.HasPrefix(e.Path, f.Path) {
			continue
		}
		if f.Method != "" && !strings.EqualFold(e.Method, f.Method) {
			continue
		}
		if f.CallerID != "" && e.CallerID != f.CallerID {
			continue
		}
		if f.Action != "" && e.Action != f.Action {
			continue
		}
		if f.ErrorOnly && e.StatusCode < 400 {
			continue
		}
		if f.TraceID != "" && e.TraceID != f.TraceID {
			continue
		}
		if f.Level != "" && e.Level != f.Level {
			continue
		}
		if f.TenantID != "" && e.RequestHeaders["X-Tenant-Id"] != f.TenantID {
			continue
		}
		if f.OrgID != "" && e.RequestHeaders["X-Enterprise-Id"] != f.OrgID {
			continue
		}
		if f.UserID != "" && e.RequestHeaders["X-User-Id"] != f.UserID {
			continue
		}
		if f.MinLatencyMs > 0 && e.LatencyMs < f.MinLatencyMs {
			continue
		}
		if f.MaxLatencyMs > 0 && e.LatencyMs > f.MaxLatencyMs {
			continue
		}
		if !f.From.IsZero() && e.Timestamp.Before(f.From) {
			continue
		}
		if !f.To.IsZero() && e.Timestamp.After(f.To) {
			continue
		}
		result = append(result, e)
	}
	return result
}

// GetByID returns the entry with the given ID, or nil if not found.
func (rl *RequestLog) GetByID(id string) *RequestEntry {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	for i := 0; i < rl.count; i++ {
		idx := (rl.pos - 1 - i + rl.size) % rl.size
		if rl.entries[idx].ID == id {
			e := rl.entries[idx]
			return &e
		}
	}
	return nil
}

// RequestStats tracks per-service request counts using atomic counters.
type RequestStats struct {
	mu     sync.RWMutex
	counts map[string]*atomic.Int64
}

// NewRequestStats creates an empty RequestStats tracker.
func NewRequestStats() *RequestStats {
	return &RequestStats{
		counts: make(map[string]*atomic.Int64),
	}
}

// Increment increments the counter for the given service.
func (rs *RequestStats) Increment(svcName string) {
	rs.mu.RLock()
	counter, ok := rs.counts[svcName]
	rs.mu.RUnlock()
	if ok {
		counter.Add(1)
		return
	}
	rs.mu.Lock()
	counter, ok = rs.counts[svcName]
	if !ok {
		counter = &atomic.Int64{}
		rs.counts[svcName] = counter
	}
	rs.mu.Unlock()
	counter.Add(1)
}

// Snapshot returns a map of service name to request count.
func (rs *RequestStats) Snapshot() map[string]int64 {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	out := make(map[string]int64, len(rs.counts))
	for k, v := range rs.counts {
		out[k] = v.Load()
	}
	return out
}

// responseRecorder wraps http.ResponseWriter to capture the status code and response body.
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	body        bytes.Buffer
	captureBody bool
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if rr.captureBody && rr.body.Len() < maxBodyCapture {
		remaining := maxBodyCapture - rr.body.Len()
		if len(b) > remaining {
			rr.body.Write(b[:remaining])
		} else {
			rr.body.Write(b)
		}
	}
	return rr.ResponseWriter.Write(b)
}

// RequestBroadcaster is an optional interface for broadcasting request events.
type RequestBroadcaster interface {
	Broadcast(eventType string, data any)
}

// LoggingMiddlewareOpts holds optional dependencies for LoggingMiddleware.
// OnRequestFunc is called after each request is logged with the service name,
// latency in milliseconds, and HTTP status code. Used for anomaly detection.
type OnRequestFunc func(service string, latencyMs float64, statusCode int)

type LoggingMiddlewareOpts struct {
	Broadcaster   RequestBroadcaster
	TraceStore    *TraceStore
	SLOEngine     *SLOEngine
	DataPlane     *dataplane.DataPlane
	OnRequest     OnRequestFunc
	CaptureStacks bool // if true, capture call stacks per-request into trace store (expensive)
}

// LoggingMiddleware wraps a gateway handler and records request data.
func LoggingMiddleware(next http.Handler, log *RequestLog, stats *RequestStats, broadcasters ...RequestBroadcaster) http.Handler {
	return LoggingMiddlewareWithOpts(next, log, stats, LoggingMiddlewareOpts{
		Broadcaster: firstBroadcaster(broadcasters),
	})
}

func firstBroadcaster(bb []RequestBroadcaster) RequestBroadcaster {
	if len(bb) > 0 {
		return bb[0]
	}
	return nil
}

// LoggingMiddlewareWithOpts wraps a gateway handler and records request data with full options.
func LoggingMiddlewareWithOpts(next http.Handler, log *RequestLog, stats *RequestStats, opts LoggingMiddlewareOpts) http.Handler {
	productionMode := opts.DataPlane != nil && opts.DataPlane.Mode == "production"
	hasTraceStore := opts.TraceStore != nil
	captureStacks := opts.CaptureStacks
	hasSLO := opts.SLOEngine != nil
	hasBroadcaster := opts.Broadcaster != nil
	hasOnRequest := opts.OnRequest != nil

	// Lightweight mode: when there is no log, stats, trace store, SLO, broadcaster,
	// or OnRequest handler, skip all observability overhead.
	lightweight := log == nil && stats == nil && !hasTraceStore && !hasSLO && !hasBroadcaster && !hasOnRequest && !productionMode

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fast path for health checks — no observability overhead.
		if r.URL.Path == "/_cloudmock/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Lightweight mode skips all observability.
		if lightweight {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		// Extract or generate trace ID. Span ID is only needed when a trace store
		// or broadcaster/production mode is active; skip the random generation otherwise.
		traceID := r.Header.Get("X-Cloudmock-Trace-Id")
		if traceID == "" {
			traceID = r.Header.Get("X-Amz-Trace-Id")
		}
		if traceID == "" {
			traceID = GenerateTraceID()
		}
		var spanID string
		var parentSpanID string
		if hasTraceStore || hasBroadcaster || productionMode {
			spanID = GenerateSpanID()
			parentSpanID = r.Header.Get("X-Cloudmock-Parent-Span-Id")
		}

		// Set trace headers on the response so callers can correlate.
		w.Header().Set("X-Cloudmock-Trace-Id", traceID)
		if spanID != "" {
			w.Header().Set("X-Cloudmock-Span-Id", spanID)
		}

		// Capture request headers — only when we have subscribers that need them.
		var reqHeaders map[string]string
		if hasBroadcaster || productionMode {
			reqHeaders = make(map[string]string, len(r.Header))
			for k := range r.Header {
				reqHeaders[k] = r.Header.Get(k)
			}
		}

		// Capture request body (first maxBodyCapture bytes), then restore it.
		// Only capture when we have subscribers that display body content.
		var reqBody string
		if (hasBroadcaster || productionMode) && r.Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, int64(maxBodyCapture)+1))
			if err == nil {
				if len(bodyBytes) > maxBodyCapture {
					reqBody = string(bodyBytes[:maxBodyCapture])
				} else {
					reqBody = string(bodyBytes)
				}
				// Restore the body so downstream handlers can read it.
				remaining, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes), bytes.NewReader(remaining)))
			}
		}

		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			captureBody:    hasBroadcaster || productionMode,
		}
		next.ServeHTTP(rec, r)

		svcName := detectServiceFromRequest(r)
		action := detectActionFromRequest(r)

		latency := time.Since(start)
		latencyMs := float64(latency.Nanoseconds()) / 1e6

		counter := requestIDCounter.Add(1)

		var errMsg string
		if rec.statusCode >= 400 {
			errMsg = fmt.Sprintf("HTTP %d", rec.statusCode)
		}

		entry := RequestEntry{
			ID:             fmt.Sprintf("%d-%d", start.UnixNano(), counter),
			TraceID:        traceID,
			SpanID:         spanID,
			Timestamp:      start,
			Service:        svcName,
			Action:         action,
			Method:         r.Method,
			Path:           r.URL.Path,
			StatusCode:     rec.statusCode,
			Latency:        latency,
			LatencyMs:      latencyMs,
			CallerID:       extractCallerID(r),
			Error:          errMsg,
			Level:          "infra", // AWS SDK calls to cloudmock gateway
			MemAllocKB:     cachedMemAllocKB(),
			Goroutines:     cachedNumGoroutine(),
			RequestHeaders: reqHeaders,
			RequestBody:    reqBody,
			ResponseBody:   rec.body.String(),
		}

		if productionMode {
			// Production mode: write request via DataPlane, skip local stores.
			if opts.DataPlane.RequestW != nil {
				dpEntry := dataplane.RequestEntry{
					ID:             entry.ID,
					TraceID:        entry.TraceID,
					SpanID:         entry.SpanID,
					Timestamp:      entry.Timestamp,
					Service:        entry.Service,
					Action:         entry.Action,
					Method:         entry.Method,
					Path:           entry.Path,
					StatusCode:     entry.StatusCode,
					Latency:        entry.Latency,
					LatencyMs:      entry.LatencyMs,
					CallerID:       entry.CallerID,
					Error:          entry.Error,
					Level:          entry.Level,
					MemAllocKB:     float64(entry.MemAllocKB),
					Goroutines:     entry.Goroutines,
					RequestHeaders: entry.RequestHeaders,
					RequestBody:    entry.RequestBody,
					ResponseBody:   entry.ResponseBody,
				}
				_ = opts.DataPlane.RequestW.Write(r.Context(), dpEntry)
			}

			// Emit an OTel span for each request.
			tracer := otel.Tracer("cloudmock-gateway")
			_, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, svcName))
			span.SetAttributes(
				attribute.String("service.name", svcName),
				attribute.String("service.action", action),
				attribute.String("http.method", r.Method),
				attribute.String("http.path", r.URL.Path),
				attribute.Int("http.status_code", rec.statusCode),
			)
			if tenantID := r.Header.Get("X-Tenant-Id"); tenantID != "" {
				span.SetAttributes(attribute.String("tenant_id", tenantID))
			}
			if errMsg != "" {
				span.SetAttributes(attribute.String("error", errMsg))
				span.SetStatus(codes.Error, errMsg)
			}
			span.End()
		} else {
			// Local mode: write directly to in-memory stores.
			if log != nil {
				log.Add(entry)
			}
			if stats != nil && svcName != "" {
				stats.Increment(svcName)
			}

			// Store trace context.
			if hasTraceStore {
				endTime := time.Now()
				// Capture distributed context from headers
				metadata := extractTraceMetadata(r)

				// Capture call stacks only when explicitly enabled — it's expensive
				// (~15μs: runtime.Callers + frame iteration + json.Marshal per request).
				if captureStacks {
					stacks := []profiling.SpanStack{profiling.CaptureStack("handler_entry", 2)}
					stackJSON, _ := json.Marshal(stacks)
					if metadata == nil {
						metadata = make(map[string]string)
					}
					metadata["stacks"] = string(stackJSON)
				}

				trace := &TraceContext{
					TraceID:      traceID,
					SpanID:       spanID,
					ParentSpanID: parentSpanID,
					Service:      svcName,
					Action:       action,
					Method:       r.Method,
					Path:         r.URL.Path,
					StartTime:    start,
					EndTime:      endTime,
					Duration:     latency,
					DurationMs:   latencyMs,
					StatusCode:   rec.statusCode,
					Error:        errMsg,
					Metadata:     metadata,
				}
				opts.TraceStore.Add(trace)
			}

			// Record SLO metrics.
			if hasSLO {
				opts.SLOEngine.Record(svcName, action, latencyMs, rec.statusCode)
			}
		}

		// Broadcast request event for SSE clients — always runs regardless of mode.
		if hasBroadcaster {
			opts.Broadcaster.Broadcast("request", entry)
		}

		// Feed request metrics to anomaly detector or similar consumers.
		if hasOnRequest && svcName != "" {
			opts.OnRequest(svcName, latencyMs, rec.statusCode)
		}
	})
}

// detectServiceFromRequest extracts the service name without importing routing to avoid a cycle.
func detectServiceFromRequest(r *http.Request) string {
	// Use the same logic as routing.DetectService but inline to avoid circular imports.
	if auth := r.Header.Get("Authorization"); auth != "" {
		if svc := serviceFromAuth(auth); svc != "" {
			return svc
		}
	}
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		return serviceFromTargetHeader(target)
	}
	return ""
}

// detectActionFromRequest extracts the action name.
func detectActionFromRequest(r *http.Request) string {
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		for i := len(target) - 1; i >= 0; i-- {
			if target[i] == '.' {
				return target[i+1:]
			}
		}
	}
	return r.URL.Query().Get("Action")
}

// extractCallerID extracts the access key ID from the Authorization header.
// extractTraceMetadata captures feature flags, cache behavior, policy decisions,
// and other distributed context from request headers for trace propagation.
func extractTraceMetadata(r *http.Request) map[string]string {
	meta := make(map[string]string)
	// Capture context propagation headers
	contextHeaders := []string{
		"x-feature-flag", "x-feature-flags",
		"x-cache-status", "x-cache-hit",
		"x-tenant-id", "x-enterprise-id",
		"x-user-id", "x-contact-id",
		"x-policy-decision", "x-authz-result",
		"x-request-id", "x-correlation-id",
		"x-environment", "x-deployment-id",
	}
	for _, h := range contextHeaders {
		if v := r.Header.Get(h); v != "" {
			meta[h] = v
		}
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

func extractCallerID(r *http.Request) string {
	// Prefer explicit source header from SDK-instrumented Lambda functions
	// e.g. X-Cloudmock-Source: autotend-order-handler
	if src := r.Header.Get("X-Cloudmock-Source"); src != "" {
		return src
	}

	// Fall back to User-Agent which AWS SDKs set to "aws-sdk-nodejs/3.x" etc.
	// Some custom SDK configs include the function name
	if ua := r.Header.Get("User-Agent"); ua != "" {
		// Check for Lambda function name pattern in User-Agent
		// e.g. "aws-sdk-nodejs/3.x autotend-order-handler"
		if strings.Contains(ua, "autotend-") {
			for _, part := range strings.Fields(ua) {
				if strings.HasPrefix(part, "autotend-") {
					return part
				}
			}
		}
	}

	// Fall back to access key from Authorization header
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Credential="
	idx := 0
	for i := 0; i <= len(auth)-len(prefix); i++ {
		if auth[i:i+len(prefix)] == prefix {
			idx = i + len(prefix)
			break
		}
	}
	if idx == 0 {
		return ""
	}
	rest := auth[idx:]
	for i, c := range rest {
		if c == '/' {
			return rest[:i]
		}
	}
	return rest
}

// serviceFromAuth extracts the service from an AWS4 Authorization header.
func serviceFromAuth(auth string) string {
	const prefix = "Credential="
	idx := -1
	for i := 0; i <= len(auth)-len(prefix); i++ {
		if auth[i:i+len(prefix)] == prefix {
			idx = i + len(prefix)
			break
		}
	}
	if idx < 0 {
		return ""
	}
	rest := auth[idx:]
	// Find end of credential value
	for i, c := range rest {
		if c == ',' || c == ' ' {
			rest = rest[:i]
			break
		}
	}
	// AKID/date/region/service/aws4_request — split by '/'
	slashCount := 0
	start := 0
	for i, c := range rest {
		if c == '/' {
			slashCount++
			if slashCount == 3 {
				start = i + 1
			}
			if slashCount == 4 {
				return rest[start:i]
			}
		}
	}
	return ""
}

// serviceFromTargetHeader extracts the service from X-Amz-Target.
func serviceFromTargetHeader(target string) string {
	dot := -1
	for i, c := range target {
		if c == '.' {
			dot = i
			break
		}
	}
	svc := target
	if dot >= 0 {
		svc = target[:dot]
	}
	under := -1
	for i, c := range svc {
		if c == '_' {
			under = i
			break
		}
	}
	if under >= 0 {
		svc = svc[:under]
	}
	// lowercase
	b := make([]byte, len(svc))
	for i := range svc {
		c := svc[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
