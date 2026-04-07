// Package filestore implements replay.Store backed by JSON files on disk.
package filestore

import (
	"fmt"
	"sort"

	"github.com/Viridian-Inc/cloudmock/pkg/filestore"
	"github.com/Viridian-Inc/cloudmock/pkg/replay"
)

// Store implements replay.Store using JSON file persistence.
type Store struct {
	fs *filestore.JSONFileStore[replay.Session]
}

// New creates a file-backed replay store.
func New(dir string) (*Store, error) {
	fs, err := filestore.New[replay.Session](dir)
	if err != nil {
		return nil, err
	}
	return &Store{fs: fs}, nil
}

func (s *Store) SaveSession(session replay.Session) error {
	return s.fs.Save(session.ID, session)
}

func (s *Store) GetSession(id string) (*replay.Session, error) {
	sess, err := s.fs.Get(id)
	if err != nil {
		return nil, nil // match memory store convention: nil,nil for not found
	}
	return &sess, nil
}

func (s *Store) ListSessions(limit int) ([]replay.Session, error) {
	all, err := s.fs.List()
	if err != nil {
		return nil, err
	}

	// Sort newest first by StartedAt.
	sort.Slice(all, func(i, j int) bool {
		return all[i].StartedAt.After(all[j].StartedAt)
	})

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (s *Store) LinkError(sessionID, errorID string) error {
	sess, err := s.fs.Get(sessionID)
	if err != nil {
		return fmt.Errorf("session %q not found", sessionID)
	}

	// Avoid duplicate links.
	for _, eid := range sess.ErrorIDs {
		if eid == errorID {
			return nil
		}
	}
	sess.ErrorIDs = append(sess.ErrorIDs, errorID)
	return s.fs.Save(sessionID, sess)
}

// Compile-time check.
var _ replay.Store = (*Store)(nil)
