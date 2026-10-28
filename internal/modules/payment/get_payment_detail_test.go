package payment

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"

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
	payment *Payment
}

func (m *mockOrderDetailPaymentRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*Payment, error) {
	return nil, nil
}
func (m *mockOrderDetailPaymentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) (*Payment, error) {
	if m.payment != nil && m.payment.OrderID == orderID {
		return m.payment, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentRepo) GetByOrderIDForUpdate(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*Payment, error) {
	return m.GetByOrderID(ctx, exec, orderID)
}
func (m *mockOrderDetailPaymentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]Payment, error) {
	return nil, nil
}
func (m *mockOrderDetailPaymentRepo) UpdateStatus(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ PaymentStatus) error {
	return nil
}
func (m *mockOrderDetailPaymentRepo) Save(_ context.Context, _ transaction.Executor, _ Payment) error {
	return nil
}
func (m *mockOrderDetailPaymentRepo) ListPendingGateway(_ context.Context, _ transaction.Executor, _ time.Time) ([]Payment, error) {
	if m.payment != nil && m.payment.Status == PaymentStatusPending && m.payment.Provider == "gateway" {
		return []Payment{*m.payment}, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentRepo) ListPastDuePending(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]Payment, error) {
	return nil, nil
}

type mockOrderDetailPaymentMethodRepo struct {
	method *PaymentMethod
}

