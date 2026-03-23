package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/neureaux/cloudmock/pkg/incident"
)

// Store implements incident.IncidentStore using a mutex-protected slice.
type Store struct {
	mu        sync.RWMutex
	incidents []incident.Incident
}

// NewStore creates a new in-memory incident store.
func NewStore() *Store {
	return &Store{}
}

// Save appends the incident. If inc.ID is empty, a UUID is generated.
func (s *Store) Save(_ context.Context, inc *incident.Incident) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if inc.ID == "" {
		inc.ID = uuid.New().String()
	}

	s.incidents = append(s.incidents, *inc)
	return nil
}

// Get returns the incident with the given ID, or ErrNotFound.
func (s *Store) Get(_ context.Context, id string) (*incident.Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.incidents {
		if s.incidents[i].ID == id {
			inc := s.incidents[i]
			return &inc, nil
		}
	}
	return nil, incident.ErrNotFound
}

// List returns incidents matching the filter, newest first (by FirstSeen).
func (s *Store) List(_ context.Context, filter incident.IncidentFilter) ([]incident.Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []incident.Incident
	// Iterate newest first (reverse order of insertion).
	for i := len(s.incidents) - 1; i >= 0; i-- {
		inc := s.incidents[i]
		if filter.Status != "" && inc.Status != filter.Status {
			continue
		}
		if filter.Severity != "" && inc.Severity != filter.Severity {
			continue
		}
		if filter.Service != "" && !containsStr(inc.AffectedServices, filter.Service) {
			continue
		}
		results = append(results, inc)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

// Update finds the incident by ID and replaces it entirely.
func (s *Store) Update(_ context.Context, inc *incident.Incident) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.incidents {
		if s.incidents[i].ID == inc.ID {
			s.incidents[i] = *inc
			return nil
		}
	}
	return incident.ErrNotFound
}

// FindActiveByKey returns the most recent active or acknowledged incident
// matching the given service and optional deploy ID, with LastSeen after since.
func (s *Store) FindActiveByKey(_ context.Context, service, deployID string, since time.Time) (*incident.Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var best *incident.Incident
	for i := range s.incidents {
		inc := &s.incidents[i]

		// Must be active or acknowledged.
		if inc.Status != "active" && inc.Status != "acknowledged" {
			continue
		}

		// LastSeen must be after since.
		if !inc.LastSeen.After(since) {
			continue
		}

		// Service must be in AffectedServices.
		if !containsStr(inc.AffectedServices, service) {
			continue
		}

		// Deploy ID must match if provided.
		if deployID != "" && inc.RelatedDeployID != deployID {
			continue
		}

		// Keep the most recent match by LastSeen.
		if best == nil || inc.LastSeen.After(best.LastSeen) {
			cp := *inc
			best = &cp
		}
	}

	if best == nil {
		return nil, incident.ErrNotFound
	}
	return best, nil
}

// containsStr reports whether ss contains s.
func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// Compile-time interface check.
var _ incident.IncidentStore = (*Store)(nil)
