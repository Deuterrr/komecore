package addressdomain

import (
	"time"

	"github.com/google/uuid"
)

type CustomerAddress struct {
	ID uuid.UUID

	CustomerID uuid.UUID

	ReceiverName string
	Phone        *string

	IsDefault bool

	Detail AddressDetail

	CreatedAt time.Time
	UpdatedAt *time.Time
}
