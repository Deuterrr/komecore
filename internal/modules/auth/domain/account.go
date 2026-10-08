package domain

import (
	"time"

	"komecore/internal/common/authctx"
	query "komecore/internal/shared/query"

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
