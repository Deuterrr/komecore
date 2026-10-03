package usecase_test

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	authenDomain "komecore/internal/modules/auth/domain"
	authenRepo "komecore/internal/modules/auth/repository"
	staffDomain "komecore/internal/modules/staff/domain"
	staffRepo "komecore/internal/modules/staff/repository"
	userDomain "komecore/internal/modules/user/domain"
	userRepo "komecore/internal/modules/user/repository"

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
	userRepo.UserRepository
	user             *userDomain.User
	getByIDError     error
	saveProfileError error
	savedProps       *userRepo.SaveProfileProps
	getByIDCalls     int
	saveProfileCalls int
}

func (m *mockUserRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*userDomain.User, error) {
	m.getByIDCalls++
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	return m.user, nil
}

func (m *mockUserRepo) SaveProfile(ctx context.Context, exec transaction.Executor, props userRepo.SaveProfileProps) error {
	m.saveProfileCalls++
	m.savedProps = &props
	return m.saveProfileError
}

type mockAccountRepo struct {
	authenRepo.AccountRepository
	account          *authenDomain.Account
	getByUserIDErr   error
	getByUserIDCalls int
}

func (m *mockAccountRepo) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*authenDomain.Account, error) {
	m.getByUserIDCalls++
	if m.getByUserIDErr != nil {
		return nil, m.getByUserIDErr
	}
	return m.account, nil
}

type mockStaffRepo struct {
	staffRepo.StaffRepository
	profile         *staffDomain.StaffProfile
	getProfileErr   error
	getProfileCalls int
}

func (m *mockStaffRepo) GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*staffDomain.StaffProfile, error) {
	m.getProfileCalls++
	if m.getProfileErr != nil {
		return nil, m.getProfileErr
	}
	return m.profile, nil
}

type mockSessionRepo struct {
	authenRepo.SessionRepository
	session      *authenDomain.Session
	getByIDErr   error
	getByIDCalls int
}

func (m *mockSessionRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*authenDomain.Session, error) {
	m.getByIDCalls++
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	return m.session, nil
}
