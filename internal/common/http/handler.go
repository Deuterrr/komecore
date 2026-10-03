package apphttp

import (
	"net/http"
)

type AppHandler func(w http.ResponseWriter, r *http.Request) error
