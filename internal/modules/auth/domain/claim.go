package domain

import (
	"time"

	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// TokenClaims represents the claims extracted from an authentication token.
//
// This type acts as the bridge between the token layer and the application.
// It mirrors the identity and authorization information stored within a
// signed token, such as a JWT, and is primarily used during token
// validation, parsing, generation, and authentication middleware.
//
// TokenClaims should not be passed directly to business use cases.
// Instead, it should be translated into an AuthContext after the token
// has been successfully validated.
type TokenClaims struct {
	UserID     uuid.UUID
	SessionID  uuid.UUID
	StaffID    *uuid.UUID
	CustomerID *uuid.UUID

	Type TokenType

	Roles []string

	IssuedAt  time.Time
	ExpiresAt time.Time
}
