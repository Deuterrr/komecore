package authsvc

import (
	"net/http"
	"slices"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	appmiddleware "komecore/internal/common/middleware"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/common/authctx"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authrepo"

	"github.com/google/uuid"
)

type authorizer struct{}

func NewAuthorizer() authrepo.Authorizer {
	return &authorizer{}
}

func (s *authorizer) RequireAccountType(allowedTypes ...authdomain.AccountType) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			actor, ok := GetActor(r.Context())
			if !ok {
				authCtx, authOk := authdomain.GetAuthContext(r.Context())
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

func (s *authorizer) RequireStaffRole(allowedRoles ...authdomain.RoleCode) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			actor, ok := GetActor(r.Context())
			if !ok {
				authCtx, authOk := authdomain.GetAuthContext(r.Context())
				if !authOk {
					return apperrors.NewUnauthorized("authentication required")
				}
				actor = ActorFromAuthContext(authCtx)
				r = r.WithContext(WithActor(r.Context(), actor))
			}

			if actor.Type != authdomain.AccountTypeStaff {
				return apperrors.NewForbidden(authdomain.ErrStaffRequired.Error())
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
				return apperrors.NewForbidden(authdomain.ErrInsufficientRole.Error())
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
				authCtx, authOk := authdomain.GetAuthContext(r.Context())
				if !authOk {
					return apperrors.NewUnauthorized("authentication required")
				}
				actor = ActorFromAuthContext(authCtx)
				r = r.WithContext(WithActor(r.Context(), actor))
			}

			if actor.Type != authdomain.AccountTypeStaff {
				return apperrors.NewForbidden(authdomain.ErrStaffRequired.Error())
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

			authCtx, ok := authdomain.GetAuthContext(r.Context())
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

			authCtx, ok := authdomain.GetAuthContext(r.Context())
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
