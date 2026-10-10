package addressusecase

import (
	"context"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/addressdomain"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type CustomerAddressRepository interface {
	ListByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) ([]addressdomain.CustomerAddress, error)
	CountByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (*int, error)
	GetByID(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) (*addressdomain.CustomerAddress, error)
	UnsetDefaultByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) error
	Save(ctx context.Context, exec transaction.Executor, address addressdomain.CustomerAddress) error
	Delete(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) error
}

type ShopAddressRepository interface {
	FindByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]addressdomain.ShopAddress, error)
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*addressdomain.ShopAddress, error)
	UnsetActiveByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) error
	Create(ctx context.Context, exec transaction.Executor, address addressdomain.ShopAddress) error
	Update(ctx context.Context, exec transaction.Executor, address addressdomain.ShopAddress) error
	Delete(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) error
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

type AddressService struct {
	customerAddressRepo CustomerAddressRepository
	shopAddressRepo     ShopAddressRepository
	executor            transaction.Executor
	transactor          transaction.Transactor
}

func NewAddressService(
	customerAddressRepo CustomerAddressRepository,
	shopAddressRepo ShopAddressRepository,
	executor transaction.Executor,
	transactor transaction.Transactor,
) *AddressService {
	return &AddressService{
		customerAddressRepo: customerAddressRepo,
		shopAddressRepo:     shopAddressRepo,
		executor:            executor,
		transactor:          transactor,
	}
}

// ListCustomerAddresses lists all addresses for a customer.
func (s *AddressService) ListCustomerAddresses(ctx context.Context, customerID uuid.UUID) ([]addressdomain.CustomerAddress, error) {
	res, err := s.customerAddressRepo.ListByCustomerID(ctx, s.executor, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve address: %w", err)
	}
	return res, nil
}

// SaveCustomerAddress creates or updates a customer address.
func (s *AddressService) SaveCustomerAddress(ctx context.Context, input SaveCustomerAddressInput) error {
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

	detail := addressdomain.AddressDetail{
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
				return apperror.NewConflict(addressdomain.ErrAddressLimitReached.Error())
			}
			if *count == 0 {
				isDefault = true
			}
		}
	} else {
		addressID = *input.ID
	}

	address := addressdomain.CustomerAddress{
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
		if err := s.customerAddressRepo.Save(ctx, exec, address); err != nil {
			return fmt.Errorf("failed to save address: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// DeleteCustomerAddress deletes an address for a customer if not default.
func (s *AddressService) DeleteCustomerAddress(ctx context.Context, addressID uuid.UUID) error {
	address, err := s.customerAddressRepo.GetByID(ctx, s.executor, addressID)
	if err != nil {
		return fmt.Errorf("failed to retrieve address: %w", err)
	}
	if address == nil {
		return apperror.NewNotFound(addressdomain.ErrAddressNotFound.Error())
	}
	if address.IsDefault {
		return apperror.NewConflict(addressdomain.ErrCannotDeleteDefaultAddress.Error())
	}

	if err := s.customerAddressRepo.Delete(ctx, s.executor, addressID); err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}

	return nil
}

// ListShopAddresses retrieves addresses associated with a shop.
func (s *AddressService) ListShopAddresses(ctx context.Context, shopID uuid.UUID) ([]addressdomain.ShopAddress, error) {
	res, err := s.shopAddressRepo.FindByShopID(ctx, s.executor, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve address: %w", err)
	}
	return res, nil
}

// CreateShopAddress adds a new address to a shop.
func (s *AddressService) CreateShopAddress(ctx context.Context, input CreateShopAddressInput) error {
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

	detail := addressdomain.AddressDetail{
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

	address := addressdomain.ShopAddress{
		ID:        uuid.New(),
		ShopID:    input.ShopID,
		Label:     input.Label,
		Phone:     input.Phone,
		IsActive:  isDefault,
		Detail:    detail,
		CreatedAt: appclock.Now(),
	}

	err := s.shopAddressRepo.Create(ctx, s.executor, address)
	if err != nil {
		return fmt.Errorf("failed to save address: %w", err)
	}

	return nil
}

// UpdateShopAddress modifies an existing shop address.
func (s *AddressService) UpdateShopAddress(ctx context.Context, input UpdateShopAddressInput) error {
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

	detail := addressdomain.AddressDetail{
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
		return apperror.NewNotFound(addressdomain.ErrAddressNotFound.Error())
	}

	isDefault := existing.IsActive
	if input.IsActive != nil {
		isDefault = *input.IsActive
	}
	now := appclock.Now()
	address := addressdomain.ShopAddress{
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
		if err := s.shopAddressRepo.Update(ctx, exec, address); err != nil {
			return fmt.Errorf("failed to update address: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// DeleteShopAddress deletes a non-active address from a shop.
func (s *AddressService) DeleteShopAddress(ctx context.Context, shopID uuid.UUID, addressID uuid.UUID) error {
	address, err := s.shopAddressRepo.GetByID(ctx, s.executor, addressID)
	if err != nil {
		return fmt.Errorf("failed to retrieve address: %w", err)
	}
	if address == nil || address.ShopID != shopID {
		return apperror.NewNotFound(addressdomain.ErrAddressNotFound.Error())
	}
	if address.IsActive {
		return apperror.NewConflict(addressdomain.ErrCannotDeleteDefaultAddress.Error())
	}

	if err := s.shopAddressRepo.Delete(ctx, s.executor, addressID); err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}

	return nil
}
