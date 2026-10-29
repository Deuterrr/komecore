package paymentpersistence

import (
	"context"
	"strings"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type querySpyExecutor struct {
	lastSQL  string
	lastArgs []any
}

func (s *querySpyExecutor) Exec(_ context.Context, sql string, args ...any) (transaction.Result, error) {
	s.lastSQL = sql
	s.lastArgs = args
	return nil, nil
}

func (s *querySpyExecutor) Query(_ context.Context, sql string, args ...any) (transaction.Rows, error) {
	s.lastSQL = sql
	s.lastArgs = args
	return &emptyRows{}, nil
}

func (s *querySpyExecutor) QueryRow(_ context.Context, sql string, args ...any) transaction.Row {
	s.lastSQL = sql
	s.lastArgs = args
	return &emptyRow{}
}

type emptyRows struct{}

func (r *emptyRows) Close()              {}
func (r *emptyRows) Next() bool          { return false }
func (r *emptyRows) Scan(_ ...any) error { return nil }
func (r *emptyRows) Err() error          { return nil }

type emptyRow struct{}

func (r *emptyRow) Scan(_ ...any) error {
	return pgx.ErrNoRows
}

func TestPaymentEventRepository_SQLSyntax(t *testing.T) {
	repo := NewEventRepository()
	spy := &querySpyExecutor{}
	ctx := context.Background()

	// 1. Verify GetByID projection has commas separating all columns
	_, _ = repo.GetByID(ctx, spy, uuid.New())
	expectedColumns := []string{"id,", "payment_id,", "event_name,", "payload,", "created_at"}
	for _, col := range expectedColumns {
		if !strings.Contains(spy.lastSQL, col) {
			t.Errorf("expected GetByID SQL to contain %q with comma separator, got:\n%s", col, spy.lastSQL)
		}
	}

	// 2. Verify ListByPaymentID projection has commas separating all columns
	_, _ = repo.ListByPaymentID(ctx, spy, uuid.New())
	for _, col := range expectedColumns {
		if !strings.Contains(spy.lastSQL, col) {
			t.Errorf("expected ListByPaymentID SQL to contain %q with comma separator, got:\n%s", col, spy.lastSQL)
		}
	}
}

func TestPaymentRepository_ListPendingGateway_Query(t *testing.T) {
	repo := New()
	spy := &querySpyExecutor{}
	ctx := context.Background()

	_, _ = repo.ListPendingGateway(ctx, spy, time.Now())

	if strings.Contains(spy.lastSQL, "provider = 'gateway'") {
		t.Errorf("ListPendingGateway should not restrict to literal 'gateway' provider, got:\n%s", spy.lastSQL)
	}

	if !strings.Contains(spy.lastSQL, "provider_order_id IS NOT NULL") {
		t.Errorf("ListPendingGateway must check provider_order_id IS NOT NULL, got:\n%s", spy.lastSQL)
	}
}

func TestPaymentMethodRepository_SoftDeleteAndConflict_Query(t *testing.T) {
	repo := NewMethodRepository()
	spy := &querySpyExecutor{}
	ctx := context.Background()

	// 1. FindByName must filter out soft-deleted payment methods
	_, _ = repo.FindByName(ctx, spy, "GoPay")
	if !strings.Contains(spy.lastSQL, "pm.deleted_at IS NULL") {
		t.Errorf("FindByName must filter with pm.deleted_at IS NULL, got:\n%s", spy.lastSQL)
	}

	// 2. GetByID must filter out soft-deleted payment methods
	_, _ = repo.GetByID(ctx, spy, uuid.New())
	if !strings.Contains(spy.lastSQL, "pm.deleted_at IS NULL") {
		t.Errorf("GetByID must filter with pm.deleted_at IS NULL, got:\n%s", spy.lastSQL)
	}

	// 3. Save must conflict on (code, provider, type) and revive soft-deleted rows
	_ = repo.Save(ctx, spy, payment.PaymentMethod{
		ID:       uuid.New(),
		Name:     "QRIS",
		Code:     "qris",
		Provider: "midtrans",
		Type:     payment.TypeQRCode,
	})
	if !strings.Contains(spy.lastSQL, "ON CONFLICT (code, provider, type)") {
		t.Errorf("Save must target ON CONFLICT (code, provider, type), got:\n%s", spy.lastSQL)
	}
	if !strings.Contains(spy.lastSQL, "deleted_at = NULL") {
		t.Errorf("Save must reset deleted_at = NULL on conflict update, got:\n%s", spy.lastSQL)
	}
}
