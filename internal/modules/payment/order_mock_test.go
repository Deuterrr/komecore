package payment

import (
	"context"
	"errors"
	"sync"
	"time"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

type mockOrder struct {
	ID            uuid.UUID
	CustomerID    uuid.UUID
	Status        string
	Number        string
	CustomerEmail string
	CustomerName  string
}

type mockOrderPaymentManager struct {
	mu         sync.RWMutex
	orders     map[uuid.UUID]*mockOrder
	invoices   map[uuid.UUID]string
	items      map[uuid.UUID][]OrderItemInfo
	confirmErr error
	expireErr  error
	cancelErr  error
	getErr     error
	updateErr  error
}

func newMockOrderPaymentManager() *mockOrderPaymentManager {
	return &mockOrderPaymentManager{
		orders:   make(map[uuid.UUID]*mockOrder),
		invoices: make(map[uuid.UUID]string),
		items:    make(map[uuid.UUID][]OrderItemInfo),
	}
}

func (m *mockOrderPaymentManager) GetOrderForPayment(_ context.Context, _ transaction.Executor, orderID uuid.UUID) (*OrderInfo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orders[orderID]
	if !ok || o == nil {
		return nil, nil
	}
	return &OrderInfo{
		ID:            o.ID,
		CustomerID:    o.CustomerID,
		Number:        o.Number,
		Total:         100000,
		CustomerEmail: o.CustomerEmail,
		CustomerName:  o.CustomerName,
	}, nil
}

func (m *mockOrderPaymentManager) GetInvoiceNumber(_ context.Context, _ transaction.Executor, orderID uuid.UUID) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.invoices[orderID], nil
}

func (m *mockOrderPaymentManager) ConfirmOrderPayment(_ context.Context, _ transaction.Executor, orderID uuid.UUID, _ time.Time, _ time.Duration) ([]OrderItemInfo, error) {
	if m.confirmErr != nil {
		return nil, m.confirmErr
	}
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[orderID]
	if !ok || o == nil {
		return nil, errors.New("order not found for payment")
	}
	if o.Status != "pending" {
		return nil, errors.New("invalid status transition")
	}
	o.Status = "confirmed"
	return m.items[orderID], nil
}

func (m *mockOrderPaymentManager) ExpireOrderPayment(_ context.Context, _ transaction.Executor, orderID uuid.UUID) ([]OrderItemInfo, error) {
	if m.expireErr != nil {
		return nil, m.expireErr
	}
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[orderID]
	if !ok || o == nil {
		return nil, errors.New("order not found for payment")
	}
	if o.Status != "pending" {
		return nil, errors.New("invalid status transition")
	}
	o.Status = "expired"
	return m.items[orderID], nil
}

func (m *mockOrderPaymentManager) CancelOrderPayment(_ context.Context, _ transaction.Executor, orderID uuid.UUID) ([]OrderItemInfo, error) {
	if m.cancelErr != nil {
		return nil, m.cancelErr
	}
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[orderID]
	if !ok || o == nil {
		return nil, errors.New("order not found for payment")
	}
	if o.Status != "pending" {
		return nil, errors.New("invalid status transition")
	}
	o.Status = "cancelled"
	return m.items[orderID], nil
}
