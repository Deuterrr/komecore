package orderusecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/infra/outbox"
	"komecore/internal/infra/paymentgateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/cart"
	"komecore/internal/modules/discount"
	"komecore/internal/modules/discount/discountusecase"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/order"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/user"
	"komecore/internal/strtpl"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	PAYMENT_PROVIDER   = "midtrans"
	PAYMENT_EXPIRATION = time.Hour * 24
)

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

type CreateOrderResult struct {
	OrderID        uuid.UUID
	PaymentAccount *PaymentAccountResult
	ChannelData    *payment.PaymentChannelData
	Instruction    *string
	Total          int64
}

type CouponRedeemer interface {
	RedeemCoupon(ctx context.Context, exec transaction.Executor, input discountusecase.RedeemCouponInput) (*discount.CouponRedemption, error)
}

type OutboxRecorder interface {
	Enqueue(ctx context.Context, exec transaction.Executor, eventType string, payload any) error
}

type CreateOrderUsecase struct {
	executor               transaction.Executor
	transactor             transaction.Transactor
	accountRepo            auth.AccountRepository
	orderRepo              order.Repository
	orderItemRepo          order.ItemRepository
	invoiceRepo            order.InvoiceRepository
	invoiceItemRepo        order.InvoiceItemRepository
	paymentRepo            payment.Repository
	paymentMethodRepo      payment.MethodRepository
	paymentEventRepo       payment.EventRepository
	paymentInstructionRepo payment.InstructionRepository
	paymentChannelDataRepo payment.ChannelDataRepository
	inventoryRepo          inventory.Repository
	cartRepo               cart.Repository
	userRepo               user.Repository
	paymentGateway         paymentgateway.Provider
	pricingService         order.PricingService
	couponService          CouponRedeemer
	outboxRecorder         OutboxRecorder
}

func NewCreateOrderUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo auth.AccountRepository,
	orderRepo order.Repository,
	orderItemRepo order.ItemRepository,
	invoiceRepo order.InvoiceRepository,
	invoiceItemRepo order.InvoiceItemRepository,
	paymentRepo payment.Repository,
	paymentMethodRepo payment.MethodRepository,
	paymentEventRepo payment.EventRepository,
	paymentInstructionRepo payment.InstructionRepository,
	paymentChannelDataRepo payment.ChannelDataRepository,
	inventoryRepo inventory.Repository,
	cartRepo cart.Repository,
	userRepo user.Repository,
	paymentGateway paymentgateway.Provider,
	pricingService order.PricingService,
) *CreateOrderUsecase {
	return &CreateOrderUsecase{
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

func (u *CreateOrderUsecase) WithCouponService(cs CouponRedeemer) *CreateOrderUsecase {
	u.couponService = cs
	return u
}

func (u *CreateOrderUsecase) WithOutboxRecorder(recorder OutboxRecorder) *CreateOrderUsecase {
	u.outboxRecorder = recorder
	return u
}

// Execute orchestrates the checkout workflow across cart validation, inventory,
// pricing, payment processing, and order persistence.
//
// The checkout must maintain consistency between the local order state and the
// external payment gateway. If persistence fails after a successful gateway
// charge, the charge MUST be compensated through CancelTransaction to prevent
// an orphaned payment.
//
// All checkout records are persisted atomically within a database transaction,
// including order, invoice, payment, item, and payment channel data.
func (u *CreateOrderUsecase) Execute(ctx context.Context, input CreateOrderInput) (*CreateOrderResult, error) {
	now := appclock.Now()

	var (
		method        *payment.PaymentMethod
		pricingResult *order.PricingResult
		customerName  string
		customerEmail string
		customerPhone string
	)

	// Concurrently fetch & validate payment method/pricing
	// and customer details
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

	ord := order.Order{
		ID:          uuid.New(),
		Number:      order.NewOrderNumber(),
		CustomerID:  input.CustomerID,
		AddressID:   input.AddressID,
		Status:      order.OrderStatusPending,
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
		orderItems   []order.OrderItem
		invoiceItems []order.InvoiceItem
		chargeItems  []paymentgateway.ChargeItem

		expiresAt = now.Add(PAYMENT_EXPIRATION)
	)

	// Generate order and invoice items from the pricing result to
	// preserve product, pricing, and shipping details at checkout time
	for _, shopRes := range pricingResult.Shops {
		var courierCode, courierService *string
		if shopRes.SelectedCourier.Code != "" {
			courierCode = &shopRes.SelectedCourier.Code
			courierService = &shopRes.SelectedCourier.Service
		}

		for _, itemRes := range shopRes.Items {
			orderItem := order.OrderItem{
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
			if _, err := u.couponService.RedeemCoupon(ctx, exec, discountusecase.RedeemCouponInput{
				Code:        *input.CouponCode,
				CustomerID:  input.CustomerID,
				OrderID:     ord.ID,
				Subtotal:    pricingResult.Subtotal,
				ShippingFee: pricingResult.TotalShippingFee,
			}); err != nil {
				return fmt.Errorf("failed to redeem coupon: %w", err)
			}
		}

		// Persist gateway channel data (QR string, VA number, deep link)
		// so it survives the initial checkout response and can be
		// retrieved on any subsequent request.
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

		for _, item := range orderItems {
			if err := u.inventoryRepo.Reserve(ctx, exec,
				item.ProductID,
				item.ShopID,
				item.Quantity,
			); err != nil {
				return fmt.Errorf("failed to reserve inventory for product %s: %w", item.ProductID, err)
			}
		}

		cart, err := u.cartRepo.GetWithItemsByCustomerID(ctx, exec, input.CustomerID)
		if err != nil {
			return fmt.Errorf("failed to load cart with items: %w", err)
		}
		if cart != nil {
			for _, shop := range pricingResult.Shops {
				for _, item := range shop.Items {
					if item.CartItemID != nil {
						if cart.RemoveItemByID(*item.CartItemID) {
							continue
						}
					}
					if !cart.RemoveItem(item.ProductID, shop.ShopID) {
						cart.RemoveProduct(item.ProductID)
					}
				}
			}
			if err := u.cartRepo.Save(ctx, exec, cart); err != nil {
				return fmt.Errorf("failed to update cart: %w", err)
			}
		}

		ins, err := u.paymentInstructionRepo.GetByPaymentMethodID(ctx, u.executor,
			input.PaymentMethodID,
		)
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
			// Best-effort rollback:
			// the payment transaction has already been created
			// at the gateway, but persisting the order/payment
			// in the system database failed.
			//
			// Attempt to cancel the gateway transaction to avoid
			// leaving an orphaned payable transaction.
			_ = u.paymentGateway.CancelTransaction(ctx, *pmt.ProviderOrderID)
		}

		return nil, err
	}

	// Render payment instructions with transaction-specific values
	// such as invoice number, amount, expiration time, and account details
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

func (u *CreateOrderUsecase) validateAndCalculatePricing(
	ctx context.Context,
	input CreateOrderInput,
) (*payment.PaymentMethod, *order.PricingResult, error) {
	// Ensure the selected payment method exists and can be used
	// before creating any order-related records
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

	pricingInput := order.PricingInput{
		CustomerID:      input.CustomerID,
		AddressID:       &input.AddressID,
		PaymentMethodID: &input.PaymentMethodID,
		CouponCode:      input.CouponCode,
		Shops:           make([]order.PricingShopInput, 0, len(input.Shops)),
	}

	for _, shop := range input.Shops {
		var courierCode, courierService *string
		if shop.Courier != nil {
			courierCode = &shop.Courier.Code
			courierService = &shop.Courier.Service
		}

		shopInput := order.PricingShopInput{
			ShopID:         shop.ShopID,
			CourierCode:    courierCode,
			CourierService: courierService,
			Items:          make([]order.PricingItemInput, 0, len(shop.Items)),
		}

		for _, item := range shop.Items {
			if item.ProductID == uuid.Nil {
				return nil, nil, apperror.NewInvalidInput("product_id is required")
			}
			shopInput.Items = append(
				shopInput.Items,
				order.PricingItemInput{
					ProductID:   item.ProductID,
					ItemOptions: item.ItemOptions,
					Quantity:    item.Quantity,
				},
			)
		}

		pricingInput.Shops = append(pricingInput.Shops, shopInput)
	}

	// Calculate the final order pricing, including item subtotals,
	// shipping fees, payment fees, and the grand total
	pricingResult, err := u.pricingService.Calculate(ctx, u.executor, pricingInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate order prices: %w", err)
	}

	return method, pricingResult, nil
}

func (u *CreateOrderUsecase) fetchCustomerDetails(
	ctx context.Context,
	userID uuid.UUID,
) (string, string, string, error) {
	user, err := u.userRepo.GetByID(ctx, u.executor, userID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to retrieve user: %w", err)
	}
	if user == nil {
		return "", "", "", apperror.NewNotFound("user not found")
	}

	account, err := u.accountRepo.GetByUserID(ctx, u.executor, user.ID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to retrieve account: %w", err)
	}
	if account == nil {
		return "", "", "", apperror.NewNotFound("account not found")
	}

	var customerPhone string
	if user.Phone != nil {
		customerPhone = *user.Phone
	}

	return user.Name, account.Email, customerPhone, nil
}
