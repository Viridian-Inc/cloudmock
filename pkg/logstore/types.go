package logstore

import (
	"context"
	"time"
)

// LogEntry represents a single log record.
type LogEntry struct {
	ID         string            `json:"id"`
	Timestamp  time.Time         `json:"timestamp"`
	Level      string            `json:"level"`   // "debug", "info", "warn", "error", "fatal"
	Message    string            `json:"message"`
	Service    string            `json:"service"`
	TraceID    string            `json:"trace_id"` // correlation with traces
	SpanID     string            `json:"span_id"`
	Source     string            `json:"source"` // "sdk", "rum", "cloudwatch", "otlp"
	Attributes map[string]string `json:"attributes"`
}

// QueryOpts specifies filters for querying logs.
type QueryOpts struct {
	Search    string    // full-text search
	Level     string    // filter by level
	Service   string    // filter by service
	TraceID   string    // filter by trace
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// TailFilter specifies filters for live-tailing logs.
type TailFilter struct {
	Level   string
	Service string
	Search  string
}

// LogStore defines the interface for persisting and querying logs.
type LogStore interface {
	// Write persists a single log entry.
	Write(entry LogEntry) error

	// WriteBatch persists multiple log entries.
	WriteBatch(entries []LogEntry) error

	// Query returns log entries matching the given options.
	Query(opts QueryOpts) ([]LogEntry, error)

	// Tail returns a channel that streams matching log entries in real time.
	Tail(ctx context.Context, filter TailFilter) <-chan LogEntry

	// Services returns the list of distinct service names that have emitted logs.
	Services() ([]string, error)

	// LevelCounts returns the count of log entries by level.
	LevelCounts() (map[string]int, error)
}
