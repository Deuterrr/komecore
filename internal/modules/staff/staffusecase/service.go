package staffusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/staffdomain"
	"komecore/internal/modules/staff/staffrepo"
	"komecore/internal/modules/user/userdomain"
	"komecore/internal/modules/user/userrepo"
	"komecore/internal/pagination"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type AddStaffAccountParams struct {
	ActorAccountID uuid.UUID
	ActorStaffID   uuid.UUID
	StaffID        uuid.UUID
	Email          string
	Password       string
}

type CreateStaffInput struct {
	Name        string
	Username    string
	Description *string
	LogoUrl     *string
	BannerUrl   *string
}

type FindStaffInput struct {
	Page  int
	Limit int
	ID    *uuid.UUID
	Name  *string
	Sort  string
}

type ListStaffAccountsParams struct {
	ActorAccountID uuid.UUID
	ActorStaffID   uuid.UUID
	StaffID        uuid.UUID
}

type UpdateStaffInput struct {
	ActorAccountID uuid.UUID
	ActorStaffID   uuid.UUID
	StaffID        uuid.UUID
	Name           string
	Description    *string
	LogoUrl        *string
	BannerUrl      *string
}

type DeleteStaffInput struct {
	ActorAccountID uuid.UUID
	ActorStaffID   uuid.UUID
	StaffID        uuid.UUID
}

type RemoveStaffAccountInput struct {
	ActorAccountID uuid.UUID
	ActorStaffID   uuid.UUID
	StaffID        uuid.UUID
	AccountID      uuid.UUID
}

type StaffRepository interface {
	Create(ctx context.Context, exec transaction.Executor, staff staffdomain.Staff) error
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*staffdomain.Staff, error)
	FindStaff(ctx context.Context, exec transaction.Executor, params staffrepo.FindStaffParams) ([]staffdomain.StaffProfile, int, error)
	Update(ctx context.Context, exec transaction.Executor, staffID uuid.UUID, name string, logoUrl *string, bannerUrl *string) error
	Delete(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error
}

type StaffMembershipRepository interface {
	GetByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) (*staffdomain.StaffMembership, error)
	ListRolesByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) ([]staffdomain.Role, error)
	Save(ctx context.Context, exec transaction.Executor, membership staffdomain.StaffMembership) error
	ListAccountsByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) ([]staffdomain.StaffAccountMember, error)
	DeleteByAccountIDAndStaffID(ctx context.Context, exec transaction.Executor, accountID, staffID uuid.UUID) error
	DeleteByStaffID(ctx context.Context, exec transaction.Executor, staffID uuid.UUID) error
}

type RoleRepository interface {
	GetByCode(ctx context.Context, exec transaction.Executor, code staffdomain.RoleCode) (*staffdomain.Role, error)
}

type UserRepository interface {
	GetByUsername(ctx context.Context, exec transaction.Executor, username string) (*userdomain.User, error)
	CreateUser(ctx context.Context, exec transaction.Executor, props userrepo.CreateUserProps) error
}

type AnalyticsRepository interface {
	GetRevenueOverTime(ctx context.Context, exec transaction.Executor, interval string, since time.Time) ([]staffdomain.RevenueBucket, error)
	GetOrderPipeline(ctx context.Context, exec transaction.Executor) ([]staffdomain.OrderStatusCount, error)
	GetTopProducts(ctx context.Context, exec transaction.Executor, limit int) ([]staffdomain.TopProduct, error)
}

type StaffService struct {
	executor            transaction.Executor
	transactor          transaction.Transactor
	staffRepo           StaffRepository
	membershipRepo      StaffMembershipRepository
	roleRepo            RoleRepository
	userRepo            UserRepository
	accountManager      AccountManager
	pwHasher            PasswordHasher
	userDeletionService UserDeletionService
	auditLogger         applogger.AuditLogger
	analyticsRepo       AnalyticsRepository
}

func NewStaffService(
	executor transaction.Executor,
	transactor transaction.Transactor,
	staffRepo StaffRepository,
	membershipRepo StaffMembershipRepository,
	roleRepo RoleRepository,
	userRepo UserRepository,
	accountManager AccountManager,
	pwHasher PasswordHasher,
	userDeletionService UserDeletionService,
	auditLogger applogger.AuditLogger,
) *StaffService {
	return &StaffService{
		executor:            executor,
		transactor:          transactor,
		staffRepo:           staffRepo,
		membershipRepo:      membershipRepo,
		roleRepo:            roleRepo,
		userRepo:            userRepo,
		accountManager:      accountManager,
		pwHasher:            pwHasher,
		userDeletionService: userDeletionService,
		auditLogger:         auditLogger,
	}
}

