package staffhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/staffdomain"
	"komecore/internal/modules/staff/staffusecase"

	"github.com/google/uuid"
)

type mockAnalyticsRepo struct {
	buckets     []staffdomain.RevenueBucket
	pipeline    []staffdomain.OrderStatusCount
	topProducts []staffdomain.TopProduct
}

func (m *mockAnalyticsRepo) GetRevenueOverTime(_ context.Context, _ transaction.Executor, _ string, _ time.Time) ([]staffdomain.RevenueBucket, error) {
	return m.buckets, nil
}

func (m *mockAnalyticsRepo) GetOrderPipeline(_ context.Context, _ transaction.Executor) ([]staffdomain.OrderStatusCount, error) {
	return m.pipeline, nil
}

func (m *mockAnalyticsRepo) GetTopProducts(_ context.Context, _ transaction.Executor, _ int) ([]staffdomain.TopProduct, error) {
	return m.topProducts, nil
}

func TestHandler_GetRevenueAnalytics(t *testing.T) {
	repo := &mockAnalyticsRepo{
		buckets: []staffdomain.RevenueBucket{
			{Bucket: time.Now().UTC(), Revenue: 1000000, Count: 10},
		},
	}

	svc := staffusecase.NewStaffService(
		&mockExec{}, &mockTx{}, nil, nil, nil, nil, nil, nil, nil, nil,
	).WithAnalyticsRepository(repo)

	handler := NewStaffHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/staff/analytics/revenue?range=30d&interval=day", nil)
	rr := httptest.NewRecorder()

	err := handler.GetRevenueAnalytics(rr, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp RevenueAnalyticsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalRevenue != 1000000 {
		t.Errorf("expected total revenue 1000000, got %d", resp.TotalRevenue)
	}
	if resp.TotalOrders != 10 {
		t.Errorf("expected total orders 10, got %d", resp.TotalOrders)
	}
	if len(resp.Buckets) != 1 {
		t.Errorf("expected 1 bucket, got %d", len(resp.Buckets))
	}
}

func TestHandler_GetOrderPipelineAnalytics(t *testing.T) {
	repo := &mockAnalyticsRepo{
		pipeline: []staffdomain.OrderStatusCount{
			{Status: "pending", Count: 5, TotalAmount: 500000},
			{Status: "processing", Count: 12, TotalAmount: 1200000},
		},
	}

	svc := staffusecase.NewStaffService(
		&mockExec{}, &mockTx{}, nil, nil, nil, nil, nil, nil, nil, nil,
	).WithAnalyticsRepository(repo)

	handler := NewStaffHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/staff/analytics/orders", nil)
	rr := httptest.NewRecorder()

	err := handler.GetOrderPipelineAnalytics(rr, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp OrderPipelineResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalOrders != 17 {
		t.Errorf("expected total orders 17, got %d", resp.TotalOrders)
	}
	if resp.TotalAmount != 1700000 {
		t.Errorf("expected total amount 1700000, got %d", resp.TotalAmount)
	}
	if len(resp.Breakdown) != 2 {
		t.Errorf("expected 2 items in breakdown, got %d", len(resp.Breakdown))
	}
}

func TestHandler_GetTopProductsAnalytics(t *testing.T) {
	prodID := uuid.New()
	repo := &mockAnalyticsRepo{
		topProducts: []staffdomain.TopProduct{
			{ProductID: prodID, ProductName: "Espresso Beans", QuantitySold: 42, TotalRevenue: 4200000},
		},
	}

	svc := staffusecase.NewStaffService(
		&mockExec{}, &mockTx{}, nil, nil, nil, nil, nil, nil, nil, nil,
	).WithAnalyticsRepository(repo)

	handler := NewStaffHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/staff/analytics/top-products?limit=5", nil)
	rr := httptest.NewRecorder()

	err := handler.GetTopProductsAnalytics(rr, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp []TopProductResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 product in response, got %d", len(resp))
	}
	if resp[0].ProductID != prodID.String() {
		t.Errorf("expected product ID %s, got %s", prodID, resp[0].ProductID)
	}
	if resp[0].QuantitySold != 42 {
		t.Errorf("expected quantity sold 42, got %d", resp[0].QuantitySold)
	}
}
