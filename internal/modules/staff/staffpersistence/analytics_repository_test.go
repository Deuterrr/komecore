package staffpersistence

import (
	"context"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
)

func TestAnalyticsRepository_EmptyQueries(t *testing.T) {
	repo := NewAnalyticsRepository()
	exec := &transaction.NoopExecutor{}

	// When exec.Query returns nil, methods should gracefully return empty slices
	buckets, err := repo.GetRevenueOverTime(context.Background(), exec, "day", time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buckets == nil {
		t.Fatal("expected non-nil slice")
	}

	pipeline, err := repo.GetOrderPipeline(context.Background(), exec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pipeline == nil {
		t.Fatal("expected non-nil slice")
	}

	products, err := repo.GetTopProducts(context.Background(), exec, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if products == nil {
		t.Fatal("expected non-nil slice")
	}
}
