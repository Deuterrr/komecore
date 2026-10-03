package domain

import (
	"time"

	"github.com/google/uuid"
)

type Staff struct {
	ID     uuid.UUID
	UserID uuid.UUID

	CreatedAt time.Time
	UpdatedAt *time.Time
}

// -------

// id
// account_id

// staff_type
// (name: personal/business)

// display_name
// description

// email
// phone

// address
// city
// province
// postal_code
// latitude
// longitude

// logo_url
// banner_url

// is_active
// is_verified

// bank_name
// bank_account_name
// bank_account_number

// created_at
// updated_at
