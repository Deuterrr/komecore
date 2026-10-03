package domain

import (
	"time"

	"github.com/google/uuid"
)

type RoleCode string

const (
	RoleStaffAdmin RoleCode = "staff_admin"
	RoleStaff      RoleCode = "staff"
)

type Role struct {
	ID   uuid.UUID
	Code RoleCode
	Name string
}

type StaffMembership struct {
	ID        uuid.UUID
	StaffID   uuid.UUID
	AccountID uuid.UUID
	RoleID    uuid.UUID
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

type StaffAccountMember struct {
	ID          uuid.UUID
	AccountID   uuid.UUID
	UserID      uuid.UUID
	Email       string
	Name        string
	Username    string
	Phone       *string
	AvatarURL   *string
	Role        Role
	CreatedBy   uuid.UUID
	LastLoginAt *time.Time
	CreatedAt   time.Time
}
