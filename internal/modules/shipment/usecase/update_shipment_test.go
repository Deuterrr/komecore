package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	apperrors "komecore/internal/common/errors"
	"komecore/internal/modules/shipment/domain"
)

func TestUpdateShipment_ShipmentNotFound(t *testing.T) {
	sRepo := &mockShipmentRepo{}
	u := NewUpdateShipmentUsecase(&mockExecutor{}, &mockTransactor{}, sRepo)

	_, err := u.Execute(context.Background(), UpdateShipmentInput{
		ShipmentID: uuid.New(),
	})

	if err == nil {
		t.Fatal("expected error when shipment not found")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeNotFound {
		t.Errorf("expected NotFound AppError, got: %v", err)
	}
}

func TestUpdateShipment_Success(t *testing.T) {
	tracking := "JNE12345"
	courier := "jne"
	service := "REG"

	shipment := &domain.Shipment{
		ID:                uuid.New(),
		FulfillmentMethod: domain.FulfillmentMethodCourier,
		Courier:           "placeholder",
		Service:           "placeholder",
		Weight:            1000,
	}

	sRepo := &mockShipmentRepo{shipment: shipment}
	u := NewUpdateShipmentUsecase(&mockExecutor{}, &mockTransactor{}, sRepo)

	res, err := u.Execute(context.Background(), UpdateShipmentInput{
		ShipmentID:     shipment.ID,
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
	shipment := &domain.Shipment{
		ID:                uuid.New(),
		FulfillmentMethod: domain.FulfillmentMethodCourier,
		Courier:           "jne",
		Service:           "REG",
		Weight:            1000,
	}

	sRepo := &mockShipmentRepo{shipment: shipment}
	u := NewUpdateShipmentUsecase(&mockExecutor{}, &mockTransactor{}, sRepo)

	// In courier mode, empty courier is invalid
	_, err := u.Execute(context.Background(), UpdateShipmentInput{
		ShipmentID: shipment.ID,
		Courier:    &emptyCourier,
	})

	if err == nil {
		t.Fatal("expected validation error when courier is empty")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeInvalidInput {
		t.Errorf("expected InvalidInput AppError, got: %v", err)
	}
}
