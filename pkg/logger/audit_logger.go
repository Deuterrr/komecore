package logger

import "context"

const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
	OutcomeBlocked = "blocked"
)

// AuditEvent describes a single security-relevant event to be recorded.
// Callers only describe what happened — actor, request, and IP data
// are extracted automatically from ctx using the Step 1 context helpers.
type AuditEvent struct {
	// Category classifies the event source.
	// Use one of the CategoryAudit / CategoryWAF constants.
	Category string

	// Action is the specific action performed, e.g. "login", "rule_matched".
	Action string

	// Resource is the type of entity the action was performed on,
	// e.g. "user", "order", "waf_rule".
	Resource string

	// ResourceID is the ID of the target entity. May be empty.
	ResourceID string

	// Outcome is the result of the action: "success", "failure", or "blocked".
	Outcome string

	// Metadata holds any extra structured context relevant to this event.
	Metadata map[string]any
}

// AuditLogger is the interface for recording security-relevant audit events.
//
// It is intentionally separate from Logger, system operational logs and
// audit accountability logs have different lifetimes, sinks, and consumers.
type AuditLogger interface {
	Log(ctx context.Context, event AuditEvent)
}

type logAuditLogger struct {
	logger Logger
}

// NewLogAuditLogger returns an AuditLogger that outputs audit events as structured logs.
func NewLogAuditLogger(logger Logger) AuditLogger {
	return &logAuditLogger{logger: logger}
}

func (l *logAuditLogger) Log(ctx context.Context, event AuditEvent) {
	if l.logger == nil {
		return
	}
	l.logger.Info(ctx, "audit_event: "+event.Action,
		Field{Key: "category", Value: event.Category},
		Field{Key: "action", Value: event.Action},
		Field{Key: "resource", Value: event.Resource},
		Field{Key: "resource_id", Value: event.ResourceID},
		Field{Key: "outcome", Value: event.Outcome},
	)
}
