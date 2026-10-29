package order

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/infra/outbox"
	shipping "komecore/internal/infra/shipping"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/product"
	"komecore/internal/modules/shipment"
	"komecore/internal/pagination"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

// DEFAULT_SHIPMENT_WEIGHT_GRAMS is a fallback weight (in grams) used when
// product weight is not specified on the product schema.
const DEFAULT_SHIPMENT_WEIGHT_GRAMS = 1000

// DEFAULT_SHIPMENT_ITEM_QTY is a placeholder item quantity used for the
// Komerce order creation call.
const DEFAULT_SHIPMENT_ITEM_QTY = 1

type OrderRefunder interface {
	ProcessOrderRefund(ctx context.Context, orderID uuid.UUID, reason string) error
}

type CustomerEmailResolver interface {
	GetEmailByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (string, error)
}

// Service coordinates order lifecycle workflows, state transitions,
// shipment tracking, and automated refund / cancellation compensation.
type Service struct {
	executor               transaction.Executor
	transactor             transaction.Transactor
	orderRepo              Repository
	orderItemRepo          ItemRepository
	inventoryRepo          inventory.Repository
	paymentRepo            payment.Repository
	paymentChannelDataRepo payment.ChannelDataRepository
	productRepo            product.Repository
	shipmentRepo           shipment.Repository
	addressRepo            address.CustomerAddressRepository
	shopAddressRepo        address.ShopAddressRepository
	logistics              shipping.LogisticsProvider
	refunder               OrderRefunder
	emailResolver          CustomerEmailResolver
	logger                 applogger.Logger
	auditLogger            applogger.AuditLogger
	outboxRecorder         OutboxRecorder
	batchSize              int
	concurrency            int
}

func NewService(
	executor transaction.Executor,
	transactor transaction.Transactor,
	orderRepo Repository,
	orderItemRepo ItemRepository,
	inventoryRepo inventory.Repository,
	paymentRepo payment.Repository,
	paymentChannelDataRepo payment.ChannelDataRepository,
	productRepo product.Repository,
	shipmentRepo shipment.Repository,
	addressRepo address.CustomerAddressRepository,
	shopAddressRepo address.ShopAddressRepository,
	logistics shipping.LogisticsProvider,
	refunder OrderRefunder,
	logger applogger.Logger,
	auditLogger applogger.AuditLogger,
) *Service {
	return &Service{
		executor:               executor,
		transactor:             transactor,
		orderRepo:              orderRepo,
		orderItemRepo:          orderItemRepo,
		inventoryRepo:          inventoryRepo,
		paymentRepo:            paymentRepo,
		paymentChannelDataRepo: paymentChannelDataRepo,
		productRepo:            productRepo,
		shipmentRepo:           shipmentRepo,
		addressRepo:            addressRepo,
		shopAddressRepo:        shopAddressRepo,
		logistics:              logistics,
		refunder:               refunder,
		logger:                 logger,
		auditLogger:            auditLogger,
		batchSize:              100,
		concurrency:            5,
	}
}

func (s *Service) WithOutboxRecorder(recorder OutboxRecorder) *Service {
	s.outboxRecorder = recorder
	return s
}

func (s *Service) WithCustomerEmailResolver(resolver CustomerEmailResolver) *Service {
	s.emailResolver = resolver
	return s
}

func (s *Service) WithExpiryConfig(batchSize, concurrency int) *Service {
	if batchSize > 0 {
		s.batchSize = batchSize
	}
	if concurrency > 0 {
		s.concurrency = concurrency
	}
	return s
}

func (s *Service) WithRefunder(refunder OrderRefunder) *Service {
	s.refunder = refunder
	return s
}

type FindOrdersInput struct {
	Page       int
	Limit      int
	ID         *uuid.UUID
	Number     *string
	CustomerID *uuid.UUID
	ShopID     *uuid.UUID
	ShopIDs    []uuid.UUID
	Status     *string
	Statuses   []string
	FromDate   *time.Time
	ToDate     *time.Time
	Sort       string
}

type OrderSearchResult struct {
	Order       Order
	Items       []OrderItem
	Payment     *payment.Payment
	ChannelData *payment.PaymentChannelData
	Shipment    *shipment.Shipment
	Shipments   []shipment.Shipment
	Address     *address.CustomerAddress
}

