// Package filestore implements logstore.LogStore by wrapping the in-memory store
// with file-backed persistence. Logs are written to a capped JSON array file for
// recovery on restart, while live-tailing remains fully in-memory.
package filestore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/Viridian-Inc/cloudmock/pkg/logstore"
	logmemory "github.com/Viridian-Inc/cloudmock/pkg/logstore/memory"
)

const defaultCap = 50000

// Store wraps the in-memory log store with file persistence.
type Store struct {
	mem     *logmemory.Store
	mu      sync.Mutex
	path    string
	entries []logstore.LogEntry
	cap     int
}

// New creates a file-backed log store. On creation, it loads existing entries
// from disk and replays them into the in-memory store.
func New(dir string, capacity int) (*Store, error) {
	if capacity <= 0 {
		capacity = defaultCap
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	mem := logmemory.NewStore(capacity)
	s := &Store{
		mem:  mem,
		path: filepath.Join(dir, "logs.json"),
		cap:  capacity,
	}

	// Load existing entries and replay into memory store.
	if data, err := os.ReadFile(s.path); err == nil {
		var entries []logstore.LogEntry
		if json.Unmarshal(data, &entries) == nil {
			s.entries = entries
			for _, e := range entries {
				_ = mem.Write(e)
			}
		}
	}

	return s, nil
}

func (s *Store) Write(entry logstore.LogEntry) error {
	if err := s.mem.Write(entry); err != nil {
		return err
	}
	s.appendAndPersist(entry)
	return nil
}

func (s *Store) WriteBatch(entries []logstore.LogEntry) error {
	if err := s.mem.WriteBatch(entries); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entries...)
	if len(s.entries) > s.cap {
		s.entries = s.entries[len(s.entries)-s.cap:]
	}
	s.persist()
	return nil
}

func (s *Store) Query(opts logstore.QueryOpts) ([]logstore.LogEntry, error) {
	return s.mem.Query(opts)
}

func (s *Store) Tail(ctx context.Context, filter logstore.TailFilter) <-chan logstore.LogEntry {
	return s.mem.Tail(ctx, filter)
}

func (s *Store) Services() ([]string, error) {
	return s.mem.Services()
}

func (s *Store) LevelCounts() (map[string]int, error) {
	return s.mem.LevelCounts()
}

func (s *Store) appendAndPersist(entry logstore.LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
	if len(s.entries) > s.cap {
		s.entries = s.entries[len(s.entries)-s.cap:]
	}
	s.persist()
}

func (s *Store) persist() {
	data, err := json.Marshal(s.entries)
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0644)
}

// Compile-time check.
var _ logstore.LogStore = (*Store)(nil)
