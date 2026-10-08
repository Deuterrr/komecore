package authusecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authsvc"
	"komecore/internal/modules/auth/authrepo"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type LoginCustomerUsecase struct {
	executor      transaction.Executor
	transactor    transaction.Transactor
	accountRepo   authrepo.AccountRepository
	pwHasher      authrepo.PasswordHasher
	customerRepo  authrepo.CustomerRepository
	sessionIssuer authrepo.SessionIssuerService
	auditLogger   applogger.AuditLogger
	sysLogger     applogger.Logger
}

func NewLoginCustomerUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo authrepo.AccountRepository,
	pwHasher authrepo.PasswordHasher,
	tokenHasher authrepo.TokenHasher,
	tokenSvc authrepo.TokenService,
	sessionRepo authrepo.SessionRepository,
	refreshTokenRepo authrepo.RefreshTokenRepository,
	customerRepo authrepo.CustomerRepository,
	auditLogger applogger.AuditLogger,
) *LoginCustomerUsecase {
	sessionIssuer := authsvc.NewSessionIssuerService(
		transactor,
		tokenSvc,
		tokenHasher,
		sessionRepo,
		refreshTokenRepo,
		accountRepo,
	)

	return &LoginCustomerUsecase{
		executor:      executor,
		transactor:    transactor,
		accountRepo:   accountRepo,
		pwHasher:      pwHasher,
		customerRepo:  customerRepo,
		sessionIssuer: sessionIssuer,
		auditLogger:   auditLogger,
	}
}

func (u *LoginCustomerUsecase) SetSessionIssuer(sessionIssuer authrepo.SessionIssuerService) {
	u.sessionIssuer = sessionIssuer
}

func (u *LoginCustomerUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type LoginCustomerParams struct {
	UserAgent *string
	IPAddress *string
	Email     string
	Password  string
}

type LoginEmailResult struct {
	AccessToken, RefreshToken authrepo.GeneratedToken
}

func (u *LoginCustomerUsecase) Execute(ctx context.Context, input LoginCustomerParams) (result *LoginEmailResult, err error) {
	existing, err := u.accountRepo.GetByEmail(ctx, u.executor, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if existing == nil {
		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidCredentials.Error())
	}
	if existing.Type != authdomain.AccountTypeCustomer {
		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidCredentials.Error())
	}
	if existing.Status != authdomain.AccountActive {
		return nil, apperrors.NewForbidden(authdomain.ErrEmailNotVerified.Error())
	}

	if err := u.pwHasher.Compare(existing.Password, input.Password); err != nil {
		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidCredentials.Error())
	}

	var customerID *uuid.UUID
	cust, err := u.customerRepo.GetByUserID(ctx, u.executor, existing.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer: %w", err)
	}
	if cust != nil {
		customerID = &cust.ID
	}

	sessionRes, err := u.sessionIssuer.Issue(ctx, authrepo.IssueSessionParams{
		UserID:     existing.UserID,
		AccountID:  existing.ID,
		UserAgent:  input.UserAgent,
		IPAddress:  input.IPAddress,
		CustomerID: customerID,
	})
	if err != nil {
		return nil, err
	}

	return &LoginEmailResult{
		AccessToken:  sessionRes.AccessToken,
		RefreshToken: sessionRes.RefreshToken,
	}, nil
}
