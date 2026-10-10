package authsvc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	appcookie "komecore/internal/httpx/cookie"
	commonmiddleware "komecore/internal/httpx/middleware"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authrepo"
	appclock "komecore/pkg/clock"
)

type jwtAuthenticator struct {
	tokenSvc         authrepo.TokenService
	sessionRepo      authrepo.SessionRepository
	tokenHasher      authrepo.TokenHasher
	refreshTokenRepo authrepo.RefreshTokenRepository
}

func NewJWTAuthenticator(
	tokenSvc authrepo.TokenService,
	sessionRepo authrepo.SessionRepository,
	tokenHasher authrepo.TokenHasher,
	refreshTokenRepo authrepo.RefreshTokenRepository,
) authrepo.Authenticator {
	return &jwtAuthenticator{
		tokenSvc:         tokenSvc,
		sessionRepo:      sessionRepo,
		tokenHasher:      tokenHasher,
		refreshTokenRepo: refreshTokenRepo,
	}
}

func extractToken(r *http.Request, cookie appcookie.CookieName) string {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "bearer ") {
			return strings.TrimSpace(authHeader[7:])
		}
	}

	if cookie != "" {
		if val, err := appcookie.Extract(r, cookie); err == nil && val != "" {
			return val
		}
	}

	return ""
}

func isValidCookieForAuth(
	cookie appcookie.CookieName,
	authCtx *authdomain.AuthContext,
) bool {
	if authCtx == nil {
		return false
	}
	switch cookie {
	case appcookie.CookieAccessToken:
		return authCtx.CustomerID != nil || authCtx.StaffID == nil
	case appcookie.CookieStaffAccessToken:
		return authCtx.StaffID != nil
	default:
		return true
	}
}

func (aM *jwtAuthenticator) RequireAuth(
	exec transaction.Executor,
	tran transaction.Transactor,
	cookie appcookie.CookieName,
) commonmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			token := extractToken(r, cookie)
			if token == "" {
				return apperror.NewUnauthorized(authdomain.ErrAuthenticationRequired.Error())
			}

			authCtx, err := aM.authenticate(r.Context(), exec, token)
			if err != nil {
				return apperror.NewUnauthorized(authdomain.ErrAuthenticationRequired.Error())
			}

			if !isValidCookieForAuth(cookie, authCtx) {
				return apperror.NewUnauthorized(authdomain.ErrAuthenticationRequired.Error())
			}

			ctx := authdomain.WithAuthContext(r.Context(), authCtx)
			ctx = WithActor(ctx, ActorFromAuthContext(authCtx))

			return next(w, r.WithContext(ctx))
		}
	}
}

func (aM *jwtAuthenticator) RequireAnyAuth(
	exec transaction.Executor,
	tran transaction.Transactor,
	cookies ...appcookie.CookieName,
) commonmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			for _, cookie := range cookies {
				token := extractToken(r, cookie)
				if token == "" {
					continue
				}

				authCtx, err := aM.authenticate(r.Context(), exec, token)
				if err != nil {
					continue
				}

				if !isValidCookieForAuth(cookie, authCtx) {
					continue
				}

				ctx := authdomain.WithAuthContext(r.Context(), authCtx)
				ctx = WithActor(ctx, ActorFromAuthContext(authCtx))

				return next(w, r.WithContext(ctx))
			}

			return apperror.NewUnauthorized(authdomain.ErrAuthenticationRequired.Error())
		}
	}
}

func (aM *jwtAuthenticator) RequireMultiAuth(
	exec transaction.Executor,
	tran transaction.Transactor,
	cookies ...appcookie.CookieName,
) commonmiddleware.Middleware {
	// In KomeCore, multi-auth collision sniffing is eliminated.
	// Each request resolves to a single authoritative authenticated context.
	return aM.RequireAnyAuth(exec, tran, cookies...)
}

func (aM *jwtAuthenticator) OptionalAuth(
	exec transaction.Executor,
	tran transaction.Transactor,
	cookies ...appcookie.CookieName,
) commonmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			for _, cookie := range cookies {
				token := extractToken(r, cookie)
				if token == "" {
					continue
				}

				authCtx, err := aM.authenticate(r.Context(), exec, token)
				if err != nil {
					continue
				}

				if !isValidCookieForAuth(cookie, authCtx) {
					continue
				}

				ctx := authdomain.WithAuthContext(r.Context(), authCtx)
				ctx = WithActor(ctx, ActorFromAuthContext(authCtx))

				return next(w, r.WithContext(ctx))
			}

			return next(w, r)
		}
	}
}

func (aM *jwtAuthenticator) authenticate(
	ctx context.Context,
	exec transaction.Executor,
	token string,
) (*authdomain.AuthContext, error) {
	claims, err := aM.tokenSvc.Validate(token)
	if err != nil {
		return nil, apperror.NewUnauthorized(authdomain.ErrInvalidToken.Error())
	}
	if claims.Type != authdomain.TokenTypeAccess {
		return nil, apperror.NewUnauthorized(authdomain.ErrInvalidToken.Error())
	}

	session, err := aM.sessionRepo.GetByID(ctx, exec, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}
	if session == nil ||
		session.UserID != claims.UserID ||
		session.RevokedAt != nil ||
		session.ExpiresAt.Before(appclock.Now()) {
		return nil, apperror.NewUnauthorized(authdomain.ErrInvalidSession.Error())
	}

	accType := authdomain.AccountTypeCustomer
	if claims.StaffID != nil {
		accType = authdomain.AccountTypeStaff
	}

	return &authdomain.AuthContext{
		UserID:          claims.UserID,
		SessionID:       claims.SessionID,
		TokenType:       claims.Type,
		AccountType:     accType,
		IsAuthenticated: true,
		StaffID:         claims.StaffID,
		CustomerID:      claims.CustomerID,
		Roles:           claims.Roles,
	}, nil
}
