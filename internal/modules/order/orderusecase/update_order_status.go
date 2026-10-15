package orderusecase

import (
	"context"
	"fmt"
	"komecore/internal/apperror"
	shipping "komecore/internal/infra/shipping"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/order"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/product"
	"komecore/internal/modules/shipment"
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

type UpdateOrderStatusUsecase struct {
	executor        transaction.Executor
	transactor      transaction.Transactor
	orderRepo       order.Repository
	orderItemRepo   order.ItemRepository
	inventoryRepo   inventory.Repository
	paymentRepo     payment.Repository
	productRepo     product.Repository
	shipmentRepo    shipment.Repository
	addressRepo     address.CustomerAddressRepository
	shopAddressRepo address.ShopAddressRepository
	logistics       shipping.LogisticsProvider
	auditLogger     applogger.AuditLogger
}

func NewUpdateOrderStatusUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	orderRepo order.Repository,
	orderItemRepo order.ItemRepository,
	inventoryRepo inventory.Repository,
	paymentRepo payment.Repository,
	productRepo product.Repository,
	shipmentRepo shipment.Repository,
	addressRepo address.CustomerAddressRepository,
	shopAddressRepo address.ShopAddressRepository,
	logistics shipping.LogisticsProvider,
	auditLogger applogger.AuditLogger,
) *UpdateOrderStatusUsecase {
	return &UpdateOrderStatusUsecase{
		executor:        executor,
		transactor:      transactor,
		orderRepo:       orderRepo,
		orderItemRepo:   orderItemRepo,
		inventoryRepo:   inventoryRepo,
		paymentRepo:     paymentRepo,
		productRepo:     productRepo,
		shipmentRepo:    shipmentRepo,
		addressRepo:     addressRepo,
		shopAddressRepo: shopAddressRepo,
		logistics:       logistics,
		auditLogger:     auditLogger,
	}
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
	Status  order.OrderStatus

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
	Order     order.Order
	Shipment  *shipment.Shipment
	Shipments []shipment.Shipment
}

