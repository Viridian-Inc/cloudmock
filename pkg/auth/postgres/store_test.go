package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Viridian-Inc/cloudmock/pkg/auth"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() { container.Terminate(ctx) })

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatal(err)
	}

	dsn := "postgres://test:test@" + host + ":" + port.Port() + "/testdb?sslmode=disable"
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	// Create schema.
	_, err = pool.Exec(ctx, `
		CREATE TABLE users (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email         TEXT UNIQUE NOT NULL,
			name          TEXT NOT NULL,
			role          TEXT NOT NULL DEFAULT 'viewer',
			tenant_id     TEXT,
			password_hash TEXT NOT NULL,
			created_at    TIMESTAMPTZ DEFAULT now()
		)`)
	if err != nil {
		t.Fatal(err)
	}

	return pool
}

func TestPostgresStore_CRUD(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	s := NewStore(pool)
	ctx := context.Background()

	user := &auth.User{
		Email:        "pg@example.com",
		Name:         "PG User",
		Role:         auth.RoleEditor,
		PasswordHash: "hashed",
	}

	// Create
	if err := s.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	if user.ID == "" {
		t.Fatal("expected ID to be set")
	}

	// GetByEmail
	got, err := s.GetByEmail(ctx, "pg@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "PG User" {
		t.Fatalf("expected PG User, got %s", got.Name)
	}

	// GetByID
	got2, err := s.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Email != "pg@example.com" {
		t.Fatalf("expected pg@example.com, got %s", got2.Email)
	}

	// List
	users, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	// UpdateRole
	if err := s.UpdateRole(ctx, user.ID, auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	got3, err := s.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got3.Role != auth.RoleAdmin {
		t.Fatalf("expected admin, got %s", got3.Role)
	}
}
