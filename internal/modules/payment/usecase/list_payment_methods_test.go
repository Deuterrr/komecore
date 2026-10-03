package usecase

import (
	"context"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/domain"
	"komecore/internal/modules/payment/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

type mockListPaymentMethodRepo struct {
	methods     []domain.PaymentMethod
	calledSorts query.Sorts
}

func (m *mockListPaymentMethodRepo) Save(_ context.Context, _ transaction.Executor, _ domain.PaymentMethod) error {
	return nil
}
func (m *mockListPaymentMethodRepo) FindByName(_ context.Context, _ transaction.Executor, _ string) (*domain.PaymentMethod, error) {
	return nil, nil
}
func (m *mockListPaymentMethodRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*domain.PaymentMethod, error) {
	return nil, nil
}
func (m *mockListPaymentMethodRepo) ListAll(_ context.Context, _ transaction.Executor, sorts query.Sorts) ([]domain.PaymentMethod, error) {
	m.calledSorts = sorts
	return m.methods, nil
}

func TestListPaymentMethods_DefaultSortAndBinding(t *testing.T) {
	ctx := context.Background()

	methodID := uuid.New()
	instructionID := uuid.New()
	methods := []domain.PaymentMethod{
		{
			ID:       methodID,
			Name:     "BCA Transfer",
			Code:     "bca_va",
			IsActive: true,
			Instruction: &domain.PaymentInstruction{
				ID:              instructionID,
				PaymentMethodID: methodID,
				Content:         "Transfer instructions",
				CreatedAt:       time.Now(),
			},
		},
	}

	repo := &mockListPaymentMethodRepo{methods: methods}
	usecase := NewListPaymentMethodUsecase(repo, &mockExecutor{})

	res, err := usecase.ListAll(ctx, ListPaymentMethodInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 payment method, got %d", len(res))
	}

	if res[0].Instruction == nil {
		t.Fatal("expected instruction to be bound, got nil")
	}

	if res[0].Instruction.ID != instructionID {
		t.Errorf("expected instruction ID %v, got %v", instructionID, res[0].Instruction.ID)
	}

	// Verify default sorting was applied (latest desc)
	if len(repo.calledSorts) != 1 {
		t.Fatalf("expected 1 sort clause, got %d", len(repo.calledSorts))
	}
	if repo.calledSorts[0].By != repository.PaymentMethodSortLatest || repo.calledSorts[0].Direction != query.SortDesc {
		t.Errorf("expected default sort to be latest desc, got %v", repo.calledSorts)
	}
}

func TestListPaymentMethods_CustomSort(t *testing.T) {
	ctx := context.Background()

	repo := &mockListPaymentMethodRepo{}
	usecase := NewListPaymentMethodUsecase(repo, &mockExecutor{})

	_, err := usecase.ListAll(ctx, ListPaymentMethodInput{
		Sort: "name:asc,code:desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.calledSorts) != 2 {
		t.Fatalf("expected 2 sort clauses, got %d", len(repo.calledSorts))
	}

	if repo.calledSorts[0].By != repository.PaymentMethodSortName || repo.calledSorts[0].Direction != query.SortAsc {
		t.Errorf("expected first sort to be name asc, got %v", repo.calledSorts[0])
	}
	if repo.calledSorts[1].By != repository.PaymentMethodSortCode || repo.calledSorts[1].Direction != query.SortDesc {
		t.Errorf("expected second sort to be code desc, got %v", repo.calledSorts[1])
	}
}
