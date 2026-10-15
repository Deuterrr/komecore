package orderusecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/shipment"

	"github.com/google/uuid"
)

type GetOrderUsecase struct {
	executor               transaction.Executor
	orderRepo              order.Repository
	orderItemRepo          order.ItemRepository
	paymentRepo            payment.Repository
	paymentChannelDataRepo payment.ChannelDataRepository
	shipmentRepo           shipment.Repository
}

func NewGetOrderUsecase(
	executor transaction.Executor,
	orderRepo order.Repository,
	orderItemRepo order.ItemRepository,
	paymentRepo payment.Repository,
	paymentChannelDataRepo payment.ChannelDataRepository,
	shipmentRepo shipment.Repository,
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
	Order       order.Order
	Items       []order.OrderItem
	Payment     *payment.Payment
	ChannelData *payment.PaymentChannelData
	Shipment    *shipment.Shipment
	Shipments   []shipment.Shipment
}

func (u *GetOrderUsecase) Execute(ctx context.Context, input GetOrderInput) (*GetOrderResult, error) {
	ord, err := u.orderRepo.GetByID(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if ord == nil {
		return nil, nil
	}

	if input.CustomerID != nil && ord.CustomerID != *input.CustomerID {
		return nil, nil
	}

	items, err := u.orderItemRepo.ListByOrderID(ctx, u.executor, ord.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list order items: %w", err)
	}

	pmt, err := u.paymentRepo.GetByOrderID(ctx, u.executor, ord.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	var channelData *payment.PaymentChannelData
	if pmt != nil {
		cd, err := u.paymentChannelDataRepo.GetByPaymentID(ctx, u.executor, pmt.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get payment channel data: %w", err)
		}
		channelData = cd
	}

	shipments, err := u.shipmentRepo.ListByOrderID(ctx, u.executor, ord.ID)
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
