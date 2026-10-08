package orderusecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/cart/cartdomain"
	"komecore/internal/modules/inventory/inventorydomain"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/user/userdomain"
	"komecore/internal/modules/user/userrepo"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

// ===========================================================================
// Mock infrastructure (all in this package)
// ===========================================================================

// --- transaction mocks ---

type coMockExecutor struct {
	transaction.NoopExecutor
}

type coMockTransactor struct{}

func (m *coMockTransactor) WithinTransaction(ctx context.Context, fn func(transaction.Executor) error) error {
	return fn(&coMockExecutor{})
}

// coFailingTransactor always returns an error from WithinTransaction.
type coFailingTransactor struct{ err error }

func (t *coFailingTransactor) WithinTransaction(_ context.Context, _ func(transaction.Executor) error) error {
	return t.err
}

// --- pricing service ---

type coMockPricingService struct {
	result *orderrepo.PricingResult
	err    error
}

func (m *coMockPricingService) Calculate(_ context.Context, _ transaction.Executor, _ orderrepo.PricingInput) (*orderrepo.PricingResult, error) {
	return m.result, m.err
}

// --- user repo ---

type coMockUserRepo struct {
	user *userdomain.User
	err  error
}

func (m *coMockUserRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*userdomain.User, error) {
	return m.user, m.err
}
func (m *coMockUserRepo) GetByUsername(_ context.Context, _ transaction.Executor, _ string) (*userdomain.User, error) {
	return nil, nil
}
func (m *coMockUserRepo) CreateUser(_ context.Context, _ transaction.Executor, _ userrepo.CreateUserProps) error {
	return nil
}
func (m *coMockUserRepo) SaveProfile(_ context.Context, _ transaction.Executor, _ userrepo.SaveProfileProps) error {
	return nil
}
func (m *coMockUserRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

// --- account repo ---

type coMockAccountRepo struct {
	account *authdomain.Account
	err     error
}

func (m *coMockAccountRepo) GetByEmail(_ context.Context, _ transaction.Executor, _ string) (*authdomain.Account, error) {
	return nil, nil
}
func (m *coMockAccountRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*authdomain.Account, error) {
	return nil, nil
}
func (m *coMockAccountRepo) GetByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*authdomain.Account, error) {
	return m.account, m.err
}
func (m *coMockAccountRepo) ActivateByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *coMockAccountRepo) Create(_ context.Context, _ transaction.Executor, _ authdomain.Account) error {
	return nil
}
func (m *coMockAccountRepo) UpdatePasswordByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ string) error {
	return nil
}
func (m *coMockAccountRepo) DeleteByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *coMockAccountRepo) UpdateLastLoginAt(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time) error {
	return nil
}

// --- payment method repo ---

type coMockPaymentMethodRepo struct {
	method *paymentdomain.PaymentMethod
	err    error
}

func (m *coMockPaymentMethodRepo) Save(_ context.Context, _ transaction.Executor, _ paymentdomain.PaymentMethod) error {
	return nil
}
func (m *coMockPaymentMethodRepo) FindByName(_ context.Context, _ transaction.Executor, _ string) (*paymentdomain.PaymentMethod, error) {
	return nil, nil
}
func (m *coMockPaymentMethodRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*paymentdomain.PaymentMethod, error) {
	return m.method, m.err
}
func (m *coMockPaymentMethodRepo) ListAll(_ context.Context, _ transaction.Executor, _ query.Sorts) ([]paymentdomain.PaymentMethod, error) {
	return nil, nil
}

// --- payment repo ---

type coMockPaymentRepo struct {
	payments map[uuid.UUID]*paymentdomain.Payment
}

func (m *coMockPaymentRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*paymentdomain.Payment, error) {
	return m.payments[id], nil
}
func (m *coMockPaymentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) (*paymentdomain.Payment, error) {
	for _, p := range m.payments {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return nil, nil
}
func (m *coMockPaymentRepo) GetByOrderIDForUpdate(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*paymentdomain.Payment, error) {
	return m.GetByOrderID(ctx, exec, orderID)
}
func (m *coMockPaymentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]paymentdomain.Payment, error) {
	return nil, nil
}
func (m *coMockPaymentRepo) UpdateStatus(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ paymentdomain.PaymentStatus) error {
	return nil
}
func (m *coMockPaymentRepo) Save(_ context.Context, _ transaction.Executor, payment paymentdomain.Payment) error {
	m.payments[payment.ID] = &payment
	return nil
}
func (m *coMockPaymentRepo) ListPendingGateway(_ context.Context, _ transaction.Executor, _ time.Time) ([]paymentdomain.Payment, error) {
	return nil, nil
}
func (m *coMockPaymentRepo) ListPastDuePending(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]paymentdomain.Payment, error) {
	return nil, nil
}

