package shipment

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/apperror"

	"github.com/google/uuid"
)

func TestUpdateShipment_ShipmentNotFound(t *testing.T) {
	sRepo := &mockShipmentRepo{}
	svc := NewService(&mockExecutor{}, &mockTransactor{}, sRepo, nil, nil, nil)

	_, err := svc.UpdateShipment(context.Background(), UpdateShipmentInput{
		ShipmentID: uuid.New(),
	})

	if err == nil {
		t.Fatal("expected error when shipment not found")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperror.ErrTypeNotFound {
		t.Errorf("expected NotFound AppError, got: %v", err)
	}
}

func TestUpdateShipment_Success(t *testing.T) {
	tracking := "JNE12345"
	courier := "jne"
	service := "REG"

	shp := &Shipment{
		ID:                uuid.New(),
		FulfillmentMethod: FulfillmentMethodCourier,
		Courier:           "placeholder",
		Service:           "placeholder",
		Weight:            1000,
	}

	sRepo := &mockShipmentRepo{shipment: shp}
	svc := NewService(&mockExecutor{}, &mockTransactor{}, sRepo, nil, nil, nil)

	res, err := svc.UpdateShipment(context.Background(), UpdateShipmentInput{
		ShipmentID:     shp.ID,
		TrackingNumber: &tracking,
		Courier:        &courier,
		Service:        &service,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Shipment.TrackingNumber == nil || *res.Shipment.TrackingNumber != tracking {
		t.Errorf("expected tracking number %s, got %v", tracking, res.Shipment.TrackingNumber)
	}

	if res.Shipment.Courier != courier {
		t.Errorf("expected courier %s, got %s", courier, res.Shipment.Courier)
	}

	if res.Shipment.Service != service {
		t.Errorf("expected service %s, got %s", service, res.Shipment.Service)
	}

	if sRepo.updatedShipment == nil || sRepo.updatedShipment.Courier != courier {
		t.Errorf("expected shipment to be updated in database")
	}
}

func TestUpdateShipment_ValidationError(t *testing.T) {
	emptyCourier := ""
	shp := &Shipment{
		ID:                uuid.New(),
		FulfillmentMethod: FulfillmentMethodCourier,
		Courier:           "jne",
		Service:           "REG",
		Weight:            1000,
	}

	sRepo := &mockShipmentRepo{shipment: shp}
	svc := NewService(&mockExecutor{}, &mockTransactor{}, sRepo, nil, nil, nil)

	// In courier mode, empty courier is invalid
	_, err := svc.UpdateShipment(context.Background(), UpdateShipmentInput{
		ShipmentID: shp.ID,
		Courier:    &emptyCourier,
	})

	if err == nil {
		t.Fatal("expected validation error when courier is empty")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperror.ErrTypeInvalidInput {
		t.Errorf("expected InvalidInput AppError, got: %v", err)
	}
}
