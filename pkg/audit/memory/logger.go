package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/audit"
)

// Logger is an in-memory audit logger backed by a mutex-protected slice.
type Logger struct {
	mu      sync.RWMutex
	entries []audit.Entry
	seq     int64
}

// NewLogger creates a new in-memory audit logger.
func NewLogger() *Logger {
	return &Logger{}
}

// Log appends an entry to the in-memory log. If the entry has no ID, one is
// generated. If the entry has a zero Timestamp, the current time is used.
func (l *Logger) Log(_ context.Context, entry audit.Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.ID == "" {
		l.seq++
		entry.ID = fmt.Sprintf("audit-%d", l.seq)
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	l.entries = append(l.entries, entry)
	return nil
}

// Query returns entries matching the filter, newest first.
func (l *Logger) Query(_ context.Context, filter audit.Filter) ([]audit.Entry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	var result []audit.Entry
	// Iterate backwards for newest-first ordering.
	for i := len(l.entries) - 1; i >= 0 && len(result) < limit; i-- {
		e := l.entries[i]
		if filter.Actor != "" && !strings.EqualFold(e.Actor, filter.Actor) {
			continue
		}
		if filter.Action != "" && !strings.EqualFold(e.Action, filter.Action) {
			continue
		}
		if filter.Resource != "" && !strings.EqualFold(e.Resource, filter.Resource) {
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

// Compile-time interface check.
var _ audit.Logger = (*Logger)(nil)
