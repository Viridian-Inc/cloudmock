package audit

import (
	"context"
	"time"
)

// Entry represents a single audit log record for a mutating API action.
type Entry struct {
	ID        string                 `json:"id"`
	Actor     string                 `json:"actor"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Logger is the interface for recording and querying audit log entries.
type Logger interface {
	Log(ctx context.Context, entry Entry) error
	Query(ctx context.Context, filter Filter) ([]Entry, error)
}

// Filter controls which entries are returned by Query.
type Filter struct {
	Actor    string
	Action   string
	Resource string
	Limit    int
}