func (s *Service) FindOrders(
	ctx context.Context,
	input FindOrdersInput,
) ([]OrderSearchResult, int, error) {
	var orderSortKeys = map[string]pagination.SortKey{
		"latest":   OrderSortLatest,
		"date":     OrderSortLatest,
		"number":   OrderSortNumber,
		"total":    OrderSortTotal,
		"status":   OrderSortStatus,
		"modified": OrderSortModify,
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

			sortKey, exists := orderSortKeys[key]
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
				By:        OrderSortLatest,
				Direction: pagination.SortDesc,
			},
		}
	}

	var statuses []string
	if len(input.Statuses) > 0 {
		statuses = input.Statuses
	} else if input.Status != nil && *input.Status != "" {
		parts := strings.SplitSeq(*input.Status, ",")
		for p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				statuses = append(statuses, trimmed)
			}
		}
	}

	var statusParam *string
	if len(statuses) == 0 {
		statusParam = input.Status
	}

	params := FindOrderParams{
		ID:         input.ID,
		Number:     input.Number,
		CustomerID: input.CustomerID,
		ShopID:     input.ShopID,
		ShopIDs:    input.ShopIDs,
		Status:     statusParam,
		Statuses:   statuses,
		FromDate:   input.FromDate,
		ToDate:     input.ToDate,
		Pagination: pagination.Pagination{
			Page:  input.Page,
			Limit: input.Limit,
		},
		Sorts: sorts,
	}

	orders, total, err := s.orderRepo.FindOrders(ctx, s.executor, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}
	if len(orders) == 0 {
		return []OrderSearchResult{}, total, nil
	}

	orderIDs := make([]uuid.UUID, len(orders))
	for i, o := range orders {
		orderIDs[i] = o.ID
	}

	orderItems, err := s.orderItemRepo.ListByOrderIDs(ctx, s.executor, orderIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list order items: %w", err)
	}

	payments, err := s.paymentRepo.ListByOrderIDs(ctx, s.executor, orderIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}

	shipments, err := s.shipmentRepo.ListByOrderIDs(ctx, s.executor, orderIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list shipments: %w", err)
	}

	itemsMap := make(map[uuid.UUID][]OrderItem)
	for _, item := range orderItems {
		if input.ShopID != nil && item.ShopID != *input.ShopID {
			continue
		}
		if len(input.ShopIDs) > 0 && !slices.Contains(input.ShopIDs, item.ShopID) {
			continue
		}
		itemsMap[item.OrderID] = append(itemsMap[item.OrderID], item)
	}

	paymentsMap := make(map[uuid.UUID]*payment.Payment)
	for i := range payments {
		p := payments[i]
		paymentsMap[p.OrderID] = &p
	}

	// Collect payment IDs to bulk-fetch persisted channel data
	paymentIDs := make([]uuid.UUID, 0, len(payments))
	for i := range payments {
		paymentIDs = append(paymentIDs, payments[i].ID)
	}

	channelDataMap, err := s.paymentChannelDataRepo.ListByPaymentIDs(ctx, s.executor, paymentIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payment channel data: %w", err)
	}

	shipmentsMap := make(map[uuid.UUID][]shipment.Shipment)
	for i := range shipments {
		shp := shipments[i]
		shipmentsMap[shp.OrderID] = append(shipmentsMap[shp.OrderID], shp)
	}

	addressIDs := make([]uuid.UUID, 0, len(orders))
	addressSeen := make(map[uuid.UUID]bool)
	for _, o := range orders {
		if o.AddressID != uuid.Nil && !addressSeen[o.AddressID] {
			addressSeen[o.AddressID] = true
			addressIDs = append(addressIDs, o.AddressID)
		}
	}

	addresses, err := s.addressRepo.ListByIDs(ctx, s.executor, addressIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list customer addresses: %w", err)
	}

	addressesMap := make(map[uuid.UUID]*address.CustomerAddress)
	for i := range addresses {
		addr := addresses[i]
		addressesMap[addr.ID] = &addr
	}

	results := make([]OrderSearchResult, len(orders))
	for i, o := range orders {
		items := itemsMap[o.ID]
		if items == nil {
			items = []OrderItem{}
		}

		var channelData *payment.PaymentChannelData
		pmt := paymentsMap[o.ID]
		if pmt != nil {
			channelData = channelDataMap[pmt.ID]
		}

		orderShipments := shipmentsMap[o.ID]
		if orderShipments == nil {
			orderShipments = []shipment.Shipment{}
		}

		var filteredShipments []shipment.Shipment
		for j := range orderShipments {
			var itemIDs []uuid.UUID
			for _, itm := range items {
				if itm.ShipmentID != nil &&
					*itm.ShipmentID == orderShipments[j].ID {
					itemIDs = append(itemIDs, itm.ID)
				}
			}
			if len(itemIDs) > 0 || (input.ShopID == nil && len(input.ShopIDs) == 0) {
				filteredShipments = append(filteredShipments, orderShipments[j])
			}
		}

		var firstShipment *shipment.Shipment
		if len(filteredShipments) > 0 {
			firstShipment = &filteredShipments[0]
		}

		results[i] = OrderSearchResult{
			Order:       o,
			Items:       items,
			Payment:     pmt,
			ChannelData: channelData,
			Shipment:    firstShipment,
			Shipments:   filteredShipments,
			Address:     addressesMap[o.AddressID],
		}
	}

	return results, total, nil
}

type GetOrderInput struct {
	OrderID uuid.UUID

	// CustomerID, when set, enforces that the order
	// must belong to this customer.
	//
	// Use for customer-facing endpoints.
	// Leave nil for admin endpoints.
	CustomerID *uuid.UUID
}

type GetOrderResult struct {
	Order       Order
	Items       []OrderItem
	Payment     *payment.Payment
	ChannelData *payment.PaymentChannelData
	Shipment    *shipment.Shipment
	Shipments   []shipment.Shipment
}

