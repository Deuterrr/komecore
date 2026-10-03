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

type UpdateShopAddressUsecase struct {
	shopAddressRepo repository.ShopAddressRepository
	executor        transaction.Executor
	transactor      transaction.Transactor
}

func NewUpdateShopAddressUsecase(
	shopAddressRepo repository.ShopAddressRepository,
	executor transaction.Executor,
	transactor transaction.Transactor,
) *UpdateShopAddressUsecase {
	return &UpdateShopAddressUsecase{
		shopAddressRepo: shopAddressRepo,
		executor:        executor,
		transactor:      transactor,
	}
}

type UpdateShopAddressInput struct {
	ID          uuid.UUID
	ShopID      uuid.UUID
	Label       string
	Phone       *string
	IsActive    *bool
	Province    string
	ProvinceID  string
	City        string
	CityID      string
	District    string
	DistrictID  string
	VillageID   string
	FullAddress string
	PostalCode  string
	Latitude    *float64
	Longitude   *float64
}

func (u *UpdateShopAddressUsecase) Execute(ctx context.Context, input UpdateShopAddressInput) error {
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

	existing, err := u.shopAddressRepo.GetByID(ctx, u.executor, input.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve address: %w", err)
	}
	if existing == nil || existing.ShopID != input.ShopID {
		return apperrors.NewNotFound(domain.ErrAddressNotFound.Error())
	}

	isDefault := existing.IsActive
	if input.IsActive != nil {
		isDefault = *input.IsActive
	}
	now := appclock.Now()
	address := domain.ShopAddress{
		ID:        input.ID,
		ShopID:    input.ShopID,
		Label:     input.Label,
		Phone:     input.Phone,
		IsActive:  isDefault,
		Detail:    detail,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: &now,
	}

	if err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if isDefault {
			if err := u.shopAddressRepo.UnsetActiveByShopID(ctx, exec, input.ShopID); err != nil {
				return fmt.Errorf("failed to unset active address: %w", err)
			}
		}
		if err := u.shopAddressRepo.Update(ctx, exec, address); err != nil {
			return fmt.Errorf("failed to update address: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
