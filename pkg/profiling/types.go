package profiling

import "time"

// StackFrame represents a single frame in a call stack.
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Module   string `json:"module,omitempty"`
}

// SpanStack captures the call stack at a named point.
type SpanStack struct {
	Point  string       `json:"point"`
	Frames []StackFrame `json:"frames"`
}

// Profile represents a captured runtime profile (CPU, heap, goroutine).
type Profile struct {
	ID         string        `json:"id"`
	Service    string        `json:"service"`
	Type       string        `json:"type"`
	FilePath   string        `json:"-"`
	CapturedAt time.Time     `json:"captured_at"`
	Duration   time.Duration `json:"duration,omitempty"`
	Size       int64         `json:"size_bytes"`
}
