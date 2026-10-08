package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	paymentDomain "komecore/internal/modules/payment/domain"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

// --- Mocks for GetPaymentDetail ---

type mockOrderDetailOrderPaymentManager struct {
	order   *OrderInfo
	invoice string
}

func (m *mockOrderDetailOrderPaymentManager) GetOrderForPayment(_ context.Context, _ transaction.Executor, id uuid.UUID) (*OrderInfo, error) {
	if m.order != nil && m.order.ID == id {
		return m.order, nil
	}
	return nil, nil
}
func (m *mockOrderDetailOrderPaymentManager) GetInvoiceNumber(_ context.Context, _ transaction.Executor, _ uuid.UUID) (string, error) {
	return m.invoice, nil
}
func (m *mockOrderDetailOrderPaymentManager) ConfirmOrderPayment(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time, _ time.Duration) ([]OrderItemInfo, error) {
	return nil, nil
}
func (m *mockOrderDetailOrderPaymentManager) ExpireOrderPayment(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]OrderItemInfo, error) {
	return nil, nil
}
func (m *mockOrderDetailOrderPaymentManager) CancelOrderPayment(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]OrderItemInfo, error) {
	return nil, nil
}

type mockOrderDetailPaymentRepo struct {
	payment *paymentDomain.Payment
}

func (m *mockOrderDetailPaymentRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*paymentDomain.Payment, error) {
	return nil, nil
}
func (m *mockOrderDetailPaymentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) (*paymentDomain.Payment, error) {
	if m.payment != nil && m.payment.OrderID == orderID {
		return m.payment, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentRepo) GetByOrderIDForUpdate(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*paymentDomain.Payment, error) {
	return m.GetByOrderID(ctx, exec, orderID)
}
func (m *mockOrderDetailPaymentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]paymentDomain.Payment, error) {
	return nil, nil
}
func (m *mockOrderDetailPaymentRepo) UpdateStatus(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ paymentDomain.PaymentStatus) error {
	return nil
}
func (m *mockOrderDetailPaymentRepo) Save(_ context.Context, _ transaction.Executor, _ paymentDomain.Payment) error {
	return nil
}
func (m *mockOrderDetailPaymentRepo) ListPendingGateway(_ context.Context, _ transaction.Executor, _ time.Time) ([]paymentDomain.Payment, error) {
	if m.payment != nil && m.payment.Status == paymentDomain.PaymentStatusPending && m.payment.Provider == "gateway" {
		return []paymentDomain.Payment{*m.payment}, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentRepo) ListPastDuePending(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]paymentDomain.Payment, error) {
	return nil, nil
}

type mockOrderDetailPaymentMethodRepo struct {
	method *paymentDomain.PaymentMethod
}

