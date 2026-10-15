package authusecase

import (
	"context"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/auth/authsvc"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type LoginCustomerUsecase struct {
	executor      transaction.Executor
	transactor    transaction.Transactor
	accountRepo   auth.AccountRepository
	pwHasher      auth.PasswordHasher
	customerRepo  auth.CustomerRepository
	sessionIssuer auth.SessionIssuerService
	auditLogger   applogger.AuditLogger
	sysLogger     applogger.Logger
}

func NewLoginCustomerUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo auth.AccountRepository,
	pwHasher auth.PasswordHasher,
	tokenHasher auth.TokenHasher,
	tokenSvc auth.TokenService,
	sessionRepo auth.SessionRepository,
	refreshTokenRepo auth.RefreshTokenRepository,
	customerRepo auth.CustomerRepository,
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

func (u *LoginCustomerUsecase) SetSessionIssuer(sessionIssuer auth.SessionIssuerService) {
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
	AccessToken, RefreshToken auth.GeneratedToken
}

func (u *LoginCustomerUsecase) Execute(ctx context.Context, input LoginCustomerParams) (result *LoginEmailResult, err error) {
	existing, err := u.accountRepo.GetByEmail(ctx, u.executor, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if existing == nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, auth.ErrInvalidCredentials.Error())
	}
	if existing.Type != auth.AccountTypeCustomer {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, auth.ErrInvalidCredentials.Error())
	}
	if existing.Status != auth.AccountActive {
		return nil, apperror.NewForbidden(auth.ErrEmailNotVerified.Error())
	}

	if err := u.pwHasher.Compare(existing.Password, input.Password); err != nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, auth.ErrInvalidCredentials.Error())
	}

	var customerID *uuid.UUID
	cust, err := u.customerRepo.GetByUserID(ctx, u.executor, existing.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer: %w", err)
	}
	if cust != nil {
		customerID = &cust.ID
	}

	sessionRes, err := u.sessionIssuer.Issue(ctx, auth.IssueSessionParams{
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
