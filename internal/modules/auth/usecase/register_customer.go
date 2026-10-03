package usecase

import (
	"context"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/infra/service"
	"komecore/internal/modules/auth/repository"
	userRepo "komecore/internal/modules/user/repository"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"
	mailer "komecore/pkg/mailer"
	otp "komecore/pkg/otp"

	"github.com/google/uuid"
)

type RegisterCustomerUsecase struct {
	executor     transaction.Executor
	transactor   transaction.Transactor
	accountRepo  repository.AccountRepository
	hasher       domain.PasswordHasher
	userRepo     userRepo.UserRepository
	customerRepo repository.CustomerRepository
	challengeSvc repository.ChallengeService
	auditLogger  applogger.AuditLogger
	sysLogger    applogger.Logger
}

func NewRegisterCustomerUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo repository.AccountRepository,
	hasher domain.PasswordHasher,
	userRepo userRepo.UserRepository,
	customerRepo repository.CustomerRepository,
	challengeRepo repository.VerificationChallengeRepository,
	otpGen otp.Generator,
	mailer mailer.Sender,
	auditLogger applogger.AuditLogger,
) *RegisterCustomerUsecase {
	var pwHasher repository.PasswordHasher
	if ph, ok := hasher.(repository.PasswordHasher); ok {
		pwHasher = ph
	}

	challengeSvc := service.NewChallengeService(
		transactor,
		challengeRepo,
		pwHasher,
		otpGen,
		mailer,
	)

	return &RegisterCustomerUsecase{
		executor:     executor,
		transactor:   transactor,
		accountRepo:  accountRepo,
		hasher:       hasher,
		userRepo:     userRepo,
		customerRepo: customerRepo,
		challengeSvc: challengeSvc,
		auditLogger:  auditLogger,
	}
}

func (u *RegisterCustomerUsecase) SetChallengeService(challengeSvc repository.ChallengeService) {
	u.challengeSvc = challengeSvc
}

func (u *RegisterCustomerUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type RegisterCustomerParams struct {
	Name     string
	Username string
	Email    string
	Password string
	Phone    *string
}

func (u *RegisterCustomerUsecase) Execute(ctx context.Context, params RegisterCustomerParams) (challengeID *uuid.UUID, err error) {
	now := appclock.Now()

	existUsr, err := u.userRepo.GetByUsername(ctx, u.executor, params.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check user: %w", err)
	}

	existAcc, err := u.accountRepo.GetByEmail(ctx, u.executor, params.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check account: %w", err)
	}
	if existAcc != nil {
		if existAcc.Password != "" {
			return nil, apperrors.NewConflict(domain.ErrAccountAlreadyExists.Error())
		}
	}

	if existUsr != nil {
		return nil, apperrors.NewConflict("username already exists")
	}

	var (
		userID     uuid.UUID
		accountID  uuid.UUID
		customerID uuid.UUID
	)

	if existAcc == nil {
		userID = uuid.New()
		accountID = uuid.New()
		customerID = uuid.New()

		hashedPassword, err := u.hasher.Hash(params.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		userProps := userRepo.CreateUserProps{
			ID:        userID,
			Name:      params.Name,
			Username:  params.Username,
			Phone:     params.Phone,
			CreatedAt: now,
		}

		acc := domain.Account{
			ID:        accountID,
			UserID:    userID,
			Email:     params.Email,
			Password:  hashedPassword,
			Status:    domain.AccountPending,
			Type:      domain.AccountTypeCustomer,
			CreatedAt: now,
		}

		cust := domain.Customer{
			ID:        customerID,
			UserID:    userID,
			CreatedAt: now,
		}

		if err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if err := u.userRepo.CreateUser(ctx, exec, userProps); err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
			if err := u.customerRepo.Create(ctx, exec, cust); err != nil {
				return fmt.Errorf("failed to create customer profile: %w", err)
			}
			if err := u.accountRepo.Create(ctx, exec, acc); err != nil {
				return fmt.Errorf("failed to create account: %w", err)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	if existAcc != nil {
		userID = existAcc.UserID
		accountID = existAcc.ID

		hashedPassword, err := u.hasher.Hash(params.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		if err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if err := u.accountRepo.UpdatePasswordByUserID(ctx, exec, userID, hashedPassword); err != nil {
				return fmt.Errorf("failed to update account password: %w", err)
			}

			userProps := userRepo.SaveProfileProps{
				UserID:    userID,
				Name:      &params.Name,
				Username:  &params.Username,
				Phone:     params.Phone,
				UpdatedAt: now,
			}
			if err := u.userRepo.SaveProfile(ctx, exec, userProps); err != nil {
				return fmt.Errorf("failed to update user profile: %w", err)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	chID, err := u.challengeSvc.CreateAndSend(ctx, repository.CreateChallengeParams{
		UserID:   &userID,
		Email:    params.Email,
		Purpose:  domain.OTPPurposeRegister,
		Duration: 15 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return chID, nil
}
