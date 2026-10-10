package authdomain

import (
	"time"

	"komecore/internal/authctx"
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type (
	AccountStatus string
	AccountType   = authctx.AccountType
)

const (
	AccountPending   AccountStatus = "pending"
	AccountActive    AccountStatus = "active"
	AccountSuspended AccountStatus = "suspended"
	AccountLocked    AccountStatus = "locked"
)

const (
	AccountTypeCustomer = authctx.AccountTypeCustomer
	AccountTypeStaff    = authctx.AccountTypeStaff
)

var (
	AccountSortLatest    pagination.SortKey = "latest"
	AccountSortEmail     pagination.SortKey = "email"
	AccountSortStatus    pagination.SortKey = "status"
	AccountSortType      pagination.SortKey = "type"
	AccountSortLastLogin pagination.SortKey = "last_login"
)

// Account represents authentication credentials
// and access control for a user.
type Account struct {
	ID     uuid.UUID
	UserID uuid.UUID

	Email    string
	Password string

	Status AccountStatus
	Type   AccountType

	LastLoginAt *time.Time

	CreatedAt time.Time
	UpdatedAt *time.Time
}
