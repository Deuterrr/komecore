package order

import (
	"context"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/shipment"

	"github.com/google/uuid"
)

type foMockExecutor struct {
	transaction.NoopExecutor
}

type foMockOrderRepo struct {
	capturedParams FindOrderParams
	orders         []Order
	total          int
	err            error
}

func (m *foMockOrderRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*Order, error) {
	return nil, nil
}
func (m *foMockOrderRepo) GetByNumber(_ context.Context, _ transaction.Executor, _ string) (*Order, error) {
	return nil, nil
}
func (m *foMockOrderRepo) UpdateStatus(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ OrderStatus) error {
	return nil
}
func (m *foMockOrderRepo) UpdateStatusWithSLA(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ OrderStatus, _ *time.Time, _ *time.Time) error {
	return nil
}
func (m *foMockOrderRepo) Save(_ context.Context, _ transaction.Executor, _ Order) error {
	return nil
}
func (m *foMockOrderRepo) FindOrders(_ context.Context, _ transaction.Executor, params FindOrderParams) ([]Order, int, error) {
	m.capturedParams = params
	return m.orders, m.total, m.err
}
func (m *foMockOrderRepo) SetConfirmedAndExpiry(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time, _ time.Time) error {
	return nil
}
func (m *foMockOrderRepo) FindExpiredUnfulfilledOrders(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]Order, error) {
	return nil, nil
}

type foMockOrderItemRepo struct{}

func (m *foMockOrderItemRepo) ListByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]OrderItem, error) {
	return nil, nil
}
func (m *foMockOrderItemRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]OrderItem, error) {
	return []OrderItem{}, nil
}
func (m *foMockOrderItemRepo) ListByShipmentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]OrderItem, error) {
	return nil, nil
}
func (m *foMockOrderItemRepo) SaveBulk(_ context.Context, _ transaction.Executor, _ []OrderItem) error {
	return nil
}
func (m *foMockOrderItemRepo) AssignShipment(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

type foMockPaymentRepo struct{}

func (m *foMockPaymentRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*payment.Payment, error) {
	return nil, nil
}
func (m *foMockPaymentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*payment.Payment, error) {
	return nil, nil
}
func (m *foMockPaymentRepo) GetByOrderIDForUpdate(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*payment.Payment, error) {
	return nil, nil
}
func (m *foMockPaymentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]payment.Payment, error) {
	return []payment.Payment{}, nil
}
func (m *foMockPaymentRepo) ListPendingGateway(_ context.Context, _ transaction.Executor, _ time.Time) ([]payment.Payment, error) {
	return nil, nil
}
func (m *foMockPaymentRepo) ListPastDuePending(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]payment.Payment, error) {
	return nil, nil
}
func (m *foMockPaymentRepo) Save(_ context.Context, _ transaction.Executor, _ payment.Payment) error {
	return nil
}
func (m *foMockPaymentRepo) UpdateStatus(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ payment.PaymentStatus) error {
	return nil
}

type foMockPaymentChannelDataRepo struct{}

func (m *foMockPaymentChannelDataRepo) GetByPaymentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*payment.PaymentChannelData, error) {
	return nil, nil
}
func (m *foMockPaymentChannelDataRepo) ListByPaymentIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID]*payment.PaymentChannelData, error) {
	return map[uuid.UUID]*payment.PaymentChannelData{}, nil
}
func (m *foMockPaymentChannelDataRepo) Save(_ context.Context, _ transaction.Executor, _ payment.PaymentChannelData) error {
	return nil
}

type foMockShipmentRepo struct{}

