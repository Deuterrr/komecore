package appmiddleware

import "net/http"

// responseRecorder wraps http.ResponseWriter to capture the HTTP status code
// written by downstream handlers in the chain.
//
// Go's net/http package only exposes WriteHeader — there is no standard way
// to read back the status code after it has been sent. This wrapper intercepts
// WriteHeader so the Logging middleware can always emit status_code in its log
// entry, even on success paths where no explicit WriteHeader call is made.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// newResponseRecorder wraps w and defaults the status code to 200 OK,
// matching Go's implicit behaviour when a body is written without
// an explicit WriteHeader call.
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader intercepts the status code before delegating to the real writer.
func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}
