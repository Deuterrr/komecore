package paymentusecase

import (
	"context"
	"errors"
	"fmt"
	"sync"

	apperrors "komecore/internal/common/errors"
	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventorydomain"
	"komecore/internal/modules/inventory/inventoryrepo"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"
)

type ExpirePastDuePaymentsUsecase struct {
	paymentRepo    paymentrepo.PaymentRepository
	paymentGateway paymentgateway.Provider
	executor       transaction.Executor
	transactor     transaction.Transactor
	orderMgr       OrderPaymentManager
	inventoryRepo  inventoryrepo.InventoryRepository
	logger         applogger.Logger
	batchSize      int
	concurrency    int
}

func NewExpirePastDuePaymentsUsecase(
	paymentRepo paymentrepo.PaymentRepository,
	paymentGateway paymentgateway.Provider,
	executor transaction.Executor,
	transactor transaction.Transactor,
	orderMgr OrderPaymentManager,
	inventoryRepo inventoryrepo.InventoryRepository,
	logger applogger.Logger,
	batchSize int,
	concurrency int,
) *ExpirePastDuePaymentsUsecase {
	if batchSize <= 0 {
		batchSize = 100
	}

	if concurrency <= 0 {
		concurrency = 5
	}

	return &ExpirePastDuePaymentsUsecase{
		paymentRepo:    paymentRepo,
		paymentGateway: paymentGateway,
		executor:       executor,
		transactor:     transactor,
		orderMgr:       orderMgr,
		inventoryRepo:  inventoryRepo,
		logger:         logger,
		batchSize:      batchSize,
		concurrency:    concurrency,
	}
}

func (u *ExpirePastDuePaymentsUsecase) Execute(ctx context.Context) {
	now := appclock.Now()
	payments, err := u.paymentRepo.ListPastDuePending(ctx, u.executor, now,
		u.batchSize,
	)
	if err != nil {
		u.logger.Error(ctx, "failed to list past-due pending payments",
			applogger.Field{Key: "error", Value: err.Error()},
		)
		return
	}

	if len(payments) == 0 {
		return
	}

	u.logger.Info(ctx, "payment expiry job: processing past-due payments",
		applogger.Field{Key: "count", Value: len(payments)},
	)

	paymentChan := make(chan paymentdomain.Payment, len(payments))
	for _, p := range payments {
		paymentChan <- p
	}
	close(paymentChan)

	numWorkers := min(len(payments), u.concurrency)

	var (
		successCount int64
		wg           sync.WaitGroup
		mu           sync.Mutex
	)

	for range numWorkers {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for payment := range paymentChan {
				if err := u.expireSinglePayment(ctx, payment); err != nil {
					u.logger.Error(ctx, "failed to expire payment",
						applogger.Field{Key: "payment_id", Value: payment.ID.String()},
						applogger.Field{Key: "order_id", Value: payment.OrderID.String()},
						applogger.Field{Key: "error", Value: err.Error()},
					)
				} else {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	u.logger.Info(ctx, "payment expiry job: processing complete",
		applogger.Field{Key: "total", Value: len(payments)},
		applogger.Field{Key: "expired_count", Value: successCount},
	)
}

func (u *ExpirePastDuePaymentsUsecase) expireSinglePayment(
	ctx context.Context,
	payment paymentdomain.Payment,
) error {
	// Best-effort gateway cancellation
	if payment.Provider == "gateway" &&
		payment.ProviderOrderID != nil &&
		*payment.ProviderOrderID != "" {

		if err := u.paymentGateway.CancelTransaction(ctx, *payment.ProviderOrderID); err != nil {
			u.logger.Warn(ctx, "failed to cancel transaction at gateway during expiry, proceeding locally",
				applogger.Field{Key: "payment_id", Value: payment.ID.String()},
				applogger.Field{Key: "gateway_order_id", Value: *payment.ProviderOrderID},
				applogger.Field{Key: "error", Value: err.Error()},
			)
		}
	}

	return u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.paymentRepo.UpdateStatus(ctx, exec,
			payment.ID,
			paymentdomain.PaymentStatusExpired,
		); err != nil {
			return fmt.Errorf("failed to update payment status to expired: %w", err)
		}

		orderItems, err := u.orderMgr.ExpireOrderPayment(ctx, exec, payment.OrderID)
		if err != nil {
			return fmt.Errorf("failed to expire order payment: %w", err)
		}

		for _, item := range orderItems {
			if err := u.inventoryRepo.Release(ctx, exec,
				item.ProductID,
				item.ShopID,
				item.Quantity,
			); err != nil {
				if errors.Is(err, inventorydomain.ErrInsufficientReserved) ||
					errors.Is(err, apperrors.ErrNotFound) {

					msg := "inventory anomaly during payment expiry: reserved stock insufficient or missing"
					u.logger.Warn(ctx, msg,
						applogger.Field{Key: "payment_id", Value: payment.ID.String()},
						applogger.Field{Key: "order_id", Value: payment.OrderID.String()},
						applogger.Field{Key: "product_id", Value: item.ProductID.String()},
						applogger.Field{Key: "shop_id", Value: item.ShopID.String()},
						applogger.Field{Key: "requested_qty", Value: item.Quantity},
						applogger.Field{Key: "reason", Value: err.Error()},
					)
					continue
				}
				return fmt.Errorf("failed to release inventory for product %s: %w", item.ProductID, err)
			}
		}

		return nil
	})
}
