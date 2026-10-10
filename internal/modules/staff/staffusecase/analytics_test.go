package staffusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/staffdomain"

	"github.com/google/uuid"
)

type mockAnalyticsRepo struct {
	revenueBuckets []staffdomain.RevenueBucket
	pipeline       []staffdomain.OrderStatusCount
	topProducts    []staffdomain.TopProduct
	err            error
	lastInterval   string
	lastLimit      int
}

func (m *mockAnalyticsRepo) GetRevenueOverTime(_ context.Context, _ transaction.Executor, interval string, _ time.Time) ([]staffdomain.RevenueBucket, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.lastInterval = interval
	return m.revenueBuckets, nil
}

func (m *mockAnalyticsRepo) GetOrderPipeline(_ context.Context, _ transaction.Executor) ([]staffdomain.OrderStatusCount, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.pipeline, nil
}

func (m *mockAnalyticsRepo) GetTopProducts(_ context.Context, _ transaction.Executor, limit int) ([]staffdomain.TopProduct, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.lastLimit = limit
	return m.topProducts, nil
}

func TestStaffService_GetRevenueAnalytics_Success(t *testing.T) {
	now := time.Now()
	repo := &mockAnalyticsRepo{
		revenueBuckets: []staffdomain.RevenueBucket{
			{Bucket: now.AddDate(0, 0, -2), Revenue: 500000, Count: 5},
			{Bucket: now.AddDate(0, 0, -1), Revenue: 750000, Count: 8},
		},
	}

	service := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		nil, nil, nil, nil, nil, nil, nil, nil,
	).WithAnalyticsRepository(repo)

	result, err := service.GetRevenueAnalytics(context.Background(), "30d", "day")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalRevenue != 1250000 {
		t.Errorf("expected total revenue 1250000, got %d", result.TotalRevenue)
	}
	if result.TotalOrders != 13 {
		t.Errorf("expected total orders 13, got %d", result.TotalOrders)
	}
	if len(result.Buckets) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(result.Buckets))
	}
	if repo.lastInterval != "day" {
		t.Errorf("expected interval day, got %s", repo.lastInterval)
	}
}

func TestStaffService_GetRevenueAnalytics_FallbackDefaults(t *testing.T) {
	repo := &mockAnalyticsRepo{}
	service := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		nil, nil, nil, nil, nil, nil, nil, nil,
	).WithAnalyticsRepository(repo)

	result, err := service.GetRevenueAnalytics(context.Background(), "invalid_range", "invalid_interval")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Range != "30d" {
		t.Errorf("expected range fallback to 30d, got %s", result.Range)
	}
	if result.Interval != "day" {
		t.Errorf("expected interval fallback to day, got %s", result.Interval)
	}
}

func TestStaffService_GetRevenueAnalytics_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("database connection failed")
	repo := &mockAnalyticsRepo{err: expectedErr}
	service := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		nil, nil, nil, nil, nil, nil, nil, nil,
	).WithAnalyticsRepository(repo)

	_, err := service.GetRevenueAnalytics(context.Background(), "30d", "day")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStaffService_GetOrderPipelineAnalytics_Success(t *testing.T) {
	repo := &mockAnalyticsRepo{
		pipeline: []staffdomain.OrderStatusCount{
			{Status: "pending", Count: 10, TotalAmount: 1000000},
			{Status: "confirmed", Count: 25, TotalAmount: 2500000},
			{Status: "delivered", Count: 100, TotalAmount: 10000000},
		},
	}

	service := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		nil, nil, nil, nil, nil, nil, nil, nil,
	).WithAnalyticsRepository(repo)

	result, err := service.GetOrderPipelineAnalytics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalOrders != 135 {
		t.Errorf("expected total orders 135, got %d", result.TotalOrders)
	}
	if result.TotalAmount != 13500000 {
		t.Errorf("expected total amount 13500000, got %d", result.TotalAmount)
	}
	if len(result.Breakdown) != 3 {
		t.Errorf("expected 3 statuses in breakdown, got %d", len(result.Breakdown))
	}
}

func TestStaffService_GetTopProductsAnalytics_Bounds(t *testing.T) {
	repo := &mockAnalyticsRepo{
		topProducts: []staffdomain.TopProduct{
			{ProductID: uuid.New(), ProductName: "Coffee Beans", QuantitySold: 50, TotalRevenue: 5000000},
		},
	}

	service := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		nil, nil, nil, nil, nil, nil, nil, nil,
	).WithAnalyticsRepository(repo)

	// Test negative/zero limit bounds to 10
	products, err := service.GetTopProductsAnalytics(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastLimit != 10 {
		t.Errorf("expected default limit 10, got %d", repo.lastLimit)
	}
	if len(products) != 1 {
		t.Errorf("expected 1 product, got %d", len(products))
	}

	// Test excess limit bounds to 100
	_, err = service.GetTopProductsAnalytics(context.Background(), 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastLimit != 100 {
		t.Errorf("expected clamped limit 100, got %d", repo.lastLimit)
	}
}
