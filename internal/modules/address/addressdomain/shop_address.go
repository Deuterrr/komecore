package addressdomain

import (
	"time"

	"github.com/google/uuid"
)

type ShopAddress struct {
	ID uuid.UUID

	ShopID uuid.UUID

	Label string
	Phone *string

	IsActive bool

	Detail AddressDetail

	CreatedAt time.Time
	UpdatedAt *time.Time
}