func (s *StaffService) WithAnalyticsRepository(repo AnalyticsRepository) *StaffService {
	s.analyticsRepo = repo
	return s
}

// AddStaffAccount adds an authentication account and membership to an existing staff entity.
func (s *StaffService) AddStaffAccount(ctx context.Context, input AddStaffAccountParams) error {
	if input.Email == "" {
		return apperror.NewBadRequest("email is required")
	}
	if input.Password == "" {
		return apperror.NewBadRequest("password is required")
	}

	actorMembership, err := s.membershipRepo.GetByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to verify actor membership: %w", err)
	}
	if actorMembership == nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "actor membership not found"},
		})
		return apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	actorRoles, err := s.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to retrieve actor roles: %w", err)
	}

	found := false
	for _, role := range actorRoles {
		if role.Code == staffdomain.RoleStaffAdmin {
			found = true
			break
		}
	}
	if !found {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "actor lacks admin role"},
		})
		return apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	existingAcc, err := s.accountManager.GetByEmail(ctx, s.executor, input.Email)
	if err != nil {
		return fmt.Errorf("failed to check existing account: %w", err)
	}
	if existingAcc != nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "email already exists"},
		})
		return apperror.NewConflict("an account with this email already exists")
	}

	existingStaff, err := s.staffRepo.GetByID(ctx, s.executor, input.StaffID)
	if err != nil {
		return fmt.Errorf("failed to check existing staff: %w", err)
	}
	if existingStaff == nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "staff not found"},
		})
		return apperror.NewNotFound(staffdomain.ErrNotFoundStaff.Error())
	}

	existingUserAcc, err := s.accountManager.GetByUserID(ctx, s.executor, existingStaff.UserID)
	if err != nil {
		return fmt.Errorf("failed to check existing staff account: %w", err)
	}
	if existingUserAcc != nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "staff user already has a bound account"},
		})
		return apperror.NewConflict("this staff entity already has a bound account (1 account per user limit)")
	}

	staffRole, err := s.roleRepo.GetByCode(ctx, s.executor, staffdomain.RoleStaff)
	if err != nil {
		return fmt.Errorf("failed to retrieve staff role: %w", err)
	}
	if staffRole == nil {
		return fmt.Errorf("staff role not found in database")
	}

	now := appclock.Now()
	newAccountID := uuid.New()

	hash, err := s.pwHasher.Hash(input.Password)
	if err != nil {
		return fmt.Errorf("failed to generate placeholder password: %w", err)
	}

	newAccountInput := CreateAccountInput{
		ID:        newAccountID,
		UserID:    existingStaff.UserID,
		Email:     input.Email,
		Password:  hash,
		CreatedAt: now,
	}

	newMembership := staffdomain.StaffMembership{
		ID:        uuid.New(),
		StaffID:   existingStaff.ID,
		AccountID: newAccountID,
		RoleID:    staffRole.ID,
		CreatedBy: input.ActorAccountID,
		CreatedAt: now,
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.accountManager.CreateStaffAccount(ctx, exec, newAccountInput); err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}
		if err := s.membershipRepo.Save(ctx, exec, newMembership); err != nil {
			return fmt.Errorf("failed to save membership: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "user_action",
		Action:     "add_staff_account",
		Resource:   "staff_account",
		ResourceID: newAccountID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata:   map[string]any{"staff_id": input.StaffID.String(), "email": input.Email},
	})

	return nil
}