func (m *mockOrderDetailPaymentMethodRepo) Save(_ context.Context, _ transaction.Executor, _ PaymentMethod) error {
	return nil
}
func (m *mockOrderDetailPaymentMethodRepo) FindByName(_ context.Context, _ transaction.Executor, _ string) (*PaymentMethod, error) {
	return nil, nil
}
func (m *mockOrderDetailPaymentMethodRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*PaymentMethod, error) {
	if m.method != nil && m.method.ID == id {
		return m.method, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentMethodRepo) ListAll(_ context.Context, _ transaction.Executor, _ pagination.Sorts) ([]PaymentMethod, error) {
	return nil, nil
}

type mockOrderDetailPaymentAccountRepo struct{}

type mockOrderDetailPaymentInstructionRepo struct {
	instruction *PaymentInstruction
}

func (m *mockOrderDetailPaymentInstructionRepo) GetByPaymentMethodID(_ context.Context, _ transaction.Executor, methodID uuid.UUID) (*PaymentInstruction, error) {
	if m.instruction != nil && m.instruction.PaymentMethodID == methodID {
		return m.instruction, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentInstructionRepo) Save(_ context.Context, _ transaction.Executor, _ PaymentInstruction) error {
	return nil
}

type mockOrderDetailPaymentChannelDataRepo struct {
	channelData *PaymentChannelData
}

func (m *mockOrderDetailPaymentChannelDataRepo) Save(_ context.Context, _ transaction.Executor, _ PaymentChannelData) error {
	return nil
}
func (m *mockOrderDetailPaymentChannelDataRepo) GetByPaymentID(_ context.Context, _ transaction.Executor, paymentID uuid.UUID) (*PaymentChannelData, error) {
	if m.channelData != nil && m.channelData.PaymentID == paymentID {
		return m.channelData, nil
	}
	return nil, nil
}
func (m *mockOrderDetailPaymentChannelDataRepo) ListByPaymentIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID]*PaymentChannelData, error) {
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

	pmt := &Payment{
		ID:        paymentID,
		OrderID:   orderID,
		MethodID:  methodID,
		Provider:  "midtrans",
		Amount:    115000,
		Status:    PaymentStatusPending,
		ExpiresAt: pointer(time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)),
	}

	method := &PaymentMethod{
		ID:   methodID,
		Name: "QRIS",
		Type: TypeQRCode,
	}

	qrString := "data:image/png;base64,qrstring"
	channelData := &PaymentChannelData{
		ID:          uuid.New(),
		PaymentID:   paymentID,
		ChannelType: TypeQRCode,
		DisplayName: "QRIS",
		QRString:    &qrString,
		ActionURL:   &qrString,
	}

	instruction := &PaymentInstruction{
		ID:              uuid.New(),
		PaymentMethodID: methodID,
		Content:         "Pay invoice {{invoice_number}} amount {{amount}} before {{expired_at}} using {{qr_string}}",
	}

	svc := NewService(
		&mockOrderDetailPaymentRepo{payment: pmt},
		&mockOrderDetailPaymentMethodRepo{method: method},
		&mockOrderDetailPaymentInstructionRepo{instruction: instruction},
		&mockOrderDetailPaymentChannelDataRepo{channelData: channelData},
		&mockOrderDetailOrderPaymentManager{order: order, invoice: "INV-12345"},
		nil,
		nil,
		nil,
		&mockExecutor{},
		&mockTransactor{},
		&mockLogger{},
		24*time.Hour,
		100,
		5,
	)

	res, err := svc.GetPaymentDetail(ctx, GetPaymentDetailInput{
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

	if res.ChannelData.ChannelType != TypeQRCode {
		t.Errorf("expected channel type %v, got %v", TypeQRCode, res.ChannelData.ChannelType)
	}

	if res.Instruction == nil {
		t.Fatal("expected non-nil rendered instruction")
	}

	expectedInstruction := "Pay invoice INV-12345 amount 115000 before 2026-07-12T10:00:00Z using data:image/png;base64,qrstring"
	if *res.Instruction != expectedInstruction {
		t.Errorf("rendered instruction mismatch:\nwant: %q\ngot:  %q", expectedInstruction, *res.Instruction)
	}
}

func TestGetPaymentDetail_QRCode_ActionURLFallback(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	customerID := uuid.New()
	methodID := uuid.New()
	paymentID := uuid.New()

	order := &OrderInfo{
		ID:         orderID,
		CustomerID: customerID,
		Number:     "ORD-QR-1",
		Total:      50000,
	}

	pmt := &Payment{
		ID:        paymentID,
		OrderID:   orderID,
		MethodID:  methodID,
		Provider:  "midtrans",
		Amount:    50000,
		Status:    PaymentStatusPending,
		CreatedAt: time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC),
	}

	method := &PaymentMethod{
		ID:   methodID,
		Name: "QRIS",
		Type: TypeQRCode,
	}

	qrURL := "https://api.midtrans.com/qris/qr-code-image.png"
	// Simulating legacy channel data where QRString is nil and ActionURL holds the payload/url
	channelData := &PaymentChannelData{
		ID:          uuid.New(),
		PaymentID:   paymentID,
		ChannelType: TypeQRCode,
		DisplayName: "QRIS",
		QRString:    nil,
		ActionURL:   &qrURL,
	}

	instruction := &PaymentInstruction{
		ID:              uuid.New(),
		PaymentMethodID: methodID,
		Content:         "VA: {{va_number}} | QR: {{qr_string}}",
	}

	svc := NewService(
		&mockOrderDetailPaymentRepo{payment: pmt},
		&mockOrderDetailPaymentMethodRepo{method: method},
		&mockOrderDetailPaymentInstructionRepo{instruction: instruction},
		&mockOrderDetailPaymentChannelDataRepo{channelData: channelData},
		&mockOrderDetailOrderPaymentManager{order: order, invoice: "INV-QR-1"},
		nil,
		nil,
		nil,
		&mockExecutor{},
		&mockTransactor{},
		&mockLogger{},
		24*time.Hour,
		100,
		5,
	)

	res, err := svc.GetPaymentDetail(ctx, GetPaymentDetailInput{
		OrderID:    orderID,
		CustomerID: &customerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Instruction == nil {
		t.Fatal("expected non-nil instruction")
	}

	expected := "VA:  | QR: https://api.midtrans.com/qris/qr-code-image.png"
	if *res.Instruction != expected {
		t.Errorf("expected instruction %q, got %q", expected, *res.Instruction)
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

	pmt := &Payment{
		ID:        paymentID,
		OrderID:   orderID,
		MethodID:  methodID,
		Provider:  "midtrans",
		Amount:    250000,
		Status:    PaymentStatusPending,
		ExpiresAt: pointer(time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)),
	}

	method := &PaymentMethod{
		ID:   methodID,
		Name: "BCA Virtual Account",
		Type: TypeBankTransfer,
	}

	vaNumber := "123456789012"
	channelData := &PaymentChannelData{
		ID:            uuid.New(),
		PaymentID:     paymentID,
		ChannelType:   TypeBankTransfer,
		DisplayName:   "BCA Virtual Account",
		AccountNumber: &vaNumber,
		ActionURL:     &vaNumber,
	}

	instruction := &PaymentInstruction{
		ID:              uuid.New(),
		PaymentMethodID: methodID,
		Content:         "Pay invoice {{invoice_number}} amount {{amount}} to {{va_number}}",
	}

	svc := NewService(
		&mockOrderDetailPaymentRepo{payment: pmt},
		&mockOrderDetailPaymentMethodRepo{method: method},
		&mockOrderDetailPaymentInstructionRepo{instruction: instruction},
		&mockOrderDetailPaymentChannelDataRepo{channelData: channelData},
		&mockOrderDetailOrderPaymentManager{order: order, invoice: "INV-54321"},
		nil,
		nil,
		nil,
		&mockExecutor{},
		&mockTransactor{},
		&mockLogger{},
		24*time.Hour,
		100,
		5,
	)

	res, err := svc.GetPaymentDetail(ctx, GetPaymentDetailInput{
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

	svc := NewService(
		&mockOrderDetailPaymentRepo{},
		&mockOrderDetailPaymentMethodRepo{},
		&mockOrderDetailPaymentInstructionRepo{},
		&mockOrderDetailPaymentChannelDataRepo{},
		&mockOrderDetailOrderPaymentManager{order: nil},
		nil,
		nil,
		nil,
		&mockExecutor{},
		&mockTransactor{},
		&mockLogger{},
		24*time.Hour,
		100,
		5,
	)

	_, err := svc.GetPaymentDetail(ctx, GetPaymentDetailInput{
		OrderID:    orderID,
		CustomerID: &customerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
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

	svc := NewService(
		&mockOrderDetailPaymentRepo{},
		&mockOrderDetailPaymentMethodRepo{},
		&mockOrderDetailPaymentInstructionRepo{},
		&mockOrderDetailPaymentChannelDataRepo{},
		&mockOrderDetailOrderPaymentManager{order: order},
		nil,
		nil,
		nil,
		&mockExecutor{},
		&mockTransactor{},
		&mockLogger{},
		24*time.Hour,
		100,
		5,
	)

	_, err := svc.GetPaymentDetail(ctx, GetPaymentDetailInput{
		OrderID:    orderID,
		CustomerID: &wrongCustomerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
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

	pmt := &Payment{
		ID:        paymentID,
		OrderID:   orderID,
		MethodID:  methodID,
		Provider:  "midtrans",
		Amount:    469000,
		Status:    PaymentStatusPending,
		ExpiresAt: pointer(time.Date(2026, 9, 8, 22, 41, 5, 0, time.UTC)),
	}

	method := &PaymentMethod{
		ID:   methodID,
		Name: "Mandiri Bill",
		Type: TypeBankTransfer,
	}

	vaNumber := "990000000260"
	channelData := &PaymentChannelData{
		ID:          uuid.New(),
		PaymentID:   paymentID,
		ChannelType: TypeBankTransfer,
		DisplayName: "MANDIRI Bill Key (Biller Code: 70012)",
		ActionURL:   &vaNumber,
	}

	instruction := &PaymentInstruction{
		ID:              uuid.New(),
		PaymentMethodID: methodID,
		Content:         "Kode Biller: 70012, Bill Key: {{bill_key}}, VA: {{va_number}}, Exp: {{expired_at}}",
	}

	svc := NewService(
		&mockOrderDetailPaymentRepo{payment: pmt},
		&mockOrderDetailPaymentMethodRepo{method: method},
		&mockOrderDetailPaymentInstructionRepo{instruction: instruction},
		&mockOrderDetailPaymentChannelDataRepo{channelData: channelData},
		&mockOrderDetailOrderPaymentManager{order: order, invoice: "INV-20260907-001"},
		nil,
		nil,
		nil,
		&mockExecutor{},
		&mockTransactor{},
		&mockLogger{},
		24*time.Hour,
		100,
		5,
	)

	res, err := svc.GetPaymentDetail(ctx, GetPaymentDetailInput{
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
