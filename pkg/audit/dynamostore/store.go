// Package dynamostore implements audit.Logger backed by DynamoDB
// via the generic dynamostore package.
package dynamostore

import (
	"context"

	"github.com/google/uuid"

	ds "github.com/Viridian-Inc/cloudmock/pkg/dynamostore"
	"github.com/Viridian-Inc/cloudmock/pkg/audit"
)

const featureAudit = "AUDIT"

// Logger implements audit.Logger.
type Logger struct {
	db *ds.Store
}

// New creates a DynamoDB-backed audit logger.
func New(db *ds.Store) *Logger {
	return &Logger{db: db}
}

func (l *Logger) Log(ctx context.Context, entry audit.Entry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	return l.db.Put(ctx, featureAudit, entry.ID, entry)
}

func (l *Logger) Query(ctx context.Context, filter audit.Filter) ([]audit.Entry, error) {
	var all []audit.Entry
	if err := l.db.List(ctx, featureAudit, &all); err != nil {
		return nil, err
	}

	var results []audit.Entry
	for i := len(all) - 1; i >= 0; i-- {
		e := all[i]
		if filter.Actor != "" && e.Actor != filter.Actor {
			continue
		}
		if filter.Action != "" && e.Action != filter.Action {
			continue
		}
		if filter.Resource != "" && e.Resource != filter.Resource {
			continue
		}
		results = append(results, e)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

// Compile-time interface check.
var _ audit.Logger = (*Logger)(nil)
