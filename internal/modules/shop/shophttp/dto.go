package shophttp

import (
	"time"

	"github.com/google/uuid"
)

type saveShopRequest struct {
	ShopID         *string `json:"id"`
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	IsActive       *string `json:"is_active"`
	ApprovalStatus *string `json:"approval_status"`
}

type getShopResponse struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Slug           string     `json:"slug"`
	Description    *string    `json:"description"`
	IsActive       bool       `json:"is_active"`
	ApprovalStatus string     `json:"approval_status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

type listShopsResponse struct {
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
	Total int               `json:"total"`
	Shops []getShopResponse `json:"shops"`
}

type shopAddressResponse struct {
	ID          uuid.UUID  `json:"id"`
	Label       string     `json:"label"`
	Phone       *string    `json:"phone"`
	IsActive    bool       `json:"is_active"`
	Province    string     `json:"province"`
	City        string     `json:"city"`
	District    string     `json:"district"`
	ProvinceID  string     `json:"province_id,omitempty"`
	CityID      string     `json:"city_id,omitempty"`
	DistrictID  string     `json:"district_id,omitempty"`
	VillageID   string     `json:"village_id,omitempty"`
	FullAddress string     `json:"full_address"`
	PostalCode  string     `json:"postal_code"`
	Latitude    *float64   `json:"latitude,omitempty"`
	Longitude   *float64   `json:"longitude,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type shopProductInventoryResponse struct {
	TotalStock    int `json:"total_stock"`
	ReservedStock int `json:"reserved_stock"`
	Available     int `json:"available"`
}

type shopProductResponse struct {
	ID          uuid.UUID                    `json:"id"`
	SKU         string                       `json:"sku"`
	Name        string                       `json:"name"`
	Slug        string                       `json:"slug"`
	Description *string                      `json:"description"`
	Status      string                       `json:"status"`
	Price       int64                        `json:"price"`
	Weight      *float64                     `json:"weight"`
	Inventory   shopProductInventoryResponse `json:"inventory"`
	CreatedAt   time.Time                    `json:"created_at"`
	UpdatedAt   *time.Time                   `json:"updated_at"`
}
