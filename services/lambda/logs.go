package lambda

import (
	"sync"
	"time"
)

// LambdaLogEntry represents a single log line from a Lambda execution.
type LambdaLogEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	FunctionName string    `json:"functionName"`
	RequestID    string    `json:"requestId"`
	Message      string    `json:"message"`
	Stream       string    `json:"stream"` // "stdout" or "stderr"
}

// LogBuffer is a thread-safe circular buffer of Lambda log entries per function.
type LogBuffer struct {
	mu         sync.RWMutex
	entries    []LambdaLogEntry
	pos        int
	count      int
	capacity   int
	onEmit     func(LambdaLogEntry) // optional callback for broadcasting
}

// NewLogBuffer creates a LogBuffer with the given per-function capacity.
func NewLogBuffer(capacity int) *LogBuffer {
	if capacity <= 0 {
		capacity = 500
	}
	return &LogBuffer{
		entries:  make([]LambdaLogEntry, capacity),
		capacity: capacity,
	}
}

// SetOnEmit sets a callback invoked for every log entry added.
func (lb *LogBuffer) SetOnEmit(fn func(LambdaLogEntry)) {
	lb.mu.Lock()
	lb.onEmit = fn
	lb.mu.Unlock()
}

// Add appends a log entry to the buffer.
func (lb *LogBuffer) Add(entry LambdaLogEntry) {
	lb.mu.Lock()
	lb.entries[lb.pos] = entry
	lb.pos = (lb.pos + 1) % lb.capacity
	if lb.count < lb.capacity {
		lb.count++
	}
	onEmit := lb.onEmit
	lb.mu.Unlock()

	if onEmit != nil {
		onEmit(entry)
	}
}

// Recent returns up to limit entries, newest first.
// If functionName is non-empty, only matching entries are returned.
func (lb *LogBuffer) Recent(functionName string, limit int) []LambdaLogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if limit <= 0 {
		limit = lb.count
	}

	var result []LambdaLogEntry
	for i := 0; i < lb.count && len(result) < limit; i++ {
		idx := (lb.pos - 1 - i + lb.capacity) % lb.capacity
		e := lb.entries[idx]
		if functionName == "" || e.FunctionName == functionName {
			result = append(result, e)
		}
	}
	return result
}