// CreateStaff creates a new staff entity with associated user.
func (s *StaffService) CreateStaff(ctx context.Context, input CreateStaffInput) error {
	if input.Name == "" {
		return apperror.NewBadRequest("name is required")
	}
	if input.Username == "" {
		return apperror.NewBadRequest("username is required")
	}

	existingUser, err := s.userRepo.GetByUsername(ctx, s.executor, input.Username)
	if err != nil {
		return fmt.Errorf("failed to check existing username: %w", err)
	}
	if existingUser != nil {
		return apperror.NewConflict("a user with this username already exists")
	}

	now := appclock.Now()
	newUserID := uuid.New()
	newStaffID := uuid.New()

	newUser := userrepo.CreateUserProps{
		ID:        newUserID,
		Name:      input.Name,
		Username:  input.Username,
		Phone:     nil,
		CreatedAt: now,
	}
	if input.LogoUrl != nil {
		newUser.AvatarURL = input.LogoUrl
	}

	staff := staffdomain.Staff{
		ID:        newStaffID,
		UserID:    newUserID,
		CreatedAt: now,
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.userRepo.CreateUser(ctx, exec, newUser); err != nil {
			return fmt.Errorf("failed to create user for staff: %w", err)
		}
		if err := s.staffRepo.Create(ctx, exec, staff); err != nil {
			return fmt.Errorf("failed to create staff entity: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "user_action",
		Action:     "create_staff_profile",
		Resource:   "staff",
		ResourceID: staff.ID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata:   map[string]any{"name": input.Name, "username": input.Username, "user_id": newUserID.String()},
	})

	return nil
}

// FindStaff searches staff entities with pagination and sorting.
func (s *StaffService) FindStaff(ctx context.Context, input FindStaffInput) ([]staffdomain.StaffProfile, int, error) {
	var staffSortKeys = map[string]pagination.SortKey{
		"latest":   staffrepo.StaffSortLatest,
		"modified": staffrepo.StaffSortModify,
	}

	var sorts pagination.Sorts
	if input.Sort != "" {
		parts := strings.SplitSeq(input.Sort, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			subparts := strings.Split(part, ":")
			key := strings.TrimSpace(subparts[0])

			var dir pagination.SortDirection = pagination.SortDesc
			if len(subparts) > 1 {
				d := strings.ToLower(strings.TrimSpace(subparts[1]))
				if d == "asc" {
					dir = pagination.SortAsc
				}
			}

			sortKey, exists := staffSortKeys[key]
			if exists {
				sorts = append(sorts, pagination.Sort{
					By:        sortKey,
					Direction: dir,
				})
			}
		}
	}

	if len(sorts) == 0 {
		sorts = pagination.Sorts{
			{
				By:        staffrepo.StaffSortLatest,
				Direction: pagination.SortDesc,
			},
		}
	}

	params := staffrepo.FindStaffParams{
		ID: input.ID,
		Pagination: pagination.Pagination{
			Page:  input.Page,
			Limit: input.Limit,
		},
		Sorts: sorts,
	}

	staff, total, err := s.staffRepo.FindStaff(ctx, s.executor, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load staff: %w", err)
	}
	if len(staff) == 0 {
		return nil, 0, apperror.NewNotFound("staff not available at the moment")
	}

	return staff, total, nil
}

// ListStaffAccounts returns accounts associated with a staff entity.
func (s *StaffService) ListStaffAccounts(ctx context.Context, input ListStaffAccountsParams) ([]staffdomain.StaffAccountMember, error) {
	actorMembership, err := s.membershipRepo.GetByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify actor membership: %w", err)
	}
	if actorMembership == nil {
		return nil, apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	actorRoles, err := s.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve actor roles: %w", err)
	}

	foundAdmin := false
	for _, role := range actorRoles {
		if role.Code == staffdomain.RoleStaffAdmin {
			foundAdmin = true
			break
		}
	}
	if !foundAdmin {
		return nil, apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	staff, err := s.staffRepo.GetByID(ctx, s.executor, input.StaffID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve staff: %w", err)
	}
	if staff == nil {
		return nil, apperror.NewNotFound(staffdomain.ErrNotFoundStaff.Error())
	}

	accounts, err := s.membershipRepo.ListAccountsByStaffID(ctx, s.executor, staff.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list staff accounts: %w", err)
	}
	if accounts == nil {
		accounts = []staffdomain.StaffAccountMember{}
	}

	return accounts, nil
}

// UpdateStaff modifies a staff entity's details.
func (s *StaffService) UpdateStaff(ctx context.Context, input UpdateStaffInput) error {
	if input.Name == "" {
		return apperror.NewBadRequest("name is required")
	}

	actorMembership, err := s.membershipRepo.GetByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to verify actor membership: %w", err)
	}
	if actorMembership == nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "update_staff",
			Resource: "staff",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "reason": "actor membership not found"},
		})
		return apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	actorRoles, err := s.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to retrieve actor roles: %w", err)
	}

	foundAdmin := false
	for _, role := range actorRoles {
		if role.Code == staffdomain.RoleStaffAdmin {
			foundAdmin = true
			break
		}
	}
	if !foundAdmin {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "update_staff",
			Resource: "staff",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "reason": "actor lacks admin role"},
		})
		return apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	staff, err := s.staffRepo.GetByID(ctx, s.executor, input.StaffID)
	if err != nil {
		return fmt.Errorf("failed to retrieve staff: %w", err)
	}
	if staff == nil {
		return apperror.NewNotFound(staffdomain.ErrNotFoundStaff.Error())
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.staffRepo.Update(ctx, exec,
			input.StaffID,
			input.Name,
			input.LogoUrl,
			input.BannerUrl,
		); err != nil {
			return fmt.Errorf("failed to update staff: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "user_action",
		Action:     "update_staff",
		Resource:   "staff",
		ResourceID: input.StaffID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata:   map[string]any{"staff_id": input.StaffID.String(), "name": input.Name},
	})

	return nil
}

