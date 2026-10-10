package userrepo

import (
	"time"

	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type FindUserParams struct {
	ID       *uuid.UUID
	Name     *string
	Username *string
	Email    *string

	pagination.Pagination
	pagination.Sorts
}

type CreateUserProps struct {
	ID        uuid.UUID
	Name      string
	Username  string
	Phone     *string
	AvatarURL *string
	CreatedAt time.Time
}

type SaveProfileProps struct {
	UserID    uuid.UUID
	Name      *string
	Phone     *string
	AvatarURL *string
	Username  *string
	UpdatedAt time.Time
}