func (s *Service) GetOrder(ctx context.Context, input GetOrderInput) (*GetOrderResult, error) {
	ord, err := s.orderRepo.GetByID(ctx, s.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if ord == nil {
		return nil, nil
	}

	if input.CustomerID != nil && ord.CustomerID != *input.CustomerID {
		return nil, nil
	}

	items, err := s.orderItemRepo.ListByOrderID(ctx, s.executor, ord.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list order items: %w", err)
	}

	pmt, err := s.paymentRepo.GetByOrderID(ctx, s.executor, ord.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	var channelData *payment.PaymentChannelData
	if pmt != nil {
		cd, err := s.paymentChannelDataRepo.GetByPaymentID(ctx, s.executor, pmt.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get payment channel data: %w", err)
		}
		channelData = cd
	}

	shipments, err := s.shipmentRepo.ListByOrderID(ctx, s.executor, ord.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list shipments: %w", err)
	}

	var firstShipment *shipment.Shipment
	if len(shipments) > 0 {
		firstShipment = &shipments[0]
	}

	return &GetOrderResult{
		Order:       *ord,
		Items:       items,
		Payment:     pmt,
		ChannelData: channelData,
		Shipment:    firstShipment,
		Shipments:   shipments,
	}, nil
}

type GetOrderTrackingInput struct {
	OrderID    uuid.UUID
	CustomerID uuid.UUID
	ShipmentID *uuid.UUID
}

type TrackingTimelineEvent struct {
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Timestamp   time.Time `json:"timestamp"`
}

type GetOrderTrackingResult struct {
	OrderID        uuid.UUID               `json:"order_id"`
	ShipmentID     uuid.UUID               `json:"shipment_id"`
	Courier        string                  `json:"courier"`
	TrackingNumber *string                 `json:"tracking_number,omitempty"`
	Warning        *string                 `json:"warning,omitempty"`
	Timeline       []TrackingTimelineEvent `json:"timeline"`
}

func (s *Service) GetOrderTracking(
	ctx context.Context,
	input GetOrderTrackingInput,
) (*GetOrderTrackingResult, error) {
	ord, err := s.orderRepo.GetByID(ctx, s.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if ord == nil {
		return nil, apperror.NewNotFoundWithCode(apperror.CodeOrderNotFound, "order not found")
	}

	if input.CustomerID != uuid.Nil && ord.CustomerID != input.CustomerID {
		return nil, apperror.NewForbidden("not authorized")
	}

	var shp *shipment.Shipment
	if input.ShipmentID != nil {
		sh, err := s.shipmentRepo.GetByID(ctx, s.executor, *input.ShipmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get shipment: %w", err)
		}
		if sh != nil && sh.OrderID == ord.ID {
			shp = sh
		}
	}

	if shp == nil {
		shipments, err := s.shipmentRepo.ListByOrderID(ctx, s.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list shipments: %w", err)
		}
		if len(shipments) > 0 {
			shp = &shipments[0]
		}
	}

	if shp == nil {
		return nil, apperror.NewNotFound("shipment not found")
	}

	var timeline []TrackingTimelineEvent
	timeline = append(timeline, TrackingTimelineEvent{
		Status:      string(shipment.ShipmentStatusCreated),
		Description: "Shipment created",
		Location:    "",
		Timestamp:   shp.CreatedAt,
	})

	if shp.ShippedAt != nil {
		timeline = append(timeline, TrackingTimelineEvent{
			Status:      string(shipment.ShipmentStatusShipped),
			Description: fmt.Sprintf("Shipped via %s (%s)", shp.Courier, shp.Service),
			Location:    "",
			Timestamp:   *shp.ShippedAt,
		})
	}
	if shp.DeliveredAt != nil {
		timeline = append(timeline, TrackingTimelineEvent{
			Status:      string(shipment.ShipmentStatusDelivered),
			Description: "Shipment delivered",
			Location:    "",
			Timestamp:   *shp.DeliveredAt,
		})
	}

	return &GetOrderTrackingResult{
		OrderID:        ord.ID,
		ShipmentID:     shp.ID,
		Courier:        shp.Courier,
		TrackingNumber: shp.TrackingNumber,
		Timeline:       timeline,
	}, nil
}

type ShipmentDispatchInput struct {
	FulfillmentMethod string      `json:"fulfillment_method"`
	Courier           string      `json:"courier"`
	Service           string      `json:"service"`
	TrackingNumber    *string     `json:"tracking_number"`
	ItemIDs           []uuid.UUID `json:"item_ids"`
}

type UpdateOrderStatusInput struct {
	OrderID uuid.UUID
	Status  OrderStatus

	// TrackingNumber is an optional override used when the server is running
	// in manual logistics mode. Automated providers (e.g. Komerce) ignore it.
	TrackingNumber *string

	// FulfillmentMethod is an optional override. If not provided, it defaults
	// to "courier".
	FulfillmentMethod *string

	// Shipments allows staff to explicitly split/group order items into
	// specific shipments with separate couriers and tracking numbers.
	Shipments []ShipmentDispatchInput
}

type preparedShipment struct {
	shipment shipment.Shipment
	itemIDs  []uuid.UUID
}

type UpdateOrderStatusResult struct {
	Order     Order
	Shipment  *shipment.Shipment
	Shipments []shipment.Shipment
}

// UpdateOrderStatus drives order state transitions, cascading inventory release/restock,
// logistics cancellation, and automated refunds within a single atomic database transaction.
func (s *Service) UpdateOrderStatus(ctx context.Context, input UpdateOrderStatusInput) (res *UpdateOrderStatusResult, err error) {
	var oldStatus string
	defer func() {
		if s.auditLogger == nil {
			return
		}
		if err != nil {
			s.auditLogger.Log(ctx, applogger.AuditEvent{
				Category:   "user_action",
				Action:     "update_order_status",
				Resource:   "order",
				ResourceID: input.OrderID.String(),
				Outcome:    applogger.OutcomeFailure,
				Metadata: map[string]any{
					"error":      err.Error(),
					"old_status": oldStatus,
					"new_status": string(input.Status),
				},
			})
		} else {
			s.auditLogger.Log(ctx, applogger.AuditEvent{
				Category:   "user_action",
				Action:     "update_order_status",
				Resource:   "order",
				ResourceID: input.OrderID.String(),
				Outcome:    applogger.OutcomeSuccess,
				Metadata: map[string]any{
					"old_status": oldStatus,
					"new_status": string(input.Status),
				},
			})
		}
	}()

	ord, err := s.orderRepo.GetByID(ctx, s.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if ord == nil {
		return nil, apperror.NewNotFoundWithCode(apperror.CodeOrderNotFound, "order not found")
	}

	oldStatus = string(ord.Status)

	// Confirm unconfirmed orders to stamp ConfirmedAt and establish the 3-day handling SLA window.
	if input.Status == OrderStatusConfirmed && ord.ConfirmedAt == nil {
		if errStatus := ord.Confirm(appclock.Now(), DefaultHandlingSLAWindow); errStatus != nil {
			return nil, apperror.NewInvalidInput(errStatus.Error())
		}
	} else {
		if errStatus := ord.UpdateStatus(input.Status); errStatus != nil {
			return nil, apperror.NewInvalidInput(errStatus.Error())
		}
	}

	switch input.Status {
	case OrderStatusConfirmed:
		pmt, err := s.paymentRepo.GetByOrderID(ctx, s.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get payment: %w", err)
		}
		if pmt == nil || pmt.Status != payment.PaymentStatusPaid {
			return nil, apperror.NewInvalidInput("cannot confirm order without confirmed payment")
		}

		items, err := s.orderItemRepo.ListByOrderID(ctx, s.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list order items: %w", err)
		}

		err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			for _, item := range items {
				if err := s.inventoryRepo.Commit(ctx, exec, item.ProductID, item.ShopID, item.Quantity); err != nil {
					return fmt.Errorf("failed to commit inventory for product %s: %w", item.ProductID, err)
				}
			}
			if err := s.orderRepo.UpdateStatusWithSLA(ctx, exec, ord.ID, input.Status, ord.ConfirmedAt, ord.HandlingExpiresAt); err != nil {
				return fmt.Errorf("failed to update order status: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		result := UpdateOrderStatusResult{Order: *ord}
		return &result, nil

	case OrderStatusProcessing:
		pmt, err := s.paymentRepo.GetByOrderID(ctx, s.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get payment: %w", err)
		}
		if pmt == nil || pmt.Status != payment.PaymentStatusPaid {
			return nil, apperror.NewInvalidInput("cannot move order to processing without confirmed payment")
		}

		err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			return s.orderRepo.UpdateStatus(ctx, exec,
				ord.ID,
				input.Status,
			)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}

		result := UpdateOrderStatusResult{Order: *ord}
		return &result, nil

	case OrderStatusDelivered:
		shipments, err := s.shipmentRepo.ListByOrderID(ctx, s.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list shipments: %w", err)
		}

		err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			for i := range shipments {
				shp := shipments[i]
				if err := shp.UpdateStatus(shipment.ShipmentStatusDelivered); err != nil {
					return fmt.Errorf("failed to update shipment status: %w", err)
				}
				if err := s.shipmentRepo.Update(ctx, exec, shp); err != nil {
					return fmt.Errorf("failed to persist shipment: %w", err)
				}
			}
			return s.orderRepo.UpdateStatus(ctx, exec, ord.ID, input.Status)
		})
		if err != nil {
			return nil, err
		}

		var first *shipment.Shipment
		if len(shipments) > 0 {
			first = &shipments[0]
		}

		result := UpdateOrderStatusResult{Order: *ord, Shipment: first, Shipments: shipments}
		return &result, nil

	case OrderStatusCancelled:
		items, err := s.orderItemRepo.ListByOrderID(ctx, s.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list order items: %w", err)
		}

		err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			for _, item := range items {
				// Restock returns committed goods to available inventory for confirmed orders,
				// whereas Release drops uncommitted holds from pending reservations.
				if oldStatus == string(OrderStatusConfirmed) || oldStatus == string(OrderStatusProcessing) {
					if err := s.inventoryRepo.Restock(ctx, exec, item.ProductID, item.ShopID, item.Quantity); err != nil {
						return fmt.Errorf("failed to restock inventory for product %s: %w", item.ProductID, err)
					}
				} else {
					if err := s.inventoryRepo.Release(ctx, exec, item.ProductID, item.ShopID, item.Quantity); err != nil {
						return fmt.Errorf("failed to release inventory for product %s: %w", item.ProductID, err)
					}
				}
			}
			return s.orderRepo.UpdateStatus(ctx, exec, ord.ID, input.Status)
		})
		if err != nil {
			return nil, err
		}

		// Delegate customer refund when cancelling an already paid/confirmed order.
		if (oldStatus == string(OrderStatusConfirmed) || oldStatus == string(OrderStatusProcessing)) && s.refunder != nil {
			_ = s.refunder.ProcessOrderRefund(ctx, ord.ID, "Order cancelled by staff")
		}

		result := UpdateOrderStatusResult{Order: *ord}
		return &result, nil

	case OrderStatusExpired:
		items, err := s.orderItemRepo.ListByOrderID(ctx, s.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list order items: %w", err)
		}

		err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			for _, item := range items {
				// Restock returns committed goods to available inventory for confirmed orders,
				// whereas Release drops uncommitted holds from pending reservations.
				if oldStatus == string(OrderStatusConfirmed) || oldStatus == string(OrderStatusProcessing) {
					if err := s.inventoryRepo.Restock(ctx, exec, item.ProductID, item.ShopID, item.Quantity); err != nil {
						return fmt.Errorf("failed to restock inventory for product %s: %w", item.ProductID, err)
					}
				} else {
					if err := s.inventoryRepo.Release(ctx, exec, item.ProductID, item.ShopID, item.Quantity); err != nil {
						return fmt.Errorf("failed to release inventory for product %s: %w", item.ProductID, err)
					}
				}
			}
			return s.orderRepo.UpdateStatus(ctx, exec, ord.ID, input.Status)
		})
		if err != nil {
			return nil, err
		}

		// Delegate customer refund when cancelling an already paid/confirmed order.
		if (oldStatus == string(OrderStatusConfirmed) || oldStatus == string(OrderStatusProcessing)) && s.refunder != nil {
			_ = s.refunder.ProcessOrderRefund(ctx, ord.ID, "Staff handling SLA (3 days) expired without fulfillment")
		}

		result := UpdateOrderStatusResult{Order: *ord}
		return &result, nil

	case OrderStatusShipped:
		// processing -> shipped
		items, err := s.orderItemRepo.ListByOrderID(ctx, s.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list order items: %w", err)
		}
		if len(items) == 0 {
			return nil, apperror.NewInvalidInput("order has no items")
		}

		itemMap := make(map[uuid.UUID]OrderItem, len(items))
		for _, item := range items {
			itemMap[item.ID] = item
		}

		// Resolve customer destination district ID from the order's address
		customerAddr, err := s.addressRepo.GetByID(ctx, s.executor, ord.AddressID)
		if err != nil {
			return nil, fmt.Errorf("failed to get customer address: %w", err)
		}
		if customerAddr == nil {
			return nil, apperror.NewNotFound("customer address not found")
		}

		var productIDs []uuid.UUID
		destAreaID := 0

		// Fetch product details for all items to calculate accurate shipping weights
		for _, item := range items {
			productIDs = append(productIDs, item.ProductID)
		}
		products, err := s.productRepo.FindByIDs(ctx, s.executor, productIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to load products: %w", err)
		}
		productMap := make(map[uuid.UUID]product.Product, len(products))
		for _, p := range products {
			productMap[p.ID] = p
		}

		now := appclock.Now()
		var preparedShipments []preparedShipment
		var createdLogisticsOrders []string
		rollbackLogistics := func() {
			for _, orderNo := range createdLogisticsOrders {
				_ = s.logistics.CancelOrder(ctx, orderNo)
			}
		}

		if len(input.Shipments) > 0 {
			// Enforce exhaustive item partitioning: each order item must appear in exactly one
			// shipment package with zero duplicate assignments or unallocated items.
			assignedItemIDs := make(map[uuid.UUID]bool, len(items))
			for idx, sInput := range input.Shipments {
				if len(sInput.ItemIDs) == 0 {
					return nil, apperror.NewInvalidInput("each shipment must contain at least one order item")
				}

				var shipmentItems []OrderItem
				for _, itemID := range sInput.ItemIDs {
					if assignedItemIDs[itemID] {
						return nil, apperror.NewInvalidInput(fmt.Sprintf("item %s is assigned to multiple shipments", itemID))
					}
					assignedItemIDs[itemID] = true

					item, ok := itemMap[itemID]
					if !ok {
						return nil, apperror.NewInvalidInput(fmt.Sprintf("order item %s does not belong to this order", itemID))
					}
					shipmentItems = append(shipmentItems, item)
				}

				method := shipment.FulfillmentMethodCourier
				if sInput.FulfillmentMethod != "" {
					method = shipment.FulfillmentMethod(sInput.FulfillmentMethod)
				}

				var courierCode, courierService string
				first := shipmentItems[0]
				if method == shipment.FulfillmentMethodCourier {
					courierCode = sInput.Courier
					courierService = sInput.Service
					if courierCode == "" && first.CourierCode != nil {
						courierCode = *first.CourierCode
					}
					if courierService == "" && first.CourierService != nil {
						courierService = *first.CourierService
					}
					if courierCode == "" || courierService == "" {
						return nil, apperror.NewInvalidInput("shipment is missing courier information")
					}
				} else {
					courierCode = string(shipment.FulfillmentMethodSelfDelivery)
					courierService = string(shipment.FulfillmentMethodSelfDelivery)
				}

				shopAddr, err := s.shopAddressRepo.GetDefaultByShopID(ctx, s.executor, first.ShopID)
				if err != nil {
					return nil, fmt.Errorf("failed to get shop address: %w", err)
				}
				if shopAddr == nil {
					return nil, apperror.NewNotFound("shop address not found")
				}

				var (
					totalWeightGrams int
					totalItemQty     int
					subtotal         int64

					originAreaID = 0
				)

				for _, item := range shipmentItems {
					qty := item.Quantity
					if qty <= 0 {
						qty = 1
					}
					weight := DEFAULT_SHIPMENT_WEIGHT_GRAMS
					if p, ok := productMap[item.ProductID]; ok && p.Weight != nil && *p.Weight > 0 {
						weight = int(*p.Weight)
					}
					totalWeightGrams += weight * qty
					totalItemQty += qty
					subtotal += item.Subtotal
				}

				trackingNumber := sInput.TrackingNumber
				if method == shipment.FulfillmentMethodCourier && (trackingNumber == nil || *trackingNumber == "") {
					itemName := first.ProductName
					if len(shipmentItems) > 1 {
						itemName = fmt.Sprintf("%s (+%d more)", first.ProductName, len(shipmentItems)-1)
					}

					uniqueOrderID := ord.Number
					if len(input.Shipments) > 1 {
						uniqueOrderID = fmt.Sprintf("%s-%d", ord.Number, idx+1)
					}

					orderInput := shipping.CreateOrderInput{
						OriginAreaID:         originAreaID,
						DestinationAreaID:    destAreaID,
						CourierCode:          courierCode,
						CourierService:       courierService,
						Weight:               totalWeightGrams,
						UniqueOrderID:        uniqueOrderID,
						ItemName:             itemName,
						ItemPrice:            subtotal,
						ItemQty:              totalItemQty,
						ShipperName:          shopAddr.Label,
						ShipperPhone:         derefPhone(shopAddr.Phone),
						ShipperAddress:       shopAddr.Detail.FullAddress,
						ReceiverName:         customerAddr.ReceiverName,
						ReceiverPhone:        derefPhone(customerAddr.Phone),
						ReceiverAddress:      customerAddr.Detail.FullAddress,
						ManualTrackingNumber: sInput.TrackingNumber,
					}

					komerceResult, err := s.logistics.CreateOrder(ctx, orderInput)
					if err != nil {
						rollbackLogistics()
						return nil, fmt.Errorf("failed to create shipment order: %w", err)
					}
					if komerceResult != nil && komerceResult.KomerceOrderNo != "" {
						createdLogisticsOrders = append(createdLogisticsOrders, komerceResult.KomerceOrderNo)
					}

					tracking := komerceResult.TrackingNumber
					trackingNumber = &tracking
				}

				shipmentCost := first.ShippingFee
				if shipmentCost == 0 && len(input.Shipments) == 1 {
					shipmentCost = ord.ShippingFee
				}

				shp := shipment.Shipment{
					ID:                uuid.New(),
					OrderID:           ord.ID,
					Status:            shipment.ShipmentStatusCreated,
					FulfillmentMethod: method,
					TrackingNumber:    trackingNumber,
					Courier:           courierCode,
					Service:           courierService,
					Cost:              shipmentCost,
					Weight:            totalWeightGrams,
					CreatedAt:         now,
				}

				if err := shp.Validate(); err != nil {
					rollbackLogistics()
					return nil, apperror.NewInvalidInput(err.Error())
				}

				preparedShipments = append(preparedShipments, preparedShipment{
					shipment: shp,
					itemIDs:  sInput.ItemIDs,
				})
			}
			if len(assignedItemIDs) != len(items) {
				rollbackLogistics()
				return nil, apperror.NewInvalidInput("all order items must be assigned to a shipment when transitioning order to shipped")
			}
		} else {
			// Fallback / legacy: Group order items by ShopID for per-shop shipment processing
			type shopGroup struct {
				shopID uuid.UUID
				items  []OrderItem
			}
			var groups []shopGroup
			groupMap := make(map[uuid.UUID]int)
			for _, item := range items {
				idx, exists := groupMap[item.ShopID]
				if !exists {
					groupMap[item.ShopID] = len(groups)
					groups = append(groups, shopGroup{
						shopID: item.ShopID,
						items:  []OrderItem{item},
					})
				} else {
					groups[idx].items = append(groups[idx].items, item)
				}
			}

			method := shipment.FulfillmentMethodCourier
			if input.FulfillmentMethod != nil && *input.FulfillmentMethod != "" {
				method = shipment.FulfillmentMethod(*input.FulfillmentMethod)
			}

			for idx, group := range groups {
				first := group.items[0]

				var courierCode, courierService string
				if method == shipment.FulfillmentMethodCourier {
					if first.CourierCode != nil {
						courierCode = *first.CourierCode
					}
					if first.CourierService != nil {
						courierService = *first.CourierService
					}
					if courierCode == "" || courierService == "" {
						return nil, apperror.NewInvalidInput("order items have no courier information")
					}
				} else {
					courierCode = string(shipment.FulfillmentMethodSelfDelivery)
					courierService = string(shipment.FulfillmentMethodSelfDelivery)
				}

				shopAddr, err := s.shopAddressRepo.GetDefaultByShopID(ctx, s.executor, group.shopID)
				if err != nil {
					return nil, fmt.Errorf("failed to get shop address: %w", err)
				}
				if shopAddr == nil {
					return nil, apperror.NewNotFound("shop address not found")
				}

				var (
					shopTotalWeightGrams int
					shopTotalItemQty     int
					shopSubtotal         int64
					groupItemIDs         []uuid.UUID

					originAreaID = 0
				)
				for _, item := range group.items {
					groupItemIDs = append(groupItemIDs, item.ID)
					qty := item.Quantity
					if qty <= 0 {
						qty = 1
					}
					weight := DEFAULT_SHIPMENT_WEIGHT_GRAMS
					if p, ok := productMap[item.ProductID]; ok && p.Weight != nil && *p.Weight > 0 {
						weight = int(*p.Weight)
					}
					shopTotalWeightGrams += weight * qty
					shopTotalItemQty += qty
					shopSubtotal += item.Subtotal
				}

				var trackingNumber *string
				if method == shipment.FulfillmentMethodCourier {
					itemName := first.ProductName
					if len(group.items) > 1 {
						itemName = fmt.Sprintf("%s (+%d more)", first.ProductName, len(group.items)-1)
					}

					uniqueOrderID := ord.Number
					if len(groups) > 1 {
						uniqueOrderID = fmt.Sprintf("%s-%d", ord.Number, idx+1)
					}

					orderInput := shipping.CreateOrderInput{
						OriginAreaID:         originAreaID,
						DestinationAreaID:    destAreaID,
						CourierCode:          courierCode,
						CourierService:       courierService,
						Weight:               shopTotalWeightGrams,
						UniqueOrderID:        uniqueOrderID,
						ItemName:             itemName,
						ItemPrice:            shopSubtotal,
						ItemQty:              shopTotalItemQty,
						ShipperName:          shopAddr.Label,
						ShipperPhone:         derefPhone(shopAddr.Phone),
						ShipperAddress:       shopAddr.Detail.FullAddress,
						ReceiverName:         customerAddr.ReceiverName,
						ReceiverPhone:        derefPhone(customerAddr.Phone),
						ReceiverAddress:      customerAddr.Detail.FullAddress,
						ManualTrackingNumber: input.TrackingNumber,
					}

					komerceResult, err := s.logistics.CreateOrder(ctx, orderInput)
					if err != nil {
						rollbackLogistics()
						return nil, fmt.Errorf("failed to create Komerce shipment order: %w", err)
					}
					if komerceResult != nil && komerceResult.KomerceOrderNo != "" {
						createdLogisticsOrders = append(createdLogisticsOrders, komerceResult.KomerceOrderNo)
					}

					tracking := komerceResult.TrackingNumber
					trackingNumber = &tracking
				}

				shipmentCost := first.ShippingFee
				if shipmentCost == 0 && len(groups) == 1 {
					shipmentCost = ord.ShippingFee
				}

				shp := shipment.Shipment{
					ID:                uuid.New(),
					OrderID:           ord.ID,
					Status:            shipment.ShipmentStatusCreated,
					FulfillmentMethod: method,
					TrackingNumber:    trackingNumber,
					Courier:           courierCode,
					Service:           courierService,
					Cost:              shipmentCost,
					Weight:            shopTotalWeightGrams,
					CreatedAt:         now,
				}

				if err := shp.Validate(); err != nil {
					rollbackLogistics()
					return nil, apperror.NewInvalidInput(err.Error())
				}

				preparedShipments = append(preparedShipments, preparedShipment{
					shipment: shp,
					itemIDs:  groupItemIDs,
				})
			}
		}

		var createdShipments []shipment.Shipment
		err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			for _, ps := range preparedShipments {
				if err := s.shipmentRepo.Create(ctx, exec, ps.shipment); err != nil {
					return fmt.Errorf("failed to persist shipment: %w", err)
				}
				if err := s.orderItemRepo.AssignShipment(ctx, exec, ps.shipment.ID, ps.itemIDs); err != nil {
					return fmt.Errorf("failed to link items to shipment: %w", err)
				}
				createdShipments = append(createdShipments, ps.shipment)
			}
			if err := s.orderRepo.UpdateStatus(ctx, exec, ord.ID, input.Status); err != nil {
				return fmt.Errorf("failed to update order status: %w", err)
			}

			if s.outboxRecorder != nil {
				customerEmail := ""
				if s.emailResolver != nil {
					if email, err := s.emailResolver.GetEmailByCustomerID(ctx, exec, ord.CustomerID); err == nil {
						customerEmail = email
					}
				}
				for _, ps := range preparedShipments {
					trackingNo := ""
					if ps.shipment.TrackingNumber != nil {
						trackingNo = *ps.shipment.TrackingNumber
					}
					outboxPayload := outbox.ShipmentDispatchedPayload{
						OrderID:        ord.ID,
						OrderNumber:    ord.Number,
						CustomerEmail:  customerEmail,
						CustomerName:   customerAddr.ReceiverName,
						TrackingNumber: trackingNo,
						Courier:        ps.shipment.Courier,
						Service:        ps.shipment.Service,
					}
					if err := s.outboxRecorder.Enqueue(ctx, exec, outbox.EventShipmentDispatched, outboxPayload); err != nil {
						return fmt.Errorf("failed to enqueue outbox shipment.dispatched event: %w", err)
					}
				}
			}

			return nil
		})
		if err != nil {
			rollbackLogistics()
			return nil, err
		}

		var firstCreated *shipment.Shipment
		if len(createdShipments) > 0 {
			firstCreated = &createdShipments[0]
		}

		return &UpdateOrderStatusResult{
			Order:     *ord,
			Shipment:  firstCreated,
			Shipments: createdShipments,
		}, nil

	default:
		return nil, apperror.NewInvalidInput(fmt.Sprintf("unsupported status transition to %s", input.Status))
	}
}

