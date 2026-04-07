// Package dynamostore implements traffic.RecordingStore backed by DynamoDB
// via the generic dynamostore package.
package dynamostore

import (
	"context"

	"github.com/google/uuid"

	ds "github.com/Viridian-Inc/cloudmock/pkg/dynamostore"
	"github.com/Viridian-Inc/cloudmock/pkg/traffic"
)

const (
	featureRecording = "RECORDING"
	featureReplayRun = "REPLAYRUN"
)

// Store implements traffic.RecordingStore.
type Store struct {
	db *ds.Store
}

// New creates a DynamoDB-backed traffic store.
func New(db *ds.Store) *Store {
	return &Store{db: db}
}

func (s *Store) SaveRecording(ctx context.Context, rec *traffic.Recording) error {
	if rec.ID == "" {
		rec.ID = uuid.New().String()
	}
	return s.db.Put(ctx, featureRecording, rec.ID, rec)
}

func (s *Store) GetRecording(ctx context.Context, id string) (*traffic.Recording, error) {
	var rec traffic.Recording
	if err := s.db.Get(ctx, featureRecording, id, &rec); err != nil {
		if err == ds.ErrNotFound {
			return nil, traffic.ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

func (s *Store) ListRecordings(ctx context.Context) ([]traffic.Recording, error) {
	var all []traffic.Recording
	if err := s.db.List(ctx, featureRecording, &all); err != nil {
		return nil, err
	}
	return all, nil
}

func (s *Store) DeleteRecording(ctx context.Context, id string) error {
	return s.db.Delete(ctx, featureRecording, id)
}

func (s *Store) SaveRun(ctx context.Context, run *traffic.ReplayRun) error {
	if run.ID == "" {
		run.ID = uuid.New().String()
	}
	return s.db.Put(ctx, featureReplayRun, run.ID, run)
}

func (s *Store) GetRun(ctx context.Context, id string) (*traffic.ReplayRun, error) {
	var run traffic.ReplayRun
	if err := s.db.Get(ctx, featureReplayRun, id, &run); err != nil {
		if err == ds.ErrNotFound {
			return nil, traffic.ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

func (s *Store) ListRuns(ctx context.Context) ([]traffic.ReplayRun, error) {
	var all []traffic.ReplayRun
	if err := s.db.List(ctx, featureReplayRun, &all); err != nil {
		return nil, err
	}
	return all, nil
}

func (s *Store) UpdateRun(ctx context.Context, run *traffic.ReplayRun) error {
	if err := s.db.UpdateData(ctx, featureReplayRun, run.ID, run); err != nil {
		if err == ds.ErrNotFound {
			return traffic.ErrNotFound
		}
		return err
	}
	return nil
}

// Compile-time interface check.
var _ traffic.RecordingStore = (*Store)(nil)
