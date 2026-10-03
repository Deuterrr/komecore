package domain

import (
	"time"

	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

type (
	AccountStatus string
	AccountType   string
)

const (
	AccountPending   AccountStatus = "pending"
	AccountActive    AccountStatus = "active"
	AccountSuspended AccountStatus = "suspended"
	AccountLocked    AccountStatus = "locked"
)

const (
	AccountTypeCustomer AccountType = "customer"
	AccountTypeStaff    AccountType = "staff"
)

var (
	AccountSortLatest    query.SortKey = "latest"
	AccountSortEmail     query.SortKey = "email"
	AccountSortStatus    query.SortKey = "status"
	AccountSortType      query.SortKey = "type"
	AccountSortLastLogin query.SortKey = "last_login"
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
