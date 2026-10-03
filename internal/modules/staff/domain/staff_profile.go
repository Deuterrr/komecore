package domain

import (
	"time"

	"github.com/google/uuid"
)

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
