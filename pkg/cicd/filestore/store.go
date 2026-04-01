// Package filestore implements cicd.Store backed by JSON files on disk.
package filestore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/neureaux/cloudmock/pkg/cicd"
)

const maxPipelines = 500

// Store implements cicd.Store using JSON file persistence.
type Store struct {
	mu  sync.RWMutex
	dir string
}

// New creates a file-backed CI/CD store.
func New(dir string) (*Store, error) {
	for _, sub := range []string{"pipelines", "testresults"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			return nil, err
		}
	}
	return &Store{dir: dir}, nil
}

func (s *Store) pipelinePath(id string) string {
	return filepath.Join(s.dir, "pipelines", id+".json")
}

func (s *Store) testResultsPath(pipelineID string) string {
	return filepath.Join(s.dir, "testresults", pipelineID+".json")
}

func (s *Store) SavePipeline(p cicd.Pipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Enforce max pipelines by counting existing files.
	entries, _ := os.ReadDir(filepath.Join(s.dir, "pipelines"))
	if len(entries) >= maxPipelines {
		// Remove the first (oldest by name) file.
		if len(entries) > 0 {
			oldest := entries[0]
			os.Remove(filepath.Join(s.dir, "pipelines", oldest.Name()))
			// Also remove associated test results.
			id := strings.TrimSuffix(oldest.Name(), ".json")
			os.Remove(s.testResultsPath(id))
		}
	}

	return s.writeJSON(s.pipelinePath(p.ID), p)
}

func (s *Store) GetPipeline(id string) (*cicd.Pipeline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var p cicd.Pipeline
	if err := s.readJSON(s.pipelinePath(id), &p); err != nil {
		return nil, cicd.ErrNotFound
	}
	return &p, nil
}

func (s *Store) ListPipelines(limit int) []cicd.Pipeline {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(filepath.Join(s.dir, "pipelines"))
	if err != nil {
		return nil
	}

	var pipelines []cicd.Pipeline
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var p cicd.Pipeline
		if err := s.readJSON(filepath.Join(s.dir, "pipelines", e.Name()), &p); err == nil {
			pipelines = append(pipelines, p)
		}
	}

	// Newest first.
	n := len(pipelines)
	if limit <= 0 || limit > n {
		limit = n
	}
	result := make([]cicd.Pipeline, limit)
	for i := 0; i < limit; i++ {
		result[i] = pipelines[n-1-i]
	}
	return result
}

func (s *Store) SaveTestResults(pipelineID string, results []cicd.TestResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeJSON(s.testResultsPath(pipelineID), results)
}

func (s *Store) GetTestResults(pipelineID string) ([]cicd.TestResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []cicd.TestResult
	if err := s.readJSON(s.testResultsPath(pipelineID), &results); err != nil {
		return nil, nil
	}
	out := make([]cicd.TestResult, len(results))
	copy(out, results)
	return out, nil
}

func (s *Store) Summary() cicd.CISummary {
	pipelines := s.ListPipelines(0)

	summary := cicd.CISummary{
		TotalPipelines: len(pipelines),
		ByProvider:     make(map[string]int),
	}

	if len(pipelines) == 0 {
		return summary
	}

	var successCount int
	var totalDuration int64

	for _, p := range pipelines {
		summary.ByProvider[p.Provider]++
		totalDuration += p.DurationMs
		if p.Status == "success" {
			successCount++
		}
	}

	summary.PassRate = float64(successCount) / float64(len(pipelines))
	summary.AvgDurationMs = float64(totalDuration) / float64(len(pipelines))

	// Detect flaky tests.
	type testKey struct {
		suite, name string
	}
	testStats := make(map[testKey]struct{ pass, total int })

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range pipelines {
		var results []cicd.TestResult
		if err := s.readJSON(s.testResultsPath(p.ID), &results); err != nil {
			continue
		}
		for _, tr := range results {
			k := testKey{tr.Suite, tr.Name}
			st := testStats[k]
			st.total++
			if tr.Status == "passed" {
				st.pass++
			}
			testStats[k] = st
		}
	}

	for k, stats := range testStats {
		if stats.total >= 3 && stats.pass > 0 && stats.pass < stats.total {
			rate := float64(stats.pass) / float64(stats.total)
			summary.FlakyTests = append(summary.FlakyTests, cicd.FlakyTest{
				Suite:    k.suite,
				Name:     k.name,
				PassRate: rate,
				Runs:     stats.total,
			})
		}
	}

	return summary
}

// --- helpers ---

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
var _ cicd.Store = (*Store)(nil)
