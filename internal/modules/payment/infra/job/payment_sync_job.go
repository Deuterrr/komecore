package job

import (
	"context"
	"time"

	"komecore/internal/modules/payment/usecase"
	applogger "komecore/pkg/logger"
)

// PaymentSyncJob is a background ticker that drives the
// SyncPendingPaymentsUsecase on a fixed interval.
type PaymentSyncJob struct {
	syncUsecase *usecase.SyncPendingPaymentsUsecase
	interval    time.Duration
	logger      applogger.Logger
}

func NewPaymentSyncJob(
	syncUsecase *usecase.SyncPendingPaymentsUsecase,
	interval time.Duration,
	logger applogger.Logger,
) *PaymentSyncJob {
	return &PaymentSyncJob{
		syncUsecase: syncUsecase,
		interval:    interval,
		logger:      logger,
	}
}

// Start begins the reconciliation ticker.
// It blocks until ctx is cancelled, then exits cleanly.
func (j *PaymentSyncJob) Start(ctx context.Context) {
	j.logger.Info(ctx, "payment sync job: started",
		applogger.Field{Key: "interval", Value: j.interval.String()},
	)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.logger.Info(ctx, "payment sync job: stopped")
			return
		case <-ticker.C:
			j.logger.Info(ctx, "payment sync job: tick — running reconciliation")
			j.syncUsecase.Execute(ctx)
		}
	}
}
