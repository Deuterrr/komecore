package usecase

import (
	"context"
	"time"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

type AccountInfo struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Email  string
}

type CreateAccountInput struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Email     string
	Password  string
	CreatedAt time.Time
}

type AccountManager interface {
	GetByEmail(ctx context.Context, exec transaction.Executor, email string) (*AccountInfo, error)
	GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*AccountInfo, error)
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*AccountInfo, error)
	CreateStaffAccount(ctx context.Context, exec transaction.Executor, input CreateAccountInput) error
	DeleteByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error
	RevokeSessionsByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type UserDeletionService interface {
	DeleteUserRecord(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error
}
