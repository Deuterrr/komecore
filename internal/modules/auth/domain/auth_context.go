package domain

import "komecore/internal/common/authctx"

// AuthContext re-exports authctx.AuthContext for backwards compatibility.
type AuthContext = authctx.AuthContext

var (
	WithAuthContext = authctx.WithAuthContext
	GetAuthContext  = authctx.GetAuthContext
)
