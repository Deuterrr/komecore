package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

// FindUserParams encapsulates search filters, email, username, and pagination for user directory lookups.
type FindUserParams struct {
	ID       *uuid.UUID
	Name     *string
	Username *string
	Email    *string

	pagination.Pagination
	pagination.Sorts
}

// CreateUserProps carries identity attributes and contact details required to provision a core user record.
type CreateUserProps struct {
	ID        uuid.UUID
	Name      string
	Username  string
	Phone     *string
	AvatarURL *string
	CreatedAt time.Time
}

// SaveProfileProps carries updated contact attributes, avatar URLs, and timestamps for profile mutation.
type SaveProfileProps struct {
	UserID    uuid.UUID
	Name      *string
	Phone     *string
	AvatarURL *string
	Username  *string
	UpdatedAt time.Time
}

// Repository defines persistence operations for base user identities, profile attributes,
// and account lifecycle states within transaction boundaries.
type Repository interface {
	// Resolves a core user record by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*User, error)
	// Looks up an active user by unique handle or username.
	GetByUsername(ctx context.Context, exec transaction.Executor, username string) (*User, error)

	// Persists a newly provisioned core user identity record within transaction boundaries.
	CreateUser(ctx context.Context, exec transaction.Executor, props CreateUserProps) error

	// Updates mutable profile attributes (display name, phone number, avatar URL, or username).
	SaveProfile(ctx context.Context, exec transaction.Executor, props SaveProfileProps) error

	// Permanently purges user identity record or cascades account deletion during account removal.
	Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error
}
