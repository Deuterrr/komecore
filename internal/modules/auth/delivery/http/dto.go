package http

import (
	"time"

	"github.com/google/uuid"
)

type signUpRequest struct {
	Name     string  `json:"name"`
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Phone    *string `json:"phone"`
}

type verifyAccountRequest struct {
	UserAgent   *string `json:"user_agent"`
	IPAddress   *string `json:"ip_address"`
	ChallengeID string  `json:"challenge_id"`
	OTP         string  `json:"otp"`
}

type signInEmailRequest struct {
	UserAgent  *string `json:"user_agent"`
	IPAddress  *string `json:"ip_address"`
	Email      string  `json:"email"`
	Password   string  `json:"password"`
	RememberMe bool    `json:"remember_me"`
}

type signInUsernameRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type signUpResponse struct {
	Message     string    `json:"message"`
	ChallengeID uuid.UUID `json:"challenge_id"`
}

type roleResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type permissionResponse struct {
	Code string `json:"code"`
}

type meResponse struct {
	AccountID       uuid.UUID            `json:"account_id"`
	AccountType     string               `json:"account_type"`
	IsAuthenticated bool                 `json:"is_authenticated"`
	AvatarURL       *string              `json:"avatar_url,omitempty"`
	StaffID         *uuid.UUID           `json:"staff_id,omitempty"`
	Roles           []roleResponse       `json:"roles,omitempty"`
	Permissions     []permissionResponse `json:"permissions,omitempty"`
	OAuthProvider   *string              `json:"oauth_provider,omitempty"`
	LastLoginAt     *time.Time           `json:"last_login_at,omitempty"`
}

type forgotPasswordCustomerRequest struct {
	Email string `json:"email"`
}

type forgotPasswordResponse struct {
	Message     string     `json:"message"`
	ChallengeID *uuid.UUID `json:"challenge_id,omitempty"`
}

type verifyPasswordResetRequest struct {
	ChallengeID string `json:"challenge_id"`
	OTP         string `json:"otp"`
}

type verifyPasswordResetResponse struct {
	Message     string    `json:"message"`
	ChallengeID uuid.UUID `json:"challenge_id"`
}

type resetPasswordRequest struct {
	ChallengeID string `json:"challenge_id"`
	NewPassword string `json:"new_password"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}
