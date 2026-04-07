package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/pkg/auth"
)

// Store is an in-memory implementation of auth.UserStore.
type Store struct {
	mu      sync.RWMutex
	byEmail map[string]*auth.User
	byID    map[string]*auth.User
}

// NewStore returns a new in-memory user store.
func NewStore() *Store {
	return &Store{
		byEmail: make(map[string]*auth.User),
		byID:    make(map[string]*auth.User),
	}
}

func (s *Store) Create(_ context.Context, user *auth.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[user.Email]; exists {
		return fmt.Errorf("user with email %q already exists", user.Email)
	}

	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	// Store a copy to avoid aliasing.
	u := *user
	s.byEmail[u.Email] = &u
	s.byID[u.ID] = &u
	// Propagate the generated ID back.
	user.ID = u.ID
	return nil
}

func (s *Store) GetByEmail(_ context.Context, email string) (*auth.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", email)
	}
	cp := *u
	return &cp, nil
}

func (s *Store) GetByID(_ context.Context, id string) (*auth.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	cp := *u
	return &cp, nil
}

func (s *Store) List(_ context.Context) ([]auth.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]auth.User, 0, len(s.byID))
	for _, u := range s.byID {
		users = append(users, *u)
	}
	return users, nil
}

func (s *Store) UpdateRole(_ context.Context, id, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("user not found: %s", id)
	}
	u.Role = role
	// byEmail points to the same struct, so it's updated too.
	return nil
}
