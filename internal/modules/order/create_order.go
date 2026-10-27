package order

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	"komecore/internal/infra/outbox"
	"komecore/internal/infra/paymentgateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/cart"
	"komecore/internal/modules/discount"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/user"
	"komecore/internal/strtpl"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	PAYMENT_EXPIRATION = time.Hour * 24
)

type CouponRedeemer interface {
	RedeemCoupon(ctx context.Context, exec transaction.Executor, input discount.RedeemCouponInput) (*discount.CouponRedemption, error)
}

type OutboxRecorder interface {
	Enqueue(ctx context.Context, exec transaction.Executor, eventType string, payload any) error
}

// CreateOrderWorkflow coordinates checkout pricing, payment gateway charging,
// and transactional persistence with compensating cancellation on failure.
type CreateOrderWorkflow struct {
	executor               transaction.Executor
	transactor             transaction.Transactor
	accountRepo            auth.AccountRepository
	orderRepo              Repository
	orderItemRepo          ItemRepository
	invoiceRepo            InvoiceRepository
	invoiceItemRepo        InvoiceItemRepository
	paymentRepo            payment.Repository
	paymentMethodRepo      payment.MethodRepository
	paymentEventRepo       payment.EventRepository
	paymentInstructionRepo payment.InstructionRepository
	paymentChannelDataRepo payment.ChannelDataRepository
	inventoryRepo          inventory.Repository
	cartRepo               cart.Repository
	userRepo               user.Repository
	paymentGateway         paymentgateway.Provider
	pricingService         PricingService
	couponService          CouponRedeemer
	outboxRecorder         OutboxRecorder
}

func NewCreateOrderWorkflow(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo auth.AccountRepository,
	orderRepo Repository,
	orderItemRepo ItemRepository,
	invoiceRepo InvoiceRepository,
	invoiceItemRepo InvoiceItemRepository,
	paymentRepo payment.Repository,
	paymentMethodRepo payment.MethodRepository,
	paymentEventRepo payment.EventRepository,
	paymentInstructionRepo payment.InstructionRepository,
	paymentChannelDataRepo payment.ChannelDataRepository,
	inventoryRepo inventory.Repository,
	cartRepo cart.Repository,
	userRepo user.Repository,
	paymentGateway paymentgateway.Provider,
	pricingService PricingService,
) *CreateOrderWorkflow {
	return &CreateOrderWorkflow{
		executor:               executor,
		transactor:             transactor,
		accountRepo:            accountRepo,
		orderRepo:              orderRepo,
		orderItemRepo:          orderItemRepo,
		invoiceRepo:            invoiceRepo,
		invoiceItemRepo:        invoiceItemRepo,
		paymentRepo:            paymentRepo,
		paymentMethodRepo:      paymentMethodRepo,
		paymentEventRepo:       paymentEventRepo,
		paymentInstructionRepo: paymentInstructionRepo,
		paymentChannelDataRepo: paymentChannelDataRepo,
		inventoryRepo:          inventoryRepo,
		cartRepo:               cartRepo,
		userRepo:               userRepo,
		paymentGateway:         paymentGateway,
		pricingService:         pricingService,
	}
}

func (u *CreateOrderWorkflow) WithCouponService(cs CouponRedeemer) *CreateOrderWorkflow {
	u.couponService = cs
	return u
}

