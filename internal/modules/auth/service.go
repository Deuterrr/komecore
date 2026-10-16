package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff"
	"komecore/internal/modules/user"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"
	mailer "komecore/pkg/mailer"
	otp "komecore/pkg/otp"

	"github.com/google/uuid"
)

const meWriteInterval = 5 * time.Minute

type Service struct {
	executor            transaction.Executor
	transactor          transaction.Transactor
	accountRepo         AccountRepository
	sessionRepo         SessionRepository
	refreshTokenRepo    RefreshTokenRepository
	challengeRepo       VerificationChallengeRepository
	oauthRepo           OAuthConnectionRepository
	customerRepo        CustomerRepository
	userRepo            user.Repository
	staffRepo           staff.Repository
	membershipRepo      staff.MembershipRepository
	pwHasher            PasswordHasher
	tokenHasher         TokenHasher
	tokenSvc            TokenService
	userDeletionService UserDeletionService
	sessionIssuer       SessionIssuerService
	challengeSvc        ChallengeService
	auditLogger         applogger.AuditLogger
	sysLogger           applogger.Logger
}

func NewService(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo AccountRepository,
	sessionRepo SessionRepository,
	refreshTokenRepo RefreshTokenRepository,
	challengeRepo VerificationChallengeRepository,
	oauthRepo OAuthConnectionRepository,
	customerRepo CustomerRepository,
	userRepo user.Repository,
	staffRepo staff.Repository,
	membershipRepo staff.MembershipRepository,
	pwHasher PasswordHasher,
	tokenHasher TokenHasher,
	tokenSvc TokenService,
	userDeletionService UserDeletionService,
	otpGen otp.Generator,
	mailer mailer.Sender,
	auditLogger applogger.AuditLogger,
) *Service {
	sessionIssuer := NewSessionIssuerService(
		transactor,
		tokenSvc,
		tokenHasher,
		sessionRepo,
		refreshTokenRepo,
		accountRepo,
	)

	challengeSvc := NewChallengeService(
		transactor,
		challengeRepo,
		pwHasher,
		otpGen,
		mailer,
	)

	if userDeletionService == nil {
		userDeletionService = NewUserDeletionService(
			accountRepo,
			oauthRepo,
			sessionRepo,
			userRepo,
		)
	}

	return &Service{
		executor:            executor,
		transactor:          transactor,
		accountRepo:         accountRepo,
		sessionRepo:         sessionRepo,
		refreshTokenRepo:    refreshTokenRepo,
		challengeRepo:       challengeRepo,
		oauthRepo:           oauthRepo,
		customerRepo:        customerRepo,
		userRepo:            userRepo,
		staffRepo:           staffRepo,
		membershipRepo:      membershipRepo,
		pwHasher:            pwHasher,
		tokenHasher:         tokenHasher,
		tokenSvc:            tokenSvc,
		userDeletionService: userDeletionService,
		sessionIssuer:       sessionIssuer,
		challengeSvc:        challengeSvc,
		auditLogger:         auditLogger,
	}
}

func (s *Service) SetSessionIssuer(sessionIssuer SessionIssuerService) {
	s.sessionIssuer = sessionIssuer
}

func (s *Service) SetChallengeService(challengeSvc ChallengeService) {
	s.challengeSvc = challengeSvc
}

func (s *Service) SetUserDeletionService(userDeletionService UserDeletionService) {
	s.userDeletionService = userDeletionService
}

func (s *Service) SetAuditLogger(auditLogger applogger.AuditLogger) {
	s.auditLogger = auditLogger
}

func (s *Service) SetSysLogger(sysLogger applogger.Logger) {
	s.sysLogger = sysLogger
}

type defaultSessionIssuer struct {
	transactor       transaction.Transactor
	tokenSvc         TokenService
	tokenHasher      TokenHasher
	sessionRepo      SessionRepository
	refreshTokenRepo RefreshTokenRepository
	accountRepo      AccountRepository
}

func NewSessionIssuerService(
	transactor transaction.Transactor,
	tokenSvc TokenService,
	tokenHasher TokenHasher,
	sessionRepo SessionRepository,
	refreshTokenRepo RefreshTokenRepository,
	accountRepo AccountRepository,
) SessionIssuerService {
	return &defaultSessionIssuer{
		transactor:       transactor,
		tokenSvc:         tokenSvc,
		tokenHasher:      tokenHasher,
		sessionRepo:      sessionRepo,
		refreshTokenRepo: refreshTokenRepo,
		accountRepo:      accountRepo,
	}
}

