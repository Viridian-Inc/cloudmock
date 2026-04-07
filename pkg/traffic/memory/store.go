package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/traffic"
)

// Store implements traffic.RecordingStore using a mutex-protected map.
type Store struct {
	mu         sync.RWMutex
	recordings map[string]traffic.Recording
	runs       map[string]traffic.ReplayRun
	recSeq     int
	runSeq     int
}

// NewStore creates a new in-memory traffic store.
func NewStore() *Store {
	return &Store{
		recordings: make(map[string]traffic.Recording),
		runs:       make(map[string]traffic.ReplayRun),
	}
}

// SaveRecording persists a recording, assigning an ID if empty.
func (s *Store) SaveRecording(_ context.Context, rec *traffic.Recording) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rec.ID == "" {
		s.recSeq++
		rec.ID = fmt.Sprintf("rec-%d", s.recSeq)
	}
	if rec.StartedAt.IsZero() {
		rec.StartedAt = time.Now().UTC()
	}
	s.recordings[rec.ID] = *rec
	return nil
}

// GetRecording returns a recording by ID.
func (s *Store) GetRecording(_ context.Context, id string) (*traffic.Recording, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.recordings[id]
	if !ok {
		return nil, traffic.ErrNotFound
	}
	return &rec, nil
}

// ListRecordings returns all recordings sorted newest first.
func (s *Store) ListRecordings(_ context.Context) ([]traffic.Recording, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]traffic.Recording, 0, len(s.recordings))
	for _, rec := range s.recordings {
		// Strip entries from listing to keep payload small.
		summary := rec
		summary.Entries = nil
		results = append(results, summary)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartedAt.After(results[j].StartedAt)
	})
	return results, nil
}

// DeleteRecording removes a recording by ID.
func (s *Store) DeleteRecording(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.recordings[id]; !ok {
		return traffic.ErrNotFound
	}
	delete(s.recordings, id)
	return nil
}

// SaveRun persists a replay run, assigning an ID if empty.
func (s *Store) SaveRun(_ context.Context, run *traffic.ReplayRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if run.ID == "" {
		s.runSeq++
		run.ID = fmt.Sprintf("run-%d", s.runSeq)
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = time.Now().UTC()
	}
	s.runs[run.ID] = *run
	return nil
}

// GetRun returns a replay run by ID.
func (s *Store) GetRun(_ context.Context, id string) (*traffic.ReplayRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, ok := s.runs[id]
	if !ok {
		return nil, traffic.ErrNotFound
	}
	return &run, nil
}

// ListRuns returns all replay runs sorted newest first.
func (s *Store) ListRuns(_ context.Context) ([]traffic.ReplayRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]traffic.ReplayRun, 0, len(s.runs))
	for _, run := range s.runs {
		// Strip individual results to keep listing compact.
		summary := run
		summary.Results = nil
		results = append(results, summary)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartedAt.After(results[j].StartedAt)
	})
	return results, nil
}

// UpdateRun overwrites an existing run.
func (s *Store) UpdateRun(_ context.Context, run *traffic.ReplayRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.runs[run.ID]; !ok {
		return traffic.ErrNotFound
	}
	s.runs[run.ID] = *run
	return nil
}

// Compile-time interface check.
var _ traffic.RecordingStore = (*Store)(nil)
