package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/regression"
)

// Store implements regression.RegressionStore using a mutex-protected slice.
type Store struct {
	mu          sync.RWMutex
	regressions []regression.Regression
	seq         int
}

// NewStore creates a new in-memory regression store.
func NewStore() *Store {
	return &Store{}
}

// Save generates an ID, sets DetectedAt if zero, and appends the regression.
func (s *Store) Save(_ context.Context, r *regression.Regression) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	r.ID = fmt.Sprintf("reg-%d", s.seq)

	if r.DetectedAt.IsZero() {
		r.DetectedAt = time.Now().UTC()
	}

	s.regressions = append(s.regressions, *r)
	return nil
}

// List returns regressions matching the filter, newest first.
func (s *Store) List(_ context.Context, filter regression.RegressionFilter) ([]regression.Regression, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []regression.Regression
	// Iterate newest first (reverse order of insertion).
	for i := len(s.regressions) - 1; i >= 0; i-- {
		r := s.regressions[i]
		if filter.Service != "" && r.Service != filter.Service {
			continue
		}
		if filter.DeployID != "" && r.DeployID != filter.DeployID {
			continue
		}
		if filter.Algorithm != "" && r.Algorithm != filter.Algorithm {
			continue
		}
		if filter.Severity != "" && r.Severity != filter.Severity {
			continue
		}
		if filter.Status != "" && r.Status != filter.Status {
			continue
		}
		results = append(results, r)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

// Get returns the regression with the given ID, or ErrNotFound.
func (s *Store) Get(_ context.Context, id string) (*regression.Regression, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.regressions {
		if s.regressions[i].ID == id {
			r := s.regressions[i]
			return &r, nil
		}
	}
	return nil, regression.ErrNotFound
}

// UpdateStatus updates the status of the regression with the given ID. If the
// new status is "resolved" or "dismissed", ResolvedAt is set to now.
func (s *Store) UpdateStatus(_ context.Context, id string, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.regressions {
		if s.regressions[i].ID == id {
			s.regressions[i].Status = status
			if status == "resolved" || status == "dismissed" {
				now := time.Now().UTC()
				s.regressions[i].ResolvedAt = &now
			}
			return nil
		}
	}
	return regression.ErrNotFound
}

// ActiveForDeploy returns all active regressions for the given deploy ID,
// newest first.
func (s *Store) ActiveForDeploy(_ context.Context, deployID string) ([]regression.Regression, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []regression.Regression
	for i := len(s.regressions) - 1; i >= 0; i-- {
		r := s.regressions[i]
		if r.DeployID == deployID && r.Status == "active" {
			results = append(results, r)
		}
	}

	// Sort newest first by DetectedAt for consistency.
	sort.Slice(results, func(i, j int) bool {
		return results[i].DetectedAt.After(results[j].DetectedAt)
	})

	return results, nil
}

// Compile-time interface check.
var _ regression.RegressionStore = (*Store)(nil)
