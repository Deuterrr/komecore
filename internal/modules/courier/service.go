package courier

import (
	"context"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
)

type Service struct {
	exec        transaction.Executor
	courierRepo Repository
}

func NewService(
	exec transaction.Executor,
	courierRepo Repository,
) *Service {
	return &Service{
		exec:        exec,
		courierRepo: courierRepo,
	}
}

func (s *Service) ListAllCouriers(ctx context.Context) ([]string, error) {
	codes, err := s.courierRepo.ListAll(ctx, s.exec)
	if err != nil {
		return nil, fmt.Errorf("failed to load couriers: %w", err)
	}
	if len(codes) == 0 {
		return nil, apperror.NewNotFound("no courier service available at the moment")
	}

	return codes, nil
}
