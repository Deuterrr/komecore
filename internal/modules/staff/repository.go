package staff

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

var (
	StaffSortLatest pagination.SortKey = "latest"
	StaffSortModify pagination.SortKey = "modified"
)

type FindStaffParams struct {
	ID *uuid.UUID

	pagination.Pagination
	pagination.Sorts
}

type Repository interface {
	Create(ctx context.Context, exec transaction.Executor, staff Staff) error

	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Staff, error)
	GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*StaffProfile, error)

	FindStaff(ctx context.Context, exec transaction.Executor, params FindStaffParams) ([]StaffProfile, int, error)

	Update(ctx context.Context, exec transaction.Executor, staffID uuid.UUID, name string, logoUrl *string, bannerUrl *string) error

	Delete(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error
}

type MembershipRepository interface {
	GetByAccountID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID) (*StaffMembership, error)
	GetByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID, staffID uuid.UUID) (*StaffMembership, error)

	ListAccountsByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) ([]StaffAccountMember, error)
	ListRolesByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID, staffID uuid.UUID) ([]Role, error)

	Save(ctx context.Context, exec transaction.Executor, membership StaffMembership) error

	DeleteByAccountID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID) error
	DeleteByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID, staffID uuid.UUID) error
	DeleteByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error
}

type RoleRepository interface {
	GetRolesByAccountAndStaff(ctx context.Context, exec transaction.Executor, accountID uuid.UUID, staffID uuid.UUID) ([]Role, error)
	GetByCode(ctx context.Context, exec transaction.Executor, code RoleCode) (*Role, error)
}

type AnalyticsRepository interface {
	GetRevenueOverTime(ctx context.Context, exec transaction.Executor, interval string, since time.Time) ([]RevenueBucket, error)
	GetOrderPipeline(ctx context.Context, exec transaction.Executor) ([]OrderStatusCount, error)
	GetTopProducts(ctx context.Context, exec transaction.Executor, limit int) ([]TopProduct, error)
}
