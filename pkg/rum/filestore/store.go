// Package filestore implements rum.RUMStore by wrapping the in-memory store
// with file-backed persistence. Events are stored as a capped JSON array file.
package filestore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/neureaux/cloudmock/pkg/rum"
	rummemory "github.com/neureaux/cloudmock/pkg/rum/memory"
)

// Store wraps the in-memory RUM store with file persistence.
type Store struct {
	mem    *rummemory.Store
	mu     sync.Mutex
	path   string
	events []rum.RUMEvent
	cap    int
}

// New creates a file-backed RUM store. On creation, it loads existing events
// from disk and replays them into the in-memory store.
func New(dir string, capacity int) (*Store, error) {
	if capacity <= 0 {
		capacity = 10000
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	mem := rummemory.NewStore(capacity)
	s := &Store{
		mem:  mem,
		path: filepath.Join(dir, "events.json"),
		cap:  capacity,
	}

	// Load existing events and replay into memory store.
	if data, err := os.ReadFile(s.path); err == nil {
		var events []rum.RUMEvent
		if json.Unmarshal(data, &events) == nil {
			s.events = events
			for _, e := range events {
				_ = mem.WriteEvent(e)
			}
		}
	}

	return s, nil
}

func (s *Store) WriteEvent(event rum.RUMEvent) error {
	if err := s.mem.WriteEvent(event); err != nil {
		return err
	}
	s.appendAndPersist(event)
	return nil
}

func (s *Store) WriteBatch(events []rum.RUMEvent) error {
	if err := s.mem.WriteBatch(events); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, events...)
	if len(s.events) > s.cap {
		s.events = s.events[len(s.events)-s.cap:]
	}
	s.persist()
	return nil
}

func (s *Store) WebVitalsOverview() (*rum.WebVitalsOverview, error) {
	return s.mem.WebVitalsOverview()
}

func (s *Store) PageLoads() ([]rum.PagePerformance, error) {
	return s.mem.PageLoads()
}

func (s *Store) ErrorGroups() ([]rum.ErrorGroup, error) {
	return s.mem.ErrorGroups()
}

func (s *Store) PagePerformance(route string) (*rum.PagePerformance, error) {
	return s.mem.PagePerformance(route)
}

func (s *Store) Sessions(limit int) ([]rum.SessionSummary, error) {
	return s.mem.Sessions(limit)
}

func (s *Store) SessionDetail(sessionID string) ([]rum.RUMEvent, error) {
	return s.mem.SessionDetail(sessionID)
}

func (s *Store) RageClicks(minutes int) ([]rum.ClickEvent, error) {
	return s.mem.RageClicks(minutes)
}

func (s *Store) UserJourneys(sessionID string) ([]rum.NavigationEvent, error) {
	return s.mem.UserJourneys(sessionID)
}

func (s *Store) PerformanceByRoute() ([]rum.RoutePerformance, error) {
	return s.mem.PerformanceByRoute()
}

func (s *Store) appendAndPersist(event rum.RUMEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	if len(s.events) > s.cap {
		s.events = s.events[len(s.events)-s.cap:]
	}
	s.persist()
}

func (s *Store) persist() {
	data, err := json.Marshal(s.events)
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0644)
}

// Compile-time check.
var _ rum.RUMStore = (*Store)(nil)
