package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	log *zap.Logger
}

func NewZapLogger(env string) Logger {
	var logger *zap.Logger

	if env == "development" {
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		}
		cfg.DisableCaller = true
		cfg.DisableStacktrace = true
		logger, _ = cfg.Build()
		return &zapLogger{log: logger}
	}

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format(time.RFC3339))
	}
	logger, _ = cfg.Build()

	return &zapLogger{log: logger}
}

func (l *zapLogger) Info(ctx context.Context, msg string, fields ...Field) {
	all := append(FieldsFromContext(ctx), fields...)
	l.log.Info(msg, toZapFields(all)...)
}

func (l *zapLogger) Error(ctx context.Context, msg string, fields ...Field) {
	all := append(FieldsFromContext(ctx), fields...)
	l.log.Error(msg, toZapFields(all)...)
}

func (l *zapLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	all := append(FieldsFromContext(ctx), fields...)
	l.log.Warn(msg, toZapFields(all)...)
}

func (l *zapLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	all := append(FieldsFromContext(ctx), fields...)
	l.log.Debug(msg, toZapFields(all)...)
}

func (l *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{
		log: l.log.With(toZapFields(fields)...),
	}
}

func toZapFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))

	for _, f := range fields {
		zapFields = append(zapFields, zap.Any(f.Key, f.Value))
	}

	return zapFields
}
