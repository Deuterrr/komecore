package courierusecase_test

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/courier/courierrepo"
	"komecore/internal/modules/courier/courierusecase"

	"github.com/stretchr/testify/assert"
)

type mockCourierRepo struct {
	courierrepo.CourierRepository
	couriers []string
	err      error
	calls    int
}

func (m *mockCourierRepo) ListAll(ctx context.Context, exec transaction.Executor) ([]string, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.couriers, nil
}

func TestListCouriers_Execute(t *testing.T) {
	t.Run("success returns courier codes", func(t *testing.T) {
		repo := &mockCourierRepo{
			couriers: []string{"jne", "jnt"},
		}

		svc := courierusecase.NewCourierService(nil, repo)
		result, err := svc.ListAllCouriers(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, []string{"jne", "jnt"}, result)
		assert.Equal(t, 1, repo.calls)
	})

	t.Run("returns not found error when courier list is empty", func(t *testing.T) {
		repo := &mockCourierRepo{
			couriers: []string{},
		}

		svc := courierusecase.NewCourierService(nil, repo)
		result, err := svc.ListAllCouriers(context.Background())

		assert.Nil(t, result)
		assert.True(t, apperror.IsNotFound(err))
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		repo := &mockCourierRepo{
			err: errors.New("db connection failure"),
		}

		svc := courierusecase.NewCourierService(nil, repo)
		result, err := svc.ListAllCouriers(context.Background())

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
