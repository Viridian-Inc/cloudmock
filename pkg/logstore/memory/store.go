package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/logstore"
)

// Store is an in-memory implementation of logstore.LogStore using a circular
// buffer and a broadcast pattern for live tailing.
type Store struct {
	mu   sync.RWMutex
	buf  []logstore.LogEntry
	cap  int
	pos  int
	full bool

	// Tail subscribers.
	subMu   sync.Mutex
	subs    map[int]chan logstore.LogEntry
	nextSub int
}

// NewStore creates a new in-memory log store with the given capacity.
func NewStore(capacity int) *Store {
	if capacity <= 0 {
		capacity = 50000
	}
	return &Store{
		buf:  make([]logstore.LogEntry, capacity),
		cap:  capacity,
		subs: make(map[int]chan logstore.LogEntry),
	}
}

// Write persists a single log entry.
func (s *Store) Write(entry logstore.LogEntry) error {
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("log-%d", time.Now().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	s.mu.Lock()
	s.buf[s.pos] = entry
	s.pos = (s.pos + 1) % s.cap
	if s.pos == 0 {
		s.full = true
	}
	s.mu.Unlock()

	// Broadcast to tail subscribers.
	s.subMu.Lock()
	for _, ch := range s.subs {
		select {
		case ch <- entry:
		default:
			// Drop if subscriber is slow.
		}
	}
	s.subMu.Unlock()

	return nil
}

// WriteBatch persists multiple log entries.
func (s *Store) WriteBatch(entries []logstore.LogEntry) error {
	for i := range entries {
		if err := s.Write(entries[i]); err != nil {
			return err
		}
	}
	return nil
}

// Query returns log entries matching the given options.
func (s *Store) Query(opts logstore.QueryOpts) ([]logstore.LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	count := s.cap
	if !s.full {
		count = s.pos
	}

	var result []logstore.LogEntry

	// Walk backwards from most recent.
	for i := 0; i < count && len(result) < limit; i++ {
		idx := (s.pos - 1 - i + s.cap) % s.cap
		e := s.buf[idx]
		if e.ID == "" {
			continue // empty slot
		}
		if !s.matchesQuery(e, opts) {
			continue
		}
		result = append(result, e)
	}

	// Reverse so oldest-first within the result set.
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result, nil
}

func (s *Store) matchesQuery(e logstore.LogEntry, opts logstore.QueryOpts) bool {
	if opts.Level != "" && !strings.EqualFold(e.Level, opts.Level) {
		return false
	}
	if opts.Service != "" && e.Service != opts.Service {
		return false
	}
	if opts.TraceID != "" && e.TraceID != opts.TraceID {
		return false
	}
	if !opts.StartTime.IsZero() && e.Timestamp.Before(opts.StartTime) {
		return false
	}
	if !opts.EndTime.IsZero() && e.Timestamp.After(opts.EndTime) {
		return false
	}
	if opts.Search != "" && !strings.Contains(strings.ToLower(e.Message), strings.ToLower(opts.Search)) {
		return false
	}
	return true
}

// Tail returns a channel that streams matching log entries in real time.
// The channel is closed when ctx is cancelled.
func (s *Store) Tail(ctx context.Context, filter logstore.TailFilter) <-chan logstore.LogEntry {
	ch := make(chan logstore.LogEntry, 64)

	s.subMu.Lock()
	id := s.nextSub
	s.nextSub++
	// We create a raw channel and filter in a goroutine.
	raw := make(chan logstore.LogEntry, 64)
	s.subs[id] = raw
	s.subMu.Unlock()

	go func() {
		defer close(ch)
		defer func() {
			s.subMu.Lock()
			delete(s.subs, id)
			s.subMu.Unlock()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case entry, ok := <-raw:
				if !ok {
					return
				}
				if !matchesTailFilter(entry, filter) {
					continue
				}
				select {
				case ch <- entry:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}

func matchesTailFilter(e logstore.LogEntry, f logstore.TailFilter) bool {
	if f.Level != "" && !strings.EqualFold(e.Level, f.Level) {
		return false
	}
	if f.Service != "" && e.Service != f.Service {
		return false
	}
	if f.Search != "" && !strings.Contains(strings.ToLower(e.Message), strings.ToLower(f.Search)) {
		return false
	}
	return true
}

// Services returns the distinct service names.
func (s *Store) Services() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	count := s.cap
	if !s.full {
		count = s.pos
	}
	for i := 0; i < count; i++ {
		e := s.buf[i]
		if e.Service != "" {
			seen[e.Service] = true
		}
	}

	result := make([]string, 0, len(seen))
	for svc := range seen {
		result = append(result, svc)
	}
	sort.Strings(result)
	return result, nil
}

// LevelCounts returns the count of entries by level.
func (s *Store) LevelCounts() (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]int)
	count := s.cap
	if !s.full {
		count = s.pos
	}
	for i := 0; i < count; i++ {
		e := s.buf[i]
		if e.Level != "" {
			counts[e.Level]++
		}
	}
	return counts, nil
}
