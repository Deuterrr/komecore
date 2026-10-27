package order

import (
	"context"
	"fmt"
	"strings"

	"komecore/internal/apperror"
	shipping "komecore/internal/infra/shipping"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address"
	"komecore/internal/modules/cart"
	"komecore/internal/modules/courier"
	"komecore/internal/modules/discount"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/product"
	"komecore/internal/modules/shop"

	"github.com/google/uuid"
)

// CouponValidator verifies promotional coupon applicability and calculates discounts
// against basket subtotals and shipping costs prior to order creation.
type CouponValidator interface {
	ValidateCoupon(ctx context.Context, input discount.ValidateCouponInput) (*discount.ValidateCouponResult, error)
}

type pricingServiceImpl struct {
	addressRepo       address.CustomerAddressRepository
	cartRepo          cart.Repository
	courierRepo       courier.Repository
	inventoryRepo     inventory.Repository
	paymentMethodRepo payment.MethodRepository
	productRepo       product.Repository
	shippingProvider  shipping.ShippingProvider
	shopAddressRepo   address.ShopAddressRepository
	shopRepo          shop.Repository
	couponService     CouponValidator
}

func NewPricingService(
	addressRepo address.CustomerAddressRepository,
	cartRepo cart.Repository,
	courierRepo courier.Repository,
	inventoryRepo inventory.Repository,
	paymentMethodRepo payment.MethodRepository,
	productRepo product.Repository,
	shippingProvider shipping.ShippingProvider,
	shopAddressRepo address.ShopAddressRepository,
	shopRepo shop.Repository,
) *pricingServiceImpl {
	return &pricingServiceImpl{
		addressRepo:       addressRepo,
		cartRepo:          cartRepo,
		courierRepo:       courierRepo,
		inventoryRepo:     inventoryRepo,
		paymentMethodRepo: paymentMethodRepo,
		productRepo:       productRepo,
		shippingProvider:  shippingProvider,
		shopAddressRepo:   shopAddressRepo,
		shopRepo:          shopRepo,
	}
}

func (s *pricingServiceImpl) WithCouponService(cs CouponValidator) *pricingServiceImpl {
	s.couponService = cs
	return s
}

const defaultShippingWeightGrams = 1000

