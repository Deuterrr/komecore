package addresshttp

import (
	"time"

	"github.com/google/uuid"
)

type saveCustomerAddressRequest struct {
	AddressID    *string  `json:"address_id"`
	ReceiverName string   `json:"receiver_name"`
	Phone        *string  `json:"phone"`
	IsDefault    *string  `json:"is_default"`
	Province     string   `json:"province"`
	ProvinceID   string   `json:"province_id"`
	City         string   `json:"city"`
	CityID       string   `json:"city_id"`
	District     string   `json:"district"`
	DistrictID   string   `json:"district_id"`
	VillageID    string   `json:"village_id"`
	FullAddress  string   `json:"full_address"`
	PostalCode   string   `json:"postal_code"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
}

func (r *saveCustomerAddressRequest) GetProvince() string {
	if r.Province != "" {
		return r.Province
	}
	return r.ProvinceID
}

func (r *saveCustomerAddressRequest) GetCity() string {
	if r.City != "" {
		return r.City
	}
	return r.CityID
}

func (r *saveCustomerAddressRequest) GetDistrict() string {
	if r.District != "" {
		return r.District
	}
	return r.DistrictID
}

type createShopAddressRequest struct {
	Label       string   `json:"label"`
	Phone       *string  `json:"phone"`
	IsActive    string   `json:"is_active"`
	Province    string   `json:"province"`
	ProvinceID  string   `json:"province_id"`
	City        string   `json:"city"`
	CityID      string   `json:"city_id"`
	District    string   `json:"district"`
	DistrictID  string   `json:"district_id"`
	VillageID   string   `json:"village_id"`
	FullAddress string   `json:"full_address"`
	PostalCode  string   `json:"postal_code"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}

func (r *createShopAddressRequest) GetProvince() string {
	if r.Province != "" {
		return r.Province
	}
	return r.ProvinceID
}

func (r *createShopAddressRequest) GetCity() string {
	if r.City != "" {
		return r.City
	}
	return r.CityID
}

func (r *createShopAddressRequest) GetDistrict() string {
	if r.District != "" {
		return r.District
	}
	return r.DistrictID
}

type customerAddressResponse struct {
	AddressID    uuid.UUID  `json:"address_id"`
	CustomerID   uuid.UUID  `json:"customer_id"`
	ReceiverName string     `json:"receiver_name"`
	Phone        *string    `json:"phone,omitempty"`
	IsDefault    bool       `json:"is_default"`
	Province     string     `json:"province"`
	ProvinceID   string     `json:"province_id,omitempty"`
	City         string     `json:"city"`
	CityID       string     `json:"city_id,omitempty"`
	District     string     `json:"district"`
	DistrictID   string     `json:"district_id,omitempty"`
	VillageID    string     `json:"village_id,omitempty"`
	FullAddress  string     `json:"full_address"`
	PostalCode   string     `json:"postal_code"`
	Latitude     *float64   `json:"latitude,omitempty"`
	Longitude    *float64   `json:"longitude,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

type shopAddressResponse struct {
	ShopID      uuid.UUID  `json:"shop_id"`
	Label       string     `json:"label"`
	Phone       *string    `json:"phone"`
	IsActive    bool       `json:"is_active"`
	Province    string     `json:"province"`
	ProvinceID  string     `json:"province_id,omitempty"`
	City        string     `json:"city"`
	CityID      string     `json:"city_id,omitempty"`
	District    string     `json:"district"`
	DistrictID  string     `json:"district_id,omitempty"`
	VillageID   string     `json:"village_id,omitempty"`
	FullAddress string     `json:"full_address"`
	PostalCode  string     `json:"postal_code"`
	Latitude    *float64   `json:"latitude,omitempty"`
	Longitude   *float64   `json:"longitude,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}
