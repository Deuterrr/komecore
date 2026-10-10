package appmiddleware

import (
	"net/http"

	"komecore/internal/httpx"
)

func Chain(h httpx.AppHandler, mws ...Middleware) http.HandlerFunc {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		_ = h(w, r)
	}
}