// Calculate evaluates multi-shop basket totals, resolving destination addresses,
// live courier rates, stock availability, coupons, and payment method fees.
func (s *pricingServiceImpl) Calculate(ctx context.Context, exec transaction.Executor, input PricingInput) (*PricingResult, error) {
	var destAddress *address.CustomerAddress
	if input.AddressID != nil {
		addr, err := s.addressRepo.GetByID(ctx, exec, *input.AddressID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve destination address: %w", err)
		}
		if addr == nil {
			return nil, apperror.NewNotFound(address.ErrAddressNotFound.Error())
		}
		destAddress = addr
	} else {
		addr, err := s.addressRepo.GetDefaultByCustomerID(ctx, exec, input.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve default destination address: %w", err)
		}
		if addr == nil {
			return nil, apperror.NewConflict(address.ErrNotFoundDefaultAddress.Error())
		}
		destAddress = addr
	}

	// Hydrate product IDs and options from stored cart items when only cart_item_id is provided.
	var cartItemMap map[uuid.UUID]cart.CartItem
	if input.CustomerID != uuid.Nil && s.cartRepo != nil {
		if userCart, err := s.cartRepo.GetWithItemsByCustomerID(ctx, exec, input.CustomerID); err == nil && userCart != nil {
			cartItemMap = make(map[uuid.UUID]cart.CartItem, len(userCart.Items))
			for _, ci := range userCart.Items {
				cartItemMap[ci.ID] = ci
			}
		}
	}
	if cartItemMap != nil {
		for i := range input.Shops {
			for j := range input.Shops[i].Items {
				item := &input.Shops[i].Items[j]
				if item.CartItemID != nil {
					if ci, ok := cartItemMap[*item.CartItemID]; ok {
						item.ProductID = ci.ProductID
						item.ItemOptions = ci.ItemOptions
					}
				}
			}
		}
	}

	// Batch load products, inventory balances, and merchant shop records upfront
	// to prevent N+1 database queries across multi-shop baskets.
	var productIDs []uuid.UUID
	var shopIDs []uuid.UUID
	var products []product.Product
	for _, shopGroup := range input.Shops {
		for _, item := range shopGroup.Items {
			productIDs = append(productIDs, item.ProductID)
		}
		shopIDs = append(shopIDs, shopGroup.ShopID)
	}
	if len(productIDs) > 0 {
		var err error
		products, err = s.productRepo.FindByIDs(ctx, exec, productIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to load products: %w", err)
		}
	}
	productMap := make(map[uuid.UUID]product.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}

	var inventoryMap map[uuid.UUID][]inventory.Inventory
	if len(productIDs) > 0 {
		var err error
		inventoryMap, err = s.inventoryRepo.ListByProductIDs(ctx, exec, productIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to load inventory for products: %w", err)
		}
	}

	shops, err := s.shopRepo.FindByIDs(ctx, exec, shopIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load shops: %w", err)
	}
	shopMap := make(map[uuid.UUID]shop.Shop, len(shops))
	for _, sh := range shops {
		shopMap[sh.ID] = sh
	}

	// Verify that each merchant shop is active and approved for transactions.
	for _, shopID := range shopIDs {
		sh, ok := shopMap[shopID]
		if !ok || !sh.IsOperable() {
			shopName := "Unknown Shop"
			if ok {
				shopName = sh.Name
			}
			return nil, apperror.NewConflict(fmt.Sprintf("shop '%s' is inactive or not approved for transactions", shopName))
		}
	}

	var activeCouriers []string
	if s.courierRepo != nil {
		activeCouriers, _ = s.courierRepo.ListAll(ctx, exec)
	}
	if len(activeCouriers) == 0 {
		activeCouriers = []string{"jne", "jnt", "sicepat", "flat"}
	}

	var (
		totalSubtotal    int64
		totalShippingFee int64
		shopsResult      []PricingShopResult
	)

	for _, shopGroup := range input.Shops {
		var (
			shopSubtotal    int64
			shopItemsWeight int
			pricingItems    []PricingItemResult
		)

		// Track remaining stock across items in the same shop to prevent multiple lines
		// of the same product from exceeding total available inventory.
		remainingStock := make(map[uuid.UUID]int)
		for _, shopItem := range shopGroup.Items {
			pid := shopItem.ProductID
			prod, ok := productMap[pid]
			if !ok || prod.Status == product.ProductStatusArchived {
				return nil, apperror.NewNotFound(product.ErrProductNotFound.Error())
			}
			if prod.Status != product.ProductStatusActive {
				return nil, apperror.NewConflict(fmt.Sprintf("product '%s' is currently not available for purchase", prod.Name))
			}

			if _, tracked := remainingStock[pid]; !tracked {
				shopInventories := inventoryMap[pid]
				var totalAvailableStock int
				for _, inv := range shopInventories {
					if inv.ShopID == shopGroup.ShopID {
						totalAvailableStock += inv.Available()
					}
				}
				remainingStock[pid] = totalAvailableStock
			}

			if shopItem.Quantity > remainingStock[pid] {
				return nil, apperror.NewConflict(fmt.Sprintf("insufficient stock for product '%s'", prod.Name))
			}
			remainingStock[pid] -= shopItem.Quantity

			itemSubtotal := prod.Price * int64(shopItem.Quantity)
			var unitWeight int
			if prod.Weight != nil && *prod.Weight > 0 {
				unitWeight = int(*prod.Weight)
			} else {
				unitWeight = defaultShippingWeightGrams
			}

			itemWeight := unitWeight * shopItem.Quantity
			shopSubtotal += itemSubtotal
			shopItemsWeight += itemWeight

			pricingItems = append(pricingItems, PricingItemResult{
				ProductID:   pid,
				CartItemID:  shopItem.CartItemID,
				ItemOptions: shopItem.ItemOptions,
				ProductName: prod.Name,
				Quantity:    shopItem.Quantity,
				UnitPrice:   prod.Price,
				Subtotal:    itemSubtotal,
				WeightGrams: itemWeight,
			})
		}

		totalSubtotal += shopSubtotal

		// Query shipping rates: validate explicitly requested courier code against active couriers,
		// or query all active providers when courier selection is left open.
		var codes []string
		if shopGroup.CourierCode != nil && *shopGroup.CourierCode != "" {
			var supported bool
			for _, c := range activeCouriers {
				if strings.EqualFold(c, *shopGroup.CourierCode) {
					supported = true
					break
				}
			}
			if !supported {
				return nil, apperror.NewConflict(fmt.Sprintf("courier '%s' is not supported", *shopGroup.CourierCode))
			}
			codes = append(codes, *shopGroup.CourierCode)
		} else {
			codes = activeCouriers
		}

		calcInput := shipping.CalculateRatesInput{
			Weight:   shopItemsWeight,
			Couriers: codes,
		}
		costOptions, err := s.shippingProvider.CalculateRates(ctx, calcInput)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate shipping cost: %w", err)
		}

		var (
			leastFee        int64
			selectedFee     int64
			leastFeeCode    string
			leastFeeService string
			hasSelected     bool
			courierOptions  = make([]CourierOption, 0, len(costOptions))
		)

		for i, option := range costOptions {
			courierOptions = append(courierOptions, CourierOption{
				Code:    option.Code,
				Service: option.Service,
				Name:    option.Name,
				ETD:     option.Etd,
				Fee:     option.Cost,
			})

			if i == 0 || option.Cost < leastFee {
				leastFee = option.Cost
				leastFeeCode = option.Code
				leastFeeService = option.Service
			}

			if shopGroup.CourierCode != nil &&
				shopGroup.CourierService != nil &&
				strings.EqualFold(option.Code, *shopGroup.CourierCode) &&
				strings.EqualFold(option.Service, *shopGroup.CourierService) {

				selectedFee = option.Cost
				hasSelected = true
			}
		}

		// Fall back to the lowest-cost available courier option if no specific service was selected.
		shippingFee := leastFee
		resolvedCourierCode := leastFeeCode
		resolvedCourierService := leastFeeService

		if shopGroup.CourierCode != nil &&
			shopGroup.CourierService != nil &&
			*shopGroup.CourierCode != "" {

			if !hasSelected {
				return nil, apperror.NewBadRequest("selected courier service is unavailable")
			}

			shippingFee = selectedFee
			resolvedCourierCode = *shopGroup.CourierCode
			resolvedCourierService = *shopGroup.CourierService
		}

		totalShippingFee += shippingFee

		shopDetails := shopMap[shopGroup.ShopID]
		shopResult := PricingShopResult{
			ShopID:   shopGroup.ShopID,
			ShopName: shopDetails.Name,
			ShopSlug: shopDetails.Slug,
			Items:    pricingItems,
			SelectedCourier: SelectedCourierResult{
				Code:    resolvedCourierCode,
				Service: resolvedCourierService,
				Fee:     shippingFee,
			},
			Subtotal: shopSubtotal,
			Total:    shopSubtotal + shippingFee,
		}

		if shopGroup.CourierCode == nil || *shopGroup.CourierCode == "" {
			shopResult.CourierOptions = courierOptions
		}

		shopsResult = append(shopsResult, shopResult)
	}

	// Validate coupon against subtotal and shipping; floor post-discount total at zero.
	var discountAmount int64
	if input.CouponCode != nil && strings.TrimSpace(*input.CouponCode) != "" && s.couponService != nil {
		res, err := s.couponService.ValidateCoupon(ctx, discount.ValidateCouponInput{
			Code:        *input.CouponCode,
			Subtotal:    totalSubtotal,
			ShippingFee: totalShippingFee,
		})
		if err != nil {
			return nil, err
		}
		discountAmount = res.DiscountAmount
	}

	totalAfterDiscount := totalSubtotal + totalShippingFee - discountAmount
	if totalAfterDiscount < 0 {
		totalAfterDiscount = 0
	}

	var (
		paymentMethods        []PaymentMethodPricingResult
		selectedPaymentMethod *PaymentMethodPricingResult
		totalAll              = totalAfterDiscount
	)

	// Calculate processing fees (flat, percentage, or mixed) for the selected payment method,
	// or evaluate all active methods to populate checkout preview options.
	if input.PaymentMethodID != nil {
		pm, err := s.paymentMethodRepo.GetByID(ctx, exec, *input.PaymentMethodID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve payment method: %w", err)
		}
		if pm == nil {
			return nil, apperror.NewNotFound("selected payment method is not available")
		}

		fee := pm.CalculateFee(totalAll)
		selectedPaymentMethod = &PaymentMethodPricingResult{
			PaymentMethodID: pm.ID,
			Name:            pm.Name,
			Type:            string(pm.Type),
			Description:     pm.Description,
			Fee:             fee,
			Subtotal:        totalAll,
			Total:           totalAll + fee,
		}
	} else {
		pms, err := s.paymentMethodRepo.ListAll(ctx, exec, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to load payment methods: %w", err)
		}
		if len(pms) == 0 {
			return nil, apperror.NewNotFound("payment method is not available")
		}

		paymentMethods = make([]PaymentMethodPricingResult, 0, len(pms))
		for _, pm := range pms {
			fee := pm.CalculateFee(totalAll)
			paymentMethods = append(paymentMethods, PaymentMethodPricingResult{
				PaymentMethodID: pm.ID,
				Name:            pm.Name,
				Type:            string(pm.Type),
				Description:     pm.Description,
				Fee:             fee,
				Subtotal:        totalAll,
				Total:           totalAll + fee,
			})
		}
	}

	var leastFeePayMethod int64
	if selectedPaymentMethod != nil {
		leastFeePayMethod = selectedPaymentMethod.Fee
	} else {
		for i, method := range paymentMethods {
			if i == 0 || method.Fee < leastFeePayMethod {
				leastFeePayMethod = method.Fee
			}
		}
	}

	result := &PricingResult{
		Address: PricingAddressResult{
			ID:            destAddress.ID,
			RecipientName: destAddress.ReceiverName,
			Phone:         destAddress.Phone,
			FullAddress:   destAddress.Detail.FullAddress,
		},
		Shops:                 shopsResult,
		Subtotal:              totalSubtotal,
		TotalShippingFee:      totalShippingFee,
		DiscountAmount:        discountAmount,
		CouponCode:            input.CouponCode,
		GrandTotal:            totalAll + leastFeePayMethod,
		PaymentMethods:        paymentMethods,
		SelectedPaymentMethod: selectedPaymentMethod,
	}

	return result, nil
}
