package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/repository"

	"github.com/google/uuid"
)

type DeleteCustomerAddressUsecase struct {
	executor            transaction.Executor
	customerAddressRepo repository.CustomerAddressRepository
}

func NewDeleteCustomerAddressUsecase(
	executor transaction.Executor,
	customerAddressRepo repository.CustomerAddressRepository,
) *DeleteCustomerAddressUsecase {
	return &DeleteCustomerAddressUsecase{
		executor:            executor,
		customerAddressRepo: customerAddressRepo,
	}
}

func (u *DeleteCustomerAddressUsecase) Execute(ctx context.Context, addressID uuid.UUID) error {
	address, err := u.customerAddressRepo.GetByID(ctx, u.executor, addressID)
	if err != nil {
		return fmt.Errorf("failed to retrieve address: %w", err)
	}
	if address == nil {
		return apperrors.NewNotFound(domain.ErrAddressNotFound.Error())
	}
	if address.IsDefault {
		return apperrors.NewConflict(domain.ErrCannotDeleteDefaultAddress.Error())
	}

	if err := u.customerAddressRepo.Delete(ctx, u.executor, addressID); err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}

	return nil
}
