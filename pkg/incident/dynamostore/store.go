// Package dynamostore implements incident.IncidentStore backed by DynamoDB
// via the generic dynamostore package.
package dynamostore

import (
	"context"
	"time"

	"github.com/google/uuid"

	ds "github.com/Viridian-Inc/cloudmock/pkg/dynamostore"
	"github.com/Viridian-Inc/cloudmock/pkg/incident"
)

const (
	featureIncident = "INCIDENT"
	featureComment  = "COMMENT"
)

// Store implements incident.IncidentStore.
type Store struct {
	db *ds.Store
}

// New creates a DynamoDB-backed incident store.
func New(db *ds.Store) *Store {
	return &Store{db: db}
}

func (s *Store) Save(ctx context.Context, inc *incident.Incident) error {
	if inc.ID == "" {
		inc.ID = uuid.New().String()
	}
	return s.db.Put(ctx, featureIncident, inc.ID, inc)
}

func (s *Store) Get(ctx context.Context, id string) (*incident.Incident, error) {
	var inc incident.Incident
	if err := s.db.Get(ctx, featureIncident, id, &inc); err != nil {
		if err == ds.ErrNotFound {
			return nil, incident.ErrNotFound
		}
		return nil, err
	}
	return &inc, nil
}

func (s *Store) List(ctx context.Context, filter incident.IncidentFilter) ([]incident.Incident, error) {
	var all []incident.Incident
	if err := s.db.List(ctx, featureIncident, &all); err != nil {
		return nil, err
	}

	var results []incident.Incident
	for i := len(all) - 1; i >= 0; i-- {
		inc := all[i]
		if filter.Status != "" && inc.Status != filter.Status {
			continue
		}
		if filter.Severity != "" && inc.Severity != filter.Severity {
			continue
		}
		if filter.Service != "" {
			found := false
			for _, svc := range inc.AffectedServices {
				if svc == filter.Service {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		results = append(results, inc)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

func (s *Store) Update(ctx context.Context, inc *incident.Incident) error {
	if err := s.db.UpdateData(ctx, featureIncident, inc.ID, inc); err != nil {
		if err == ds.ErrNotFound {
			return incident.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Store) FindActiveByKey(ctx context.Context, service, deployID string, since time.Time) (*incident.Incident, error) {
	var all []incident.Incident
	if err := s.db.List(ctx, featureIncident, &all); err != nil {
		return nil, err
	}

	for _, inc := range all {
		if inc.Status != "open" && inc.Status != "acknowledged" {
			continue
		}
		if inc.FirstSeen.Before(since) {
			continue
		}
		for _, svc := range inc.AffectedServices {
			if svc == service {
				if deployID == "" || inc.RelatedDeployID == deployID {
					return &inc, nil
				}
			}
		}
	}
	return nil, incident.ErrNotFound
}

func (s *Store) AddComment(incidentID string, comment incident.Comment) error {
	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	comment.IncidentID = incidentID
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now()
	}
	// Store comments with a composite key: incidentID/commentID
	return s.db.Put(context.Background(), featureComment, incidentID+"/"+comment.ID, comment)
}

func (s *Store) GetComments(incidentID string) ([]incident.Comment, error) {
	// Query all comments -- then filter by incident ID prefix.
	var all []incident.Comment
	if err := s.db.List(context.Background(), featureComment, &all); err != nil {
		return nil, err
	}
	var result []incident.Comment
	for _, c := range all {
		if c.IncidentID == incidentID {
			result = append(result, c)
		}
	}
	return result, nil
}

// Compile-time interface check.
var _ incident.IncidentStore = (*Store)(nil)
