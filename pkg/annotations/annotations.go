package annotations

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when an annotation does not exist.
var ErrNotFound = errors.New("annotation not found")

// Annotation marks an event on a metric timeline.
type Annotation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
	Tags      []string  `json:"tags"` // e.g., "deploy", "incident", "config-change"
	Service   string    `json:"service,omitempty"`
}

// Store provides in-memory storage for annotations.
type Store struct {
	mu          sync.RWMutex
	annotations []Annotation
}

// NewStore creates a new in-memory annotation store.
func NewStore() *Store {
	return &Store{}
}

// Create adds an annotation to the store, assigning an ID if empty.
func (s *Store) Create(a Annotation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.Timestamp.IsZero() {
		a.Timestamp = time.Now()
	}
	s.annotations = append(s.annotations, a)
	return nil
}

// List returns annotations within the given time range and optional service filter.
// If service is empty, all services are included.
func (s *Store) List(start, end time.Time, service string) ([]Annotation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []Annotation
	for _, a := range s.annotations {
		if !a.Timestamp.Before(start) && !a.Timestamp.After(end) {
			if service == "" || a.Service == service {
				results = append(results, a)
			}
		}
	}
	return results, nil
}

// Delete removes an annotation by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, a := range s.annotations {
		if a.ID == id {
			s.annotations = append(s.annotations[:i], s.annotations[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
