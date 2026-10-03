package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/repository"

	"github.com/google/uuid"
)

type SavePaymentMethodUsecase struct {
	paymentMethodRepo repository.PaymentMethodRepository
	executor          transaction.Executor
}

func NewSavePaymentMethodUsecase(
	paymentMethodRepo repository.PaymentMethodRepository,
	executor transaction.Executor,
) *SavePaymentMethodUsecase {
	return &SavePaymentMethodUsecase{
		paymentMethodRepo: paymentMethodRepo,
		executor:          executor,
	}
}

type SavePaymentMethodInput struct {
	ID       uuid.UUID
	IsActive bool
}

func (u *SavePaymentMethodUsecase) Execute(
	ctx context.Context,
	input SavePaymentMethodInput,
) error {
	paymentMethod, err := u.paymentMethodRepo.GetByID(ctx, u.executor,
		input.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to retrieve payment method: %w", err)
	}

	if paymentMethod == nil {
		return apperrors.NewNotFound("payment method not found")
	}

	paymentMethod.IsActive = input.IsActive

	err = u.paymentMethodRepo.Save(ctx, u.executor,
		*paymentMethod,
	)
	if err != nil {
		return fmt.Errorf("failed to save payment method: %w", err)
	}

	return nil
}