// --- payment account repo ---

type coMockPaymentAccountRepo struct{}

// --- payment event repo ---

type coMockPaymentEventRepo struct {
	events []paymentdomain.PaymentEvent
}

func (m *coMockPaymentEventRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*paymentdomain.PaymentEvent, error) {
	return nil, nil
}
func (m *coMockPaymentEventRepo) ListByPaymentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]paymentdomain.PaymentEvent, error) {
	return nil, nil
}
func (m *coMockPaymentEventRepo) Create(_ context.Context, _ transaction.Executor, event paymentdomain.PaymentEvent) error {
	m.events = append(m.events, event)
	return nil
}

// --- payment instruction repo ---

type coMockPaymentInstructionRepo struct {
	instruction *paymentdomain.PaymentInstruction
}

func (m *coMockPaymentInstructionRepo) GetByPaymentMethodID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*paymentdomain.PaymentInstruction, error) {
	return m.instruction, nil
}
func (m *coMockPaymentInstructionRepo) Save(_ context.Context, _ transaction.Executor, _ paymentdomain.PaymentInstruction) error {
	return nil
}

// --- payment channel data repo ---

type coMockChannelDataRepo struct {
	saved []paymentdomain.PaymentChannelData
}

func (m *coMockChannelDataRepo) Save(_ context.Context, _ transaction.Executor, data paymentdomain.PaymentChannelData) error {
	m.saved = append(m.saved, data)
	return nil
}
func (m *coMockChannelDataRepo) GetByPaymentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*paymentdomain.PaymentChannelData, error) {
	return nil, nil
}
func (m *coMockChannelDataRepo) ListByPaymentIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID]*paymentdomain.PaymentChannelData, error) {
	return map[uuid.UUID]*paymentdomain.PaymentChannelData{}, nil
}

// --- order repo ---

type coMockOrderRepo struct {
	orders  map[uuid.UUID]*orderdomain.Order
	saveErr error
}

func (m *coMockOrderRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*orderdomain.Order, error) {
	return m.orders[id], nil
}
func (m *coMockOrderRepo) GetByNumber(_ context.Context, _ transaction.Executor, _ string) (*orderdomain.Order, error) {
	return nil, nil
}
func (m *coMockOrderRepo) UpdateStatus(_ context.Context, _ transaction.Executor, id uuid.UUID, status orderdomain.OrderStatus) error {
	if o, ok := m.orders[id]; ok {
		o.Status = status
	}
	return nil
}
func (m *coMockOrderRepo) UpdateStatusWithSLA(_ context.Context, _ transaction.Executor, id uuid.UUID, status orderdomain.OrderStatus, confirmedAt *time.Time, expiresAt *time.Time) error {
	if o, ok := m.orders[id]; ok {
		o.Status = status
		if confirmedAt != nil {
			o.ConfirmedAt = confirmedAt
		}
		if expiresAt != nil {
			o.HandlingExpiresAt = expiresAt
		}
	}
	return nil
}
func (m *coMockOrderRepo) Save(_ context.Context, _ transaction.Executor, order orderdomain.Order) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.orders[order.ID] = &order
	return nil
}
func (m *coMockOrderRepo) FindOrders(_ context.Context, _ transaction.Executor, _ orderrepo.FindOrderParams) ([]orderdomain.Order, int, error) {
	return nil, 0, nil
}
func (m *coMockOrderRepo) SetConfirmedAndExpiry(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time, _ time.Time) error {
	return nil
}
func (m *coMockOrderRepo) FindExpiredUnfulfilledOrders(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]orderdomain.Order, error) {
	return nil, nil
}

// --- order item repo ---

type coMockOrderItemRepo struct {
	items map[uuid.UUID][]orderdomain.OrderItem
}

func (m *coMockOrderItemRepo) ListByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) ([]orderdomain.OrderItem, error) {
	return m.items[orderID], nil
}
func (m *coMockOrderItemRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]orderdomain.OrderItem, error) {
	return nil, nil
}
func (m *coMockOrderItemRepo) ListByShipmentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]orderdomain.OrderItem, error) {
	return nil, nil
}

func (m *coMockOrderItemRepo) SaveBulk(_ context.Context, _ transaction.Executor, _ []orderdomain.OrderItem) error {
	return nil
}

func (m *coMockOrderItemRepo) AssignShipment(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

// --- invoice repos ---

type coMockInvoiceRepo struct{}

func (m *coMockInvoiceRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*orderdomain.Invoice, error) {
	return nil, nil
}
func (m *coMockInvoiceRepo) GetByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*orderdomain.Invoice, error) {
	return nil, nil
}
func (m *coMockInvoiceRepo) Save(_ context.Context, _ transaction.Executor, _ orderdomain.Invoice) error {
	return nil
}

