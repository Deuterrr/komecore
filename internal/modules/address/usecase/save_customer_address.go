package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/repository"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type SaveCustomerAddressUsecase struct {
	executor            transaction.Executor
	transactor          transaction.Transactor
	customerAddressRepo repository.CustomerAddressRepository
}

func NewSaveCustomerAddressUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	customerAddressRepo repository.CustomerAddressRepository,
) *SaveCustomerAddressUsecase {
	return &SaveCustomerAddressUsecase{
		executor:            executor,
		transactor:          transactor,
		customerAddressRepo: customerAddressRepo,
	}
}

type SaveCustomerAddressInput struct {
	ID           *uuid.UUID
	CustomerID   uuid.UUID
	ReceiverName string
	Phone        *string
	IsDefault    *bool
	Province     string
	ProvinceID   string
	City         string
	CityID       string
	District     string
	DistrictID   string
	VillageID    string
	FullAddress  string
	PostalCode   string
	Latitude     *float64
	Longitude    *float64
}

func (u *SaveCustomerAddressUsecase) Execute(ctx context.Context, input SaveCustomerAddressInput) error {
	prov := input.Province
	if prov == "" {
		prov = input.ProvinceID
	}
	city := input.City
	if city == "" {
		city = input.CityID
	}
	dist := input.District
	if dist == "" {
		dist = input.DistrictID
	}

	detail := domain.AddressDetail{
		Province:    prov,
		City:        city,
		District:    dist,
		FullAddress: input.FullAddress,
		PostalCode:  input.PostalCode,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
	}

	if err := detail.ValidateCoordinates(); err != nil {
		return apperrors.NewBadRequest(err.Error())
	}

	var addressID uuid.UUID

	isCreate := input.ID == nil
	isDefault := input.IsDefault != nil && *input.IsDefault
	if isCreate {
		addressID = uuid.New()

		count, err := u.customerAddressRepo.CountByCustomerID(ctx, u.executor, input.CustomerID)
		if err != nil {
			return fmt.Errorf("failed to count addresses: %w", err)
		}

		if count != nil {
			if *count >= 10 {
				return apperrors.NewConflict(domain.ErrAddressLimitReached.Error())
			}
			if *count == 0 {
				isDefault = true
			}
		}
	} else {
		addressID = *input.ID
	}

	address := domain.CustomerAddress{
		ID:           addressID,
		CustomerID:   input.CustomerID,
		ReceiverName: input.ReceiverName,
		Phone:        input.Phone,
		IsDefault:    isDefault,
		Detail:       detail,
		CreatedAt:    appclock.Now(),
	}

	if err := u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if isDefault {
			if err := u.customerAddressRepo.UnsetDefaultByCustomerID(ctx, exec, input.CustomerID); err != nil {
				return fmt.Errorf("failed to unset default address: %w", err)
			}
		}
		if err := u.customerAddressRepo.Save(ctx, exec, address); err != nil {
			return fmt.Errorf("failed to save address: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
