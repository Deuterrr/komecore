package apphttp

import "net/http"

const (
	// HeaderAccountType is the header key used to specify the client application identity (customer vs staff).
	HeaderAccountType = "X-Account-Type"
)

// GetHeader returns the value of the specified header key from the request.
func GetHeader(r *http.Request, key string) string {
	if r == nil {
		return ""
	}
	return r.Header.Get(key)
}

// GetAccountTypeHeader returns the X-Account-Type header value from the HTTP request.
func GetAccountTypeHeader(r *http.Request) string {
	return GetHeader(r, HeaderAccountType)
}
