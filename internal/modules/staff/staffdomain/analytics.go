package staffdomain

import (
	"time"

	"github.com/google/uuid"
)

type RevenueBucket struct {
	Bucket  time.Time `json:"bucket"`
	Revenue int64     `json:"revenue"`
	Count   int64     `json:"count"`
}

type RevenueAnalytics struct {
	Range        string          `json:"range"`
	Interval     string          `json:"interval"`
	TotalRevenue int64           `json:"total_revenue"`
	TotalOrders  int64           `json:"total_orders"`
	Buckets      []RevenueBucket `json:"buckets"`
}

type OrderStatusCount struct {
	Status      string `json:"status"`
	Count       int64  `json:"count"`
	TotalAmount int64  `json:"total_amount"`
}

type OrderPipelineAnalytics struct {
	TotalOrders int64              `json:"total_orders"`
	TotalAmount int64              `json:"total_amount"`
	Breakdown   []OrderStatusCount `json:"breakdown"`
}

type TopProduct struct {
	ProductID    uuid.UUID `json:"product_id"`
	ProductName  string    `json:"product_name"`
	QuantitySold int64     `json:"quantity_sold"`
	TotalRevenue int64     `json:"total_revenue"`
}
