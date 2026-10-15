package authusecase

import (
	"context"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/auth/authsvc"
	"komecore/internal/modules/staff"
	applogger "komecore/pkg/logger"
)

type LoginStaffUsecase struct {
	executor       transaction.Executor
	transactor     transaction.Transactor
	accountRepo    auth.AccountRepository
	pwHasher       auth.PasswordHasher
	staffRepo      staff.Repository
	membershipRepo staff.MembershipRepository
	sessionIssuer  auth.SessionIssuerService
	auditLogger    applogger.AuditLogger
	sysLogger      applogger.Logger
}

func NewLoginStaffUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo auth.AccountRepository,
	pwHasher auth.PasswordHasher,
	tokenHasher auth.TokenHasher,
	tokenSvc auth.TokenService,
	sessionRepo auth.SessionRepository,
	refreshTokenRepo auth.RefreshTokenRepository,
	staffRepo staff.Repository,
	membershipRepo staff.MembershipRepository,
	auditLogger applogger.AuditLogger,
) *LoginStaffUsecase {
	sessionIssuer := authsvc.NewSessionIssuerService(
		transactor,
		tokenSvc,
		tokenHasher,
		sessionRepo,
		refreshTokenRepo,
		accountRepo,
	)

	return &LoginStaffUsecase{
		executor:       executor,
		transactor:     transactor,
		accountRepo:    accountRepo,
		pwHasher:       pwHasher,
		staffRepo:      staffRepo,
		membershipRepo: membershipRepo,
		sessionIssuer:  sessionIssuer,
		auditLogger:    auditLogger,
	}
}

func (u *LoginStaffUsecase) SetSessionIssuer(sessionIssuer auth.SessionIssuerService) {
	u.sessionIssuer = sessionIssuer
}

func (u *LoginStaffUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type LoginStaffParams struct {
	UserAgent *string
	IPAddress *string
	Email     string
	Password  string
}

func (u *LoginStaffUsecase) Execute(ctx context.Context, input LoginStaffParams) (result *LoginEmailResult, err error) {
	existing, err := u.accountRepo.GetByEmail(ctx, u.executor, input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if existing == nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, auth.ErrInvalidCredentials.Error())
	}
	if existing.Type != auth.AccountTypeStaff {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, auth.ErrInvalidCredentials.Error())
	}
	if existing.Status != auth.AccountActive {
		return nil, apperror.NewForbidden(auth.ErrEmailNotVerified.Error())
	}

	if err := u.pwHasher.Compare(existing.Password, input.Password); err != nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, auth.ErrInvalidCredentials.Error())
	}

	memberStaff, err := u.membershipRepo.GetByAccountID(ctx, u.executor, existing.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve membership: %w", err)
	}
	if memberStaff == nil {
		return nil, apperror.NewUnauthorizedWithCode(apperror.CodeAuthInvalidCredentials, auth.ErrInvalidCredentials.Error())
	}

	roles, err := u.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, u.executor,
		existing.ID,
		memberStaff.StaffID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve roles: %w", err)
	}
	if roles == nil {
		return nil, apperror.NewForbidden("no role associated with this account")
	}

	roleCodes := make([]auth.RoleCode, len(roles))
	for i, r := range roles {
		roleCodes[i] = auth.RoleCode(r.Code)
	}

	sessionRes, err := u.sessionIssuer.Issue(ctx, auth.IssueSessionParams{
		UserID:    existing.UserID,
		AccountID: existing.ID,
		UserAgent: input.UserAgent,
		IPAddress: input.IPAddress,
		StaffID:   &memberStaff.StaffID,
		Roles:     roleCodes,
	})
	if err != nil {
		return nil, err
	}

	return &LoginEmailResult{
		AccessToken:  sessionRes.AccessToken,
		RefreshToken: sessionRes.RefreshToken,
	}, nil
}
