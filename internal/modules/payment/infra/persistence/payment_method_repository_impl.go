package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/domain"
	"komecore/internal/modules/payment/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type paymentMethodRepositoryImpl struct{}

func NewPaymentMethodRepository() repository.PaymentMethodRepository {
	return &paymentMethodRepositoryImpl{}
}

func (r *paymentMethodRepositoryImpl) Save(
	ctx context.Context,
	exec transaction.Executor,
	method domain.PaymentMethod,
) error {
	query := `
		INSERT INTO payment_methods (
			id,
			name,
			code,
			provider,
			type,
			is_active,
			description,
			fee_type,
			fee_amount,
			fee_rate,
			fee_max,
			created_at,
			updated_at
		) VALUES (
		 	$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			code = EXCLUDED.code,
			provider = EXCLUDED.provider,
			type = EXCLUDED.type,
			is_active = EXCLUDED.is_active,
			description = EXCLUDED.description,
			fee_type = EXCLUDED.fee_type,
			fee_amount = EXCLUDED.fee_amount,
			fee_rate = EXCLUDED.fee_rate,
			fee_max = EXCLUDED.fee_max,
			updated_at = NOW()
	`

	_, err := exec.Exec(ctx, query,
		method.ID,
		method.Name,
		method.Code,
		method.Provider,
		method.Type,
		method.IsActive,
		method.Description,
		method.FeeType,
		method.FeeFixed,
		method.FeePercentage,
		method.FeeMax,
		method.CreatedAt,
		method.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("save payment method failed: %w", err)
	}

	return nil
}

func (r *paymentMethodRepositoryImpl) FindByName(
	ctx context.Context,
	exec transaction.Executor,
	name string,
) (*domain.PaymentMethod, error) {
	query := `
		SELECT
			pm.id,
			pm.name,
			pm.code,
			pm.provider,
			pm.type,
			pm.is_active,
			pm.description,
			pm.fee_type,
			pm.fee_amount::BIGINT AS fee_amount,
			pm.fee_rate,
			pm.fee_max::BIGINT AS fee_max,
			pm.created_at,
			pm.updated_at,
			pi.id AS pi_id,
			pi.content AS pi_content,
			pi.created_at AS pi_created_at
		FROM payment_methods pm
		LEFT JOIN payment_instructions pi ON pm.id = pi.payment_method_id
		WHERE
			pm.name LIKE $1 || '%'
		LIMIT 1
	`

	var method domain.PaymentMethod
	var piID *uuid.UUID
	var piContent *string
	var piCreatedAt *time.Time

	err := exec.QueryRow(ctx, query, name).Scan(
		&method.ID,
		&method.Name,
		&method.Code,
		&method.Provider,
		&method.Type,
		&method.IsActive,
		&method.Description,
		&method.FeeType,
		&method.FeeFixed,
		&method.FeePercentage,
		&method.FeeMax,
		&method.CreatedAt,
		&method.UpdatedAt,
		&piID,
		&piContent,
		&piCreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query payment method by name failed: %w", err)
	}

	if piID != nil {
		var content string
		if piContent != nil {
			content = *piContent
		}

		var createdAt time.Time
		if piCreatedAt != nil {
			createdAt = *piCreatedAt
		}

		method.Instruction = &domain.PaymentInstruction{
			ID:              *piID,
			PaymentMethodID: method.ID,
			Content:         content,
			CreatedAt:       createdAt,
		}
	}

	return &method, nil
}