type coMockInvoiceItemRepo struct{}

func (m *coMockInvoiceItemRepo) ListByInvoiceID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]orderdomain.InvoiceItem, error) {
	return nil, nil
}
func (m *coMockInvoiceItemRepo) SaveBulk(_ context.Context, _ transaction.Executor, _ []orderdomain.InvoiceItem) error {
	return nil
}

// --- inventory repo ---

type coMockInventoryRepo struct {
	reserveCalls []string
	reserveErr   error
}

func (m *coMockInventoryRepo) GetByProductIDAndShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID) (*inventorydomain.Inventory, error) {
	return nil, nil
}
func (m *coMockInventoryRepo) ListByProductID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventorydomain.Inventory, error) {
	return nil, nil
}
func (m *coMockInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID][]inventorydomain.Inventory, error) {
	return nil, nil
}
func (m *coMockInventoryRepo) ListByShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventorydomain.Inventory, error) {
	return nil, nil
}
func (m *coMockInventoryRepo) Create(_ context.Context, _ transaction.Executor, _ *inventorydomain.Inventory) error {
	return nil
}
func (m *coMockInventoryRepo) Reserve(_ context.Context, _ transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error {
	m.reserveCalls = append(m.reserveCalls, fmt.Sprintf("%s-%s-%d", productID, shopID, qty))
	return m.reserveErr
}
func (m *coMockInventoryRepo) Release(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *coMockInventoryRepo) Commit(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *coMockInventoryRepo) Restock(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *coMockInventoryRepo) Update(_ context.Context, _ transaction.Executor, _ *inventorydomain.Inventory) error {
	return nil
}
func (m *coMockInventoryRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

// --- cart repo ---

type coMockCartRepo struct {
	cart  *cartdomain.Cart
	saved bool
}

func (m *coMockCartRepo) GetWithItemsByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*cartdomain.Cart, error) {
	return m.cart, nil
}
func (m *coMockCartRepo) NewCart(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*cartdomain.Cart, error) {
	return nil, nil
}
func (m *coMockCartRepo) Save(_ context.Context, _ transaction.Executor, _ *cartdomain.Cart) error {
	m.saved = true
	return nil
}
func (m *coMockCartRepo) DeleteByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

// --- gateway mock ---

type coMockGateway struct {
	chargeResp    *paymentgateway.ChargeResponse
	chargeErr     error
	cancelCalled  bool
	cancelOrderID string
}

func (m *coMockGateway) Name() string                                                 { return "mock_gateway" }
func (m *coMockGateway) AllowedPaymentMethods() []paymentgateway.AllowedPaymentMethod { return nil }
func (m *coMockGateway) Charge(_ context.Context, _ paymentgateway.ChargeRequest) (*paymentgateway.ChargeResponse, error) {
	return m.chargeResp, m.chargeErr
}
func (m *coMockGateway) ParseNotification(_ context.Context, _ paymentgateway.NotificationPayload) (*paymentgateway.NotificationResult, error) {
	return nil, nil
}
func (m *coMockGateway) CancelTransaction(_ context.Context, gatewayOrderID string) error {
	m.cancelCalled = true
	m.cancelOrderID = gatewayOrderID
	return nil
}
func (m *coMockGateway) RefundTransaction(_ context.Context, _ paymentgateway.RefundRequest) (*paymentgateway.RefundResponse, error) {
	return nil, nil
}
func (m *coMockGateway) GetTransactionStatus(_ context.Context, _ string) (*paymentgateway.NotificationResult, error) {
	return nil, nil
}
func (m *coMockGateway) Supports(_ string) bool {
	return true
}

// capturingGateway captures the ChargeRequest for inspection.
type capturingGateway struct {
	resp     *paymentgateway.ChargeResponse
	captured *paymentgateway.ChargeRequest
}

func (c *capturingGateway) Name() string                                                 { return "mock_gateway" }
func (c *capturingGateway) AllowedPaymentMethods() []paymentgateway.AllowedPaymentMethod { return nil }
func (c *capturingGateway) Charge(_ context.Context, req paymentgateway.ChargeRequest) (*paymentgateway.ChargeResponse, error) {
	*c.captured = req
	return c.resp, nil
}
func (c *capturingGateway) ParseNotification(_ context.Context, _ paymentgateway.NotificationPayload) (*paymentgateway.NotificationResult, error) {
	return nil, nil
}
func (c *capturingGateway) CancelTransaction(_ context.Context, _ string) error { return nil }
func (c *capturingGateway) RefundTransaction(_ context.Context, _ paymentgateway.RefundRequest) (*paymentgateway.RefundResponse, error) {
	return nil, nil
}
func (c *capturingGateway) GetTransactionStatus(_ context.Context, _ string) (*paymentgateway.NotificationResult, error) {
	return nil, nil
}
func (c *capturingGateway) Supports(_ string) bool {
	return true
}

// ===========================================================================
// Test helpers
// ===========================================================================

func coDefaultUser() *userdomain.User {
	ph := "+628111111111"
	return &userdomain.User{ID: uuid.New(), Name: "Test User", Phone: &ph}
}

func coDefaultAccount(userID uuid.UUID) *authdomain.Account {
	return &authdomain.Account{ID: uuid.New(), UserID: userID, Email: "test@komecore.com"}
}

func coDefaultMethod(methodID uuid.UUID, name string) *paymentdomain.PaymentMethod {
	return &paymentdomain.PaymentMethod{ID: methodID, Name: name, Code: name, Provider: "midtrans", Type: paymentdomain.TypeQRCode, IsActive: true}
}

func coDefaultPricing(productID, shopID uuid.UUID) *orderrepo.PricingResult {
	return &orderrepo.PricingResult{
		Subtotal:         100000,
		TotalShippingFee: 15000,
		GrandTotal:       115000,
		Shops: []orderrepo.PricingShopResult{
			{
				ShopID:   shopID,
				ShopName: "Kome Store",
				Items: []orderrepo.PricingItemResult{
					{ProductID: productID, ProductName: "Product A", Quantity: 2, UnitPrice: 50000, Subtotal: 100000},
				},
				SelectedCourier: orderrepo.SelectedCourierResult{Code: "jne", Service: "REG", Fee: 15000},
			},
		},
	}
}

func coDefaultChargeResp(providerOrderID string) *paymentgateway.ChargeResponse {
	return &paymentgateway.ChargeResponse{
		GatewayTransactionID: "midtrans-tx-001",
		GatewayOrderID:       providerOrderID,
		PaymentType:          "qris",
		GrossAmount:          115000,
		Status:               "pending",
		ExpiresAt:            time.Now().Add(24 * time.Hour),
	}
}

// buildUC assembles a CreateOrderUsecase from the provided dependencies.
func buildUC(
	transactor transaction.Transactor,
	gateway paymentgateway.Provider,
	pricing *coMockPricingService,
	paymentMethodRepo *coMockPaymentMethodRepo,
	paymentAccRepo *coMockPaymentAccountRepo,
	userRepo *coMockUserRepo,
	accountRepo *coMockAccountRepo,
	inventoryRepo *coMockInventoryRepo,
	paymentStore *coMockPaymentRepo,
	orderStore *coMockOrderRepo,
	cartRepo *coMockCartRepo,
) *CreateOrderUsecase {
	return buildUCWith(transactor, gateway, pricing, paymentMethodRepo, paymentAccRepo,
		userRepo, accountRepo, inventoryRepo, paymentStore, orderStore, cartRepo, nil)
}

// buildUCWith is the full constructor; pass nil for channelDataRepo to use the default no-op mock.
func buildUCWith(
	transactor transaction.Transactor,
	gateway paymentgateway.Provider,
	pricing *coMockPricingService,
	paymentMethodRepo *coMockPaymentMethodRepo,
	paymentAccRepo *coMockPaymentAccountRepo,
	userRepo *coMockUserRepo,
	accountRepo *coMockAccountRepo,
	inventoryRepo *coMockInventoryRepo,
	paymentStore *coMockPaymentRepo,
	orderStore *coMockOrderRepo,
	cartRepo *coMockCartRepo,
	channelDataRepo *coMockChannelDataRepo,
) *CreateOrderUsecase {
	if transactor == nil {
		transactor = &coMockTransactor{}
	}
	if cartRepo == nil {
		cartRepo = &coMockCartRepo{}
	}
	if channelDataRepo == nil {
		channelDataRepo = &coMockChannelDataRepo{}
	}
	return NewCreateOrderUsecase(
		&coMockExecutor{},
		transactor,
		accountRepo,
		orderStore,
		&coMockOrderItemRepo{items: map[uuid.UUID][]orderdomain.OrderItem{}},
		&coMockInvoiceRepo{},
		&coMockInvoiceItemRepo{},
		paymentStore,
		paymentMethodRepo,
		&coMockPaymentEventRepo{},
		&coMockPaymentInstructionRepo{},
		channelDataRepo,
		inventoryRepo,
		cartRepo,
		userRepo,
		gateway,
		pricing,
	)
}

func coInput(customerID, paymentMethodID uuid.UUID, isManual bool, productID, shopID uuid.UUID) CreateOrderInput {
	return CreateOrderInput{
		UserID:          uuid.New(),
		CustomerID:      customerID,
		AddressID:       uuid.New(),
		PaymentMethodID: paymentMethodID,
		Shops: []OrderShopInput{
			{
				ShopID:   shopID,
				ShopName: "Tech Emporium",
				Items:    []OrderItemInput{{ProductID: productID, ProductName: "Mechanical Keyboard", Quantity: 2}},
			},
		},
	}
}

// ===========================================================================
// Happy-path: gateway (non-manual) order
// ===========================================================================

func TestCreateOrder_Gateway_Success(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()

	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)
	providerOrderID := uuid.New().String()

	gateway := &coMockGateway{
		chargeResp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "midtrans-tx-001",
			GatewayOrderID:       providerOrderID,
			PaymentType:          "qris",
			GrossAmount:          115000,
			Status:               "pending",
			Instructions: []paymentgateway.PaymentInstruction{
				{Type: "qris", Label: "QRIS", Value: "qr-string-value"},
			},
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}

	paymentStore := &coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}}
	orderStore := &coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		paymentStore,
		orderStore,
		nil,
	)

	result, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OrderID == uuid.Nil {
		t.Error("OrderID should not be nil")
	}
	if result.Total != 115000 {
		t.Errorf("Total = %v, want 115000", result.Total)
	}
	if result.PaymentAccount == nil {
		t.Fatal("PaymentAccount should not be nil")
	}
	if result.PaymentAccount.QRString == nil || *result.PaymentAccount.QRString != "qr-string-value" {
		t.Errorf("QRString mismatch: %v", result.PaymentAccount.QRString)
	}
}

