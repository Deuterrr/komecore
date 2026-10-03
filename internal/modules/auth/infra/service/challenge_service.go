package service

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/repository"
	appclock "komecore/pkg/clock"
	mailer "komecore/pkg/mailer"
	otp "komecore/pkg/otp"

	"github.com/google/uuid"
)

type challengeServiceImpl struct {
	transactor    transaction.Transactor
	challengeRepo repository.VerificationChallengeRepository
	pwHasher      repository.PasswordHasher
	otpGen        otp.Generator
	mailer        mailer.Sender
}

func NewChallengeService(
	transactor transaction.Transactor,
	challengeRepo repository.VerificationChallengeRepository,
	pwHasher repository.PasswordHasher,
	otpGen otp.Generator,
	mailer mailer.Sender,
) repository.ChallengeService {
	return &challengeServiceImpl{
		transactor:    transactor,
		challengeRepo: challengeRepo,
		pwHasher:      pwHasher,
		otpGen:        otpGen,
		mailer:        mailer,
	}
}

func (s *challengeServiceImpl) CreateAndSend(
	ctx context.Context,
	params repository.CreateChallengeParams,
) (*uuid.UUID, error) {
	now := appclock.Now()

	code, err := s.otpGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification code: %w", err)
	}
	codeHash, err := s.pwHasher.Hash(code)
	if err != nil {
		return nil, fmt.Errorf("failed to hash verification code: %w", err)
	}

	challengeID := uuid.New()
	challenge := domain.VerificationChallenge{
		ID:        challengeID,
		UserID:    params.UserID,
		Type:      domain.OTPTypeNumeric,
		Channel:   domain.OTPChannelEmail,
		Purpose:   params.Purpose,
		Target:    params.Email,
		CodeHash:  codeHash,
		ExpiresAt: now.Add(params.Duration),
		CreatedAt: now,
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.challengeRepo.Save(ctx, exec, challenge); err != nil {
			return fmt.Errorf("failed to save verification challenge: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	subject := "Verification Code"
	if params.Purpose == domain.OTPPurposePasswordReset {
		subject = "Password Reset Verification Code"
	}

	err = s.mailer.Send(mailer.SendInput{
		To:      params.Email,
		Subject: subject,
		Text:    fmt.Sprintf("Your verification code is: %s", code),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send verification email: %w", err)
	}

	return &challengeID, nil
}
