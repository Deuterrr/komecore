package auth

import (
	"context"
	"testing"
	"time"

	"komecore/internal/modules/user"

	"github.com/google/uuid"
)

func TestMeUsecase_Execute_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()

	existingAcc := &Account{
		ID:     accountID,
		UserID: userID,
		Email:  "me@gmail.com",
		Status: AccountActive,
	}

	avatar := "http://example.com/me.png"
	existingUser := &user.User{
		ID:        userID,
		Name:      "Me User",
		AvatarURL: &avatar,
	}

	accountRepoMock := &mockAccountRepo{account: existingAcc}
	userRepoMock := &mockUserRepo{user: existingUser}

	svc := NewService(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		&mockChallengeRepo{},
		&mockOAuthRepo{},
		&mockCustomerRepo{},
		userRepoMock,
		nil,
		nil,
		&mockHasher{},
		&mockTokenHasher{},
		&mockTokenService{},
		nil,
		&mockOTPGen{},
		&mockMailer{},
		&mockAuditLogger{},
	)

	authCtx := AuthContext{
		UserID:          userID,
		IsAuthenticated: true,
	}

	result, err := svc.Me(ctx, authCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to be returned")
	}

	if result.User == nil || result.User.AvatarURL == nil || *result.User.AvatarURL != avatar {
		t.Errorf("expected AvatarURL to be %s, got %v", avatar, result.User.AvatarURL)
	}

	if accountRepoMock.updateLastLoginAtCalls != 1 {
		t.Errorf("expected updateLastLoginAtCalls to be 1, got %d", accountRepoMock.updateLastLoginAtCalls)
	}
	if result.Account.LastLoginAt == nil {
		t.Error("expected LastLoginAt to be updated on account, got nil")
	}
}

func TestMeUsecase_SmartWrite_SkippedWhenRecent(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()

	recentTime := time.Now().Add(-1 * time.Minute)
	existingAcc := &Account{
		ID:          accountID,
		UserID:      userID,
		Email:       "recent@gmail.com",
		Status:      AccountActive,
		LastLoginAt: &recentTime,
	}

	accountRepoMock := &mockAccountRepo{account: existingAcc}
	userRepoMock := &mockUserRepo{user: &user.User{ID: userID}}

	svc := NewService(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		&mockChallengeRepo{},
		&mockOAuthRepo{},
		&mockCustomerRepo{},
		userRepoMock,
		nil,
		nil,
		&mockHasher{},
		&mockTokenHasher{},
		&mockTokenService{},
		nil,
		&mockOTPGen{},
		&mockMailer{},
		&mockAuditLogger{},
	)

	authCtx := AuthContext{
		UserID:          userID,
		IsAuthenticated: true,
	}

	result, err := svc.Me(ctx, authCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if accountRepoMock.updateLastLoginAtCalls != 0 {
		t.Errorf("expected smart write to be skipped when recent, but updateLastLoginAtCalls was %d", accountRepoMock.updateLastLoginAtCalls)
	}

	if result.Account.LastLoginAt != &recentTime {
		t.Errorf("expected LastLoginAt to remain %v, got %v", recentTime, result.Account.LastLoginAt)
	}
}

func TestMeUsecase_SmartWrite_TriggersWhenOlderThanThreshold(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()

	oldTime := time.Now().Add(-10 * time.Minute)
	existingAcc := &Account{
		ID:          accountID,
		UserID:      userID,
		Email:       "old@gmail.com",
		Status:      AccountActive,
		LastLoginAt: &oldTime,
	}

	accountRepoMock := &mockAccountRepo{account: existingAcc}
	userRepoMock := &mockUserRepo{user: &user.User{ID: userID}}

	svc := NewService(
		&mockExecutor{},
		&mockTransactor{},
		accountRepoMock,
		&mockSessionRepo{},
		&mockRefreshTokenRepo{},
		&mockChallengeRepo{},
		&mockOAuthRepo{},
		&mockCustomerRepo{},
		userRepoMock,
		nil,
		nil,
		&mockHasher{},
		&mockTokenHasher{},
		&mockTokenService{},
		nil,
		&mockOTPGen{},
		&mockMailer{},
		&mockAuditLogger{},
	)

	authCtx := AuthContext{
		UserID:          userID,
		IsAuthenticated: true,
	}

	result, err := svc.Me(ctx, authCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if accountRepoMock.updateLastLoginAtCalls != 1 {
		t.Errorf("expected smart write to trigger when older than threshold, got %d calls", accountRepoMock.updateLastLoginAtCalls)
	}

	if result.Account.LastLoginAt == &oldTime {
		t.Error("expected LastLoginAt to be updated to a newer timestamp")
	}
}