// DeleteStaff deletes a staff entity and its associated accounts/memberships.
func (s *StaffService) DeleteStaff(ctx context.Context, input DeleteStaffInput) error {
	actorMembership, err := s.membershipRepo.GetByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to verify actor membership: %w", err)
	}
	if actorMembership == nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "delete_staff",
			Resource: "staff",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "reason": "actor membership not found"},
		})
		return apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	actorRoles, err := s.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to retrieve actor roles: %w", err)
	}

	foundAdmin := false
	for _, role := range actorRoles {
		if role.Code == staffdomain.RoleStaffAdmin {
			foundAdmin = true
			break
		}
	}
	if !foundAdmin {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "delete_staff",
			Resource: "staff",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "reason": "actor lacks admin role"},
		})
		return apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	staff, err := s.staffRepo.GetByID(ctx, s.executor, input.StaffID)
	if err != nil {
		return fmt.Errorf("failed to retrieve staff: %w", err)
	}
	if staff == nil {
		return apperror.NewNotFound(staffdomain.ErrNotFoundStaff.Error())
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.staffRepo.Delete(ctx, exec, input.StaffID); err != nil {
			return fmt.Errorf("failed to soft-delete staff: %w", err)
		}
		if err := s.membershipRepo.DeleteByStaffID(ctx, exec, input.StaffID); err != nil {
			return fmt.Errorf("failed to delete staff memberships: %w", err)
		}
		if err := s.userDeletionService.DeleteUserRecord(ctx, exec, staff.UserID); err != nil {
			return fmt.Errorf("failed to soft-delete user and accounts for staff: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "user_action",
		Action:     "delete_staff",
		Resource:   "staff",
		ResourceID: input.StaffID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata:   map[string]any{"staff_id": input.StaffID.String(), "user_id": staff.UserID.String()},
	})

	return nil
}

// RemoveStaffAccount removes an account membership from a staff entity.
func (s *StaffService) RemoveStaffAccount(ctx context.Context, input RemoveStaffAccountInput) error {
	if input.ActorAccountID == input.AccountID {
		return apperror.NewBadRequest("cannot remove own account from staff")
	}

	actorMembership, err := s.membershipRepo.GetByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to verify actor membership: %w", err)
	}
	if actorMembership == nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "remove_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "account_id": input.AccountID.String(), "reason": "actor membership not found"},
		})
		return apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	actorRoles, err := s.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, s.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to retrieve actor roles: %w", err)
	}

	foundAdmin := false
	for _, role := range actorRoles {
		if role.Code == staffdomain.RoleStaffAdmin {
			foundAdmin = true
			break
		}
	}
	if !foundAdmin {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "remove_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "account_id": input.AccountID.String(), "reason": "actor lacks admin role"},
		})
		return apperror.NewForbidden(staffdomain.ErrInsufficientRole.Error())
	}

	staff, err := s.staffRepo.GetByID(ctx, s.executor, input.StaffID)
	if err != nil {
		return fmt.Errorf("failed to retrieve staff: %w", err)
	}
	if staff == nil {
		return apperror.NewNotFound(staffdomain.ErrNotFoundStaff.Error())
	}

	targetMembership, err := s.membershipRepo.GetByAccountIDAndStaffID(ctx, s.executor,
		input.AccountID,
		input.StaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to retrieve target membership: %w", err)
	}
	if targetMembership == nil {
		return apperror.NewNotFound("staff account membership not found")
	}

	targetAccount, err := s.accountManager.GetByID(ctx, s.executor, input.AccountID)
	if err != nil {
		return fmt.Errorf("failed to retrieve target account: %w", err)
	}
	if targetAccount == nil {
		return apperror.NewNotFound("target account not found")
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.membershipRepo.DeleteByAccountIDAndStaffID(ctx, exec,
			input.AccountID,
			input.StaffID,
		); err != nil {
			return fmt.Errorf("failed to delete staff membership: %w", err)
		}

		if err := s.accountManager.DeleteByUserID(ctx, exec, targetAccount.UserID); err != nil {
			return fmt.Errorf("failed to soft delete account: %w", err)
		}

		if err := s.accountManager.RevokeSessionsByUserID(ctx, exec, targetAccount.UserID); err != nil {
			return fmt.Errorf("failed to revoke sessions: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "user_action",
		Action:     "remove_staff_account",
		Resource:   "staff_account",
		ResourceID: input.AccountID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata:   map[string]any{"staff_id": input.StaffID.String(), "account_id": input.AccountID.String(), "user_id": targetAccount.UserID.String()},
	})

	return nil
}

