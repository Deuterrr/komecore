package authusecase

import (
	"context"
	"fmt"
	"strings"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/auth/authsvc"
	"komecore/internal/modules/user"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type AuthenticateOAuthUsecase struct {
	executor      transaction.Executor
	transactor    transaction.Transactor
	accountRepo   auth.AccountRepository
	oauthRepo     auth.OAuthConnectionRepository
	userRepo      user.Repository
	customerRepo  auth.CustomerRepository
	sessionIssuer auth.SessionIssuerService
	auditLogger   applogger.AuditLogger
	sysLogger     applogger.Logger
}

func NewAuthenticateOAuthUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo auth.AccountRepository,
	oauthRepo auth.OAuthConnectionRepository,
	userRepo user.Repository,
	customerRepo auth.CustomerRepository,
	tokenHasher auth.TokenHasher,
	tokenSvc auth.TokenService,
	sessionRepo auth.SessionRepository,
	refreshTokenRepo auth.RefreshTokenRepository,
	auditLogger applogger.AuditLogger,
) *AuthenticateOAuthUsecase {
	sessionIssuer := authsvc.NewSessionIssuerService(
		transactor,
		tokenSvc,
		tokenHasher,
		sessionRepo,
		refreshTokenRepo,
		accountRepo,
	)

	return &AuthenticateOAuthUsecase{
		executor:      executor,
		transactor:    transactor,
		accountRepo:   accountRepo,
		oauthRepo:     oauthRepo,
		userRepo:      userRepo,
		customerRepo:  customerRepo,
		sessionIssuer: sessionIssuer,
		auditLogger:   auditLogger,
	}
}

func (u *AuthenticateOAuthUsecase) SetSessionIssuer(sessionIssuer auth.SessionIssuerService) {
	u.sessionIssuer = sessionIssuer
}

func (u *AuthenticateOAuthUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type AuthenticateOAuthParams struct {
	UserAgent *string
	IPAddress *string
	Provider  auth.OAuthProvider
	Subject   string
	Email     string
	Name      string
	AvatarURL *string
}

type AuthenticateOAuthResult struct {
	AccessToken  auth.GeneratedToken
	RefreshToken auth.GeneratedToken
}

func (u *AuthenticateOAuthUsecase) Execute(ctx context.Context, input AuthenticateOAuthParams) (result *AuthenticateOAuthResult, err error) {
	now := appclock.Now()

	conn, err := u.oauthRepo.GetByProviderAndSubject(ctx, u.executor,
		input.Provider,
		input.Subject,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve oauth connection: %w", err)
	}

	var userID uuid.UUID
	var account *auth.Account

	// OAuth identity is already linked;
	// update login metadata and continue authentication flow with existing account.
	if conn != nil {
		userID = conn.UserID

		err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if err := u.oauthRepo.UpdateLastLogin(ctx, exec,
				conn.ID,
				now,
			); err != nil {
				return fmt.Errorf("failed to update last login: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		account, err = u.accountRepo.GetByUserID(ctx, u.executor, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get account: %w", err)
		}
		if account == nil {
			return nil, apperror.NewNotFound("account not found")
		}

		if account.Status != auth.AccountActive &&
			account.Status != auth.AccountPending {
			return nil, apperror.NewForbidden("account is suspended or locked")
		}

		if account.Status == auth.AccountPending {
			err = u.accountRepo.ActivateByUserID(ctx, u.executor, userID)
			if err != nil {
				return nil, fmt.Errorf("failed to activate account: %w", err)
			}
			account.Status = auth.AccountActive
		}
	}

	// No linked OAuth identity found;
	// check whether the email belongs to an existing account.
	if conn == nil {
		account, err = u.accountRepo.GetByEmail(ctx, u.executor, input.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve account by email: %w", err)
		}
	}

	if conn == nil && account != nil {
		existingConn, err := u.oauthRepo.GetByUserID(ctx, u.executor, account.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing connection by user id: %w", err)
		}
		if existingConn != nil {
			return nil, apperror.NewConflict("this email is already linked with another OAuth account")
		}

		userID = account.UserID
		newConn := auth.OAuthConnection{
			ID:          uuid.New(),
			UserID:      userID,
			Provider:    input.Provider,
			Subject:     input.Subject,
			Email:       &input.Email,
			LastLoginAt: &now,
			CreatedAt:   now,
		}

		if err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if account.Status == auth.AccountPending {
				if err := u.accountRepo.ActivateByUserID(ctx, exec, userID); err != nil {
					return fmt.Errorf("failed to activate account: %w", err)
				}
				account.Status = auth.AccountActive
			}
			if err := u.oauthRepo.Create(ctx, exec, newConn); err != nil {
				return fmt.Errorf("failed to link oauth connection: %w", err)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	// Neither OAuth identity nor local Account found;
	// Create a new User, Customer, Account, and OAuth Connection.
	if conn == nil && account == nil {
		userID = uuid.New()
		accountID := uuid.New()
		customerID := uuid.New()

		baseUsername, _, _ := strings.Cut(input.Email, "@")
		username := fmt.Sprintf("%s_%s", baseUsername, uuid.New().String()[:7])

		userProps := user.CreateUserProps{
			ID:        userID,
			Name:      input.Name,
			Username:  username,
			AvatarURL: input.AvatarURL,
			CreatedAt: now,
		}

		newAcc := auth.Account{
			ID:        accountID,
			UserID:    userID,
			Email:     input.Email,
			Password:  "",
			Status:    auth.AccountActive,
			Type:      auth.AccountTypeCustomer,
			CreatedAt: now,
		}

		cust := auth.Customer{
			ID:        customerID,
			UserID:    userID,
			CreatedAt: now,
		}

		newConn := auth.OAuthConnection{
			ID:          uuid.New(),
			UserID:      userID,
			Provider:    input.Provider,
			Subject:     input.Subject,
			Email:       &input.Email,
			LastLoginAt: &now,
			CreatedAt:   now,
		}

		profile := user.SaveProfileProps{
			UserID:    userID,
			Name:      &input.Name,
			AvatarURL: input.AvatarURL,
			UpdatedAt: now,
		}

		if err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if err := u.userRepo.CreateUser(ctx, exec, userProps); err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
			if err := u.userRepo.SaveProfile(ctx, exec, profile); err != nil {
				return fmt.Errorf("failed to save user profile: %w", err)
			}
			if err := u.customerRepo.Create(ctx, exec, cust); err != nil {
				return fmt.Errorf("failed to create customer profile: %w", err)
			}
			if err := u.accountRepo.Create(ctx, exec, newAcc); err != nil {
				return fmt.Errorf("failed to create account: %w", err)
			}
			if err := u.oauthRepo.Create(ctx, exec, newConn); err != nil {
				return fmt.Errorf("failed to create oauth connection: %w", err)
			}
			return nil
		}); err != nil {
			return nil, err
		}

		account = &newAcc
	}

	var customerID *uuid.UUID
	custProfile, err := u.customerRepo.GetByUserID(ctx, u.executor, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer profile: %w", err)
	}
	if custProfile != nil {
		customerID = &custProfile.ID
	}

	sessionRes, err := u.sessionIssuer.Issue(ctx, auth.IssueSessionParams{
		UserID:     userID,
		AccountID:  account.ID,
		UserAgent:  input.UserAgent,
		IPAddress:  input.IPAddress,
		CustomerID: customerID,
	})
	if err != nil {
		return nil, err
	}

	return &AuthenticateOAuthResult{
		AccessToken:  sessionRes.AccessToken,
		RefreshToken: sessionRes.RefreshToken,
	}, nil
}
