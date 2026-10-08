package authusecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authsvc"
	"komecore/internal/modules/auth/authrepo"
	"komecore/internal/modules/staff/staffrepo"
	applogger "komecore/pkg/logger"
)

type LoginStaffUsecase struct {
	executor       transaction.Executor
	transactor     transaction.Transactor
	accountRepo    authrepo.AccountRepository
	pwHasher       authrepo.PasswordHasher
	staffRepo      staffrepo.StaffRepository
	membershipRepo staffrepo.StaffMembershipRepository
	sessionIssuer  authrepo.SessionIssuerService
	auditLogger    applogger.AuditLogger
	sysLogger      applogger.Logger
}

func NewLoginStaffUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo authrepo.AccountRepository,
	pwHasher authrepo.PasswordHasher,
	tokenHasher authrepo.TokenHasher,
	tokenSvc authrepo.TokenService,
	sessionRepo authrepo.SessionRepository,
	refreshTokenRepo authrepo.RefreshTokenRepository,
	staffRepo staffrepo.StaffRepository,
	membershipRepo staffrepo.StaffMembershipRepository,
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

func (u *LoginStaffUsecase) SetSessionIssuer(sessionIssuer authrepo.SessionIssuerService) {
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
		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidCredentials.Error())
	}
	if existing.Type != authdomain.AccountTypeStaff {
		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidCredentials.Error())
	}
	if existing.Status != authdomain.AccountActive {
		return nil, apperrors.NewForbidden(authdomain.ErrEmailNotVerified.Error())
	}

	if err := u.pwHasher.Compare(existing.Password, input.Password); err != nil {
		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidCredentials.Error())
	}

	memberStaff, err := u.membershipRepo.GetByAccountID(ctx, u.executor, existing.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve membership: %w", err)
	}
	if memberStaff == nil {
		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidCredentials.Error())
	}

	roles, err := u.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, u.executor,
		existing.ID,
		memberStaff.StaffID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve roles: %w", err)
	}
	if roles == nil {
		return nil, apperrors.NewForbidden("no role associated with this account")
	}

	roleCodes := make([]authdomain.RoleCode, len(roles))
	for i, r := range roles {
		roleCodes[i] = authdomain.RoleCode(r.Code)
	}

	sessionRes, err := u.sessionIssuer.Issue(ctx, authrepo.IssueSessionParams{
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
