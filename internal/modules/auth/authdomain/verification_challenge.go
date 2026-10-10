package authdomain

import (
	"time"

	"github.com/google/uuid"
)

type (
	OTPType    string
	OTPChannel string
	OTPPurpose string
)

const (
	OTPTypeNumeric   OTPType = "numeric"
	OTPTypeMagicLink OTPType = "magic_link"
)

const (
	OTPChannelEmail OTPChannel = "email"
	OTPChannelSMS   OTPChannel = "sms"
)

const (
	OTPPurposeRegister          OTPPurpose = "register"
	OTPPurposeLogin             OTPPurpose = "login"
	OTPPurposePasswordReset     OTPPurpose = "password_reset"
	OTPPurposeEmailVerification OTPPurpose = "email_verification"
)

type VerificationChallenge struct {
	ID uuid.UUID

	// Nullable because some challenges may exist
	// before a full account/user is created.
	//
	// Example:
	// pre-registration email verification.
	UserID *uuid.UUID

	Type OTPType

	// Defines delivery medium.
	//
	// Examples:
	// email
	// sms
	Channel OTPChannel

	// Defines business/authentication intent.
	//
	// Important because the same target may have
	// multiple active challenges for different flows.
	Purpose OTPPurpose

	// Raw delivery target.
	//
	// Examples:
	// user@email.com
	// +628123456789
	Target string

	// Never store raw OTP/token values.
	//
	// Always store hashed challenge secrets.
	CodeHash string

	// Absolute challenge expiration timestamp.
	ExpiresAt time.Time
	// Set when challenge successfully validated.
	VerifiedAt *time.Time

	// Set when challenge has been consumed/finalized.
	//
	// Important distinction:
	// verified != consumed
	//
	// Example:
	// verified OTP used to activate account.
	ConsumedAt *time.Time

	// Number of failed verification attempts.
	AttemptCount int

	CreatedAt time.Time
}
