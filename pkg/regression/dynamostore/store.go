// Package dynamostore implements regression.RegressionStore backed by DynamoDB
// via the generic dynamostore package.
package dynamostore

import (
	"context"

	"github.com/google/uuid"

	ds "github.com/Viridian-Inc/cloudmock/pkg/dynamostore"
	"github.com/Viridian-Inc/cloudmock/pkg/regression"
)

const featureRegression = "REGRESSION"

// Store implements regression.RegressionStore.
type Store struct {
	db *ds.Store
}

// New creates a DynamoDB-backed regression store.
func New(db *ds.Store) *Store {
	return &Store{db: db}
}

func (s *Store) Save(ctx context.Context, r *regression.Regression) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return s.db.Put(ctx, featureRegression, r.ID, r)
}

func (s *Store) Get(ctx context.Context, id string) (*regression.Regression, error) {
	var r regression.Regression
	if err := s.db.Get(ctx, featureRegression, id, &r); err != nil {
		if err == ds.ErrNotFound {
			return nil, regression.ErrNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (s *Store) List(ctx context.Context, filter regression.RegressionFilter) ([]regression.Regression, error) {
	var all []regression.Regression
	if err := s.db.List(ctx, featureRegression, &all); err != nil {
		return nil, err
	}

	var results []regression.Regression
	for i := len(all) - 1; i >= 0; i-- {
		r := all[i]
		if filter.Service != "" && r.Service != filter.Service {
			continue
		}
		if filter.DeployID != "" && r.DeployID != filter.DeployID {
			continue
		}
		if filter.Algorithm != "" && r.Algorithm != filter.Algorithm {
			continue
		}
		if filter.Severity != "" && r.Severity != filter.Severity {
			continue
		}
		if filter.Status != "" && r.Status != filter.Status {
			continue
		}
		results = append(results, r)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

func (s *Store) UpdateStatus(ctx context.Context, id string, status string) error {
	r, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	r.Status = status
	return s.db.UpdateData(ctx, featureRegression, id, r)
}

func (s *Store) ActiveForDeploy(ctx context.Context, deployID string) ([]regression.Regression, error) {
	var all []regression.Regression
	if err := s.db.List(ctx, featureRegression, &all); err != nil {
		return nil, err
	}
	var result []regression.Regression
	for _, r := range all {
		if r.DeployID == deployID && r.Status == "active" {
			result = append(result, r)
		}
	}
	return result, nil
}

// Compile-time interface check.
var _ regression.RegressionStore = (*Store)(nil)
