package usecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/repository"

	"github.com/google/uuid"
)

type ListCustomerAddressesUsecase struct {
	customerAddressRepo repository.CustomerAddressRepository
	executor            transaction.Executor
}

func NewListCustomerAddressesUsecase(
	customerAddressRepo repository.CustomerAddressRepository,
	executor transaction.Executor,
) *ListCustomerAddressesUsecase {
	return &ListCustomerAddressesUsecase{
		customerAddressRepo: customerAddressRepo,
		executor:            executor,
	}
}

func (u *ListCustomerAddressesUsecase) ListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]domain.CustomerAddress, error) {
	res, err := u.customerAddressRepo.ListByCustomerID(ctx, u.executor, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve address: %w", err)
	}

	return res, nil
}