func (u *UpdateOrderStatusUsecase) Execute(ctx context.Context, input UpdateOrderStatusInput) (res *UpdateOrderStatusResult, err error) {
	var oldStatus string
	defer func() {
		if err != nil {
			u.auditLogger.Log(ctx, applogger.AuditEvent{
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
			u.auditLogger.Log(ctx, applogger.AuditEvent{
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

	ord, err := u.orderRepo.GetByID(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if ord == nil {
		return nil, apperror.NewNotFoundWithCode(apperror.CodeOrderNotFound, "order not found")
	}

	oldStatus = string(ord.Status)

	// When confirming an unconfirmed order,
	// invoke the domain Confirm method to stamp
	//
	// ConfirmedAt and calculate the 3-day handling SLA expiration
	// (HandlingExpiresAt).
	if input.Status == order.OrderStatusConfirmed && ord.ConfirmedAt == nil {
		if errStatus := ord.Confirm(appclock.Now(), order.DefaultHandlingSLAWindow); errStatus != nil {
			return nil, apperror.NewInvalidInput(errStatus.Error())
		}
	} else {
		if errStatus := ord.UpdateStatus(input.Status); errStatus != nil {
			return nil, apperror.NewInvalidInput(errStatus.Error())
		}
	}

	switch input.Status {
	case order.OrderStatusConfirmed:
		pmt, err := u.paymentRepo.GetByOrderID(ctx, u.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get payment: %w", err)
		}
		if pmt == nil || pmt.Status != payment.PaymentStatusPaid {
			return nil, apperror.NewInvalidInput("cannot confirm order without confirmed payment")
		}

		items, err := u.orderItemRepo.ListByOrderID(ctx, u.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list order items: %w", err)
		}

		err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			for _, item := range items {
				if err := u.inventoryRepo.Commit(ctx, exec, item.ProductID, item.ShopID, item.Quantity); err != nil {
					return fmt.Errorf("failed to commit inventory for product %s: %w", item.ProductID, err)
				}
			}
			if err := u.orderRepo.UpdateStatusWithSLA(ctx, exec, ord.ID, input.Status, ord.ConfirmedAt, ord.HandlingExpiresAt); err != nil {
				return fmt.Errorf("failed to update order status: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		result := UpdateOrderStatusResult{Order: *ord}
		return &result, nil

	case order.OrderStatusProcessing:
		pmt, err := u.paymentRepo.GetByOrderID(ctx, u.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get payment: %w", err)
		}
		if pmt == nil || pmt.Status != payment.PaymentStatusPaid {
			return nil, apperror.NewInvalidInput("cannot move order to processing without confirmed payment")
		}

		err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			return u.orderRepo.UpdateStatus(ctx, exec,
				ord.ID,
				input.Status,
			)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}

		result := UpdateOrderStatusResult{Order: *ord}
		return &result, nil

	case order.OrderStatusDelivered:
		shipments, err := u.shipmentRepo.ListByOrderID(ctx, u.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list shipments: %w", err)
		}

		err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			for i := range shipments {
				s := shipments[i]
				if err := s.UpdateStatus(shipment.ShipmentStatusDelivered); err != nil {
					return fmt.Errorf("failed to update shipment status: %w", err)
				}
				if err := u.shipmentRepo.Update(ctx, exec, s); err != nil {
					return fmt.Errorf("failed to persist shipment: %w", err)
				}
			}
			return u.orderRepo.UpdateStatus(ctx, exec, ord.ID, input.Status)
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

	case order.OrderStatusCancelled:
		items, err := u.orderItemRepo.ListByOrderID(ctx, u.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list order items: %w", err)
		}

		err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
			for _, item := range items {
				if err := u.inventoryRepo.Release(ctx, exec, item.ProductID, item.ShopID, item.Quantity); err != nil {
					return fmt.Errorf("failed to release inventory for product %s: %w", item.ProductID, err)
				}
			}
			return u.orderRepo.UpdateStatus(ctx, exec, ord.ID, input.Status)
		})
		if err != nil {
			return nil, err
		}

		result := UpdateOrderStatusResult{Order: *ord}
		return &result, nil
	}

	// processing â†’ shipped
	items, err := u.orderItemRepo.ListByOrderID(ctx, u.executor, ord.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list order items: %w", err)
	}
	if len(items) == 0 {
		return nil, apperror.NewInvalidInput("order has no items")
	}

	itemMap := make(map[uuid.UUID]order.OrderItem, len(items))
	for _, item := range items {
		itemMap[item.ID] = item
	}

	// Resolve customer destination district ID from the order's address
	customerAddr, err := u.addressRepo.GetByID(ctx, u.executor, ord.AddressID)
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
	products, err := u.productRepo.FindByIDs(ctx, u.executor, productIDs)
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
			_ = u.logistics.CancelOrder(ctx, orderNo)
		}
	}

	if len(input.Shipments) > 0 {
		// Staff explicitly configured shipment grouping (split / multi shipment)
		for idx, sInput := range input.Shipments {
			if len(sInput.ItemIDs) == 0 {
				return nil, apperror.NewInvalidInput("each shipment must contain at least one order item")
			}

			var shipmentItems []order.OrderItem
			for _, itemID := range sInput.ItemIDs {
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

			shopAddr, err := u.shopAddressRepo.GetDefaultByShopID(ctx, u.executor, first.ShopID)
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

				komerceResult, err := u.logistics.CreateOrder(ctx, orderInput)
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
	} else {
		// Fallback / legacy: Group order items by ShopID for per-shop shipment processing
		type shopGroup struct {
			shopID uuid.UUID
			items  []order.OrderItem
		}
		var groups []shopGroup
		groupMap := make(map[uuid.UUID]int)
		for _, item := range items {
			idx, exists := groupMap[item.ShopID]
			if !exists {
				groupMap[item.ShopID] = len(groups)
				groups = append(groups, shopGroup{
					shopID: item.ShopID,
					items:  []order.OrderItem{item},
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

			shopAddr, err := u.shopAddressRepo.GetDefaultByShopID(ctx, u.executor, group.shopID)
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

				komerceResult, err := u.logistics.CreateOrder(ctx, orderInput)
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
	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		for _, ps := range preparedShipments {
			if err := u.shipmentRepo.Create(ctx, exec, ps.shipment); err != nil {
				return fmt.Errorf("failed to persist shipment: %w", err)
			}
			if err := u.orderItemRepo.AssignShipment(ctx, exec, ps.shipment.ID, ps.itemIDs); err != nil {
				return fmt.Errorf("failed to link items to shipment: %w", err)
			}
			createdShipments = append(createdShipments, ps.shipment)
		}
		if err := u.orderRepo.UpdateStatus(ctx, exec, ord.ID, input.Status); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
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
}

// derefPhone safely dereferences a nullable phone pointer, returning an
// empty string when nil. Komerce accepts empty phone numbers gracefully.
func derefPhone(phone *string) string {
	if phone == nil {
		return ""
	}
	return *phone
}