func TestCreateOrder_Gateway_Success_NilPhone(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()

	user := &userdomain.User{ID: uuid.New(), Name: "Test User", Phone: nil}
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)
	providerOrderID := uuid.New().String()

	var capturedReq paymentgateway.ChargeRequest
	gateway := &capturingGateway{
		resp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "midtrans-tx-001",
			GatewayOrderID:       providerOrderID,
			PaymentType:          "qris",
			GrossAmount:          115000,
			Status:               "pending",
			Instructions: []paymentgateway.PaymentInstruction{
				{Type: "qris", Label: "QRIS", Value: "qr-string-value"},
			},
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
		captured: &capturedReq,
	}

	paymentStore := &coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}}
	orderStore := &coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		paymentStore,
		orderStore,
		nil,
	)

	result, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OrderID == uuid.Nil {
		t.Error("OrderID should not be nil")
	}
	if capturedReq.CustomerPhone != "" {
		t.Errorf("expected empty CustomerPhone, got %q", capturedReq.CustomerPhone)
	}
}

// ===========================================================================
// Gateway channel data: persisted on checkout and returned in result
// ===========================================================================

func TestCreateOrder_Gateway_ChannelDataPersisted(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()

	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)

	gateway := &coMockGateway{
		chargeResp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "tx-ch-001",
			GatewayOrderID:       uuid.New().String(),
			PaymentType:          "qris",
			GrossAmount:          115000,
			Status:               "pending",
			Instructions: []paymentgateway.PaymentInstruction{
				{Type: "qris", Label: "QRIS Payment", Value: "data:image/png;base64,qrdata"},
			},
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}

	channelRepo := &coMockChannelDataRepo{}

	uc := buildUCWith(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
		channelRepo,
	)

	result, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Result should surface channel data Ã¢â‚¬â€ ChannelType is sourced from method.Type,
	// not the raw gateway instruction type.
	if result.ChannelData == nil {
		t.Fatal("expected ChannelData to be set in CreateOrderResult, got nil")
	}
	if result.ChannelData.ChannelType != paymentdomain.TypeQRCode {
		t.Errorf("ChannelType = %q, want %q", result.ChannelData.ChannelType, paymentdomain.TypeQRCode)
	}
	if result.ChannelData.ActionURL == nil || *result.ChannelData.ActionURL != "data:image/png;base64,qrdata" {
		t.Errorf("ActionURL = %v, want qrdata", result.ChannelData.ActionURL)
	}

	// Repo must have received exactly one Save call
	if len(channelRepo.saved) != 1 {
		t.Fatalf("expected 1 channel data record saved, got %d", len(channelRepo.saved))
	}
	cd := channelRepo.saved[0]
	if cd.ChannelType != paymentdomain.TypeQRCode {
		t.Errorf("saved ChannelType = %q, want %q", cd.ChannelType, paymentdomain.TypeQRCode)
	}
	if cd.DisplayName != "QRIS Payment" {
		t.Errorf("saved DisplayName = %q, want \"QRIS Payment\"", cd.DisplayName)
	}
	if cd.ActionURL == nil || *cd.ActionURL != "data:image/png;base64,qrdata" {
		t.Errorf("saved ActionURL mismatch: %v", cd.ActionURL)
	}
}

