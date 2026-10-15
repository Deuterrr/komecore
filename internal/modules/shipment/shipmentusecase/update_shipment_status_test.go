package shipmentusecase

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shipment"

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
	shipment        *shipment.Shipment
	updatedShipment *shipment.Shipment
	getErr          error
	updateErr       error
}

func (m *mockShipmentRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*shipment.Shipment, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.shipment != nil && m.shipment.ID == id {
		return m.shipment, nil
	}
	return nil, nil
}

func (m *mockShipmentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*shipment.Shipment, error) {
	return nil, nil
}

func (m *mockShipmentRepo) ListByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]shipment.Shipment, error) {
	return nil, nil
}

func (m *mockShipmentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]shipment.Shipment, error) {
	return nil, nil
}

func (m *mockShipmentRepo) Create(_ context.Context, _ transaction.Executor, _ shipment.Shipment) error {
	return nil
}

func (m *mockShipmentRepo) Update(_ context.Context, _ transaction.Executor, shipment shipment.Shipment) error {
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
		Status:     shipment.ShipmentStatusPacked,
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
	shp := &shipment.Shipment{
		ID:     uuid.New(),
		Status: shipment.ShipmentStatusDelivered,
	}
	sRepo := &mockShipmentRepo{shipment: shp}
	oUpdater := &mockOrderDeliveryUpdater{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oUpdater)

	_, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID: shp.ID,
		Status:     shipment.ShipmentStatusCreated,
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
	shp := &shipment.Shipment{
		ID:     uuid.New(),
		Status: shipment.ShipmentStatusCreated,
	}
	sRepo := &mockShipmentRepo{shipment: shp}
	oUpdater := &mockOrderDeliveryUpdater{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oUpdater)

	desc := "Items are packed nicely"
	loc := "Central Hub"

	res, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID:  shp.ID,
		Status:      shipment.ShipmentStatusPacked,
		Description: &desc,
		Location:    &loc,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Shipment.Status != shipment.ShipmentStatusPacked {
		t.Errorf("expected status Packed, got %s", res.Shipment.Status)
	}

	if sRepo.updatedShipment == nil || sRepo.updatedShipment.Status != shipment.ShipmentStatusPacked {
		t.Errorf("expected repository to update shipment to Packed")
	}
}

func TestUpdateShipmentStatus_DeliveredTransitionsOrder(t *testing.T) {
	orderID := uuid.New()
	shp := &shipment.Shipment{
		ID:      uuid.New(),
		OrderID: orderID,
		Status:  shipment.ShipmentStatusOutForDelivery,
	}

	sRepo := &mockShipmentRepo{shipment: shp}
	oUpdater := &mockOrderDeliveryUpdater{}
	u := NewUpdateShipmentStatusUsecase(&mockExecutor{}, &mockTransactor{}, sRepo, oUpdater)

	_, err := u.Execute(context.Background(), UpdateShipmentStatusInput{
		ShipmentID: shp.ID,
		Status:     shipment.ShipmentStatusDelivered,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if oUpdater.deliveredID != orderID {
		t.Errorf("expected order status update for ID %s, got %s", orderID, oUpdater.deliveredID)
	}
}
