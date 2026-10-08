package paymentpersistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PaymentInstructionRepository struct{}

func NewPaymentInstructionRepository() *PaymentInstructionRepository {
	return &PaymentInstructionRepository{}
}

func (r *PaymentInstructionRepository) GetByPaymentMethodID(
	ctx context.Context,
	exec transaction.Executor,
	methodID uuid.UUID,
) (*paymentdomain.PaymentInstruction, error) {
	query := `
		SELECT
			id,
			payment_method_id,
			content,
			created_at
		FROM payment_instructions
		WHERE payment_method_id = $1
		LIMIT 1
	`

	var inst paymentdomain.PaymentInstruction
	err := exec.QueryRow(ctx, query, methodID).Scan(
		&inst.ID,
		&inst.PaymentMethodID,
		&inst.Content,
		&inst.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query payment instruction by payment_method_id failed: %w", err)
	}

	return &inst, nil
}

func (r *PaymentInstructionRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	instruction paymentdomain.PaymentInstruction,
) error {
	query := `
		INSERT INTO payment_instructions (
			id,
			payment_method_id,
			content,
			created_at
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (payment_method_id) DO UPDATE SET
			content = EXCLUDED.content
	`

	_, err := exec.Exec(ctx, query,
		instruction.ID,
		instruction.PaymentMethodID,
		instruction.Content,
		instruction.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save payment instruction failed: %w", err)
	}

	return nil
}
