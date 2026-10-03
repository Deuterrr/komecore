package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/infra/service"
	"komecore/internal/modules/auth/repository"
	staffRepo "komecore/internal/modules/staff/repository"
	userRepo "komecore/internal/modules/user/repository"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type VerifyAccountUsecase struct {
	executor       transaction.Executor
	transactor     transaction.Transactor
	accountRepo    repository.AccountRepository
	pwHasher       repository.PasswordHasher
	userRepo       userRepo.UserRepository
	customerRepo   repository.CustomerRepository
	membershipRepo staffRepo.StaffMembershipRepository
	challengeRepo  repository.VerificationChallengeRepository
	sessionIssuer  repository.SessionIssuerService
	auditLogger    applogger.AuditLogger
	sysLogger      applogger.Logger
}

func NewVerifyAccountUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo repository.AccountRepository,
	pwHasher repository.PasswordHasher,
	tokenHasher repository.TokenHasher,
	userRepo userRepo.UserRepository,
	customerRepo repository.CustomerRepository,
	membershipRepo staffRepo.StaffMembershipRepository,
	challengeRepo repository.VerificationChallengeRepository,
	tokenSvc repository.TokenService,
	sessionRepo repository.SessionRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	auditLogger applogger.AuditLogger,
) *VerifyAccountUsecase {
	sessionIssuer := service.NewSessionIssuerService(
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

func (u *VerifyAccountUsecase) SetSessionIssuer(sessionIssuer repository.SessionIssuerService) {
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
	AccessToken, RefreshToken repository.GeneratedToken
}

func (u *VerifyAccountUsecase) Execute(ctx context.Context, input VerifyAccountParams) (result *VerifyAccountResult, err error) {
	now := appclock.Now()

	challenge, err := u.challengeRepo.GetByID(ctx, u.executor, input.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}
	if challenge == nil {
		return nil, apperrors.NewNotFound(domain.ErrNotFoundChallenge.Error())
	}
	if challenge.ConsumedAt != nil {
		return nil, apperrors.NewConflict(domain.ErrConsumedChallenge.Error())
	}
	if challenge.VerifiedAt != nil {
		return nil, apperrors.NewConflict(domain.ErrVerifiedChallenge.Error())
	}
	if challenge.ExpiresAt.Before(now) {
		return nil, apperrors.NewConflict(domain.ErrExpiredChallenge.Error())
	}
	if challenge.AttemptCount >= 5 {
		return nil, apperrors.NewConflict(domain.ErrMaxAttemptReached.Error())
	}

	if err := u.pwHasher.Compare(challenge.CodeHash, input.OTP); err != nil {
		challenge.AttemptCount++
		if err := u.challengeRepo.Save(ctx, u.executor, *challenge); err != nil {
			return nil, fmt.Errorf("failed to update challenge attempts: %w", err)
		}
		return nil, apperrors.NewUnauthorized(domain.ErrInvalidOTP.Error())
	}

	challenge.VerifiedAt = &now
	challenge.ConsumedAt = &now

	var (
		accountID  uuid.UUID
		staffID    *uuid.UUID
		customerID *uuid.UUID
		roleCodes  []domain.RoleCode
	)

	account, err := u.accountRepo.GetByUserID(ctx, u.executor, *challenge.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account != nil {
		accountID = account.ID

		switch account.Type {
		case domain.AccountTypeCustomer:
			cust, err := u.customerRepo.GetByUserID(ctx, u.executor, account.UserID)
			if err != nil {
				return nil, fmt.Errorf("failed to get customer profile: %w", err)
			}
			if cust != nil {
				customerID = &cust.ID
			}

		case domain.AccountTypeStaff:
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
				roleCodes = make([]domain.RoleCode, len(roles))
				for i, r := range roles {
					roleCodes[i] = domain.RoleCode(r.Code)
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

	sessionRes, err := u.sessionIssuer.Issue(ctx, repository.IssueSessionParams{
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