func (s *defaultSessionIssuer) Issue(ctx context.Context, params IssueSessionParams) (*IssueSessionResult, error) {
	now := appclock.Now()

	session := Session{
		ID:        uuid.New(),
		UserID:    params.UserID,
		UserAgent: params.UserAgent,
		IPAddress: params.IPAddress,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	accessTkn, err := s.tokenSvc.Generate(GenerateTokenParams{
		UserID:     params.UserID,
		SessionID:  session.ID,
		StaffID:    params.StaffID,
		CustomerID: params.CustomerID,
		Roles:      params.Roles,
		Type:       TokenTypeAccess,
		Duration:   30 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshTkn, err := s.tokenSvc.Generate(GenerateTokenParams{
		UserID:     params.UserID,
		SessionID:  session.ID,
		StaffID:    params.StaffID,
		CustomerID: params.CustomerID,
		Roles:      params.Roles,
		Type:       TokenTypeRefresh,
		Duration:   7 * 24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshTknHashed := s.tokenHasher.Hash(refreshTkn.Token)
	refreshTknDomain := RefreshToken{
		ID:        uuid.New(),
		SessionID: session.ID,
		TokenHash: refreshTknHashed,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.sessionRepo.Save(ctx, exec, session); err != nil {
			return fmt.Errorf("failed to save session: %w", err)
		}
		if err := s.refreshTokenRepo.Save(ctx, exec, refreshTknDomain); err != nil {
			return fmt.Errorf("failed to save refresh token: %w", err)
		}
		if s.accountRepo != nil && params.AccountID != uuid.Nil {
			if err := s.accountRepo.UpdateLastLoginAt(ctx, exec, params.AccountID, now); err != nil {
				return fmt.Errorf("failed to update last login at: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &IssueSessionResult{
		SessionID:    session.ID,
		AccessToken:  accessTkn,
		RefreshToken: refreshTkn,
	}, nil
}

type defaultChallengeService struct {
	transactor    transaction.Transactor
	challengeRepo VerificationChallengeRepository
	pwHasher      PasswordHasher
	otpGen        otp.Generator
	mailer        mailer.Sender
}

func NewChallengeService(
	transactor transaction.Transactor,
	challengeRepo VerificationChallengeRepository,
	pwHasher PasswordHasher,
	otpGen otp.Generator,
	mailer mailer.Sender,
) ChallengeService {
	return &defaultChallengeService{
		transactor:    transactor,
		challengeRepo: challengeRepo,
		pwHasher:      pwHasher,
		otpGen:        otpGen,
		mailer:        mailer,
	}
}

func (s *defaultChallengeService) CreateAndSend(ctx context.Context, params CreateChallengeParams) (*uuid.UUID, error) {
	now := appclock.Now()

	code, err := s.otpGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification code: %w", err)
	}
	codeHash, err := s.pwHasher.Hash(code)
	if err != nil {
		return nil, fmt.Errorf("failed to hash verification code: %w", err)
	}

	challengeID := uuid.New()
	challenge := VerificationChallenge{
		ID:        challengeID,
		UserID:    params.UserID,
		Type:      OTPTypeNumeric,
		Channel:   OTPChannelEmail,
		Purpose:   params.Purpose,
		Target:    params.Email,
		CodeHash:  codeHash,
		ExpiresAt: now.Add(params.Duration),
		CreatedAt: now,
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.challengeRepo.Save(ctx, exec, challenge); err != nil {
			return fmt.Errorf("failed to save verification challenge: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	subject := "Verification Code"
	if params.Purpose == OTPPurposePasswordReset {
		subject = "Password Reset Verification Code"
	}

	err = s.mailer.Send(mailer.SendInput{
		To:      params.Email,
		Subject: subject,
		Text:    fmt.Sprintf("Your verification code is: %s", code),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send verification email: %w", err)
	}

	return &challengeID, nil
}

type MeOutput struct {
	Account Account
	Actor   Actor
	User    *user.User
	OAuth   *OAuthConnection
}

func (s *Service) Me(ctx context.Context, authCtx AuthContext) (*MeOutput, error) {
	account, err := s.accountRepo.GetByUserID(ctx, s.executor, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if account == nil {
		return nil, apperror.NewNotFound("account not found")
	}

	if account.Status != AccountActive {
		return nil, apperror.NewForbidden(ErrEmailNotVerified.Error())
	}

	now := appclock.Now()
	if account.LastLoginAt == nil ||
		now.Sub(*account.LastLoginAt) >= meWriteInterval {
		if err := s.accountRepo.UpdateLastLoginAt(ctx, s.executor,
			account.ID,
			now,
		); err != nil {
			return nil, fmt.Errorf("failed to update last login at: %w", err)
		}
		account.LastLoginAt = &now
	}

	actor := authctx.ActorFromAuthContext(&authCtx)
	if actor == nil {
		actor = &Actor{}
	}

	usr, err := s.userRepo.GetByID(ctx, s.executor, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	var oauthConn *OAuthConnection
	if s.oauthRepo != nil {
		oauthConn, err = s.oauthRepo.GetByUserID(ctx, s.executor, authCtx.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve oauth connection: %w", err)
		}
	}

	return &MeOutput{
		Account: *account,
		Actor:   *actor,
		User:    usr,
		OAuth:   oauthConn,
	}, nil
}

type LogoutInput struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

func (s *Service) Logout(ctx context.Context, input LogoutInput) error {
	err := s.transactor.WithinTransaction(
		ctx,
		func(exec transaction.Executor) error {
			if err := s.refreshTokenRepo.RevokeBySessionID(
				ctx,
				exec,
				input.SessionID,
			); err != nil {
				return fmt.Errorf("failed to revoke refresh token %w", err)
			}

			if err := s.sessionRepo.RevokeByID(
				ctx,
				exec,
				input.SessionID,
			); err != nil {
				return fmt.Errorf("failed to revoke refresh token %w", err)
			}

			return nil
		},
	)
	if err != nil {
		return err
	}

	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category:   "user_action",
			Action:     "logout",
			Resource:   "session",
			ResourceID: input.SessionID.String(),
			Outcome:    applogger.OutcomeSuccess,
			Metadata:   map[string]any{"user_id": input.UserID.String()},
		})
	}

	return nil
}

type LoginCustomerInput struct {
	UserAgent *string
	IPAddress *string
	Email     string
	Password  string
}

type LoginCustomerOutput struct {
	AccessToken  GeneratedToken
	RefreshToken GeneratedToken
}

func (s *Service) LoginCustomer(ctx context.Context, input LoginCustomerInput) (*LoginCustomerOutput, error) {
	existing, err := s.accountRepo.GetByEmail(ctx, s.executor, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if existing == nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, ErrInvalidCredentials.Error())
	}
	if existing.Type != AccountTypeCustomer {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, ErrInvalidCredentials.Error())
	}
	if existing.Status != AccountActive {
		return nil, apperror.NewForbidden(ErrEmailNotVerified.Error())
	}

	if err := s.pwHasher.Compare(existing.Password, input.Password); err != nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, ErrInvalidCredentials.Error())
	}

	var customerID *uuid.UUID
	cust, err := s.customerRepo.GetByUserID(ctx, s.executor, existing.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer: %w", err)
	}
	if cust != nil {
		customerID = &cust.ID
	}

	sessionRes, err := s.sessionIssuer.Issue(ctx, IssueSessionParams{
		UserID:     existing.UserID,
		AccountID:  existing.ID,
		UserAgent:  input.UserAgent,
		IPAddress:  input.IPAddress,
		CustomerID: customerID,
	})
	if err != nil {
		return nil, err
	}

	return &LoginCustomerOutput{
		AccessToken:  sessionRes.AccessToken,
		RefreshToken: sessionRes.RefreshToken,
	}, nil
}

type LoginStaffInput struct {
	UserAgent *string
	IPAddress *string
	Email     string
	Password  string
}

type LoginStaffOutput struct {
	AccessToken  GeneratedToken
	RefreshToken GeneratedToken
}

func (s *Service) LoginStaff(ctx context.Context, input LoginStaffInput) (*LoginStaffOutput, error) {
	existing, err := s.accountRepo.GetByEmail(ctx, s.executor, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if existing == nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, ErrInvalidCredentials.Error())
	}
	if existing.Type != AccountTypeStaff {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, ErrInvalidCredentials.Error())
	}
	if existing.Status != AccountActive {
		return nil, apperror.NewForbidden(ErrEmailNotVerified.Error())
	}

	if err := s.pwHasher.Compare(existing.Password, input.Password); err != nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, ErrInvalidCredentials.Error())
	}

	memberStaff, err := s.membershipRepo.GetByAccountID(ctx, s.executor, existing.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve membership: %w", err)
	}
	if memberStaff == nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, ErrInvalidCredentials.Error())
	}

	roles, err := s.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, s.executor,
		existing.ID,
		memberStaff.StaffID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve roles: %w", err)
	}
	if roles == nil {
		return nil, apperror.NewForbidden("no role associated with this account")
	}

	roleCodes := make([]RoleCode, len(roles))
	for i, r := range roles {
		roleCodes[i] = RoleCode(r.Code)
	}

	sessionRes, err := s.sessionIssuer.Issue(ctx, IssueSessionParams{
		UserID:    existing.UserID,
		AccountID: existing.ID,
		UserAgent: input.UserAgent,
		IPAddress: input.IPAddress,
		StaffID:   &memberStaff.StaffID,
		Roles:     roleCodes,
	})
	if err != nil {
		return nil, err
	}

	return &LoginStaffOutput{
		AccessToken:  sessionRes.AccessToken,
		RefreshToken: sessionRes.RefreshToken,
	}, nil
}