func (m *foMockShipmentRepo) Create(_ context.Context, _ transaction.Executor, _ shipment.Shipment) error {
	return nil
}
func (m *foMockShipmentRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*shipment.Shipment, error) {
	return nil, nil
}
func (m *foMockShipmentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*shipment.Shipment, error) {
	return nil, nil
}
func (m *foMockShipmentRepo) ListByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]shipment.Shipment, error) {
	return []shipment.Shipment{}, nil
}
func (m *foMockShipmentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]shipment.Shipment, error) {
	return []shipment.Shipment{}, nil
}
func (m *foMockShipmentRepo) Save(_ context.Context, _ transaction.Executor, _ shipment.Shipment) error {
	return nil
}
func (m *foMockShipmentRepo) Update(_ context.Context, _ transaction.Executor, _ shipment.Shipment) error {
	return nil
}

type foMockAddressRepo struct{}

func (m *foMockAddressRepo) CountByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*int, error) {
	zero := 0
	return &zero, nil
}
func (m *foMockAddressRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *foMockAddressRepo) DeleteByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}
func (m *foMockAddressRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*address.CustomerAddress, error) {
	return nil, nil
}
func (m *foMockAddressRepo) GetDefaultByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*address.CustomerAddress, error) {
	return nil, nil
}
func (m *foMockAddressRepo) ListByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]address.CustomerAddress, error) {
	return nil, nil
}
func (m *foMockAddressRepo) ListByIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]address.CustomerAddress, error) {
	return []address.CustomerAddress{}, nil
}
func (m *foMockAddressRepo) Save(_ context.Context, _ transaction.Executor, _ address.CustomerAddress) error {
	return nil
}
func (m *foMockAddressRepo) UnsetDefaultByCustomerID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

func TestFindOrdersUsecase_ShopFilter(t *testing.T) {
	mockOrderRepo := &foMockOrderRepo{}
	svc := NewService(
		&foMockExecutor{},
		nil,
		mockOrderRepo,
		&foMockOrderItemRepo{},
		nil,
		&foMockPaymentRepo{},
		&foMockPaymentChannelDataRepo{},
		nil,
		&foMockShipmentRepo{},
		&foMockAddressRepo{},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	shopID := uuid.New()
	shopIDs := []uuid.UUID{uuid.New(), uuid.New()}

	input := FindOrdersInput{
		Page:    1,
		Limit:   10,
		ShopID:  &shopID,
		ShopIDs: shopIDs,
	}

	results, total, err := svc.FindOrders(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 0 || len(results) != 0 {
		t.Errorf("expected empty results, got total=%d len=%d", total, len(results))
	}

	if mockOrderRepo.capturedParams.ShopID == nil || *mockOrderRepo.capturedParams.ShopID != shopID {
		t.Errorf("expected captured ShopID to be %v, got %v", shopID, mockOrderRepo.capturedParams.ShopID)
	}

	if len(mockOrderRepo.capturedParams.ShopIDs) != 2 {
		t.Errorf("expected 2 ShopIDs, got %d", len(mockOrderRepo.capturedParams.ShopIDs))
	}
}

func TestFindOrdersUsecase_StatusFilter_ClearsSingleStatusWhenStatusesProvided(t *testing.T) {
	mockOrderRepo := &foMockOrderRepo{}
	svc := NewService(
		&foMockExecutor{},
		nil,
		mockOrderRepo,
		&foMockOrderItemRepo{},
		nil,
		&foMockPaymentRepo{},
		&foMockPaymentChannelDataRepo{},
		nil,
		&foMockShipmentRepo{},
		&foMockAddressRepo{},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	statusStr := "pending"
	statuses := []string{"pending", "confirmed"}
	input := FindOrdersInput{
		Page:     1,
		Limit:    10,
		Status:   &statusStr,
		Statuses: statuses,
	}

	_, _, err := svc.FindOrders(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockOrderRepo.capturedParams.Status != nil {
		t.Errorf("expected Status to be nil when Statuses slice is provided, got %v", *mockOrderRepo.capturedParams.Status)
	}
	if len(mockOrderRepo.capturedParams.Statuses) != 2 {
		t.Errorf("expected Statuses slice length 2, got %d", len(mockOrderRepo.capturedParams.Statuses))
	}
}