func (u *CreateOrderWorkflow) WithOutboxRecorder(recorder OutboxRecorder) *CreateOrderWorkflow {
	u.outboxRecorder = recorder
	return u
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

// CheckoutInput specifies customer checkout parameters for pre-purchase preview.
type CheckoutInput struct {
	CustomerID      uuid.UUID
	PaymentMethodID *uuid.UUID
	AddressID       *uuid.UUID
	CouponCode      *string
	ShopInput       []CheckoutShopInput
}

// Checkout evaluates pricing totals and courier options for checkout preview
// without creating database records or holding inventory locks.
func (u *CreateOrderWorkflow) Checkout(ctx context.Context, input CheckoutInput) (*PricingResult, error) {
	customerID := input.CustomerID
	if customerID == uuid.Nil {
		if actor, ok := authctx.GetActor(ctx); ok && actor.CustomerID != nil {
			customerID = *actor.CustomerID
		}
	}

	pricingInput := PricingInput{
		CustomerID:      customerID,
		AddressID:       input.AddressID,
		PaymentMethodID: input.PaymentMethodID,
		CouponCode:      input.CouponCode,
		Shops: make(
			[]PricingShopInput,
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

		shopInput := PricingShopInput{
			ShopID:         shop.ShopID,
			CourierCode:    courierCode,
			CourierService: courierService,
			Items: make(
				[]PricingItemInput,
				0,
				len(shop.Items),
			),
		}
		for _, item := range shop.Items {
			shopInput.Items = append(
				shopInput.Items,
				PricingItemInput{
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

type OrderItemInput struct {
	ProductID   uuid.UUID
	ItemOptions cart.ItemOptions
	ProductName string
	Quantity    int
}

type OrderCourierInput struct {
	Code    string
	Service string
}

type OrderShopInput struct {
	ShopID   uuid.UUID
	ShopName string
	Courier  *OrderCourierInput
	Items    []OrderItemInput
}

// CreateOrderInput captures verified parameters required to charge payment and persist orders.
type CreateOrderInput struct {
	UserID          uuid.UUID
	CustomerID      uuid.UUID
	AddressID       uuid.UUID
	PaymentMethodID uuid.UUID
	CouponCode      *string
	Shops           []OrderShopInput
}

type PaymentAccountResult struct {
	AccountName   string
	AccountNumber *string
	PhoneNumber   *string
	QRString      *string
}

// CreateOrderResult contains order identifiers and payment channel guidance
// (virtual account numbers, QR codes, or redirect URLs) for customer presentation.
type CreateOrderResult struct {
	OrderID        uuid.UUID
	PaymentAccount *PaymentAccountResult
	ChannelData    *payment.PaymentChannelData
	Instruction    *string
	Total          int64
}

func (u *CreateOrderWorkflow) Execute(ctx context.Context, input CreateOrderInput) (*CreateOrderResult, error) {
	return u.CreateOrder(ctx, input)
}

// CreateOrder orchestrates order placement: it charges the payment gateway before
// persisting to DB, and cancels the charge if local transaction rollback occurs.
func (u *CreateOrderWorkflow) CreateOrder(ctx context.Context, input CreateOrderInput) (*CreateOrderResult, error) {
	now := appclock.Now()

	var (
		method        *payment.PaymentMethod
		pricingResult *PricingResult
		customerName  string
		customerEmail string
		customerPhone string
	)

	// Concurrently validate pricing/method availability and retrieve customer identity
	// using errgroup to minimize checkout latency prior to initiating payment charge.
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		method, pricingResult, err = u.validateAndCalculatePricing(gCtx, input)
		return err
	})

	g.Go(func() error {
		var err error
		customerName, customerEmail, customerPhone, err = u.fetchCustomerDetails(gCtx, input.UserID)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	ord := Order{
		ID:          uuid.New(),
		Number:      NewOrderNumber(),
		CustomerID:  input.CustomerID,
		AddressID:   input.AddressID,
		Status:      OrderStatusPending,
		Subtotal:    pricingResult.Subtotal,
		ShippingFee: pricingResult.TotalShippingFee,
		Total:       pricingResult.GrandTotal,
		CreatedAt:   now,
	}
	if err := ord.Validate(); err != nil {
		return nil, fmt.Errorf("invalid order domain state: %w", err)
	}
	invoice := ord.NewInvoice()

	var (
		orderItems   []OrderItem
		invoiceItems []InvoiceItem
		chargeItems  []paymentgateway.ChargeItem

		expiresAt = now.Add(PAYMENT_EXPIRATION)
	)

	// Snapshot product, pricing, and courier details into immutable order and invoice line items.
	for _, shopRes := range pricingResult.Shops {
		var courierCode, courierService *string

		if shopRes.SelectedCourier.Code != "" {
			courierCode = &shopRes.SelectedCourier.Code
			courierService = &shopRes.SelectedCourier.Service
		}

		for _, itemRes := range shopRes.Items {
			orderItem := OrderItem{
				ID:             uuid.New(),
				OrderID:        ord.ID,
				ShopID:         shopRes.ShopID,
				ShopName:       shopRes.ShopName,
				ProductID:      itemRes.ProductID,
				ProductName:    itemRes.ProductName,
				Quantity:       itemRes.Quantity,
				UnitPrice:      itemRes.UnitPrice,
				Subtotal:       itemRes.Subtotal,
				ItemOptions:    itemRes.ItemOptions,
				CourierCode:    courierCode,
				CourierService: courierService,
				ShippingFee:    shopRes.SelectedCourier.Fee,
			}
			invoiceItem := invoice.NewInvoiceItemFromOrderItem(orderItem)

			orderItems = append(orderItems, orderItem)
			invoiceItems = append(invoiceItems, invoiceItem)
		}
	}

	// Format gateway charge items: product line items plus positive shipping fees and negative voucher discounts.
	for _, item := range orderItems {
		cItem := paymentgateway.ChargeItem{
			ID:       item.ProductID.String(),
			Name:     item.ProductName,
			Quantity: item.Quantity,
			Price:    item.UnitPrice,
		}
		chargeItems = append(chargeItems, cItem)
	}
	if pricingResult.TotalShippingFee > 0 {
		chargeItems = append(chargeItems, paymentgateway.ChargeItem{
			ID:       "shipping_fee",
			Name:     "Shipping Fee",
			Quantity: 1,
			Price:    pricingResult.TotalShippingFee,
		})
	}
	if pricingResult.DiscountAmount > 0 {
		chargeItems = append(chargeItems, paymentgateway.ChargeItem{
			ID:       "discount",
			Name:     "Discount Voucher",
			Quantity: 1,
			Price:    -pricingResult.DiscountAmount,
		})
	}

	// Align line-item sums with grand total using an adjustment line to prevent
	// gateway rejection caused by rounding discrepancies or fractional fees.
	var itemSum int64
	for _, ci := range chargeItems {
		itemSum += ci.Price * int64(ci.Quantity)
	}

	if diff := ord.Total - itemSum; diff != 0 {
		chargeItems = append(chargeItems, paymentgateway.ChargeItem{
			ID:       "adjustment",
			Name:     "Fees & Adjustments",
			Quantity: 1,
			Price:    diff,
		})
	}

	pmt := payment.Payment{
		ID:        uuid.New(),
		OrderID:   ord.ID,
		MethodID:  input.PaymentMethodID,
		Amount:    ord.Total,
		Status:    payment.PaymentStatusPending,
		CreatedAt: now,
	}

	chargeResp, err := u.paymentGateway.Charge(
		ctx,
		paymentgateway.ChargeRequest{
			PaymentID:     pmt.ID,
			OrderID:       ord.ID,
			Amount:        ord.Total,
			PaymentType:   method.Code,
			ExpiresAt:     expiresAt,
			CustomerEmail: customerEmail,
			CustomerName:  customerName,
			CustomerPhone: customerPhone,
			Items:         chargeItems,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("payment gateway charge failed: %w", err)
	}

	// Process payments through the configured gateway for
	// methods that require external payment handling.
	pmt.Provider = method.Provider
	pmt.ProviderPaymentID = &chargeResp.GatewayTransactionID
	pmt.ProviderOrderID = &chargeResp.GatewayOrderID
	if !chargeResp.ExpiresAt.IsZero() {
		pmt.ExpiresAt = &chargeResp.ExpiresAt
	}

	var (
		instructionContent   *string
		instruction          *payment.PaymentInstruction
		channelData          *payment.PaymentChannelData
		paymentAccountResult *PaymentAccountResult
	)

	if len(chargeResp.Instructions) > 0 {
		inst := chargeResp.Instructions[0]
		accountResult := &PaymentAccountResult{
			AccountName: inst.Label,
		}

		switch inst.Type {
		case "qris", "ewallet":
			accountResult.QRString = &inst.Value

		case "bank_transfer":
			accountResult.AccountNumber = &inst.Value
		}

		paymentAccountResult = accountResult
	}

	pendingPayload, err := json.Marshal(map[string]string{"status": "pending"})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pending payment event payload: %w", err)
	}

	paymentEvent := payment.PaymentEvent{
		ID:        uuid.New(),
		PaymentID: pmt.ID,
		EventName: string(payment.PaymentEventStatusPending),
		Payload:   pendingPayload,
		CreatedAt: now,
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.orderRepo.Save(ctx, exec, ord); err != nil {
			return fmt.Errorf("failed to save order: %w", err)
		}
		if err := u.invoiceRepo.Save(ctx, exec, invoice); err != nil {
			return fmt.Errorf("failed to save invoice: %w", err)
		}
		if err := u.orderItemRepo.SaveBulk(ctx, exec, orderItems); err != nil {
			return fmt.Errorf("failed to save order items: %w", err)
		}
		if err := u.invoiceItemRepo.SaveBulk(ctx, exec, invoiceItems); err != nil {
			return fmt.Errorf("failed to save invoice items: %w", err)
		}
		if err := u.paymentRepo.Save(ctx, exec, pmt); err != nil {
			return fmt.Errorf("failed to save payment: %w", err)
		}
		if err := u.paymentEventRepo.Create(ctx, exec, paymentEvent); err != nil {
			return fmt.Errorf("failed to save payment event: %w", err)
		}

		if input.CouponCode != nil && strings.TrimSpace(*input.CouponCode) != "" && u.couponService != nil {
			if _, err := u.couponService.RedeemCoupon(ctx, exec, discount.RedeemCouponInput{
				Code:        *input.CouponCode,
				CustomerID:  input.CustomerID,
				OrderID:     ord.ID,
				Subtotal:    pricingResult.Subtotal,
				ShippingFee: pricingResult.TotalShippingFee,
			}); err != nil {
				return fmt.Errorf("failed to redeem coupon: %w", err)
			}
		}

		// Normalize gateway-specific payment instructions (direct VA fields vs instruction slices,
		// and QR codes vs deep links) into persistent channel data retrievable across customer sessions.
		if chargeResp != nil && len(chargeResp.Instructions) > 0 {
			var (
				actionURL     *string
				accountNumber *string
				qrString      *string
				redirectURL   *string

				inst = chargeResp.Instructions[0]
			)

			if chargeResp.AccountNumber != nil ||
				chargeResp.QRString != nil ||
				chargeResp.RedirectURL != nil {

				accountNumber = chargeResp.AccountNumber
				qrString = chargeResp.QRString
				redirectURL = chargeResp.RedirectURL
				if accountNumber != nil {
					actionURL = accountNumber
				} else if qrString != nil {
					actionURL = qrString
				} else if redirectURL != nil {
					actionURL = redirectURL
				}
			} else if inst.Value != "" {
				v := inst.Value
				actionURL = &v
				switch inst.Type {
				case "bank_transfer":
					accountNumber = &v
				case "qris":
					qrString = &v
				case "ewallet":
					if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
						redirectURL = &v
					} else {
						qrString = &v
					}
				default:
					actionURL = &v
				}
			}

			cd := payment.PaymentChannelData{
				ID:            uuid.New(),
				PaymentID:     pmt.ID,
				ChannelType:   method.Type,
				DisplayName:   inst.Label,
				AccountNumber: accountNumber,
				QRString:      qrString,
				RedirectURL:   redirectURL,
				ActionURL:     actionURL,
				Metadata:      make(map[string]any),
				ExpiresAt:     pmt.ExpiresAt,
				CreatedAt:     now,
			}
			if err := u.paymentChannelDataRepo.Save(ctx, exec, cd); err != nil {
				return fmt.Errorf("failed to save payment channel data: %w", err)
			}
			channelData = &cd
		}

		// Deterministically sort order items by (ShopID, ProductID) prior to inventory reservation
		// to enforce a strict monotonic lock hierarchy and prevent relational database deadlocks.
		slices.SortStableFunc(orderItems, func(a, b OrderItem) int {
			if c := cmp.Compare(a.ShopID.String(), b.ShopID.String()); c != 0 {
				return c
			}
			return cmp.Compare(a.ProductID.String(), b.ProductID.String())
		})

		for _, item := range orderItems {
			if err := u.inventoryRepo.Reserve(ctx, exec,
				item.ProductID,
				item.ShopID,
				item.Quantity,
			); err != nil {
				return fmt.Errorf("failed to reserve inventory for product %s: %w", item.ProductID, err)
			}
		}

		// Remove purchased items from persistent cart: match by CartItemID first,
		// then fall back to (ProductID, ShopID, ItemOptions), and finally ProductID.
		crt, err := u.cartRepo.GetWithItemsByCustomerID(ctx, exec, input.CustomerID)
		if err != nil {
			return fmt.Errorf("failed to load cart with items: %w", err)
		}
		if crt != nil {
			for _, shop := range pricingResult.Shops {
				for _, item := range shop.Items {
					if item.CartItemID != nil {
						if crt.RemoveItemByID(*item.CartItemID) {
							continue
						}
					}
					if item.CartItemID != nil {
						if crt.RemoveItemByID(*item.CartItemID) {
							continue
						}
					}
					var removed bool
					if len(item.ItemOptions) > 0 {
						removed = crt.RemoveItem(item.ProductID, shop.ShopID, item.ItemOptions)
					}
					if !removed && !crt.RemoveItem(item.ProductID, shop.ShopID) {
						crt.RemoveProduct(item.ProductID)
					}
				}
			}
			if err := u.cartRepo.Save(ctx, exec, crt); err != nil {
				return fmt.Errorf("failed to update cart: %w", err)
			}
		}

		ins, err := u.paymentInstructionRepo.GetByPaymentMethodID(ctx, exec, input.PaymentMethodID)
		if err != nil {
			return fmt.Errorf("failed to retrieve payment instruction: %w", err)
		}

		instruction = ins

		if u.outboxRecorder != nil {
			outboxItems := make([]outbox.OrderCreatedItem, 0, len(orderItems))
			for _, item := range orderItems {
				outboxItems = append(outboxItems, outbox.OrderCreatedItem{
					ProductName: item.ProductName,
					Quantity:    item.Quantity,
					UnitPrice:   item.UnitPrice,
					Subtotal:    item.Subtotal,
				})
			}
			outboxPayload := outbox.OrderCreatedPayload{
				OrderID:       ord.ID,
				OrderNumber:   ord.Number,
				CustomerEmail: customerEmail,
				CustomerName:  customerName,
				Total:         ord.Total,
				Items:         outboxItems,
			}
			if err := u.outboxRecorder.Enqueue(ctx, exec, outbox.EventOrderCreated, outboxPayload); err != nil {
				return fmt.Errorf("failed to enqueue outbox order.created event: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		if pmt.ProviderOrderID != nil {
			// Compensate external gateway charge on transaction failure to prevent orphaned
			// payments when the local database rollback occurs.
			_ = u.paymentGateway.CancelTransaction(ctx, *pmt.ProviderOrderID)
		}

		return nil, err
	}

	// Render dynamic payment instructions (substituting invoice number, VA/QR, amounts, and expiration).
	if instruction != nil {
		var vaNumber string
		if channelData != nil && channelData.AccountNumber != nil {
			vaNumber = *channelData.AccountNumber
		} else if chargeResp != nil && len(chargeResp.Instructions) > 0 {
			inst := chargeResp.Instructions[0]
			if inst.Type == "bank_transfer" {
				vaNumber = inst.Value
			}
		}

		var effectiveExpiresAt time.Time
		if pmt.ExpiresAt != nil {
			effectiveExpiresAt = *pmt.ExpiresAt
		} else {
			effectiveExpiresAt = expiresAt
		}

		content, err := strtpl.Render(
			instruction.Content,
			map[string]string{
				"invoice_number": invoice.Number,
				"amount":         strconv.FormatInt(ord.Total, 10),
				"expired_at":     effectiveExpiresAt.Format(time.RFC3339),
				"va_number":      vaNumber,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to format payment instruction: %w", err)
		}

		instructionContent = &content
	}

	result := CreateOrderResult{
		OrderID:        ord.ID,
		PaymentAccount: paymentAccountResult,
		ChannelData:    channelData,
		Instruction:    instructionContent,
		Total:          ord.Total,
	}

	return &result, nil
}

func (u *CreateOrderWorkflow) validateAndCalculatePricing(ctx context.Context, input CreateOrderInput) (*payment.PaymentMethod, *PricingResult, error) {
	method, err := u.paymentMethodRepo.GetByID(ctx, u.executor, input.PaymentMethodID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve payment method: %w", err)
	}
	if method == nil {
		return nil, nil, apperror.NewNotFound("payment method not found")
	}
	if !u.paymentGateway.Supports(method.Code) {
		return nil, nil, apperror.NewBadRequest(fmt.Sprintf("payment method %q is not supported by the payment gateway", method.Code))
	}

	pricingInput := PricingInput{
		CustomerID:      input.CustomerID,
		AddressID:       &input.AddressID,
		PaymentMethodID: &input.PaymentMethodID,
		CouponCode:      input.CouponCode,
		Shops:           make([]PricingShopInput, 0, len(input.Shops)),
	}

	for _, shop := range input.Shops {
		var courierCode, courierService *string
		if shop.Courier != nil {
			courierCode = &shop.Courier.Code
			courierService = &shop.Courier.Service
		}

		shopInput := PricingShopInput{
			ShopID:         shop.ShopID,
			CourierCode:    courierCode,
			CourierService: courierService,
			Items:          make([]PricingItemInput, 0, len(shop.Items)),
		}
		for _, item := range shop.Items {
			if item.ProductID == uuid.Nil {
				return nil, nil, apperror.NewInvalidInput("product_id is required")
			}
			shopInput.Items = append(
				shopInput.Items,
				PricingItemInput{
					ProductID:   item.ProductID,
					ItemOptions: item.ItemOptions,
					Quantity:    item.Quantity,
				},
			)
		}

		pricingInput.Shops = append(pricingInput.Shops, shopInput)
	}

	pricingResult, err := u.pricingService.Calculate(ctx, u.executor, pricingInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate order prices: %w", err)
	}

	return method, pricingResult, nil
}

func (u *CreateOrderWorkflow) fetchCustomerDetails(ctx context.Context, userID uuid.UUID) (string, string, string, error) {
	usr, err := u.userRepo.GetByID(ctx, u.executor, userID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to retrieve user: %w", err)
	}
	if usr == nil {
		return "", "", "", apperror.NewNotFound("user not found")
	}

	account, err := u.accountRepo.GetByUserID(ctx, u.executor, usr.ID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to retrieve account: %w", err)
	}
	if account == nil {
		return "", "", "", apperror.NewNotFound("account not found")
	}

	var customerPhone string
	if usr.Phone != nil {
		customerPhone = *usr.Phone
	}

	return usr.Name, account.Email, customerPhone, nil
}
