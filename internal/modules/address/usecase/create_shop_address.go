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

type CreateShopAddressUsecase struct {
	shopAddressRepo repository.ShopAddressRepository
	executor        transaction.Executor
}

func NewCreateShopAddressUsecase(
	shopAddressRepo repository.ShopAddressRepository,
	executor transaction.Executor,
) *CreateShopAddressUsecase {
	return &CreateShopAddressUsecase{
		shopAddressRepo: shopAddressRepo,
		executor:        executor,
	}
}

type CreateShopAddressInput struct {
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

func (u *CreateShopAddressUsecase) Execute(ctx context.Context, input CreateShopAddressInput) error {
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

	var isDefault bool
	if *input.IsActive {
		isDefault = *input.IsActive
	} else {
		isDefault = false
	}

	address := domain.ShopAddress{
		ID:        uuid.New(),
		ShopID:    input.ShopID,
		Label:     input.Label,
		Phone:     input.Phone,
		IsActive:  isDefault,
		Detail:    detail,
		CreatedAt: appclock.Now(),
	}

	err := u.shopAddressRepo.Create(ctx, u.executor, address)
	if err != nil {
		return fmt.Errorf("failed to save address: %w", err)
	}

	return nil
}
