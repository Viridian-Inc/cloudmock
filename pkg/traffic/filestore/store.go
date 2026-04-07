// Package filestore persists traffic recordings and replay runs as JSON files
// on disk. Recordings survive gateway restarts without requiring a database.
//
// Storage layout:
//
//	{dir}/
//	  recordings/
//	    {id}.json
//	  runs/
//	    {id}.json
package filestore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Viridian-Inc/cloudmock/pkg/traffic"
)

// Store implements traffic.RecordingStore backed by JSON files on disk.
type Store struct {
	dir string
	mu  sync.RWMutex

	recCounter int
	runCounter int
}

// New creates a file-backed traffic store. The directory is created if it
// doesn't exist. Existing recordings and runs are loaded on first access.
func New(dir string) (*Store, error) {
	recDir := filepath.Join(dir, "recordings")
	runDir := filepath.Join(dir, "runs")

	if err := os.MkdirAll(recDir, 0755); err != nil {
		return nil, fmt.Errorf("create recordings dir: %w", err)
	}
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, fmt.Errorf("create runs dir: %w", err)
	}

	s := &Store{dir: dir}

	// Count existing files to set counters
	if entries, err := os.ReadDir(recDir); err == nil {
		s.recCounter = len(entries)
	}
	if entries, err := os.ReadDir(runDir); err == nil {
		s.runCounter = len(entries)
	}

	return s, nil
}

// --- Recordings ---

func (s *Store) SaveRecording(_ context.Context, rec *traffic.Recording) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rec.ID == "" {
		s.recCounter++
		rec.ID = fmt.Sprintf("rec-%d", s.recCounter)
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return os.WriteFile(s.recPath(rec.ID), data, 0644)
}

func (s *Store) GetRecording(_ context.Context, id string) (*traffic.Recording, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.recPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("recording %s: %w", id, traffic.ErrNotFound)
		}
		return nil, err
	}

	var rec traffic.Recording
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *Store) ListRecordings(_ context.Context) ([]traffic.Recording, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.dir, "recordings")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var recs []traffic.Recording
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var rec traffic.Recording
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		recs = append(recs, rec)
	}

	// Newest first
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].StartedAt.After(recs[j].StartedAt)
	})
	return recs, nil
}

func (s *Store) DeleteRecording(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.recPath(id))
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("recording %s: %w", id, traffic.ErrNotFound)
	}
	return err
}

// --- Runs ---

func (s *Store) SaveRun(_ context.Context, run *traffic.ReplayRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if run.ID == "" {
		s.runCounter++
		run.ID = fmt.Sprintf("run-%d", s.runCounter)
	}

	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	return os.WriteFile(s.runPath(run.ID), data, 0644)
}

func (s *Store) GetRun(_ context.Context, id string) (*traffic.ReplayRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.runPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("run %s: %w", id, traffic.ErrNotFound)
		}
		return nil, err
	}

	var run traffic.ReplayRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *Store) ListRuns(_ context.Context) ([]traffic.ReplayRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.dir, "runs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var runs []traffic.ReplayRun
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var run traffic.ReplayRun
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}
		runs = append(runs, run)
	}

	// Newest first
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})
	return runs, nil
}

func (s *Store) UpdateRun(ctx context.Context, run *traffic.ReplayRun) error {
	return s.SaveRun(ctx, run)
}

// --- Paths ---

func (s *Store) recPath(id string) string {
	return filepath.Join(s.dir, "recordings", id+".json")
}

func (s *Store) runPath(id string) string {
	return filepath.Join(s.dir, "runs", id+".json")
}

// Compile-time check
var _ traffic.RecordingStore = (*Store)(nil)
