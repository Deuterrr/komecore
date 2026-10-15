package authsvc

import (
	"net/http"
	"slices"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	"komecore/internal/httpx"
	appmiddleware "komecore/internal/httpx/middleware"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"

	"github.com/google/uuid"
)

type authorizer struct{}

func NewAuthorizer() Authorizer {
	return &authorizer{}
}

func (s *authorizer) RequireAccountType(allowedTypes ...auth.AccountType) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			actor, ok := GetActor(r.Context())
			if !ok {
				authCtx, authOk := auth.GetAuthContext(r.Context())
				if !authOk {
					return apperror.NewUnauthorized("authentication required")
				}
				actor = ActorFromAuthContext(authCtx)
				r = r.WithContext(WithActor(r.Context(), actor))
			}

			if slices.Contains(allowedTypes, actor.Type) {
				return next(w, r)
			}

			return apperror.NewForbidden("insufficient account type")
		}
	}
}

func (s *authorizer) RequireStaffRole(allowedRoles ...auth.RoleCode) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			actor, ok := GetActor(r.Context())
			if !ok {
				authCtx, authOk := auth.GetAuthContext(r.Context())
				if !authOk {
					return apperror.NewUnauthorized("authentication required")
				}
				actor = ActorFromAuthContext(authCtx)
				r = r.WithContext(WithActor(r.Context(), actor))
			}

			if actor.Type != auth.AccountTypeStaff {
				return apperror.NewForbidden(auth.ErrStaffRequired.Error())
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
				return apperror.NewForbidden(auth.ErrInsufficientRole.Error())
			}

			return next(w, r)
		}
	}
}

func (s *authorizer) RequirePermission(permission string) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			actor, ok := GetActor(r.Context())
			if !ok {
				authCtx, authOk := auth.GetAuthContext(r.Context())
				if !authOk {
					return apperror.NewUnauthorized("authentication required")
				}
				actor = ActorFromAuthContext(authCtx)
				r = r.WithContext(WithActor(r.Context(), actor))
			}

			if actor.Type != auth.AccountTypeStaff {
				return apperror.NewForbidden(auth.ErrStaffRequired.Error())
			}

			if actor.HasPermission(uuid.Nil, permission) {
				return next(w, r)
			}

			return apperror.NewForbidden("insufficient permission")
		}
	}
}

func (s *authorizer) LoadActor(exec transaction.Executor) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			if _, ok := GetActor(r.Context()); ok {
				return next(w, r)
			}

			authCtx, ok := auth.GetAuthContext(r.Context())
			if !ok {
				return apperror.NewUnauthorized("authentication required")
			}

			actor := ActorFromAuthContext(authCtx)
			return next(w, r.WithContext(WithActor(r.Context(), actor)))
		}
	}
}

func (s *authorizer) OptionalLoadActor(exec transaction.Executor) appmiddleware.Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			if _, ok := GetActor(r.Context()); ok {
				return next(w, r)
			}

			authCtx, ok := auth.GetAuthContext(r.Context())
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
