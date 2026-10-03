package usecase

import (
	"context"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/repository"
	userDomain "komecore/internal/modules/user/domain"
	userRepo "komecore/internal/modules/user/repository"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

// ===========================================================================
// Mocks for AuthenticateOAuthUsecase Tests
// ===========================================================================

type mockExecutor struct {
	transaction.NoopExecutor
}

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(transaction.Executor) error) error {
	return fn(&mockExecutor{})
}

type mockAuditLogger struct{}

func (m *mockAuditLogger) Log(_ context.Context, _ applogger.AuditEvent) {}

type mockTokenHasher struct{}

func (m *mockTokenHasher) Hash(token string) string               { return token }
func (m *mockTokenHasher) Compare(hash string, token string) bool { return hash == token }

type mockTokenService struct{}

func (m *mockTokenService) Generate(params repository.GenerateTokenParams) (repository.GeneratedToken, error) {
	return repository.GeneratedToken{
		Token:     "mock-token-" + string(params.Type),
		ExpiresAt: time.Now().Add(params.Duration),
	}, nil
}
func (m *mockTokenService) Validate(token string) (*domain.TokenClaims, error) {
	return &domain.TokenClaims{UserID: uuid.New()}, nil
}

type mockSessionRepo struct{}

func (m *mockSessionRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepo) RevokeByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *mockSessionRepo) RevokeAllByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *mockSessionRepo) UpdateLastActivityByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *mockSessionRepo) Save(_ context.Context, _ transaction.Executor, _ domain.Session) error {
	return nil
}

type mockRefreshTokenRepo struct{}

func (m *mockRefreshTokenRepo) GetBySessionID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*domain.RefreshToken, error) {
	return nil, nil
}
func (m *mockRefreshTokenRepo) RevokeBySessionID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *mockRefreshTokenRepo) Save(_ context.Context, _ transaction.Executor, _ domain.RefreshToken) error {
	return nil
}

type mockCustomerRepo struct {
	customer *domain.Customer
}

func (m *mockCustomerRepo) Create(_ context.Context, _ transaction.Executor, _ domain.Customer) error {
	return nil
}
func (m *mockCustomerRepo) GetByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*domain.Customer, error) {
	return m.customer, nil
}
func (m *mockCustomerRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

// User Repo Mock
type mockUserRepo struct {
	user        *userDomain.User
	saveProfile func(props userRepo.SaveProfileProps) error
}

func (m *mockUserRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*userDomain.User, error) {
	return m.user, nil
}
func (m *mockUserRepo) GetByUsername(_ context.Context, _ transaction.Executor, _ string) (*userDomain.User, error) {
	return m.user, nil
}
func (m *mockUserRepo) CreateUser(_ context.Context, _ transaction.Executor, _ userRepo.CreateUserProps) error {
	return nil
}
func (m *mockUserRepo) SaveProfile(_ context.Context, _ transaction.Executor, props userRepo.SaveProfileProps) error {
	if m.saveProfile != nil {
		return m.saveProfile(props)
	}
	return nil
}
func (m *mockUserRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

// Account Repo Mock
type mockAccountRepo struct {
	account                *domain.Account
	activateCalls          int
	updatePasswordCalls    int
	lastPasswordUpdateHash string
	updateLastLoginAtCalls int
	lastLoginAtRecorded    time.Time
}

func (m *mockAccountRepo) GetByEmail(_ context.Context, _ transaction.Executor, _ string) (*domain.Account, error) {
	return m.account, nil
}
func (m *mockAccountRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*domain.Account, error) {
	return m.account, nil
}
func (m *mockAccountRepo) GetByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*domain.Account, error) {
	return m.account, nil
}
func (m *mockAccountRepo) ActivateByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	m.activateCalls++
	return nil
}
func (m *mockAccountRepo) Create(_ context.Context, _ transaction.Executor, _ domain.Account) error {
	return nil
}
func (m *mockAccountRepo) UpdatePasswordByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID, hash string) error {
	m.updatePasswordCalls++
	m.lastPasswordUpdateHash = hash
	return nil
}
func (m *mockAccountRepo) DeleteByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *mockAccountRepo) UpdateLastLoginAt(_ context.Context, _ transaction.Executor, _ uuid.UUID, lastLoginAt time.Time) error {
	m.updateLastLoginAtCalls++
	m.lastLoginAtRecorded = lastLoginAt
	if m.account != nil {
		m.account.LastLoginAt = &lastLoginAt
	}
	return nil
}

// OAuth connection mock
type mockOAuthRepo struct {
	connection           *domain.OAuthConnection
	createCalls          int
	updateLastLoginCalls int
}

func (m *mockOAuthRepo) GetByProviderAndSubject(_ context.Context, _ transaction.Executor, _ domain.OAuthProvider, _ string) (*domain.OAuthConnection, error) {
	return m.connection, nil
}
func (m *mockOAuthRepo) GetByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*domain.OAuthConnection, error) {
	return m.connection, nil
}
func (m *mockOAuthRepo) Create(_ context.Context, _ transaction.Executor, _ domain.OAuthConnection) error {
	m.createCalls++
	return nil
}
func (m *mockOAuthRepo) UpdateLastLogin(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time) error {
	m.updateLastLoginCalls++
	return nil
}
func (m *mockOAuthRepo) DeleteByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

// ===========================================================================
// Tests
// ===========================================================================

