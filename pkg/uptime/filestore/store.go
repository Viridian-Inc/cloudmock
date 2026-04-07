// Package filestore implements uptime.Store backed by JSON files on disk.
// Checks are stored as individual files. Results are stored per-check as
// a JSON array file, capped at maxResults entries.
package filestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Viridian-Inc/cloudmock/pkg/uptime"
)

const defaultMaxResults = 1000

// Store implements uptime.Store using JSON file persistence.
type Store struct {
	mu         sync.RWMutex
	dir        string
	maxResults int
}

// New creates a file-backed uptime store.
func New(dir string, maxResults int) (*Store, error) {
	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}
	for _, sub := range []string{"checks", "results"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			return nil, fmt.Errorf("uptime filestore: create dir: %w", err)
		}
	}
	return &Store{dir: dir, maxResults: maxResults}, nil
}

func (s *Store) checkPath(id string) string {
	return filepath.Join(s.dir, "checks", id+".json")
}

func (s *Store) resultsPath(checkID string) string {
	return filepath.Join(s.dir, "results", checkID+".json")
}

func (s *Store) CreateCheck(check uptime.Check) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.checkPath(check.ID)); err == nil {
		return fmt.Errorf("check %q already exists", check.ID)
	}
	return s.writeJSON(s.checkPath(check.ID), check)
}

func (s *Store) UpdateCheck(check uptime.Check) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.checkPath(check.ID)); os.IsNotExist(err) {
		return fmt.Errorf("check %q not found", check.ID)
	}
	return s.writeJSON(s.checkPath(check.ID), check)
}

func (s *Store) DeleteCheck(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.checkPath(id)); os.IsNotExist(err) {
		return fmt.Errorf("check %q not found", id)
	}
	os.Remove(s.checkPath(id))
	os.Remove(s.resultsPath(id))
	return nil
}

func (s *Store) GetCheck(id string) (*uptime.Check, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var c uptime.Check
	if err := s.readJSON(s.checkPath(id), &c); err != nil {
		return nil, fmt.Errorf("check %q not found", id)
	}
	return &c, nil
}

func (s *Store) ListChecks() ([]uptime.Check, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(filepath.Join(s.dir, "checks"))
	if err != nil {
		return nil, err
	}

	var checks []uptime.Check
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var c uptime.Check
		if err := s.readJSON(filepath.Join(s.dir, "checks", e.Name()), &c); err == nil {
			checks = append(checks, c)
		}
	}
	return checks, nil
}

func (s *Store) AddResult(result uptime.CheckResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.checkPath(result.CheckID)); os.IsNotExist(err) {
		return fmt.Errorf("check %q not found", result.CheckID)
	}

	results := s.loadResults(result.CheckID)
	results = append(results, result)
	if len(results) > s.maxResults {
		results = results[len(results)-s.maxResults:]
	}
	return s.writeJSON(s.resultsPath(result.CheckID), results)
}

func (s *Store) Results(checkID string) ([]uptime.CheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := os.Stat(s.checkPath(checkID)); os.IsNotExist(err) {
		return nil, fmt.Errorf("check %q not found", checkID)
	}

	results := s.loadResults(checkID)
	// Reverse to newest first.
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}
	return results, nil
}

func (s *Store) LastResult(checkID string) (*uptime.CheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := os.Stat(s.checkPath(checkID)); os.IsNotExist(err) {
		return nil, fmt.Errorf("check %q not found", checkID)
	}

	results := s.loadResults(checkID)
	if len(results) == 0 {
		return nil, nil
	}
	r := results[len(results)-1]
	return &r, nil
}

func (s *Store) LastFailure(checkID string) (*uptime.CheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := os.Stat(s.checkPath(checkID)); os.IsNotExist(err) {
		return nil, fmt.Errorf("check %q not found", checkID)
	}

	results := s.loadResults(checkID)
	// Walk backwards (newest first).
	for i := len(results) - 1; i >= 0; i-- {
		if !results[i].Success {
			r := results[i]
			return &r, nil
		}
	}
	return nil, nil
}

func (s *Store) ConsecutiveFailures(checkID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := s.loadResults(checkID)
	count := 0
	for i := len(results) - 1; i >= 0; i-- {
		if !results[i].Success {
			count++
		} else {
			break
		}
	}
	return count
}

// --- helpers ---

func (s *Store) loadResults(checkID string) []uptime.CheckResult {
	var results []uptime.CheckResult
	_ = s.readJSON(s.resultsPath(checkID), &results)
	return results
}

func (s *Store) writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *Store) readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Compile-time check.
var _ uptime.Store = (*Store)(nil)
