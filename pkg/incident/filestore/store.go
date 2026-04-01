// Package filestore implements incident.IncidentStore backed by JSON files.
package filestore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/neureaux/cloudmock/pkg/incident"
)

// Store implements incident.IncidentStore using JSON file persistence.
type Store struct {
	mu  sync.RWMutex
	dir string
}

// New creates a file-backed incident store.
func New(dir string) (*Store, error) {
	for _, sub := range []string{"incidents", "comments"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			return nil, fmt.Errorf("incident filestore: create dir: %w", err)
		}
	}
	return &Store{dir: dir}, nil
}

func (s *Store) incPath(id string) string {
	return filepath.Join(s.dir, "incidents", id+".json")
}

func (s *Store) commentsPath(incidentID string) string {
	return filepath.Join(s.dir, "comments", incidentID+".json")
}

func (s *Store) Save(_ context.Context, inc *incident.Incident) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if inc.ID == "" {
		inc.ID = uuid.New().String()
	}
	return s.writeJSON(s.incPath(inc.ID), inc)
}

func (s *Store) Get(_ context.Context, id string) (*incident.Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var inc incident.Incident
	if err := s.readJSON(s.incPath(id), &inc); err != nil {
		return nil, incident.ErrNotFound
	}
	return &inc, nil
}

func (s *Store) List(_ context.Context, filter incident.IncidentFilter) ([]incident.Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(filepath.Join(s.dir, "incidents"))
	if err != nil {
		return nil, err
	}

	var all []incident.Incident
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var inc incident.Incident
		if err := s.readJSON(filepath.Join(s.dir, "incidents", e.Name()), &inc); err == nil {
			all = append(all, inc)
		}
	}

	// Filter and reverse (newest first).
	var results []incident.Incident
	for i := len(all) - 1; i >= 0; i-- {
		inc := all[i]
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

func (s *Store) Update(_ context.Context, inc *incident.Incident) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.incPath(inc.ID)); os.IsNotExist(err) {
		return incident.ErrNotFound
	}
	return s.writeJSON(s.incPath(inc.ID), inc)
}

func (s *Store) FindActiveByKey(_ context.Context, service, deployID string, since time.Time) (*incident.Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(filepath.Join(s.dir, "incidents"))
	if err != nil {
		return nil, incident.ErrNotFound
	}

	var best *incident.Incident
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var inc incident.Incident
		if err := s.readJSON(filepath.Join(s.dir, "incidents", e.Name()), &inc); err != nil {
			continue
		}
		if inc.Status != "active" && inc.Status != "acknowledged" {
			continue
		}
		if !inc.LastSeen.After(since) {
			continue
		}
		if !containsStr(inc.AffectedServices, service) {
			continue
		}
		if deployID != "" && inc.RelatedDeployID != deployID {
			continue
		}
		if best == nil || inc.LastSeen.After(best.LastSeen) {
			cp := inc
			best = &cp
		}
	}

	if best == nil {
		return nil, incident.ErrNotFound
	}
	return best, nil
}

func (s *Store) AddComment(incidentID string, comment incident.Comment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify incident exists.
	if _, err := os.Stat(s.incPath(incidentID)); os.IsNotExist(err) {
		return incident.ErrNotFound
	}

	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	comment.IncidentID = incidentID

	var comments []incident.Comment
	_ = s.readJSON(s.commentsPath(incidentID), &comments)
	comments = append(comments, comment)
	return s.writeJSON(s.commentsPath(incidentID), comments)
}

func (s *Store) GetComments(incidentID string) ([]incident.Comment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var comments []incident.Comment
	_ = s.readJSON(s.commentsPath(incidentID), &comments)
	return comments, nil
}

// --- helpers ---

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
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
var _ incident.IncidentStore = (*Store)(nil)
