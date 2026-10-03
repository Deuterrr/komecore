package job

import (
	"context"
	"time"

	"komecore/internal/modules/payment/usecase"
	applogger "komecore/pkg/logger"
)

// PaymentExpiryJob is a background ticker that runs the automated
// ExpirePastDuePaymentsUsecase on a fixed 1-minute interval.
type PaymentExpiryJob struct {
	expiryUsecase *usecase.ExpirePastDuePaymentsUsecase
	interval      time.Duration
	logger        applogger.Logger
}

func NewPaymentExpiryJob(
	expiryUsecase *usecase.ExpirePastDuePaymentsUsecase,
	interval time.Duration,
	logger applogger.Logger,
) *PaymentExpiryJob {
	return &PaymentExpiryJob{
		expiryUsecase: expiryUsecase,
		interval:      interval,
		logger:        logger,
	}
}

// Start begins the payment expiry ticker.
// It blocks until ctx is cancelled, then exits cleanly.
func (j *PaymentExpiryJob) Start(ctx context.Context) {
	j.logger.Info(ctx, "payment expiry job: started",
		applogger.Field{Key: "interval", Value: j.interval.String()},
	)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.logger.Info(ctx, "payment expiry job: stopped")
			return
		case <-ticker.C:
			j.logger.Info(ctx, "payment expiry job: tick — expiring past-due payments")
			j.expiryUsecase.Execute(ctx)
		}
	}
}
