package memory

import (
	"context"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/auth"
)

func TestStore_CreateAndGet(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	user := &auth.User{
		Email:        "alice@example.com",
		Name:         "Alice",
		Role:         auth.RoleViewer,
		PasswordHash: "hashed",
	}

	if err := s.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	if user.ID == "" {
		t.Fatal("expected ID to be generated")
	}

	got, err := s.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", got.Name)
	}

	gotByID, err := s.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotByID.Email != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %s", gotByID.Email)
	}
}

func TestStore_DuplicateEmail(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	user := &auth.User{Email: "dup@example.com", Name: "A", Role: auth.RoleViewer, PasswordHash: "h"}
	if err := s.Create(ctx, user); err != nil {
		t.Fatal(err)
	}

	user2 := &auth.User{Email: "dup@example.com", Name: "B", Role: auth.RoleViewer, PasswordHash: "h"}
	if err := s.Create(ctx, user2); err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestStore_List(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	for _, email := range []string{"a@x.com", "b@x.com", "c@x.com"} {
		if err := s.Create(ctx, &auth.User{Email: email, Name: email, Role: auth.RoleViewer, PasswordHash: "h"}); err != nil {
			t.Fatal(err)
		}
	}

	users, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}
}

func TestStore_UpdateRole(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	user := &auth.User{Email: "role@x.com", Name: "R", Role: auth.RoleViewer, PasswordHash: "h"}
	if err := s.Create(ctx, user); err != nil {
		t.Fatal(err)
	}

	if err := s.UpdateRole(ctx, user.ID, auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Role != auth.RoleAdmin {
		t.Fatalf("expected admin, got %s", got.Role)
	}
}

func TestStore_NotFound(t *testing.T) {
	s := NewStore()
	ctx := context.Background()

	if _, err := s.GetByEmail(ctx, "nope@x.com"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := s.GetByID(ctx, "nope"); err == nil {
		t.Fatal("expected error")
	}
	if err := s.UpdateRole(ctx, "nope", auth.RoleAdmin); err == nil {
		t.Fatal("expected error")
	}
}
