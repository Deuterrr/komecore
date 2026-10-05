package http

import (
	"time"

	"github.com/google/uuid"
)

type productStatusDTO string

const (
	ProductStatusActive   productStatusDTO = "active"
	ProductStatusInactive productStatusDTO = "inactive"
	ProductStatusArchived productStatusDTO = "archived"
)

type saveProductRequest struct {
	ID                   *string          `json:"id"`
	SKU                  string           `json:"sku"`
	Name                 string           `json:"name"`
	Description          *string          `json:"description"`
	IsAvailable          bool             `json:"is_available"`
	Status               productStatusDTO `json:"status"`
	Price                int64            `json:"price"`
	Weight               *float64         `json:"weight"`
	CostPrice            *int64           `json:"cost_price"`
	SupplierLeadTimeDays *int             `json:"supplier_lead_time_days"`
}

type productImageResponse struct {
	Thumbnail *string `json:"thumbnail"`
	Preview   *string `json:"preview"`
	Detail    *string `json:"detail"`
}

type productAvailabilityResponse struct {
	ShopID string `json:"shop_id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Stock  int    `json:"stock"`
}

type productBaseResponse struct {
	ID            uuid.UUID            `json:"id"`
	SKU           string               `json:"sku"`
	Name          string               `json:"name"`
	Slug          string               `json:"slug"`
	Status        string               `json:"status"`
	IsAvailable   bool                 `json:"is_available"`
	Price         int64                `json:"price"`
	TotalStock    int                  `json:"stock"`
	AverageRating float64              `json:"average_rating"`
	ReviewCount   int                  `json:"review_count"`
	Banner        productImageResponse `json:"banner"`
}

type productCatalogResponse struct {
	productBaseResponse

	Availability []productAvailabilityResponse `json:"availability"`
}

type productDetailResponse struct {
	productBaseResponse

	Description  *string                       `json:"description"`
	Weight       *float64                      `json:"weight"`
	UpdatedAt    *time.Time                    `json:"updated_at"`
	Gallery      []productImageResponse        `json:"gallery"`
	Availability []productAvailabilityResponse `json:"availability"`
}

type productInventoryView struct {
	ID         uuid.UUID `json:"id"`
	ShopID     uuid.UUID `json:"shop_id"`
	TotalStock int       `json:"stock"`
	Available  int       `json:"available"`
}

type productStatsResponse struct {
	ID     uuid.UUID `json:"id"`
	SKU    string    `json:"sku"`
	Name   string    `json:"name"`
	Slug   string    `json:"slug"`
	Status string    `json:"status"`
	Price  int64     `json:"price"`

	CostPrice            *int64   `json:"cost_price"`
	SupplierLeadTimeDays *int     `json:"supplier_lead_time_days"`
	GrossMarginPct       *float64 `json:"gross_margin_pct"`
	ViewCount            int64    `json:"view_count"`
	TotalStock           int      `json:"stock"`

	SalesVelocity7d  int `json:"sales_velocity_7d"`
	SalesVelocity30d int `json:"sales_velocity_30d"`
	SalesVelocity90d int `json:"sales_velocity_90d"`

	ConversionRate    float64 `json:"conversion_rate"`
	RevenueContribPct float64 `json:"revenue_contribution_percentage"`

	ReturnRate    *float64 `json:"return_rate"`
	AverageRating *float64 `json:"average_rating"`
	ReviewCount   *int     `json:"review_count"`

	Thumbnail *string `json:"thumbnail"`
}
