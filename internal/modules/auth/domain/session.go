package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID     uuid.UUID
	UserID uuid.UUID

	UserAgent *string
	IPAddress *string

	ExpiresAt time.Time
	RevokedAt *time.Time

	CreatedAt      time.Time
	LastActivityAt *time.Time
}