func (r *paymentMethodRepositoryImpl) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	paymentID uuid.UUID,
) (*domain.PaymentMethod, error) {
	query := `
		SELECT
			pm.id,
			pm.name,
			pm.code,
			pm.provider,
			pm.type,
			pm.is_active,
			pm.description,
			pm.fee_type,
			pm.fee_amount::BIGINT AS fee_amount,
			pm.fee_rate,
			pm.fee_max::BIGINT AS fee_max,
			pm.created_at,
			pm.updated_at,
			pi.id AS pi_id,
			pi.content AS pi_content,
			pi.created_at AS pi_created_at
		FROM payment_methods pm
		LEFT JOIN payment_instructions pi ON pm.id = pi.payment_method_id
		WHERE
			pm.id = $1
		LIMIT 1
	`

	var method domain.PaymentMethod
	var piID *uuid.UUID
	var piContent *string
	var piCreatedAt *time.Time

	err := exec.QueryRow(ctx, query, paymentID).Scan(
		&method.ID,
		&method.Name,
		&method.Code,
		&method.Provider,
		&method.Type,
		&method.IsActive,
		&method.Description,
		&method.FeeType,
		&method.FeeFixed,
		&method.FeePercentage,
		&method.FeeMax,
		&method.CreatedAt,
		&method.UpdatedAt,
		&piID,
		&piContent,
		&piCreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query payment method by id failed: %w", err)
	}

	if piID != nil {
		var content string
		if piContent != nil {
			content = *piContent
		}

		var createdAt time.Time
		if piCreatedAt != nil {
			createdAt = *piCreatedAt
		}

		method.Instruction = &domain.PaymentInstruction{
			ID:              *piID,
			PaymentMethodID: method.ID,
			Content:         content,
			CreatedAt:       createdAt,
		}
	}

	return &method, nil
}

func (r *paymentMethodRepositoryImpl) ListAll(
	ctx context.Context,
	exec transaction.Executor,
	sorts query.Sorts,
) ([]domain.PaymentMethod, error) {
	var pmSortKeys = map[query.SortKey]string{
		repository.PaymentMethodSortLatest: "pm.created_at",
		repository.PaymentMethodSortName:   "pm.name",
		repository.PaymentMethodSortCode:   "pm.code",
		repository.PaymentMethodSortType:   "pm.type",
	}

	var sortClauses []string
	for _, sort := range sorts {
		colName, exists := pmSortKeys[sort.By]
		if !exists {
			continue
		}

		dir := "DESC"
		if sort.Direction == query.SortAsc {
			dir = "ASC"
		}

		sortClauses = append(
			sortClauses,
			fmt.Sprintf("%s %s", colName, dir),
		)
	}

	orderBy := "ORDER BY pm.created_at DESC"
	if len(sortClauses) > 0 {
		orderBy = "ORDER BY " + strings.Join(sortClauses, ", ")
	}

	rawQuery := fmt.Sprintf(`
		SELECT
			pm.id,
			pm.name,
			pm.code,
			pm.provider,
			pm.type,
			pm.is_active,
			pm.description,
			pm.fee_type,
			pm.fee_amount::BIGINT AS fee_amount,
			pm.fee_rate,
			pm.fee_max::BIGINT AS fee_max,
			pm.created_at,
			pm.updated_at,
			pi.id AS pi_id,
			pi.content AS pi_content,
			pi.created_at AS pi_created_at
		FROM payment_methods pm
		LEFT JOIN payment_instructions pi ON pm.id = pi.payment_method_id
		WHERE pm.deleted_at IS NULL
		%s
	`, orderBy)

	rows, err := exec.Query(ctx, rawQuery)
	if err != nil {
		return nil, fmt.Errorf("query payment methods failed: %w", err)
	}
	defer rows.Close()

	var result []domain.PaymentMethod
	for rows.Next() {
		var row domain.PaymentMethod
		var piID *uuid.UUID
		var piContent *string
		var piCreatedAt *time.Time

		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Code,
			&row.Provider,
			&row.Type,
			&row.IsActive,
			&row.Description,
			&row.FeeType,
			&row.FeeFixed,
			&row.FeePercentage,
			&row.FeeMax,
			&row.CreatedAt,
			&row.UpdatedAt,
			&piID,
			&piContent,
			&piCreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("mapping payment method model to domain failed: %w", err)
		}

		if piID != nil {
			var content string
			if piContent != nil {
				content = *piContent
			}
			var createdAt time.Time
			if piCreatedAt != nil {
				createdAt = *piCreatedAt
			}
			row.Instruction = &domain.PaymentInstruction{
				ID:              *piID,
				PaymentMethodID: row.ID,
				Content:         content,
				CreatedAt:       createdAt,
			}
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment methods failed: %w", err)
	}

	return result, nil
}
