package userusecase_test

import (
	"context"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/user"
	"komecore/internal/modules/user/userusecase"

	"github.com/google/uuid"
)

type mockExecutor struct {
	transaction.Executor
}

type mockTransactor struct {
	transaction.Transactor
	err error
}

func (m *mockTransactor) WithinTransaction(
	ctx context.Context,
	fn func(exec transaction.Executor) error,
) error {
	if m.err != nil {
		return m.err
	}
	return fn(&mockExecutor{})
}

type mockUserRepo struct {
	user             *user.User
	getByIDError     error
	saveProfileError error
	savedProps       *user.SaveProfileProps
	getByIDCalls     int
	saveProfileCalls int
}

func (m *mockUserRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*user.User, error) {
	m.getByIDCalls++
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	return m.user, nil
}

func (m *mockUserRepo) SaveProfile(ctx context.Context, exec transaction.Executor, props user.SaveProfileProps) error {
	m.saveProfileCalls++
	m.savedProps = &props
	return m.saveProfileError
}

type mockAccountRepo struct {
	account          *userusecase.UserAccount
	getByUserIDErr   error
	getByUserIDCalls int
}

func (m *mockAccountRepo) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*userusecase.UserAccount, error) {
	m.getByUserIDCalls++
	if m.getByUserIDErr != nil {
		return nil, m.getByUserIDErr
	}
	return m.account, nil
}

type mockStaffRepo struct {
	profile         *user.StaffProfile
	getProfileErr   error
	getProfileCalls int
}

func (m *mockStaffRepo) GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*user.StaffProfile, error) {
	m.getProfileCalls++
	if m.getProfileErr != nil {
		return nil, m.getProfileErr
	}
	return m.profile, nil
}

type mockSessionRepo struct {
	lastActivity *time.Time
	getByIDErr   error
	getByIDCalls int
}

func (m *mockSessionRepo) GetLastActivity(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*time.Time, error) {
	m.getByIDCalls++
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	return m.lastActivity, nil
}
