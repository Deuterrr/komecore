package usecase

import (
	"context"
	"errors"

	transaction "komecore/internal/infra/transactor"
	staffDomain "komecore/internal/modules/staff/domain"
	staffRepo "komecore/internal/modules/staff/repository"
	userDomain "komecore/internal/modules/user/domain"
	userRepo "komecore/internal/modules/user/repository"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type mockExecutor struct{}

func (m *mockExecutor) Exec(ctx context.Context, query string, args ...any) (transaction.Result, error) {
	return transaction.NewResult(1), nil
}

func (m *mockExecutor) Query(ctx context.Context, query string, args ...any) (transaction.Rows, error) {
	return nil, nil
}

func (m *mockExecutor) QueryRow(ctx context.Context, query string, args ...any) transaction.Row {
	return &mockRow{}
}

type mockRow struct{}

func (r *mockRow) Scan(dest ...any) error {
	return nil
}

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(exec transaction.Executor) error) error {
	return fn(&mockExecutor{})
}

type mockAuditLogger struct {
	events []applogger.AuditEvent
}

func (m *mockAuditLogger) Log(ctx context.Context, event applogger.AuditEvent) {
	m.events = append(m.events, event)
}

type mockStaffRepo struct {
	staff        *staffDomain.Staff
	getByIDError error
	updateError  error
	deleteError  error
	createCalls  int
	updateCalls  int
	deleteCalls  int
}

func (m *mockStaffRepo) Create(ctx context.Context, exec transaction.Executor, staff staffDomain.Staff) error {
	m.createCalls++
	return nil
}

func (m *mockStaffRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*staffDomain.Staff, error) {
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	return m.staff, nil
}

func (m *mockStaffRepo) FindStaff(ctx context.Context, exec transaction.Executor, params staffRepo.FindStaffParams) ([]staffDomain.StaffProfile, int, error) {
	return nil, 0, nil
}

func (m *mockStaffRepo) Update(ctx context.Context, exec transaction.Executor, staffID uuid.UUID, name string, logoUrl *string, bannerUrl *string) error {
	m.updateCalls++
	return m.updateError
}

func (m *mockStaffRepo) Delete(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error {
	m.deleteCalls++
	return m.deleteError
}

var _ StaffRepository = (*mockStaffRepo)(nil)

type mockStaffMembershipRepo struct {
	membership         *staffDomain.StaffMembership
	targetMembership   *staffDomain.StaffMembership
	roles              []staffDomain.Role
	accounts           []staffDomain.StaffAccountMember
	getMemError        error
	getRolesError      error
	listAccsError      error
	deleteByStaffErr   error
	deleteByAccErr     error
	saveCalls          int
	deleteByAccCalls   int
	deleteByStaffCalls int
}

func (m *mockStaffMembershipRepo) GetByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) (*staffDomain.StaffMembership, error) {
	if m.getMemError != nil {
		return nil, m.getMemError
	}
	if m.targetMembership != nil && accountID == m.targetMembership.AccountID {
		return m.targetMembership, nil
	}
	return m.membership, nil
}

func (m *mockStaffMembershipRepo) ListRolesByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) ([]staffDomain.Role, error) {
	if m.getRolesError != nil {
		return nil, m.getRolesError
	}
	return m.roles, nil
}

func (m *mockStaffMembershipRepo) Save(ctx context.Context, exec transaction.Executor, membership staffDomain.StaffMembership) error {
	m.saveCalls++
	return nil
}

func (m *mockStaffMembershipRepo) ListAccountsByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) ([]staffDomain.StaffAccountMember, error) {
	if m.listAccsError != nil {
		return nil, m.listAccsError
	}
	return m.accounts, nil
}

func (m *mockStaffMembershipRepo) DeleteByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) error {
	m.deleteByAccCalls++
	return m.deleteByAccErr
}

func (m *mockStaffMembershipRepo) DeleteByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error {
	m.deleteByStaffCalls++
	return m.deleteByStaffErr
}

var _ StaffMembershipRepository = (*mockStaffMembershipRepo)(nil)

type mockAccountRepo struct {
	account        *AccountInfo
	deleteCalls    int
	revokeCalls    int
	revokedUserIDs []uuid.UUID
	sessionErr     error
}

func (m *mockAccountRepo) GetByEmail(ctx context.Context, exec transaction.Executor, email string) (*AccountInfo, error) {
	return nil, nil
}

func (m *mockAccountRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*AccountInfo, error) {
	return m.account, nil
}

func (m *mockAccountRepo) GetByUserID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*AccountInfo, error) {
	return m.account, nil
}

func (m *mockAccountRepo) CreateStaffAccount(ctx context.Context, exec transaction.Executor, input CreateAccountInput) error {
	return nil
}

func (m *mockAccountRepo) DeleteByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error {
	m.deleteCalls++
	return nil
}

func (m *mockAccountRepo) RevokeSessionsByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error {
	if m.sessionErr != nil {
		return m.sessionErr
	}
	m.revokeCalls++
	m.revokedUserIDs = append(m.revokedUserIDs, userID)
	return nil
}

var _ AccountManager = (*mockAccountRepo)(nil)

type mockUserRepo struct {
	user        *userDomain.User
	createCalls int
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, exec transaction.Executor, username string) (*userDomain.User, error) {
	return m.user, nil
}

func (m *mockUserRepo) CreateUser(ctx context.Context, exec transaction.Executor, props userRepo.CreateUserProps) error {
	m.createCalls++
	return nil
}

var _ UserRepository = (*mockUserRepo)(nil)

type mockRoleRepo struct {
	role *staffDomain.Role
}

func (m *mockRoleRepo) GetByCode(ctx context.Context, exec transaction.Executor, code staffDomain.RoleCode) (*staffDomain.Role, error) {
	return m.role, nil
}

var _ RoleRepository = (*mockRoleRepo)(nil)

type mockPwHasher struct{}

func (m *mockPwHasher) Hash(password string) (string, error) {
	return "hashed_" + password, nil
}

func (m *mockPwHasher) Compare(hash, password string) error {
	return nil
}

var _ PasswordHasher = (*mockPwHasher)(nil)

type mockUserDeletionService struct {
	deletedUsers []uuid.UUID
	err          error
}

func (m *mockUserDeletionService) DeleteUserRecord(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.deletedUsers = append(m.deletedUsers, userID)
	return nil
}

var _ UserDeletionService = (*mockUserDeletionService)(nil)

var errMock = errors.New("mock error")
