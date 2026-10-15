package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

type AccountRepository interface {
	GetByEmail(
		ctx context.Context,
		exec transaction.Executor,
		email string,
	) (*Account, error)
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*Account, error)
	GetByUserID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*Account, error)

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
		account Account,
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
	) (*Session, error)

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
		session Session,
	) error
}

type VerificationChallengeRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*VerificationChallenge, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		challenge VerificationChallenge,
	) error
}

type RefreshTokenRepository interface {
	GetBySessionID(
		ctx context.Context,
		exec transaction.Executor,
		sessionID uuid.UUID,
	) (*RefreshToken, error)

	RevokeBySessionID(
		ctx context.Context,
		exec transaction.Executor,
		sessionID uuid.UUID,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		challenge RefreshToken,
	) error
}

type TokenService interface {
	Generate(params GenerateTokenParams) (GeneratedToken, error)
	Validate(token string) (*TokenClaims, error)
}

type TokenHasher interface {
	Hash(token string) string
	Compare(hash string, token string) bool
}

type OAuthConnectionRepository interface {
	GetByProviderAndSubject(
		ctx context.Context,
		exec transaction.Executor,
		provider OAuthProvider,
		subject string,
	) (*OAuthConnection, error)

	GetByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) (*OAuthConnection, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		conn OAuthConnection,
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

type IssueSessionParams struct {
	UserID     uuid.UUID
	AccountID  uuid.UUID
	UserAgent  *string
	IPAddress  *string
	StaffID    *uuid.UUID
	CustomerID *uuid.UUID
	Roles      []RoleCode
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
	Purpose  OTPPurpose
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
		customer Customer,
	) error

	GetByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) (*Customer, error)

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error
}
