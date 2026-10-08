package paymentusecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type SavePaymentInstructionUsecase struct {
	paymentMethodRepo      paymentrepo.PaymentMethodRepository
	paymentInstructionRepo paymentrepo.PaymentInstructionRepository
	executor               transaction.Executor
}

func NewSavePaymentInstructionUsecase(
	paymentMethodRepo paymentrepo.PaymentMethodRepository,
	paymentInstructionRepo paymentrepo.PaymentInstructionRepository,
	executor transaction.Executor,
) *SavePaymentInstructionUsecase {
	return &SavePaymentInstructionUsecase{
		paymentMethodRepo:      paymentMethodRepo,
		paymentInstructionRepo: paymentInstructionRepo,
		executor:               executor,
	}
}

type SavePaymentInstructionInput struct {
	PaymentMethodID uuid.UUID
	Content         string
}

func (u *SavePaymentInstructionUsecase) Execute(
	ctx context.Context,
	input SavePaymentInstructionInput,
) error {
	method, err := u.paymentMethodRepo.GetByID(ctx, u.executor, input.PaymentMethodID)
	if err != nil {
		return fmt.Errorf("failed to retrieve payment method: %w", err)
	}

	if method == nil {
		return apperrors.NewNotFound("payment method not found")
	}

	existing, err := u.paymentInstructionRepo.GetByPaymentMethodID(ctx, u.executor, input.PaymentMethodID)
	if err != nil {
		return fmt.Errorf("failed to retrieve existing payment instruction: %w", err)
	}

	var instruction paymentdomain.PaymentInstruction
	if existing != nil {
		instruction = *existing
		instruction.Content = input.Content
	} else {
		instruction = paymentdomain.PaymentInstruction{
			ID:              uuid.New(),
			PaymentMethodID: input.PaymentMethodID,
			Content:         input.Content,
			CreatedAt:       appclock.Now(),
		}
	}

	err = u.paymentInstructionRepo.Save(ctx, u.executor,
		instruction,
	)
	if err != nil {
		return fmt.Errorf("failed to save payment instruction: %w", err)
	}

	return nil
}
