package traffic

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a recording or run does not exist.
var ErrNotFound = errors.New("not found")

// RecordingStore persists traffic recordings and replay runs.
type RecordingStore interface {
	SaveRecording(ctx context.Context, rec *Recording) error
	GetRecording(ctx context.Context, id string) (*Recording, error)
	ListRecordings(ctx context.Context) ([]Recording, error)
	DeleteRecording(ctx context.Context, id string) error

	SaveRun(ctx context.Context, run *ReplayRun) error
	GetRun(ctx context.Context, id string) (*ReplayRun, error)
	ListRuns(ctx context.Context) ([]ReplayRun, error)
	UpdateRun(ctx context.Context, run *ReplayRun) error
}
