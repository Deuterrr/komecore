package courierrepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
)

type CourierRepository interface {
	ListAll(
		ctx context.Context,
		exec transaction.Executor,
	) ([]string, error)

	GetActiveCodes(
		ctx context.Context,
		exec transaction.Executor,
		codes []string,
	) ([]string, error)

	ValidateCouriers(
		ctx context.Context,
		exec transaction.Executor,
		codes []string,
	) ([]string, error)
}
