package order

import (
	"context"
	"testing"

	shipping "komecore/internal/infra/shipping"
	"komecore/internal/modules/address"
	"komecore/internal/modules/shipment"

	"github.com/google/uuid"
)

func TestDispatchShopShipmentUsecase_PartialAndFull(t *testing.T) {
	ctx := context.Background()

	shopA := uuid.New()
	shopB := uuid.New()
	orderID := uuid.New()
	customerAddrID := uuid.New()

	itemA := OrderItem{
		ID:          uuid.New(),
		OrderID:     orderID,
		ShopID:      shopA,
		ProductName: "Cotton T-Shirt",
		Quantity:    2,
		ShippingFee: 15000,
	}

	itemB := OrderItem{
		ID:          uuid.New(),
		OrderID:     orderID,
		ShopID:      shopB,
		ProductName: "Denim Jeans",
		Quantity:    1,
		ShippingFee: 20000,
	}

	customerAddr := &address.CustomerAddress{
		ID: customerAddrID,
		Detail: address.AddressDetail{
			District:    "1001",
			FullAddress: "Jl. Customer 1",
		},
		ReceiverName: "Customer John",
	}

	shopAddressA := &address.ShopAddress{
		ID:     uuid.New(),
		ShopID: shopA,
		Detail: address.AddressDetail{
			District:    "2001",
			FullAddress: "Jl. Shop Address A",
		},
	}

	shopAddressB := &address.ShopAddress{
		ID:     uuid.New(),
		ShopID: shopB,
		Detail: address.AddressDetail{
			District:    "2002",
			FullAddress: "Jl. Shop Address B",
		},
	}

	ord := &Order{
		ID:        orderID,
		Number:    "ORD-TEST-001",
		Status:    OrderStatusProcessing,
		AddressID: customerAddrID,
	}

	orderRepoMock := &uosMockOrderRepo{order: ord}
	orderItemRepoMock := &uosMockOrderItemRepo{items: []OrderItem{itemA, itemB}}
	shipmentRepoMock := &uosMockShipmentRepo{}
	addrRepoMock := &uosMockAddressRepo{addr: customerAddr}
	shopAddrRepoMock := &uosMockShopAddressRepo{
		addrs: map[uuid.UUID]*address.ShopAddress{
			shopA: shopAddressA,
			shopB: shopAddressB,
		},
	}
	productRepoMock := &uosMockProductRepo{}
	logisticsMock := &uosMockLogisticsProvider{
		result: &shipping.CreateOrderResult{
			TrackingNumber: "AUTO-TRACK-123",
			KomerceOrderNo: "KOMERCE-ORDER-1",
		},
	}
	auditLoggerMock := &uosMockAuditLogger{}

	svc := newUpdateStatusTestService(
		&uosMockExecutor{},
		&uosMockTransactor{},
		orderRepoMock,
		orderItemRepoMock,
		nil,
		nil,
		productRepoMock,
		shipmentRepoMock,
		addrRepoMock,
		shopAddrRepoMock,
		logisticsMock,
		auditLoggerMock,
	)

	// Step 1: Shop A dispatches item A
	trackNoA := "TRACK-A-999"
	resA, err := svc.DispatchShopShipment(ctx, DispatchShopShipmentInput{
		OrderID:           orderID,
		ShopID:            shopA,
		FulfillmentMethod: "courier",
		Courier:           "JNE",
		Service:           "REG",
		TrackingNumber:    &trackNoA,
		ItemIDs:           []uuid.UUID{itemA.ID},
	})

	if err != nil {
		t.Fatalf("Shop A dispatch failed: %v", err)
	}

	if resA.AllItemsShipped {
		t.Errorf("expected AllItemsShipped to be false when Shop B has not dispatched, got true")
	}

	if resA.Order.Status != OrderStatusProcessing {
		t.Errorf("expected order status to remain processing, got %s", resA.Order.Status)
	}

	if len(shipmentRepoMock.createdShipments) != 1 {
		t.Fatalf("expected 1 shipment created, got %d", len(shipmentRepoMock.createdShipments))
	}

	// Update itemA to simulate having a shipment assigned
	itemA.ShipmentID = &resA.Shipment.ID
	orderItemRepoMock.items = []OrderItem{itemA, itemB}

	// Step 2: Shop B dispatches item B
	trackNoB := "TRACK-B-888"
	resB, err := svc.DispatchShopShipment(ctx, DispatchShopShipmentInput{
		OrderID:           orderID,
		ShopID:            shopB,
		FulfillmentMethod: "courier",
		Courier:           "SICEPAT",
		Service:           "BEST",
		TrackingNumber:    &trackNoB,
		ItemIDs:           []uuid.UUID{itemB.ID},
	})

	if err != nil {
		t.Fatalf("Shop B dispatch failed: %v", err)
	}

	if !resB.AllItemsShipped {
		t.Errorf("expected AllItemsShipped to be true after all items dispatched, got false")
	}

	if resB.Order.Status != OrderStatusShipped {
		t.Errorf("expected order status to auto-advance to shipped, got %s", resB.Order.Status)
	}

	if orderRepoMock.updatedState.status != OrderStatusShipped {
		t.Errorf("expected order status in repo to be updated to shipped, got %s", orderRepoMock.updatedState.status)
	}

	if len(shipmentRepoMock.createdShipments) != 2 {
		t.Fatalf("expected 2 shipments created in total, got %d", len(shipmentRepoMock.createdShipments))
	}
}