func (s *Service) MarkOrderDelivered(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) error {
	if exec == nil {
		exec = s.executor
	}
	ord, err := s.orderRepo.GetByID(ctx, exec, orderID)
	if err != nil {
		return fmt.Errorf("failed to fetch order: %w", err)
	}
	if ord == nil {
		return apperror.NewNotFound("order not found")
	}
	if err := ord.UpdateStatus(OrderStatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	if err := s.orderRepo.UpdateStatus(ctx, exec, ord.ID, OrderStatusDelivered); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

type DispatchShopShipmentInput struct {
	OrderID           uuid.UUID
	ShopID            uuid.UUID
	FulfillmentMethod string
	Courier           string
	Service           string
	TrackingNumber    *string
	ItemIDs           []uuid.UUID
}

type DispatchShopShipmentResult struct {
	Order           Order
	Shipment        shipment.Shipment
	AllItemsShipped bool
}

// DispatchShopShipment handles partial fulfillment for a single merchant shop,
// creating courier shipments and advancing the order to Shipped when all items are dispatched.
func (s *Service) DispatchShopShipment(
	ctx context.Context,
	input DispatchShopShipmentInput,
) (res *DispatchShopShipmentResult, err error) {
	defer func() {
		if err != nil && s.auditLogger != nil {
			s.auditLogger.Log(ctx, applogger.AuditEvent{
				Category:   "user_action",
				Action:     "dispatch_shop_shipment",
				Resource:   "order",
				ResourceID: input.OrderID.String(),
				Outcome:    applogger.OutcomeFailure,
				Metadata: map[string]any{
					"error":   err.Error(),
					"shop_id": input.ShopID.String(),
				},
			})
		}
	}()

	ord, err := s.orderRepo.GetByID(ctx, s.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if ord == nil {
		return nil, apperror.NewNotFoundWithCode(apperror.CodeOrderNotFound, "order not found")
	}
	if ord.Status != OrderStatusProcessing {
		return nil, apperror.NewConflict(
			fmt.Sprintf("cannot dispatch shipment for order in '%s' status", ord.Status),
		)
	}

	allItems, err := s.orderItemRepo.ListByOrderID(ctx, s.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	if len(input.ItemIDs) == 0 {
		return nil, apperror.NewInvalidInput("item_ids cannot be empty")
	}

	itemMap := make(map[uuid.UUID]OrderItem)
	for _, item := range allItems {
		itemMap[item.ID] = item
	}

	var shipmentItems []OrderItem
	seenItemIDs := make(map[uuid.UUID]bool, len(input.ItemIDs))
	for _, itemID := range input.ItemIDs {
		if seenItemIDs[itemID] {
			return nil, apperror.NewInvalidInput(fmt.Sprintf("duplicate item %s in item_ids", itemID))
		}
		seenItemIDs[itemID] = true

		item, exists := itemMap[itemID]
		if !exists {
			return nil, apperror.NewNotFound(
				fmt.Sprintf("item %s not found in order", itemID),
			)
		}
		if item.ShopID != input.ShopID {
			return nil, apperror.NewConflict(
				fmt.Sprintf("item %s does not belong to shop %s", itemID, input.ShopID),
			)
		}
		if item.ShipmentID != nil {
			return nil, apperror.NewConflict(
				fmt.Sprintf("item %s has already been shipped", itemID),
			)
		}
		shipmentItems = append(shipmentItems, item)
	}

	first := shipmentItems[0]
	method := shipment.FulfillmentMethodCourier
	if input.FulfillmentMethod != "" {
		method = shipment.FulfillmentMethod(input.FulfillmentMethod)
	}

	courierCode := input.Courier
	if courierCode == "" && first.CourierCode != nil {
		courierCode = *first.CourierCode
	}
	courierService := input.Service
	if courierService == "" && first.CourierService != nil {
		courierService = *first.CourierService
	}

	customerAddr, err := s.addressRepo.GetByID(ctx, s.executor, ord.AddressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer address: %w", err)
	}
	if customerAddr == nil {
		return nil, apperror.NewNotFound("customer address not found")
	}

	shopAddr, err := s.shopAddressRepo.GetDefaultByShopID(ctx, s.executor, input.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop address: %w", err)
	}
	if shopAddr == nil {
		return nil, apperror.NewNotFound("shop address not found")
	}

	var productIDs []uuid.UUID
	for _, item := range shipmentItems {
		productIDs = append(productIDs, item.ProductID)
	}
	products, err := s.productRepo.FindByIDs(ctx, s.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load products: %w", err)
	}
	productMap := make(map[uuid.UUID]product.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}

	var totalWeightGrams int
	var totalItemQty int
	var subtotal int64
	for _, item := range shipmentItems {
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		weight := DEFAULT_SHIPMENT_WEIGHT_GRAMS
		if p, ok := productMap[item.ProductID]; ok && p.Weight != nil && *p.Weight > 0 {
			weight = int(*p.Weight)
		}
		totalWeightGrams += weight * qty
		totalItemQty += qty
		subtotal += item.Subtotal
	}

	var createdLogisticsOrderNo string
	rollbackLogistics := func() {
		if createdLogisticsOrderNo != "" {
			_ = s.logistics.CancelOrder(ctx, createdLogisticsOrderNo)
		}
	}

	trackingNumber := input.TrackingNumber
	if method == shipment.FulfillmentMethodCourier && (trackingNumber == nil || *trackingNumber == "") {
		itemName := first.ProductName
		if len(shipmentItems) > 1 {
			itemName = fmt.Sprintf("%s (+%d more)", first.ProductName, len(shipmentItems)-1)
		}

		uniqueOrderID := fmt.Sprintf("%s-SH-%s", ord.Number, input.ShopID.String()[:8])

		orderInput := shipping.CreateOrderInput{
			OriginAreaID:         0,
			DestinationAreaID:    0,
			CourierCode:          courierCode,
			CourierService:       courierService,
			Weight:               totalWeightGrams,
			UniqueOrderID:        uniqueOrderID,
			ItemName:             itemName,
			ItemPrice:            subtotal,
			ItemQty:              totalItemQty,
			ShipperName:          shopAddr.Label,
			ShipperPhone:         derefPhone(shopAddr.Phone),
			ShipperAddress:       shopAddr.Detail.FullAddress,
			ReceiverName:         customerAddr.ReceiverName,
			ReceiverPhone:        derefPhone(customerAddr.Phone),
			ReceiverAddress:      customerAddr.Detail.FullAddress,
			ManualTrackingNumber: input.TrackingNumber,
		}

		komerceResult, err := s.logistics.CreateOrder(ctx, orderInput)
		if err != nil {
			return nil, fmt.Errorf("failed to create shipment order: %w", err)
		}
		if komerceResult != nil && komerceResult.KomerceOrderNo != "" {
			createdLogisticsOrderNo = komerceResult.KomerceOrderNo
		}

		if komerceResult != nil && komerceResult.TrackingNumber != "" {
			tracking := komerceResult.TrackingNumber
			trackingNumber = &tracking
		}
	}

	shipmentCost := first.ShippingFee

	now := appclock.Now()
	shp := shipment.Shipment{
		ID:                uuid.New(),
		OrderID:           ord.ID,
		Status:            shipment.ShipmentStatusCreated,
		FulfillmentMethod: method,
		TrackingNumber:    trackingNumber,
		Courier:           courierCode,
		Service:           courierService,
		Cost:              shipmentCost,
		Weight:            totalWeightGrams,
		CreatedAt:         now,
	}

	if err := shp.Validate(); err != nil {
		rollbackLogistics()
		return nil, apperror.NewInvalidInput(err.Error())
	}

	unshippedCount := 0
	for _, item := range allItems {
		if item.ShipmentID == nil {
			unshippedCount++
		}
	}
	allItemsShipped := unshippedCount <= len(input.ItemIDs)

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.shipmentRepo.Create(ctx, exec, shp); err != nil {
			return fmt.Errorf("failed to persist shipment: %w", err)
		}
		if err := s.orderItemRepo.AssignShipment(ctx, exec, shp.ID, input.ItemIDs); err != nil {
			return fmt.Errorf("failed to link items to shipment: %w", err)
		}
		// Advance aggregate order status to Shipped only once all items across all shops are dispatched.
		if allItemsShipped {
			if err := s.orderRepo.UpdateStatus(ctx, exec, ord.ID, OrderStatusShipped); err != nil {
				return fmt.Errorf("failed to update order status to shipped: %w", err)
			}
			ord.Status = OrderStatusShipped
		}

		if s.outboxRecorder != nil {
			customerEmail := ""
			if s.emailResolver != nil {
				if email, err := s.emailResolver.GetEmailByCustomerID(ctx, exec, ord.CustomerID); err == nil {
					customerEmail = email
				}
			}
			trackingNo := ""
			if shp.TrackingNumber != nil {
				trackingNo = *shp.TrackingNumber
			}
			outboxPayload := outbox.ShipmentDispatchedPayload{
				OrderID:        ord.ID,
				OrderNumber:    ord.Number,
				CustomerEmail:  customerEmail,
				CustomerName:   customerAddr.ReceiverName,
				TrackingNumber: trackingNo,
				Courier:        shp.Courier,
				Service:        shp.Service,
			}
			if err := s.outboxRecorder.Enqueue(ctx, exec, outbox.EventShipmentDispatched, outboxPayload); err != nil {
				return fmt.Errorf("failed to enqueue outbox shipment.dispatched event: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		rollbackLogistics()
		return nil, err
	}

	return &DispatchShopShipmentResult{
		Order:           *ord,
		Shipment:        shp,
		AllItemsShipped: allItemsShipped,
	}, nil
}

// ExpireUnfulfilledOrders executes background cancellation of merchant orders
// exceeding the 3-day SLA using a bounded worker pool.
func (s *Service) ExpireUnfulfilledOrders(ctx context.Context) {
	if s == nil || s.orderRepo == nil {
		return
	}
	batchSize := s.batchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	concurrency := s.concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	now := appclock.Now()
	orders, err := s.orderRepo.FindExpiredUnfulfilledOrders(ctx, s.executor,
		now,
		batchSize,
	)
	if err != nil {
		if s.logger != nil {
			s.logger.Error(ctx, "failed to find expired unfulfilled orders",
				applogger.Field{Key: "error", Value: err.Error()},
			)
		}
		return
	}

	if len(orders) == 0 {
		return
	}

	if s.logger != nil {
		s.logger.Info(ctx, "order staff expiry job: processing unfulfilled orders exceeding 3 days SLA",
			applogger.Field{Key: "count", Value: len(orders)},
		)
	}

	orderChan := make(chan Order, len(orders))
	for _, o := range orders {
		orderChan <- o
	}
	close(orderChan)

	var (
		successCount int64
		wg           sync.WaitGroup
	)

	// Distribute expired orders across a bounded worker pool to prevent database
	// connection pool exhaustion while running concurrent status updates.
	numWorkers := min(len(orders), concurrency)
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ord := range orderChan {
				expired, err := s.expireSingleOrder(ctx, ord)
				if err != nil {
					if s.logger != nil {
						s.logger.Error(ctx, "failed to expire unfulfilled order",
							applogger.Field{Key: "order_id", Value: ord.ID.String()},
							applogger.Field{Key: "error", Value: err.Error()},
						)
					}
				} else if expired {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	if s.logger != nil {
		s.logger.Info(ctx, "order staff expiry job: processing complete",
			applogger.Field{Key: "total", Value: len(orders)},
			applogger.Field{Key: "expired_count", Value: atomic.LoadInt64(&successCount)},
		)
	}
}

func (s *Service) expireSingleOrder(ctx context.Context, ord Order) (bool, error) {
	var (
		oldStatus    string
		shouldRefund bool
	)

	err := s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		// Lock order row for update and verify active status.
		// Prevents concurrent processing of already handled orders.
		currentOrd, err := s.orderRepo.GetByIDForUpdate(ctx, exec, ord.ID)
		if err != nil {
			return fmt.Errorf("failed to lock order for expiry: %w", err)
		}
		if currentOrd == nil {
			return nil
		}
		// Skip restock and refund if status has already changed.
		// Handles concurrent updates from other replicas or workers.
		if currentOrd.Status != OrderStatusConfirmed && currentOrd.Status != OrderStatusProcessing {
			if s.logger != nil {
				s.logger.Info(ctx, "order already processed or no longer eligible for staff expiry, skipping",
					applogger.Field{Key: "order_id", Value: ord.ID.String()},
					applogger.Field{Key: "current_status", Value: string(currentOrd.Status)},
				)
			}
			return nil
		}

		oldStatus = string(currentOrd.Status)
		if err := currentOrd.UpdateStatus(OrderStatusExpired); err != nil {
			return fmt.Errorf("invalid domain state transition: %w", err)
		}

		if err := s.orderRepo.UpdateStatus(ctx, exec, currentOrd.ID, OrderStatusExpired); err != nil {
			return fmt.Errorf("failed to update order status to expired: %w", err)
		}

		items, err := s.orderItemRepo.ListByOrderID(ctx, exec, currentOrd.ID)
		if err != nil {
			return fmt.Errorf("failed to list order items: %w", err)
		}
		for _, item := range items {
			if err := s.inventoryRepo.Restock(ctx, exec,
				item.ProductID,
				item.ShopID,
				item.Quantity,
			); err != nil {
				if s.logger != nil {
					s.logger.Warn(ctx, "restock anomaly during order expiry",
						applogger.Field{Key: "order_id", Value: currentOrd.ID.String()},
						applogger.Field{Key: "product_id", Value: item.ProductID.String()},
						applogger.Field{Key: "reason", Value: err.Error()},
					)
				}
			}
		}

		shouldRefund = true
		return nil
	})
	if err != nil {
		if s.auditLogger != nil {
			s.auditLogger.Log(ctx, applogger.AuditEvent{
				Category:   "system_job",
				Action:     "expire_unfulfilled_order",
				Resource:   "order",
				ResourceID: ord.ID.String(),
				Outcome:    applogger.OutcomeFailure,
				Metadata: map[string]any{
					"error": err.Error(),
				},
			})
		}
		return false, err
	}
	if !shouldRefund {
		return false, nil
	}

	refundReason := "Staff handling SLA (3 days) expired without fulfillment"
	if s.refunder != nil {
		if refundErr := s.refunder.ProcessOrderRefund(ctx,
			ord.ID,
			refundReason,
		); refundErr != nil && s.logger != nil {
			errMsg := fmt.Sprintf("failed to trigger refund for expired order: %v", refundErr)
			s.logger.Error(ctx, errMsg,
				applogger.Field{Key: "order_id", Value: ord.ID.String()},
				applogger.Field{Key: "error", Value: refundErr.Error()},
			)
		}
	}

	if s.auditLogger != nil {
		s.auditLogger.Log(ctx, applogger.AuditEvent{
			Category:   "system_job",
			Action:     "expire_unfulfilled_order",
			Resource:   "order",
			ResourceID: ord.ID.String(),
			Outcome:    applogger.OutcomeSuccess,
			Metadata: map[string]any{
				"old_status": oldStatus,
				"new_status": string(OrderStatusExpired),
			},
		})
	}

	return true, nil
}

// derefPhone safely dereferences a nullable phone pointer, returning an
// empty string when nil. Komerce accepts empty phone numbers gracefully.
func derefPhone(phone *string) string {
	if phone == nil {
		return ""
	}
	return *phone
}
