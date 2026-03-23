package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role constants for RBAC.
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// User represents an authenticated user.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	TenantID     string    `json:"tenant_id,omitempty"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Claims are the JWT claims embedded in each token.
type Claims struct {
	jwt.RegisteredClaims
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id,omitempty"`
}

// UserStore is the interface for user persistence.
type UserStore interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context) ([]User, error)
	UpdateRole(ctx context.Context, id, role string) error
}

// ValidRoles is the set of recognised role strings.
var ValidRoles = map[string]bool{
	RoleAdmin:  true,
	RoleEditor: true,
	RoleViewer: true,
}
