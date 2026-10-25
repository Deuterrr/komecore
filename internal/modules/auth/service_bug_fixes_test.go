package auth

import (
	"context"
	"testing"
	"time"

	"komecore/internal/authctx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyAccount_WrongPurpose_Fails(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	challengeID := uuid.New()

	chRepo := &mockChallengeRepo{
		challenge: &VerificationChallenge{
			ID:        challengeID,
			UserID:    &userID,
			Purpose:   OTPPurposePasswordReset, // Wrong purpose for account verification
			CodeHash:  "hashed-123456",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		},
	}

	svc := NewService(
		&mockExecutor{},
		&mockTransactor{},
		&mockAccountRepo{},
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		chRepo,
		&mockOAuthRepo{},
		&mockCustomerRepo{},
		&mockUserRepo{},
		nil, nil,
		&mockHasher{},
		&mockTokenHasher{},
		&mockTokenService{},
		nil,
		&mockOTPGen{},
		&mockMailer{},
		&mockAuditLogger{},
	)

	_, err := svc.VerifyAccount(ctx, VerifyAccountInput{
		ChallengeID: challengeID,
		OTP:         "123456",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrNotFoundChallenge.Error())
}

func TestVerifyAccount_NilUserID_ReturnsErrorWithoutPanic(t *testing.T) {
	ctx := context.Background()
	challengeID := uuid.New()

	chRepo := &mockChallengeRepo{
		challenge: &VerificationChallenge{
			ID:        challengeID,
			UserID:    nil, // Nullable user ID
			Purpose:   OTPPurposeRegister,
			CodeHash:  "hashed-123456",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		},
	}

	svc := NewService(
		&mockExecutor{},
		&mockTransactor{},
		&mockAccountRepo{},
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		chRepo,
		&mockOAuthRepo{},
		&mockCustomerRepo{},
		&mockUserRepo{},
		nil, nil,
		&mockHasher{},
		&mockTokenHasher{},
		&mockTokenService{},
		nil,
		&mockOTPGen{},
		&mockMailer{},
		&mockAuditLogger{},
	)

	_, err := svc.VerifyAccount(ctx, VerifyAccountInput{
		ChallengeID: challengeID,
		OTP:         "123456",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no user_id bound")
}

func TestRequestPasswordReset_MismatchOrInactive_ReturnsSilentSuccess(t *testing.T) {
	ctx := context.Background()

	// Staff account when customer reset requested
	accountRepoMock := &mockAccountRepo{
		account: &Account{
			ID:     uuid.New(),
			UserID: uuid.New(),
			Email:  "staff@example.com",
			Type:   AccountTypeStaff,
			Status: AccountActive,
		},
	}

	svc := NewService(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		&mockChallengeRepo{},
		&mockOAuthRepo{},
		&mockCustomerRepo{},
		&mockUserRepo{},
		nil, nil,
		&mockHasher{},
		&mockTokenHasher{},
		&mockTokenService{},
		nil,
		&mockOTPGen{},
		&mockMailer{},
		&mockAuditLogger{},
	)

	out, err := svc.RequestPasswordReset(ctx, RequestPasswordResetInput{
		Email:       "staff@example.com",
		AccountType: AccountTypeCustomer, // Mismatched type
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Nil(t, out.ChallengeID, "must return nil challenge ID to avoid leaking email registration status")
}

func TestLogout_EvictsFromGraceStore(t *testing.T) {
	ctx := context.Background()
	sessionID := uuid.New()
	userID := uuid.New()

	svc := NewService(
		&mockExecutor{},
		&mockTransactor{},
		&mockAccountRepo{},
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		&mockChallengeRepo{},
		&mockOAuthRepo{},
		&mockCustomerRepo{},
		&mockUserRepo{},
		nil, nil,
		&mockHasher{},
		&mockTokenHasher{},
		&mockTokenService{},
		nil,
		&mockOTPGen{},
		&mockMailer{},
		&mockAuditLogger{},
	)

	// Populate grace store directly
	svc.graceStore[sessionID] = tokenRotationGraceEntry{
		previousTokenHash: "old-hash",
		expiresAt:         time.Now().Add(15 * time.Second),
	}

	err := svc.Logout(ctx, LogoutInput{
		UserID:    userID,
		SessionID: sessionID,
	})
	require.NoError(t, err)

	svc.graceMu.RLock()
	_, exists := svc.graceStore[sessionID]
	svc.graceMu.RUnlock()
	assert.False(t, exists, "session must be evicted from grace store immediately upon logout")
}

func TestActorFromAuthContext_PreservesAccountID(t *testing.T) {
	userID := uuid.New()
	accountID := uuid.New()
	staffID := uuid.New()

	authCtx := &authctx.AuthContext{
		UserID:          userID,
		AccountID:       accountID,
		StaffID:         &staffID,
		AccountType:     authctx.AccountTypeStaff,
		IsAuthenticated: true,
		Roles:           []string{"staff_admin"},
	}

	actor := authctx.ActorFromAuthContext(authCtx)
	require.NotNil(t, actor)
	assert.Equal(t, accountID, actor.AccountID, "actor.AccountID must hold the genuine accountID, not the userID")
	assert.True(t, actor.IsSuperAdmin())
}
