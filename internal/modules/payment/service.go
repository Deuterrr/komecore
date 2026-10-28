package payment

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/infra/paymentgateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"
	"komecore/internal/pagination"
	"komecore/internal/strtpl"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type OrderInfo struct {
	ID            uuid.UUID
	CustomerID    uuid.UUID
	Number        string
	Total         int64
	CustomerEmail string
	CustomerName  string
}

type OrderItemInfo struct {
	ProductID uuid.UUID
	ShopID    uuid.UUID
	Quantity  int
}

type OrderPaymentManager interface {
	GetOrderForPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*OrderInfo, error)
	GetInvoiceNumber(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (string, error)
	ConfirmOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID, confirmedAt time.Time, handlingWindow time.Duration) ([]OrderItemInfo, error)
	ExpireOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]OrderItemInfo, error)
	CancelOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]OrderItemInfo, error)
}

type Service struct {
	paymentRepo            Repository
	paymentMethodRepo      MethodRepository
	paymentInstructionRepo InstructionRepository
	paymentChannelDataRepo ChannelDataRepository
	orderMgr               OrderPaymentManager
	inventoryRepo          inventory.Repository
	paymentGateway         paymentgateway.Provider
	processWebhook         *WebhookProcessor
	executor               transaction.Executor
	transactor             transaction.Transactor
	logger                 applogger.Logger
	outboxRecorder         OutboxRecorder
	lookbackWindow         time.Duration
	batchSize              int
	concurrency            int
}

