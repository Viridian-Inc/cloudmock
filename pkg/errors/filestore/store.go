// Package filestore implements errors.ErrorStore backed by JSON files on disk.
// Groups are stored as individual files, events as a capped JSON array.
package filestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	errs "github.com/neureaux/cloudmock/pkg/errors"
)

const defaultEventCap = 10000

// Store implements errs.ErrorStore using file persistence.
type Store struct {
	mu        sync.RWMutex
	dir       string
	eventCap  int
}

// New creates a file-backed error store.
func New(dir string, eventCap int) (*Store, error) {
	if eventCap <= 0 {
		eventCap = defaultEventCap
	}
	for _, sub := range []string{"groups", "events"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			return nil, fmt.Errorf("errors filestore: create dir: %w", err)
		}
	}
	return &Store{dir: dir, eventCap: eventCap}, nil
}

func (s *Store) groupPath(id string) string {
	return filepath.Join(s.dir, "groups", id+".json")
}

func (s *Store) eventsFile() string {
	return filepath.Join(s.dir, "events", "events.json")
}

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
	var g errs.ErrorGroup
	if err := s.readJSON(s.groupPath(fp), &g); err != nil {
		g = errs.ErrorGroup{
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
	}
	g.Count++
	if event.Timestamp.After(g.LastSeen) {
		g.LastSeen = event.Timestamp
	}
	if event.SessionID != "" {
		g.Sessions++
	}
	_ = s.writeJSON(s.groupPath(fp), g)

	// Append event to events file.
	events := s.loadEvents()
	events = append(events, event)
	if len(events) > s.eventCap {
		events = events[len(events)-s.eventCap:]
	}
	_ = s.writeJSON(s.eventsFile(), events)

	return nil
}

func (s *Store) GetGroups(status string, limit int) ([]errs.ErrorGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	entries, err := os.ReadDir(filepath.Join(s.dir, "groups"))
	if err != nil {
		return nil, err
	}

	var groups []errs.ErrorGroup
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var g errs.ErrorGroup
		if err := s.readJSON(filepath.Join(s.dir, "groups", e.Name()), &g); err == nil {
			if status != "" && g.Status != status {
				continue
			}
			groups = append(groups, g)
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].LastSeen.After(groups[j].LastSeen)
	})

	if len(groups) > limit {
		groups = groups[:limit]
	}
	return groups, nil
}

func (s *Store) GetGroup(id string) (*errs.ErrorGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var g errs.ErrorGroup
	if err := s.readJSON(s.groupPath(id), &g); err != nil {
		return nil, fmt.Errorf("error group %q not found", id)
	}
	return &g, nil
}

func (s *Store) GetEvents(groupID string, limit int) ([]errs.ErrorEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	events := s.loadEvents()
	var result []errs.ErrorEvent
	// Walk backwards for most recent first.
	for i := len(events) - 1; i >= 0 && len(result) < limit; i-- {
		if events[i].GroupID == groupID {
			result = append(result, events[i])
		}
	}
	return result, nil
}

func (s *Store) UpdateGroupStatus(id string, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var g errs.ErrorGroup
	if err := s.readJSON(s.groupPath(id), &g); err != nil {
		return fmt.Errorf("error group %q not found", id)
	}
	g.Status = status
	return s.writeJSON(s.groupPath(id), g)
}

// --- helpers ---

func (s *Store) loadEvents() []errs.ErrorEvent {
	var events []errs.ErrorEvent
	_ = s.readJSON(s.eventsFile(), &events)
	return events
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
var _ errs.ErrorStore = (*Store)(nil)
