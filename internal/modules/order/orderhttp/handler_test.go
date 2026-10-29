package orderhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order"
	"komecore/internal/modules/payment"
	"komecore/internal/modules/shipment"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Mocks for handler tests
type hMockExecutor struct {
	transaction.NoopExecutor
}

type hMockOrderRepo struct {
	order *order.Order
}

func (m *hMockOrderRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*order.Order, error) {
	if m.order != nil && m.order.ID == id {
		return m.order, nil
	}
	return nil, nil
}
func (m *hMockOrderRepo) GetByIDForUpdate(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*order.Order, error) {
	return m.GetByID(ctx, exec, id)
}
func (m *hMockOrderRepo) GetByNumber(_ context.Context, _ transaction.Executor, _ string) (*order.Order, error) {
	return nil, nil
}
func (m *hMockOrderRepo) UpdateStatus(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ order.OrderStatus) error {
	return nil
}
func (m *hMockOrderRepo) UpdateStatusWithSLA(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ order.OrderStatus, _ *time.Time, _ *time.Time) error {
	return nil
}
func (m *hMockOrderRepo) Save(_ context.Context, _ transaction.Executor, _ order.Order) error {
	return nil
}
func (m *hMockOrderRepo) FindOrders(_ context.Context, _ transaction.Executor, _ order.FindOrderParams) ([]order.Order, int, error) {
	return nil, 0, nil
}
func (m *hMockOrderRepo) SetConfirmedAndExpiry(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time, _ time.Time) error {
	return nil
}
func (m *hMockOrderRepo) FindExpiredUnfulfilledOrders(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]order.Order, error) {
	return nil, nil
}

type hMockOrderItemRepo struct {
	items []order.OrderItem
}

func (m *hMockOrderItemRepo) ListByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) ([]order.OrderItem, error) {
	return m.items, nil
}
func (m *hMockOrderItemRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]order.OrderItem, error) {
	return m.items, nil
}
func (m *hMockOrderItemRepo) ListByShipmentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]order.OrderItem, error) {
	return nil, nil
}
func (m *hMockOrderItemRepo) SaveBulk(_ context.Context, _ transaction.Executor, _ []order.OrderItem) error {
	return nil
}
func (m *hMockOrderItemRepo) AssignShipment(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

type hMockPaymentRepo struct{}

func (m *hMockPaymentRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*payment.Payment, error) {
	return nil, nil
}
func (m *hMockPaymentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*payment.Payment, error) {
	return nil, nil
}
func (m *hMockPaymentRepo) GetByOrderIDForUpdate(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*payment.Payment, error) {
	return nil, nil
}
func (m *hMockPaymentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]payment.Payment, error) {
	return nil, nil
}
func (m *hMockPaymentRepo) ListPendingGateway(_ context.Context, _ transaction.Executor, _ time.Time) ([]payment.Payment, error) {
	return nil, nil
}
func (m *hMockPaymentRepo) ListPastDuePending(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]payment.Payment, error) {
	return nil, nil
}
func (m *hMockPaymentRepo) Save(_ context.Context, _ transaction.Executor, _ payment.Payment) error {
	return nil
}
func (m *hMockPaymentRepo) UpdateStatus(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ payment.PaymentStatus) error {
	return nil
}

type hMockPaymentChannelDataRepo struct{}

func (m *hMockPaymentChannelDataRepo) GetByPaymentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*payment.PaymentChannelData, error) {
	return nil, nil
}
func (m *hMockPaymentChannelDataRepo) ListByPaymentIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID]*payment.PaymentChannelData, error) {
	return nil, nil
}
func (m *hMockPaymentChannelDataRepo) Save(_ context.Context, _ transaction.Executor, _ payment.PaymentChannelData) error {
	return nil
}

type hMockShipmentRepo struct{}

func (m *hMockShipmentRepo) Create(_ context.Context, _ transaction.Executor, _ shipment.Shipment) error {
	return nil
}
func (m *hMockShipmentRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*shipment.Shipment, error) {
	return nil, nil
}
func (m *hMockShipmentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*shipment.Shipment, error) {
	return nil, nil
}
func (m *hMockShipmentRepo) ListByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]shipment.Shipment, error) {
	return nil, nil
}
func (m *hMockShipmentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]shipment.Shipment, error) {
	return nil, nil
}
func (m *hMockShipmentRepo) Save(_ context.Context, _ transaction.Executor, _ shipment.Shipment) error {
	return nil
}
func (m *hMockShipmentRepo) Update(_ context.Context, _ transaction.Executor, _ shipment.Shipment) error {
	return nil
}

