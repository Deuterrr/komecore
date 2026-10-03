package bootstrap

import (
	"context"
	"sync"
	"time"

	orderJob "komecore/internal/modules/order/infra/job"
	paymentJob "komecore/internal/modules/payment/infra/job"
	applogger "komecore/pkg/logger"
)

const defaultOrderStaffExpiryInterval = 15 * time.Minute

// Job defines the contract for any background runnable scheduled job.
type Job interface {
	Start(ctx context.Context)
}

// Scheduler handles registration and lifecycle management of background jobs.
type Scheduler struct {
	jobs []Job
	wg   sync.WaitGroup
}

func NewScheduler(cfg Config, container *Container, logger applogger.Logger) *Scheduler {
	s := &Scheduler{
		jobs: make([]Job, 0),
	}

	if container == nil {
		return s
	}

	syncInterval := time.Duration(cfg.PaymentSync.IntervalMinutes) * time.Minute
	paymentSyncJob := paymentJob.NewPaymentSyncJob(
		&container.SyncPendingPayments,
		syncInterval,
		logger,
	)
	s.Register(paymentSyncJob)

	expiryInterval := time.Duration(cfg.PaymentExpiry.IntervalMinutes) * time.Minute
	paymentExpiryJob := paymentJob.NewPaymentExpiryJob(
		&container.ExpirePastDuePayments,
		expiryInterval,
		logger,
	)
	s.Register(paymentExpiryJob)

	orderStaffExpiryJob := orderJob.NewOrderStaffExpiryJob(
		&container.ExpireUnfulfilledOrders,
		defaultOrderStaffExpiryInterval,
		logger,
	)
	s.Register(orderStaffExpiryJob)

	return s
}

func (s *Scheduler) Register(job Job) {
	if job != nil {
		s.jobs = append(s.jobs, job)
	}
}

// Start launches all registered background jobs in separate goroutines.
func (s *Scheduler) Start(ctx context.Context) {
	for _, job := range s.jobs {
		s.wg.Add(1)
		go func(j Job) {
			defer s.wg.Done()
			j.Start(ctx)
		}(job)
	}
}

// Wait waits for all running jobs to complete or until the context expires.
func (s *Scheduler) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
