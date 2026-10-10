package staffrepo

import (
	"context"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/staffdomain"
)

type AnalyticsRepository interface {
	GetRevenueOverTime(ctx context.Context, exec transaction.Executor, interval string, since time.Time) ([]staffdomain.RevenueBucket, error)
	GetOrderPipeline(ctx context.Context, exec transaction.Executor) ([]staffdomain.OrderStatusCount, error)
	GetTopProducts(ctx context.Context, exec transaction.Executor, limit int) ([]staffdomain.TopProduct, error)
}
