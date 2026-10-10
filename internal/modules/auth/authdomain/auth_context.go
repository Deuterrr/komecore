package authdomain

import "komecore/internal/authctx"

// AuthContext re-exports authctx.AuthContext for backwards compatibility.
type AuthContext = authctx.AuthContext

var (
	WithAuthContext = authctx.WithAuthContext
	GetAuthContext  = authctx.GetAuthContext
)