// Tests

func TestResolveShopFilter_NilGetShopDependency_ReturnsError(t *testing.T) {
	handler := &Handler{
		shopService: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/orders?shop_slug=test-shop", nil)
	_, specified, err := handler.resolveShopFilter(req)

	if !specified {
		t.Errorf("expected specified=true when shop_slug query param is passed")
	}
	if err == nil {
		t.Fatalf("expected internal server error when getShop dependency is nil, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected HTTP status 500, got %v", err)
	}
}

func TestGetOrderTrackingForStaff_CustomerActor_ReturnsForbidden(t *testing.T) {
	customerID := uuid.New()
	orderID := uuid.New()

	ord := &order.Order{
		ID: orderID,
	}
	items := []order.OrderItem{
		{ID: uuid.New(), OrderID: orderID},
	}

	orderRepoMock := &hMockOrderRepo{order: ord}
	orderItemRepoMock := &hMockOrderItemRepo{items: items}

	orderSvc := order.NewService(
		&hMockExecutor{},
		nil,
		orderRepoMock,
		orderItemRepoMock,
		nil,
		&hMockPaymentRepo{},
		&hMockPaymentChannelDataRepo{},
		nil,
		&hMockShipmentRepo{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	handler := &Handler{
		service: orderSvc,
	}

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/tracking", nil)
	actor := &authctx.Actor{
		Type:       authctx.AccountTypeCustomer,
		CustomerID: &customerID,
	}
	ctx := authctx.WithActor(req.Context(), actor)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	err := handler.GetOrderTrackingForStaff(w, req)

	if err == nil {
		t.Fatalf("expected forbidden error for non-staff, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden, got %v", err)
	}
}

func TestUpdateOrderStatus_CustomerActor_ReturnsForbidden(t *testing.T) {
	customerID := uuid.New()
	orderID := uuid.New()

	handler := &Handler{}

	actor := &authctx.Actor{
		Type:       authctx.AccountTypeCustomer,
		CustomerID: &customerID,
	}

	body, _ := json.Marshal(map[string]any{
		"status": "shipped",
	})
	req := httptest.NewRequest(http.MethodPatch, "/orders/"+orderID.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := authctx.WithActor(req.Context(), actor)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	err := handler.UpdateOrderStatus(w, req)

	if err == nil {
		t.Fatalf("expected forbidden error for non-staff, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden, got %v", err)
	}
}

func TestUpdateOrderStatus_MultiShop_PartialPermission_Forbidden(t *testing.T) {
	staffID := uuid.New()
	orderID := uuid.New()
	shop1 := uuid.New()
	shop2 := uuid.New()

	ord := &order.Order{
		ID:     orderID,
		Status: order.OrderStatusProcessing,
	}
	items := []order.OrderItem{
		{ID: uuid.New(), OrderID: orderID, ShopID: shop1},
		{ID: uuid.New(), OrderID: orderID, ShopID: shop2},
	}

	orderRepoMock := &hMockOrderRepo{order: ord}
	orderItemRepoMock := &hMockOrderItemRepo{items: items}

	orderSvc := order.NewService(
		&hMockExecutor{},
		nil,
		orderRepoMock,
		orderItemRepoMock,
		nil,
		&hMockPaymentRepo{},
		&hMockPaymentChannelDataRepo{},
		nil,
		&hMockShipmentRepo{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	handler := &Handler{
		service: orderSvc,
	}

	actor := &authctx.Actor{
		Type:    authctx.AccountTypeStaff,
		StaffID: &staffID,
		ShopPermissions: map[uuid.UUID][]string{
			shop1: {authctx.PermissionOrderUpdateStatus},
			// Missing permission for shop2!
		},
	}

	body, _ := json.Marshal(map[string]any{
		"status": "cancelled",
	})
	req := httptest.NewRequest(http.MethodPatch, "/orders/"+orderID.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := authctx.WithActor(req.Context(), actor)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	err := handler.UpdateOrderStatus(w, req)

	if err == nil {
		t.Fatalf("expected forbidden error for staff missing permission on shop2, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden, got %v", err)
	}
}