type RegisterCustomerInput struct {
	Name     string
	Username string
	Email    string
	Password string
	Phone    *string
}

type RegisterCustomerOutput struct {
	ChallengeID uuid.UUID
}

func (s *Service) RegisterCustomer(ctx context.Context, input RegisterCustomerInput) (*RegisterCustomerOutput, error) {
	now := appclock.Now()

	existUsr, err := s.userRepo.GetByUsername(ctx, s.executor, input.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check user: %w", err)
	}

	existAcc, err := s.accountRepo.GetByEmail(ctx, s.executor, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check account: %w", err)
	}
	if existAcc != nil {
		if existAcc.Password != "" {
			return nil, apperror.NewConflict(ErrAccountAlreadyExists.Error())
		}
	}

	if existUsr != nil {
		return nil, apperror.NewConflict("username already exists")
	}

	var (
		userID     uuid.UUID
		accountID  uuid.UUID
		customerID uuid.UUID
	)

	if existAcc == nil {
		userID = uuid.New()
		accountID = uuid.New()
		customerID = uuid.New()

		hashedPassword, err := s.pwHasher.Hash(input.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		userProps := user.CreateUserProps{
			ID:        userID,
			Name:      input.Name,
			Username:  input.Username,
			Phone:     input.Phone,
			CreatedAt: now,
		}

		acc := Account{
			ID:        accountID,
			UserID:    userID,
			Email:     input.Email,
			Password:  hashedPassword,
			Status:    AccountPending,
			Type:      AccountTypeCustomer,
			CreatedAt: now,
		}

		cust := Customer{
			ID:        customerID,
			UserID:    userID,
			CreatedAt: now,
		}

		if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if err := s.userRepo.CreateUser(ctx, exec, userProps); err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
			if err := s.customerRepo.Create(ctx, exec, cust); err != nil {
				return fmt.Errorf("failed to create customer profile: %w", err)
			}
			if err := s.accountRepo.Create(ctx, exec, acc); err != nil {
				return fmt.Errorf("failed to create account: %w", err)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	if existAcc != nil {
		userID = existAcc.UserID
		accountID = existAcc.ID

		hashedPassword, err := s.pwHasher.Hash(input.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if err := s.accountRepo.UpdatePasswordByUserID(ctx, exec, userID, hashedPassword); err != nil {
				return fmt.Errorf("failed to update account password: %w", err)
			}

			userProps := user.SaveProfileProps{
				UserID:    userID,
				Name:      &input.Name,
				Username:  &input.Username,
				Phone:     input.Phone,
				UpdatedAt: now,
			}
			if err := s.userRepo.SaveProfile(ctx, exec, userProps); err != nil {
				return fmt.Errorf("failed to update user profile: %w", err)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	chID, err := s.challengeSvc.CreateAndSend(ctx, CreateChallengeParams{
		UserID:   &userID,
		Email:    input.Email,
		Purpose:  OTPPurposeRegister,
		Duration: 15 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	var challengeID uuid.UUID
	if chID != nil {
		challengeID = *chID
	}

	return &RegisterCustomerOutput{
		ChallengeID: challengeID,
	}, nil
}

type VerifyAccountInput struct {
	UserAgent   *string
	IPAddress   *string
	ChallengeID uuid.UUID
	OTP         string
}

type VerifyAccountOutput struct {
	AccessToken  GeneratedToken
	RefreshToken GeneratedToken
}

func (s *Service) VerifyAccount(ctx context.Context, input VerifyAccountInput) (*VerifyAccountOutput, error) {
	now := appclock.Now()

	challenge, err := s.challengeRepo.GetByID(ctx, s.executor, input.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}
	if challenge == nil {
		return nil, apperror.NewNotFound(ErrNotFoundChallenge.Error())
	}
	if challenge.ConsumedAt != nil {
		return nil, apperror.NewConflict(ErrConsumedChallenge.Error())
	}
	if challenge.VerifiedAt != nil {
		return nil, apperror.NewConflict(ErrVerifiedChallenge.Error())
	}
	if challenge.ExpiresAt.Before(now) {
		return nil, apperror.NewConflict(ErrExpiredChallenge.Error())
	}
	if challenge.AttemptCount >= 5 {
		return nil, apperror.NewConflict(ErrMaxAttemptReached.Error())
	}

	if err := s.pwHasher.Compare(challenge.CodeHash, input.OTP); err != nil {
		challenge.AttemptCount++
		if err := s.challengeRepo.Save(ctx, s.executor, *challenge); err != nil {
			return nil, fmt.Errorf("failed to update challenge attempts: %w", err)
		}
		return nil, apperror.NewUnauthorized(ErrInvalidOTP.Error())
	}

	challenge.VerifiedAt = &now
	challenge.ConsumedAt = &now

	var (
		accountID  uuid.UUID
		staffID    *uuid.UUID
		customerID *uuid.UUID
		roleCodes  []RoleCode
	)

	account, err := s.accountRepo.GetByUserID(ctx, s.executor, *challenge.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account != nil {
		accountID = account.ID

		switch account.Type {
		case AccountTypeCustomer:
			cust, err := s.customerRepo.GetByUserID(ctx, s.executor, account.UserID)
			if err != nil {
				return nil, fmt.Errorf("failed to get customer profile: %w", err)
			}
			if cust != nil {
				customerID = &cust.ID
			}

		case AccountTypeStaff:
			memberStaff, err := s.membershipRepo.GetByAccountID(ctx, s.executor, account.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get staff membership: %w", err)
			}
			if memberStaff != nil {
				staffID = &memberStaff.StaffID
				roles, err := s.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, s.executor,
					account.ID,
					memberStaff.StaffID,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to list staff roles: %w", err)
				}
				roleCodes = make([]RoleCode, len(roles))
				for i, r := range roles {
					roleCodes[i] = RoleCode(r.Code)
				}
			}
		}
	}

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.challengeRepo.Save(ctx, exec, *challenge); err != nil {
			return fmt.Errorf("failed to consume challenge: %w", err)
		}
		if err := s.accountRepo.ActivateByUserID(ctx, exec, *challenge.UserID); err != nil {
			return fmt.Errorf("failed to activate account: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	sessionRes, err := s.sessionIssuer.Issue(ctx, IssueSessionParams{
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

	return &VerifyAccountOutput{
		AccessToken:  sessionRes.AccessToken,
		RefreshToken: sessionRes.RefreshToken,
	}, nil
}

func (s *Service) GetAccount(ctx context.Context, userID uuid.UUID) (*Account, error) {
	acc, err := s.accountRepo.GetByUserID(ctx, s.executor, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	return acc, nil
}

type AuthenticateOAuthInput struct {
	UserAgent *string
	IPAddress *string
	Provider  OAuthProvider
	Subject   string
	Email     string
	Name      string
	AvatarURL *string
}

type AuthenticateOAuthOutput struct {
	AccessToken  GeneratedToken
	RefreshToken GeneratedToken
}

func (s *Service) AuthenticateOAuth(ctx context.Context, input AuthenticateOAuthInput) (*AuthenticateOAuthOutput, error) {
	now := appclock.Now()

	conn, err := s.oauthRepo.GetByProviderAndSubject(ctx, s.executor,
		input.Provider,
		input.Subject,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve oauth connection: %w", err)
	}

	var userID uuid.UUID
	var account *Account

	// OAuth identity is already linked;
	// update login metadata and continue authentication flow with existing account.
	if conn != nil {
		userID = conn.UserID

		err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if err := s.oauthRepo.UpdateLastLogin(ctx, exec,
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

		account, err = s.accountRepo.GetByUserID(ctx, s.executor, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get account: %w", err)
		}
		if account == nil {
			return nil, apperror.NewNotFound("account not found")
		}

		if account.Status != AccountActive && account.Status != AccountPending {
			return nil, apperror.NewForbidden("account is suspended or locked")
		}
		if account.Status == AccountPending {
			err = s.accountRepo.ActivateByUserID(ctx, s.executor, userID)
			if err != nil {
				return nil, fmt.Errorf("failed to activate account: %w", err)
			}
			account.Status = AccountActive
		}
	}

	// No linked OAuth identity found;
	// check whether the email belongs to an existing account.
	if conn == nil {
		account, err = s.accountRepo.GetByEmail(ctx, s.executor, input.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve account by email: %w", err)
		}
	}

	if conn == nil && account != nil {
		existingConn, err := s.oauthRepo.GetByUserID(ctx, s.executor, account.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing connection by user id: %w", err)
		}
		if existingConn != nil {
			return nil, apperror.NewConflict("this email is already linked with another OAuth account")
		}

		userID = account.UserID
		newConn := OAuthConnection{
			ID:          uuid.New(),
			UserID:      userID,
			Provider:    input.Provider,
			Subject:     input.Subject,
			Email:       &input.Email,
			LastLoginAt: &now,
			CreatedAt:   now,
		}

		if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if account.Status == AccountPending {
				if err := s.accountRepo.ActivateByUserID(ctx, exec, userID); err != nil {
					return fmt.Errorf("failed to activate account: %w", err)
				}
				account.Status = AccountActive
			}
			if err := s.oauthRepo.Create(ctx, exec, newConn); err != nil {
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

		newAcc := Account{
			ID:        accountID,
			UserID:    userID,
			Email:     input.Email,
			Password:  "",
			Status:    AccountActive,
			Type:      AccountTypeCustomer,
			CreatedAt: now,
		}

		cust := Customer{
			ID:        customerID,
			UserID:    userID,
			CreatedAt: now,
		}

		newConn := OAuthConnection{
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

		if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if err := s.userRepo.CreateUser(ctx, exec, userProps); err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
			if err := s.userRepo.SaveProfile(ctx, exec, profile); err != nil {
				return fmt.Errorf("failed to save user profile: %w", err)
			}
			if err := s.customerRepo.Create(ctx, exec, cust); err != nil {
				return fmt.Errorf("failed to create customer profile: %w", err)
			}
			if err := s.accountRepo.Create(ctx, exec, newAcc); err != nil {
				return fmt.Errorf("failed to create account: %w", err)
			}
			if err := s.oauthRepo.Create(ctx, exec, newConn); err != nil {
				return fmt.Errorf("failed to create oauth connection: %w", err)
			}
			return nil
		}); err != nil {
			return nil, err
		}

		account = &newAcc
	}

	var customerID *uuid.UUID
	custProfile, err := s.customerRepo.GetByUserID(ctx, s.executor, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer profile: %w", err)
	}
	if custProfile != nil {
		customerID = &custProfile.ID
	}

	sessionRes, err := s.sessionIssuer.Issue(ctx, IssueSessionParams{
		UserID:     userID,
		AccountID:  account.ID,
		UserAgent:  input.UserAgent,
		IPAddress:  input.IPAddress,
		CustomerID: customerID,
	})
	if err != nil {
		return nil, err
	}

	return &AuthenticateOAuthOutput{
		AccessToken:  sessionRes.AccessToken,
		RefreshToken: sessionRes.RefreshToken,
	}, nil
}

type RequestPasswordResetInput struct {
	Email       string
	AccountType AccountType
}

type RequestPasswordResetOutput struct {
	ChallengeID *uuid.UUID
}

func (s *Service) RequestPasswordReset(ctx context.Context, input RequestPasswordResetInput) (*RequestPasswordResetOutput, error) {
	account, err := s.accountRepo.GetByEmail(ctx, s.executor, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		// Silent success for non-existent accounts to prevent email enumeration
		return &RequestPasswordResetOutput{ChallengeID: nil}, nil
	}
	if account.Type != input.AccountType {
		return nil, apperror.NewForbidden("account type mismatch")
	}
	if account.Status != AccountActive {
		return nil, apperror.NewForbidden(ErrEmailNotVerified.Error())
	}

	chID, err := s.challengeSvc.CreateAndSend(ctx, CreateChallengeParams{
		UserID:   &account.UserID,
		Email:    input.Email,
		Purpose:  OTPPurposePasswordReset,
		Duration: 15 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return &RequestPasswordResetOutput{
		ChallengeID: chID,
	}, nil
}

type VerifyPasswordResetInput struct {
	ChallengeID uuid.UUID
	OTP         string
}

type VerifyPasswordResetOutput struct {
	ChallengeID uuid.UUID
}

func (s *Service) VerifyPasswordReset(ctx context.Context, input VerifyPasswordResetInput) (*VerifyPasswordResetOutput, error) {
	now := appclock.Now()

	challenge, err := s.challengeRepo.GetByID(ctx, s.executor, input.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}
	if challenge == nil {
		return nil, apperror.NewNotFound(ErrNotFoundChallenge.Error())
	}
	if challenge.Purpose != OTPPurposePasswordReset {
		return nil, apperror.NewNotFound(ErrNotFoundChallenge.Error())
	}
	if challenge.ConsumedAt != nil {
		return nil, apperror.NewConflict(ErrConsumedChallenge.Error())
	}
	if challenge.VerifiedAt != nil {
		return nil, apperror.NewConflict(ErrVerifiedChallenge.Error())
	}
	if challenge.ExpiresAt.Before(now) {
		return nil, apperror.NewConflict(ErrExpiredChallenge.Error())
	}
	if challenge.AttemptCount >= 5 {
		return nil, apperror.NewConflict(ErrMaxAttemptReached.Error())
	}

	if err := s.pwHasher.Compare(challenge.CodeHash, input.OTP); err != nil {
		challenge.AttemptCount++

		if err := s.challengeRepo.Save(ctx, s.executor, *challenge); err != nil {
			return nil, fmt.Errorf("failed to update challenge attempts: %w", err)
		}

		return nil, apperror.NewUnauthorized(ErrInvalidOTP.Error())
	}

	challenge.VerifiedAt = &now
	if err := s.challengeRepo.Save(ctx, s.executor, *challenge); err != nil {
		return nil, fmt.Errorf("failed to update challenge verification state: %w", err)
	}

	return &VerifyPasswordResetOutput{
		ChallengeID: challenge.ID,
	}, nil
}

type ResetPasswordInput struct {
	ChallengeID uuid.UUID
	NewPassword string
}

func (s *Service) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	now := appclock.Now()

	challenge, err := s.challengeRepo.GetByID(ctx, s.executor, input.ChallengeID)
	if err != nil {
		return fmt.Errorf("failed to get challenge: %w", err)
	}
	if challenge == nil {
		return apperror.NewNotFound(ErrNotFoundChallenge.Error())
	}
	if challenge.Purpose != OTPPurposePasswordReset {
		return apperror.NewNotFound(ErrNotFoundChallenge.Error())
	}
	if challenge.ConsumedAt != nil {
		return apperror.NewConflict(ErrConsumedChallenge.Error())
	}
	if challenge.VerifiedAt == nil {
		return apperror.NewConflict("challenge is not verified")
	}
	if challenge.ExpiresAt.Before(now) {
		return apperror.NewConflict(ErrExpiredChallenge.Error())
	}
	if challenge.UserID == nil {
		return apperror.NewInternal(fmt.Errorf("challenge has no user_id bound"))
	}

	hashedPassword, err := s.pwHasher.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	challenge.ConsumedAt = &now

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.accountRepo.UpdatePasswordByUserID(ctx, exec, *challenge.UserID, hashedPassword); err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
		if err := s.sessionRepo.RevokeAllByUserID(ctx, exec, *challenge.UserID); err != nil {
			return fmt.Errorf("failed to revoke sessions: %w", err)
		}
		if err := s.challengeRepo.Save(ctx, exec, *challenge); err != nil {
			return fmt.Errorf("failed to consume challenge: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

type RefreshTokenInput struct {
	RefreshToken string
	AccountType  AccountType
}

type RefreshTokenOutput struct {
	AccessToken  GeneratedToken
	RefreshToken GeneratedToken
}

func (s *Service) RefreshToken(ctx context.Context, input RefreshTokenInput) (*RefreshTokenOutput, error) {
	if input.RefreshToken == "" {
		return nil, apperror.NewUnauthorized("refresh token is required")
	}

	claims, err := s.tokenSvc.Validate(input.RefreshToken)
	if err != nil {
		return nil, apperror.NewUnauthorized(ErrInvalidToken.Error())
	}
	if claims.Type != TokenTypeRefresh {
		return nil, apperror.NewUnauthorized("invalid token type for refresh")
	}

	if input.AccountType == AccountTypeStaff && claims.StaffID == nil {
		return nil, apperror.NewForbidden("staff account required")
	}
	if input.AccountType == AccountTypeCustomer && claims.StaffID != nil {
		return nil, apperror.NewForbidden("customer account required")
	}

	session, err := s.sessionRepo.GetByID(ctx, s.executor, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil || session.RevokedAt != nil || session.ExpiresAt.Before(appclock.Now()) {
		return nil, apperror.NewUnauthorized(ErrInvalidSession.Error())
	}

	dbRefreshToken, err := s.refreshTokenRepo.GetBySessionID(ctx, s.executor, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	if dbRefreshToken == nil || dbRefreshToken.RevokedAt != nil || dbRefreshToken.ExpiresAt.Before(appclock.Now()) {
		return nil, apperror.NewUnauthorized("refresh token is invalid or expired")
	}

	if !s.tokenHasher.Compare(dbRefreshToken.TokenHash, input.RefreshToken) {
		return nil, apperror.NewUnauthorized("refresh token mismatch")
	}

	roleCodes := make([]RoleCode, len(claims.Roles))
	for i, rCode := range claims.Roles {
		roleCodes[i] = RoleCode(rCode)
	}

	newAccessTkn, err := s.tokenSvc.Generate(GenerateTokenParams{
		UserID:     claims.UserID,
		SessionID:  claims.SessionID,
		StaffID:    claims.StaffID,
		CustomerID: claims.CustomerID,
		Roles:      roleCodes,
		Type:       TokenTypeAccess,
		Duration:   30 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	newRefreshTkn, err := s.tokenSvc.Generate(GenerateTokenParams{
		UserID:     claims.UserID,
		SessionID:  claims.SessionID,
		StaffID:    claims.StaffID,
		CustomerID: claims.CustomerID,
		Roles:      roleCodes,
		Type:       TokenTypeRefresh,
		Duration:   7 * 24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	now := appclock.Now()
	session.ExpiresAt = now.Add(7 * 24 * time.Hour)
	session.LastActivityAt = &now

	newRefreshTknHashed := s.tokenHasher.Hash(newRefreshTkn.Token)
	dbRefreshToken.TokenHash = newRefreshTknHashed
	dbRefreshToken.ExpiresAt = now.Add(7 * 24 * time.Hour)

	if err := s.transactor.WithinTransaction(ctx, func(e transaction.Executor) error {
		if err := s.sessionRepo.Save(ctx, e, *session); err != nil {
			return fmt.Errorf("failed to save updated session: %w", err)
		}
		if err := s.refreshTokenRepo.Save(ctx, e, *dbRefreshToken); err != nil {
			return fmt.Errorf("failed to save updated refresh token: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &RefreshTokenOutput{
		AccessToken:  newAccessTkn,
		RefreshToken: newRefreshTkn,
	}, nil
}

func (s *Service) DeleteAccount(ctx context.Context, authCtx AuthContext) error {
	userID := authCtx.UserID

	if err := s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if s.customerRepo != nil && authCtx.CustomerID != nil {
			if err := s.customerRepo.Delete(ctx, exec, *authCtx.CustomerID); err != nil {
				return fmt.Errorf("failed to delete customer record: %w", err)
			}
		}
		if err := s.userDeletionService.DeleteUserRecord(ctx, exec, userID); err != nil {
			return fmt.Errorf("failed to delete user record: %w", err)
		}
		return nil
	}); err != nil {
		if s.auditLogger != nil {
			s.auditLogger.Log(ctx, applogger.AuditEvent{
				Category: "user_action",
				Action:   "delete_account",
				Resource: "user",
				Outcome:  applogger.OutcomeFailure,
				Metadata: map[string]any{
					"user_id": userID.String(),
					"error":   err.Error(),
				},
			})
		}
		return err
	}

	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "delete_account",
			Resource: "user",
			Outcome:  applogger.OutcomeSuccess,
			Metadata: map[string]any{
				"user_id": userID.String(),
			},
		})
	}

	return nil
}
