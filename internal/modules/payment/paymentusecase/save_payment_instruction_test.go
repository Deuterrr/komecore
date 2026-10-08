package paymentusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

// --- mocks ---

type mockSavePaymentMethodRepo struct {
	method *paymentdomain.PaymentMethod
	getErr error
}

func (m *mockSavePaymentMethodRepo) Save(_ context.Context, _ transaction.Executor, _ paymentdomain.PaymentMethod) error {
	return nil
}
func (m *mockSavePaymentMethodRepo) FindByName(_ context.Context, _ transaction.Executor, _ string) (*paymentdomain.PaymentMethod, error) {
	return nil, nil
}
func (m *mockSavePaymentMethodRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*paymentdomain.PaymentMethod, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.method != nil && m.method.ID == id {
		return m.method, nil
	}
	return nil, nil
}
func (m *mockSavePaymentMethodRepo) ListAll(_ context.Context, _ transaction.Executor, _ query.Sorts) ([]paymentdomain.PaymentMethod, error) {
	return nil, nil
}

type mockSavePaymentInstructionRepo struct {
	instruction *paymentdomain.PaymentInstruction
	getErr      error
	saveErr     error
	saved       []paymentdomain.PaymentInstruction
}

func (m *mockSavePaymentInstructionRepo) GetByPaymentMethodID(_ context.Context, _ transaction.Executor, methodID uuid.UUID) (*paymentdomain.PaymentInstruction, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.instruction != nil && m.instruction.PaymentMethodID == methodID {
		return m.instruction, nil
	}
	return nil, nil
}

func (m *mockSavePaymentInstructionRepo) Save(_ context.Context, _ transaction.Executor, instruction paymentdomain.PaymentInstruction) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, instruction)
	return nil
}

// --- tests ---

func TestSavePaymentInstruction_Create(t *testing.T) {
	ctx := context.Background()

	methodID := uuid.New()
	method := &paymentdomain.PaymentMethod{
		ID:       methodID,
		Name:     "BCA Transfer",
		Code:     "bca_va",
		IsActive: true,
	}

	methodRepo := &mockSavePaymentMethodRepo{method: method}
	instructionRepo := &mockSavePaymentInstructionRepo{}

	usecase := NewSavePaymentInstructionUsecase(methodRepo, instructionRepo, &mockExecutor{})

	input := SavePaymentInstructionInput{
		PaymentMethodID: methodID,
		Content:         "Transfer to 12345678",
	}

	err := usecase.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(instructionRepo.saved) != 1 {
		t.Fatalf("expected 1 instruction saved, got %d", len(instructionRepo.saved))
	}

	saved := instructionRepo.saved[0]
	if saved.PaymentMethodID != methodID {
		t.Errorf("expected payment method ID %v, got %v", methodID, saved.PaymentMethodID)
	}
	if saved.Content != "Transfer to 12345678" {
		t.Errorf("expected content 'Transfer to 12345678', got %v", saved.Content)
	}
	if saved.ID == uuid.Nil {
		t.Errorf("expected a valid generated UUID for instruction, got Nil")
	}
}

func TestSavePaymentInstruction_Update(t *testing.T) {
	ctx := context.Background()

	methodID := uuid.New()
	method := &paymentdomain.PaymentMethod{
		ID:       methodID,
		Name:     "BCA Transfer",
		Code:     "bca_va",
		IsActive: true,
	}

	instructionID := uuid.New()
	existingInstruction := &paymentdomain.PaymentInstruction{
		ID:              instructionID,
		PaymentMethodID: methodID,
		Content:         "Old Content",
		CreatedAt:       time.Now().Add(-1 * time.Hour),
	}

	methodRepo := &mockSavePaymentMethodRepo{method: method}
	instructionRepo := &mockSavePaymentInstructionRepo{instruction: existingInstruction}

	usecase := NewSavePaymentInstructionUsecase(methodRepo, instructionRepo, &mockExecutor{})

	input := SavePaymentInstructionInput{
		PaymentMethodID: methodID,
		Content:         "New Content",
	}

	err := usecase.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(instructionRepo.saved) != 1 {
		t.Fatalf("expected 1 instruction saved, got %d", len(instructionRepo.saved))
	}

	saved := instructionRepo.saved[0]
	if saved.ID != instructionID {
		t.Errorf("expected instruction ID to remain %v, got %v", instructionID, saved.ID)
	}
	if saved.PaymentMethodID != methodID {
		t.Errorf("expected payment method ID %v, got %v", methodID, saved.PaymentMethodID)
	}
	if saved.Content != "New Content" {
		t.Errorf("expected updated content 'New Content', got %v", saved.Content)
	}
}

func TestSavePaymentInstruction_MethodNotFound(t *testing.T) {
	ctx := context.Background()

	methodID := uuid.New()
	methodRepo := &mockSavePaymentMethodRepo{method: nil}
	instructionRepo := &mockSavePaymentInstructionRepo{}

	usecase := NewSavePaymentInstructionUsecase(methodRepo, instructionRepo, &mockExecutor{})

	input := SavePaymentInstructionInput{
		PaymentMethodID: methodID,
		Content:         "Content",
	}

	err := usecase.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		if appErr.Type != apperrors.ErrTypeNotFound {
			t.Errorf("expected NOT_FOUND error, got %v", appErr.Type)
		}
	} else {
		t.Errorf("expected AppError, got %T: %v", err, err)
	}
}

func TestSavePaymentInstruction_MethodRepoError(t *testing.T) {
	ctx := context.Background()

	methodRepo := &mockSavePaymentMethodRepo{getErr: errors.New("db error")}
	instructionRepo := &mockSavePaymentInstructionRepo{}

	usecase := NewSavePaymentInstructionUsecase(methodRepo, instructionRepo, &mockExecutor{})

	input := SavePaymentInstructionInput{
		PaymentMethodID: uuid.New(),
		Content:         "Content",
	}

	err := usecase.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSavePaymentInstruction_InstructionRepoError(t *testing.T) {
	ctx := context.Background()

	methodID := uuid.New()
	method := &paymentdomain.PaymentMethod{
		ID: methodID,
	}

	methodRepo := &mockSavePaymentMethodRepo{method: method}
	instructionRepo := &mockSavePaymentInstructionRepo{getErr: errors.New("db error")}

	usecase := NewSavePaymentInstructionUsecase(methodRepo, instructionRepo, &mockExecutor{})

	input := SavePaymentInstructionInput{
		PaymentMethodID: methodID,
		Content:         "Content",
	}

	err := usecase.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSavePaymentInstruction_SaveError(t *testing.T) {
	ctx := context.Background()

	methodID := uuid.New()
	method := &paymentdomain.PaymentMethod{
		ID: methodID,
	}

	methodRepo := &mockSavePaymentMethodRepo{method: method}
	instructionRepo := &mockSavePaymentInstructionRepo{saveErr: errors.New("db save error")}

	usecase := NewSavePaymentInstructionUsecase(methodRepo, instructionRepo, &mockExecutor{})

	input := SavePaymentInstructionInput{
		PaymentMethodID: methodID,
		Content:         "Content",
	}

	err := usecase.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
