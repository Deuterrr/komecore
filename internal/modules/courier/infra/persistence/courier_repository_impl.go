package persistence

import (
	"context"
	"fmt"
	"strings"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/courier/repository"
)

type courierRepositoryImpl struct{}

func NewCourierRepositoryImpl() repository.CourierRepository {
	return &courierRepositoryImpl{}
}

func (r *courierRepositoryImpl) ListAll(
	ctx context.Context,
	exec transaction.Executor,
) ([]string, error) {
	query := `
		SELECT
			code
		FROM
			couriers
		WHERE
			is_active = true
	`

	rows, err := exec.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query load couriers failed: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("scan courier failed: %w", err)
		}
		result = append(result, code)
	}

	return result, nil
}

func (r *courierRepositoryImpl) GetActiveCodes(
	ctx context.Context,
	exec transaction.Executor,
	codes []string,
) ([]string, error) {
	query := `
		SELECT code
		FROM couriers
		WHERE code = ANY($1)
		AND is_active = true
	`

	rows, err := exec.Query(ctx, query, codes)
	if err != nil {
		return nil, fmt.Errorf("query active couriers failed: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("scan courier code failed: %w", err)
		}
		result = append(result, code)
	}

	return result, nil
}

func (r *courierRepositoryImpl) ValidateCouriers(
	ctx context.Context,
	exec transaction.Executor,
	codes []string,
) ([]string, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("courier codes cannot be empty")
	}

	normalized := r.normalizeCodes(codes)
	validCodes, err := r.GetActiveCodes(ctx, exec, normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve couriers: %w", err)
	}
	if len(validCodes) != len(normalized) {
		return nil, fmt.Errorf("one or more courier codes are invalid")
	}

	return validCodes, nil
}

func (r *courierRepositoryImpl) normalizeCodes(codes []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(codes))

	for _, code := range codes {
		code = strings.ToLower(strings.TrimSpace(code))

		if code == "" {
			continue
		}

		if _, exists := seen[code]; !exists {
			result = append(result, code)
			seen[code] = struct{}{}
		}
	}

	return result
}
