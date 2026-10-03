package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	orderDomain "komecore/internal/modules/order/domain"
	orderRepo "komecore/internal/modules/order/repository"
	"komecore/internal/modules/shipment/domain"

	"github.com/google/uuid"
)

type mockExecutor struct {
	transaction.NoopExecutor
}

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(transaction.Executor) error) error {
	return fn(&mockExecutor{})
}

type mockShipmentRepo struct {
	shipment        *domain.Shipment
	updatedShipment *domain.Shipment
	getErr          error
	updateErr       error
}

func (m *mockShipmentRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*domain.Shipment, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.shipment != nil && m.shipment.ID == id {
		return m.shipment, nil
	}
	return nil, nil
}

func (m *mockShipmentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*domain.Shipment, error) {
	return nil, nil
}

func (m *mockShipmentRepo) ListByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]domain.Shipment, error) {
	return nil, nil
}

func (m *mockShipmentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]domain.Shipment, error) {
	return nil, nil
}

func (m *mockShipmentRepo) Create(_ context.Context, _ transaction.Executor, _ domain.Shipment) error {
	return nil
}

func (m *mockShipmentRepo) Update(_ context.Context, _ transaction.Executor, shipment domain.Shipment) error {
	m.updatedShipment = &shipment
	return m.updateErr
}

type mockOrderRepo struct {
	order        *orderDomain.Order
	updatedState struct {
		id     uuid.UUID
		status orderDomain.OrderStatus
	}
	getErr    error
	updateErr error
}

func (m *mockOrderRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*orderDomain.Order, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.order != nil && m.order.ID == id {
		return m.order, nil
	}
	return nil, nil
}

func (m *mockOrderRepo) GetByNumber(_ context.Context, _ transaction.Executor, _ string) (*orderDomain.Order, error) {
	return nil, nil
}

func (m *mockOrderRepo) UpdateStatus(_ context.Context, _ transaction.Executor, id uuid.UUID, status orderDomain.OrderStatus) error {
	m.updatedState.id = id
	m.updatedState.status = status
	return m.updateErr
}

func (m *mockOrderRepo) UpdateStatusWithSLA(_ context.Context, _ transaction.Executor, id uuid.UUID, status orderDomain.OrderStatus, _ *time.Time, _ *time.Time) error {
	m.updatedState.id = id
	m.updatedState.status = status
	return m.updateErr
}

func (m *mockOrderRepo) Save(_ context.Context, _ transaction.Executor, _ orderDomain.Order) error {
	return nil
}

func (m *mockOrderRepo) FindOrders(_ context.Context, _ transaction.Executor, _ orderRepo.FindOrderParams) ([]orderDomain.Order, int, error) {
	return nil, 0, nil
}

func (m *mockOrderRepo) SetConfirmedAndExpiry(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time, _ time.Time) error {
	return nil
}

func (m *mockOrderRepo) FindExpiredUnfulfilledOrders(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]orderDomain.Order, error) {
	return nil, nil
}

func TestUpdateShipmentStatus_ShipmentNotFound(t *testing.T) {
	sRepo := &mockShipmentRepo{}
	oRepo := &mockOrderRepo{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oRepo)

	_, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID: uuid.New(),
		Status:     domain.ShipmentStatusPacked,
	})

	if err == nil {
		t.Fatal("expected error when shipment not found")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeNotFound {
		t.Errorf("expected NotFound AppError, got: %v", err)
	}
}

func TestUpdateShipmentStatus_InvalidTransition(t *testing.T) {
	shipment := &domain.Shipment{
		ID:     uuid.New(),
		Status: domain.ShipmentStatusDelivered,
	}
	sRepo := &mockShipmentRepo{shipment: shipment}
	oRepo := &mockOrderRepo{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oRepo)

	_, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID: shipment.ID,
		Status:     domain.ShipmentStatusCreated,
	})

	if err == nil {
		t.Fatal("expected error when status transition is invalid")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeInvalidInput {
		t.Errorf("expected InvalidInput AppError, got: %v", err)
	}
}

func TestUpdateShipmentStatus_Success(t *testing.T) {
	shipment := &domain.Shipment{
		ID:     uuid.New(),
		Status: domain.ShipmentStatusCreated,
	}
	sRepo := &mockShipmentRepo{shipment: shipment}
	oRepo := &mockOrderRepo{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oRepo)

	desc := "Items are packed nicely"
	loc := "Central Hub"

	res, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID:  shipment.ID,
		Status:      domain.ShipmentStatusPacked,
		Description: &desc,
		Location:    &loc,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Shipment.Status != domain.ShipmentStatusPacked {
		t.Errorf("expected status Packed, got %s", res.Shipment.Status)
	}

	if sRepo.updatedShipment == nil || sRepo.updatedShipment.Status != domain.ShipmentStatusPacked {
		t.Errorf("expected repository to update shipment to Packed")
	}
}

func TestUpdateShipmentStatus_DeliveredTransitionsOrder(t *testing.T) {
	orderID := uuid.New()
	shipment := &domain.Shipment{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  domain.ShipmentStatusOutForDelivery,
	}
	order := &orderDomain.Order{
		ID:     orderID,
		Status: orderDomain.OrderStatusShipped,
	}

	sRepo := &mockShipmentRepo{shipment: shipment}
	oRepo := &mockOrderRepo{order: order}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oRepo)

	_, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID: shipment.ID,
		Status:     domain.ShipmentStatusDelivered,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if oRepo.updatedState.id != orderID {
		t.Errorf("expected order status update for ID %s, got %s", orderID, oRepo.updatedState.id)
	}

	if oRepo.updatedState.status != orderDomain.OrderStatusDelivered {
		t.Errorf("expected order status to transition to delivered, got %s", oRepo.updatedState.status)
	}
}
