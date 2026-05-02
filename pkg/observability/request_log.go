package observability

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

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
	MemAllocKB     int64             `json:"mem_alloc_kb,omitempty"` // heap allocation at request time
	Goroutines     int               `json:"goroutines,omitempty"`   // goroutine count at request time
	RequestHeaders map[string]string `json:"request_headers,omitempty"`
	RequestBody    string            `json:"request_body,omitempty"`
	ResponseBody   string            `json:"response_body,omitempty"`
}

// requestIDCounter is a monotonic counter for generating unique request IDs.
var requestIDCounter atomic.Int64

// NextRequestID returns a fresh monotonic int from the shared request-ID
// counter. Used by middleware that constructs RequestEntry IDs.
func NextRequestID() int64 {
	return requestIDCounter.Add(1)
}

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

// RequestBroadcaster is an optional interface for broadcasting request events.
type RequestBroadcaster interface {
	Broadcast(eventType string, data any)
}
