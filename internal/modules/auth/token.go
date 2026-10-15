package auth

import (
	"time"

	"github.com/google/uuid"
)

type GenerateTokenParams struct {
	UserID     uuid.UUID
	SessionID  uuid.UUID
	StaffID    *uuid.UUID
	CustomerID *uuid.UUID
	Roles      []RoleCode

	Type TokenType

	Duration time.Duration
}

type GeneratedToken struct {
	Token     string
	ExpiresAt time.Time
	Type      TokenType
}
