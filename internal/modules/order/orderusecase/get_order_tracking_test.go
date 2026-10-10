package orderusecase

import (
	"context"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/modules/shipment/shipmentdomain"

	"github.com/google/uuid"
)

type gotMockExecutor struct {
	transaction.NoopExecutor
}

type gotMockOrderRepo struct {
	order *orderdomain.Order
	err   error
}

func (m *gotMockOrderRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*orderdomain.Order, error) {
	if m.order != nil && m.order.ID == id {
		return m.order, nil
	}
	return nil, m.err
}
func (m *gotMockOrderRepo) GetByNumber(_ context.Context, _ transaction.Executor, _ string) (*orderdomain.Order, error) {
	return nil, nil
}
func (m *gotMockOrderRepo) UpdateStatus(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ orderdomain.OrderStatus) error {
	return nil
}
func (m *gotMockOrderRepo) UpdateStatusWithSLA(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ orderdomain.OrderStatus, _ *time.Time, _ *time.Time) error {
	return nil
}
func (m *gotMockOrderRepo) Save(_ context.Context, _ transaction.Executor, _ orderdomain.Order) error {
	return nil
}
func (m *gotMockOrderRepo) FindOrders(_ context.Context, _ transaction.Executor, _ orderrepo.FindOrderParams) ([]orderdomain.Order, int, error) {
	return nil, 0, nil
}
func (m *gotMockOrderRepo) SetConfirmedAndExpiry(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time, _ time.Time) error {
	return nil
}
func (m *gotMockOrderRepo) FindExpiredUnfulfilledOrders(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]orderdomain.Order, error) {
	return nil, nil
}

type gotMockShipmentRepo struct {
	shipment  *shipmentdomain.Shipment
	shipments []shipmentdomain.Shipment
	err       error
}

func (m *gotMockShipmentRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*shipmentdomain.Shipment, error) {
	if m.shipment != nil && m.shipment.ID == id {
		return m.shipment, nil
	}
	return nil, m.err
}

func (m *gotMockShipmentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) (*shipmentdomain.Shipment, error) {
	if m.shipment != nil && m.shipment.OrderID == orderID {
		return m.shipment, nil
	}
	return nil, m.err
}

func (m *gotMockShipmentRepo) ListByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) ([]shipmentdomain.Shipment, error) {
	if m.shipments != nil {
		return m.shipments, nil
	}
	if m.shipment != nil && m.shipment.OrderID == orderID {
		return []shipmentdomain.Shipment{*m.shipment}, nil
	}
	return nil, m.err
}

func (m *gotMockShipmentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]shipmentdomain.Shipment, error) {
	return nil, nil
}
func (m *gotMockShipmentRepo) Create(_ context.Context, _ transaction.Executor, _ shipmentdomain.Shipment) error {
	return nil
}
func (m *gotMockShipmentRepo) Update(_ context.Context, _ transaction.Executor, _ shipmentdomain.Shipment) error {
	return nil
}

func TestGetOrderTracking_OrderNotFound(t *testing.T) {
	uc := NewGetOrderTrackingUsecase(
		&gotMockExecutor{},
		&gotMockOrderRepo{order: nil},
		&gotMockShipmentRepo{},
	)

	result, err := uc.Execute(context.Background(), GetOrderTrackingInput{
		OrderID:    uuid.New(),
		CustomerID: uuid.New(),
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "order not found" {
		t.Fatalf("expected 'order not found' error, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}
}

func TestGetOrderTracking_CustomerIDMismatch(t *testing.T) {
	orderID := uuid.New()
	customerID := uuid.New()
	mismatchCustomerID := uuid.New()

	order := &orderdomain.Order{
		ID:         orderID,
		CustomerID: customerID,
	}

	uc := NewGetOrderTrackingUsecase(
		&gotMockExecutor{},
		&gotMockOrderRepo{order: order},
		&gotMockShipmentRepo{},
	)

	result, err := uc.Execute(context.Background(), GetOrderTrackingInput{
		OrderID:    orderID,
		CustomerID: mismatchCustomerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "not authorized" {
		t.Fatalf("expected 'not authorized' error, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}
}

func TestGetOrderTracking_ShipmentNotFound(t *testing.T) {
	orderID := uuid.New()
	customerID := uuid.New()

	order := &orderdomain.Order{
		ID:         orderID,
		CustomerID: customerID,
	}

	uc := NewGetOrderTrackingUsecase(
		&gotMockExecutor{},
		&gotMockOrderRepo{order: order},
		&gotMockShipmentRepo{shipment: nil},
	)

	result, err := uc.Execute(context.Background(), GetOrderTrackingInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "shipment not found" {
		t.Fatalf("expected 'shipment not found' error, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}
}

func TestGetOrderTracking_TimelineProgress(t *testing.T) {
	orderID := uuid.New()
	customerID := uuid.New()
	shipmentID := uuid.New()

	now := time.Now()
	shippedAt := now.Add(1 * time.Hour)
	deliveredAt := now.Add(5 * time.Hour)
	trackingNo := "TRK123456"

	order := &orderdomain.Order{
		ID:         orderID,
		CustomerID: customerID,
	}

	shipment := &shipmentdomain.Shipment{
		ID:                shipmentID,
		OrderID:           orderID,
		Status:            shipmentdomain.ShipmentStatusDelivered,
		FulfillmentMethod: shipmentdomain.FulfillmentMethodCourier,
		Courier:           "JNE",
		Service:           "REG",
		TrackingNumber:    &trackingNo,
		CreatedAt:         now,
		ShippedAt:         &shippedAt,
		DeliveredAt:       &deliveredAt,
	}

	uc := NewGetOrderTrackingUsecase(
		&gotMockExecutor{},
		&gotMockOrderRepo{order: order},
		&gotMockShipmentRepo{shipment: shipment},
	)

	result, err := uc.Execute(context.Background(), GetOrderTrackingInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if len(result.Timeline) != 3 {
		t.Fatalf("expected 3 timeline events, got %d", len(result.Timeline))
	}
	if result.Timeline[0].Status != string(shipmentdomain.ShipmentStatusCreated) {
		t.Errorf("expected first event created, got %s", result.Timeline[0].Status)
	}
	if result.Timeline[1].Status != string(shipmentdomain.ShipmentStatusShipped) {
		t.Errorf("expected second event shipped, got %s", result.Timeline[1].Status)
	}
	if result.Timeline[2].Status != string(shipmentdomain.ShipmentStatusDelivered) {
		t.Errorf("expected third event delivered, got %s", result.Timeline[2].Status)
	}
}

func TestGetOrderTracking_ExecuteForStaff(t *testing.T) {
	orderID := uuid.New()
	customerID := uuid.New()
	shipmentID := uuid.New()
	now := time.Now()

	order := &orderdomain.Order{
		ID:         orderID,
		CustomerID: customerID,
	}

	shipment := &shipmentdomain.Shipment{
		ID:                shipmentID,
		OrderID:           orderID,
		Status:            shipmentdomain.ShipmentStatusCreated,
		FulfillmentMethod: shipmentdomain.FulfillmentMethodCourier,
		Courier:           "JNE",
		Service:           "REG",
		CreatedAt:         now,
	}

	uc := NewGetOrderTrackingUsecase(
		&gotMockExecutor{},
		&gotMockOrderRepo{order: order},
		&gotMockShipmentRepo{shipment: shipment},
	)

	result, err := uc.Execute(context.Background(), GetOrderTrackingInput{
		OrderID:    orderID,
		CustomerID: uuid.Nil,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Timeline) != 1 {
		t.Fatalf("expected 1 timeline event, got %d", len(result.Timeline))
	}
}
