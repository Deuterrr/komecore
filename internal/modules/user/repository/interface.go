package repository

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/user/domain"

	"github.com/google/uuid"
)

type UserRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.User, error)

	GetByUsername(
		ctx context.Context,
		exec transaction.Executor,
		username string,
	) (*domain.User, error)

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
