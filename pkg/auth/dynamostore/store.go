// Package dynamostore implements auth.UserStore backed by DynamoDB
// via the generic dynamostore package.
package dynamostore

import (
	"context"

	"github.com/google/uuid"

	ds "github.com/neureaux/cloudmock/pkg/dynamostore"
	"github.com/neureaux/cloudmock/pkg/auth"
)

const featureUser = "USER"

// Store implements auth.UserStore.
type Store struct {
	db *ds.Store
}

// New creates a DynamoDB-backed user store.
func New(db *ds.Store) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, user *auth.User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	return s.db.Put(ctx, featureUser, user.ID, user)
}

func (s *Store) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	var all []auth.User
	if err := s.db.List(ctx, featureUser, &all); err != nil {
		return nil, err
	}
	for _, u := range all {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, ds.ErrNotFound
}

func (s *Store) GetByID(ctx context.Context, id string) (*auth.User, error) {
	var u auth.User
	if err := s.db.Get(ctx, featureUser, id, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) List(ctx context.Context) ([]auth.User, error) {
	var all []auth.User
	if err := s.db.List(ctx, featureUser, &all); err != nil {
		return nil, err
	}
	return all, nil
}

func (s *Store) UpdateRole(ctx context.Context, id, role string) error {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	u.Role = role
	return s.db.UpdateData(ctx, featureUser, id, u)
}

// Compile-time interface check.
var _ auth.UserStore = (*Store)(nil)
