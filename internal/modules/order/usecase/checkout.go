package usecase

import (
	"context"

	"komecore/internal/common/authctx"
	transaction "komecore/internal/infra/transactor"
	cartDomain "komecore/internal/modules/cart/domain"
	orderRepo "komecore/internal/modules/order/repository"

	"github.com/google/uuid"
)

type CheckoutUsecase struct {
	executor       transaction.Executor
	pricingService orderRepo.PricingService
}

func NewCheckoutUsecase(
	executor transaction.Executor,
	pricingService orderRepo.PricingService,
) *CheckoutUsecase {
	return &CheckoutUsecase{
		executor:       executor,
		pricingService: pricingService,
	}
}

type CheckoutItemInput struct {
	ProductID   uuid.UUID
	CartItemID  *uuid.UUID
	ItemOptions cartDomain.ItemOptions
	Quantity    int
}

type SelectedCourierInput struct {
	Code    string
	Service string
}

type CheckoutShopInput struct {
	ShopID  uuid.UUID
	Items   []CheckoutItemInput
	Courier *SelectedCourierInput
}

type CheckoutInput struct {
	PaymentMethodID *uuid.UUID
	AddressID       *uuid.UUID
	ShopInput       []CheckoutShopInput
}

func (u *CheckoutUsecase) Execute(ctx context.Context, authCtx authctx.AuthContext, input CheckoutInput) (*orderRepo.PricingResult, error) {
	pricingInput := orderRepo.PricingInput{
		CustomerID:      *authCtx.CustomerID,
		AddressID:       input.AddressID,
		PaymentMethodID: input.PaymentMethodID,
		Shops: make(
			[]orderRepo.PricingShopInput,
			0,
			len(input.ShopInput),
		),
	}

	for _, shop := range input.ShopInput {
		var courierCode, courierService *string
		if shop.Courier != nil {
			courierCode = &shop.Courier.Code
			courierService = &shop.Courier.Service
		}

		shopInput := orderRepo.PricingShopInput{
			ShopID:         shop.ShopID,
			CourierCode:    courierCode,
			CourierService: courierService,
			Items: make(
				[]orderRepo.PricingItemInput,
				0,
				len(shop.Items),
			),
		}

		for _, item := range shop.Items {
			shopInput.Items = append(
				shopInput.Items,
				orderRepo.PricingItemInput{
					ProductID:   item.ProductID,
					CartItemID:  item.CartItemID,
					ItemOptions: item.ItemOptions,
					Quantity:    item.Quantity,
				},
			)
		}

		pricingInput.Shops = append(pricingInput.Shops, shopInput)
	}

	return u.pricingService.Calculate(ctx, u.executor, pricingInput)
}