func TestAuthenticateOAuth_NewUserRegistration(t *testing.T) {
	ctx := context.Background()
	userRepoMock := &mockUserRepo{}
	accountRepoMock := &mockAccountRepo{}
	oauthRepoMock := &mockOAuthRepo{}

	uc := NewAuthenticateOAuthUsecase(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		oauthRepoMock,
		userRepoMock,
		&mockCustomerRepo{},
		&mockTokenHasher{},
		&mockTokenService{},
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		&mockAuditLogger{},
	)

	input := AuthenticateOAuthParams{
		Provider:  domain.OAuthProviderGoogle,
		Subject:   "google-sub-123",
		Email:     "newuser@gmail.com",
		Name:      "New User",
		AvatarURL: nil,
	}

	result, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result not to be nil")
	}

	if oauthRepoMock.createCalls != 1 {
		t.Errorf("expected 1 Create call on oauthRepo, got %d", oauthRepoMock.createCalls)
	}
}

func TestAuthenticateOAuth_ExistingLinkedAccount_UpdatesLastLoginAndDoesNotSyncProfile(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	existingConn := &domain.OAuthConnection{
		ID:       uuid.New(),
		UserID:   userID,
		Provider: domain.OAuthProviderGoogle,
		Subject:  "google-sub-123",
	}

	oauthRepoMock := &mockOAuthRepo{connection: existingConn}
	accountRepoMock := &mockAccountRepo{
		account: &domain.Account{
			UserID:   userID,
			Email:    "existing@gmail.com",
			Password: "hashedpassword123",
			Status:   domain.AccountActive,
		},
	}

	var saveProfileCalled bool
	userRepoMock := &mockUserRepo{
		user: &userDomain.User{ID: userID, Name: "Old Name"},
		saveProfile: func(props userRepo.SaveProfileProps) error {
			saveProfileCalled = true
			return nil
		},
	}

	uc := NewAuthenticateOAuthUsecase(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		oauthRepoMock,
		userRepoMock,
		&mockCustomerRepo{},
		&mockTokenHasher{},
		&mockTokenService{},
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		&mockAuditLogger{},
	)

	avatar := "http://example.com/new-pic.jpg"
	input := AuthenticateOAuthParams{
		Provider:  domain.OAuthProviderGoogle,
		Subject:   "google-sub-123",
		Email:     "existing@gmail.com",
		Name:      "New Name",
		AvatarURL: &avatar,
	}

	_, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if oauthRepoMock.updateLastLoginCalls != 1 {
		t.Errorf("expected UpdateLastLogin call, got %d", oauthRepoMock.updateLastLoginCalls)
	}
	if saveProfileCalled {
		t.Error("expected SaveProfile NOT to be called")
	}

	if accountRepoMock.updatePasswordCalls != 0 {
		t.Error("expected password NOT to be modified")
	}
}

func TestAuthenticateOAuth_ExistingLocalAccount_LinksAndDoesNotSyncProfile(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountRepoMock := &mockAccountRepo{
		account: &domain.Account{
			UserID:   userID,
			Email:    "localuser@gmail.com",
			Password: "hashedlocalpassword",
			Status:   domain.AccountPending,
		},
	}
	oauthRepoMock := &mockOAuthRepo{connection: nil} // No OAuth connection initially

	var saveProfileCalled bool
	userRepoMock := &mockUserRepo{
		user: &userDomain.User{ID: userID, Name: "Local Name"},
		saveProfile: func(props userRepo.SaveProfileProps) error {
			saveProfileCalled = true
			return nil
		},
	}

	uc := NewAuthenticateOAuthUsecase(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		oauthRepoMock,
		userRepoMock,
		&mockCustomerRepo{},
		&mockTokenHasher{},
		&mockTokenService{},
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		&mockAuditLogger{},
	)

	input := AuthenticateOAuthParams{
		Provider:  domain.OAuthProviderGoogle,
		Subject:   "google-sub-123",
		Email:     "localuser@gmail.com",
		Name:      "Google Name",
		AvatarURL: nil,
	}

	_, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if accountRepoMock.activateCalls != 1 {
		t.Errorf("expected ActivateByUserID to be called once for pending account, got %d", accountRepoMock.activateCalls)
	}
	if oauthRepoMock.createCalls != 1 {
		t.Errorf("expected 1 Create call on oauthRepo, got %d", oauthRepoMock.createCalls)
	}
	if saveProfileCalled {
		t.Error("expected SaveProfile NOT to be called during account linking")
	}
	if accountRepoMock.updatePasswordCalls != 0 {
		t.Error("expected password NOT to be modified during account linking")
	}
}

func TestAuthenticateOAuth_ExistingLocalAccount_AlreadyLinkedToOtherProvider(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountRepoMock := &mockAccountRepo{
		account: &domain.Account{
			UserID: userID,
			Email:  "user@gmail.com",
		},
	}

	// This user already has an OAuth connection (e.g., github)
	oauthRepoMock := &mockOAuthRepo{
		connection: &domain.OAuthConnection{
			ID:       uuid.New(),
			UserID:   userID,
			Provider: "github",
		},
	}

	uc := NewAuthenticateOAuthUsecase(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		oauthRepoMock,
		&mockUserRepo{},
		&mockCustomerRepo{},
		&mockTokenHasher{},
		&mockTokenService{},
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		&mockAuditLogger{},
	)

	input := AuthenticateOAuthParams{
		Provider: domain.OAuthProviderGoogle,
		Subject:  "google-sub-123",
		Email:    "user@gmail.com",
	}

	_, err := uc.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}
