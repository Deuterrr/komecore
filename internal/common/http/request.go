package apphttp

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

func DecodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func ClientIP(r *http.Request) string {
	var ipVal string

	xff := r.Header.Get("X-Forwarded-For")
	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))

	if xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			xff = xff[:i]
		}
		ipVal = strings.TrimSpace(xff)
	} else if realIP != "" {
		ipVal = realIP
	} else {
		ipVal = r.RemoteAddr
	}

	if host, _, err := net.SplitHostPort(ipVal); err == nil {
		return host
	}
	return ipVal
}
