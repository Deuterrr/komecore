package logger

import "context"

type (
	// ContextKey is a typed key for values stored in context.Context.
	// Owning the keys here avoids circular imports between the logger
	// package and any auth/authorization module.
	ContextKey string
)

const (
	ContextKeyRequestID ContextKey = "request_id"
	ContextKeyActorID   ContextKey = "actor_id"
	ContextKeyClientIP  ContextKey = "client_ip"
)

// WithRequestID returns a new context with the given
// request ID stamped in.
//
// Called once per request in the Logging middleware.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, id)
}

// WithActorID returns a new context with the
// authenticated actor ID stamped in.
//
// Called by the authorization middleware after
// resolving the actor.
func WithActorID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ContextKeyActorID, id)
}

// WithClientIP returns a new context with the
// client IP stamped in.
//
// Called once per request in the Logging middleware.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ContextKeyClientIP, ip)
}

// --- Context extraction helpers ---

// RequestIDFromContext retrieves the request ID
// from the context.
//
// Returns an empty string if not set.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyRequestID).(string)
	return v
}

// ActorIDFromContext retrieves the actor ID
// from the context.
//
// Returns an empty string if not set.
func ActorIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyActorID).(string)
	return v
}

// ClientIPFromContext retrieves the client IP
// from the context.
//
// Returns an empty string if not set.
func ClientIPFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyClientIP).(string)
	return v
}

// FieldsFromContext returns a slice of log Fields
// populated from the context.
//
// Only non-empty values are included.
func FieldsFromContext(ctx context.Context) []Field {
	fields := make([]Field, 0, 3)

	if id := RequestIDFromContext(ctx); id != "" {
		fields = append(fields, Field{Key: "request_id", Value: id})
	}

	if id := ActorIDFromContext(ctx); id != "" {
		fields = append(fields, Field{Key: "actor_id", Value: id})
	}

	if ip := ClientIPFromContext(ctx); ip != "" {
		fields = append(fields, Field{Key: "client_ip", Value: ip})
	}

	return fields
}
