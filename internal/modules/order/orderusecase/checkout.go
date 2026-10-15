package orderusecase

import (
	"context"

	"komecore/internal/authctx"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart"
	"komecore/internal/modules/order"

	"github.com/google/uuid"
)

type CheckoutUsecase struct {
	executor       transaction.Executor
	pricingService order.PricingService
}

func NewCheckoutUsecase(
	executor transaction.Executor,
	pricingService order.PricingService,
) *CheckoutUsecase {
	return &CheckoutUsecase{
		executor:       executor,
		pricingService: pricingService,
	}
}

type CheckoutItemInput struct {
	ProductID   uuid.UUID
	CartItemID  *uuid.UUID
	ItemOptions cart.ItemOptions
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
	CouponCode      *string
	ShopInput       []CheckoutShopInput
}

func (u *CheckoutUsecase) Execute(ctx context.Context, authCtx authctx.AuthContext, input CheckoutInput) (*order.PricingResult, error) {
	pricingInput := order.PricingInput{
		CustomerID:      *authCtx.CustomerID,
		AddressID:       input.AddressID,
		PaymentMethodID: input.PaymentMethodID,
		CouponCode:      input.CouponCode,
		Shops: make(
			[]order.PricingShopInput,
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

		shopInput := order.PricingShopInput{
			ShopID:         shop.ShopID,
			CourierCode:    courierCode,
			CourierService: courierService,
			Items: make(
				[]order.PricingItemInput,
				0,
				len(shop.Items),
			),
		}

		for _, item := range shop.Items {
			shopInput.Items = append(
				shopInput.Items,
				order.PricingItemInput{
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
