// Package dynamostore implements webhook.Store backed by DynamoDB
// via the generic dynamostore package.
package dynamostore

import (
	"context"

	ds "github.com/Viridian-Inc/cloudmock/pkg/dynamostore"
	"github.com/Viridian-Inc/cloudmock/pkg/webhook"
)

const featureWebhook = "WEBHOOK"

// Store implements webhook.Store.
type Store struct {
	db *ds.Store
}

// New creates a DynamoDB-backed webhook store.
func New(db *ds.Store) *Store {
	return &Store{db: db}
}

func (s *Store) Save(ctx context.Context, cfg *webhook.Config) error {
	return s.db.Put(ctx, featureWebhook, cfg.ID, cfg)
}

func (s *Store) Get(ctx context.Context, id string) (*webhook.Config, error) {
	var cfg webhook.Config
	if err := s.db.Get(ctx, featureWebhook, id, &cfg); err != nil {
		if err == ds.ErrNotFound {
			return nil, webhook.ErrNotFound
		}
		return nil, err
	}
	return &cfg, nil
}

func (s *Store) List(ctx context.Context) ([]webhook.Config, error) {
	var all []webhook.Config
	if err := s.db.List(ctx, featureWebhook, &all); err != nil {
		return nil, err
	}
	return all, nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	return s.db.Delete(ctx, featureWebhook, id)
}

func (s *Store) ListByEvent(ctx context.Context, event string) ([]webhook.Config, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var result []webhook.Config
	for _, cfg := range all {
		if !cfg.Active {
			continue
		}
		for _, e := range cfg.Events {
			if e == event {
				result = append(result, cfg)
				break
			}
		}
	}
	return result, nil
}

// Compile-time interface check.
var _ webhook.Store = (*Store)(nil)
