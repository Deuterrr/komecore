package courier

import (
	"context"

	transaction "komecore/internal/infra/transactor"
)

// Repository defines persistence operations for shipping carriers, courier provider codes,
// and third-party logistics routing validation.
type Repository interface {
	// Queries all supported logistics provider codes (e.g. jne, jnt, sicepat) available for delivery estimation.
	ListAll(ctx context.Context, exec transaction.Executor) ([]string, error)

	// Filters a slice of courier codes against enabled carriers, discarding decommissioned providers.
	GetActiveCodes(ctx context.Context, exec transaction.Executor, codes []string) ([]string, error)

	// Validates whether requested logistics carrier codes are active and supported for merchant dispatch.
	ValidateCouriers(ctx context.Context, exec transaction.Executor, codes []string) ([]string, error)
}
