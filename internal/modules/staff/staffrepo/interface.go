package staffrepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/staffdomain"

	"github.com/google/uuid"
)

type StaffRepository interface {
	Create(
		ctx context.Context,
		exec transaction.Executor,
		staff staffdomain.Staff,
	) error

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*staffdomain.Staff, error)

	GetProfileByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) (*staffdomain.StaffProfile, error)

	FindStaff(
		ctx context.Context,
		exec transaction.Executor,
		params FindStaffParams,
	) ([]staffdomain.StaffProfile, int, error)

	Update(
		ctx context.Context,
		exec transaction.Executor,
		staffID uuid.UUID,
		name string,
		logoUrl *string,
		bannerUrl *string,
	) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		staffID uuid.UUID,
	) error
}

type StaffMembershipRepository interface {
	GetByAccountID(
		ctx context.Context,
		exec transaction.Executor,
		accountID uuid.UUID,
	) (*staffdomain.StaffMembership, error)

	GetByAccountIDAndStaffID(
		ctx context.Context,
		exec transaction.Executor,
		accountID uuid.UUID,
		staffID uuid.UUID,
	) (*staffdomain.StaffMembership, error)

	ListRolesByAccountIDAndStaffID(
		ctx context.Context,
		exec transaction.Executor,
		accountID uuid.UUID,
		staffID uuid.UUID,
	) ([]staffdomain.Role, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		membership staffdomain.StaffMembership,
	) error

	DeleteByAccountID(
		ctx context.Context,
		exec transaction.Executor,
		accountID uuid.UUID,
	) error

	DeleteByAccountIDAndStaffID(
		ctx context.Context,
		exec transaction.Executor,
		accountID uuid.UUID,
		staffID uuid.UUID,
	) error

	DeleteByStaffID(
		ctx context.Context,
		exec transaction.Executor,
		staffID uuid.UUID,
	) error

	ListAccountsByStaffID(
		ctx context.Context,
		exec transaction.Executor,
		staffID uuid.UUID,
	) ([]staffdomain.StaffAccountMember, error)
}

type RoleRepository interface {
	GetRolesByAccountAndStaff(
		ctx context.Context,
		exec transaction.Executor,
		accountID uuid.UUID,
		staffID uuid.UUID,
	) ([]staffdomain.Role, error)

	GetByCode(
		ctx context.Context,
		exec transaction.Executor,
		code staffdomain.RoleCode,
	) (*staffdomain.Role, error)
}
