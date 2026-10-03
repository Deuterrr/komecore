package usecase_test

import (
	"context"
	"errors"
	"testing"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/courier/repository"
	"komecore/internal/modules/courier/usecase"

	"github.com/stretchr/testify/assert"
)

type mockCourierRepo struct {
	repository.CourierRepository
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

		uc := usecase.NewListCouriersUsecase(nil, repo)
		result, err := uc.Execute(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, []string{"jne", "jnt"}, result)
		assert.Equal(t, 1, repo.calls)
	})

	t.Run("returns not found error when courier list is empty", func(t *testing.T) {
		repo := &mockCourierRepo{
			couriers: []string{},
		}

		uc := usecase.NewListCouriersUsecase(nil, repo)
		result, err := uc.Execute(context.Background())

		assert.Nil(t, result)
		assert.True(t, apperrors.IsNotFound(err))
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		repo := &mockCourierRepo{
			err: errors.New("db connection failure"),
		}

		uc := usecase.NewListCouriersUsecase(nil, repo)
		result, err := uc.Execute(context.Background())

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
