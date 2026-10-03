package http

import (
	"time"

	"github.com/google/uuid"
)

type userResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Username    string     `json:"username"`
	Phone       *string    `json:"phone"`
	AvatarURL   *string    `json:"avatar_url"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

type profileResponse struct {
	CustomerID  *uuid.UUID `json:"customer_id,omitempty"`
	StaffID     *uuid.UUID `json:"staff_id,omitempty"`
	UserID      uuid.UUID  `json:"user_id"`
	Name        string     `json:"Name"`
	Username    string     `json:"Username"`
	Phone       *string    `json:"Phone"`
	AvatarURL   *string    `json:"AvatarURL"`
	LastLoginAt *time.Time `json:"LastLoginAt"`
	CreatedAt   time.Time  `json:"CreatedAt"`
	UpdatedAt   *time.Time `json:"UpdatedAt"`
}

type updateProfileRequest struct {
	Name      *string `json:"name"`
	Phone     *string `json:"phone"`
	AvatarURL *string `json:"avatar_url"`
}
