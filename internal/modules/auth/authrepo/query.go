package authrepo

import (
	"time"

	"komecore/internal/modules/auth/authdomain"

	"github.com/google/uuid"
)

type GenerateTokenParams struct {
	UserID     uuid.UUID
	SessionID  uuid.UUID
	StaffID    *uuid.UUID
	CustomerID *uuid.UUID
	Roles      []authdomain.RoleCode

	Type authdomain.TokenType

	Duration time.Duration
}

type GeneratedToken struct {
	Token     string
	ExpiresAt time.Time
	Type      authdomain.TokenType
}
