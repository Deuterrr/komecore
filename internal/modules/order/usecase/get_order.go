package usecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/domain"
	"komecore/internal/modules/order/repository"
	paymentDomain "komecore/internal/modules/payment/domain"
	paymentRepo "komecore/internal/modules/payment/repository"
	shipmentDomain "komecore/internal/modules/shipment/domain"
	shipmentRepo "komecore/internal/modules/shipment/repository"

	"github.com/google/uuid"
)

type GetOrderUsecase struct {
	executor               transaction.Executor
	orderRepo              repository.OrderRepository
	orderItemRepo          repository.OrderItemRepository
	paymentRepo            paymentRepo.PaymentRepository
	paymentChannelDataRepo paymentRepo.PaymentChannelDataRepository
	shipmentRepo           shipmentRepo.ShipmentRepository
}

func NewGetOrderUsecase(
	executor transaction.Executor,
	orderRepo repository.OrderRepository,
	orderItemRepo repository.OrderItemRepository,
	paymentRepo paymentRepo.PaymentRepository,
	paymentChannelDataRepo paymentRepo.PaymentChannelDataRepository,
	shipmentRepo shipmentRepo.ShipmentRepository,
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
	Order       domain.Order
	Items       []domain.OrderItem
	Payment     *paymentDomain.Payment
	ChannelData *paymentDomain.PaymentChannelData
	Shipment    *shipmentDomain.Shipment
	Shipments   []shipmentDomain.Shipment
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

	var channelData *paymentDomain.PaymentChannelData
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

	var firstShipment *shipmentDomain.Shipment
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
