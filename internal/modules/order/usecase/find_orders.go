package usecase

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	transaction "komecore/internal/infra/transactor"
	addressDomain "komecore/internal/modules/address/domain"
	addressRepo "komecore/internal/modules/address/repository"
	"komecore/internal/modules/order/domain"
	"komecore/internal/modules/order/repository"
	paymentDomain "komecore/internal/modules/payment/domain"
	paymentRepo "komecore/internal/modules/payment/repository"
	shipmentDomain "komecore/internal/modules/shipment/domain"
	shipmentRepo "komecore/internal/modules/shipment/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

type FindOrdersUsecase struct {
	executor               transaction.Executor
	orderRepo              repository.OrderRepository
	orderItemRepo          repository.OrderItemRepository
	paymentRepo            paymentRepo.PaymentRepository
	paymentChannelDataRepo paymentRepo.PaymentChannelDataRepository
	shipmentRepo           shipmentRepo.ShipmentRepository
	addressRepo            addressRepo.CustomerAddressRepository
}

func NewFindOrdersUsecase(
	executor transaction.Executor,
	orderRepo repository.OrderRepository,
	orderItemRepo repository.OrderItemRepository,
	paymentRepo paymentRepo.PaymentRepository,
	paymentChannelDataRepo paymentRepo.PaymentChannelDataRepository,
	shipmentRepo shipmentRepo.ShipmentRepository,
	addressRepo addressRepo.CustomerAddressRepository,
) *FindOrdersUsecase {
	return &FindOrdersUsecase{
		executor:               executor,
		orderRepo:              orderRepo,
		orderItemRepo:          orderItemRepo,
		paymentRepo:            paymentRepo,
		paymentChannelDataRepo: paymentChannelDataRepo,
		shipmentRepo:           shipmentRepo,
		addressRepo:            addressRepo,
	}
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
	Order       domain.Order
	Items       []domain.OrderItem
	Payment     *paymentDomain.Payment
	ChannelData *paymentDomain.PaymentChannelData
	Shipment    *shipmentDomain.Shipment
	Shipments   []shipmentDomain.Shipment
	Address     *addressDomain.CustomerAddress
}

func (u *FindOrdersUsecase) Execute(
	ctx context.Context,
	input FindOrdersInput,
) ([]OrderSearchResult, int, error) {
	var orderSortKeys = map[string]query.SortKey{
		"latest":   repository.OrderSortLatest,
		"date":     repository.OrderSortLatest,
		"number":   repository.OrderSortNumber,
		"total":    repository.OrderSortTotal,
		"status":   repository.OrderSortStatus,
		"modified": repository.OrderSortModify,
	}

	var sorts query.Sorts
	if input.Sort != "" {
		parts := strings.SplitSeq(input.Sort, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			subparts := strings.Split(part, ":")
			key := strings.TrimSpace(subparts[0])

			var dir query.SortDirection = query.SortDesc
			if len(subparts) > 1 {
				d := strings.ToLower(strings.TrimSpace(subparts[1]))
				if d == "asc" {
					dir = query.SortAsc
				}
			}

			sortKey, exists := orderSortKeys[key]
			if exists {
				sorts = append(sorts, query.Sort{
					By:        sortKey,
					Direction: dir,
				})
			}
		}
	}

	if len(sorts) == 0 {
		sorts = query.Sorts{
			{
				By:        repository.OrderSortLatest,
				Direction: query.SortDesc,
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

	params := repository.FindOrderParams{
		ID:         input.ID,
		Number:     input.Number,
		CustomerID: input.CustomerID,
		ShopID:     input.ShopID,
		ShopIDs:    input.ShopIDs,
		Status:     statusParam,
		Statuses:   statuses,
		FromDate:   input.FromDate,
		ToDate:     input.ToDate,
		Pagination: query.Pagination{
			Page:  input.Page,
			Limit: input.Limit,
		},
		Sorts: sorts,
	}

	orders, total, err := u.orderRepo.FindOrders(ctx, u.executor, params)
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

	orderItems, err := u.orderItemRepo.ListByOrderIDs(ctx, u.executor, orderIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list order items: %w", err)
	}

	payments, err := u.paymentRepo.ListByOrderIDs(ctx, u.executor, orderIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}

	shipments, err := u.shipmentRepo.ListByOrderIDs(ctx, u.executor, orderIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list shipments: %w", err)
	}

	itemsMap := make(map[uuid.UUID][]domain.OrderItem)
	for _, item := range orderItems {
		if input.ShopID != nil && item.ShopID != *input.ShopID {
			continue
		}
		if len(input.ShopIDs) > 0 && !slices.Contains(input.ShopIDs, item.ShopID) {
			continue
		}
		itemsMap[item.OrderID] = append(itemsMap[item.OrderID], item)
	}

	paymentsMap := make(map[uuid.UUID]*paymentDomain.Payment)
	for i := range payments {
		p := payments[i]
		paymentsMap[p.OrderID] = &p
	}

	// Collect payment IDs to bulk-fetch persisted channel data
	paymentIDs := make([]uuid.UUID, 0, len(payments))
	for i := range payments {
		paymentIDs = append(paymentIDs, payments[i].ID)
	}

	channelDataMap, err := u.paymentChannelDataRepo.ListByPaymentIDs(ctx, u.executor, paymentIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payment channel data: %w", err)
	}

	shipmentsMap := make(map[uuid.UUID][]shipmentDomain.Shipment)
	for i := range shipments {
		s := shipments[i]
		shipmentsMap[s.OrderID] = append(shipmentsMap[s.OrderID], s)
	}

	addressIDs := make([]uuid.UUID, 0, len(orders))
	addressSeen := make(map[uuid.UUID]bool)
	for _, o := range orders {
		if o.AddressID != uuid.Nil && !addressSeen[o.AddressID] {
			addressSeen[o.AddressID] = true
			addressIDs = append(addressIDs, o.AddressID)
		}
	}

	addresses, err := u.addressRepo.ListByIDs(ctx, u.executor, addressIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list customer addresses: %w", err)
	}

	addressesMap := make(map[uuid.UUID]*addressDomain.CustomerAddress)
	for i := range addresses {
		addr := addresses[i]
		addressesMap[addr.ID] = &addr
	}

	results := make([]OrderSearchResult, len(orders))
	for i, o := range orders {
		items := itemsMap[o.ID]
		if items == nil {
			items = []domain.OrderItem{}
		}

		var channelData *paymentDomain.PaymentChannelData
		payment := paymentsMap[o.ID]
		if payment != nil {
			channelData = channelDataMap[payment.ID]
		}

		orderShipments := shipmentsMap[o.ID]
		if orderShipments == nil {
			orderShipments = []shipmentDomain.Shipment{}
		}

		var filteredShipments []shipmentDomain.Shipment
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

		var firstShipment *shipmentDomain.Shipment
		if len(filteredShipments) > 0 {
			firstShipment = &filteredShipments[0]
		}

		results[i] = OrderSearchResult{
			Order:       o,
			Items:       items,
			Payment:     payment,
			ChannelData: channelData,
			Shipment:    firstShipment,
			Shipments:   filteredShipments,
			Address:     addressesMap[o.AddressID],
		}
	}

	return results, total, nil
}
