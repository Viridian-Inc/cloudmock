package memory

import (
	"fmt"
	"sync"

	"github.com/neureaux/cloudmock/pkg/replay"
)

// Store is an in-memory circular-buffer replay session store.
type Store struct {
	mu       sync.RWMutex
	sessions []replay.Session
	cap      int
	pos      int
	full     bool
	index    map[string]int // session ID -> position in buffer
}

// NewStore creates an in-memory store with the given capacity.
func NewStore(capacity int) *Store {
	if capacity <= 0 {
		capacity = 100
	}
	return &Store{
		sessions: make([]replay.Session, capacity),
		cap:      capacity,
		index:    make(map[string]int),
	}
}

// SaveSession persists a recorded session into the circular buffer.
func (s *Store) SaveSession(session replay.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If we're overwriting an old session, remove it from the index.
	if s.full || s.pos < len(s.sessions) {
		old := s.sessions[s.pos]
		if old.ID != "" {
			delete(s.index, old.ID)
		}
	}

	s.sessions[s.pos] = session
	s.index[session.ID] = s.pos
	s.pos = (s.pos + 1) % s.cap
	if s.pos == 0 {
		s.full = true
	}
	return nil
}

// GetSession returns a single session by ID, or nil if not found.
func (s *Store) GetSession(id string) (*replay.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, ok := s.index[id]
	if !ok {
		return nil, nil
	}
	sess := s.sessions[idx]
	return &sess, nil
}

// ListSessions returns the most recent sessions, newest first.
func (s *Store) ListSessions(limit int) ([]replay.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	if s.full {
		count = s.cap
	} else {
		count = s.pos
	}

	if limit <= 0 || limit > count {
		limit = count
	}

	result := make([]replay.Session, 0, limit)
	// Walk backwards from the most recent write position.
	for i := 0; i < limit; i++ {
		idx := (s.pos - 1 - i + s.cap) % s.cap
		result = append(result, s.sessions[idx])
	}
	return result, nil
}

// LinkError associates an error ID with a session.
func (s *Store) LinkError(sessionID, errorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.index[sessionID]
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	// Avoid duplicate links.
	for _, eid := range s.sessions[idx].ErrorIDs {
		if eid == errorID {
			return nil
		}
	}
	s.sessions[idx].ErrorIDs = append(s.sessions[idx].ErrorIDs, errorID)
	return nil
}
