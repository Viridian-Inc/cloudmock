package memory

import (
	"errors"
	"sync"

	"github.com/neureaux/cloudmock/pkg/cicd"
)

const maxPipelines = 500

// ErrNotFound is returned when a pipeline is not found.
var ErrNotFound = errors.New("pipeline not found")

// Store is an in-memory CI/CD store with a circular buffer for pipelines.
type Store struct {
	mu          sync.RWMutex
	pipelines   []cicd.Pipeline
	testResults map[string][]cicd.TestResult // keyed by pipeline ID
}

// NewStore creates a new in-memory CI/CD store.
func NewStore() *Store {
	return &Store{
		testResults: make(map[string][]cicd.TestResult),
	}
}

// SavePipeline adds or updates a pipeline. If the buffer exceeds maxPipelines,
// the oldest entry is evicted.
func (s *Store) SavePipeline(p cicd.Pipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing pipeline (update case).
	for i, existing := range s.pipelines {
		if existing.ID == p.ID {
			s.pipelines[i] = p
			return nil
		}
	}

	// Append new pipeline, evict oldest if at capacity.
	if len(s.pipelines) >= maxPipelines {
		// Remove oldest and its test results.
		oldest := s.pipelines[0]
		delete(s.testResults, oldest.ID)
		s.pipelines = s.pipelines[1:]
	}
	s.pipelines = append(s.pipelines, p)
	return nil
}

// GetPipeline returns a pipeline by ID.
func (s *Store) GetPipeline(id string) (*cicd.Pipeline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.pipelines {
		if s.pipelines[i].ID == id {
			p := s.pipelines[i]
			return &p, nil
		}
	}
	return nil, ErrNotFound
}

// ListPipelines returns the most recent pipelines, newest first.
// If limit <= 0, all pipelines are returned.
func (s *Store) ListPipelines(limit int) []cicd.Pipeline {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := len(s.pipelines)
	if limit <= 0 || limit > n {
		limit = n
	}

	result := make([]cicd.Pipeline, limit)
	for i := 0; i < limit; i++ {
		result[i] = s.pipelines[n-1-i] // newest first
	}
	return result
}

// SaveTestResults stores test results for a pipeline.
func (s *Store) SaveTestResults(pipelineID string, results []cicd.TestResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.testResults[pipelineID] = results
	return nil
}

// GetTestResults returns test results for a pipeline.
func (s *Store) GetTestResults(pipelineID string) ([]cicd.TestResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results, ok := s.testResults[pipelineID]
	if !ok {
		return nil, nil
	}
	out := make([]cicd.TestResult, len(results))
	copy(out, results)
	return out, nil
}

// Summary computes overall CI health metrics.
func (s *Store) Summary() cicd.CISummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := cicd.CISummary{
		TotalPipelines: len(s.pipelines),
		ByProvider:     make(map[string]int),
	}

	if len(s.pipelines) == 0 {
		return summary
	}

	var successCount int
	var totalDuration int64

	for _, p := range s.pipelines {
		summary.ByProvider[p.Provider]++
		totalDuration += p.DurationMs
		if p.Status == "success" {
			successCount++
		}
	}

	summary.PassRate = float64(successCount) / float64(len(s.pipelines))
	summary.AvgDurationMs = float64(totalDuration) / float64(len(s.pipelines))

	// Detect flaky tests: tests that have both passes and failures.
	type testKey struct {
		suite, name string
	}
	testStats := make(map[testKey]struct{ pass, total int })
	for _, results := range s.testResults {
		for _, tr := range results {
			k := testKey{tr.Suite, tr.Name}
			s := testStats[k]
			s.total++
			if tr.Status == "passed" {
				s.pass++
			}
			testStats[k] = s
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
