package logger

import (
	"context"
	"log/slog"
	"os"
)

type slogLogger struct {
	log *slog.Logger
}

func NewSlogLogger(env string) Logger {
	if env == "development" {
		return &slogLogger{
			log: slog.New(slog.NewTextHandler(os.Stdout, nil)),
		}
	}

	return &slogLogger{
		log: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (l *slogLogger) Info(ctx context.Context, msg string, fields ...Field) {
	all := append(FieldsFromContext(ctx), fields...)
	l.log.InfoContext(ctx, msg, toAttrs(all)...)
}

func (l *slogLogger) Error(ctx context.Context, msg string, fields ...Field) {
	all := append(FieldsFromContext(ctx), fields...)
	l.log.ErrorContext(ctx, msg, toAttrs(all)...)
}

func (l *slogLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	all := append(FieldsFromContext(ctx), fields...)
	l.log.WarnContext(ctx, msg, toAttrs(all)...)
}

func (l *slogLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	all := append(FieldsFromContext(ctx), fields...)
	l.log.DebugContext(ctx, msg, toAttrs(all)...)
}

func (l *slogLogger) With(fields ...Field) Logger {
	return &slogLogger{
		log: l.log.With(toAttrs(fields)...),
	}
}

func toAttrs(fields []Field) []any {
	attrs := make([]any, 0, len(fields))
	for _, f := range fields {
		attrs = append(attrs, f.Key, f.Value)
	}
	return attrs
}