func (m *mockOrderDetailPaymentMethodRepo) Save(_ context.Context, _ transaction.Executor, _ paymentDomain.PaymentMethod) error {
	return nil
}
func (m *mockOrderDetailPaymentMethodRepo) FindByName(_ context.Context, _ transaction.Executor, _ string) (*paymentDomain.PaymentMethod, error) {
	return nil, nil
}
func (m *mockOrderDetailPaymentMethodRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*paymentDomain.PaymentMethod, error) {
	if m.method != nil && m.method.ID == id {
		return m.method, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentMethodRepo) ListAll(_ context.Context, _ transaction.Executor, _ query.Sorts) ([]paymentDomain.PaymentMethod, error) {
	return nil, nil
}

type mockOrderDetailPaymentAccountRepo struct{}

type mockOrderDetailPaymentInstructionRepo struct {
	instruction *paymentDomain.PaymentInstruction
}

func (m *mockOrderDetailPaymentInstructionRepo) GetByPaymentMethodID(_ context.Context, _ transaction.Executor, methodID uuid.UUID) (*paymentDomain.PaymentInstruction, error) {
	if m.instruction != nil && m.instruction.PaymentMethodID == methodID {
		return m.instruction, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentInstructionRepo) Save(_ context.Context, _ transaction.Executor, _ paymentDomain.PaymentInstruction) error {
	return nil
}

type mockOrderDetailPaymentChannelDataRepo struct {
	channelData *paymentDomain.PaymentChannelData
}

func (m *mockOrderDetailPaymentChannelDataRepo) Save(_ context.Context, _ transaction.Executor, _ paymentDomain.PaymentChannelData) error {
	return nil
}
func (m *mockOrderDetailPaymentChannelDataRepo) GetByPaymentID(_ context.Context, _ transaction.Executor, paymentID uuid.UUID) (*paymentDomain.PaymentChannelData, error) {
	if m.channelData != nil && m.channelData.PaymentID == paymentID {
		return m.channelData, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentChannelDataRepo) ListByPaymentIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID]*paymentDomain.PaymentChannelData, error) {
	return nil, nil
}

// --- Tests ---

func TestGetPaymentDetail_Gateway_Success(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	customerID := uuid.New()
	methodID := uuid.New()
	paymentID := uuid.New()

	order := &OrderInfo{
		ID:         orderID,
		CustomerID: customerID,
		Number:     "ORD-12345",
		Total:      115000,
	}

	payment := &paymentDomain.Payment{
		ID:        paymentID,
		OrderID:   orderID,
		MethodID:  methodID,
		Provider:  "midtrans",
		Amount:    115000,
		Status:    paymentDomain.PaymentStatusPending,
		ExpiresAt: pointer(time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)),
	}

	method := &paymentDomain.PaymentMethod{
		ID:   methodID,
		Name: "QRIS",
		Type: paymentDomain.TypeQRCode,
	}

	qrString := "data:image/png;base64,qrstring"
	channelData := &paymentDomain.PaymentChannelData{
		ID:          uuid.New(),
		PaymentID:   paymentID,
		ChannelType: paymentDomain.TypeQRCode,
		DisplayName: "QRIS",
		QRString:    &qrString,
		ActionURL:   &qrString,
	}

	instruction := &paymentDomain.PaymentInstruction{
		ID:              uuid.New(),
		PaymentMethodID: methodID,
		Content:         "Pay invoice {{invoice_number}} amount {{amount}} before {{expired_at}} using {{qr_string}}",
	}

	uc := NewGetPaymentDetailUsecase(
		&mockExecutor{},
		&mockOrderDetailOrderPaymentManager{order: order, invoice: "INV-12345"},
		&mockOrderDetailPaymentRepo{payment: payment},
		&mockOrderDetailPaymentMethodRepo{method: method},
		&mockOrderDetailPaymentInstructionRepo{instruction: instruction},
		&mockOrderDetailPaymentChannelDataRepo{channelData: channelData},
	)

	res, err := uc.Execute(ctx, GetPaymentDetailInput{
		OrderID:    orderID,
		CustomerID: &customerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil result")
	}

	if res.Payment.ID != paymentID {
		t.Errorf("expected payment ID %v, got %v", paymentID, res.Payment.ID)
	}

	if res.ChannelData == nil {
		t.Fatal("expected non-nil channel data")
	}

	if res.ChannelData.ChannelType != paymentDomain.TypeQRCode {
		t.Errorf("expected channel type %v, got %v", paymentDomain.TypeQRCode, res.ChannelData.ChannelType)
	}

	if res.Instruction == nil {
		t.Fatal("expected non-nil rendered instruction")
	}

	expectedInstruction := "Pay invoice INV-12345 amount 115000 before 2026-07-12T10:00:00Z using data:image/png;base64,qrstring"
	if *res.Instruction != expectedInstruction {
		t.Errorf("rendered instruction mismatch:\nwant: %q\ngot:  %q", expectedInstruction, *res.Instruction)
	}
}

func TestGetPaymentDetail_BankTransfer_Success(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	customerID := uuid.New()
	methodID := uuid.New()
	paymentID := uuid.New()

	order := &OrderInfo{
		ID:         orderID,
		CustomerID: customerID,
		Number:     "ORD-54321",
		Total:      250000,
	}

	payment := &paymentDomain.Payment{
		ID:        paymentID,
		OrderID:   orderID,
		MethodID:  methodID,
		Provider:  "midtrans",
		Amount:    250000,
		Status:    paymentDomain.PaymentStatusPending,
		ExpiresAt: pointer(time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)),
	}

	method := &paymentDomain.PaymentMethod{
		ID:   methodID,
		Name: "BCA Virtual Account",
		Type: paymentDomain.TypeBankTransfer,
	}

	vaNumber := "123456789012"
	channelData := &paymentDomain.PaymentChannelData{
		ID:            uuid.New(),
		PaymentID:     paymentID,
		ChannelType:   paymentDomain.TypeBankTransfer,
		DisplayName:   "BCA Virtual Account",
		AccountNumber: &vaNumber,
		ActionURL:     &vaNumber,
	}

	instruction := &paymentDomain.PaymentInstruction{
		ID:              uuid.New(),
		PaymentMethodID: methodID,
		Content:         "Pay invoice {{invoice_number}} amount {{amount}} to {{va_number}}",
	}

	uc := NewGetPaymentDetailUsecase(
		&mockExecutor{},
		&mockOrderDetailOrderPaymentManager{order: order, invoice: "INV-54321"},
		&mockOrderDetailPaymentRepo{payment: payment},
		&mockOrderDetailPaymentMethodRepo{method: method},
		&mockOrderDetailPaymentInstructionRepo{instruction: instruction},
		&mockOrderDetailPaymentChannelDataRepo{channelData: channelData},
	)

	res, err := uc.Execute(ctx, GetPaymentDetailInput{
		OrderID:    orderID,
		CustomerID: &customerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ChannelData.AccountNumber == nil || *res.ChannelData.AccountNumber != vaNumber {
		t.Errorf("expected account number %s, got %v", vaNumber, res.ChannelData.AccountNumber)
	}

	expectedInstruction := "Pay invoice INV-54321 amount 250000 to 123456789012"
	if res.Instruction == nil || *res.Instruction != expectedInstruction {
		t.Errorf("rendered instruction mismatch:\nwant: %q\ngot:  %v", expectedInstruction, res.Instruction)
	}
}

func TestGetPaymentDetail_OrderNotFound(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	customerID := uuid.New()

	uc := NewGetPaymentDetailUsecase(
		&mockExecutor{},
		&mockOrderDetailOrderPaymentManager{order: nil},
		&mockOrderDetailPaymentRepo{},
		&mockOrderDetailPaymentMethodRepo{},
		&mockOrderDetailPaymentInstructionRepo{},
		&mockOrderDetailPaymentChannelDataRepo{},
	)

	_, err := uc.Execute(ctx, GetPaymentDetailInput{
		OrderID:    orderID,
		CustomerID: &customerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %v", err)
	}

	if appErr.StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", appErr.StatusCode)
	}
}

func TestGetPaymentDetail_WrongCustomer(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	customerID := uuid.New()
	wrongCustomerID := uuid.New()

	order := &OrderInfo{
		ID:         orderID,
		CustomerID: customerID,
	}

	uc := NewGetPaymentDetailUsecase(
		&mockExecutor{},
		&mockOrderDetailOrderPaymentManager{order: order},
		&mockOrderDetailPaymentRepo{},
		&mockOrderDetailPaymentMethodRepo{},
		&mockOrderDetailPaymentInstructionRepo{},
		&mockOrderDetailPaymentChannelDataRepo{},
	)

	_, err := uc.Execute(ctx, GetPaymentDetailInput{
		OrderID:    orderID,
		CustomerID: &wrongCustomerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %v", err)
	}

	if appErr.StatusCode != 404 {
		t.Errorf("expected status code 404 (masked from Forbidden), got %d", appErr.StatusCode)
	}
}

func TestGetPaymentDetail_BankTransfer_InstructionsRenderedWithVANumberAndBillerCode(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	customerID := uuid.New()
	methodID := uuid.New()
	paymentID := uuid.New()

	order := &OrderInfo{
		ID:         orderID,
		CustomerID: customerID,
		Number:     "ORD-20260907-001",
		Total:      469000,
	}

	payment := &paymentDomain.Payment{
		ID:        paymentID,
		OrderID:   orderID,
		MethodID:  methodID,
		Provider:  "midtrans",
		Amount:    469000,
		Status:    paymentDomain.PaymentStatusPending,
		ExpiresAt: pointer(time.Date(2026, 9, 8, 22, 41, 5, 0, time.UTC)),
	}

	method := &paymentDomain.PaymentMethod{
		ID:   methodID,
		Name: "Mandiri Bill",
		Type: paymentDomain.TypeBankTransfer,
	}

	vaNumber := "990000000260"
	channelData := &paymentDomain.PaymentChannelData{
		ID:          uuid.New(),
		PaymentID:   paymentID,
		ChannelType: paymentDomain.TypeBankTransfer,
		DisplayName: "MANDIRI Bill Key (Biller Code: 70012)",
		ActionURL:   &vaNumber,
	}

	instruction := &paymentDomain.PaymentInstruction{
		ID:              uuid.New(),
		PaymentMethodID: methodID,
		Content:         "Kode Biller: 70012, Bill Key: {{bill_key}}, VA: {{va_number}}, Exp: {{expired_at}}",
	}

	uc := NewGetPaymentDetailUsecase(
		&mockExecutor{},
		&mockOrderDetailOrderPaymentManager{order: order, invoice: "INV-20260907-001"},
		&mockOrderDetailPaymentRepo{payment: payment},
		&mockOrderDetailPaymentMethodRepo{method: method},
		&mockOrderDetailPaymentInstructionRepo{instruction: instruction},
		&mockOrderDetailPaymentChannelDataRepo{channelData: channelData},
	)

	res, err := uc.Execute(ctx, GetPaymentDetailInput{
		OrderID:    orderID,
		CustomerID: &customerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil || res.Instruction == nil {
		t.Fatal("expected non-nil rendered instruction")
	}

	rendered := *res.Instruction
	if !strings.Contains(rendered, "Kode Biller: 70012") {
		t.Errorf("rendered instruction missing biller code: %s", rendered)
	}
	if !strings.Contains(rendered, "Bill Key: 990000000260") {
		t.Errorf("rendered instruction missing bill_key: %s", rendered)
	}
	if !strings.Contains(rendered, "VA: 990000000260") {
		t.Errorf("rendered instruction missing va_number: %s", rendered)
	}
}

func pointer[T any](v T) *T {
	return &v
}