func NewService(
	paymentRepo Repository,
	paymentMethodRepo MethodRepository,
	paymentInstructionRepo InstructionRepository,
	paymentChannelDataRepo ChannelDataRepository,
	orderMgr OrderPaymentManager,
	inventoryRepo inventory.Repository,
	paymentGateway paymentgateway.Provider,
	processWebhook *WebhookProcessor,
	executor transaction.Executor,
	transactor transaction.Transactor,
	logger applogger.Logger,
	lookbackWindow time.Duration,
	batchSize int,
	concurrency int,
) *Service {
	if lookbackWindow <= 0 {
		lookbackWindow = 24 * time.Hour
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if concurrency <= 0 {
		concurrency = 5
	}

	return &Service{
		paymentRepo:            paymentRepo,
		paymentMethodRepo:      paymentMethodRepo,
		paymentInstructionRepo: paymentInstructionRepo,
		paymentChannelDataRepo: paymentChannelDataRepo,
		orderMgr:               orderMgr,
		inventoryRepo:          inventoryRepo,
		paymentGateway:         paymentGateway,
		processWebhook:         processWebhook,
		executor:               executor,
		transactor:             transactor,
		logger:                 logger,
		lookbackWindow:         lookbackWindow,
		batchSize:              batchSize,
		concurrency:            concurrency,
	}
}

func (s *Service) WithOutboxRecorder(r OutboxRecorder) *Service {
	s.outboxRecorder = r
	if s.processWebhook != nil {
		s.processWebhook.WithOutboxRecorder(r)
	}
	return s
}

type CheckPaymentStatusInput struct {
	OrderID    uuid.UUID
	CustomerID uuid.UUID
}

type CheckPaymentStatusResult struct {
	// Status is the payment status after the check
	// (may be unchanged if Midtrans still reports pending).
	Status PaymentStatus

	// Synced is true when the status was resolved
	// from Midtrans during this call
	// (i.e. it was pending before and is now terminal).
	Synced bool
}

// CheckPaymentStatus performs a synchronous, customer-triggered payment verification
// against the external gateway to resolve payment states that may temporarily
// lag behind asynchronous webhook notifications.
func (s *Service) CheckPaymentStatus(ctx context.Context, input CheckPaymentStatusInput) (*CheckPaymentStatusResult, error) {
	order, err := s.orderMgr.GetOrderForPayment(ctx, s.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("check payment status: retrieve order: %w", err)
	}
	if order == nil {
		return nil, apperror.NewNotFound("order not found")
	}
	if order.CustomerID != input.CustomerID {
		return nil, apperror.NewNotFound("order not found")
	}

	pmt, err := s.paymentRepo.GetByOrderID(ctx, s.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("check payment status: retrieve payment: %w", err)
	}
	if pmt == nil {
		return nil, apperror.NewNotFound("payment not found for order")
	}
	// Only gateway payments with a ProviderOrderID can be synced.
	if pmt.ProviderOrderID == nil {
		return &CheckPaymentStatusResult{Status: pmt.Status, Synced: false}, nil
	}
	// Only pending payments need a sync — return early for any terminal state.
	if pmt.Status != PaymentStatusPending {
		return &CheckPaymentStatusResult{Status: pmt.Status, Synced: false}, nil
	}

	result, err := s.paymentGateway.GetTransactionStatus(ctx, *pmt.ProviderOrderID)
	if err != nil {
		return nil, fmt.Errorf("check payment status: gateway status check failed: %w", err)
	}

	if result.Status == paymentgateway.NotificationStatusPending {
		return &CheckPaymentStatusResult{
			Status: pmt.Status,
			Synced: false,
		}, nil
	}

	// Status has resolved — drive it through the standard
	// webhook processing pipeline directly without redundant outbound calls.
	if err := s.processWebhook.Execute(ctx, ProcessPaymentWebhookInput{
		NotificationResult: result,
	}); err != nil {
		return nil, fmt.Errorf("check payment status: process resolved status: %w", err)
	}

	resolvedStatus := mapGatewayStatus(result.Status)
	res := CheckPaymentStatusResult{
		Status: resolvedStatus,
		Synced: true,
	}
	return &res, nil
}

// mapGatewayStatus converts a gateway NotificationStatus to the domain
// PaymentStatus matching the WebhookProcessor state machine transitions.
func mapGatewayStatus(s paymentgateway.NotificationStatus) PaymentStatus {
	switch s {
	case paymentgateway.NotificationStatusSettlement:
		return PaymentStatusPaid
	case paymentgateway.NotificationStatusExpire:
		return PaymentStatusExpired
	case paymentgateway.NotificationStatusCancel:
		return PaymentStatusCancelled
	case paymentgateway.NotificationStatusDeny:
		return PaymentStatusFailed
	default:
		return PaymentStatusPending
	}
}

func (s *Service) ExpirePastDuePayments(ctx context.Context) {
	now := appclock.Now()
	payments, err := s.paymentRepo.ListPastDuePending(ctx, s.executor, now,
		s.batchSize,
	)
	if err != nil {
		s.logger.Error(ctx, "failed to list past-due pending payments",
			applogger.Field{Key: "error", Value: err.Error()},
		)
		return
	}

	if len(payments) == 0 {
		return
	}

	s.logger.Info(ctx, "payment expiry job: processing past-due payments",
		applogger.Field{Key: "count", Value: len(payments)},
	)

	paymentChan := make(chan Payment, len(payments))
	for _, p := range payments {
		paymentChan <- p
	}
	close(paymentChan)

	numWorkers := min(len(payments), s.concurrency)

	var (
		successCount int64
		wg           sync.WaitGroup
		mu           sync.Mutex
	)

	for range numWorkers {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for pmt := range paymentChan {
				if err := s.expireSinglePayment(ctx, pmt); err != nil {
					s.logger.Error(ctx, "failed to expire payment",
						applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
						applogger.Field{Key: "order_id", Value: pmt.OrderID.String()},
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

	s.logger.Info(ctx, "payment expiry job: processing complete",
		applogger.Field{Key: "total", Value: len(payments)},
		applogger.Field{Key: "expired_count", Value: successCount},
	)
}

func (s *Service) expireSinglePayment(ctx context.Context, pmt Payment) error {
	// Invariant: Best-effort external cancellation runs prior to DB transaction to prevent orphan charges.
	if pmt.ProviderOrderID != nil && *pmt.ProviderOrderID != "" {
		if err := s.paymentGateway.CancelTransaction(ctx, *pmt.ProviderOrderID); err != nil {
			s.logger.Warn(ctx, "failed to cancel transaction at gateway during expiry, proceeding locally",
				applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
				applogger.Field{Key: "gateway_order_id", Value: *pmt.ProviderOrderID},
				applogger.Field{Key: "error", Value: err.Error()},
			)
		}
	}

	return s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		// Acquire pessimistic row lock to guard against concurrent settlement webhooks.
		current, err := s.paymentRepo.GetByOrderIDForUpdate(ctx, exec, pmt.OrderID)
		if err != nil {
			return fmt.Errorf("failed to lock payment for expiry: %w", err)
		}
		if current == nil || current.Status != PaymentStatusPending {
			return nil
		}

		if err := s.paymentRepo.UpdateStatus(ctx, exec,
			pmt.ID,
			PaymentStatusExpired,
		); err != nil {
			return fmt.Errorf("failed to update payment status to expired: %w", err)
		}

		orderItems, err := s.orderMgr.ExpireOrderPayment(ctx, exec, pmt.OrderID)
		if err != nil {
			return fmt.Errorf("failed to expire order payment: %w", err)
		}

		for _, item := range orderItems {
			if err := s.inventoryRepo.Release(ctx, exec,
				item.ProductID,
				item.ShopID,
				item.Quantity,
			); err != nil {
				if errors.Is(err, inventory.ErrInsufficientReserved) ||
					errors.Is(err, apperror.ErrNotFound) {

					msg := "inventory anomaly during payment expiry: reserved stock insufficient or missing"
					s.logger.Warn(ctx, msg,
						applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
						applogger.Field{Key: "order_id", Value: pmt.OrderID.String()},
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

type GetPaymentDetailInput struct {
	OrderID uuid.UUID
	// Optional customer ownership enforcement
	CustomerID *uuid.UUID
}

type GetPaymentDetailResult struct {
	Payment     Payment
	ChannelData *PaymentChannelData
	Instruction *string
}

func (s *Service) GetPaymentDetail(ctx context.Context, input GetPaymentDetailInput) (*GetPaymentDetailResult, error) {
	order, err := s.orderMgr.GetOrderForPayment(ctx, s.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve order: %w", err)
	}
	if order == nil {
		return nil, apperror.NewNotFound("order not found")
	}

	if input.CustomerID != nil &&
		order.CustomerID != *input.CustomerID {
		return nil, apperror.NewNotFound("order not found")
	}

	pmt, err := s.paymentRepo.GetByOrderID(ctx, s.executor, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment: %w", err)
	}
	if pmt == nil {
		return nil, apperror.NewNotFound("payment not found")
	}

	var (
		channelData *PaymentChannelData
		vaNumber    string
		qrString    string
		redirectURL string
	)

	cd, err := s.paymentChannelDataRepo.GetByPaymentID(ctx, s.executor, pmt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment channel data: %w", err)
	}

	channelData = cd
	if cd != nil {
		if cd.AccountNumber != nil {
			vaNumber = *cd.AccountNumber
		} else if cd.ChannelType == TypeBankTransfer && cd.ActionURL != nil {
			vaNumber = *cd.ActionURL
		}
		if cd.QRString != nil {
			qrString = *cd.QRString
		} else if cd.ChannelType == TypeQRCode && cd.ActionURL != nil {
			qrString = *cd.ActionURL
		}
		if cd.RedirectURL != nil {
			redirectURL = *cd.RedirectURL
		} else if cd.ChannelType == TypeEWallet && cd.ActionURL != nil {
			redirectURL = *cd.ActionURL
		}
	}

	var renderedInstruction *string
	instruction, err := s.paymentInstructionRepo.GetByPaymentMethodID(ctx, s.executor, pmt.MethodID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment instruction: %w", err)
	}

	if instruction != nil {
		invoiceNumber, err := s.orderMgr.GetInvoiceNumber(ctx, s.executor, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve invoice: %w", err)
		}
		if invoiceNumber == "" {
			invoiceNumber = order.Number
		}

		expiredAtStr := ""
		if pmt.ExpiresAt != nil {
			expiredAtStr = pmt.ExpiresAt.Format(time.RFC3339)
		} else {
			expiredAtStr = pmt.CreatedAt.Add(24 * time.Hour).Format(time.RFC3339)
		}

		vars := map[string]string{
			"invoice_number": invoiceNumber,
			"amount":         strconv.FormatInt(order.Total, 10),
			"expired_at":     expiredAtStr,
			"va_number":      vaNumber,
			"bill_key":       vaNumber,
			"qr_string":      qrString,
			"redirect_url":   redirectURL,
		}

		content, err := strtpl.Render(instruction.Content, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to format payment instruction: %w", err)
		}
		renderedInstruction = &content
	}

	return &GetPaymentDetailResult{
		Payment:     *pmt,
		ChannelData: channelData,
		Instruction: renderedInstruction,
	}, nil
}

type ListPaymentMethodInput struct {
	Sort string
}

func (s *Service) ListPaymentMethods(ctx context.Context, input ListPaymentMethodInput) ([]PaymentMethod, error) {
	var pmSortKeys = map[string]pagination.SortKey{
		"latest": PaymentMethodSortLatest,
		"name":   PaymentMethodSortName,
		"code":   PaymentMethodSortCode,
		"type":   PaymentMethodSortType,
	}

	var sorts pagination.Sorts
	if input.Sort != "" {
		parts := strings.SplitSeq(input.Sort, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			subparts := strings.Split(part, ":")
			key := strings.TrimSpace(subparts[0])

			var dir pagination.SortDirection = pagination.SortDesc
			if len(subparts) > 1 {
				d := strings.ToLower(strings.TrimSpace(subparts[1]))
				if d == "asc" {
					dir = pagination.SortAsc
				}
			}

			sortKey, exists := pmSortKeys[key]
			if exists {
				sorts = append(sorts, pagination.Sort{
					By:        sortKey,
					Direction: dir,
				})
			}
		}
	}

	if len(sorts) == 0 {
		sorts = pagination.Sorts{
			{
				By:        PaymentMethodSortLatest,
				Direction: pagination.SortDesc,
			},
		}
	}

	methods, err := s.paymentMethodRepo.ListAll(ctx, s.executor, sorts)
	if err != nil {
		return nil, fmt.Errorf("failed to load payment methods: %w", err)
	}

	return methods, nil
}

func (s *Service) ListPaymentMethod(ctx context.Context, input ListPaymentMethodInput) ([]PaymentMethod, error) {
	return s.ListPaymentMethods(ctx, input)
}

func (s *Service) ProcessOrderRefund(ctx context.Context, orderID uuid.UUID, reason string) error {
	pmt, err := s.paymentRepo.GetByOrderID(ctx, s.executor, orderID)
	if err != nil {
		return fmt.Errorf("failed to get payment for order %s: %w", orderID, err)
	}
	if pmt == nil {
		return fmt.Errorf("payment not found for order %s", orderID)
	}

	if pmt.Status != PaymentStatusPaid {
		s.logger.Info(ctx, "skipping refund as payment is not in paid status",
			applogger.Field{Key: "order_id", Value: orderID.String()},
			applogger.Field{Key: "payment_status", Value: string(pmt.Status)},
		)
		return nil
	}

	// Invariant: Presence of provider_order_id confirms an active upstream charge eligible for automated gateway refund.
	canAutoRefund := pmt.ProviderOrderID != nil &&
		*pmt.ProviderOrderID != ""

	if canAutoRefund {
		refundReq := paymentgateway.RefundRequest{
			GatewayOrderID: *pmt.ProviderOrderID,
			RefundAmount:   pmt.Amount,
			Reason:         reason,
		}

		s.logger.Info(ctx, "attempting automated gateway refund",
			applogger.Field{Key: "order_id", Value: orderID.String()},
			applogger.Field{Key: "gateway_order_id", Value: *pmt.ProviderOrderID},
		)

		resp, gatewayErr := s.paymentGateway.RefundTransaction(ctx, refundReq)
		if gatewayErr == nil && resp != nil {
			s.logger.Info(ctx, "automated gateway refund successful",
				applogger.Field{Key: "order_id", Value: orderID.String()},
				applogger.Field{Key: "gateway_transaction_id", Value: resp.GatewayTransactionID},
			)
			return s.paymentRepo.UpdateStatus(ctx, s.executor, pmt.ID, PaymentStatusRefunded)
		}

		s.logger.Warn(ctx, "automated gateway refund failed or not supported by channel; flagging for manual refund queue",
			applogger.Field{Key: "order_id", Value: orderID.String()},
			applogger.Field{Key: "error", Value: func() string {
				if gatewayErr != nil {
					return gatewayErr.Error()
				}
				return "unknown gateway refund response"
			}()},
		)
	} else {
		s.logger.Info(ctx, "payment method does not support automated refund; enqueueing for manual refund",
			applogger.Field{Key: "order_id", Value: orderID.String()},
		)
	}

	// Fallback to manual refund queue status
	return s.paymentRepo.UpdateStatus(ctx, s.executor, pmt.ID, PaymentStatusRefundPending)
}

func isAutoRefundableChannel(channel string) bool {
	channelLower := strings.ToLower(channel)
	return strings.Contains(channelLower, "gopay") ||
		strings.Contains(channelLower, "shopeepay") ||
		strings.Contains(channelLower, "qris") ||
		strings.Contains(channelLower, "credit_card")
}

type SavePaymentInstructionInput struct {
	PaymentMethodID uuid.UUID
	Content         string
}

func (s *Service) SavePaymentInstruction(ctx context.Context, input SavePaymentInstructionInput) error {
	method, err := s.paymentMethodRepo.GetByID(ctx, s.executor, input.PaymentMethodID)
	if err != nil {
		return fmt.Errorf("failed to retrieve payment method: %w", err)
	}
	if method == nil {
		return apperror.NewNotFound("payment method not found")
	}

	existing, err := s.paymentInstructionRepo.GetByPaymentMethodID(ctx, s.executor, input.PaymentMethodID)
	if err != nil {
		return fmt.Errorf("failed to retrieve existing payment instruction: %w", err)
	}

	var instruction PaymentInstruction
	if existing != nil {
		instruction = *existing
		instruction.Content = input.Content
	} else {
		instruction = PaymentInstruction{
			ID:              uuid.New(),
			PaymentMethodID: input.PaymentMethodID,
			Content:         input.Content,
			CreatedAt:       appclock.Now(),
		}
	}

	err = s.paymentInstructionRepo.Save(ctx, s.executor,
		instruction,
	)
	if err != nil {
		return fmt.Errorf("failed to save payment instruction: %w", err)
	}

	return nil
}

type SavePaymentMethodInput struct {
	ID       uuid.UUID
	IsActive bool
}

func (s *Service) SavePaymentMethod(ctx context.Context, input SavePaymentMethodInput) error {
	paymentMethod, err := s.paymentMethodRepo.GetByID(ctx, s.executor, input.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve payment method: %w", err)
	}
	if paymentMethod == nil {
		return apperror.NewNotFound("payment method not found")
	}

	paymentMethod.IsActive = input.IsActive

	err = s.paymentMethodRepo.Save(ctx, s.executor,
		*paymentMethod,
	)
	if err != nil {
		return fmt.Errorf("failed to save payment method: %w", err)
	}

	return nil
}

func (s *Service) SyncPaymentMethods(ctx context.Context) error {
	allowedMethods := s.paymentGateway.AllowedPaymentMethods()
	providerName := s.paymentGateway.Name()

	existingMethods, err := s.paymentMethodRepo.ListAll(ctx, s.executor, pagination.Sorts{})
	if err != nil {
		return fmt.Errorf("failed to list existing payment methods: %w", err)
	}

	type key struct {
		code     string
		provider string
		mType    string
	}
	existingMap := make(map[key]*PaymentMethod)
	for i := range existingMethods {
		m := &existingMethods[i]
		existingMap[key{
			code:     m.Code,
			provider: m.Provider,
			mType:    string(m.Type),
		}] = m
	}

	activeKeys := make(map[key]bool)
	for _, am := range allowedMethods {
		k := key{
			code:     am.Code,
			provider: providerName,
			mType:    am.Type,
		}
		activeKeys[k] = true

		existing, ok := existingMap[k]
		if !ok {
			newMethod := PaymentMethod{
				ID:            uuid.New(),
				Name:          am.Name,
				Code:          am.Code,
				Provider:      providerName,
				Type:          PaymentMethodType(am.Type),
				IsActive:      true,
				Description:   am.Description,
				FeeType:       PaymentFeeType(am.FeeType),
				FeeFixed:      am.FeeFixed,
				FeePercentage: am.FeePercentage,
				FeeMax:        am.FeeMax,
				CreatedAt:     appclock.Now(),
			}
			if err := s.paymentMethodRepo.Save(ctx, s.executor, newMethod); err != nil {
				return fmt.Errorf("failed to save new payment method %s: %w", am.Code, err)
			}
		} else {
			updatedMethod := *existing
			updatedMethod.Name = am.Name
			updatedMethod.Description = am.Description
			updatedMethod.FeeType = PaymentFeeType(am.FeeType)
			updatedMethod.FeeFixed = am.FeeFixed
			updatedMethod.FeePercentage = am.FeePercentage
			updatedMethod.FeeMax = am.FeeMax

			if err := s.paymentMethodRepo.Save(ctx, s.executor, updatedMethod); err != nil {
				return fmt.Errorf("failed to update payment method %s: %w", am.Code, err)
			}
		}
	}

	for k, method := range existingMap {
		if !activeKeys[k] && method.IsActive {
			method.IsActive = false
			if err := s.paymentMethodRepo.Save(ctx, s.executor, *method); err != nil {
				return fmt.Errorf("failed to deactivate obsolete payment method %s: %w", method.Code, err)
			}
		}
	}

	return nil
}

// SyncPendingPayments reconciles pending payments with the upstream gateway within the
// configured lookback window using a bounded parallel worker pool.
func (s *Service) SyncPendingPayments(ctx context.Context) {
	since := appclock.Now().Add(-s.lookbackWindow)
	var msg string

	payments, err := s.paymentRepo.ListPendingGateway(ctx, s.executor, since)
	if err != nil {
		msg = "failed to list pending gateway payments"
		s.logger.Error(ctx, msg,
			applogger.Field{Key: "error", Value: err.Error()},
		)
		return
	}
	if len(payments) == 0 {
		msg = "no pending gateway payments in window"
		s.logger.Info(ctx, msg)
		return
	}

	msg = "payment sync: starting reconciliation cycle"
	s.logger.Info(ctx, msg,
		applogger.Field{Key: "count", Value: len(payments)},
		applogger.Field{Key: "since", Value: since.Format(time.RFC3339)},
	)

	workerCount := s.concurrency
	if workerCount <= 0 {
		workerCount = 5
	}
	if len(payments) < workerCount {
		workerCount = len(payments)
	}

	paymentChan := make(chan Payment, len(payments))
	for _, pmt := range payments {
		paymentChan <- pmt
	}
	close(paymentChan)

	var resolved int32
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pmt := range paymentChan {
				if ctx.Err() != nil {
					return
				}
				if s.reconcileSinglePayment(ctx, pmt) {
					atomic.AddInt32(&resolved, 1)
				}
			}
		}()
	}
	wg.Wait()

	totalResolved := int(atomic.LoadInt32(&resolved))
	msg = "reconciliation cycle complete"
	s.logger.Info(ctx, msg,
		applogger.Field{Key: "total", Value: len(payments)},
		applogger.Field{Key: "resolved", Value: totalResolved},
		applogger.Field{Key: "skipped", Value: fmt.Sprintf("%d", len(payments)-totalResolved)},
	)
}

func (s *Service) reconcileSinglePayment(ctx context.Context, pmt Payment) bool {
	var msg string
	if pmt.ProviderOrderID == nil {
		s.logger.Error(ctx, "gateway payment is missing provider order id",
			applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
		)
		return false
	}

	gatewayOrderID := *pmt.ProviderOrderID
	result, err := s.paymentGateway.GetTransactionStatus(ctx, gatewayOrderID)
	if err != nil {
		msg = "failed to fetch transaction status"
		s.logger.Error(ctx, msg,
			applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
			applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
			applogger.Field{Key: "error", Value: err.Error()},
		)
		return false
	}

	if result.Status == paymentgateway.NotificationStatusPending {
		if pmt.ExpiresAt != nil && appclock.Now().After(*pmt.ExpiresAt) {
			msg := "payment has expired locally, cancelling at gateway and expiring locally"
			s.logger.Info(ctx, msg,
				applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
				applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
			)

			// Best effort cancel at gateway
			if err := s.paymentGateway.CancelTransaction(ctx, gatewayOrderID); err != nil {
				s.logger.Warn(ctx, "failed to cancel transaction at gateway, proceeding with local expiry",
					applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
					applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
					applogger.Field{Key: "error", Value: err.Error()},
				)
			}

			if err := s.expirePaymentLocally(ctx, pmt); err != nil {
				s.logger.Error(ctx, "failed to locally expire payment",
					applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
					applogger.Field{Key: "error", Value: err.Error()},
				)
				return false
			}
			return true
		}
		return false
	}

	// Drive resolved status directly through the webhook processing pipeline
	// without redundant outbound network calls.
	if err := s.processWebhook.Execute(ctx, ProcessPaymentWebhookInput{
		NotificationResult: result,
	}); err != nil {
		msg := "failed to process reconciled payment"
		s.logger.Error(ctx, msg,
			applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
			applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
			applogger.Field{Key: "gateway_status", Value: string(result.Status)},
			applogger.Field{Key: "error", Value: err.Error()},
		)
		return false
	}

	msg = "successfully reconciled payment"
	s.logger.Info(ctx, msg,
		applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
		applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
		applogger.Field{Key: "gateway_status", Value: string(result.Status)},
	)
	return true
}

func (s *Service) expirePaymentLocally(ctx context.Context, pmt Payment) error {
	return s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		// Acquire pessimistic row lock to guard against concurrent settlement webhooks.
		current, err := s.paymentRepo.GetByOrderIDForUpdate(ctx, exec, pmt.OrderID)
		if err != nil {
			return fmt.Errorf("failed to lock payment for expiry: %w", err)
		}
		if current == nil || current.Status != PaymentStatusPending {
			return nil
		}

		if err := s.paymentRepo.UpdateStatus(ctx, exec,
			pmt.ID,
			PaymentStatusExpired,
		); err != nil {
			return fmt.Errorf("failed to update payment status: %w", err)
		}

		orderItems, err := s.orderMgr.ExpireOrderPayment(ctx, exec, pmt.OrderID)
		if err != nil {
			return fmt.Errorf("failed to expire order payment: %w", err)
		}

		for _, item := range orderItems {
			if err := s.inventoryRepo.Release(ctx, exec,
				item.ProductID,
				item.ShopID,
				item.Quantity,
			); err != nil {
				if errors.Is(err, inventory.ErrInsufficientReserved) ||
					errors.Is(err, apperror.ErrNotFound) {

					msg := "inventory anomaly during payment reconciliation expiry: reserved stock insufficient or missing"
					s.logger.Warn(ctx, msg,
						applogger.Field{Key: "payment_id", Value: pmt.ID.String()},
						applogger.Field{Key: "order_id", Value: pmt.OrderID.String()},
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
