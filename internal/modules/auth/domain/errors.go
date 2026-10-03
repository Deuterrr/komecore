package domain

import "errors"

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrNotFoundAccount      = errors.New("account not found")
	ErrAccountAlreadyExists = errors.New("account already exists")
)

var (
	ErrEmailNotVerified = errors.New("email not verified")
)

var (
	ErrNotFoundChallenge = errors.New("challenge not found")
	ErrExpiredChallenge  = errors.New("challenge expired")
	ErrConsumedChallenge = errors.New("challenge already consumed")
	ErrVerifiedChallenge = errors.New("challenge already verified")
	ErrInvalidToken      = errors.New("invalid access token")
	ErrMaxAttemptReached = errors.New("max otp attempts reached")
	ErrInvalidOTP        = errors.New("invalid otp")
)

var (
	ErrInvalidSession  = errors.New("invalid session")
	ErrNotFoundSession = errors.New("not found session")
)

var (
	ErrAuthenticationRequired = errors.New("authentication required")
)

var (
	ErrStaffRequired    = errors.New("staff account required")
	ErrInsufficientRole = errors.New("insufficient role for operation")
)
