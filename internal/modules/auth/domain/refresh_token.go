package domain

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	SessionID uuid.UUID

	TokenHash string

	ExpiresAt time.Time
	RevokedAt *time.Time

	CreatedAt time.Time
}
