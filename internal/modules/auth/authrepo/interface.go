package authrepo

import (
	"context"
	"time"

	appcookie "komecore/internal/httpx/cookie"
	appmiddleware "komecore/internal/httpx/middleware"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"

	"github.com/google/uuid"
)

type AccountRepository interface {
	GetByEmail(
		ctx context.Context,
		exec transaction.Executor,
		email string,
	) (*authdomain.Account, error)
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*authdomain.Account, error)
	GetByUserID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*authdomain.Account, error)

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
		account authdomain.Account,
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
	) (*authdomain.Session, error)

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
		session authdomain.Session,
	) error
}

type VerificationChallengeRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*authdomain.VerificationChallenge, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		challenge authdomain.VerificationChallenge,
	) error
}

type RefreshTokenRepository interface {
	GetBySessionID(
		ctx context.Context,
		exec transaction.Executor,
		sessionID uuid.UUID,
	) (*authdomain.RefreshToken, error)

	RevokeBySessionID(
		ctx context.Context,
		exec transaction.Executor,
		sessionID uuid.UUID,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		challenge authdomain.RefreshToken,
	) error
}

type TokenService interface {
	Generate(params GenerateTokenParams) (GeneratedToken, error)
	Validate(token string) (*authdomain.TokenClaims, error)
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
		provider authdomain.OAuthProvider,
		subject string,
	) (*authdomain.OAuthConnection, error)

	GetByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) (*authdomain.OAuthConnection, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		conn authdomain.OAuthConnection,
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
		allowedTypes ...authdomain.AccountType,
	) appmiddleware.Middleware

	RequireStaffRole(
		allowedRoles ...authdomain.RoleCode,
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
	Roles      []authdomain.RoleCode
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
	Purpose  authdomain.OTPPurpose
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
		customer authdomain.Customer,
	) error

	GetByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) (*authdomain.Customer, error)

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error
}
