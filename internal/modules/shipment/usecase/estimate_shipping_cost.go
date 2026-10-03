package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	shipping "komecore/internal/infra/shipping"
	transaction "komecore/internal/infra/transactor"
	courierRepo "komecore/internal/modules/courier/repository"
	"komecore/internal/modules/shipment/domain"

	"github.com/google/uuid"
)

type EstimateShippingOptionsUsecase struct {
	shippingProvider shipping.ShippingProvider
	executor         transaction.Executor
	courierRepo      courierRepo.CourierRepository
}

func NewEstimateShippingOptionsUsecase(
	shippingProvider shipping.ShippingProvider,
	executor transaction.Executor,
	courierRepo courierRepo.CourierRepository,
) *EstimateShippingOptionsUsecase {
	return &EstimateShippingOptionsUsecase{
		shippingProvider: shippingProvider,
		executor:         executor,
		courierRepo:      courierRepo,
	}
}

type EstimateShippingOptionsInput struct {
	ShopID      uuid.UUID
	Origin      int
	Destination int
	Weight      int
	PriceFilter *string
}

func (u *EstimateShippingOptionsUsecase) Execute(
	ctx context.Context,
	input EstimateShippingOptionsInput,
) ([]shipping.RateOption, error) {
	if input.Weight <= 0 {
		return nil, apperrors.NewInvalidInput(domain.ErrInvalidWeight.Error())
	}

	var courierCodes []string
	if u.courierRepo != nil {
		codes, err := u.courierRepo.ListAll(ctx, u.executor)
		if err == nil && len(codes) > 0 {
			courierCodes = codes
		}
	}
	if len(courierCodes) == 0 {
		courierCodes = []string{"jne", "jnt", "sicepat", "flat"}
	}

	query := shipping.CalculateRatesInput{
		OriginID:      input.Origin,
		DestinationID: input.Destination,
		Weight:        input.Weight,
		Couriers:      courierCodes,
		PriceFilter:   input.PriceFilter,
	}
	costOptions, err := u.shippingProvider.CalculateRates(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate shipping cost: %w", err)
	}

	return costOptions, nil
}