func TestCreateOrder_ChargeResponse_QRIS_PopulatesActionURLAndQRString(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()

	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)

	gateway := &coMockGateway{
		chargeResp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "tx-ch-qris-002",
			GatewayOrderID:       uuid.New().String(),
			PaymentType:          "qris",
			GrossAmount:          115000,
			Status:               "pending",
			Instructions: []paymentgateway.PaymentInstruction{
				{Type: "qris", Label: "QRIS", Value: "00020101021226590014ID.LINKAJA.WWW01189360091100220085430215000000000000000520459995802ID5914KOMECORE STORE"},
			},
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}

	channelRepo := &coMockChannelDataRepo{}

	uc := buildUCWith(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
		channelRepo,
	)

	result, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// QRIS raw QR string must be saved to ChannelData.ActionURL so it persists in DB
	if result.ChannelData == nil {
		t.Fatal("expected ChannelData to be set in CreateOrderResult, got nil")
	}
	if result.ChannelData.ActionURL == nil || !strings.HasPrefix(*result.ChannelData.ActionURL, "000201") {
		t.Errorf("expected ActionURL to be populated with QR string, got %v", result.ChannelData.ActionURL)
	}
	// And PaymentAccount.QRString MUST be populated for on-screen QR generation
	if result.PaymentAccount == nil || result.PaymentAccount.QRString == nil || !strings.HasPrefix(*result.PaymentAccount.QRString, "000201") {
		t.Errorf("expected PaymentAccount.QRString to be populated, got %v", result.PaymentAccount)
	}
}