func TestDispatchShopShipmentUsecase_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	shopA := uuid.New()
	shopB := uuid.New()
	orderID := uuid.New()
	customerAddrID := uuid.New()

	alreadyShippedID := uuid.New()
	itemA := OrderItem{
		ID:          uuid.New(),
		OrderID:     orderID,
		ShopID:      shopA,
		ProductName: "Cotton T-Shirt",
		Quantity:    2,
		ShipmentID:  &alreadyShippedID,
	}

	itemB := OrderItem{
		ID:          uuid.New(),
		OrderID:     orderID,
		ShopID:      shopB,
		ProductName: "Denim Jeans",
		Quantity:    1,
	}

	ord := &Order{
		ID:        orderID,
		Number:    "ORD-TEST-002",
		Status:    OrderStatusProcessing,
		AddressID: customerAddrID,
	}

	orderRepoMock := &uosMockOrderRepo{order: ord}
	orderItemRepoMock := &uosMockOrderItemRepo{items: []OrderItem{itemA, itemB}}
	shipmentRepoMock := &uosMockShipmentRepo{}
	addrRepoMock := &uosMockAddressRepo{addr: &address.CustomerAddress{
		ID: customerAddrID,
		Detail: address.AddressDetail{
			District:    "1001",
			FullAddress: "Jl. Customer",
		},
		ReceiverName: "Customer",
	}}
	shopAddrRepoMock := &uosMockShopAddressRepo{
		addrs: map[uuid.UUID]*address.ShopAddress{
			shopA: {
				ID:     uuid.New(),
				ShopID: shopA,
				Detail: address.AddressDetail{
					District:    "2001",
					FullAddress: "Jl. Shop",
				},
			},
		},
	}

	svc := newUpdateStatusTestService(
		&uosMockExecutor{},
		&uosMockTransactor{},
		orderRepoMock,
		orderItemRepoMock,
		nil,
		nil,
		&uosMockProductRepo{},
		shipmentRepoMock,
		addrRepoMock,
		shopAddrRepoMock,
		&uosMockLogisticsProvider{},
		&uosMockAuditLogger{},
	)

	// Test 1: Empty item IDs
	_, err := svc.DispatchShopShipment(ctx, DispatchShopShipmentInput{
		OrderID: orderID,
		ShopID:  shopA,
		ItemIDs: []uuid.UUID{},
	})
	if err == nil {
		t.Errorf("expected error for empty item IDs, got nil")
	}

	// Test 2: Item already assigned to a shipment
	trackNo := "TRACK-1"
	_, err = svc.DispatchShopShipment(ctx, DispatchShopShipmentInput{
		OrderID:           orderID,
		ShopID:            shopA,
		FulfillmentMethod: string(shipment.FulfillmentMethodCourier),
		Courier:           "JNE",
		Service:           "REG",
		TrackingNumber:    &trackNo,
		ItemIDs:           []uuid.UUID{itemA.ID},
	})
	if err == nil {
		t.Errorf("expected error when item already has shipment, got nil")
	}

	// Test 3: Item belonging to another shop
	_, err = svc.DispatchShopShipment(ctx, DispatchShopShipmentInput{
		OrderID:           orderID,
		ShopID:            shopA,
		FulfillmentMethod: string(shipment.FulfillmentMethodCourier),
		Courier:           "JNE",
		Service:           "REG",
		TrackingNumber:    &trackNo,
		ItemIDs:           []uuid.UUID{itemB.ID}, // belongs to shopB
	})
	if err == nil {
		t.Errorf("expected error when dispatching item belonging to another shop, got nil")
	}
}
