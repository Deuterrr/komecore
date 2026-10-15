package courierusecase

import (
	"context"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/courier"
)

type CourierService struct {
	exec        transaction.Executor
	courierRepo courier.Repository
}

func NewCourierService(
	exec transaction.Executor,
	courierRepo courier.Repository,
) *CourierService {
	return &CourierService{
		exec:        exec,
		courierRepo: courierRepo,
	}
}

func (s *CourierService) ListAllCouriers(ctx context.Context) ([]string, error) {
	codes, err := s.courierRepo.ListAll(ctx, s.exec)
	if err != nil {
		return nil, fmt.Errorf("failed to load couriers: %w", err)
	}
	if len(codes) == 0 {
		return nil, apperror.NewNotFound("no courier service available at the moment")
	}

	return codes, nil
}
