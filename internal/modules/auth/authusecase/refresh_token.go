package authusecase

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	appclock "komecore/pkg/clock"
)

type RefreshTokenUsecase struct {
	executor         transaction.Executor
	transactor       transaction.Transactor
	tokenSvc         auth.TokenService
	tokenHasher      auth.TokenHasher
	sessionRepo      auth.SessionRepository
	refreshTokenRepo auth.RefreshTokenRepository
}

func NewRefreshTokenUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	tokenSvc auth.TokenService,
	tokenHasher auth.TokenHasher,
	sessionRepo auth.SessionRepository,
	refreshTokenRepo auth.RefreshTokenRepository,
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
	AccountType  auth.AccountType
}

type RefreshTokenResult struct {
	AccessToken  auth.GeneratedToken
	RefreshToken auth.GeneratedToken
}

func (u *RefreshTokenUsecase) Execute(
	ctx context.Context,
	params RefreshTokenParams,
) (*RefreshTokenResult, error) {
	if params.RefreshToken == "" {
		return nil, apperror.NewUnauthorized("refresh token is required")
	}

	claims, err := u.tokenSvc.Validate(params.RefreshToken)
	if err != nil {
		return nil, apperror.NewUnauthorized(auth.ErrInvalidToken.Error())
	}
	if claims.Type != auth.TokenTypeRefresh {
		return nil, apperror.NewUnauthorized("invalid token type for refresh")
	}

	if params.AccountType == auth.AccountTypeStaff && claims.StaffID == nil {
		return nil, apperror.NewForbidden("staff account required")
	}
	if params.AccountType == auth.AccountTypeCustomer && claims.StaffID != nil {
		return nil, apperror.NewForbidden("customer account required")
	}

	session, err := u.sessionRepo.GetByID(ctx, u.executor, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil || session.RevokedAt != nil || session.ExpiresAt.Before(appclock.Now()) {
		return nil, apperror.NewUnauthorized(auth.ErrInvalidSession.Error())
	}

	dbRefreshToken, err := u.refreshTokenRepo.GetBySessionID(ctx, u.executor, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	if dbRefreshToken == nil || dbRefreshToken.RevokedAt != nil || dbRefreshToken.ExpiresAt.Before(appclock.Now()) {
		return nil, apperror.NewUnauthorized("refresh token is invalid or expired")
	}

	if !u.tokenHasher.Compare(dbRefreshToken.TokenHash, params.RefreshToken) {
		return nil, apperror.NewUnauthorized("refresh token mismatch")
	}

	roleCodes := make([]auth.RoleCode, len(claims.Roles))
	for i, rCode := range claims.Roles {
		roleCodes[i] = auth.RoleCode(rCode)
	}

	newAccessTkn, err := u.tokenSvc.Generate(auth.GenerateTokenParams{
		UserID:     claims.UserID,
		SessionID:  claims.SessionID,
		StaffID:    claims.StaffID,
		CustomerID: claims.CustomerID,
		Roles:      roleCodes,
		Type:       auth.TokenTypeAccess,
		Duration:   30 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	newRefreshTkn, err := u.tokenSvc.Generate(auth.GenerateTokenParams{
		UserID:     claims.UserID,
		SessionID:  claims.SessionID,
		StaffID:    claims.StaffID,
		CustomerID: claims.CustomerID,
		Roles:      roleCodes,
		Type:       auth.TokenTypeRefresh,
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
