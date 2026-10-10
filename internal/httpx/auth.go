package httpx

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/authctx"

	"github.com/google/uuid"
)

// RequireAuth extracts and validates that the request context has an authenticated session.
// Returns 401 Unauthorized if missing or unauthenticated.
func RequireAuth(r *http.Request) (*authctx.AuthContext, error) {
	authCtx, ok := authctx.GetAuthContext(r.Context())
	if !ok || !authCtx.IsAuthenticated {
		return nil, apperror.NewUnauthorized("authentication required")
	}
	return authCtx, nil
}

// RequireCustomer extracts and validates that the request is authenticated by a customer.
// Returns 401 Unauthorized if unauthenticated, or 403 Forbidden if not a customer account.
func RequireCustomer(r *http.Request) (*authctx.AuthContext, uuid.UUID, error) {
	authCtx, err := RequireAuth(r)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if authCtx.CustomerID == nil {
		return nil, uuid.Nil, apperror.NewForbidden("customer account required")
	}
	return authCtx, *authCtx.CustomerID, nil
}

// RequireStaff extracts and validates that the request is authenticated by a staff member.
// Returns 401 Unauthorized if unauthenticated, or 403 Forbidden if not a staff account.
func RequireStaff(r *http.Request) (*authctx.AuthContext, uuid.UUID, error) {
	authCtx, err := RequireAuth(r)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if authCtx.StaffID == nil {
		return nil, uuid.Nil, apperror.NewForbidden("staff account required")
	}
	return authCtx, *authCtx.StaffID, nil
}
