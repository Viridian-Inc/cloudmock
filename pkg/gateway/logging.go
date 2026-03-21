package gateway

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// RequestEntry holds data about a single request processed by the gateway.
type RequestEntry struct {
	Timestamp  time.Time     `json:"timestamp"`
	Service    string        `json:"service"`
	Action     string        `json:"action"`
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency_ns"`
	CallerID   string        `json:"caller_id"`
	Error      string        `json:"error,omitempty"`
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

// Recent returns up to limit entries, newest first.
// If service is non-empty, only entries matching that service are returned.
func (rl *RequestLog) Recent(service string, limit int) []RequestEntry {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if limit <= 0 {
		limit = rl.count
	}

	var result []RequestEntry
	for i := 0; i < rl.count && len(result) < limit; i++ {
		idx := (rl.pos - 1 - i + rl.size) % rl.size
		e := rl.entries[idx]
		if service == "" || e.Service == service {
			result = append(result, e)
		}
	}
	return result
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

// responseRecorder wraps http.ResponseWriter to capture the status code.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware wraps a gateway handler and records request data.
func LoggingMiddleware(next http.Handler, log *RequestLog, stats *RequestStats) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		svcName := detectServiceFromRequest(r)
		action := detectActionFromRequest(r)

		latency := time.Since(start)

		entry := RequestEntry{
			Timestamp:  start,
			Service:    svcName,
			Action:     action,
			Method:     r.Method,
			Path:       r.URL.Path,
			StatusCode: rec.statusCode,
			Latency:    latency,
			CallerID:   extractCallerID(r),
		}

		log.Add(entry)
		if svcName != "" {
			stats.Increment(svcName)
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
func extractCallerID(r *http.Request) string {
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
