package service

import (
	"net/http"
	"slices"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	appmiddleware "komecore/internal/common/middleware"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/common/authctx"
	"komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/repository"

	"github.com/google/uuid"
)

type authorizer struct{}

func NewAuthorizer() repository.Authorizer {
	return &authorizer{}
}

func (s *authorizer) RequireAccountType(allowedTypes ...domain.AccountType) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			actor, ok := GetActor(r.Context())
			if !ok {
				authCtx, authOk := domain.GetAuthContext(r.Context())
				if !authOk {
					return apperrors.NewUnauthorized("authentication required")
				}
				actor = ActorFromAuthContext(authCtx)
				r = r.WithContext(WithActor(r.Context(), actor))
			}

			if slices.Contains(allowedTypes, actor.Type) {
				return next(w, r)
			}

			return apperrors.NewForbidden("insufficient account type")
		}
	}
}

func (s *authorizer) RequireStaffRole(allowedRoles ...domain.RoleCode) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			actor, ok := GetActor(r.Context())
			if !ok {
				authCtx, authOk := domain.GetAuthContext(r.Context())
				if !authOk {
					return apperrors.NewUnauthorized("authentication required")
				}
				actor = ActorFromAuthContext(authCtx)
				r = r.WithContext(WithActor(r.Context(), actor))
			}

			if actor.Type != domain.AccountTypeStaff {
				return apperrors.NewForbidden(domain.ErrStaffRequired.Error())
			}

			if actor.IsSuperAdmin() {
				return next(w, r)
			}

			allowed := false
			for _, actorRole := range actor.Roles {
				if slices.Contains(allowedRoles, actorRole.Code) {
					allowed = true
					break
				}
			}

			if !allowed {
				return apperrors.NewForbidden(domain.ErrInsufficientRole.Error())
			}

			return next(w, r)
		}
	}
}

func (s *authorizer) RequirePermission(permission string) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			actor, ok := GetActor(r.Context())
			if !ok {
				authCtx, authOk := domain.GetAuthContext(r.Context())
				if !authOk {
					return apperrors.NewUnauthorized("authentication required")
				}
				actor = ActorFromAuthContext(authCtx)
				r = r.WithContext(WithActor(r.Context(), actor))
			}

			if actor.Type != domain.AccountTypeStaff {
				return apperrors.NewForbidden(domain.ErrStaffRequired.Error())
			}

			if actor.HasPermission(uuid.Nil, permission) {
				return next(w, r)
			}

			return apperrors.NewForbidden("insufficient permission")
		}
	}
}

func (s *authorizer) LoadActor(exec transaction.Executor) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			if _, ok := GetActor(r.Context()); ok {
				return next(w, r)
			}

			authCtx, ok := domain.GetAuthContext(r.Context())
			if !ok {
				return apperrors.NewUnauthorized("authentication required")
			}

			actor := ActorFromAuthContext(authCtx)
			return next(w, r.WithContext(WithActor(r.Context(), actor)))
		}
	}
}

func (s *authorizer) OptionalLoadActor(exec transaction.Executor) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			if _, ok := GetActor(r.Context()); ok {
				return next(w, r)
			}

			authCtx, ok := domain.GetAuthContext(r.Context())
			if !ok {
				return next(w, r)
			}

			actor := ActorFromAuthContext(authCtx)
			return next(w, r.WithContext(WithActor(r.Context(), actor)))
		}
	}
}

var (
	WithActor            = authctx.WithActor
	GetActor             = authctx.GetActor
	ActorFromAuthContext = authctx.ActorFromAuthContext
)
