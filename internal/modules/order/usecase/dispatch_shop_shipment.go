package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	shipping "komecore/internal/infra/shipping"
	transaction "komecore/internal/infra/transactor"
	addressRepo "komecore/internal/modules/address/repository"
	"komecore/internal/modules/order/domain"
	"komecore/internal/modules/order/repository"
	productDomain "komecore/internal/modules/product/domain"
	productRepo "komecore/internal/modules/product/repository"
	shipmentDomain "komecore/internal/modules/shipment/domain"
	shipmentRepo "komecore/internal/modules/shipment/repository"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type DispatchShopShipmentUsecase struct {
	executor        transaction.Executor
	transactor      transaction.Transactor
	orderRepo       repository.OrderRepository
	orderItemRepo   repository.OrderItemRepository
	productRepo     productRepo.ProductRepository
	shipmentRepo    shipmentRepo.ShipmentRepository
	addressRepo     addressRepo.CustomerAddressRepository
	shopAddressRepo addressRepo.ShopAddressRepository
	logistics       shipping.LogisticsProvider
	auditLogger     applogger.AuditLogger
}

func NewDispatchShopShipmentUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	orderRepo repository.OrderRepository,
	orderItemRepo repository.OrderItemRepository,
	productRepo productRepo.ProductRepository,
	shipmentRepo shipmentRepo.ShipmentRepository,
	addressRepo addressRepo.CustomerAddressRepository,
	shopAddressRepo addressRepo.ShopAddressRepository,
	logistics shipping.LogisticsProvider,
	auditLogger applogger.AuditLogger,
) *DispatchShopShipmentUsecase {
	return &DispatchShopShipmentUsecase{
		executor:        executor,
		transactor:      transactor,
		orderRepo:       orderRepo,
		orderItemRepo:   orderItemRepo,
		productRepo:     productRepo,
		shipmentRepo:    shipmentRepo,
		addressRepo:     addressRepo,
		shopAddressRepo: shopAddressRepo,
		logistics:       logistics,
		auditLogger:     auditLogger,
	}
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
	Order           domain.Order
	Shipment        shipmentDomain.Shipment
	AllItemsShipped bool
}

func (u *DispatchShopShipmentUsecase) Execute(
	ctx context.Context,
	input DispatchShopShipmentInput,
) (res *DispatchShopShipmentResult, err error) {
	defer func() {
		if err != nil {
			u.auditLogger.Log(ctx, applogger.AuditEvent{
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

	order, err := u.orderRepo.GetByID(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return nil, apperrors.NewNotFound("order not found")
	}
	if order.Status != domain.OrderStatusProcessing {
		return nil, apperrors.NewConflict(
			fmt.Sprintf("cannot dispatch shipment for order in '%s' status", order.Status),
		)
	}

	allItems, err := u.orderItemRepo.ListByOrderID(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	if len(input.ItemIDs) == 0 {
		return nil, apperrors.NewInvalidInput("item_ids cannot be empty")
	}

	itemMap := make(map[uuid.UUID]domain.OrderItem)
	for _, item := range allItems {
		itemMap[item.ID] = item
	}

	var shipmentItems []domain.OrderItem
	for _, itemID := range input.ItemIDs {
		item, exists := itemMap[itemID]
		if !exists {
			return nil, apperrors.NewNotFound(
				fmt.Sprintf("item %s not found in order", itemID),
			)
		}
		if item.ShopID != input.ShopID {
			return nil, apperrors.NewConflict(
				fmt.Sprintf("item %s does not belong to shop %s", itemID, input.ShopID),
			)
		}
		if item.ShipmentID != nil {
			return nil, apperrors.NewConflict(
				fmt.Sprintf("item %s has already been shipped", itemID),
			)
		}
		shipmentItems = append(shipmentItems, item)
	}

	first := shipmentItems[0]
	method := shipmentDomain.FulfillmentMethodCourier
	if input.FulfillmentMethod != "" {
		method = shipmentDomain.FulfillmentMethod(input.FulfillmentMethod)
	}

	courierCode := input.Courier
	if courierCode == "" && first.CourierCode != nil {
		courierCode = *first.CourierCode
	}
	courierService := input.Service
	if courierService == "" && first.CourierService != nil {
		courierService = *first.CourierService
	}

	customerAddr, err := u.addressRepo.GetByID(ctx, u.executor, order.AddressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer address: %w", err)
	}
	if customerAddr == nil {
		return nil, apperrors.NewNotFound("customer address not found")
	}

	shopAddr, err := u.shopAddressRepo.GetDefaultByShopID(ctx, u.executor, input.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop address: %w", err)
	}
	if shopAddr == nil {
		return nil, apperrors.NewNotFound("shop address not found")
	}

	var productIDs []uuid.UUID
	for _, item := range shipmentItems {
		productIDs = append(productIDs, item.ProductID)
	}
	products, err := u.productRepo.FindByIDs(ctx, u.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load products: %w", err)
	}
	productMap := make(map[uuid.UUID]productDomain.Product, len(products))
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
			_ = u.logistics.CancelOrder(ctx, createdLogisticsOrderNo)
		}
	}

	trackingNumber := input.TrackingNumber
	if method == shipmentDomain.FulfillmentMethodCourier && (trackingNumber == nil || *trackingNumber == "") {
		itemName := first.ProductName
		if len(shipmentItems) > 1 {
			itemName = fmt.Sprintf("%s (+%d more)", first.ProductName, len(shipmentItems)-1)
		}

		uniqueOrderID := fmt.Sprintf("%s-SH-%s", order.Number, input.ShopID.String()[:8])

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

		komerceResult, err := u.logistics.CreateOrder(ctx, orderInput)
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
	shipment := shipmentDomain.Shipment{
		ID:                uuid.New(),
		OrderID:           order.ID,
		Status:            shipmentDomain.ShipmentStatusCreated,
		FulfillmentMethod: method,
		TrackingNumber:    trackingNumber,
		Courier:           courierCode,
		Service:           courierService,
		Cost:              shipmentCost,
		Weight:            totalWeightGrams,
		CreatedAt:         now,
	}

	if err := shipment.Validate(); err != nil {
		rollbackLogistics()
		return nil, apperrors.NewInvalidInput(err.Error())
	}

	unshippedCount := 0
	for _, item := range allItems {
		if item.ShipmentID == nil {
			unshippedCount++
		}
	}
	allItemsShipped := unshippedCount <= len(input.ItemIDs)

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.shipmentRepo.Create(ctx, exec, shipment); err != nil {
			return fmt.Errorf("failed to persist shipment: %w", err)
		}
		if err := u.orderItemRepo.AssignShipment(ctx, exec, shipment.ID, input.ItemIDs); err != nil {
			return fmt.Errorf("failed to link items to shipment: %w", err)
		}
		if allItemsShipped {
			if err := u.orderRepo.UpdateStatus(ctx, exec, order.ID, domain.OrderStatusShipped); err != nil {
				return fmt.Errorf("failed to update order status to shipped: %w", err)
			}
			order.Status = domain.OrderStatusShipped
		}
		return nil
	})
	if err != nil {
		rollbackLogistics()
		return nil, err
	}

	return &DispatchShopShipmentResult{
		Order:           *order,
		Shipment:        shipment,
		AllItemsShipped: allItemsShipped,
	}, nil
}
