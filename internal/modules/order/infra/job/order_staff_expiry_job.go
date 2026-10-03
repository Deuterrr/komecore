package job

import (
	"context"
	"time"

	"komecore/internal/modules/order/usecase"
	applogger "komecore/pkg/logger"
)

type OrderStaffExpiryJob struct {
	expiryUsecase *usecase.ExpireUnfulfilledOrdersUsecase
	interval      time.Duration
	logger        applogger.Logger
}

func NewOrderStaffExpiryJob(
	expiryUsecase *usecase.ExpireUnfulfilledOrdersUsecase,
	interval time.Duration,
	logger applogger.Logger,
) *OrderStaffExpiryJob {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &OrderStaffExpiryJob{
		expiryUsecase: expiryUsecase,
		interval:      interval,
		logger:        logger,
	}
}

func (j *OrderStaffExpiryJob) Start(ctx context.Context) {
	j.logger.Info(ctx, "order staff expiry job: started",
		applogger.Field{Key: "interval", Value: j.interval.String()},
	)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.logger.Info(ctx, "order staff expiry job: stopped")
			return
		case <-ticker.C:
			j.logger.Info(ctx, "order staff expiry job: tick — expiring unfulfilled orders exceeding 3 days SLA")
			j.expiryUsecase.Execute(ctx)
		}
	}
}
