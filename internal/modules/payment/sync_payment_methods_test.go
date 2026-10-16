package payment

import (
	"context"
	"testing"
	"time"

	"komecore/internal/infra/paymentgateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type mockSyncPaymentMethodRepo struct {
	methods map[uuid.UUID]PaymentMethod
}

func (m *mockSyncPaymentMethodRepo) Save(_ context.Context, _ transaction.Executor, method PaymentMethod) error {
	m.methods[method.ID] = method
	return nil
}

func (m *mockSyncPaymentMethodRepo) FindByName(_ context.Context, _ transaction.Executor, _ string) (*PaymentMethod, error) {
	return nil, nil
}

func (m *mockSyncPaymentMethodRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*PaymentMethod, error) {
	if method, ok := m.methods[id]; ok {
		return &method, nil
	}
	return nil, nil
}

func (m *mockSyncPaymentMethodRepo) ListAll(_ context.Context, _ transaction.Executor, _ pagination.Sorts) ([]PaymentMethod, error) {
	result := make([]PaymentMethod, 0, len(m.methods))
	for _, val := range m.methods {
		result = append(result, val)
	}
	return result, nil
}

type mockSyncInstructionRepo struct {
	instructions map[uuid.UUID]PaymentInstruction
}

func (m *mockSyncInstructionRepo) GetByPaymentMethodID(_ context.Context, _ transaction.Executor, methodID uuid.UUID) (*PaymentInstruction, error) {
	if inst, ok := m.instructions[methodID]; ok {
		return &inst, nil
	}
	return nil, nil
}

func (m *mockSyncInstructionRepo) Save(_ context.Context, _ transaction.Executor, instruction PaymentInstruction) error {
	m.instructions[instruction.PaymentMethodID] = instruction
	return nil
}

type mockSyncGateway struct {
	providerName   string
	allowedMethods []paymentgateway.AllowedPaymentMethod
}

func (g *mockSyncGateway) Name() string {
	return g.providerName
}

func (g *mockSyncGateway) AllowedPaymentMethods() []paymentgateway.AllowedPaymentMethod {
	return g.allowedMethods
}

func (g *mockSyncGateway) Supports(code string) bool {
	for _, m := range g.allowedMethods {
		if m.Code == code {
			return true
		}
	}
	return false
}

func (g *mockSyncGateway) Charge(_ context.Context, _ paymentgateway.ChargeRequest) (*paymentgateway.ChargeResponse, error) {
	return nil, nil
}

func (g *mockSyncGateway) ParseNotification(_ context.Context, _ paymentgateway.NotificationPayload) (*paymentgateway.NotificationResult, error) {
	return nil, nil
}

func (g *mockSyncGateway) GetTransactionStatus(_ context.Context, _ string) (*paymentgateway.NotificationResult, error) {
	return nil, nil
}

func (g *mockSyncGateway) CancelTransaction(_ context.Context, _ string) error {
	return nil
}

func (g *mockSyncGateway) RefundTransaction(_ context.Context, _ paymentgateway.RefundRequest) (*paymentgateway.RefundResponse, error) {
	return nil, nil
}

func TestSyncPaymentMethods_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockSyncPaymentMethodRepo{
		methods: make(map[uuid.UUID]PaymentMethod),
	}

	gateway := &mockSyncGateway{
		providerName: "midtrans",
		allowedMethods: []paymentgateway.AllowedPaymentMethod{
			{
				Code:          "gopay",
				Name:          "GoPay E-Wallet",
				Type:          "ewallet",
				FeeType:       "percentage",
				FeePercentage: 0.02,
				Description:   "Pay with GoPay",
			},
			{
				Code:     "qris",
				Name:     "QRIS QR Code",
				Type:     "qr_code",
				FeeType:  "flat",
				FeeFixed: 1000,
			},
		},
	}

	// 1. First sync - inserts all allowed methods
	svc1 := NewService(nil, repo, nil, nil, nil, nil, gateway, nil, &mockExecutor{}, &mockTransactor{}, &mockLogger{}, 24*time.Hour, 100, 5)
	err := svc1.SyncPaymentMethods(ctx)
	if err != nil {
		t.Fatalf("unexpected error on first sync: %v", err)
	}

	if len(repo.methods) != 2 {
		t.Errorf("expected 2 methods saved in DB, got %d", len(repo.methods))
	}

	var gopayID uuid.UUID
	for _, m := range repo.methods {
		if m.Code == "gopay" {
			gopayID = m.ID
			if m.Name != "GoPay E-Wallet" || m.FeePercentage != 0.02 {
				t.Errorf("GoPay properties not set correctly: %+v", m)
			}
			if !m.IsActive {
				t.Error("newly synced payment method should be active by default")
			}
		}
	}

	// 2. Customise active state on gopay
	gopay := repo.methods[gopayID]
	gopay.IsActive = false
	repo.methods[gopayID] = gopay

	// 3. Second sync - update properties, preserve active state
	gateway.allowedMethods[0].Name = "GoPay New Name"
	svc2 := NewService(nil, repo, nil, nil, nil, nil, gateway, nil, &mockExecutor{}, &mockTransactor{}, &mockLogger{}, 24*time.Hour, 100, 5)
	err = svc2.SyncPaymentMethods(ctx)
	if err != nil {
		t.Fatalf("unexpected error on second sync: %v", err)
	}

	gopay = repo.methods[gopayID]
	if gopay.Name != "GoPay New Name" {
		t.Errorf("expected name updated to GoPay New Name, got %s", gopay.Name)
	}
	if gopay.IsActive {
		t.Error("expected GoPay customized IsActive=false state to be preserved")
	}

	// 4. Change provider / gateway name - old records deactivated automatically
	gateway2 := &mockSyncGateway{
		providerName: "stripe",
		allowedMethods: []paymentgateway.AllowedPaymentMethod{
			{
				Code:     "qris",
				Name:     "Stripe QRIS",
				Type:     "qr_code",
				FeeType:  "flat",
				FeeFixed: 500,
			},
		},
	}

	svc3 := NewService(nil, repo, nil, nil, nil, nil, gateway2, nil, &mockExecutor{}, &mockTransactor{}, &mockLogger{}, 24*time.Hour, 100, 5)
	err = svc3.SyncPaymentMethods(ctx)
	if err != nil {
		t.Fatalf("unexpected error on third sync: %v", err)
	}

	// Midtrans GoPay & QRIS should be disabled (IsActive = false)
	// Stripe QRIS should be created and active
	var stripeQrisCount int
	for _, m := range repo.methods {
		if m.Provider == "midtrans" {
			if m.IsActive {
				t.Errorf("expected midtrans method %s to be deactivated, but it is active", m.Code)
			}
		} else if m.Provider == "stripe" && m.Code == "qris" {
			stripeQrisCount++
			if !m.IsActive {
				t.Error("expected stripe qris to be active")
			}
		}
	}

	if stripeQrisCount != 1 {
		t.Errorf("expected stripe qris method count 1, got %d", stripeQrisCount)
	}
}
