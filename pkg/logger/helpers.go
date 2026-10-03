package logger

import "context"

// Business logs a meaningful domain event from the usecase layer.
//
// It automatically attaches layer=usecase and action fields so callers
// only need to describe what happened, without building field slices manually.
//
// Usage:
//
//	applogger.Business(ctx, log, "order_created",
//	    applogger.Field{Key: "order_id", Value: id},
//	    applogger.Field{Key: "total",    Value: 250000},
//	)
func Business(ctx context.Context, log Logger, action string, fields ...Field) {
	all := append([]Field{
		{Key: "layer", Value: string(LayerUsecase)},
		{Key: "action", Value: action},
	}, fields...)
	log.Info(ctx, action, all...)
}

// Dependency logs the outcome of a call to an external dependency from
// the infrastructure layer.
//
// It automatically attaches layer=infra, provider, operation, duration_ms,
// and status fields. On error, logs at Error level with the error message.
//
// Usage:
//
//	start := time.Now()
//	resp, err := midtransClient.Charge(req)
//	applogger.Dependency(ctx, log, "midtrans", "charge",
//	    time.Since(start).Milliseconds(), err)
func Dependency(
	ctx context.Context,
	log Logger,
	provider string,
	operation string,
	durationMs int64,
	err error,
) {
	fields := []Field{
		{Key: "layer", Value: string(LayerInfra)},
		{Key: "provider", Value: provider},
		{Key: "operation", Value: operation},
		{Key: "duration_ms", Value: durationMs},
	}

	if err != nil {
		fields = append(fields,
			Field{Key: "status", Value: "failure"},
			Field{Key: "error", Value: err.Error()},
		)
		log.Error(ctx, operation, fields...)
		return
	}

	fields = append(fields, Field{Key: "status", Value: "success"})
	log.Info(ctx, operation, fields...)
}

// LogError logs an error with layer and error fields automatically attached,
// matching the issue's expected shape for structured error logs.
//
// Usage:
//
//	applogger.LogError(ctx, log, applogger.LayerUsecase, err,
//	    applogger.Field{Key: "order_id", Value: id},
//	)
func LogError(ctx context.Context, log Logger, layer LogLayer, err error, fields ...Field) {
	all := append([]Field{
		{Key: "layer", Value: string(layer)},
		{Key: "error", Value: err.Error()},
	}, fields...)
	log.Error(ctx, err.Error(), all...)
}
