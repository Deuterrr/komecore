package outbox

import (
	"context"
	"fmt"
	"time"

	transaction "komecore/internal/infra/transactor"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"
)

// CalculateBackoff calculates the exponential retry backoff duration:
// Delta t = 2^(retry) * 10 seconds.
func CalculateBackoff(retryCount int) time.Duration {
	if retryCount < 0 {
		retryCount = 0
	}
	if retryCount > 10 {
		retryCount = 10 // Prevent integer overflow on bit shift
	}
	return time.Duration(1<<retryCount) * 10 * time.Second
}

// Worker autonomously polls and processes pending outbox events.
type Worker struct {
	repo       Repository
	dispatcher Dispatcher
	executor   transaction.Executor
	transactor transaction.Transactor
	logger     applogger.Logger
	interval   time.Duration
	batchSize  int
}

// NewWorker constructs a new outbox background worker.
func NewWorker(
	repo Repository,
	dispatcher Dispatcher,
	executor transaction.Executor,
	transactor transaction.Transactor,
	logger applogger.Logger,
	interval time.Duration,
	batchSize int,
) *Worker {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 20
	}

	return &Worker{
		repo:       repo,
		dispatcher: dispatcher,
		executor:   executor,
		transactor: transactor,
		logger:     logger,
		interval:   interval,
		batchSize:  batchSize,
	}
}

// Start begins the polling loop, satisfying the bootstrap.Job interface.
func (w *Worker) Start(ctx context.Context) {
	if w.logger != nil {
		w.logger.Info(ctx, "outbox worker: started",
			applogger.Field{Key: "interval", Value: w.interval.String()},
			applogger.Field{Key: "batch_size", Value: fmt.Sprintf("%d", w.batchSize)},
		)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if w.logger != nil {
				w.logger.Info(ctx, "outbox worker: stopped")
			}
			return
		case <-ticker.C:
			processed, err := w.ProcessBatch(ctx)
			if err != nil && w.logger != nil {
				w.logger.Error(ctx, "outbox worker: batch processing failed",
					applogger.Field{Key: "error", Value: err.Error()},
				)
			} else if processed > 0 && w.logger != nil {
				w.logger.Info(ctx, "outbox worker: processed batch",
					applogger.Field{Key: "count", Value: fmt.Sprintf("%d", processed)},
				)
			}
		}
	}
}

// ProcessBatch polls and delivers a single batch of pending outbox events.
func (w *Worker) ProcessBatch(ctx context.Context) (int, error) {
	var events []Event

	// Fetch pending rows inside a transaction boundary to hold row-level locks
	err := w.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		var err error
		events, err = w.repo.FetchPending(ctx, exec, w.batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch pending outbox events: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	processedCount := 0
	for _, event := range events {
		dispatchErr := w.dispatcher.Dispatch(ctx, event)
		now := appclock.Now()

		execErr := w.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			if dispatchErr == nil {
				if err := w.repo.MarkCompleted(ctx, exec, event.ID, now); err != nil {
					return fmt.Errorf("failed to mark outbox event %s completed: %w", event.ID, err)
				}
				return nil
			}

			newRetry := event.RetryCount + 1
			nextScheduled := now.Add(CalculateBackoff(newRetry))
			if err := w.repo.MarkFailed(ctx, exec, event.ID, newRetry, nextScheduled, dispatchErr.Error()); err != nil {
				return fmt.Errorf("failed to mark outbox event %s failed: %w", event.ID, err)
			}
			return nil
		})

		if execErr != nil {
			if w.logger != nil {
				w.logger.Error(ctx, "outbox worker: persistence error updating event status",
					applogger.Field{Key: "event_id", Value: event.ID.String()},
					applogger.Field{Key: "error", Value: execErr.Error()},
				)
			}
		} else {
			processedCount++
		}
	}

	return processedCount, nil
}
