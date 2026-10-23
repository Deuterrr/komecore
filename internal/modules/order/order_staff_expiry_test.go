package order

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockExpiryOrderRepo struct {
	mu       sync.RWMutex
	orders   map[uuid.UUID]*Order
	rowLocks sync.Map // uuid.UUID -> *sync.Mutex
}

func newMockExpiryOrderRepo() *mockExpiryOrderRepo {
	return &mockExpiryOrderRepo{
		orders: make(map[uuid.UUID]*Order),
	}
}

func (m *mockExpiryOrderRepo) getRowLock(id uuid.UUID) *sync.Mutex {
	actual, _ := m.rowLocks.LoadOrStore(id, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

func (m *mockExpiryOrderRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orders[id]
	if !ok || o == nil {
		return nil, nil
	}
	copied := *o
	return &copied, nil
}

func (m *mockExpiryOrderRepo) GetByIDForUpdate(_ context.Context, exec transaction.Executor, id uuid.UUID) (*Order, error) {
	// Mutex simulates database row-level locking (FOR UPDATE)
	rowMu := m.getRowLock(id)
	rowMu.Lock()
	if le, ok := exec.(*mockLockingExecutor); ok && le.onLock != nil {
		le.onLock(rowMu)
	} else {
		defer rowMu.Unlock()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orders[id]
	if !ok || o == nil {
		return nil, nil
	}
	copied := *o
	return &copied, nil
}

func (m *mockExpiryOrderRepo) GetByNumber(_ context.Context, _ transaction.Executor, _ string) (*Order, error) {
	return nil, nil
}

func (m *mockExpiryOrderRepo) UpdateStatus(_ context.Context, _ transaction.Executor, id uuid.UUID, status OrderStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if o, ok := m.orders[id]; ok {
		o.Status = status
	}
	return nil
}

func (m *mockExpiryOrderRepo) UpdateStatusWithSLA(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ OrderStatus, _ *time.Time, _ *time.Time) error {
	return nil
}

func (m *mockExpiryOrderRepo) Save(_ context.Context, _ transaction.Executor, order Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders[order.ID] = &order
	return nil
}

func (m *mockExpiryOrderRepo) FindOrders(_ context.Context, _ transaction.Executor, _ FindOrderParams) ([]Order, int, error) {
	return nil, 0, nil
}

func (m *mockExpiryOrderRepo) SetConfirmedAndExpiry(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time, _ time.Time) error {
	return nil
}

func (m *mockExpiryOrderRepo) FindExpiredUnfulfilledOrders(_ context.Context, _ transaction.Executor, now time.Time, limit int) ([]Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var results []Order
	for _, o := range m.orders {
		if (o.Status == OrderStatusConfirmed || o.Status == OrderStatusProcessing) &&
			o.HandlingExpiresAt != nil &&
			(o.HandlingExpiresAt.Before(now) || o.HandlingExpiresAt.Equal(now)) {
			results = append(results, *o)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

type mockExpiryItemRepo struct {
	mu    sync.RWMutex
	items map[uuid.UUID][]OrderItem
}

func newMockExpiryItemRepo() *mockExpiryItemRepo {
	return &mockExpiryItemRepo{
		items: make(map[uuid.UUID][]OrderItem),
	}
}

func (m *mockExpiryItemRepo) ListByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) ([]OrderItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.items[orderID], nil
}

func (m *mockExpiryItemRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]OrderItem, error) {
	return nil, nil
}

func (m *mockExpiryItemRepo) ListByShipmentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]OrderItem, error) {
	return nil, nil
}

func (m *mockExpiryItemRepo) SaveBulk(_ context.Context, _ transaction.Executor, _ []OrderItem) error {
	return nil
}

func (m *mockExpiryItemRepo) AssignShipment(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

type mockExpiryInventoryRepo struct {
	restockCalls int32
}

func (m *mockExpiryInventoryRepo) Restock(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	atomic.AddInt32(&m.restockCalls, 1)
	return nil
}

func (m *mockExpiryInventoryRepo) GetByProductIDAndShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID) (*inventory.Inventory, error) {
	return nil, nil
}
func (m *mockExpiryInventoryRepo) ListByProductID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventory.Inventory, error) {
	return nil, nil
}
func (m *mockExpiryInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID][]inventory.Inventory, error) {
	return nil, nil
}
func (m *mockExpiryInventoryRepo) ListByShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventory.Inventory, error) {
	return nil, nil
}
func (m *mockExpiryInventoryRepo) Create(_ context.Context, _ transaction.Executor, _ *inventory.Inventory) error {
	return nil
}
func (m *mockExpiryInventoryRepo) Reserve(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *mockExpiryInventoryRepo) Release(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *mockExpiryInventoryRepo) Commit(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *mockExpiryInventoryRepo) Update(_ context.Context, _ transaction.Executor, _ *inventory.Inventory) error {
	return nil
}
func (m *mockExpiryInventoryRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

type mockExpiryRefunder struct {
	refundCalls int32
}

func (m *mockExpiryRefunder) ProcessOrderRefund(_ context.Context, _ uuid.UUID, _ string) error {
	atomic.AddInt32(&m.refundCalls, 1)
	return nil
}

type mockLockingExecutor struct {
	transaction.NoopExecutor
	onLock func(*sync.Mutex)
}

type mockLockingTransactor struct{}

func (t *mockLockingTransactor) WithinTransaction(ctx context.Context, fn func(transaction.Executor) error) error {
	var locks []*sync.Mutex
	exec := &mockLockingExecutor{
		onLock: func(mu *sync.Mutex) {
			locks = append(locks, mu)
		},
	}
	defer func() {
		for i := len(locks) - 1; i >= 0; i-- {
			locks[i].Unlock()
		}
	}()
	return fn(exec)
}

func TestExpireUnfulfilledOrders_MultiReplicaProtection(t *testing.T) {
	ctx := context.Background()
	const numOrders = 10
	now := time.Now()
	expiredTime := now.Add(-1 * time.Hour)

	orderRepo := newMockExpiryOrderRepo()
	itemRepo := newMockExpiryItemRepo()
	invRepo := &mockExpiryInventoryRepo{}
	refunder := &mockExpiryRefunder{}
	transactor := &mockLockingTransactor{}
	executor := &transaction.NoopExecutor{}

	// Seed 10 expired orders with items
	for i := 0; i < numOrders; i++ {
		orderID := uuid.New()
		shopID := uuid.New()
		prodID := uuid.New()

		ord := &Order{
			ID:                orderID,
			Number:            fmt.Sprintf("ORD-%d", i+1),
			Status:            OrderStatusConfirmed,
			HandlingExpiresAt: &expiredTime,
			CreatedAt:         now.Add(-4 * 24 * time.Hour),
		}
		orderRepo.orders[orderID] = ord

		itemRepo.items[orderID] = []OrderItem{
			{
				ID:        uuid.New(),
				OrderID:   orderID,
				ProductID: prodID,
				ShopID:    shopID,
				Quantity:  2,
			},
		}
	}

	// Create 2 service instances representing Replica A and Replica B
	replicaA := NewService(
		executor, transactor, orderRepo, itemRepo, invRepo, nil, nil, nil, nil, nil, nil, nil, refunder, nil, nil,
	)
	replicaB := NewService(
		executor, transactor, orderRepo, itemRepo, invRepo, nil, nil, nil, nil, nil, nil, nil, refunder, nil, nil,
	)

	// Simulate simultaneous execution from both replicas
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	wg.Add(2)
	go func() {
		defer wg.Done()
		<-startBarrier
		replicaA.ExpireUnfulfilledOrders(ctx)
	}()
	go func() {
		defer wg.Done()
		<-startBarrier
		replicaB.ExpireUnfulfilledOrders(ctx)
	}()

	close(startBarrier)
	wg.Wait()

	// Critical assertions for multi-replica safety:
	// 1. Exactly 10 restock calls across both replicas (zero duplicate restocking)
	assert.Equal(t, int32(numOrders), atomic.LoadInt32(&invRepo.restockCalls),
		"expected each order to be restocked exactly once, zero duplicate restocks")

	// 2. Exactly 10 refund calls across both replicas (zero duplicate refunds)
	assert.Equal(t, int32(numOrders), atomic.LoadInt32(&refunder.refundCalls),
		"expected each order to be refunded exactly once, zero duplicate refunds")

	// 3. All orders transitioned to OrderStatusExpired
	orderRepo.mu.RLock()
	defer orderRepo.mu.RUnlock()
	for id, ord := range orderRepo.orders {
		assert.Equal(t, OrderStatusExpired, ord.Status, "order %s should be expired", id)
	}
}
