package shipmentusecase

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shipment/shipmentdomain"

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
	shipment        *shipmentdomain.Shipment
	updatedShipment *shipmentdomain.Shipment
	getErr          error
	updateErr       error
}

func (m *mockShipmentRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*shipmentdomain.Shipment, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.shipment != nil && m.shipment.ID == id {
		return m.shipment, nil
	}
	return nil, nil
}

func (m *mockShipmentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*shipmentdomain.Shipment, error) {
	return nil, nil
}

func (m *mockShipmentRepo) ListByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]shipmentdomain.Shipment, error) {
	return nil, nil
}

func (m *mockShipmentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]shipmentdomain.Shipment, error) {
	return nil, nil
}

func (m *mockShipmentRepo) Create(_ context.Context, _ transaction.Executor, _ shipmentdomain.Shipment) error {
	return nil
}

func (m *mockShipmentRepo) Update(_ context.Context, _ transaction.Executor, shipment shipmentdomain.Shipment) error {
	m.updatedShipment = &shipment
	return m.updateErr
}

type mockOrderDeliveryUpdater struct {
	deliveredID uuid.UUID
	err         error
}

func (m *mockOrderDeliveryUpdater) MarkOrderDelivered(_ context.Context, _ transaction.Executor, orderID uuid.UUID) error {
	m.deliveredID = orderID
	return m.err
}

func TestUpdateShipmentStatus_ShipmentNotFound(t *testing.T) {
	sRepo := &mockShipmentRepo{}
	oUpdater := &mockOrderDeliveryUpdater{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oUpdater)

	_, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID: uuid.New(),
		Status:     shipmentdomain.ShipmentStatusPacked,
	})

	if err == nil {
		t.Fatal("expected error when shipment not found")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperror.ErrTypeNotFound {
		t.Errorf("expected NotFound AppError, got: %v", err)
	}
}

func TestUpdateShipmentStatus_InvalidTransition(t *testing.T) {
	shipment := &shipmentdomain.Shipment{
		ID:     uuid.New(),
		Status: shipmentdomain.ShipmentStatusDelivered,
	}
	sRepo := &mockShipmentRepo{shipment: shipment}
	oUpdater := &mockOrderDeliveryUpdater{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oUpdater)

	_, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID: shipment.ID,
		Status:     shipmentdomain.ShipmentStatusCreated,
	})

	if err == nil {
		t.Fatal("expected error when status transition is invalid")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperror.ErrTypeInvalidInput {
		t.Errorf("expected InvalidInput AppError, got: %v", err)
	}
}

func TestUpdateShipmentStatus_Success(t *testing.T) {
	shipment := &shipmentdomain.Shipment{
		ID:     uuid.New(),
		Status: shipmentdomain.ShipmentStatusCreated,
	}
	sRepo := &mockShipmentRepo{shipment: shipment}
	oUpdater := &mockOrderDeliveryUpdater{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oUpdater)

	desc := "Items are packed nicely"
	loc := "Central Hub"

	res, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID:  shipment.ID,
		Status:      shipmentdomain.ShipmentStatusPacked,
		Description: &desc,
		Location:    &loc,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Shipment.Status != shipmentdomain.ShipmentStatusPacked {
		t.Errorf("expected status Packed, got %s", res.Shipment.Status)
	}

	if sRepo.updatedShipment == nil || sRepo.updatedShipment.Status != shipmentdomain.ShipmentStatusPacked {
		t.Errorf("expected repository to update shipment to Packed")
	}
}

func TestUpdateShipmentStatus_DeliveredTransitionsOrder(t *testing.T) {
	orderID := uuid.New()
	shipment := &shipmentdomain.Shipment{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  shipmentdomain.ShipmentStatusOutForDelivery,
	}

	sRepo := &mockShipmentRepo{shipment: shipment}
	oUpdater := &mockOrderDeliveryUpdater{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oUpdater)

	_, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID: shipment.ID,
		Status:     shipmentdomain.ShipmentStatusDelivered,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if oUpdater.deliveredID != orderID {
		t.Errorf("expected order status update for ID %s, got %s", orderID, oUpdater.deliveredID)
	}
}
