package regression

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("regression not found")

type RegressionStore interface {
	Save(ctx context.Context, r *Regression) error
	List(ctx context.Context, filter RegressionFilter) ([]Regression, error)
	Get(ctx context.Context, id string) (*Regression, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	ActiveForDeploy(ctx context.Context, deployID string) ([]Regression, error)
}

type RegressionFilter struct {
	Service   string
	DeployID  string
	Algorithm AlgorithmType
	Severity  Severity
	Status    string
	Limit     int
}
