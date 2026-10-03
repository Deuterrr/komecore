package usecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/repository"

	"github.com/google/uuid"
)

type GetAccountUsecase struct {
	accountRepo repository.AccountRepository
	executor    transaction.Executor
}

func NewGetAccountUsecase(
	accountRepo repository.AccountRepository,
	executor transaction.Executor,
) *GetAccountUsecase {
	return &GetAccountUsecase{
		accountRepo: accountRepo,
	}
}

func (u *GetAccountUsecase) Execute(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	acc, err := u.accountRepo.GetByUserID(ctx, u.executor, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	return acc, nil
}
