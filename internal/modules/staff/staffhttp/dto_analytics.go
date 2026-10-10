package staffhttp

type RevenueBucketResponse struct {
	Bucket  string `json:"bucket"`
	Revenue int64  `json:"revenue"`
	Count   int64  `json:"count"`
}

type RevenueAnalyticsResponse struct {
	Range        string                  `json:"range"`
	Interval     string                  `json:"interval"`
	TotalRevenue int64                   `json:"total_revenue"`
	TotalOrders  int64                   `json:"total_orders"`
	Buckets      []RevenueBucketResponse `json:"buckets"`
}

type OrderStatusCountResponse struct {
	Status      string `json:"status"`
	Count       int64  `json:"count"`
	TotalAmount int64  `json:"total_amount"`
}

type OrderPipelineResponse struct {
	TotalOrders int64                      `json:"total_orders"`
	TotalAmount int64                      `json:"total_amount"`
	Breakdown   []OrderStatusCountResponse `json:"breakdown"`
}

type TopProductResponse struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	QuantitySold int64  `json:"quantity_sold"`
	TotalRevenue int64  `json:"total_revenue"`
}
