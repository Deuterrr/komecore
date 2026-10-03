package domain

import (
	"time"

	"github.com/google/uuid"
)

type OAuthProvider string

const (
	OAuthProviderGoogle OAuthProvider = "google"
)

type OAuthConnection struct {
	ID     uuid.UUID
	UserID uuid.UUID

	Provider OAuthProvider
	Subject  string

	Email *string

	LastLoginAt *time.Time

	CreatedAt time.Time
}
