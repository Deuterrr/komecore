package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleCustomer   Role = "customer"
	RoleStaff      Role = "staff"
	RoleStaffAdmin Role = "staff_admin"
)

// User represents shared profile information used across account types.
type User struct {
	ID       uuid.UUID
	Name     string
	Username string
	Phone    *string
	Role     Role

	AvatarURL *string

	CreatedAt   time.Time
	UpdatedAt   *time.Time
	LastLoginAt *time.Time
}

type CustomerProfile struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Username    string
	Phone       *string
	AvatarURL   *string
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

type StaffProfile struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Username    string
	Phone       *string
	AvatarURL   *string
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}
