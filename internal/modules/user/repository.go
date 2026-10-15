package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

type FindUserParams struct {
	ID       *uuid.UUID
	Name     *string
	Username *string
	Email    *string

	pagination.Pagination
	pagination.Sorts
}

type CreateUserProps struct {
	ID        uuid.UUID
	Name      string
	Username  string
	Phone     *string
	AvatarURL *string
	CreatedAt time.Time
}

type SaveProfileProps struct {
	UserID    uuid.UUID
	Name      *string
	Phone     *string
	AvatarURL *string
	Username  *string
	UpdatedAt time.Time
}

type Repository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*User, error)

	GetByUsername(
		ctx context.Context,
		exec transaction.Executor,
		username string,
	) (*User, error)

	CreateUser(
		ctx context.Context,
		exec transaction.Executor,
		props CreateUserProps,
	) error

	SaveProfile(
		ctx context.Context,
		exec transaction.Executor,
		props SaveProfileProps,
	) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error
}

type UserRepository = Repository
