package orderusecase

import (
	"context"
	"fmt"
	"sync"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventoryrepo"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/modules/payment/paymentusecase"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"
)

type ExpireUnfulfilledOrdersUsecase struct {
	orderRepo     orderrepo.OrderRepository
	orderItemRepo orderrepo.OrderItemRepository
	inventoryRepo inventoryrepo.InventoryRepository
	refundUsecase *paymentusecase.ProcessOrderRefundUsecase
	executor      transaction.Executor
	transactor    transaction.Transactor
	logger        applogger.Logger
	auditLogger   applogger.AuditLogger
	batchSize     int
	concurrency   int
}

func NewExpireUnfulfilledOrdersUsecase(
	orderRepo orderrepo.OrderRepository,
	orderItemRepo orderrepo.OrderItemRepository,
	inventoryRepo inventoryrepo.InventoryRepository,
	refundUsecase *paymentusecase.ProcessOrderRefundUsecase,
	executor transaction.Executor,
	transactor transaction.Transactor,
	logger applogger.Logger,
	auditLogger applogger.AuditLogger,
	batchSize int,
	concurrency int,
) *ExpireUnfulfilledOrdersUsecase {
	if batchSize <= 0 {
		batchSize = 100
	}
	if concurrency <= 0 {
		concurrency = 5
	}

	return &ExpireUnfulfilledOrdersUsecase{
		orderRepo:     orderRepo,
		orderItemRepo: orderItemRepo,
		inventoryRepo: inventoryRepo,
		refundUsecase: refundUsecase,
		executor:      executor,
		transactor:    transactor,
		logger:        logger,
		auditLogger:   auditLogger,
		batchSize:     batchSize,
		concurrency:   concurrency,
	}
}

func (u *ExpireUnfulfilledOrdersUsecase) Execute(ctx context.Context) {
	now := appclock.Now()
	orders, err := u.orderRepo.FindExpiredUnfulfilledOrders(ctx, u.executor,
		now,
		u.batchSize,
	)
	if err != nil {
		errMsg := "failed to find expired unfulfilled orders"
		u.logger.Error(ctx, errMsg,
			applogger.Field{Key: "error", Value: err.Error()},
		)
		return
	}

	if len(orders) == 0 {
		return
	}

	u.logger.Info(ctx, "order staff expiry job: processing unfulfilled orders exceeding 3 days SLA",
		applogger.Field{Key: "count", Value: len(orders)},
	)

	orderChan := make(chan orderdomain.Order, len(orders))
	for _, o := range orders {
		orderChan <- o
	}
	close(orderChan)

	var (
		successCount int64
		wg           sync.WaitGroup
		mu           sync.Mutex
	)

	numWorkers := min(len(orders), u.concurrency)
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for order := range orderChan {
				if err := u.expireSingleOrder(ctx, order); err != nil {
					u.logger.Error(ctx, "failed to expire unfulfilled order",
						applogger.Field{Key: "order_id", Value: order.ID.String()},
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

	u.logger.Info(ctx, "order staff expiry job: processing complete",
		applogger.Field{Key: "total", Value: len(orders)},
		applogger.Field{Key: "expired_count", Value: successCount},
	)
}

func (u *ExpireUnfulfilledOrdersUsecase) expireSingleOrder(ctx context.Context, order orderdomain.Order) error {
	if err := order.UpdateStatus(orderdomain.OrderStatusExpired); err != nil {
		return fmt.Errorf("invalid domain state transition: %w", err)
	}

	err := u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.orderRepo.UpdateStatus(ctx, exec, order.ID, orderdomain.OrderStatusExpired); err != nil {
			return fmt.Errorf("failed to update order status to expired: %w", err)
		}

		items, err := u.orderItemRepo.ListByOrderID(ctx, exec, order.ID)
		if err != nil {
			return fmt.Errorf("failed to list order items: %w", err)
		}

		for _, item := range items {
			if err := u.inventoryRepo.Restock(ctx, exec,
				item.ProductID,
				item.ShopID,
				item.Quantity,
			); err != nil {
				u.logger.Warn(ctx, "restock anomaly during order expiry",
					applogger.Field{Key: "order_id", Value: order.ID.String()},
					applogger.Field{Key: "product_id", Value: item.ProductID.String()},
					applogger.Field{Key: "reason", Value: err.Error()},
				)
			}
		}

		return nil
	})

	if err != nil {
		u.auditLogger.Log(ctx, applogger.AuditEvent{
			Category:   "system_job",
			Action:     "expire_unfulfilled_order",
			Resource:   "order",
			ResourceID: order.ID.String(),
			Outcome:    applogger.OutcomeFailure,
			Metadata: map[string]any{
				"error": err.Error(),
			},
		})
		return err
	}

	refundReason := "Staff handling SLA (3 days) expired without fulfillment"
	if refundErr := u.refundUsecase.Execute(ctx,
		order.ID,
		refundReason,
	); refundErr != nil {
		errMsg := fmt.Sprintf("failed to trigger refund for expired order: %v", refundErr)
		u.logger.Error(ctx, errMsg,
			applogger.Field{Key: "order_id", Value: order.ID.String()},
			applogger.Field{Key: "error", Value: refundErr.Error()},
		)
	}

	u.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "system_job",
		Action:     "expire_unfulfilled_order",
		Resource:   "order",
		ResourceID: order.ID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata: map[string]any{
			"old_status": string(order.Status),
			"new_status": string(orderdomain.OrderStatusExpired),
		},
	})

	return nil
}
