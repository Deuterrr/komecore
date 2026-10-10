package authsvc

import (
	"context"
	"fmt"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authrepo"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type sessionIssuerImpl struct {
	transactor       transaction.Transactor
	tokenSvc         authrepo.TokenService
	tokenHasher      authrepo.TokenHasher
	sessionRepo      authrepo.SessionRepository
	refreshTokenRepo authrepo.RefreshTokenRepository
	accountRepo      authrepo.AccountRepository
}

func NewSessionIssuerService(
	transactor transaction.Transactor,
	tokenSvc authrepo.TokenService,
	tokenHasher authrepo.TokenHasher,
	sessionRepo authrepo.SessionRepository,
	refreshTokenRepo authrepo.RefreshTokenRepository,
	accountRepo authrepo.AccountRepository,
) authrepo.SessionIssuerService {
	return &sessionIssuerImpl{
		transactor:       transactor,
		tokenSvc:         tokenSvc,
		tokenHasher:      tokenHasher,
		sessionRepo:      sessionRepo,
		refreshTokenRepo: refreshTokenRepo,
		accountRepo:      accountRepo,
	}
}

func (s *sessionIssuerImpl) Issue(
	ctx context.Context,
	params authrepo.IssueSessionParams,
) (*authrepo.IssueSessionResult, error) {
	now := appclock.Now()

	session := authdomain.Session{
		ID:        uuid.New(),
		UserID:    params.UserID,
		UserAgent: params.UserAgent,
		IPAddress: params.IPAddress,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	accessTkn, err := s.tokenSvc.Generate(authrepo.GenerateTokenParams{
		UserID:     params.UserID,
		SessionID:  session.ID,
		StaffID:    params.StaffID,
		CustomerID: params.CustomerID,
		Roles:      params.Roles,
		Type:       authdomain.TokenTypeAccess,
		Duration:   30 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshTkn, err := s.tokenSvc.Generate(authrepo.GenerateTokenParams{
		UserID:     params.UserID,
		SessionID:  session.ID,
		StaffID:    params.StaffID,
		CustomerID: params.CustomerID,
		Roles:      params.Roles,
		Type:       authdomain.TokenTypeRefresh,
		Duration:   7 * 24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshTknHashed := s.tokenHasher.Hash(refreshTkn.Token)
	refreshTknDomain := authdomain.RefreshToken{
		ID:        uuid.New(),
		SessionID: session.ID,
		TokenHash: refreshTknHashed,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.sessionRepo.Save(ctx, exec, session); err != nil {
			return fmt.Errorf("failed to save session: %w", err)
		}
		if err := s.refreshTokenRepo.Save(ctx, exec, refreshTknDomain); err != nil {
			return fmt.Errorf("failed to save refresh token: %w", err)
		}
		if s.accountRepo != nil && params.AccountID != uuid.Nil {
			if err := s.accountRepo.UpdateLastLoginAt(ctx, exec, params.AccountID, now); err != nil {
				return fmt.Errorf("failed to update last login at: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &authrepo.IssueSessionResult{
		SessionID:    session.ID,
		AccessToken:  accessTkn,
		RefreshToken: refreshTkn,
	}, nil
}
