package appmiddleware

import (
	apphttp "komecore/internal/common/http"
)

type Middleware func(apphttp.AppHandler) apphttp.AppHandler
