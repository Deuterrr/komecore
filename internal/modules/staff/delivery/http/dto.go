package http

import (
	"time"

	"github.com/google/uuid"
)

type addStaffAccountRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type createStaffRequest struct {
	Name        string  `json:"name"`
	Username    string  `json:"username"`
	Description *string `json:"description"`
	LogoUrl     *string `json:"logo_url"`
	BannerUrl   *string `json:"banner_url"`
}

// staffResponse is used in list endpoints.
type staffResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Phone     *string   `json:"phone"`
	AvatarURL *string   `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

// staffProfileResponse is the dedicated detail response used in
// the profile endpoint.
type staffProfileResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Phone     *string   `json:"phone"`
	AvatarURL *string   `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

type updateStaffRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	LogoUrl     *string `json:"logo_url"`
	BannerUrl   *string `json:"banner_url"`
}

type staffAccountRoleResponse struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

type staffAccountResponse struct {
	AccountID   uuid.UUID                `json:"account_id"`
	UserID      uuid.UUID                `json:"user_id"`
	Email       string                   `json:"email"`
	Name        string                   `json:"name"`
	Username    string                   `json:"username"`
	Phone       *string                  `json:"phone"`
	AvatarURL   *string                  `json:"avatar_url"`
	Role        staffAccountRoleResponse `json:"role"`
	LastLoginAt *time.Time               `json:"last_login_at"`
	CreatedAt   time.Time                `json:"created_at"`
}

type listStaffAccountsResponse struct {
	StaffID  uuid.UUID              `json:"staff_id"`
	Total    int                    `json:"total"`
	Accounts []staffAccountResponse `json:"accounts"`
}
