package orderusecase

import (
	"context"

	"komecore/internal/common/authctx"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart/cartdomain"
	"komecore/internal/modules/order/orderrepo"

	"github.com/google/uuid"
)

type CheckoutUsecase struct {
	executor       transaction.Executor
	pricingService orderrepo.PricingService
}

func NewCheckoutUsecase(
	executor transaction.Executor,
	pricingService orderrepo.PricingService,
) *CheckoutUsecase {
	return &CheckoutUsecase{
		executor:       executor,
		pricingService: pricingService,
	}
}

type CheckoutItemInput struct {
	ProductID   uuid.UUID
	CartItemID  *uuid.UUID
	ItemOptions cartdomain.ItemOptions
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

func (u *CheckoutUsecase) Execute(ctx context.Context, authCtx authctx.AuthContext, input CheckoutInput) (*orderrepo.PricingResult, error) {
	pricingInput := orderrepo.PricingInput{
		CustomerID:      *authCtx.CustomerID,
		AddressID:       input.AddressID,
		PaymentMethodID: input.PaymentMethodID,
		Shops: make(
			[]orderrepo.PricingShopInput,
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

		shopInput := orderrepo.PricingShopInput{
			ShopID:         shop.ShopID,
			CourierCode:    courierCode,
			CourierService: courierService,
			Items: make(
				[]orderrepo.PricingItemInput,
				0,
				len(shop.Items),
			),
		}

		for _, item := range shop.Items {
			shopInput.Items = append(
				shopInput.Items,
				orderrepo.PricingItemInput{
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
