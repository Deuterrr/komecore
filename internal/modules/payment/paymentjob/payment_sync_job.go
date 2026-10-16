package paymentjob

import (
	"context"
	"time"

	"komecore/internal/modules/payment"
	applogger "komecore/pkg/logger"
)

// PaymentSyncJob is a background ticker that drives the
// SyncPendingPayments on a fixed interval.
type PaymentSyncJob struct {
	service  *payment.Service
	interval time.Duration
	logger   applogger.Logger
}

func NewPaymentSyncJob(
	service *payment.Service,
	interval time.Duration,
	logger applogger.Logger,
) *PaymentSyncJob {
	return &PaymentSyncJob{
		service:  service,
		interval: interval,
		logger:   logger,
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
			j.service.SyncPendingPayments(ctx)
		}
	}
}
