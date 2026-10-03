package usecase

import (
	"context"
	"testing"

	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/domain"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

type mockSyncPaymentMethodRepo struct {
	methods map[uuid.UUID]domain.PaymentMethod
}

func (m *mockSyncPaymentMethodRepo) Save(_ context.Context, _ transaction.Executor, method domain.PaymentMethod) error {
	m.methods[method.ID] = method
	return nil
}

func (m *mockSyncPaymentMethodRepo) FindByName(_ context.Context, _ transaction.Executor, _ string) (*domain.PaymentMethod, error) {
	return nil, nil
}

func (m *mockSyncPaymentMethodRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*domain.PaymentMethod, error) {
	if method, ok := m.methods[id]; ok {
		return &method, nil
	}
	return nil, nil
}

func (m *mockSyncPaymentMethodRepo) ListAll(_ context.Context, _ transaction.Executor, _ query.Sorts) ([]domain.PaymentMethod, error) {
	result := make([]domain.PaymentMethod, 0, len(m.methods))
	for _, val := range m.methods {
		result = append(result, val)
	}
	return result, nil
}

type mockSyncInstructionRepo struct {
	instructions map[uuid.UUID]domain.PaymentInstruction
}

func (m *mockSyncInstructionRepo) GetByPaymentMethodID(_ context.Context, _ transaction.Executor, methodID uuid.UUID) (*domain.PaymentInstruction, error) {
	if inst, ok := m.instructions[methodID]; ok {
		return &inst, nil
	}
	return nil, nil
}

func (m *mockSyncInstructionRepo) Save(_ context.Context, _ transaction.Executor, instruction domain.PaymentInstruction) error {
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
		methods: make(map[uuid.UUID]domain.PaymentMethod),
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
	uc1 := NewSyncPaymentMethodsUsecase(repo, &mockExecutor{}, gateway)
	err := uc1.Execute(ctx)
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
	uc2 := NewSyncPaymentMethodsUsecase(repo, &mockExecutor{}, gateway)
	err = uc2.Execute(ctx)
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

	uc3 := NewSyncPaymentMethodsUsecase(repo, &mockExecutor{}, gateway2)
	err = uc3.Execute(ctx)
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
