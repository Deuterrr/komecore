package repository

import (
	"time"

	"komecore/internal/modules/auth/domain"

	"github.com/google/uuid"
)

type GenerateTokenParams struct {
	UserID     uuid.UUID
	SessionID  uuid.UUID
	StaffID    *uuid.UUID
	CustomerID *uuid.UUID
	Roles      []domain.RoleCode

	Type domain.TokenType

	Duration time.Duration
}

type GeneratedToken struct {
	Token     string
	ExpiresAt time.Time
	Type      domain.TokenType
}
