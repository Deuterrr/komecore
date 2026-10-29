package staff

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

var (
	// Orders staff directory by descending creation timestamp.
	StaffSortLatest pagination.SortKey = "latest"
	// Orders staff directory by descending modification timestamp.
	StaffSortModify pagination.SortKey = "modified"
)

// FindStaffParams encapsulates pagination and filtering parameters for staff profile queries.
type FindStaffParams struct {
	ID *uuid.UUID

	pagination.Pagination
	pagination.Sorts
}

// Repository defines persistence operations for staff operator profiles, business branding,
// and administrative metadata within transaction boundaries.
type Repository interface {
	// Persists a new staff operator profile linked to a base user account.
	Create(ctx context.Context, exec transaction.Executor, staff Staff) error

	// Resolves a staff profile record by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Staff, error)
	// Resolves full staff profile details joined with account metadata by user identifier.
	GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*StaffProfile, error)

	// Queries paginated staff members matching administrative search filters.
	FindStaff(ctx context.Context, exec transaction.Executor, params FindStaffParams) ([]StaffProfile, int, error)

	// Updates staff display name, brand logo URL, or storefront banner URL.
	Update(ctx context.Context, exec transaction.Executor, staffID uuid.UUID, name string, logoUrl *string, bannerUrl *string) error

	// Soft-deletes a staff record, deactivating merchant workspace operations.
	Delete(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error
}

// MembershipRepository defines persistence operations for staff team memberships, account bindings,
// and RBAC role assignments across organizations.
type MembershipRepository interface {
	// Resolves staff organization membership associated with an authentication account.
	GetByAccountID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID) (*StaffMembership, error)
	// Looks up specific membership pairing between an account and a staff organization.
	GetByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID, staffID uuid.UUID) (*StaffMembership, error)

	// Queries all team members and login accounts bound to a staff organization.
	ListAccountsByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) ([]StaffAccountMember, error)
	// Queries RBAC permission roles granted to an account within a staff organization.
	ListRolesByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID, staffID uuid.UUID) ([]Role, error)

	// Persists staff organization membership linking account, staff ID, and assigned roles.
	Save(ctx context.Context, exec transaction.Executor, membership StaffMembership) error

	// Revokes all staff organization memberships held by an account upon role removal.
	DeleteByAccountID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID) error
	// Revokes an account's membership from a specific staff organization.
	DeleteByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID uuid.UUID, staffID uuid.UUID) error
	// Cascades deletion of all team memberships when a staff organization is deleted.
	DeleteByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error
}

// RoleRepository defines persistence operations for RBAC role definitions and system permission grants.
type RoleRepository interface {
	// Resolves active RBAC roles assigned to an account in a staff workspace.
	GetRolesByAccountAndStaff(ctx context.Context, exec transaction.Executor, accountID uuid.UUID, staffID uuid.UUID) ([]Role, error)
	// Resolves a role definition record by its unique system code (e.g. staff_admin, staff).
	GetByCode(ctx context.Context, exec transaction.Executor, code RoleCode) (*Role, error)
}

// AnalyticsRepository defines aggregation queries for business telemetry, revenue metrics, and pipeline KPIs.
type AnalyticsRepository interface {
	// Aggregates historical gross merchandise revenue into periodic time buckets (hour, day, month).
	GetRevenueOverTime(ctx context.Context, exec transaction.Executor, interval string, since time.Time) ([]RevenueBucket, error)
	// Aggregates active order counts and gross values grouped across lifecycle statuses.
	GetOrderPipeline(ctx context.Context, exec transaction.Executor) ([]OrderStatusCount, error)
	// Identifies top-selling catalog products ranked by sales volume and revenue.
	GetTopProducts(ctx context.Context, exec transaction.Executor, limit int) ([]TopProduct, error)
}
