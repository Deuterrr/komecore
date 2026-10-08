package orderusecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	apperrors "komecore/internal/common/errors"
	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authrepo"
	"komecore/internal/modules/cart/cartdomain"
	"komecore/internal/modules/cart/cartrepo"
	"komecore/internal/modules/inventory/inventoryrepo"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"
	"komecore/internal/modules/user/userrepo"
	markdown "komecore/internal/shared/markdown"
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
	ItemOptions cartdomain.ItemOptions
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
	ChannelData    *paymentdomain.PaymentChannelData
	Instruction    *string
	Total          int64
}

type CreateOrderUsecase struct {
	executor               transaction.Executor
	transactor             transaction.Transactor
	accountRepo            authrepo.AccountRepository
	orderRepo              orderrepo.OrderRepository
	orderItemRepo          orderrepo.OrderItemRepository
	invoiceRepo            orderrepo.InvoiceRepository
	invoiceItemRepo        orderrepo.InvoiceItemRepository
	paymentRepo            paymentrepo.PaymentRepository
	paymentMethodRepo      paymentrepo.PaymentMethodRepository
	paymentEventRepo       paymentrepo.PaymentEventRepository
	paymentInstructionRepo paymentrepo.PaymentInstructionRepository
	paymentChannelDataRepo paymentrepo.PaymentChannelDataRepository
	inventoryRepo          inventoryrepo.InventoryRepository
	cartRepo               cartrepo.CartRepository
	userRepo               userrepo.UserRepository
	paymentGateway         paymentgateway.Provider
	pricingService         orderrepo.PricingService
}

func NewCreateOrderUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo authrepo.AccountRepository,
	orderRepo orderrepo.OrderRepository,
	orderItemRepo orderrepo.OrderItemRepository,
	invoiceRepo orderrepo.InvoiceRepository,
	invoiceItemRepo orderrepo.InvoiceItemRepository,
	paymentRepo paymentrepo.PaymentRepository,
	paymentMethodRepo paymentrepo.PaymentMethodRepository,
	paymentEventRepo paymentrepo.PaymentEventRepository,
	paymentInstructionRepo paymentrepo.PaymentInstructionRepository,
	paymentChannelDataRepo paymentrepo.PaymentChannelDataRepository,
	inventoryRepo inventoryrepo.InventoryRepository,
	cartRepo cartrepo.CartRepository,
	userRepo userrepo.UserRepository,
	paymentGateway paymentgateway.Provider,
	pricingService orderrepo.PricingService,
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
		method        *paymentdomain.PaymentMethod
		pricingResult *orderrepo.PricingResult
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

	order := orderdomain.Order{
		ID:          uuid.New(),
		Number:      orderdomain.NewOrderNumber(),
		CustomerID:  input.CustomerID,
		AddressID:   input.AddressID,
		Status:      orderdomain.OrderStatusPending,
		Subtotal:    pricingResult.Subtotal,
		ShippingFee: pricingResult.TotalShippingFee,
		Total:       pricingResult.GrandTotal,
		CreatedAt:   now,
	}
	if err := order.Validate(); err != nil {
		return nil, fmt.Errorf("invalid order domain state: %w", err)
	}
	invoice := order.NewInvoice()

	var (
		orderItems   []orderdomain.OrderItem
		invoiceItems []orderdomain.InvoiceItem
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
			orderItem := orderdomain.OrderItem{
				ID:             uuid.New(),
				OrderID:        order.ID,
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

	var itemSum int64
	for _, ci := range chargeItems {
		itemSum += ci.Price * int64(ci.Quantity)
	}

	if diff := order.Total - itemSum; diff != 0 {
		chargeItems = append(chargeItems, paymentgateway.ChargeItem{
			ID:       "adjustment",
			Name:     "Fees & Adjustments",
			Quantity: 1,
			Price:    diff,
		})
	}

	payment := paymentdomain.Payment{
		ID:        uuid.New(),
		OrderID:   order.ID,
		MethodID:  input.PaymentMethodID,
		Amount:    order.Total,
		Status:    paymentdomain.PaymentStatusPending,
		CreatedAt: now,
	}

	chargeResp, err := u.paymentGateway.Charge(
		ctx,
		paymentgateway.ChargeRequest{
			PaymentID:     payment.ID,
			OrderID:       order.ID,
			Amount:        order.Total,
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
	payment.Provider = method.Provider
	payment.ProviderPaymentID = &chargeResp.GatewayTransactionID
	payment.ProviderOrderID = &chargeResp.GatewayOrderID
	if !chargeResp.ExpiresAt.IsZero() {
		payment.ExpiresAt = &chargeResp.ExpiresAt
	}

	var (
		instructionContent   *string
		instruction          *paymentdomain.PaymentInstruction
		channelData          *paymentdomain.PaymentChannelData
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

	paymentEvent := paymentdomain.PaymentEvent{
		ID:        uuid.New(),
		PaymentID: payment.ID,
		EventName: string(paymentdomain.PaymentEventStatusPending),
		Payload:   pendingPayload,
		CreatedAt: now,
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.orderRepo.Save(ctx, exec, order); err != nil {
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
		if err := u.paymentRepo.Save(ctx, exec, payment); err != nil {
			return fmt.Errorf("failed to save payment: %w", err)
		}
		if err := u.paymentEventRepo.Create(ctx, exec, paymentEvent); err != nil {
			return fmt.Errorf("failed to save payment event: %w", err)
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

			cd := paymentdomain.PaymentChannelData{
				ID:            uuid.New(),
				PaymentID:     payment.ID,
				ChannelType:   method.Type,
				DisplayName:   inst.Label,
				AccountNumber: accountNumber,
				QRString:      qrString,
				RedirectURL:   redirectURL,
				ActionURL:     actionURL,
				Metadata:      make(map[string]any),
				ExpiresAt:     payment.ExpiresAt,
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

		return nil
	})
	if err != nil {
		if payment.ProviderOrderID != nil {
			// Best-effort rollback:
			// the payment transaction has already been created
			// at the gateway, but persisting the order/payment
			// in the system database failed.
			//
			// Attempt to cancel the gateway transaction to avoid
			// leaving an orphaned payable transaction.
			_ = u.paymentGateway.CancelTransaction(ctx, *payment.ProviderOrderID)
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
		if payment.ExpiresAt != nil {
			effectiveExpiresAt = *payment.ExpiresAt
		} else {
			effectiveExpiresAt = expiresAt
		}

		content, err := markdown.Render(
			instruction.Content,
			map[string]string{
				"invoice_number": invoice.Number,
				"amount":         strconv.FormatInt(order.Total, 10),
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
		OrderID:        order.ID,
		PaymentAccount: paymentAccountResult,
		ChannelData:    channelData,
		Instruction:    instructionContent,
		Total:          order.Total,
	}

	return &result, nil
}

func (u *CreateOrderUsecase) validateAndCalculatePricing(
	ctx context.Context,
	input CreateOrderInput,
) (*paymentdomain.PaymentMethod, *orderrepo.PricingResult, error) {
	// Ensure the selected payment method exists and can be used
	// before creating any order-related records
	method, err := u.paymentMethodRepo.GetByID(ctx, u.executor, input.PaymentMethodID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve payment method: %w", err)
	}
	if method == nil {
		return nil, nil, apperrors.NewNotFound("payment method not found")
	}
	if !u.paymentGateway.Supports(method.Code) {
		return nil, nil, apperrors.NewBadRequest(fmt.Sprintf("payment method %q is not supported by the payment gateway", method.Code))
	}

	pricingInput := orderrepo.PricingInput{
		CustomerID:      input.CustomerID,
		AddressID:       &input.AddressID,
		PaymentMethodID: &input.PaymentMethodID,
		Shops:           make([]orderrepo.PricingShopInput, 0, len(input.Shops)),
	}

	for _, shop := range input.Shops {
		var courierCode, courierService *string
		if shop.Courier != nil {
			courierCode = &shop.Courier.Code
			courierService = &shop.Courier.Service
		}

		shopInput := orderrepo.PricingShopInput{
			ShopID:         shop.ShopID,
			CourierCode:    courierCode,
			CourierService: courierService,
			Items:          make([]orderrepo.PricingItemInput, 0, len(shop.Items)),
		}

		for _, item := range shop.Items {
			if item.ProductID == uuid.Nil {
				return nil, nil, apperrors.NewInvalidInput("product_id is required")
			}
			shopInput.Items = append(
				shopInput.Items,
				orderrepo.PricingItemInput{
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
		return "", "", "", apperrors.NewNotFound("user not found")
	}

	account, err := u.accountRepo.GetByUserID(ctx, u.executor, user.ID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to retrieve account: %w", err)
	}
	if account == nil {
		return "", "", "", apperrors.NewNotFound("account not found")
	}

	var customerPhone string
	if user.Phone != nil {
		customerPhone = *user.Phone
	}

	return user.Name, account.Email, customerPhone, nil
}
