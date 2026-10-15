package authusecase

import (
	"context"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/auth/authsvc"
	"komecore/internal/modules/staff"
	"komecore/internal/modules/user"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type VerifyAccountUsecase struct {
	executor       transaction.Executor
	transactor     transaction.Transactor
	accountRepo    auth.AccountRepository
	pwHasher       auth.PasswordHasher
	userRepo       user.Repository
	customerRepo   auth.CustomerRepository
	membershipRepo staff.MembershipRepository
	challengeRepo  auth.VerificationChallengeRepository
	sessionIssuer  auth.SessionIssuerService
	auditLogger    applogger.AuditLogger
	sysLogger      applogger.Logger
}

func NewVerifyAccountUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo auth.AccountRepository,
	pwHasher auth.PasswordHasher,
	tokenHasher auth.TokenHasher,
	userRepo user.Repository,
	customerRepo auth.CustomerRepository,
	membershipRepo staff.MembershipRepository,
	challengeRepo auth.VerificationChallengeRepository,
	tokenSvc auth.TokenService,
	sessionRepo auth.SessionRepository,
	refreshTokenRepo auth.RefreshTokenRepository,
	auditLogger applogger.AuditLogger,
) *VerifyAccountUsecase {
	sessionIssuer := authsvc.NewSessionIssuerService(
		transactor,
		tokenSvc,
		tokenHasher,
		sessionRepo,
		refreshTokenRepo,
		accountRepo,
	)

	return &VerifyAccountUsecase{
		executor:       executor,
		transactor:     transactor,
		accountRepo:    accountRepo,
		pwHasher:       pwHasher,
		userRepo:       userRepo,
		customerRepo:   customerRepo,
		membershipRepo: membershipRepo,
		challengeRepo:  challengeRepo,
		sessionIssuer:  sessionIssuer,
		auditLogger:    auditLogger,
	}
}

func (u *VerifyAccountUsecase) SetSessionIssuer(sessionIssuer auth.SessionIssuerService) {
	u.sessionIssuer = sessionIssuer
}

func (u *VerifyAccountUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type VerifyAccountParams struct {
	UserAgent   *string
	IPAddress   *string
	ChallengeID uuid.UUID
	OTP         string
}

type VerifyAccountResult struct {
	AccessToken, RefreshToken auth.GeneratedToken
}

func (u *VerifyAccountUsecase) Execute(ctx context.Context, input VerifyAccountParams) (result *VerifyAccountResult, err error) {
	now := appclock.Now()

	challenge, err := u.challengeRepo.GetByID(ctx, u.executor, input.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}
	if challenge == nil {
		return nil, apperror.NewNotFound(auth.ErrNotFoundChallenge.Error())
	}
	if challenge.ConsumedAt != nil {
		return nil, apperror.NewConflict(auth.ErrConsumedChallenge.Error())
	}
	if challenge.VerifiedAt != nil {
		return nil, apperror.NewConflict(auth.ErrVerifiedChallenge.Error())
	}
	if challenge.ExpiresAt.Before(now) {
		return nil, apperror.NewConflict(auth.ErrExpiredChallenge.Error())
	}
	if challenge.AttemptCount >= 5 {
		return nil, apperror.NewConflict(auth.ErrMaxAttemptReached.Error())
	}

	if err := u.pwHasher.Compare(challenge.CodeHash, input.OTP); err != nil {
		challenge.AttemptCount++
		if err := u.challengeRepo.Save(ctx, u.executor, *challenge); err != nil {
			return nil, fmt.Errorf("failed to update challenge attempts: %w", err)
		}
		return nil, apperror.NewUnauthorized(auth.ErrInvalidOTP.Error())
	}

	challenge.VerifiedAt = &now
	challenge.ConsumedAt = &now

	var (
		accountID  uuid.UUID
		staffID    *uuid.UUID
		customerID *uuid.UUID
		roleCodes  []auth.RoleCode
	)

	account, err := u.accountRepo.GetByUserID(ctx, u.executor, *challenge.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account != nil {
		accountID = account.ID

		switch account.Type {
		case auth.AccountTypeCustomer:
			cust, err := u.customerRepo.GetByUserID(ctx, u.executor, account.UserID)
			if err != nil {
				return nil, fmt.Errorf("failed to get customer profile: %w", err)
			}
			if cust != nil {
				customerID = &cust.ID
			}

		case auth.AccountTypeStaff:
			memberStaff, err := u.membershipRepo.GetByAccountID(ctx, u.executor, account.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get staff membership: %w", err)
			}
			if memberStaff != nil {
				staffID = &memberStaff.StaffID
				roles, err := u.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, u.executor,
					account.ID,
					memberStaff.StaffID,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to list staff roles: %w", err)
				}
				roleCodes = make([]auth.RoleCode, len(roles))
				for i, r := range roles {
					roleCodes[i] = auth.RoleCode(r.Code)
				}
			}
		}
	}

	if err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.challengeRepo.Save(ctx, exec, *challenge); err != nil {
			return fmt.Errorf("failed to consume challenge: %w", err)
		}
		if err := u.accountRepo.ActivateByUserID(ctx, exec, *challenge.UserID); err != nil {
			return fmt.Errorf("failed to activate account: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	sessionRes, err := u.sessionIssuer.Issue(ctx, auth.IssueSessionParams{
		UserID:     *challenge.UserID,
		AccountID:  accountID,
		UserAgent:  input.UserAgent,
		IPAddress:  input.IPAddress,
		StaffID:    staffID,
		CustomerID: customerID,
		Roles:      roleCodes,
	})
	if err != nil {
		return nil, err
	}

	return &VerifyAccountResult{
		AccessToken:  sessionRes.AccessToken,
		RefreshToken: sessionRes.RefreshToken,
	}, nil
}
