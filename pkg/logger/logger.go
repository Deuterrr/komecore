package logger

import "context"

type (
	// LogLevel represents the severity of a log entry.
	//
	// Log levels are ordered from least to most severe:
	// ERROR, WARN, INFO, DEBUG.
	LogLevel string

	// LogCategory distinguishes operational system logs
	// from security-relevant audit and WAF event logs
	// when filtering in a log aggregator.
	LogCategory string

	// LogLayer identifies which architectural layer emitted a log entry.
	// Used as a standard field to separate middleware, business, and
	// infrastructure logs when filtering in a log aggregator.
	LogLayer string
)

const (
	LogLevelError LogLevel = "ERROR"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelDebug LogLevel = "DEBUG"

	CategorySystem LogCategory = "system"
	CategoryAudit  LogCategory = "audit"
	CategoryWAF    LogCategory = "waf"

	LayerMiddleware LogLayer = "middleware"
	LayerUsecase    LogLayer = "usecase"
	LayerInfra      LogLayer = "infra"
)

type Field struct {
	Key   string
	Value any
}

// This interface defines the methods for structured
// operational system logging.
//
// Implementations should log messages at the appropriate
// log level, with fields enriched from the context where available.
type Logger interface {
	Info(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Debug(ctx context.Context, msg string, fields ...Field)

	With(fields ...Field) Logger
}
