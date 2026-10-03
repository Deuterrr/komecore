package repository

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/domain"

	"github.com/google/uuid"
)

type StaffRepository interface {
	Create(
		ctx context.Context,
		exec transaction.Executor,
		staff domain.Staff,
	) error

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.Staff, error)

	GetProfileByUserID(
		ctx context.Context,
		exec transaction.Executor,
		userID uuid.UUID,
	) (*domain.StaffProfile, error)

	FindStaff(
		ctx context.Context,
		exec transaction.Executor,
		params FindStaffParams,
	) ([]domain.StaffProfile, int, error)

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
	) (*domain.StaffMembership, error)

	GetByAccountIDAndStaffID(
		ctx context.Context,
		exec transaction.Executor,
		accountID uuid.UUID,
		staffID uuid.UUID,
	) (*domain.StaffMembership, error)

	ListRolesByAccountIDAndStaffID(
		ctx context.Context,
		exec transaction.Executor,
		accountID uuid.UUID,
		staffID uuid.UUID,
	) ([]domain.Role, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		membership domain.StaffMembership,
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
	) ([]domain.StaffAccountMember, error)
}

type RoleRepository interface {
	GetRolesByAccountAndStaff(
		ctx context.Context,
		exec transaction.Executor,
		accountID uuid.UUID,
		staffID uuid.UUID,
	) ([]domain.Role, error)

	GetByCode(
		ctx context.Context,
		exec transaction.Executor,
		code domain.RoleCode,
	) (*domain.Role, error)
}