// ===========================================================================
// Charge response instruction type mapping
// ===========================================================================

func TestCreateOrder_ChargeResponse_BankTransfer_PopulatesAccountNumber(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := &paymentdomain.PaymentMethod{ID: methodID, Name: "mandiri", Code: "mandiri", Provider: "midtrans", Type: paymentdomain.TypeBankTransfer, IsActive: true}
	pricing := coDefaultPricing(productID, shopID)

	gateway := &coMockGateway{
		chargeResp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "tx-001",
			GatewayOrderID:       uuid.New().String(),
			GrossAmount:          115000,
			Instructions: []paymentgateway.PaymentInstruction{
				{Type: "bank_transfer", Label: "BCA Virtual Account", Value: "1234567890"},
			},
		},
	}

	channelRepo := &coMockChannelDataRepo{}

	uc := buildUCWith(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
		channelRepo,
	)

	result, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentAccount == nil {
		t.Fatal("PaymentAccount should not be nil")
	}
	if result.PaymentAccount.AccountNumber == nil || *result.PaymentAccount.AccountNumber != "1234567890" {
		t.Errorf("AccountNumber = %v, want 1234567890", result.PaymentAccount.AccountNumber)
	}
	if result.PaymentAccount.QRString != nil {
		t.Error("QRString should be nil for bank_transfer")
	}
	if result.ChannelData == nil {
		t.Fatal("ChannelData should not be nil")
	}
	if result.ChannelData.AccountNumber == nil || *result.ChannelData.AccountNumber != "1234567890" {
		t.Errorf("ChannelData.AccountNumber = %v, want 1234567890", result.ChannelData.AccountNumber)
	}
	if result.ChannelData.ActionURL == nil || *result.ChannelData.ActionURL != "1234567890" {
		t.Errorf("ChannelData.ActionURL = %v, want 1234567890", result.ChannelData.ActionURL)
	}
}

func TestCreateOrder_ChargeResponse_EWallet_PopulatesQRString(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := &paymentdomain.PaymentMethod{ID: methodID, Name: "gopay", Code: "gopay", Provider: "midtrans", Type: paymentdomain.TypeEWallet, IsActive: true}
	pricing := coDefaultPricing(productID, shopID)

	gateway := &coMockGateway{
		chargeResp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "tx-002",
			GatewayOrderID:       uuid.New().String(),
			GrossAmount:          115000,
			Instructions: []paymentgateway.PaymentInstruction{
				{Type: "ewallet", Label: "deeplink-redirect", Value: "https://gopay.co/pay/link"},
			},
		},
	}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	)

	result, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentAccount == nil {
		t.Fatal("PaymentAccount should not be nil")
	}
	if result.PaymentAccount.QRString == nil || *result.PaymentAccount.QRString != "https://gopay.co/pay/link" {
		t.Errorf("QRString = %v, want deeplink URL", result.PaymentAccount.QRString)
	}
}