// GetRevenueAnalytics retrieves time-bucketed revenue metrics for the specified range and interval.
func (s *StaffService) GetRevenueAnalytics(ctx context.Context, rangeParam, intervalParam string) (*staffdomain.RevenueAnalytics, error) {
	if s.analyticsRepo == nil {
		return &staffdomain.RevenueAnalytics{
			Range:    rangeParam,
			Interval: intervalParam,
			Buckets:  []staffdomain.RevenueBucket{},
		}, nil
	}

	interval := strings.ToLower(strings.TrimSpace(intervalParam))
	if interval == "" {
		interval = "day"
	}
	switch interval {
	case "day", "week", "month":
		// valid
	default:
		interval = "day"
	}

	rangeStr := strings.ToLower(strings.TrimSpace(rangeParam))
	if rangeStr == "" {
		rangeStr = "30d"
	}

	now := appclock.Now()
	var since time.Time
	switch rangeStr {
	case "7d":
		since = now.AddDate(0, 0, -7)
	case "30d":
		since = now.AddDate(0, 0, -30)
	case "90d":
		since = now.AddDate(0, 0, -90)
	case "1y":
		since = now.AddDate(-1, 0, 0)
	default:
		rangeStr = "30d"
		since = now.AddDate(0, 0, -30)
	}

	buckets, err := s.analyticsRepo.GetRevenueOverTime(ctx, s.executor, interval, since)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve revenue analytics: %w", err)
	}

	var totalRevenue int64
	var totalOrders int64
	for _, b := range buckets {
		totalRevenue += b.Revenue
		totalOrders += b.Count
	}

	return &staffdomain.RevenueAnalytics{
		Range:        rangeStr,
		Interval:     interval,
		TotalRevenue: totalRevenue,
		TotalOrders:  totalOrders,
		Buckets:      buckets,
	}, nil
}

// GetOrderPipelineAnalytics retrieves current order counts and revenue totals grouped by order status.
func (s *StaffService) GetOrderPipelineAnalytics(ctx context.Context) (*staffdomain.OrderPipelineAnalytics, error) {
	if s.analyticsRepo == nil {
		return &staffdomain.OrderPipelineAnalytics{
			Breakdown: []staffdomain.OrderStatusCount{},
		}, nil
	}

	counts, err := s.analyticsRepo.GetOrderPipeline(ctx, s.executor)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve order pipeline analytics: %w", err)
	}

	var totalOrders int64
	var totalAmount int64
	for _, c := range counts {
		totalOrders += c.Count
		totalAmount += c.TotalAmount
	}

	return &staffdomain.OrderPipelineAnalytics{
		TotalOrders: totalOrders,
		TotalAmount: totalAmount,
		Breakdown:   counts,
	}, nil
}

// GetTopProductsAnalytics retrieves highest revenue-generating settled products.
func (s *StaffService) GetTopProductsAnalytics(ctx context.Context, limit int) ([]staffdomain.TopProduct, error) {
	if s.analyticsRepo == nil {
		return []staffdomain.TopProduct{}, nil
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	products, err := s.analyticsRepo.GetTopProducts(ctx, s.executor, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve top products analytics: %w", err)
	}

	return products, nil
}
