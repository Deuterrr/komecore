package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

// AccountRepository defines persistence operations for user authentication accounts.
//
// It manages account lifecycle states, credential updates, login tracking, and
// lookups across primary identifiers within transaction boundaries.
type AccountRepository interface {
	// Account lookups.
	GetByEmail(ctx context.Context, exec transaction.Executor, email string) (*Account, error)
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Account, error)
	GetByUserID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Account, error)

	// Status transitions.
	ActivateByUserID(ctx context.Context, exec transaction.Executor, id uuid.UUID) error
	SetPendingByUserID(ctx context.Context, exec transaction.Executor, id uuid.UUID) error

	// Credential and record updates.
	UpdatePasswordByUserID(ctx context.Context, exec transaction.Executor, id uuid.UUID, hashedPassword string) error
	UpdateLastLoginAt(ctx context.Context, exec transaction.Executor, accID uuid.UUID, lastLoginAt time.Time) error

	// Lifecycle management.
	Create(ctx context.Context, exec transaction.Executor, account Account) error
	DeleteByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error
}

// SessionRepository defines persistence operations for user authentication sessions.
//
// It handles session lookups, activity heartbeat tracking, and individual or
// bulk session revocations across devices.
type SessionRepository interface {
	// Session lookups.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Session, error)

	// Session revocations.
	RevokeByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) error
	RevokeAllByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error

	// Session lifecycle and activity.
	UpdateLastActivityByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) error
	Save(ctx context.Context, exec transaction.Executor, session Session) error
}

// VerificationChallengeRepository manages storage and retrieval of OTP challenges.
//
// It persists short-lived verification codes used for account registration,
// password resets, and email verifications.
type VerificationChallengeRepository interface {
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*VerificationChallenge, error)
	Save(ctx context.Context, exec transaction.Executor, challenge VerificationChallenge) error
}

// RefreshTokenRepository defines persistence operations for cryptographic refresh tokens.
//
// It stores hashed refresh tokens linked to active sessions and ensures invalidated
// tokens cannot be reused to exchange for new access tokens.
type RefreshTokenRepository interface {
	GetBySessionID(ctx context.Context, exec transaction.Executor, sessionID uuid.UUID) (*RefreshToken, error)
	RevokeBySessionID(ctx context.Context, exec transaction.Executor, sessionID uuid.UUID) error
	Save(ctx context.Context, exec transaction.Executor, challenge RefreshToken) error
}

// TokenService provides functionality to issue and cryptographically verify tokens.
//
// It encapsulates token signing, expiration policies, and parsing of encoded
// claims for access and refresh tokens.
type TokenService interface {
	Generate(params GenerateTokenParams) (GeneratedToken, error)
	Validate(token string) (*TokenClaims, error)
}

// TokenHasher hashes and verifies raw tokens against cryptographic digests.
//
// It prevents plaintext storage of sensitive tokens such as session refresh
// tokens within persistence layers.
type TokenHasher interface {
	Hash(token string) string
	Compare(hash string, token string) bool
}

// OAuthConnectionRepository manages external third-party OAuth provider connections.
//
// It links external identity provider subjects (e.g., Google) to internal user
// accounts and tracks federated login activity.
type OAuthConnectionRepository interface {
	// Connection lookups.
	GetByProviderAndSubject(ctx context.Context, exec transaction.Executor, provider OAuthProvider, subject string) (*OAuthConnection, error)
	GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*OAuthConnection, error)

	// Lifecycle management.
	Create(ctx context.Context, exec transaction.Executor, conn OAuthConnection) error
	UpdateLastLogin(ctx context.Context, exec transaction.Executor, id uuid.UUID, lastLoginAt time.Time) error
	DeleteByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error
}

// UserDeletionService coordinates domain cleanup when a user account is deleted.
//
// It ensures that user records and associated dependencies across modules are
// safely purged within a single transactional execution context.
type UserDeletionService interface {
	DeleteUserRecord(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error
}

// IssueSessionParams encapsulates contextual parameters needed to issue a session.
//
// It carries actor identifiers, client metadata (IP, user agent), and assigned
// role permissions to construct authenticated session states.
type IssueSessionParams struct {
	UserID     uuid.UUID
	AccountID  uuid.UUID
	UserAgent  *string
	IPAddress  *string
	StaffID    *uuid.UUID
	CustomerID *uuid.UUID
	Roles      []RoleCode
}

// IssueSessionResult represents the outcome of a successful session creation.
//
// It bundles the newly assigned session identifier along with the signed
// access and refresh token pair.
type IssueSessionResult struct {
	SessionID    uuid.UUID
	AccessToken  GeneratedToken
	RefreshToken GeneratedToken
}

// SessionIssuerService encapsulates business logic for issuing authentication sessions.
//
// It creates and persists the session record, generates the token pair, and
// manages refresh token hashing and storage.
type SessionIssuerService interface {
	Issue(ctx context.Context, params IssueSessionParams) (*IssueSessionResult, error)
}

// CreateChallengeParams defines parameters required to initiate an OTP challenge.
//
// It specifies the recipient email, the intended verification purpose, and the
// validity duration for the generated verification code.
type CreateChallengeParams struct {
	UserID   *uuid.UUID
	Email    string
	Purpose  OTPPurpose
	Duration time.Duration
}

// ChallengeService manages generation, persistence, and dispatch of OTP challenges.
//
// It handles secure code generation, expiration configuration, and delivery
// through communication channels such as email.
type ChallengeService interface {
	CreateAndSend(ctx context.Context, params CreateChallengeParams) (*uuid.UUID, error)
}

// CustomerRepository provides access to customer profile persistence in auth flows.
//
// It supports basic customer lifecycle operations required during registration,
// account verification, and account removal.
type CustomerRepository interface {
	Create(ctx context.Context, exec transaction.Executor, customer Customer) error
	GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*Customer, error)
	Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error
}
