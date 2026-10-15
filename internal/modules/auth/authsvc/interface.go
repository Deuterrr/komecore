package authsvc

import (
	appcookie "komecore/internal/httpx/cookie"
	appmiddleware "komecore/internal/httpx/middleware"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
)

type Authenticator interface {
	RequireAuth(
		exec transaction.Executor,
		tran transaction.Transactor,
		cookie appcookie.CookieName,
	) appmiddleware.Middleware

	RequireAnyAuth(
		exec transaction.Executor,
		tran transaction.Transactor,
		cookies ...appcookie.CookieName,
	) appmiddleware.Middleware

	RequireMultiAuth(
		exec transaction.Executor,
		tran transaction.Transactor,
		cookies ...appcookie.CookieName,
	) appmiddleware.Middleware

	OptionalAuth(
		exec transaction.Executor,
		tran transaction.Transactor,
		cookies ...appcookie.CookieName,
	) appmiddleware.Middleware
}

type Authorizer interface {
	RequireAccountType(
		allowedTypes ...auth.AccountType,
	) appmiddleware.Middleware

	RequireStaffRole(
		allowedRoles ...auth.RoleCode,
	) appmiddleware.Middleware

	RequirePermission(
		permission string,
	) appmiddleware.Middleware

	LoadActor(
		exec transaction.Executor,
	) appmiddleware.Middleware

	OptionalLoadActor(
		exec transaction.Executor,
	) appmiddleware.Middleware
}
