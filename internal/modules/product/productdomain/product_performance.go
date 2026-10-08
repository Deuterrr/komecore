package productdomain

import (
	"time"

	"github.com/google/uuid"
)

type ProductPerformance struct {
	ProductID            uuid.UUID
	CostPrice            *int64
	SupplierLeadTimeDays *int
	GrossMarginPct       *float64
	ViewCount            int64
	CreatedAt            time.Time
	UpdatedAt            *time.Time
}

type ProductStockEvent struct {
	ID         uuid.UUID
	ProductID  uuid.UUID
	ShopID     uuid.UUID
	Available  int
	RecordedAt time.Time
}

type ProductStats struct {
	Product           Product
	Performance       ProductPerformance
	TotalStock        int
	ReservedStock     int
	SalesVelocity7d   int
	SalesVelocity30d  int
	SalesVelocity90d  int
	ConversionRate    float64
	RevenueContribPct float64
	ReturnRate        *float64
	AverageRating     *float64
	ReviewCount       *int
	Thumbnail         string
}
