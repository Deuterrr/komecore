package usecase

import (
	"context"
	"errors"
	"testing"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	userDomain "komecore/internal/modules/user/domain"
	userRepo "komecore/internal/modules/user/repository"
	mailer "komecore/pkg/mailer"

	"github.com/google/uuid"
)

// ===========================================================================
// Mocks for RegisterCustomerUsecase Tests
// ===========================================================================

type mockMailer struct {
	sentCalls int
}

func (m *mockMailer) Send(_ mailer.SendInput) error {
	m.sentCalls++
	return nil
}

type mockOTPGen struct {
	otp string
}

func (m *mockOTPGen) Generate() (string, error) {
	return m.otp, nil
}

type mockChallengeRepo struct {
	challenge *domain.VerificationChallenge
	saveCalls int
}

func (m *mockChallengeRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*domain.VerificationChallenge, error) {
	return m.challenge, nil
}
func (m *mockChallengeRepo) Save(_ context.Context, _ transaction.Executor, c domain.VerificationChallenge) error {
	m.challenge = &c
	m.saveCalls++
	return nil
}

type mockHasher struct{}

func (m *mockHasher) Hash(password string) (string, error) {
	return "hashed-" + password, nil
}
func (m *mockHasher) Compare(hash string, password string) error {
	if hash == "hashed-"+password {
		return nil
	}
	return errors.New("mismatch")
}

// ===========================================================================
// Tests
// ===========================================================================

func TestRegisterCustomer_HappyPath_NewRegistration(t *testing.T) {
	ctx := context.Background()
	userRepoMock := &mockUserRepo{user: nil}
	accountRepoMock := &mockAccountRepo{account: nil}
	challengeRepoMock := &mockChallengeRepo{}
	mailerMock := &mockMailer{}
	otpGenMock := &mockOTPGen{otp: "123456"}

	uc := NewRegisterCustomerUsecase(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		&mockHasher{},
		userRepoMock,
		&mockCustomerRepo{},
		challengeRepoMock,
		otpGenMock,
		mailerMock,
		&mockAuditLogger{},
	)

	params := RegisterCustomerParams{
		Name:     "Test Customer",
		Username: "testcustomer",
		Email:    "customer@gmail.com",
		Password: "password123",
	}

	challengeID, err := uc.Execute(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if challengeID == nil {
		t.Fatal("expected challenge ID not to be nil")
	}

	if challengeRepoMock.saveCalls != 1 {
		t.Errorf("expected 1 Save call on challengeRepo, got %d", challengeRepoMock.saveCalls)
	}
	if mailerMock.sentCalls != 1 {
		t.Errorf("expected 1 email to be sent, got %d", mailerMock.sentCalls)
	}
}

func TestRegisterCustomer_LinkLocalToGoogleAccount_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	// Existing Google account with same email, but NO password (empty string)
	existingAcc := &domain.Account{
		ID:       uuid.New(),
		UserID:   userID,
		Email:    "googleuser@gmail.com",
		Password: "", // Empty means Google-first / OAuth-only
		Status:   domain.AccountActive,
	}

	accountRepoMock := &mockAccountRepo{account: existingAcc}
	// The username "newusername" is currently not taken by anyone
	userRepoMock := &mockUserRepo{user: nil}
	challengeRepoMock := &mockChallengeRepo{}
	mailerMock := &mockMailer{}
	otpGenMock := &mockOTPGen{otp: "654321"}

	var savedUsername string
	userRepoMock.saveProfile = func(props userRepo.SaveProfileProps) error {
		if props.UserID != userID {
			t.Errorf("expected UserID %v, got %v", userID, props.UserID)
		}
		if props.Username != nil {
			savedUsername = *props.Username
		}
		return nil
	}

	uc := NewRegisterCustomerUsecase(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		&mockHasher{},
		userRepoMock,
		&mockCustomerRepo{},
		challengeRepoMock,
		otpGenMock,
		mailerMock,
		&mockAuditLogger{},
	)

	params := RegisterCustomerParams{
		Name:     "Linked Name",
		Username: "newusername",
		Email:    "googleuser@gmail.com",
		Password: "newlocalpassword",
	}

	challengeID, err := uc.Execute(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if challengeID == nil {
		t.Fatal("expected challenge ID to be returned")
	}

	if accountRepoMock.updatePasswordCalls != 1 {
		t.Errorf("expected UpdatePasswordByUserID to be called once, got %d", accountRepoMock.updatePasswordCalls)
	}
	if accountRepoMock.lastPasswordUpdateHash != "hashed-newlocalpassword" {
		t.Errorf("expected password hash update to match, got %s", accountRepoMock.lastPasswordUpdateHash)
	}
	if savedUsername != "newusername" {
		t.Errorf("expected username in user profile update to be set to 'newusername', got '%s'", savedUsername)
	}
}

func TestRegisterCustomer_LinkLocalToGoogleAccount_UsernameConflict(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	existingAcc := &domain.Account{
		ID:       uuid.New(),
		UserID:   userID,
		Email:    "googleuser@gmail.com",
		Password: "", // Empty means Google-first / OAuth-only
	}

	accountRepoMock := &mockAccountRepo{account: existingAcc}
	// The username "taken_username" is already owned by ANOTHER user (with different ID)
	userRepoMock := &mockUserRepo{
		user: &userDomain.User{ID: uuid.New(), Username: "taken_username"},
	}

	uc := NewRegisterCustomerUsecase(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		&mockHasher{},
		userRepoMock,
		&mockCustomerRepo{},
		&mockChallengeRepo{},
		&mockOTPGen{otp: "111111"},
		&mockMailer{},
		&mockAuditLogger{},
	)

	params := RegisterCustomerParams{
		Name:     "Linked Name",
		Username: "taken_username",
		Email:    "googleuser@gmail.com",
		Password: "newlocalpassword",
	}

	_, err := uc.Execute(ctx, params)
	if err == nil {
		t.Fatal("expected conflict error due to username being taken by another user, got nil")
	}
}

func TestRegisterCustomer_LocalAccountAlreadyExistsWithPassword_Conflict(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	existingAcc := &domain.Account{
		ID:       uuid.New(),
		UserID:   userID,
		Email:    "localuser@gmail.com",
		Password: "hashedlocalpassword", // Already has local password
	}

	accountRepoMock := &mockAccountRepo{account: existingAcc}
	userRepoMock := &mockUserRepo{user: nil}

	uc := NewRegisterCustomerUsecase(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		&mockHasher{},
		userRepoMock,
		&mockCustomerRepo{},
		&mockChallengeRepo{},
		&mockOTPGen{otp: "222222"},
		&mockMailer{},
		&mockAuditLogger{},
	)

	params := RegisterCustomerParams{
		Name:     "Local User",
		Username: "localuser",
		Email:    "localuser@gmail.com",
		Password: "newpassword",
	}

	_, err := uc.Execute(ctx, params)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}
