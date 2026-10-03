package repository

import (
	"context"
	"time"

	appcookie "komecore/internal/common/http/cookie"
	appmiddleware "komecore/internal/common/middleware"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"

	"github.com/google/uuid"
)

type AccountRepository interface {
	GetByEmail(
		ctx context.Context,
		exec transaction.Executor,
		email string,
	) (*domain.Account, error)
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.Account, error)
	GetByUserID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.Account, error)

	ActivateByUserID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	UpdatePasswordByUserID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		hashedPassword string,
	) error

	Create(
		ctx context.Context,
		exec transaction.Executor,
		account domain.Account,
	) error

	DeleteByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) error

	UpdateLastLoginAt(
		ctx context.Context,
		exec transaction.Executor,
		accID uuid.UUID,
		lastLoginAt time.Time,
	) error
}

type SessionRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.Session, error)

	RevokeByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	RevokeAllByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) error

	UpdateLastActivityByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		session domain.Session,
	) error
}

type VerificationChallengeRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.VerificationChallenge, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		challenge domain.VerificationChallenge,
	) error
}

type RefreshTokenRepository interface {
	GetBySessionID(
		ctx context.Context,
		exec transaction.Executor,
		sessionID uuid.UUID,
	) (*domain.RefreshToken, error)

	RevokeBySessionID(
		ctx context.Context,
		exec transaction.Executor,
		sessionID uuid.UUID,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		challenge domain.RefreshToken,
	) error
}

type TokenService interface {
	Generate(params GenerateTokenParams) (GeneratedToken, error)
	Validate(token string) (*domain.TokenClaims, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

type TokenHasher interface {
	Hash(token string) string
	Compare(hash string, token string) bool
}

type Authenticator interface {
	RequireAuth(
		exec transaction.Executor,
		tran transaction.Transactor,
		cookie appcookie.CookieName,
	) appmiddleware.Middleware

	RequireAnyAuth(
		exec transaction.Executor,
		tran transaction.Transactor,
		cookies ...appcookie.CookieName,
	) appmiddleware.Middleware

	RequireMultiAuth(
		exec transaction.Executor,
		tran transaction.Transactor,
		cookies ...appcookie.CookieName,
	) appmiddleware.Middleware

	OptionalAuth(
		exec transaction.Executor,
		tran transaction.Transactor,
		cookies ...appcookie.CookieName,
	) appmiddleware.Middleware
}

type OAuthConnectionRepository interface {
	GetByProviderAndSubject(
		ctx context.Context,
		exec transaction.Executor,
		provider domain.OAuthProvider,
		subject string,
	) (*domain.OAuthConnection, error)

	GetByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) (*domain.OAuthConnection, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		conn domain.OAuthConnection,
	) error

	UpdateLastLogin(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		lastLoginAt time.Time,
	) error

	DeleteByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) error
}

type UserDeletionService interface {
	DeleteUserRecord(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) error
}

type Authorizer interface {
	RequireAccountType(
		allowedTypes ...domain.AccountType,
	) appmiddleware.Middleware

	RequireStaffRole(
		allowedRoles ...domain.RoleCode,
	) appmiddleware.Middleware

	RequirePermission(
		permission string,
	) appmiddleware.Middleware

	LoadActor(
		exec transaction.Executor,
	) appmiddleware.Middleware

	OptionalLoadActor(
		exec transaction.Executor,
	) appmiddleware.Middleware
}

type IssueSessionParams struct {
	UserID     uuid.UUID
	AccountID  uuid.UUID
	UserAgent  *string
	IPAddress  *string
	StaffID    *uuid.UUID
	CustomerID *uuid.UUID
	Roles      []domain.RoleCode
}

type IssueSessionResult struct {
	SessionID    uuid.UUID
	AccessToken  GeneratedToken
	RefreshToken GeneratedToken
}

type SessionIssuerService interface {
	Issue(
		ctx context.Context,
		params IssueSessionParams,
	) (*IssueSessionResult, error)
}

type CreateChallengeParams struct {
	UserID   *uuid.UUID
	Email    string
	Purpose  domain.OTPPurpose
	Duration time.Duration
}

type ChallengeService interface {
	CreateAndSend(
		ctx context.Context,
		params CreateChallengeParams,
	) (*uuid.UUID, error)
}

type CustomerRepository interface {
	Create(
		ctx context.Context,
		exec transaction.Executor,
		customer domain.Customer,
	) error

	GetByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) (*domain.Customer, error)

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error
}
