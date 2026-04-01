package memory

import (
	"fmt"
	"sort"
	"sync"
	"time"

	errs "github.com/neureaux/cloudmock/pkg/errors"
)

// Store is an in-memory implementation of errs.ErrorStore using a circular
// buffer for events and a map for groups.
type Store struct {
	mu     sync.RWMutex
	groups map[string]*errs.ErrorGroup // keyed by fingerprint
	events []errs.ErrorEvent
	cap    int
	pos    int
	full   bool
}

// NewStore creates a new in-memory error store. eventCap controls the maximum
// number of error events kept (circular buffer).
func NewStore(eventCap int) *Store {
	if eventCap <= 0 {
		eventCap = 10000
	}
	return &Store{
		groups: make(map[string]*errs.ErrorGroup),
		events: make([]errs.ErrorEvent, eventCap),
		cap:    eventCap,
	}
}

// IngestError adds an error event and creates/updates the corresponding group.
func (s *Store) IngestError(event errs.ErrorEvent) error {
	fp := errs.Fingerprint(event.Message, event.Stack)
	if event.GroupID == "" {
		event.GroupID = fp
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Upsert group.
	g, ok := s.groups[fp]
	if !ok {
		g = &errs.ErrorGroup{
			ID:        fp,
			Message:   event.Message,
			Stack:     event.Stack,
			Source:    event.Service,
			FirstSeen: event.Timestamp,
			LastSeen:  event.Timestamp,
			Status:    "unresolved",
			Release:   event.Release,
			Tags:      make(map[string]string),
		}
		s.groups[fp] = g
	}
	g.Count++
	if event.Timestamp.After(g.LastSeen) {
		g.LastSeen = event.Timestamp
	}
	// Track unique sessions.
	if event.SessionID != "" {
		// We track sessions via a simple counter bump. For true uniqueness
		// at scale you'd use a set, but for a dev mock this is sufficient
		// when combined with the fingerprint dedup.
		if !ok {
			g.Sessions = 1
		} else {
			g.Sessions++
		}
	}

	// Write event to circular buffer.
	s.events[s.pos] = event
	s.pos = (s.pos + 1) % s.cap
	if s.pos == 0 {
		s.full = true
	}

	return nil
}

// GetGroups returns error groups optionally filtered by status.
func (s *Store) GetGroups(status string, limit int) ([]errs.ErrorGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	result := make([]errs.ErrorGroup, 0, len(s.groups))
	for _, g := range s.groups {
		if status != "" && g.Status != status {
			continue
		}
		result = append(result, *g)
	}

	// Sort by last seen descending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeen.After(result[j].LastSeen)
	})

	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// GetGroup returns a single error group by fingerprint ID.
func (s *Store) GetGroup(id string) (*errs.ErrorGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.groups[id]
	if !ok {
		return nil, fmt.Errorf("error group %q not found", id)
	}
	cpy := *g
	return &cpy, nil
}

// GetEvents returns events for a group, most recent first.
func (s *Store) GetEvents(groupID string, limit int) ([]errs.ErrorEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var result []errs.ErrorEvent
	count := s.cap
	if !s.full {
		count = s.pos
	}

	// Walk backwards from the most recent write position.
	for i := 0; i < count && len(result) < limit; i++ {
		idx := (s.pos - 1 - i + s.cap) % s.cap
		e := s.events[idx]
		if e.GroupID == groupID {
			result = append(result, e)
		}
	}

	return result, nil
}

// UpdateGroupStatus updates the status of an error group.
func (s *Store) UpdateGroupStatus(id string, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.groups[id]
	if !ok {
		return fmt.Errorf("error group %q not found", id)
	}
	g.Status = status
	return nil
}
