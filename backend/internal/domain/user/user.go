// Package user contains the User aggregate root and its value objects.
// This package is the heart of the domain – it has zero external dependencies.
package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ============================================================
// Sentinel errors (domain-level, not infrastructure-level)
// ============================================================

var (
	ErrNotFound          = errors.New("user: not found")
	ErrAlreadyExists     = errors.New("user: already exists")
	ErrInvalidCredentials = errors.New("user: invalid credentials")
	ErrInactive          = errors.New("user: account is inactive")
	ErrInvalidID         = errors.New("user: invalid id")
)

// ============================================================
// Value Objects
// ============================================================

// Role is a type-safe enum for user roles.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleManager  Role = "manager"
	RoleMember   Role = "member"
	RoleGuest    Role = "guest"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleManager, RoleMember, RoleGuest:
		return true
	}
	return false
}

// ============================================================
// Aggregate Root
// ============================================================

// User is the aggregate root for the user bounded context.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Role         Role
	IsActive     bool
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// FullName returns the concatenated display name.
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// Activate enables the user account.
func (u *User) Activate() {
	u.IsActive = true
	u.UpdatedAt = time.Now().UTC()
}

// Deactivate disables the user account.
func (u *User) Deactivate() {
	u.IsActive = false
	u.UpdatedAt = time.Now().UTC()
}

// RecordLogin sets the last login timestamp.
func (u *User) RecordLogin() {
	now := time.Now().UTC()
	u.LastLoginAt = &now
}

// CanAccess returns true if the user has a sufficient role.
func (u *User) CanAccess(required Role) bool {
	if !u.IsActive {
		return false
	}
	hierarchy := map[Role]int{
		RoleGuest:   0,
		RoleMember:  1,
		RoleManager: 2,
		RoleAdmin:   3,
	}
	return hierarchy[u.Role] >= hierarchy[required]
}

// ============================================================
// Repository Interface (Port – implemented by infra layer)
// ============================================================

// Repository defines the persistence contract for Users.
// Implementations live in internal/repository/postgres.
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ListFilter) ([]*User, int64, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// ListFilter encapsulates pagination and filtering for list queries.
type ListFilter struct {
	Page     int
	PageSize int
	SortBy   string
	Order    string // "asc" | "desc"
	IsActive *bool
	Role     *Role
	Search   string
}

// ============================================================
// Service Interface (for cross-domain use)
// ============================================================

type Service interface {
	Create(ctx context.Context, cmd CreateCommand) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, cmd UpdateCommand) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ListFilter) ([]*User, int64, error)
	Login(ctx context.Context, email, password string) (*User, string, string, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, error)
}

// ============================================================
// Commands (DTOs for write operations)
// ============================================================

type CreateCommand struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
	Role      Role
}

type UpdateCommand struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
}
