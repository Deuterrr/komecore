package address

import (
	"context"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type Service struct {
	customerAddressRepo CustomerAddressRepository
	shopAddressRepo     ShopAddressRepository
	executor            transaction.Executor
	transactor          transaction.Transactor
}

func NewService(
	customerAddressRepo CustomerAddressRepository,
	shopAddressRepo ShopAddressRepository,
	executor transaction.Executor,
	transactor transaction.Transactor,
) *Service {
	return &Service{
		customerAddressRepo: customerAddressRepo,
		shopAddressRepo:     shopAddressRepo,
		executor:            executor,
		transactor:          transactor,
	}
}

// ListCustomerAddresses lists all addresses for a customer.
func (s *Service) ListCustomerAddresses(ctx context.Context, customerID uuid.UUID) ([]CustomerAddress, error) {
	res, err := s.customerAddressRepo.ListByCustomerID(ctx, s.executor, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve address: %w", err)
	}
	return res, nil
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

// SaveCustomerAddress creates or updates a customer address.
func (s *Service) SaveCustomerAddress(ctx context.Context, input SaveCustomerAddressInput) error {
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

	detail := AddressDetail{
		Province:    prov,
		City:        city,
		District:    dist,
		FullAddress: input.FullAddress,
		PostalCode:  input.PostalCode,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
	}
	if err := detail.ValidateCoordinates(); err != nil {
		return apperror.NewBadRequest(err.Error())
	}

	var addressID uuid.UUID
	isCreate := input.ID == nil
	isDefault := input.IsDefault != nil && *input.IsDefault
	if isCreate {
		addressID = uuid.New()

		count, err := s.customerAddressRepo.CountByCustomerID(ctx, s.executor, input.CustomerID)
		if err != nil {
			return fmt.Errorf("failed to count addresses: %w", err)
		}
		if count != nil {
			if *count >= 10 {
				return apperror.NewConflict(ErrAddressLimitReached.Error())
			}
			if *count == 0 {
				isDefault = true
			}
		}
	} else {
		addressID = *input.ID
	}

	addr := CustomerAddress{
		ID:           addressID,
		CustomerID:   input.CustomerID,
		ReceiverName: input.ReceiverName,
		Phone:        input.Phone,
		IsDefault:    isDefault,
		Detail:       detail,
		CreatedAt:    appclock.Now(),
	}

	if err := s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if isDefault {
			if err := s.customerAddressRepo.UnsetDefaultByCustomerID(ctx, exec, input.CustomerID); err != nil {
				return fmt.Errorf("failed to unset default address: %w", err)
			}
		}
		if err := s.customerAddressRepo.Save(ctx, exec, addr); err != nil {
			return fmt.Errorf("failed to save address: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// DeleteCustomerAddress deletes an address for a customer if not default.
func (s *Service) DeleteCustomerAddress(ctx context.Context, addressID uuid.UUID) error {
	addr, err := s.customerAddressRepo.GetByID(ctx, s.executor, addressID)
	if err != nil {
		return fmt.Errorf("failed to retrieve address: %w", err)
	}
	if addr == nil {
		return apperror.NewNotFound(ErrAddressNotFound.Error())
	}
	if addr.IsDefault {
		return apperror.NewConflict(ErrCannotDeleteDefaultAddress.Error())
	}

	if err := s.customerAddressRepo.Delete(ctx, s.executor, addressID); err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}

	return nil
}

// ListShopAddresses retrieves addresses associated with a shop.
func (s *Service) ListShopAddresses(ctx context.Context, shopID uuid.UUID) ([]ShopAddress, error) {
	res, err := s.shopAddressRepo.FindByShopID(ctx, s.executor, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve address: %w", err)
	}
	return res, nil
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

// CreateShopAddress adds a new address to a shop.
func (s *Service) CreateShopAddress(ctx context.Context, input CreateShopAddressInput) error {
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

	detail := AddressDetail{
		Province:    prov,
		City:        city,
		District:    dist,
		FullAddress: input.FullAddress,
		PostalCode:  input.PostalCode,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
	}
	if err := detail.ValidateCoordinates(); err != nil {
		return apperror.NewBadRequest(err.Error())
	}

	var isDefault bool
	if input.IsActive != nil && *input.IsActive {
		isDefault = true
	} else {
		isDefault = false
	}

	addr := ShopAddress{
		ID:        uuid.New(),
		ShopID:    input.ShopID,
		Label:     input.Label,
		Phone:     input.Phone,
		IsActive:  isDefault,
		Detail:    detail,
		CreatedAt: appclock.Now(),
	}
	err := s.shopAddressRepo.Create(ctx, s.executor, addr)
	if err != nil {
		return fmt.Errorf("failed to save address: %w", err)
	}

	return nil
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

// UpdateShopAddress modifies an existing shop address.
func (s *Service) UpdateShopAddress(ctx context.Context, input UpdateShopAddressInput) error {
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

	detail := AddressDetail{
		Province:    prov,
		City:        city,
		District:    dist,
		FullAddress: input.FullAddress,
		PostalCode:  input.PostalCode,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
	}
	if err := detail.ValidateCoordinates(); err != nil {
		return apperror.NewBadRequest(err.Error())
	}

	existing, err := s.shopAddressRepo.GetByID(ctx, s.executor, input.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve address: %w", err)
	}
	if existing == nil || existing.ShopID != input.ShopID {
		return apperror.NewNotFound(ErrAddressNotFound.Error())
	}

	isDefault := existing.IsActive
	if input.IsActive != nil {
		isDefault = *input.IsActive
	}
	now := appclock.Now()
	addr := ShopAddress{
		ID:        input.ID,
		ShopID:    input.ShopID,
		Label:     input.Label,
		Phone:     input.Phone,
		IsActive:  isDefault,
		Detail:    detail,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: &now,
	}

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if isDefault {
			if err := s.shopAddressRepo.UnsetActiveByShopID(ctx, exec, input.ShopID); err != nil {
				return fmt.Errorf("failed to unset active address: %w", err)
			}
		}
		if err := s.shopAddressRepo.Update(ctx, exec, addr); err != nil {
			return fmt.Errorf("failed to update address: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// DeleteShopAddress deletes a non-active address from a shop.
func (s *Service) DeleteShopAddress(ctx context.Context, shopID uuid.UUID, addressID uuid.UUID) error {
	addr, err := s.shopAddressRepo.GetByID(ctx, s.executor, addressID)
	if err != nil {
		return fmt.Errorf("failed to retrieve address: %w", err)
	}
	if addr == nil || addr.ShopID != shopID {
		return apperror.NewNotFound(ErrAddressNotFound.Error())
	}
	if addr.IsActive {
		return apperror.NewConflict(ErrCannotDeleteDefaultAddress.Error())
	}

	if err := s.shopAddressRepo.Delete(ctx, s.executor, addressID); err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}

	return nil
}
