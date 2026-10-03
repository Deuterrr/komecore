package domain

import (
	"context"

	"github.com/google/uuid"
)

// authContextKey identifies the request authentication context.
type authContextKey struct{}

var requestAuthContextKey authContextKey

// AuthContext represents the authenticated identity available to the
// application during a request lifecycle.
//
// This type is independent from any specific authentication mechanism
// and provides the minimum identity and authorization information
// required by application services and use cases.
//
// AuthContext is typically constructed by authentication middleware
// after a token has been validated and should be passed into business
// workflows instead of token-specific structures.
type AuthContext struct {
	UserID    uuid.UUID
	SessionID uuid.UUID

	StaffID    *uuid.UUID
	CustomerID *uuid.UUID

	AccountType AccountType

	TokenType TokenType

	IsAuthenticated bool

	Roles []string
}

func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, requestAuthContextKey, authCtx)
}

func GetAuthContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(requestAuthContextKey).(*AuthContext)
	if !ok {
		return nil, false
	}
	return authCtx, true
}
