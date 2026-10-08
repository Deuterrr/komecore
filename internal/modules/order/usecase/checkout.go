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

type CheckoutAddressResult struct {
	ID            uuid.UUID
	RecipientName string
	Phone         *string
	FullAddress   string
}

type CheckoutCouriersResult struct {
	Code    string
	Service string
	Name    string
	ETD     string
	Fee     int64
}

type CheckoutItemResult struct {
	ProductID   uuid.UUID
	CartItemID  *uuid.UUID
	ItemOptions cartDomain.ItemOptions
	ShopID      uuid.UUID
	Name        string
	Price       int64
	Quantity    int
	Subtotal    int64
	TotalWeight int
}

type SelectedCourierResult struct {
	Code    string
	Service string
	Fee     int64
}

type ShopResult struct {
	ShopID          uuid.UUID
	ShopSlug        string
	ShopName        string
	Items           []CheckoutItemResult
	SelectedCourier *SelectedCourierResult
	CostCouriers    []CheckoutCouriersResult
	Subtotal        int64
	Total           int64
}

type CheckoutPaymentMethodResult struct {
	ID          uuid.UUID
	Name        string
	Type        string
	Description string
	Fee         int64
	Subtotal    int64
	Total       int64
}

type CheckoutResult struct {
	Address               CheckoutAddressResult
	PaymentMethods        []CheckoutPaymentMethodResult
	SelectedPaymentMethod *CheckoutPaymentMethodResult
	Shops                 []ShopResult
	Subtotal              int64
	TotalShippingFee      int64
	TotalAll              int64
}

func (u *CheckoutUsecase) Execute(ctx context.Context, authCtx authctx.AuthContext, input CheckoutInput) (*CheckoutResult, error) {
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

	pricingResult, err := u.pricingService.Calculate(ctx, u.executor, pricingInput)
	if err != nil {
		return nil, err
	}

	result := &CheckoutResult{
		Address: CheckoutAddressResult{
			ID:            pricingResult.Address.ID,
			RecipientName: pricingResult.Address.RecipientName,
			Phone:         pricingResult.Address.Phone,
			FullAddress:   pricingResult.Address.FullAddress,
		},
		Subtotal:         pricingResult.Subtotal,
		TotalShippingFee: pricingResult.TotalShippingFee,
		TotalAll:         pricingResult.GrandTotal,
		Shops: make(
			[]ShopResult,
			0,
			len(pricingResult.Shops),
		),
	}

	for _, shop := range pricingResult.Shops {
		shopRes := ShopResult{
			ShopID:   shop.ShopID,
			ShopName: shop.ShopName,
			ShopSlug: shop.ShopSlug,
			Subtotal: shop.Subtotal,
			Total:    shop.Total,
			Items: make(
				[]CheckoutItemResult,
				0,
				len(shop.Items),
			),
			SelectedCourier: &SelectedCourierResult{
				Code:    shop.SelectedCourier.Code,
				Service: shop.SelectedCourier.Service,
				Fee:     shop.SelectedCourier.Fee,
			},
			CostCouriers: make(
				[]CheckoutCouriersResult,
				0,
				len(shop.CourierOptions),
			),
		}

		for _, item := range shop.Items {
			shopRes.Items = append(
				shopRes.Items,
				CheckoutItemResult{
					ProductID:   item.ProductID,
					CartItemID:  item.CartItemID,
					ItemOptions: item.ItemOptions,
					ShopID:      shop.ShopID,
					Name:        item.ProductName,
					Price:       item.UnitPrice,
					Quantity:    item.Quantity,
					Subtotal:    item.Subtotal,
					TotalWeight: item.WeightGrams,
				},
			)
		}

		for _, opt := range shop.CourierOptions {
			shopRes.CostCouriers = append(
				shopRes.CostCouriers,
				CheckoutCouriersResult{
					Code:    opt.Code,
					Service: opt.Service,
					Name:    opt.Name,
					ETD:     opt.ETD,
					Fee:     opt.Fee,
				},
			)
		}

		result.Shops = append(result.Shops, shopRes)
	}

	if pricingResult.SelectedPaymentMethod != nil {
		result.SelectedPaymentMethod = &CheckoutPaymentMethodResult{
			ID:          pricingResult.SelectedPaymentMethod.PaymentMethodID,
			Name:        pricingResult.SelectedPaymentMethod.Name,
			Type:        pricingResult.SelectedPaymentMethod.Type,
			Description: pricingResult.SelectedPaymentMethod.Description,
			Fee:         pricingResult.SelectedPaymentMethod.Fee,
			Subtotal:    pricingResult.SelectedPaymentMethod.Subtotal,
			Total:       pricingResult.SelectedPaymentMethod.Total,
		}
	} else {
		result.PaymentMethods = make(
			[]CheckoutPaymentMethodResult,
			0,
			len(pricingResult.PaymentMethods),
		)
		for _, pm := range pricingResult.PaymentMethods {
			result.PaymentMethods = append(
				result.PaymentMethods,
				CheckoutPaymentMethodResult{
					ID:          pm.PaymentMethodID,
					Name:        pm.Name,
					Type:        pm.Type,
					Description: pm.Description,
					Fee:         pm.Fee,
					Subtotal:    pm.Subtotal,
					Total:       pm.Total,
				},
			)
		}
	}

	return result, nil
}