// ===========================================================================
// Aggregate invariant: adjustment item when item sum != order total
// ===========================================================================

func TestCreateOrder_DatabaseFailure_TriggersCompensatingCancelTransaction(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()

	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)

	gatewayOrderID := "gw-order-compensate-123"
	gateway := &coMockGateway{
		chargeResp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "tx-comp-001",
			GatewayOrderID:       gatewayOrderID,
			PaymentType:          "qris",
			GrossAmount:          115000,
			Status:               "pending",
		},
	}

	orderRepoFail := &coMockOrderRepo{
		saveErr: errors.New("db save failure"),
	}

	uc := buildUCWith(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		orderRepoFail,
		nil,
		&coMockChannelDataRepo{},
	)

	_, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err == nil {
		t.Fatal("expected error from failed order save, got nil")
	}

	if !gateway.cancelCalled {
		t.Errorf("expected gateway.CancelTransaction to be called on database failure")
	}
	if gateway.cancelOrderID != gatewayOrderID {
		t.Errorf("cancelOrderID = %q, want %q", gateway.cancelOrderID, gatewayOrderID)
	}
}

func TestCreateOrder_InvariantAdjustmentItemAddedWhenSumDiffers(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")

	// GrandTotal=120000, but items+shipping=115000 Ã¢â€ â€™ 5000 adjustment
	pricing := &orderrepo.PricingResult{
		Subtotal:         100000,
		TotalShippingFee: 15000,
		GrandTotal:       120000,
		Shops: []orderrepo.PricingShopResult{
			{
				ShopID:   shopID,
				ShopName: "Kage",
				Items: []orderrepo.PricingItemResult{
					{ProductID: productID, ProductName: "Product", Quantity: 2, UnitPrice: 50000, Subtotal: 100000},
				},
				SelectedCourier: orderrepo.SelectedCourierResult{Code: "jne", Service: "REG", Fee: 15000},
			},
		},
	}

	var capturedReq paymentgateway.ChargeRequest
	gateway := &capturingGateway{
		resp:     coDefaultChargeResp(uuid.New().String()),
		captured: &capturedReq,
	}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	)

	_, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hasAdjustment bool
	var itemSum int64
	for _, item := range capturedReq.Items {
		itemSum += item.Price * int64(item.Quantity)
		if item.ID == "adjustment" {
			hasAdjustment = true
		}
	}
	if !hasAdjustment {
		t.Error("expected an adjustment charge item when item sum differs from order total")
	}
	if itemSum != 120000 {
		t.Errorf("charge items sum = %d, want 120000", itemSum)
	}
}

// ===========================================================================
// Aggregate invariant: inventory reserved for each order item
// ===========================================================================

func TestCreateOrder_InvariantInventoryReservedPerItem(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)

	invRepo := &coMockInventoryRepo{}
	gateway := &coMockGateway{chargeResp: coDefaultChargeResp(uuid.New().String())}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		invRepo,
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	)

	_, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := fmt.Sprintf("%s-%s-2", productID, shopID)
	if len(invRepo.reserveCalls) != 1 || invRepo.reserveCalls[0] != expected {
		t.Errorf("expected inventory reserve %v, got %v", expected, invRepo.reserveCalls)
	}
}

// ===========================================================================
// Aggregate invariant: payment amount equals order total
// ===========================================================================

func TestCreateOrder_InvariantPaymentAmountEqualsOrderTotal(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID) // GrandTotal = 115000

	paymentStore := &coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}}
	gateway := &coMockGateway{chargeResp: coDefaultChargeResp(uuid.New().String())}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		paymentStore,
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	)

	result, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 115000 {
		t.Errorf("result.Total = %v, want 115000", result.Total)
	}
	for _, p := range paymentStore.payments {
		if p.Amount != 115000 {
			t.Errorf("payment.Amount = %v, want 115000", p.Amount)
		}
	}
}

// ===========================================================================
// Aggregate invariant: cart items removed after order
// ===========================================================================

func TestCreateOrder_InvariantCartItemsRemovedAfterOrder(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)

	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items: []cartdomain.CartItem{
			{ID: uuid.New(), ProductID: productID, ShopID: shopID, Quantity: 2},
		},
	}
	cartRepo := &coMockCartRepo{cart: cart}
	gateway := &coMockGateway{chargeResp: coDefaultChargeResp(uuid.New().String())}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		cartRepo,
	)

	_, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cartRepo.saved {
		t.Error("expected cart to be saved after order creation")
	}
	if len(cart.Items) == 0 || cart.Items[0].DeletedAt == nil {
		t.Errorf("expected standard cart item to be marked as deleted (DeletedAt != nil), got %+v", cart.Items)
	}
}

