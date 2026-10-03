package service

import (
	"context"
	"net/http"
	"slices"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	appmiddleware "komecore/internal/common/middleware"
	transaction "komecore/internal/infra/transactor"
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

type actorContextKey struct{}

func WithActor(ctx context.Context, actor *domain.Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

func GetActor(ctx context.Context) (*domain.Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(*domain.Actor)
	return actor, ok
}

func ActorFromAuthContext(authCtx *domain.AuthContext) *domain.Actor {
	if authCtx == nil {
		return nil
	}

	accType := authCtx.AccountType
	if accType == "" {
		if authCtx.StaffID != nil {
			accType = domain.AccountTypeStaff
		} else {
			accType = domain.AccountTypeCustomer
		}
	}

	var roles []domain.Role
	for _, r := range authCtx.Roles {
		roles = append(roles, domain.Role{
			Code: domain.RoleCode(r),
			Name: r,
		})
	}

	return &domain.Actor{
		AccountID:  authCtx.UserID,
		Type:       accType,
		StaffID:    authCtx.StaffID,
		CustomerID: authCtx.CustomerID,
		Roles:      roles,
	}
}
