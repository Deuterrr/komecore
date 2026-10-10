package appmiddleware

import (
	"komecore/internal/httpx"
)

type Middleware func(httpx.AppHandler) httpx.AppHandler
