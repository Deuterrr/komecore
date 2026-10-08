package orderusecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"
	"komecore/internal/modules/shipment/shipmentdomain"
	"komecore/internal/modules/shipment/shipmentrepo"

	"github.com/google/uuid"
)

type GetOrderUsecase struct {
	executor               transaction.Executor
	orderRepo              orderrepo.OrderRepository
	orderItemRepo          orderrepo.OrderItemRepository
	paymentRepo            paymentrepo.PaymentRepository
	paymentChannelDataRepo paymentrepo.PaymentChannelDataRepository
	shipmentRepo           shipmentrepo.ShipmentRepository
}

func NewGetOrderUsecase(
	executor transaction.Executor,
	orderRepo orderrepo.OrderRepository,
	orderItemRepo orderrepo.OrderItemRepository,
	paymentRepo paymentrepo.PaymentRepository,
	paymentChannelDataRepo paymentrepo.PaymentChannelDataRepository,
	shipmentRepo shipmentrepo.ShipmentRepository,
) *GetOrderUsecase {
	return &GetOrderUsecase{
		executor:               executor,
		orderRepo:              orderRepo,
		orderItemRepo:          orderItemRepo,
		paymentRepo:            paymentRepo,
		paymentChannelDataRepo: paymentChannelDataRepo,
		shipmentRepo:           shipmentRepo,
	}
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
	Order       orderdomain.Order
	Items       []orderdomain.OrderItem
	Payment     *paymentdomain.Payment
	ChannelData *paymentdomain.PaymentChannelData
	Shipment    *shipmentdomain.Shipment
	Shipments   []shipmentdomain.Shipment
}

func (u *GetOrderUsecase) Execute(ctx context.Context, input GetOrderInput) (*GetOrderResult, error) {
	order, err := u.orderRepo.GetByID(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return nil, nil
	}

	if input.CustomerID != nil && order.CustomerID != *input.CustomerID {
		return nil, nil
	}

	items, err := u.orderItemRepo.ListByOrderID(ctx, u.executor, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list order items: %w", err)
	}

	payment, err := u.paymentRepo.GetByOrderID(ctx, u.executor, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	var channelData *paymentdomain.PaymentChannelData
	if payment != nil {
		cd, err := u.paymentChannelDataRepo.GetByPaymentID(ctx, u.executor, payment.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get payment channel data: %w", err)
		}
		channelData = cd
	}

	shipments, err := u.shipmentRepo.ListByOrderID(ctx, u.executor, order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list shipments: %w", err)
	}

	var firstShipment *shipmentdomain.Shipment
	if len(shipments) > 0 {
		firstShipment = &shipments[0]
	}

	return &GetOrderResult{
		Order:       *order,
		Items:       items,
		Payment:     payment,
		ChannelData: channelData,
		Shipment:    firstShipment,
		Shipments:   shipments,
	}, nil
}
