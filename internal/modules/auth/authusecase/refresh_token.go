package authusecase

import (
	"context"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authrepo"
	appclock "komecore/pkg/clock"
)

type RefreshTokenUsecase struct {
	executor         transaction.Executor
	transactor       transaction.Transactor
	tokenSvc         authrepo.TokenService
	tokenHasher      authrepo.TokenHasher
	sessionRepo      authrepo.SessionRepository
	refreshTokenRepo authrepo.RefreshTokenRepository
}

func NewRefreshTokenUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	tokenSvc authrepo.TokenService,
	tokenHasher authrepo.TokenHasher,
	sessionRepo authrepo.SessionRepository,
	refreshTokenRepo authrepo.RefreshTokenRepository,
) *RefreshTokenUsecase {
	return &RefreshTokenUsecase{
		executor:         executor,
		transactor:       transactor,
		tokenSvc:         tokenSvc,
		tokenHasher:      tokenHasher,
		sessionRepo:      sessionRepo,
		refreshTokenRepo: refreshTokenRepo,
	}
}

type RefreshTokenParams struct {
	RefreshToken string
	AccountType  authdomain.AccountType
}

type RefreshTokenResult struct {
	AccessToken  authrepo.GeneratedToken
	RefreshToken authrepo.GeneratedToken
}

func (u *RefreshTokenUsecase) Execute(
	ctx context.Context,
	params RefreshTokenParams,
) (*RefreshTokenResult, error) {
	if params.RefreshToken == "" {
		return nil, apperrors.NewUnauthorized("refresh token is required")
	}

	claims, err := u.tokenSvc.Validate(params.RefreshToken)
	if err != nil {
		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidToken.Error())
	}
	if claims.Type != authdomain.TokenTypeRefresh {
		return nil, apperrors.NewUnauthorized("invalid token type for refresh")
	}

	if params.AccountType == authdomain.AccountTypeStaff && claims.StaffID == nil {
		return nil, apperrors.NewForbidden("staff account required")
	}
	if params.AccountType == authdomain.AccountTypeCustomer && claims.StaffID != nil {
		return nil, apperrors.NewForbidden("customer account required")
	}

	session, err := u.sessionRepo.GetByID(ctx, u.executor, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil || session.RevokedAt != nil || session.ExpiresAt.Before(appclock.Now()) {
		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidSession.Error())
	}

	dbRefreshToken, err := u.refreshTokenRepo.GetBySessionID(ctx, u.executor, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	if dbRefreshToken == nil || dbRefreshToken.RevokedAt != nil || dbRefreshToken.ExpiresAt.Before(appclock.Now()) {
		return nil, apperrors.NewUnauthorized("refresh token is invalid or expired")
	}

	if !u.tokenHasher.Compare(dbRefreshToken.TokenHash, params.RefreshToken) {
		return nil, apperrors.NewUnauthorized("refresh token mismatch")
	}

	roleCodes := make([]authdomain.RoleCode, len(claims.Roles))
	for i, rCode := range claims.Roles {
		roleCodes[i] = authdomain.RoleCode(rCode)
	}

	newAccessTkn, err := u.tokenSvc.Generate(authrepo.GenerateTokenParams{
		UserID:     claims.UserID,
		SessionID:  claims.SessionID,
		StaffID:    claims.StaffID,
		CustomerID: claims.CustomerID,
		Roles:      roleCodes,
		Type:       authdomain.TokenTypeAccess,
		Duration:   30 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	newRefreshTkn, err := u.tokenSvc.Generate(authrepo.GenerateTokenParams{
		UserID:     claims.UserID,
		SessionID:  claims.SessionID,
		StaffID:    claims.StaffID,
		CustomerID: claims.CustomerID,
		Roles:      roleCodes,
		Type:       authdomain.TokenTypeRefresh,
		Duration:   7 * 24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	now := appclock.Now()
	session.ExpiresAt = now.Add(7 * 24 * time.Hour)
	session.LastActivityAt = &now

	newRefreshTknHashed := u.tokenHasher.Hash(newRefreshTkn.Token)
	dbRefreshToken.TokenHash = newRefreshTknHashed
	dbRefreshToken.ExpiresAt = now.Add(7 * 24 * time.Hour)

	if err := u.transactor.WithinTransaction(ctx, func(e transaction.Executor) error {
		if err := u.sessionRepo.Save(ctx, e, *session); err != nil {
			return fmt.Errorf("failed to save updated session: %w", err)
		}
		if err := u.refreshTokenRepo.Save(ctx, e, *dbRefreshToken); err != nil {
			return fmt.Errorf("failed to save updated refresh token: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &RefreshTokenResult{
		AccessToken:  newAccessTkn,
		RefreshToken: newRefreshTkn,
	}, nil
}
