package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Viridian-Inc/cloudmock/pkg/auth"
)

// Store is a PostgreSQL-backed implementation of auth.UserStore.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a new PostgreSQL user store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, user *auth.User) error {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, role, tenant_id, password_hash)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		user.Email, user.Name, user.Role, user.TenantID, user.PasswordHash,
	)
	return row.Scan(&user.ID, &user.CreatedAt)
}

func (s *Store) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	u := &auth.User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, role, tenant_id, password_hash, created_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.TenantID, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %s: %w", email, err)
	}
	return u, nil
}

func (s *Store) GetByID(ctx context.Context, id string) (*auth.User, error) {
	u := &auth.User{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, name, role, tenant_id, password_hash, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.TenantID, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %s: %w", id, err)
	}
	return u, nil
}

func (s *Store) List(ctx context.Context) ([]auth.User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, email, name, role, tenant_id, password_hash, created_at
		 FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []auth.User
	for rows.Next() {
		var u auth.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.TenantID, &u.PasswordHash, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateRole(ctx context.Context, id, role string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET role = $1 WHERE id = $2`, role, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}
