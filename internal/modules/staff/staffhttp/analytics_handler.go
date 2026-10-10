package staffhttp

import (
	"net/http"
	"strconv"
	"time"

	apphttp "komecore/internal/common/http"
)

func (h *staffHandler) GetRevenueAnalytics(w http.ResponseWriter, r *http.Request) error {
	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "30d"
	}
	intervalParam := r.URL.Query().Get("interval")
	if intervalParam == "" {
		intervalParam = "day"
	}

	result, err := h.service.GetRevenueAnalytics(r.Context(), rangeParam, intervalParam)
	if err != nil {
		return err
	}

	buckets := make([]RevenueBucketResponse, 0, len(result.Buckets))
	for _, b := range result.Buckets {
		buckets = append(buckets, RevenueBucketResponse{
			Bucket:  b.Bucket.UTC().Format(time.RFC3339),
			Revenue: b.Revenue,
			Count:   b.Count,
		})
	}

	resp := RevenueAnalyticsResponse{
		Range:        result.Range,
		Interval:     result.Interval,
		TotalRevenue: result.TotalRevenue,
		TotalOrders:  result.TotalOrders,
		Buckets:      buckets,
	}

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *staffHandler) GetOrderPipelineAnalytics(w http.ResponseWriter, r *http.Request) error {
	result, err := h.service.GetOrderPipelineAnalytics(r.Context())
	if err != nil {
		return err
	}

	breakdown := make([]OrderStatusCountResponse, 0, len(result.Breakdown))
	for _, item := range result.Breakdown {
		breakdown = append(breakdown, OrderStatusCountResponse{
			Status:      item.Status,
			Count:       item.Count,
			TotalAmount: item.TotalAmount,
		})
	}

	resp := OrderPipelineResponse{
		TotalOrders: result.TotalOrders,
		TotalAmount: result.TotalAmount,
		Breakdown:   breakdown,
	}

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}

func (h *staffHandler) GetTopProductsAnalytics(w http.ResponseWriter, r *http.Request) error {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	products, err := h.service.GetTopProductsAnalytics(r.Context(), limit)
	if err != nil {
		return err
	}

	resp := make([]TopProductResponse, 0, len(products))
	for _, p := range products {
		resp = append(resp, TopProductResponse{
			ProductID:    p.ProductID.String(),
			ProductName:  p.ProductName,
			QuantitySold: p.QuantitySold,
			TotalRevenue: p.TotalRevenue,
		})
	}

	apphttp.WriteJSON(w, http.StatusOK, resp)
	return nil
}