func TestCreateOrder_CartItemShopMismatch_RemovedAfterOrder(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	cartShopID := uuid.New()
	orderShopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, orderShopID)

	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items: []cartdomain.CartItem{
			{ID: uuid.New(), ProductID: productID, ShopID: cartShopID, Quantity: 2},
		},
	}
	cartRepo := &coMockCartRepo{cart: cart}
	gateway := &coMockGateway{chargeResp: coDefaultChargeResp(uuid.New().String())}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		cartRepo,
	)

	_, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, orderShopID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cartRepo.saved {
		t.Error("expected cart to be saved after order creation")
	}
	if len(cart.Items) == 0 || cart.Items[0].DeletedAt == nil {
		t.Errorf("expected standard cart item with shop mismatch to be marked as deleted (DeletedAt != nil), got %+v", cart.Items)
	}
}

// ===========================================================================
// Failure recovery: gateway charge fails Ã¢â€ â€™ no DB writes
// ===========================================================================

func TestCreateOrder_GatewayChargeFailure_NoDBWrites(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)

	gateway := &coMockGateway{chargeErr: errors.New("midtrans: gateway unavailable")}
	paymentStore := &coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		paymentStore,
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	)

	_, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err == nil {
		t.Fatal("expected error when gateway charge fails, got nil")
	}
	if len(paymentStore.payments) != 0 {
		t.Errorf("expected no payments saved after charge failure, got %d", len(paymentStore.payments))
	}
}

// ===========================================================================
// Coordination protocol: DB fails after successful charge Ã¢â€ â€™ cancel called
// ===========================================================================

func TestCreateOrder_DBWriteFailAfterCharge_BestEffortCancel(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)

	providerOrderID := "gateway-order-ref"
	gateway := &coMockGateway{
		chargeResp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "tx-cancel",
			GatewayOrderID:       providerOrderID,
			GrossAmount:          115000,
		},
	}

	uc := buildUC(
		&coFailingTransactor{err: errors.New("database: connection lost")},
		gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	)

	_, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err == nil {
		t.Fatal("expected error when DB write fails")
	}
	if !gateway.cancelCalled {
		t.Error("expected CancelTransaction to be called as best-effort rollback after DB failure")
	}
	if gateway.cancelOrderID != providerOrderID {
		t.Errorf("CancelTransaction called with %q, want %q", gateway.cancelOrderID, providerOrderID)
	}
}

// ===========================================================================
// Coordination protocol: charge never succeeded Ã¢â€ â€™ cancel NOT called
// ===========================================================================

func TestCreateOrder_DBWriteFailNoCharge_NoBestEffortCancel(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)

	gateway := &coMockGateway{chargeErr: errors.New("gateway error")}

	uc := buildUC(
		&coFailingTransactor{err: errors.New("db error")},
		gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	)

	_, err := uc.Execute(ctx, coInput(customerID, methodID, false, productID, shopID))
	if err == nil {
		t.Fatal("expected error")
	}
	if gateway.cancelCalled {
		t.Error("CancelTransaction should NOT be called when no gateway charge was made")
	}
}

// ===========================================================================
// Failure: pricing service error Ã¢â€ â€™ early exit
// ===========================================================================

func TestCreateOrder_PricingServiceFailure(t *testing.T) {
	ctx := context.Background()
	methodID := uuid.New()
	method := coDefaultMethod(methodID, "qris")
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)

	gateway := &coMockGateway{}

	uc := buildUC(nil, gateway,
		&coMockPricingService{err: errors.New("pricing: product not found")},
		&coMockPaymentMethodRepo{method: method},
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	)

	_, err := uc.Execute(ctx, coInput(uuid.New(), methodID, false, uuid.New(), uuid.New()))
	if err == nil {
		t.Fatal("expected error when pricing fails")
	}
	if gateway.cancelCalled {
		t.Error("CancelTransaction should not be called when pricing fails before charge")
	}
}

// ===========================================================================
// Failure: payment method not found
// ===========================================================================

func TestCreateOrder_PaymentMethodNotFound(t *testing.T) {
	ctx := context.Background()
	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)

	uc := buildUC(nil, &coMockGateway{},
		&coMockPricingService{result: coDefaultPricing(uuid.New(), uuid.New())},
		&coMockPaymentMethodRepo{method: nil}, // not found
		&coMockPaymentAccountRepo{},
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	)

	_, err := uc.Execute(ctx, coInput(uuid.New(), uuid.New(), false, uuid.New(), uuid.New()))
	if err == nil {
		t.Fatal("expected error when payment method not found")
	}
}
